package service

import (
	"context"
	"sort"
	"time"

	"github.com/rs/zerolog"

	"github.com/coach-link/platform/services/analytics-service/internal/model"
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

func badRequest(msg string) *ServiceError {
	return &ServiceError{Code: "VALIDATION_ERROR", Message: msg, Status: 400}
}

func internalError(msg string) *ServiceError {
	return &ServiceError{Code: "INTERNAL_ERROR", Message: msg, Status: 500}
}

// ──────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────

// TrainingClient is the training data dependency used by the service.
type TrainingClient interface {
	GetReports(ctx context.Context, athleteID, dateFrom, dateTo string) ([]model.ReportWithPlan, error)
	GetAthleteStats(ctx context.Context, athleteID string) (*model.AthleteStats, error)
	GetCoachAthleteIDs(ctx context.Context, coachID string) ([]string, error)
	GetCoachOverview(ctx context.Context, coachID string) (*model.CoachOverviewStats, error)
}

type Service struct {
	trainingClient TrainingClient
	log            zerolog.Logger
}

func New(trainingClient TrainingClient, log zerolog.Logger) *Service {
	return &Service{trainingClient: trainingClient, log: log}
}

// GetAthleteSummary fetches stats from training-service and returns an AthleteSummary.
func (s *Service) GetAthleteSummary(ctx context.Context, athleteID string) (*model.AthleteSummary, error) {
	stats, err := s.trainingClient.GetAthleteStats(ctx, athleteID)
	if err != nil {
		s.log.Error().Err(err).Str("athlete_id", athleteID).Msg("failed to fetch athlete stats")
		return nil, internalError("failed to fetch athlete stats")
	}

	return &model.AthleteSummary{
		AthleteID:          athleteID,
		TotalReports:       stats.TotalReports,
		TotalDurationMin:   stats.TotalDurationMin,
		TotalDistanceKm:    stats.TotalDistanceKm,
		AvgDurationMin:     stats.AvgDurationMin,
		AvgPerceivedEffort: stats.AvgPerceivedEffort,
		AvgHeartRate:       stats.AvgHeartRate,
		MaxHeartRateEver:   stats.MaxHeartRateEver,
		CompletionRate:     stats.CompletionRate,
		TotalAssignments:   stats.TotalAssignments,
	}, nil
}

// GetAthleteProgress fetches reports and groups them by week or month, computing averages per period.
func (s *Service) GetAthleteProgress(ctx context.Context, athleteID, period, dateFrom, dateTo string) (*model.ProgressResponse, error) {
	if period == "" {
		period = "week"
	}
	if period != "week" && period != "month" {
		return nil, badRequest("period must be 'week' or 'month'")
	}

	// Default date range: last 12 weeks if no date_from provided
	if dateFrom == "" {
		dateFrom = time.Now().AddDate(0, 0, -12*7).Format("2006-01-02")
	}

	reports, err := s.trainingClient.GetReports(ctx, athleteID, dateFrom, dateTo)
	if err != nil {
		s.log.Error().Err(err).Str("athlete_id", athleteID).Msg("failed to fetch reports")
		return nil, internalError("failed to fetch reports")
	}

	buckets := s.groupReports(reports, period)

	// Sort by period_start ascending
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].PeriodStart < buckets[j].PeriodStart
	})

	return &model.ProgressResponse{
		AthleteID: athleteID,
		Period:    period,
		Points:    buckets,
	}, nil
}

