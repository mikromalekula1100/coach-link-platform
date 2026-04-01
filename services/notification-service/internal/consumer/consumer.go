package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/coach-link/platform/pkg/events"
	"github.com/coach-link/platform/services/notification-service/internal/service"
)

// ConsumerManager manages all NATS JetStream consumers for the notification service.
type ConsumerManager struct {
	js   nats.JetStreamContext
	svc  *service.Service
	log  zerolog.Logger
	subs []*nats.Subscription
}

func NewConsumerManager(js nats.JetStreamContext, svc *service.Service, log zerolog.Logger) *ConsumerManager {
	return &ConsumerManager{
		js:  js,
		svc: svc,
		log: log,
	}
}

// Start subscribes to all relevant event subjects.
func (cm *ConsumerManager) Start() error {
	consumers := []struct {
		subject  string
		durable  string
		handler  nats.MsgHandler
	}{
		{
			subject: events.SubjectConnectionRequested,
			durable: "notification-service-connection-requested",
			handler: cm.handleConnectionRequested,
		},
		{
			subject: events.SubjectConnectionAccepted,
			durable: "notification-service-connection-accepted",
			handler: cm.handleConnectionAccepted,
		},
		{
			subject: events.SubjectConnectionRejected,
			durable: "notification-service-connection-rejected",
			handler: cm.handleConnectionRejected,
		},
		{
			subject: events.SubjectTrainingAssigned,
			durable: "notification-service-training-assigned",
			handler: cm.handleTrainingAssigned,
		},
		{
			subject: events.SubjectTrainingDeleted,
			durable: "notification-service-training-deleted",
			handler: cm.handleTrainingDeleted,
		},
		{
			subject: events.SubjectReportSubmitted,
			durable: "notification-service-report-submitted",
			handler: cm.handleReportSubmitted,
		},
		{
			subject: events.SubjectGroupAthleteAdded,
			durable: "notification-service-group-athlete-added",
			handler: cm.handleGroupAthleteAdded,
		},
		{
			subject: events.SubjectGroupAthleteRemoved,
			durable: "notification-service-group-athlete-removed",
			handler: cm.handleGroupAthleteRemoved,
		},
	}

	for _, c := range consumers {
		sub, err := cm.js.Subscribe(
			c.subject,
			c.handler,
			nats.Durable(c.durable),
			nats.DeliverAll(),
			nats.AckExplicit(),
		)
		if err != nil {
			return fmt.Errorf("subscribe to %s: %w", c.subject, err)
		}
		cm.subs = append(cm.subs, sub)
		cm.log.Info().Str("subject", c.subject).Str("durable", c.durable).Msg("consumer started")
	}

	return nil
}

// Stop drains all subscriptions gracefully.
func (cm *ConsumerManager) Stop() {
	for _, sub := range cm.subs {
		if err := sub.Drain(); err != nil {
			cm.log.Error().Err(err).Str("subject", sub.Subject).Msg("failed to drain subscription")
		}
	}
	cm.log.Info().Msg("all consumers stopped")
}

// ──────────────────────────────────────────────
// Event handlers
// ──────────────────────────────────────────────

