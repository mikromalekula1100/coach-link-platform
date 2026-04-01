package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/coach-link/platform/services/training-service/internal/client"
	"github.com/coach-link/platform/services/training-service/internal/config"
	"github.com/coach-link/platform/services/training-service/internal/handler"
	"github.com/coach-link/platform/services/training-service/internal/repository"
	"github.com/coach-link/platform/services/training-service/internal/service"
	"github.com/coach-link/platform/services/training-service/migrations"
)

func main() {
	// Logger
	zerolog.TimeFieldFormat = time.RFC3339
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "training-service").Logger()
	log.Logger = logger

	cfg := config.Load()
	logger.Info().Str("port", cfg.AppPort).Msg("starting training-service")

	// ── PostgreSQL ──────────────────────────────
	db, err := connectDB(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// ── Goose migrations ───────────────────────
	if err := runMigrations(db.DB); err != nil {
		logger.Fatal().Err(err).Msg("failed to run migrations")
	}
	logger.Info().Msg("database migrations applied")

	// ── NATS JetStream ─────────────────────────
	nc, js, err := connectNATS(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	defer nc.Close()

	// ── HTTP client for User Service ───────────
	userClient := client.NewUserClient(cfg.UserServiceURL)

	// ── Layers ─────────────────────────────────
	repo := repository.New(db)
	svc := service.New(repo, js, logger, userClient)
	h := handler.New(svc)

	// ── Echo HTTP ──────────────────────────────
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = handler.NewValidator()
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

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	handler.RegisterRoutes(e, h)

	// ── Start & graceful shutdown ──────────────
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}
	logger.Info().Msg("training-service stopped")
}

func connectDB(cfg *config.Config, logger zerolog.Logger) (*sqlx.DB, error) {
	dsn := cfg.DSN()
	deadline := time.Now().Add(30 * time.Second)

	for {
		db, err := sqlx.Connect("postgres", dsn)
		if err == nil {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)
			logger.Info().Msg("connected to PostgreSQL")
			return db, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("database connection timeout after 30s: %w", err)
		}

		logger.Warn().Err(err).Msg("waiting for database...")
		time.Sleep(2 * time.Second)
	}
}

func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, ".")
}

func connectNATS(cfg *config.Config, logger zerolog.Logger) (*nats.Conn, nats.JetStreamContext, error) {
	nc, err := nats.Connect(cfg.NATSURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Warn().Err(err).Msg("NATS disconnected")
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info().Msg("NATS reconnected")
		}),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("jetstream: %w", err)
	}

	// Ensure stream exists (idempotent)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "COACHLINK",
		Subjects: []string{"coachlink.>"},
		Storage:  nats.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("add stream: %w", err)
	}

	logger.Info().Str("url", cfg.NATSURL).Msg("connected to NATS JetStream")
	return nc, js, nil
}
