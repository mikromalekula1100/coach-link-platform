package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
)

// CORSConfig returns an Echo CORS middleware configured for development use.
// It allows all origins and a set of common methods and headers used by the
// CoachLink front-end.
func CORSConfig() echo.MiddlewareFunc {
	return emw.CORSWithConfig(emw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Authorization",
			"Content-Type",
			"X-User-ID",
			"X-User-Role",
		},
		AllowCredentials: false,
		MaxAge:           3600,
	})
}
