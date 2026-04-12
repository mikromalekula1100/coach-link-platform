package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/coach-link/platform/services/ai-service/internal/client"
	"github.com/coach-link/platform/services/ai-service/internal/model"
)

// ──────────────────────────────────────────────
// Error types
// ──────────────────────────────────────────────

type ServiceError struct {
	Code    string
	Message string
	Status  int
}

func (e *ServiceError) Error() string { return e.Message }

func IsServiceError(err error) (*ServiceError, bool) {
	if se, ok := err.(*ServiceError); ok {
		return se, true
	}
	return nil, false
}

type athleteGroup struct {
	FullName string
	Login    string
	Reports  []model.ReportWithPlan
}

func badRequest(msg string) *ServiceError {
	return &ServiceError{Code: "VALIDATION_ERROR", Message: msg, Status: 400}
}

func internalError(msg string) *ServiceError {
	return &ServiceError{Code: "INTERNAL_ERROR", Message: msg, Status: 500}
}

func serviceUnavailable(msg string) *ServiceError {
	return &ServiceError{Code: "SERVICE_UNAVAILABLE", Message: msg, Status: 503}
}

// ──────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────

type Service struct {
	analyticsClient *client.AnalyticsClient
	trainingClient  *client.TrainingClient
	ollamaClient    *client.OllamaClient
	log             zerolog.Logger
}

func New(analyticsClient *client.AnalyticsClient, trainingClient *client.TrainingClient, ollamaClient *client.OllamaClient, log zerolog.Logger) *Service {
	return &Service{
		analyticsClient: analyticsClient,
		trainingClient:  trainingClient,
		ollamaClient:    ollamaClient,
		log:             log,
	}
}

// GenerateRecommendations produces training recommendations for the given athlete.
func (s *Service) GenerateRecommendations(ctx context.Context, athleteID, coachContext string) (*model.AIResponse, error) {
	summary, reports, err := s.fetchAthleteData(ctx, athleteID)
	if err != nil {
		return nil, err
	}

	systemPrompt := "Ты — ассистент тренера по лёгкой атлетике. На основе данных о тренировках спортсмена дай конкретные рекомендации по тренировочному процессу.\nОтвечай на русском языке. Будь конкретным и практичным."

	userPrompt := s.buildUserPrompt(summary, reports, coachContext, "Дай рекомендации по дальнейшему тренировочному процессу")

	content, err := s.ollamaClient.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		s.log.Error().Err(err).Str("athlete_id", athleteID).Msg("ollama generation failed")
		return nil, serviceUnavailable("AI service temporarily unavailable")
	}

	return &model.AIResponse{
		AthleteID:   athleteID,
		Type:        "recommendations",
		Content:     content,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Model:       s.ollamaClient.Model(),
	}, nil
}

// GenerateAnalysis produces a training analysis for the given athlete.
func (s *Service) GenerateAnalysis(ctx context.Context, athleteID, coachContext string) (*model.AIResponse, error) {
	summary, reports, err := s.fetchAthleteData(ctx, athleteID)
	if err != nil {
		return nil, err
	}

	systemPrompt := "Проанализируй тренировочные данные спортсмена. Выяви тенденции, сильные и слабые стороны.\nОтвечай на русском языке. Будь конкретным и практичным."

	userPrompt := s.buildUserPrompt(summary, reports, coachContext, "Проанализируй тренировочный процесс этого спортсмена")

	content, err := s.ollamaClient.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		s.log.Error().Err(err).Str("athlete_id", athleteID).Msg("ollama generation failed")
		return nil, serviceUnavailable("AI service temporarily unavailable")
	}

	return &model.AIResponse{
		AthleteID:   athleteID,
		Type:        "analysis",
		Content:     content,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Model:       s.ollamaClient.Model(),
	}, nil
}

