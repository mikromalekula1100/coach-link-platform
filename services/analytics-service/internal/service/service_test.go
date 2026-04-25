package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coach-link/platform/services/analytics-service/internal/model"
	"github.com/coach-link/platform/services/analytics-service/internal/service"
)

// ── Mock ───────────────────────────────────────

type mockTrainingClient struct {
	reports         []model.ReportWithPlan
	stats           *model.AthleteStats
	athleteIDs      []string
	overviewStats   *model.CoachOverviewStats
	reportsErr      error
	statsErr        error
	athleteIDsErr   error
	overviewErr     error
}

func (m *mockTrainingClient) GetReports(_ context.Context, _, _, _ string) ([]model.ReportWithPlan, error) {
	return m.reports, m.reportsErr
}

func (m *mockTrainingClient) GetAthleteStats(_ context.Context, _ string) (*model.AthleteStats, error) {
	return m.stats, m.statsErr
}

func (m *mockTrainingClient) GetCoachAthleteIDs(_ context.Context, _ string) ([]string, error) {
	return m.athleteIDs, m.athleteIDsErr
}

func (m *mockTrainingClient) GetCoachOverview(_ context.Context, _ string) (*model.CoachOverviewStats, error) {
	return m.overviewStats, m.overviewErr
}

func newSvc(tc service.TrainingClient) *service.Service {
	return service.New(tc, zerolog.Nop())
}

// ── GetAthleteSummary ──────────────────────────

func TestGetAthleteSummary_Success(t *testing.T) {
	stats := &model.AthleteStats{
		TotalReports:       15,
		TotalDurationMin:   900,
		AvgDurationMin:     60.0,
		AvgPerceivedEffort: 7.5,
		AvgHeartRate:       160.0,
		TotalDistanceKm:    75.0,
		TotalAssignments:   18,
		CompletedCount:     15,
		CompletionRate:     83.3,
	}
	svc := newSvc(&mockTrainingClient{stats: stats})

	summary, err := svc.GetAthleteSummary(context.Background(), "athlete-1")
	require.NoError(t, err)
	assert.Equal(t, "athlete-1", summary.AthleteID)
	assert.Equal(t, 15, summary.TotalReports)
	assert.InDelta(t, 7.5, summary.AvgPerceivedEffort, 0.01)
	assert.InDelta(t, 83.3, summary.CompletionRate, 0.01)
}

func TestGetAthleteSummary_ClientError_ReturnsInternalError(t *testing.T) {
	svc := newSvc(&mockTrainingClient{statsErr: errors.New("db connection failed")})

	_, err := svc.GetAthleteSummary(context.Background(), "athlete-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 500, se.Status)
}

// ── GetAthleteProgress ─────────────────────────

func TestGetAthleteProgress_InvalidPeriod_ReturnsBadRequest(t *testing.T) {
	svc := newSvc(&mockTrainingClient{})

	_, err := svc.GetAthleteProgress(context.Background(), "athlete-1", "daily", "", "")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

func TestGetAthleteProgress_Week_GroupsCorrectly(t *testing.T) {
	reports := []model.ReportWithPlan{
		{ScheduledDate: "2026-04-14", DurationMinutes: 60, PerceivedEffort: 7},
		{ScheduledDate: "2026-04-15", DurationMinutes: 45, PerceivedEffort: 6},
		{ScheduledDate: "2026-04-21", DurationMinutes: 90, PerceivedEffort: 8},
	}
	svc := newSvc(&mockTrainingClient{reports: reports})

	resp, err := svc.GetAthleteProgress(context.Background(), "athlete-1", "week", "2026-04-14", "2026-04-21")
	require.NoError(t, err)
	assert.Equal(t, "athlete-1", resp.AthleteID)
	assert.Equal(t, "week", resp.Period)
	// Two weeks: 14-20 Apr and 21-27 Apr
	assert.Len(t, resp.Points, 2)
}

func TestGetAthleteProgress_Month_GroupsCorrectly(t *testing.T) {
	reports := []model.ReportWithPlan{
		{ScheduledDate: "2026-03-10", DurationMinutes: 60, PerceivedEffort: 7},
		{ScheduledDate: "2026-04-05", DurationMinutes: 70, PerceivedEffort: 8},
		{ScheduledDate: "2026-04-20", DurationMinutes: 50, PerceivedEffort: 6},
	}
	svc := newSvc(&mockTrainingClient{reports: reports})

	resp, err := svc.GetAthleteProgress(context.Background(), "athlete-1", "month", "", "")
	require.NoError(t, err)
	assert.Equal(t, "month", resp.Period)
	// Two months: March and April
	assert.Len(t, resp.Points, 2)
}

func TestGetAthleteProgress_EmptyReports_ReturnsEmptyPoints(t *testing.T) {
	svc := newSvc(&mockTrainingClient{reports: []model.ReportWithPlan{}})

	resp, err := svc.GetAthleteProgress(context.Background(), "athlete-1", "week", "", "")
	require.NoError(t, err)
	assert.Empty(t, resp.Points)
}

func TestGetAthleteProgress_DefaultPeriodIsWeek(t *testing.T) {
	svc := newSvc(&mockTrainingClient{reports: []model.ReportWithPlan{}})

	resp, err := svc.GetAthleteProgress(context.Background(), "athlete-1", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, "week", resp.Period)
}

func TestGetAthleteProgress_PointsSortedByPeriodStart(t *testing.T) {
	reports := []model.ReportWithPlan{
		{ScheduledDate: "2026-04-21", DurationMinutes: 90, PerceivedEffort: 8},
		{ScheduledDate: "2026-04-07", DurationMinutes: 60, PerceivedEffort: 7},
		{ScheduledDate: "2026-04-14", DurationMinutes: 45, PerceivedEffort: 6},
	}
	svc := newSvc(&mockTrainingClient{reports: reports})

	resp, err := svc.GetAthleteProgress(context.Background(), "athlete-1", "week", "", "")
	require.NoError(t, err)
	require.Len(t, resp.Points, 3)
	assert.LessOrEqual(t, resp.Points[0].PeriodStart, resp.Points[1].PeriodStart)
	assert.LessOrEqual(t, resp.Points[1].PeriodStart, resp.Points[2].PeriodStart)
}

func TestGetAthleteProgress_ReportsErr_ReturnsInternalError(t *testing.T) {
	svc := newSvc(&mockTrainingClient{reportsErr: errors.New("timeout")})

	_, err := svc.GetAthleteProgress(context.Background(), "athlete-1", "week", "", "")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 500, se.Status)
}

// ── GetCoachOverview ───────────────────────────

func TestGetCoachOverview_Success(t *testing.T) {
	overview := &model.CoachOverviewStats{TotalAthletes: 3, TotalAssignments: 30, TotalReports: 25}
	stats := &model.AthleteStats{TotalAssignments: 10, CompletedCount: 8, CompletionRate: 80.0, TotalReports: 8}

	svc := newSvc(&mockTrainingClient{
		overviewStats: overview,
		athleteIDs:    []string{"a1", "a2", "a3"},
		stats:         stats,
	})

	resp, err := svc.GetCoachOverview(context.Background(), "coach-1")
	require.NoError(t, err)
	assert.Equal(t, 3, resp.TotalAthletes)
	assert.Len(t, resp.Athletes, 3)
}

func TestGetCoachOverview_OverviewError_ReturnsInternalError(t *testing.T) {
	svc := newSvc(&mockTrainingClient{overviewErr: errors.New("service down")})

	_, err := svc.GetCoachOverview(context.Background(), "coach-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 500, se.Status)
}
