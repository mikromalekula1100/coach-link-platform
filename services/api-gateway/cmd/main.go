package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/coach-link/platform/services/api-gateway/internal/config"
	"github.com/coach-link/platform/services/api-gateway/internal/middleware"
	"github.com/coach-link/platform/services/api-gateway/internal/proxy"
)

func main() {
	// Initialise structured logging.
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Str("service", "api-gateway").
		Logger()

	// Load configuration from environment.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Create Echo instance.
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Global middleware.
	e.Use(emw.Recover())
	e.Use(emw.RequestID())
	e.Use(middleware.CORSConfig())
	e.Use(requestLogger())
	e.Use(middleware.JWTAuth(cfg.JWTSecret))

	// Build reverse proxy router and register routes.
	router, err := proxy.NewRouter(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create proxy router")
	}
	router.RegisterRoutes(e)

	// Start server in a goroutine so that graceful shutdown can proceed.
	addr := fmt.Sprintf(":%s", cfg.Port)
	go func() {
		log.Info().Str("addr", addr).Msg("starting api-gateway")
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down api-gateway")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("forced shutdown")
	}

	log.Info().Msg("api-gateway stopped")
}

// requestLogger returns Echo middleware that emits a structured zerolog entry
// for every handled request.
func requestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()
			latency := time.Since(start)

			evt := log.Info()
			if res.Status >= 500 {
				evt = log.Error()
			} else if res.Status >= 400 {
				evt = log.Warn()
			}

			evt.
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Int("status", res.Status).
				Dur("latency", latency).
				Str("remote_ip", c.RealIP()).
				Str("request_id", res.Header().Get(echo.HeaderXRequestID)).
				Msg("request")

			return nil
		}
	}
}
