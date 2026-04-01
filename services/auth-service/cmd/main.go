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

	"github.com/coach-link/platform/pkg/events"
	"github.com/coach-link/platform/services/auth-service/internal/config"
	"github.com/coach-link/platform/services/auth-service/internal/handler"
	"github.com/coach-link/platform/services/auth-service/internal/repository"
	"github.com/coach-link/platform/services/auth-service/internal/service"
	"github.com/coach-link/platform/services/auth-service/migrations"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if cfg.JWTSecret == "" {
		log.Fatal().Msg("JWT_SECRET environment variable is required")
	}

	db, err := connectDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("connected to PostgreSQL")

	if err := runMigrations(db.DB); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	log.Info().Msg("migrations applied")

	nc, js, err := connectNATS(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	defer nc.Close()
	log.Info().Msg("connected to NATS JetStream")

	repo := repository.New(db)
	svc := service.New(repo, cfg, js)
	h := handler.New(svc)

	e := echo.New()
	e.HideBanner = true
	e.Validator = handler.NewValidator()
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(requestLogger())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	h.Register(e)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Info().Str("addr", addr).Msg("starting HTTP server")
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	log.Info().Msg("auth-service stopped")
}

func connectDB(cfg *config.Config) (*sqlx.DB, error) {
	dsn := cfg.DSN()
	deadline := time.Now().Add(30 * time.Second)

	for {
		db, err := sqlx.Connect("postgres", dsn)
		if err == nil {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(10)
			db.SetConnMaxLifetime(5 * time.Minute)
			return db, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("could not connect to database after 30s: %w", err)
		}

		log.Warn().Err(err).Msg("database not ready, retrying in 1s...")
		time.Sleep(1 * time.Second)
	}
}

func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	return goose.Up(db, ".")
}

func connectNATS(cfg *config.Config) (*nats.Conn, nats.JetStreamContext, error) {
	nc, err := nats.Connect(cfg.NATSURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(30),
		nats.ReconnectWait(1*time.Second),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("jetstream context: %w", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     events.StreamName,
		Subjects: []string{"coachlink.>"},
		Storage:  nats.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("add stream: %w", err)
	}

	return nc, js, nil
}

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

			log.Info().
				Str("method", req.Method).
				Str("uri", req.RequestURI).
				Int("status", res.Status).
				Dur("latency", time.Since(start)).
				Str("request_id", res.Header().Get(echo.HeaderXRequestID)).
				Msg("request")

			return nil
		}
	}
}
