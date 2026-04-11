package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AI tests may fail if Ollama is not running or model is not loaded.
// We test API contract (auth, validation) and accept 503 as valid for AI generation.

func testAIRecommendationsAsCoach(t *testing.T) {
	// AI generation on CPU can take minutes — use a dedicated client with longer timeout
	aiClient := NewAPIClient(getBaseURL())
	aiClient.httpClient.Timeout = 6 * time.Minute

	status, data, err := aiClient.Post(
		fmt.Sprintf("/api/v1/ai/athletes/%s/recommendations", athlete1ID),
		map[string]string{
			"context": "Подготовка к соревнованиям на 800м через 2 месяца",
		},
		coach1Token,
	)
	require.NoError(t, err)
	// Accept 200 (Ollama working) or 503 (Ollama not available)
	assert.True(t, status == http.StatusOK || status == http.StatusServiceUnavailable,
		"expected 200 or 503, got %d: %s", status, string(data))

	if status == http.StatusOK {
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &resp))
		assert.Equal(t, athlete1ID, resp["athlete_id"])
		assert.Equal(t, "recommendations", resp["type"])
		assert.NotEmpty(t, resp["content"])
		assert.NotEmpty(t, resp["model"])
	}
}

func testAIAnalysisAsCoach(t *testing.T) {
	aiClient := NewAPIClient(getBaseURL())
	aiClient.httpClient.Timeout = 6 * time.Minute

	status, data, err := aiClient.Post(
		fmt.Sprintf("/api/v1/ai/athletes/%s/analysis", athlete1ID),
		nil,
		coach1Token,
	)
	require.NoError(t, err)
	assert.True(t, status == http.StatusOK || status == http.StatusServiceUnavailable,
		"expected 200 or 503, got %d: %s", status, string(data))

	if status == http.StatusOK {
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &resp))
		assert.Equal(t, "analysis", resp["type"])
	}
}

func testAIRecommendationsAsAthlete(t *testing.T) {
	status, data, err := client.Post(
		fmt.Sprintf("/api/v1/ai/athletes/%s/recommendations", athlete1ID),
		nil,
		athlete1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testAIRecommendationsNoAuth(t *testing.T) {
	status, data, err := client.Post(
		fmt.Sprintf("/api/v1/ai/athletes/%s/recommendations", athlete1ID),
		nil,
		"",
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, data)
}