// coachlink.connection.requested → notify COACH
func (cm *ConsumerManager) handleConnectionRequested(msg *nats.Msg) {
	var payload events.ConnectionRequestedPayload
	if err := cm.parsePayload(msg, &payload); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Новая заявка от спортсмена"
	body := fmt.Sprintf("%s отправил(а) вам заявку на привязку", payload.AthleteFullName)
	data := map[string]interface{}{
		"request_id": payload.RequestID,
	}

	if err := cm.svc.CreateNotificationFromEvent(ctx, payload.CoachID, "connection_requested", title, body, data); err != nil {
		cm.log.Error().Err(err).Str("event", events.SubjectConnectionRequested).Msg("failed to create notification")
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}

// coachlink.connection.accepted → notify ATHLETE
func (cm *ConsumerManager) handleConnectionAccepted(msg *nats.Msg) {
	var payload events.ConnectionAcceptedPayload
	if err := cm.parsePayload(msg, &payload); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Заявка принята"
	body := fmt.Sprintf("Тренер %s принял(а) вашу заявку", payload.CoachFullName)
	data := map[string]interface{}{
		"request_id": payload.RequestID,
	}

	if err := cm.svc.CreateNotificationFromEvent(ctx, payload.AthleteID, "connection_accepted", title, body, data); err != nil {
		cm.log.Error().Err(err).Str("event", events.SubjectConnectionAccepted).Msg("failed to create notification")
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}

// coachlink.connection.rejected → notify ATHLETE
func (cm *ConsumerManager) handleConnectionRejected(msg *nats.Msg) {
	var payload events.ConnectionRejectedPayload
	if err := cm.parsePayload(msg, &payload); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Заявка отклонена"
	body := fmt.Sprintf("Тренер %s отклонил(а) вашу заявку", payload.CoachFullName)
	data := map[string]interface{}{
		"request_id": payload.RequestID,
	}

	if err := cm.svc.CreateNotificationFromEvent(ctx, payload.AthleteID, "connection_rejected", title, body, data); err != nil {
		cm.log.Error().Err(err).Str("event", events.SubjectConnectionRejected).Msg("failed to create notification")
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}

// coachlink.training.assigned → notify ATHLETE
func (cm *ConsumerManager) handleTrainingAssigned(msg *nats.Msg) {
	var payload events.TrainingAssignedPayload
	if err := cm.parsePayload(msg, &payload); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Новое задание от тренера"
	body := fmt.Sprintf("Тренер %s назначил вам тренировку \"%s\" на %s", payload.CoachFullName, payload.Title, payload.ScheduledDate)
	data := map[string]interface{}{
		"assignment_id": payload.AssignmentID,
	}

	if err := cm.svc.CreateNotificationFromEvent(ctx, payload.AthleteID, "training_assigned", title, body, data); err != nil {
		cm.log.Error().Err(err).Str("event", events.SubjectTrainingAssigned).Msg("failed to create notification")
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}

// coachlink.training.deleted → notify ATHLETE
func (cm *ConsumerManager) handleTrainingDeleted(msg *nats.Msg) {
	var payload events.TrainingDeletedPayload
	if err := cm.parsePayload(msg, &payload); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Задание удалено"
	body := fmt.Sprintf("Тренер удалил задание \"%s\"", payload.Title)
	data := map[string]interface{}{
		"assignment_id": payload.AssignmentID,
	}

	if err := cm.svc.CreateNotificationFromEvent(ctx, payload.AthleteID, "training_deleted", title, body, data); err != nil {
		cm.log.Error().Err(err).Str("event", events.SubjectTrainingDeleted).Msg("failed to create notification")
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}

// coachlink.report.submitted → notify COACH
func (cm *ConsumerManager) handleReportSubmitted(msg *nats.Msg) {
	var payload events.ReportSubmittedPayload
	if err := cm.parsePayload(msg, &payload); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Новый отчёт"
	body := fmt.Sprintf("%s отправил(а) отчёт по заданию \"%s\"", payload.AthleteFullName, payload.Title)
	data := map[string]interface{}{
		"assignment_id": payload.AssignmentID,
	}

	if err := cm.svc.CreateNotificationFromEvent(ctx, payload.CoachID, "report_submitted", title, body, data); err != nil {
		cm.log.Error().Err(err).Str("event", events.SubjectReportSubmitted).Msg("failed to create notification")
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}

// coachlink.group.athlete_added → notify ATHLETE
func (cm *ConsumerManager) handleGroupAthleteAdded(msg *nats.Msg) {
	var payload events.GroupAthleteAddedPayload
	if err := cm.parsePayload(msg, &payload); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Добавление в группу"
	body := fmt.Sprintf("Вы добавлены в группу \"%s\"", payload.GroupName)
	data := map[string]interface{}{
		"group_id": payload.GroupID,
	}

	if err := cm.svc.CreateNotificationFromEvent(ctx, payload.AthleteID, "group_athlete_added", title, body, data); err != nil {
		cm.log.Error().Err(err).Str("event", events.SubjectGroupAthleteAdded).Msg("failed to create notification")
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}

// coachlink.group.athlete_removed → notify ATHLETE
func (cm *ConsumerManager) handleGroupAthleteRemoved(msg *nats.Msg) {
	var payload events.GroupAthleteRemovedPayload
	if err := cm.parsePayload(msg, &payload); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Удаление из группы"
	body := fmt.Sprintf("Вы удалены из группы \"%s\"", payload.GroupName)
	data := map[string]interface{}{
		"group_id": payload.GroupID,
	}

	if err := cm.svc.CreateNotificationFromEvent(ctx, payload.AthleteID, "group_athlete_removed", title, body, data); err != nil {
		cm.log.Error().Err(err).Str("event", events.SubjectGroupAthleteRemoved).Msg("failed to create notification")
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

// parsePayload unmarshals a NATS message into an Event envelope, then re-marshals
// the payload into the given target struct. On failure it NAKs the message.
func (cm *ConsumerManager) parsePayload(msg *nats.Msg, target interface{}) error {
	var evt events.Event
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		cm.log.Error().Err(err).Str("subject", msg.Subject).Msg("failed to unmarshal event envelope")
		_ = msg.Nak()
		return err
	}

	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		cm.log.Error().Err(err).Str("subject", msg.Subject).Msg("failed to re-marshal payload")
		_ = msg.Nak()
		return err
	}

	if err := json.Unmarshal(payloadBytes, target); err != nil {
		cm.log.Error().Err(err).Str("subject", msg.Subject).Msg("failed to unmarshal payload into target type")
		_ = msg.Nak()
		return err
	}

	cm.log.Debug().
		Str("subject", msg.Subject).
		Str("event_id", evt.EventID).
		Msg("event received")

	return nil
}
