package service

import (
	"context"
	"encoding/json"
	"math"

	"github.com/rs/zerolog"

	"github.com/coach-link/platform/services/notification-service/internal/model"
	"github.com/coach-link/platform/services/notification-service/internal/repository"
)

// ──────────────────────────────────────────────
// Error types
// ──────────────────────────────────────────────

type ServiceError struct {
	Code    string
	Message string
	Status  int
}

func (e *ServiceError) Error() string { return e.Message }

func notFound(msg string) *ServiceError {
	return &ServiceError{Code: "NOT_FOUND", Message: msg, Status: 404}
}

func badRequest(msg string) *ServiceError {
	return &ServiceError{Code: "VALIDATION_ERROR", Message: msg, Status: 400}
}

// IsServiceError checks if an error is a ServiceError and returns it.
func IsServiceError(err error) (*ServiceError, bool) {
	if se, ok := err.(*ServiceError); ok {
		return se, true
	}
	return nil, false
}

// ──────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────

type Service struct {
	repo *repository.Repository
	log  zerolog.Logger
}

func New(repo *repository.Repository, log zerolog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// CreateNotificationFromEvent creates a notification in the database from an event payload.
func (s *Service) CreateNotificationFromEvent(ctx context.Context, userID, nType, title, body string, data map[string]interface{}) error {
	var bodyPtr *string
	if body != "" {
		bodyPtr = &body
	}

	var dataBytes json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			s.log.Error().Err(err).Msg("failed to marshal notification data")
			return err
		}
		dataBytes = b
	}

	n := &model.Notification{
		UserID: userID,
		Type:   nType,
		Title:  title,
		Body:   bodyPtr,
		Data:   dataBytes,
	}

	if err := s.repo.CreateNotification(ctx, n); err != nil {
		s.log.Error().Err(err).
			Str("user_id", userID).
			Str("type", nType).
			Msg("failed to create notification")
		return err
	}

	s.log.Info().
		Str("notification_id", n.ID).
		Str("user_id", userID).
		Str("type", nType).
		Msg("notification created")

	return nil
}

// GetNotifications returns a paginated list of notifications for a user with unread count.
func (s *Service) GetNotifications(ctx context.Context, userID string, isRead *bool, page, pageSize int) (*model.NotificationsListResponse, error) {
	notifications, total, err := s.repo.GetNotifications(ctx, userID, isRead, page, pageSize)
	if err != nil {
		return nil, err
	}

	unreadCount, err := s.repo.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]model.NotificationResponse, 0, len(notifications))
	for i := range notifications {
		items = append(items, model.ToNotificationResponse(&notifications[i]))
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}

	return &model.NotificationsListResponse{
		Items: items,
		Pagination: model.Pagination{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
		UnreadCount: unreadCount,
	}, nil
}

// MarkRead marks a single notification as read and returns it.
func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (*model.NotificationResponse, error) {
	n, err := s.repo.MarkRead(ctx, userID, notificationID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, notFound("Notification not found")
		}
		return nil, err
	}

	resp := model.ToNotificationResponse(n)
	return &resp, nil
}

// MarkAllRead marks all unread notifications as read for a user.
func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// RegisterDeviceToken registers or updates a device token for push notifications.
func (s *Service) RegisterDeviceToken(ctx context.Context, userID, fcmToken, deviceInfo string) error {
	if fcmToken == "" {
		return badRequest("fcm_token is required")
	}

	if err := s.repo.UpsertDeviceToken(ctx, userID, fcmToken, deviceInfo); err != nil {
		s.log.Error().Err(err).
			Str("user_id", userID).
			Msg("failed to upsert device token")
		return err
	}

	s.log.Info().
		Str("user_id", userID).
		Msg("device token registered")

	return nil
}
