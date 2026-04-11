package pusher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/rs/zerolog"
	"google.golang.org/api/option"
)

// FCMPusher sends push notifications via Firebase Cloud Messaging.
type FCMPusher struct {
	client *messaging.Client
	log    zerolog.Logger
}

// NewFCMPusher initializes the Firebase app and returns an FCMPusher.
// credentialsFile is the path to the service account JSON.
// If credentialsFile is empty, FCM is disabled (Send becomes a no-op).
func NewFCMPusher(ctx context.Context, credentialsFile string, log zerolog.Logger) (*FCMPusher, error) {
	if credentialsFile == "" {
		log.Warn().Msg("FCM credentials not configured, push notifications disabled")
		return &FCMPusher{client: nil, log: log}, nil
	}

	// Check if file exists and contains valid service account JSON
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Warn().Err(err).Msg("FCM credentials file not readable, push notifications disabled")
		return &FCMPusher{client: nil, log: log}, nil
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil || parsed["project_id"] == nil {
		log.Warn().Msg("FCM credentials file is not a valid service account, push notifications disabled")
		return &FCMPusher{client: nil, log: log}, nil
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase messaging: %w", err)
	}

	log.Info().Msg("FCM push notifications enabled")
	return &FCMPusher{client: client, log: log}, nil
}

// Enabled returns true if FCM is configured.
func (p *FCMPusher) Enabled() bool {
	return p.client != nil
}

// Send sends a push notification to the given FCM tokens.
// Silently skips if FCM is not configured or tokens list is empty.
func (p *FCMPusher) Send(ctx context.Context, tokens []string, title, body string, data map[string]string) {
	if p.client == nil || len(tokens) == 0 {
		return
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}
	if len(data) > 0 {
		msg.Data = data
	}

	resp, err := p.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		p.log.Error().Err(err).Int("token_count", len(tokens)).Msg("FCM multicast send failed")
		return
	}

	p.log.Info().
		Int("success", resp.SuccessCount).
		Int("failure", resp.FailureCount).
		Msg("FCM push sent")
}
