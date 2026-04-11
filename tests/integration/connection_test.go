package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConnectionGetCoachConnected(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/coach", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var coach CoachInfo
	require.NoError(t, json.Unmarshal(data, &coach))
	assert.Equal(t, coach1ID, coach.ID)
	assert.Equal(t, coach1Login, coach.Login)
	assert.Equal(t, "Integration Coach One", coach.FullName)
	assert.NotEmpty(t, coach.ConnectedAt)
}

func testConnectionGetCoachUnconnected(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/coach", athlete3Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, data)
}

func testConnectionGetCoachAsCoach(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/coach", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testConnectionGetAthletesCoach(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/athletes", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedAthletes
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.Equal(t, 2, resp.Pagination.TotalItems)

	ids := map[string]bool{}
	for _, a := range resp.Items {
		ids[a.ID] = true
		assert.NotEmpty(t, a.Login)
		assert.NotEmpty(t, a.FullName)
	}
	assert.True(t, ids[athlete1ID], "athlete1 should be in list")
	assert.True(t, ids[athlete2ID], "athlete2 should be in list")
}

func testConnectionGetAthletesAsAthlete(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/athletes", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testConnectionRequestAlreadyHasCoach(t *testing.T) {
	// athlete1 already has coach1, trying to request coach2
	status, data, err := client.Post("/api/v1/connections/request", map[string]string{
		"coach_id": coach2ID,
	}, athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusConflict, status, data)

	svcErr, err := parseServiceError(data)
	require.NoError(t, err)
	assert.Equal(t, "ALREADY_HAS_COACH", svcErr.Error.Code)
}

func testConnectionRequestCoachSends(t *testing.T) {
	status, data, err := client.Post("/api/v1/connections/request", map[string]string{
		"coach_id": coach2ID,
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testConnectionRequestTargetNotCoach(t *testing.T) {
	// athlete3 sends request to athlete1 (not a coach)
	status, data, err := client.Post("/api/v1/connections/request", map[string]string{
		"coach_id": athlete1ID,
	}, athlete3Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testConnectionRequestSuccess(t *testing.T) {
	// athlete3 → coach2
	status, data, err := client.Post("/api/v1/connections/request", map[string]string{
		"coach_id": coach2ID,
	}, athlete3Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var cr ConnectionRequestResponse
	require.NoError(t, json.Unmarshal(data, &cr))
	assert.Equal(t, "pending", cr.Status)
	assert.Equal(t, athlete3ID, cr.Athlete.ID)
	assert.Equal(t, coach2ID, cr.Coach.ID)
	assert.NotEmpty(t, cr.ID)

	// Store for later tests
	connectionRequestID = cr.ID
}

func testConnectionRequestDuplicate(t *testing.T) {
	require.NotEmpty(t, connectionRequestID, "depends on Request_Success")

	status, data, err := client.Post("/api/v1/connections/request", map[string]string{
		"coach_id": coach2ID,
	}, athlete3Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusConflict, status, data)

	svcErr, err := parseServiceError(data)
	require.NoError(t, err)
	assert.Equal(t, "REQUEST_ALREADY_EXISTS", svcErr.Error.Code)
}

func testConnectionOutgoingAthlete(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/requests/outgoing", athlete3Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var cr ConnectionRequestResponse
	require.NoError(t, json.Unmarshal(data, &cr))
	assert.Equal(t, "pending", cr.Status)
	assert.Equal(t, coach2ID, cr.Coach.ID)
}

func testConnectionOutgoingAsCoach(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/requests/outgoing", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testConnectionIncomingCoach(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/requests/incoming", coach2Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedConnectionRequests
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 1)

	found := false
	for _, cr := range resp.Items {
		if cr.ID == connectionRequestID {
			found = true
			assert.Equal(t, "pending", cr.Status)
		}
	}
	assert.True(t, found, "expected to find the pending request")
}

func testConnectionIncomingAsAthlete(t *testing.T) {
	status, data, err := client.Get("/api/v1/connections/requests/incoming", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testConnectionRejectSuccess(t *testing.T) {
	require.NotEmpty(t, connectionRequestID, "depends on Request_Success")

	status, data, err := client.Put(
		fmt.Sprintf("/api/v1/connections/requests/%s/reject", connectionRequestID),
		nil,
		coach2Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var cr ConnectionRequestResponse
	require.NoError(t, json.Unmarshal(data, &cr))
	assert.Equal(t, "rejected", cr.Status)
}

func testConnectionRejectAlreadyRejected(t *testing.T) {
	require.NotEmpty(t, connectionRequestID, "depends on Reject_Success")

	status, data, err := client.Put(
		fmt.Sprintf("/api/v1/connections/requests/%s/reject", connectionRequestID),
		nil,
		coach2Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}
