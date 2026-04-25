package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coach-link/platform/services/ai-service/internal/model"
	"github.com/coach-link/platform/services/ai-service/internal/service"
)

// ── Mocks ──────────────────────────────────────

type mockAnalyticsClient struct {
	summary *model.AthleteSummary
	reports []model.ReportWithPlan
	err     error
}

func (m *mockAnalyticsClient) GetAthleteSummary(_ context.Context, _ string) (*model.AthleteSummary, error) {
	return m.summary, m.err
}

func (m *mockAnalyticsClient) GetAthleteReports(_ context.Context, _ string) ([]model.ReportWithPlan, error) {
	return m.reports, m.err
}

type mockOllamaClient struct {
	content string
	err     error
}

func (m *mockOllamaClient) Generate(_ context.Context, _, _ string) (string, error) {
	return m.content, m.err
}

func (m *mockOllamaClient) Model() string { return "gemma3:4b" }

// capturingOllamaClient records the prompts it receives so tests can assert on them.
type capturingOllamaClient struct {
	lastSystemPrompt string
	lastUserPrompt   string
}

func (c *capturingOllamaClient) Generate(_ context.Context, sys, user string) (string, error) {
	c.lastSystemPrompt = sys
	c.lastUserPrompt = user
	return "generated text", nil
}

func (c *capturingOllamaClient) Model() string { return "gemma3:4b" }

// ── Helpers ────────────────────────────────────

func makeSummary() *model.AthleteSummary {
	return &model.AthleteSummary{
		AthleteID:          "athlete-1",
		TotalReports:       20,
		TotalDurationMin:   1200,
		TotalDistanceKm:    100.0,
		AvgDurationMin:     60.0,
		AvgPerceivedEffort: 7.0,
		AvgHeartRate:       155.0,
		CompletionRate:     0.9, // 90% — stored as fraction 0–1
		TotalAssignments:   22,
	}
}

func makeReports(n int) []model.ReportWithPlan {
	reports := make([]model.ReportWithPlan, n)
	for i := range reports {
		reports[i] = model.ReportWithPlan{
			ID:              fmt.Sprintf("report-%d", i),
			AthleteID:       "athlete-1",
			DurationMinutes: 60,
			PerceivedEffort: (i % 10) + 1,
			Title:           fmt.Sprintf("Тренировка %d", i+1),
			ScheduledDate:   "2026-04-20",
		}
	}
	return reports
}

func newSvc(ac service.AnalyticsClient, oc service.OllamaClient) *service.Service {
	return service.New(ac, oc, zerolog.Nop())
}

// ── Tests ──────────────────────────────────────

func TestGenerateRecommendations_Success(t *testing.T) {
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: makeReports(3)},
		&mockOllamaClient{content: "Рекомендации по тренировкам"},
	)

	resp, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.NoError(t, err)
	assert.Equal(t, "athlete-1", resp.AthleteID)
	assert.Equal(t, "recommendations", resp.Type)
	assert.Equal(t, "Рекомендации по тренировкам", resp.Content)
	assert.Equal(t, "gemma3:4b", resp.Model)
	assert.NotEmpty(t, resp.GeneratedAt)
}

func TestGenerateRecommendations_NoReports_Returns400(t *testing.T) {
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: []model.ReportWithPlan{}},
		&mockOllamaClient{},
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
	assert.Equal(t, "VALIDATION_ERROR", se.Code)
}

func TestGenerateRecommendations_AnalyticsError_Returns500(t *testing.T) {
	svc := newSvc(
		&mockAnalyticsClient{err: errors.New("connection refused")},
		&mockOllamaClient{},
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 500, se.Status)
	assert.Equal(t, "INTERNAL_ERROR", se.Code)
}

func TestGenerateRecommendations_OllamaUnavailable_Returns503(t *testing.T) {
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: makeReports(3)},
		&mockOllamaClient{err: errors.New("dial tcp: connection refused")},
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 503, se.Status)
	assert.Equal(t, "SERVICE_UNAVAILABLE", se.Code)
}

func TestGenerateRecommendations_LimitsTo5Reports(t *testing.T) {
	ollama := &capturingOllamaClient{}
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: makeReports(10)},
		ollama,
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.NoError(t, err)

	// Prompt must say "5 тренировок", not "10 тренировок"
	assert.Contains(t, ollama.lastUserPrompt, "Последние 5 тренировок")
	// First reports (1-5) are excluded — only the last 5 appear
	assert.NotContains(t, ollama.lastUserPrompt, "Тренировка 1\n")
}

func TestGenerateRecommendations_WhenExactly5Reports_AllUsed(t *testing.T) {
	ollama := &capturingOllamaClient{}
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: makeReports(5)},
		ollama,
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.NoError(t, err)
	assert.Contains(t, ollama.lastUserPrompt, "Последние 5 тренировок")
}

func TestGenerateRecommendations_CoachContextAppearsInPrompt(t *testing.T) {
	ollama := &capturingOllamaClient{}
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: makeReports(2)},
		ollama,
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "Готовимся к чемпионату")
	require.NoError(t, err)
	assert.Contains(t, ollama.lastUserPrompt, "Готовимся к чемпионату")
	assert.Contains(t, ollama.lastUserPrompt, "Контекст от тренера")
}

func TestGenerateRecommendations_NoContextBlock_WhenContextEmpty(t *testing.T) {
	ollama := &capturingOllamaClient{}
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: makeReports(2)},
		ollama,
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.NoError(t, err)
	assert.NotContains(t, ollama.lastUserPrompt, "Контекст от тренера")
}

func TestGenerateRecommendations_PromptContainsSummaryStats(t *testing.T) {
	ollama := &capturingOllamaClient{}
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: makeReports(2)},
		ollama,
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.NoError(t, err)
	assert.Contains(t, ollama.lastUserPrompt, "Общая статистика спортсмена")
	assert.Contains(t, ollama.lastUserPrompt, "90") // CompletionRate
}

func TestGenerateRecommendations_SystemPromptMentionsAnalysisAndRecommendations(t *testing.T) {
	ollama := &capturingOllamaClient{}
	svc := newSvc(
		&mockAnalyticsClient{summary: makeSummary(), reports: makeReports(2)},
		ollama,
	)

	_, err := svc.GenerateRecommendations(context.Background(), "athlete-1", "")
	require.NoError(t, err)

	lower := strings.ToLower(ollama.lastSystemPrompt)
	assert.True(t,
		strings.Contains(lower, "тенденц") || strings.Contains(lower, "анализ") || strings.Contains(lower, "наблюден"),
		"system prompt must include analysis/trends instruction")
	assert.True(t,
		strings.Contains(lower, "рекоменд"),
		"system prompt must include recommendations instruction")
}
