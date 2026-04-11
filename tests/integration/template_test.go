package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTemplateCreateSuccess(t *testing.T) {
	status, data, err := client.Post("/api/v1/training/templates", map[string]string{
		"title":       "Sprint Drills Template",
		"description": "100m sprint drills with warm-up and cool-down",
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var tmpl TemplateResponse
	require.NoError(t, json.Unmarshal(data, &tmpl))
	assert.NotEmpty(t, tmpl.ID)
	assert.Equal(t, "Sprint Drills Template", tmpl.Title)
	assert.Equal(t, "100m sprint drills with warm-up and cool-down", tmpl.Description)
	assert.NotEmpty(t, tmpl.CreatedAt)
	assert.NotEmpty(t, tmpl.UpdatedAt)

	templateID = tmpl.ID
}

func testTemplateCreateAsAthlete(t *testing.T) {
	status, data, err := client.Post("/api/v1/training/templates", map[string]string{
		"title":       "Forbidden Template",
		"description": "Should not work",
	}, athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testTemplateCreateMissingTitle(t *testing.T) {
	status, data, err := client.Post("/api/v1/training/templates", map[string]string{
		"description": "No title provided",
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testTemplateListSuccess(t *testing.T) {
	require.NotEmpty(t, templateID, "depends on Create_Success")

	status, data, err := client.Get("/api/v1/training/templates", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedTemplates
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 1)
}

func testTemplateListAsAthlete(t *testing.T) {
	status, data, err := client.Get("/api/v1/training/templates", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testTemplateGetSuccess(t *testing.T) {
	require.NotEmpty(t, templateID, "depends on Create_Success")

	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/training/templates/%s", templateID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var tmpl TemplateResponse
	require.NoError(t, json.Unmarshal(data, &tmpl))
	assert.Equal(t, templateID, tmpl.ID)
	assert.Equal(t, "Sprint Drills Template", tmpl.Title)
}

func testTemplateGetOtherCoach(t *testing.T) {
	require.NotEmpty(t, templateID, "depends on Create_Success")

	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/training/templates/%s", templateID),
		coach2Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testTemplateGetNotFound(t *testing.T) {
	status, data, err := client.Get(
		"/api/v1/training/templates/00000000-0000-0000-0000-000000000000",
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, data)
}

func testTemplateUpdateSuccess(t *testing.T) {
	require.NotEmpty(t, templateID, "depends on Create_Success")

	status, data, err := client.Put(
		fmt.Sprintf("/api/v1/training/templates/%s", templateID),
		map[string]string{
			"title": "Updated Sprint Drills",
		},
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var tmpl TemplateResponse
	require.NoError(t, json.Unmarshal(data, &tmpl))
	assert.Equal(t, "Updated Sprint Drills", tmpl.Title)
}

func testTemplateUpdateEmptyBody(t *testing.T) {
	require.NotEmpty(t, templateID, "depends on Create_Success")

	status, data, err := client.Put(
		fmt.Sprintf("/api/v1/training/templates/%s", templateID),
		map[string]interface{}{},
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testTemplateDeleteSuccess(t *testing.T) {
	// Create a throwaway template to delete
	status, data, err := client.Post("/api/v1/training/templates", map[string]string{
		"title":       "Throwaway Template",
		"description": "Will be deleted",
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var tmpl TemplateResponse
	require.NoError(t, json.Unmarshal(data, &tmpl))

	status, data, err = client.Delete(
		fmt.Sprintf("/api/v1/training/templates/%s", tmpl.ID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNoContent, status, data)
}

func testTemplateDeleteNotFound(t *testing.T) {
	status, data, err := client.Delete(
		"/api/v1/training/templates/00000000-0000-0000-0000-000000000000",
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, data)
}
