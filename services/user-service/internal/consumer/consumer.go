package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/coach-link/platform/pkg/events"
	"github.com/coach-link/platform/services/user-service/internal/model"
	"github.com/coach-link/platform/services/user-service/internal/repository"
)

type Consumer struct {
	js   nats.JetStreamContext
	repo *repository.Repository
	log  zerolog.Logger
	sub  *nats.Subscription
}

func New(js nats.JetStreamContext, repo *repository.Repository, log zerolog.Logger) *Consumer {
	return &Consumer{js: js, repo: repo, log: log}
}

// Start subscribes to user.registered events and processes them.
func (c *Consumer) Start() error {
	sub, err := c.js.Subscribe(
		events.SubjectUserRegistered,
		c.handleUserRegistered,
		nats.Durable("user-service-user-registered"),
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	if err != nil {
		return err
	}
	c.sub = sub
	c.log.Info().Str("subject", events.SubjectUserRegistered).Msg("consumer started")
	return nil
}

// Stop drains the subscription gracefully.
func (c *Consumer) Stop() {
	if c.sub != nil {
		if err := c.sub.Drain(); err != nil {
			c.log.Error().Err(err).Msg("failed to drain subscription")
		}
	}
}

func (c *Consumer) handleUserRegistered(msg *nats.Msg) {
	var evt events.Event
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		c.log.Error().Err(err).Msg("failed to unmarshal event envelope")
		_ = msg.Nak()
		return
	}

	// Re-marshal payload to parse into the specific type
	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("failed to re-marshal payload")
		_ = msg.Nak()
		return
	}

	var payload events.UserRegisteredPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		c.log.Error().Err(err).Msg("failed to unmarshal UserRegisteredPayload")
		_ = msg.Nak()
		return
	}

	now := time.Now().UTC()
	profile := model.UserProfile{
		ID:        payload.UserID,
		Login:     payload.Login,
		Email:     payload.Email,
		FullName:  payload.FullName,
		Role:      payload.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.repo.CreateProfile(ctx, profile); err != nil {
		c.log.Error().Err(err).
			Str("user_id", payload.UserID).
			Str("login", payload.Login).
			Msg("failed to create user profile from event")
		_ = msg.Nak()
		return
	}

	c.log.Info().
		Str("user_id", payload.UserID).
		Str("login", payload.Login).
		Str("role", payload.Role).
		Msg("user profile created from registration event")

	_ = msg.Ack()
}
