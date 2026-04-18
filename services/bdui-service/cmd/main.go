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
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/coach-link/platform/services/bdui-service/internal/client"
	"github.com/coach-link/platform/services/bdui-service/internal/config"
	"github.com/coach-link/platform/services/bdui-service/internal/handler"
	"github.com/coach-link/platform/services/bdui-service/internal/service"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "bdui-service").Logger()
	log.Logger = logger

	cfg := config.Load()
	logger.Info().Str("port", cfg.AppPort).Msg("starting bdui-service")

	userClient := client.NewUserClient(cfg.UserServiceURL)
	trainingClient := client.NewTrainingClient(cfg.TrainingServiceURL)

	svc := service.New(userClient, trainingClient, logger)
	h := handler.New(svc)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogMethod: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info().
				Str("method", v.Method).
				Str("uri", v.URI).
				Int("status", v.Status).
				Msg("request")
			return nil
		},
	}))

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	handler.RegisterRoutes(e, h)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.AppPort)
		logger.Info().Str("addr", addr).Msg("HTTP server listening")
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}
	logger.Info().Msg("bdui-service stopped")
}
