package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// ErrorResponse is the standard JSON error envelope returned by the gateway.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// skipRoute defines a method + path combination that bypasses JWT auth.
type skipRoute struct {
	Method string
	Path   string
}

// publicRoutes lists endpoints that do not require authentication.
var publicRoutes = []skipRoute{
	{Method: http.MethodPost, Path: "/api/v1/auth/register"},
	{Method: http.MethodPost, Path: "/api/v1/auth/login"},
	{Method: http.MethodPost, Path: "/api/v1/auth/refresh"},
}

// isPublicRoute returns true when the request matches a route that should skip
// JWT validation.
func isPublicRoute(method, path string) bool {
	for _, r := range publicRoutes {
		if r.Method == method && r.Path == path {
			return true
		}
	}
	return false
}

// JWTAuth returns Echo middleware that validates Bearer tokens, extracts claims,
// and propagates user context via request headers.
func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Allow health check through without auth.
			if c.Request().URL.Path == "/health" {
				return next(c)
			}

			// Skip authentication for public routes.
			if isPublicRoute(c.Request().Method, c.Request().URL.Path) {
				return next(c)
			}

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				log.Warn().Str("path", c.Request().URL.Path).Msg("missing authorization header")
				return c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "unauthorized",
					Message: "missing authorization header",
				})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				log.Warn().Str("path", c.Request().URL.Path).Msg("invalid authorization header format")
				return c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "unauthorized",
					Message: "invalid authorization header format, expected: Bearer <token>",
				})
			}

			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))

			if err != nil {
				log.Warn().Err(err).Str("path", c.Request().URL.Path).Msg("jwt validation failed")
				return c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "unauthorized",
					Message: "invalid or expired token",
				})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || !token.Valid {
				log.Warn().Str("path", c.Request().URL.Path).Msg("invalid jwt claims")
				return c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "unauthorized",
					Message: "invalid token claims",
				})
			}

			// Extract standard claims and propagate them as headers.
			userID, _ := claims.GetSubject()
			role, _ := claims["role"].(string)
			login, _ := claims["login"].(string)

			if userID != "" {
				c.Request().Header.Set("X-User-ID", userID)
			}
			if role != "" {
				c.Request().Header.Set("X-User-Role", role)
			}
			if login != "" {
				c.Request().Header.Set("X-User-Login", login)
			}

			return next(c)
		}
	}
}
