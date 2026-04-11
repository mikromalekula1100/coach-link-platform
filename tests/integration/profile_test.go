package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProfileGetMeCoach(t *testing.T) {
	status, data, err := client.Get("/api/v1/users/me", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var profile UserProfileResponse
	require.NoError(t, json.Unmarshal(data, &profile))
	assert.Equal(t, coach1ID, profile.ID)
	assert.Equal(t, coach1Login, profile.Login)
	assert.Equal(t, "coach", profile.Role)
	assert.Equal(t, "Integration Coach One", profile.FullName)
	assert.NotEmpty(t, profile.Email)
}

func testProfileGetMeAthlete(t *testing.T) {
	status, data, err := client.Get("/api/v1/users/me", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var profile UserProfileResponse
	require.NoError(t, json.Unmarshal(data, &profile))
	assert.Equal(t, athlete1ID, profile.ID)
	assert.Equal(t, "athlete", profile.Role)
}

func testProfileGetMeNoAuth(t *testing.T) {
	status, data, err := client.Get("/api/v1/users/me", "")
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, data)
}

func testProfileSearchByLogin(t *testing.T) {
	q := url.QueryEscape(coach1Login)
	status, data, err := client.Get("/api/v1/users/search?q="+q+"&role=coach", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedSearchResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 1)

	found := false
	for _, u := range resp.Items {
		if u.ID == coach1ID {
			found = true
			break
		}
	}
	assert.True(t, found, "expected to find coach1 in search results")
}

func testProfileSearchByRole(t *testing.T) {
	// Search for athletes with prefix "int"
	status, data, err := client.Get("/api/v1/users/search?q=intathlete&role=athlete", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedSearchResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 3)
	for _, u := range resp.Items {
		assert.Equal(t, "athlete", u.Role)
	}
}

func testProfileSearchNoResults(t *testing.T) {
	status, data, err := client.Get("/api/v1/users/search?q=nonexistent-zzz-xyz-2024", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedSearchResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.Equal(t, 0, resp.Pagination.TotalItems)
}

func testProfileSearchPagination(t *testing.T) {
	status, data, err := client.Get("/api/v1/users/search?q=int&page=1&page_size=1", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedSearchResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.LessOrEqual(t, len(resp.Items), 1)
	assert.Equal(t, 1, resp.Pagination.Page)
	assert.Equal(t, 1, resp.Pagination.PageSize)
}
