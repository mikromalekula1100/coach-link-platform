module github.com/coach-link/platform/services/training-service

go 1.22

require (
	github.com/coach-link/platform/pkg/events v0.0.0
	github.com/go-playground/validator/v10 v10.22.1
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/labstack/echo/v4 v4.12.0
	github.com/lib/pq v1.10.9
	github.com/nats-io/nats.go v1.37.0
	github.com/pressly/goose/v3 v3.22.1
	github.com/rs/zerolog v1.33.0
)

replace github.com/coach-link/platform/pkg/events => ../../pkg/events