// GenerateSummary produces a digest of all athletes' reports for the coach over a date range.
func (s *Service) GenerateSummary(ctx context.Context, coachID string, req model.SummaryRequest) (*model.SummaryResponse, error) {
	dateFrom := req.DateFrom
	dateTo := req.DateTo
	if dateFrom == "" {
		dateFrom = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = time.Now().Format("2006-01-02")
	}

	// Get all athlete IDs for this coach
	athleteIDs, err := s.trainingClient.GetCoachAthleteIDs(ctx, coachID)
	if err != nil {
		s.log.Error().Err(err).Str("coach_id", coachID).Msg("failed to fetch coach athlete IDs")
		return nil, internalError("failed to fetch athlete list")
	}
	if len(athleteIDs) == 0 {
		return nil, badRequest("No athletes found for this coach")
	}

	// Fetch reports per athlete
	var allAthletes []athleteGroup
	totalReports := 0

	for _, aid := range athleteIDs {
		reports, err := s.trainingClient.GetReportsByAthlete(ctx, aid, dateFrom, dateTo)
		if err != nil {
			s.log.Warn().Err(err).Str("athlete_id", aid).Msg("failed to fetch reports, skipping")
			continue
		}
		if len(reports) == 0 {
			continue
		}
		name := reports[0].AthleteFullName
		login := reports[0].AthleteLogin
		allAthletes = append(allAthletes, athleteGroup{FullName: name, Login: login, Reports: reports})
		totalReports += len(reports)
	}

	if totalReports == 0 {
		return nil, badRequest("No reports found for the specified period")
	}

	// Build prompt
	systemPrompt := "Ты — ассистент тренера по лёгкой атлетике. Тебе даны отчёты спортсменов за период. " +
		"Сделай краткую структурированную сводку для тренера: общие тенденции, на что обратить внимание, " +
		"кому нужна корректировка нагрузки, есть ли жалобы на самочувствие.\n" +
		"Отвечай на русском языке. Будь конкретным, ссылайся на конкретных спортсменов по имени."

	userPrompt := s.buildSummaryPrompt(allAthletes, dateFrom, dateTo, req.Context)

	content, err := s.ollamaClient.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		s.log.Error().Err(err).Str("coach_id", coachID).Msg("ollama generation failed for summary")
		return nil, serviceUnavailable("AI service temporarily unavailable")
	}

	return &model.SummaryResponse{
		Type:        "summary",
		Content:     content,
		DateFrom:    dateFrom,
		DateTo:      dateTo,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Model:       s.ollamaClient.Model(),
	}, nil
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func (s *Service) buildSummaryPrompt(athletes []athleteGroup, dateFrom, dateTo, coachContext string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("=== Отчёты за %s — %s ===\n", dateFrom, dateTo))

	for _, a := range athletes {
		b.WriteString(fmt.Sprintf("\n--- %s (%s) ---\n", a.FullName, a.Login))
		for _, r := range a.Reports {
			b.WriteString(fmt.Sprintf("%s: %s (%d мин, нагрузка %d/10",
				r.ScheduledDate, r.Title, r.DurationMinutes, r.PerceivedEffort))
			if r.DistanceKm != nil {
				b.WriteString(fmt.Sprintf(", %.1f км", *r.DistanceKm))
			}
			if r.AvgHeartRate != nil {
				b.WriteString(fmt.Sprintf(", пульс %d", *r.AvgHeartRate))
			}
			b.WriteString(")\n")
			if r.Content != "" {
				b.WriteString(fmt.Sprintf("  Комментарий: %s\n", r.Content))
			}
		}
	}

	if coachContext != "" {
		b.WriteString(fmt.Sprintf("\n=== Фокус тренера ===\n%s\n", coachContext))
	}

	b.WriteString("\nСделай краткую сводку для тренера.")

	return b.String()
}