// GetCoachOverview fetches coach stats and per-athlete summary.
func (s *Service) GetCoachOverview(ctx context.Context, coachID string) (*model.CoachOverview, error) {
	// Fetch coach totals
	overview, err := s.trainingClient.GetCoachOverview(ctx, coachID)
	if err != nil {
		s.log.Error().Err(err).Str("coach_id", coachID).Msg("failed to fetch coach overview")
		return nil, internalError("failed to fetch coach overview")
	}

	// Fetch athlete IDs for this coach
	athleteIDs, err := s.trainingClient.GetCoachAthleteIDs(ctx, coachID)
	if err != nil {
		s.log.Error().Err(err).Str("coach_id", coachID).Msg("failed to fetch coach athlete IDs")
		return nil, internalError("failed to fetch coach athlete IDs")
	}

	// Fetch per-athlete stats
	athletes := make([]model.AthleteOverview, 0, len(athleteIDs))
	var totalCompleted, totalAssignments int

	for _, aid := range athleteIDs {
		stats, err := s.trainingClient.GetAthleteStats(ctx, aid)
		if err != nil {
			s.log.Warn().Err(err).Str("athlete_id", aid).Msg("failed to fetch athlete stats, skipping")
			continue
		}

		athletes = append(athletes, model.AthleteOverview{
			AthleteID:      aid,
			TotalReports:   stats.TotalReports,
			AvgEffort:      stats.AvgPerceivedEffort,
			CompletionRate: stats.CompletionRate,
		})

		totalCompleted += stats.CompletedCount
		totalAssignments += stats.TotalAssignments
	}

	var completionRate float64
	if totalAssignments > 0 {
		completionRate = float64(totalCompleted) / float64(totalAssignments)
	}

	return &model.CoachOverview{
		TotalAthletes:    overview.TotalAthletes,
		TotalAssignments: overview.TotalAssignments,
		TotalReports:     overview.TotalReports,
		CompletionRate:   completionRate,
		Athletes:         athletes,
	}, nil
}

// GetReports fetches reports for an athlete (used by internal endpoints).
func (s *Service) GetReports(ctx context.Context, athleteID, dateFrom, dateTo string) ([]model.ReportWithPlan, error) {
	reports, err := s.trainingClient.GetReports(ctx, athleteID, dateFrom, dateTo)
	if err != nil {
		s.log.Error().Err(err).Str("athlete_id", athleteID).Msg("failed to fetch reports")
		return nil, internalError("failed to fetch reports")
	}
	return reports, nil
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

// groupReports groups reports into time buckets (week or month) and computes aggregates.
func (s *Service) groupReports(reports []model.ReportWithPlan, period string) []model.ProgressPoint {
	type bucket struct {
		start           time.Time
		end             time.Time
		count           int
		totalDuration   int
		totalEffort     int
		totalHeartRate  int
		heartRateCount  int
		totalDistanceKm float64
	}

	bucketMap := make(map[string]*bucket)

	for _, r := range reports {
		t, err := time.Parse("2006-01-02", r.ScheduledDate)
		if err != nil {
			s.log.Warn().Str("date", r.ScheduledDate).Msg("invalid scheduled_date, skipping report")
			continue
		}

		var bStart, bEnd time.Time
		if period == "week" {
			bStart = weekStart(t)
			bEnd = bStart.AddDate(0, 0, 6)
		} else {
			bStart = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			bEnd = bStart.AddDate(0, 1, -1)
		}

		key := bStart.Format("2006-01-02")

		b, ok := bucketMap[key]
		if !ok {
			b = &bucket{start: bStart, end: bEnd}
			bucketMap[key] = b
		}

		b.count++
		b.totalDuration += r.DurationMinutes
		b.totalEffort += r.PerceivedEffort

		if r.AvgHeartRate != nil {
			b.totalHeartRate += *r.AvgHeartRate
			b.heartRateCount++
		}
		if r.DistanceKm != nil {
			b.totalDistanceKm += *r.DistanceKm
		}
	}

	points := make([]model.ProgressPoint, 0, len(bucketMap))
	for _, b := range bucketMap {
		var avgEffort, avgHR float64
		if b.count > 0 {
			avgEffort = float64(b.totalEffort) / float64(b.count)
		}
		if b.heartRateCount > 0 {
			avgHR = float64(b.totalHeartRate) / float64(b.heartRateCount)
		}

		points = append(points, model.ProgressPoint{
			PeriodStart:        b.start.Format("2006-01-02"),
			PeriodEnd:          b.end.Format("2006-01-02"),
			ReportCount:        b.count,
			TotalDurationMin:   b.totalDuration,
			AvgPerceivedEffort: avgEffort,
			AvgHeartRate:       avgHR,
			TotalDistanceKm:    b.totalDistanceKm,
		})
	}

	return points
}

// weekStart returns the Monday of the week containing the given date.
func weekStart(t time.Time) time.Time {
	weekday := t.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	offset := int(weekday) - int(time.Monday)
	return time.Date(t.Year(), t.Month(), t.Day()-offset, 0, 0, 0, 0, time.UTC)
}
