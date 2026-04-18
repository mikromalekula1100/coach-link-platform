package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/coach-link/platform/services/api-gateway/internal/config"
)

// Router holds reverse-proxy instances for every back-end service and exposes
// an API to register routes on an Echo instance.
type Router struct {
	authProxy         *httputil.ReverseProxy
	userProxy         *httputil.ReverseProxy
	trainingProxy     *httputil.ReverseProxy
	notificationProxy *httputil.ReverseProxy
	analyticsProxy    *httputil.ReverseProxy
	aiProxy           *httputil.ReverseProxy
	bduiProxy         *httputil.ReverseProxy
}

// NewRouter parses the upstream URLs from config and creates a Router with
// pre-configured reverse proxies.
func NewRouter(cfg *config.Config) (*Router, error) {
	authURL, err := url.Parse(cfg.AuthServiceURL)
	if err != nil {
		return nil, err
	}
	userURL, err := url.Parse(cfg.UserServiceURL)
	if err != nil {
		return nil, err
	}
	trainingURL, err := url.Parse(cfg.TrainingServiceURL)
	if err != nil {
		return nil, err
	}
	notificationURL, err := url.Parse(cfg.NotificationServiceURL)
	if err != nil {
		return nil, err
	}
	analyticsURL, err := url.Parse(cfg.AnalyticsServiceURL)
	if err != nil {
		return nil, err
	}
	aiURL, err := url.Parse(cfg.AIServiceURL)
	if err != nil {
		return nil, err
	}
	bduiURL, err := url.Parse(cfg.BDUIServiceURL)
	if err != nil {
		return nil, err
	}

	return &Router{
		authProxy:         newProxy(authURL),
		userProxy:         newProxy(userURL),
		trainingProxy:     newProxy(trainingURL),
		notificationProxy: newProxy(notificationURL),
		analyticsProxy:    newProxy(analyticsURL),
		aiProxy:           newProxy(aiURL),
		bduiProxy:         newProxy(bduiURL),
	}, nil
}

// RegisterRoutes wires path-prefix groups and the health endpoint to the Echo
// instance.
func (r *Router) RegisterRoutes(e *echo.Echo) {
	// Health check -- does not go through JWT middleware.
	e.GET("/health", healthCheck)

	// Auth service routes.
	e.Any("/api/v1/auth/*", r.proxyHandler(r.authProxy))

	// User service routes.
	e.Any("/api/v1/users/*", r.proxyHandler(r.userProxy))
	e.Any("/api/v1/connections/*", r.proxyHandler(r.userProxy))
	e.Any("/api/v1/groups", r.proxyHandler(r.userProxy))
	e.Any("/api/v1/groups/*", r.proxyHandler(r.userProxy))

	// Training service routes.
	e.Any("/api/v1/training/*", r.proxyHandler(r.trainingProxy))

	// Notification service routes.
	e.Any("/api/v1/notifications", r.proxyHandler(r.notificationProxy))
	e.Any("/api/v1/notifications/*", r.proxyHandler(r.notificationProxy))

	// Analytics service routes.
	e.Any("/api/v1/analytics/*", r.proxyHandler(r.analyticsProxy))

	// AI service routes.
	e.Any("/api/v1/ai/*", r.proxyHandler(r.aiProxy))

	// BDUI service routes.
	e.Any("/api/v1/bdui/*", r.proxyHandler(r.bduiProxy))
}

// healthCheck responds to liveness probes.
func healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// proxyHandler returns an Echo handler that delegates the request to the given
// reverse proxy while forwarding identity headers set by the JWT middleware.
func (r *Router) proxyHandler(proxy *httputil.ReverseProxy) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		res := c.Response().Writer

		// Forward identity headers added by JWT middleware.
		copyHeader(req, "X-User-ID")
		copyHeader(req, "X-User-Role")
		copyHeader(req, "X-User-Login")

		proxy.ServeHTTP(res, req)
		return nil
	}
}

// copyHeader is a no-op safety net -- the headers are already on the request
// from the JWT middleware, but this documents intent and ensures presence.
func copyHeader(req *http.Request, header string) {
	if v := req.Header.Get(header); v != "" {
		req.Header.Set(header, v)
	}
}

// newProxy creates a reverse proxy for the given target URL.
func newProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Preserve the original path as-is so upstream services see the
			// full /api/v1/... prefix.
			// req.URL.Path is already set by Echo.

			// Strip hop-by-hop headers that should not be forwarded.
			if _, ok := req.Header["Te"]; ok {
				req.Header.Del("Te")
			}

			// Make sure X-Forwarded-For is set.
			if clientIP := req.Header.Get("X-Real-Ip"); clientIP == "" {
				if fwd := req.Header.Get("X-Forwarded-For"); fwd != "" {
					req.Header.Set("X-Real-Ip", strings.Split(fwd, ",")[0])
				}
			}
		},
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			log.Error().Err(err).
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Str("upstream", target.String()).
				Msg("proxy error")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"bad_gateway","message":"upstream service unavailable"}`))
		},
	}
	return proxy
}