func (s *Service) fetchAthleteData(ctx context.Context, athleteID string) (*model.AthleteSummary, []model.ReportWithPlan, error) {
	summary, err := s.analyticsClient.GetAthleteSummary(ctx, athleteID)
	if err != nil {
		s.log.Error().Err(err).Str("athlete_id", athleteID).Msg("failed to fetch athlete summary")
		return nil, nil, internalError("failed to fetch athlete data")
	}

	reports, err := s.analyticsClient.GetAthleteReports(ctx, athleteID)
	if err != nil {
		s.log.Error().Err(err).Str("athlete_id", athleteID).Msg("failed to fetch athlete reports")
		return nil, nil, internalError("failed to fetch athlete data")
	}

	if len(reports) == 0 {
		return nil, nil, badRequest("Not enough training data for analysis")
	}

	// Limit to last 15 reports to avoid token overflow
	if len(reports) > 15 {
		reports = reports[len(reports)-15:]
	}

	return summary, reports, nil
}

func (s *Service) buildUserPrompt(summary *model.AthleteSummary, reports []model.ReportWithPlan, coachContext, instruction string) string {
	var b strings.Builder

	// Summary stats
	b.WriteString("=== Общая статистика спортсмена ===\n")
	b.WriteString(fmt.Sprintf("Всего тренировок: %d\n", summary.TotalReports))
	b.WriteString(fmt.Sprintf("Общая продолжительность: %d мин\n", summary.TotalDurationMin))
	b.WriteString(fmt.Sprintf("Общая дистанция: %.1f км\n", summary.TotalDistanceKm))
	b.WriteString(fmt.Sprintf("Средняя продолжительность: %.1f мин\n", summary.AvgDurationMin))
	b.WriteString(fmt.Sprintf("Средняя воспринимаемая нагрузка: %.1f / 10\n", summary.AvgPerceivedEffort))
	b.WriteString(fmt.Sprintf("Средний пульс: %.0f уд/мин\n", summary.AvgHeartRate))
	if summary.MaxHeartRateEver != nil {
		b.WriteString(fmt.Sprintf("Максимальный пульс за всё время: %d уд/мин\n", *summary.MaxHeartRateEver))
	}
	b.WriteString(fmt.Sprintf("Процент выполнения заданий: %.0f%%\n", summary.CompletionRate))
	b.WriteString(fmt.Sprintf("Всего заданий: %d\n", summary.TotalAssignments))

	// Reports
	b.WriteString(fmt.Sprintf("\n=== Последние %d тренировок ===\n", len(reports)))
	for i, r := range reports {
		b.WriteString(fmt.Sprintf("\n--- Тренировка %d ---\n", i+1))
		b.WriteString(fmt.Sprintf("Дата: %s\n", r.ScheduledDate))
		b.WriteString(fmt.Sprintf("Название: %s\n", r.Title))
		b.WriteString(fmt.Sprintf("Продолжительность: %d мин\n", r.DurationMinutes))
		b.WriteString(fmt.Sprintf("Воспринимаемая нагрузка: %d / 10\n", r.PerceivedEffort))
		if r.AvgHeartRate != nil {
			b.WriteString(fmt.Sprintf("Средний пульс: %d уд/мин\n", *r.AvgHeartRate))
		}
		if r.MaxHeartRate != nil {
			b.WriteString(fmt.Sprintf("Макс. пульс: %d уд/мин\n", *r.MaxHeartRate))
		}
		if r.DistanceKm != nil {
			b.WriteString(fmt.Sprintf("Дистанция: %.1f км\n", *r.DistanceKm))
		}
		if r.Content != "" {
			b.WriteString(fmt.Sprintf("Комментарий спортсмена: %s\n", r.Content))
		}
	}

	// Optional coach context
	if coachContext != "" {
		b.WriteString(fmt.Sprintf("\n=== Контекст от тренера ===\n%s\n", coachContext))
	}

	b.WriteString(fmt.Sprintf("\n%s.", instruction))

	return b.String()
}
