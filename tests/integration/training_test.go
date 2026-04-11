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

func testTrainingCreatePlanWithAthleteIDs(t *testing.T) {
	scheduledDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	status, data, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"title":          "Morning Run",
		"description":    "Easy 5km morning run at conversational pace",
		"scheduled_date": scheduledDate,
		"athlete_ids":    []string{athlete1ID},
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var resp CreateTrainingPlanResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.NotEmpty(t, resp.Plan.ID)
	assert.Equal(t, "Morning Run", resp.Plan.Title)
	assert.Equal(t, scheduledDate, resp.Plan.ScheduledDate)
	require.Len(t, resp.Assignments, 1)
	assert.Equal(t, athlete1ID, resp.Assignments[0].AthleteID)
	assert.NotEmpty(t, resp.Assignments[0].ID)

	// Store the first assignment for report/archive tests
	assignmentID = resp.Assignments[0].ID
}

func testTrainingCreatePlanWithGroupID(t *testing.T) {
	require.NotEmpty(t, groupID, "depends on group tests")

	scheduledDate := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	status, data, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"title":          "Group Interval Session",
		"description":    "400m intervals x8 with 90s rest",
		"scheduled_date": scheduledDate,
		"group_id":       groupID,
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var resp CreateTrainingPlanResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	// Group has athlete1 (athlete2 was removed in group tests)
	assert.GreaterOrEqual(t, len(resp.Assignments), 1)
}

func testTrainingCreatePlanWithSaveAsTemplate(t *testing.T) {
	scheduledDate := time.Now().Add(72 * time.Hour).Format("2006-01-02")
	status, data, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"title":            "Tempo Run",
		"description":      "30 min tempo run at threshold pace",
		"scheduled_date":   scheduledDate,
		"athlete_ids":      []string{athlete2ID},
		"save_as_template": true,
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var resp CreateTrainingPlanResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	require.NotNil(t, resp.Template, "template should be created")
	assert.NotEmpty(t, resp.Template.ID)
	assert.Equal(t, "Tempo Run", resp.Template.Title)
}

