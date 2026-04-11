package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAnalyticsMeSummary(t *testing.T) {
	status, data, err := client.Get("/api/v1/analytics/me/summary", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var summary map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &summary))
	assert.Equal(t, athlete1ID, summary["athlete_id"])
	// athlete1 has submitted at least 1 report in training tests
	assert.GreaterOrEqual(t, int(summary["total_reports"].(float64)), 1)
}

func testAnalyticsMeProgress(t *testing.T) {
	status, data, err := client.Get("/api/v1/analytics/me/progress?period=week", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var progress map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &progress))
	assert.Equal(t, athlete1ID, progress["athlete_id"])
	assert.Equal(t, "week", progress["period"])
	_, ok := progress["points"].([]interface{})
	assert.True(t, ok, "points should be an array")
}

func testAnalyticsAthleteSummaryAsCoach(t *testing.T) {
	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/analytics/athletes/%s/summary", athlete1ID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var summary map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &summary))
	assert.Equal(t, athlete1ID, summary["athlete_id"])
}

func testAnalyticsAthleteProgressAsCoach(t *testing.T) {
	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/analytics/athletes/%s/progress?period=month", athlete1ID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var progress map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &progress))
	assert.Equal(t, "month", progress["period"])
}

func testAnalyticsOverview(t *testing.T) {
	status, data, err := client.Get("/api/v1/analytics/overview", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var overview map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &overview))
	assert.GreaterOrEqual(t, int(overview["total_athletes"].(float64)), 1)
}

func testAnalyticsOverviewAsAthlete(t *testing.T) {
	status, data, err := client.Get("/api/v1/analytics/overview", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testAnalyticsNoAuth(t *testing.T) {
	status, data, err := client.Get("/api/v1/analytics/me/summary", "")
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, data)
}
