package service_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coach-link/platform/services/notification-service/internal/model"
	"github.com/coach-link/platform/services/notification-service/internal/repository"
	"github.com/coach-link/platform/services/notification-service/internal/service"
)

// ── Mocks ──────────────────────────────────────

type mockFCM struct{ enabled bool }

func (m *mockFCM) Enabled() bool { return m.enabled }
func (m *mockFCM) Send(_ context.Context, _ []string, _, _ string, _ map[string]string) {}

type mockRepo struct {
	notifications    []model.Notification
	notificationsErr error
	unreadCount      int
	unreadErr        error
	markReadResult   *model.Notification
	markReadErr      error
	createErr        error
	upsertErr        error
	deviceTokens     []string
	deviceTokensErr  error
}

func (m *mockRepo) CreateNotification(_ context.Context, n *model.Notification) error {
	n.ID = "notif-1"
	return m.createErr
}
func (m *mockRepo) GetNotifications(_ context.Context, _ string, _ *bool, _, _ int) ([]model.Notification, int, error) {
	return m.notifications, len(m.notifications), m.notificationsErr
}
func (m *mockRepo) GetUnreadCount(_ context.Context, _ string) (int, error) {
	return m.unreadCount, m.unreadErr
}
func (m *mockRepo) MarkRead(_ context.Context, _, _ string) (*model.Notification, error) {
	return m.markReadResult, m.markReadErr
}
func (m *mockRepo) MarkAllRead(_ context.Context, _ string) error { return nil }
func (m *mockRepo) UpsertDeviceToken(_ context.Context, _, _, _ string) error {
	return m.upsertErr
}
func (m *mockRepo) GetDeviceTokensByUserID(_ context.Context, _ string) ([]string, error) {
	return m.deviceTokens, m.deviceTokensErr
}

func newSvc(repo service.NotificationRepository) *service.Service {
	return service.New(repo, &mockFCM{}, zerolog.Nop())
}

// ── RegisterDeviceToken ────────────────────────

func TestRegisterDeviceToken_EmptyToken_Returns400(t *testing.T) {
	svc := newSvc(&mockRepo{})

	err := svc.RegisterDeviceToken(context.Background(), "user-1", "", "")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
	assert.Equal(t, "VALIDATION_ERROR", se.Code)
}

func TestRegisterDeviceToken_Success(t *testing.T) {
	svc := newSvc(&mockRepo{})

	err := svc.RegisterDeviceToken(context.Background(), "user-1", "fcm-token-abc", "iPhone 15")
	require.NoError(t, err)
}

func TestRegisterDeviceToken_RepoError_Propagates(t *testing.T) {
	svc := newSvc(&mockRepo{upsertErr: repository.ErrNotFound})

	err := svc.RegisterDeviceToken(context.Background(), "user-1", "some-token", "")
	require.Error(t, err)
}

// ── MarkRead ───────────────────────────────────

func TestMarkRead_NotFound_Returns404(t *testing.T) {
	svc := newSvc(&mockRepo{markReadErr: repository.ErrNotFound})

	_, err := svc.MarkRead(context.Background(), "user-1", "notif-99")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 404, se.Status)
}

func TestMarkRead_Success(t *testing.T) {
	n := &model.Notification{ID: "notif-1", UserID: "user-1", Type: "training_assigned", Title: "New plan"}
	svc := newSvc(&mockRepo{markReadResult: n})

	resp, err := svc.MarkRead(context.Background(), "user-1", "notif-1")
	require.NoError(t, err)
	assert.Equal(t, "notif-1", resp.ID)
}

// ── GetNotifications (pagination) ─────────────

func TestGetNotifications_PaginationMetadata(t *testing.T) {
	notifications := make([]model.Notification, 5)
	for i := range notifications {
		notifications[i] = model.Notification{
			ID:     "n",
			UserID: "user-1",
			Type:   "test",
			Title:  "title",
		}
	}
	svc := newSvc(&mockRepo{notifications: notifications, unreadCount: 3})

	result, err := svc.GetNotifications(context.Background(), "user-1", nil, 1, 5)
	require.NoError(t, err)
	assert.Len(t, result.Items, 5)
	assert.Equal(t, 3, result.UnreadCount)
	assert.Equal(t, 1, result.Pagination.TotalPages)
}

func TestGetNotifications_MultiplePages(t *testing.T) {
	notifications := make([]model.Notification, 3)
	for i := range notifications {
		notifications[i] = model.Notification{ID: "n", UserID: "u", Type: "t", Title: "t"}
	}
	// 3 items with page_size=2 → 2 pages
	svc := newSvc(&mockRepo{notifications: notifications})

	result, err := svc.GetNotifications(context.Background(), "user-1", nil, 1, 2)
	require.NoError(t, err)
	// mock returns 3 items but TotalPages depends on total count from repo
	// Our mock returns len(notifications) as total = 3, page_size = 2 → ceil(3/2) = 2 pages
	assert.Equal(t, 2, result.Pagination.TotalPages)
}

// ── CreateNotificationFromEvent ────────────────

func TestCreateNotificationFromEvent_Success(t *testing.T) {
	svc := newSvc(&mockRepo{})

	err := svc.CreateNotificationFromEvent(
		context.Background(),
		"user-1",
		"training_assigned",
		"Новая тренировка",
		"Тренер назначил план",
		map[string]interface{}{"assignment_id": "assign-1"},
	)
	require.NoError(t, err)
}

func TestCreateNotificationFromEvent_EmptyBody_StillCreates(t *testing.T) {
	svc := newSvc(&mockRepo{})

	err := svc.CreateNotificationFromEvent(
		context.Background(), "user-1", "group_added", "Добавлен в группу", "", nil,
	)
	require.NoError(t, err)
}