func testTrainingCreatePlanAsAthlete(t *testing.T) {
	status, data, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"title":          "Forbidden Plan",
		"description":    "Should not work",
		"scheduled_date": "2026-12-01",
		"athlete_ids":    []string{athlete1ID},
	}, athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testTrainingCreatePlanNoTargets(t *testing.T) {
	status, data, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"title":          "No Targets",
		"description":    "No athlete_ids or group_id",
		"scheduled_date": "2026-12-01",
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testTrainingCreatePlanMissingTitle(t *testing.T) {
	status, data, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"description":    "Missing title",
		"scheduled_date": "2026-12-01",
		"athlete_ids":    []string{athlete1ID},
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testTrainingCreatePlanInvalidDate(t *testing.T) {
	status, data, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"title":          "Bad Date",
		"description":    "Invalid date format",
		"scheduled_date": "not-a-date",
		"athlete_ids":    []string{athlete1ID},
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testTrainingGetAssignmentsAthlete(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on CreatePlan_WithAthleteIDs")

	status, data, err := client.Get("/api/v1/training/assignments", athlete1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedAssignments
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 1)

	// Find our assignment
	found := false
	for _, a := range resp.Items {
		if a.ID == assignmentID {
			found = true
			assert.Equal(t, "assigned", a.Status)
			assert.False(t, a.HasReport)
			assert.Equal(t, "Morning Run", a.Title)
		}
	}
	assert.True(t, found, "expected to find the assignment in athlete's list")
}

func testTrainingGetAssignmentsCoach(t *testing.T) {
	status, data, err := client.Get("/api/v1/training/assignments", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedAssignments
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 1)
}

func testTrainingGetAssignmentSuccess(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on CreatePlan_WithAthleteIDs")

	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/training/assignments/%s", assignmentID),
		athlete1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var detail AssignmentDetail
	require.NoError(t, json.Unmarshal(data, &detail))
	assert.Equal(t, assignmentID, detail.ID)
	assert.Equal(t, "Morning Run", detail.Title)
	assert.Equal(t, "Easy 5km morning run at conversational pace", detail.Description)
	assert.Equal(t, athlete1ID, detail.AthleteID)
	assert.Equal(t, "assigned", detail.Status)
}

func testTrainingGetAssignmentOtherAthlete(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on CreatePlan_WithAthleteIDs")

	// athlete2 should not see athlete1's assignment
	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/training/assignments/%s", assignmentID),
		athlete2Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testTrainingGetAssignmentNotFound(t *testing.T) {
	status, data, err := client.Get(
		"/api/v1/training/assignments/00000000-0000-0000-0000-000000000000",
		athlete1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, data)
}

func testTrainingSubmitReportSuccess(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on CreatePlan_WithAthleteIDs")

	maxHR := 185
	avgHR := 155
	distance := 5.2
	status, data, err := client.Post(
		fmt.Sprintf("/api/v1/training/assignments/%s/report", assignmentID),
		map[string]interface{}{
			"content":          "Completed the morning run. Felt good overall, slight fatigue in the last km.",
			"duration_minutes": 28,
			"perceived_effort": 6,
			"max_heart_rate":   maxHR,
			"avg_heart_rate":   avgHR,
			"distance_km":      distance,
		},
		athlete1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var report TrainingReportResponse
	require.NoError(t, json.Unmarshal(data, &report))
	assert.NotEmpty(t, report.ID)
	assert.Equal(t, assignmentID, report.AssignmentID)
	assert.Equal(t, athlete1ID, report.AthleteID)
	assert.Equal(t, 28, report.DurationMinutes)
	assert.Equal(t, 6, report.PerceivedEffort)
	assert.NotNil(t, report.MaxHeartRate)
	assert.Equal(t, maxHR, *report.MaxHeartRate)
	assert.NotNil(t, report.AvgHeartRate)
	assert.Equal(t, avgHR, *report.AvgHeartRate)
	assert.NotNil(t, report.DistanceKm)
	assert.InDelta(t, distance, *report.DistanceKm, 0.01)
}

func testTrainingSubmitReportAsCoach(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on CreatePlan_WithAthleteIDs")

	status, data, err := client.Post(
		fmt.Sprintf("/api/v1/training/assignments/%s/report", assignmentID),
		map[string]interface{}{
			"content":          "Coach cannot submit",
			"duration_minutes": 30,
			"perceived_effort": 5,
		},
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusForbidden, status, data)
}

func testTrainingSubmitReportDuplicate(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on SubmitReport_Success")

	status, data, err := client.Post(
		fmt.Sprintf("/api/v1/training/assignments/%s/report", assignmentID),
		map[string]interface{}{
			"content":          "Duplicate report",
			"duration_minutes": 30,
			"perceived_effort": 5,
		},
		athlete1Token,
	)
	require.NoError(t, err)
	// assignment is now "completed", so submitting again should fail
	// Could be 409 REPORT_ALREADY_EXISTS or 400 ASSIGNMENT_NOT_ASSIGNED
	assert.True(t, status == http.StatusConflict || status == http.StatusBadRequest,
		"expected 409 or 400, got %d: %s", status, string(data))
}

func testTrainingGetReportAsAthlete(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on SubmitReport_Success")

	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/training/assignments/%s/report", assignmentID),
		athlete1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var report TrainingReportResponse
	require.NoError(t, json.Unmarshal(data, &report))
	assert.Equal(t, assignmentID, report.AssignmentID)
	assert.Equal(t, 28, report.DurationMinutes)
}

func testTrainingGetReportAsCoach(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on SubmitReport_Success")

	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/training/assignments/%s/report", assignmentID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var report TrainingReportResponse
	require.NoError(t, json.Unmarshal(data, &report))
	assert.Equal(t, assignmentID, report.AssignmentID)
}

func testTrainingAssignmentStatusCompleted(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on SubmitReport_Success")

	status, data, err := client.Get(
		fmt.Sprintf("/api/v1/training/assignments/%s", assignmentID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var detail AssignmentDetail
	require.NoError(t, json.Unmarshal(data, &detail))
	assert.Equal(t, "completed", detail.Status)
	assert.True(t, detail.HasReport)
	assert.NotNil(t, detail.CompletedAt)
}

func testTrainingArchiveSuccess(t *testing.T) {
	require.NotEmpty(t, assignmentID, "depends on SubmitReport_Success (status=completed)")

	status, data, err := client.Put(
		fmt.Sprintf("/api/v1/training/assignments/%s/archive", assignmentID),
		nil,
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var result map[string]string
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "archived", result["status"])
}

func testTrainingArchiveNotCompleted(t *testing.T) {
	// Create a fresh plan with "assigned" status, then try to archive
	scheduledDate := time.Now().Add(96 * time.Hour).Format("2006-01-02")
	createStatus, createData, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"title":          "Archive Test Plan",
		"description":    "This plan will not be completed",
		"scheduled_date": scheduledDate,
		"athlete_ids":    []string{athlete2ID},
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, createStatus, createData)

	var createResp CreateTrainingPlanResponse
	require.NoError(t, json.Unmarshal(createData, &createResp))
	require.Len(t, createResp.Assignments, 1)

	freshAssignmentID := createResp.Assignments[0].ID

	status, data, err := client.Put(
		fmt.Sprintf("/api/v1/training/assignments/%s/archive", freshAssignmentID),
		nil,
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)

	svcErr, err := parseServiceError(data)
	require.NoError(t, err)
	assert.Equal(t, "ASSIGNMENT_NOT_COMPLETED", svcErr.Error.Code)
}

func testTrainingGetArchived(t *testing.T) {
	status, data, err := client.Get("/api/v1/training/assignments/archived", coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp PaginatedAssignments
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.GreaterOrEqual(t, resp.Pagination.TotalItems, 1)
}

func testTrainingDeleteAssignmentSuccess(t *testing.T) {
	// Create a fresh plan then delete the assignment
	scheduledDate := time.Now().Add(120 * time.Hour).Format("2006-01-02")
	createStatus, createData, err := client.Post("/api/v1/training/plans", map[string]interface{}{
		"title":          "Delete Test Plan",
		"description":    "This assignment will be deleted",
		"scheduled_date": scheduledDate,
		"athlete_ids":    []string{athlete1ID},
	}, coach1Token)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, createStatus, createData)

	var createResp CreateTrainingPlanResponse
	require.NoError(t, json.Unmarshal(createData, &createResp))
	require.Len(t, createResp.Assignments, 1)

	deleteID := createResp.Assignments[0].ID

	status, data, err := client.Delete(
		fmt.Sprintf("/api/v1/training/assignments/%s", deleteID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNoContent, status, data)

	// Verify it's gone
	status, data, err = client.Get(
		fmt.Sprintf("/api/v1/training/assignments/%s", deleteID),
		coach1Token,
	)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, data)
}
