package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNotificationCoachHas(t *testing.T) {
	// Coach1 should have notifications from: connection requests, report submissions, etc.
	result := waitForNotifications(t, coach1Token, 1, notifyTimeout)
	assert.Greater(t, result.UnreadCount, 0)
	assert.NotEmpty(t, result.Items)

	// Verify notification structure
	n := result.Items[0]
	assert.NotEmpty(t, n.ID)
	assert.NotEmpty(t, n.Type)
	assert.NotEmpty(t, n.Title)
	assert.False(t, n.IsRead)
	assert.NotEmpty(t, n.CreatedAt)
}

func testNotificationAthleteHas(t *testing.T) {
	// Athlete1 should have notifications from: connection accepted, training assigned, etc.
	result := waitForNotifications(t, athlete1Token, 1, notifyTimeout)
	assert.Greater(t, result.UnreadCount, 0)
	assert.NotEmpty(t, result.Items)
}

func testNotificationFilterUnread(t *testing.T) {
	status, data, err := client.Get("/api/v1/notifications?is_read=false", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp NotificationsListResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	for _, n := range resp.Items {
		assert.False(t, n.IsRead, "all items should be unread when filtering is_read=false")
	}
}

func testNotificationPagination(t *testing.T) {
	status, data, err := client.Get("/api/v1/notifications?page=1&page_size=1", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp NotificationsListResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.LessOrEqual(t, len(resp.Items), 1)
	assert.Equal(t, 1, resp.Pagination.Page)
	assert.Equal(t, 1, resp.Pagination.PageSize)
}

func testNotificationNoAuth(t *testing.T) {
	status, data, err := client.Get("/api/v1/notifications", "")
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, data)
}

func testNotificationMarkReadSuccess(t *testing.T) {
	// Get a notification to mark as read
	status, data, err := client.Get("/api/v1/notifications?is_read=false&page_size=1", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp NotificationsListResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	require.NotEmpty(t, resp.Items, "need at least one unread notification")

	notifID := resp.Items[0].ID

	status, data, err = client.Put(
		fmt.Sprintf("/api/v1/notifications/%s/read", notifID),
		nil,
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var notif NotificationResponse
	require.NoError(t, json.Unmarshal(data, &notif))
	assert.True(t, notif.IsRead)
	assert.Equal(t, notifID, notif.ID)
}

func testNotificationMarkReadNotFound(t *testing.T) {
	status, data, err := client.Put(
		"/api/v1/notifications/00000000-0000-0000-0000-000000000000/read",
		nil,
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, data)
}

func testNotificationMarkAllRead(t *testing.T) {
	// Use athlete2's notifications (untouched so far)
	status, data, err := client.Put("/api/v1/notifications/read-all", nil, athlete2Token)
	require.NoError(t, err)
	// Accept both 200 and 204
	assert.True(t, status == http.StatusOK || status == http.StatusNoContent,
		"expected 200 or 204, got %d: %s", status, string(data))

	// Verify all are now read
	status, data, err = client.Get("/api/v1/notifications", athlete2Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp NotificationsListResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.Equal(t, 0, resp.UnreadCount)
}

func testNotificationDeviceTokenSuccess(t *testing.T) {
	status, data, err := client.Post("/api/v1/notifications/device-token", map[string]string{
		"fcm_token":   "test-fcm-token-" + runSuffix,
		"device_info": "Integration Test Device",
	}, coach1Token)
	require.NoError(t, err)
	// Accept 200 or 201
	assert.True(t, status == http.StatusOK || status == http.StatusCreated,
		"expected 200 or 201, got %d: %s", status, string(data))
}

func testNotificationDeviceTokenEmpty(t *testing.T) {
	status, data, err := client.Post("/api/v1/notifications/device-token",
		map[string]string{},
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}
