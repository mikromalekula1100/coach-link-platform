package service

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/rs/zerolog"

	"github.com/coach-link/platform/services/bdui-service/internal/builder"
	"github.com/coach-link/platform/services/bdui-service/internal/client"
	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

// ──────────────────────────────────────────────
// Errors
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

func notFound(msg string) *ServiceError {
	return &ServiceError{Code: "NOT_FOUND", Message: msg, Status: http.StatusNotFound}
}

func forbidden(msg string) *ServiceError {
	return &ServiceError{Code: "FORBIDDEN", Message: msg, Status: http.StatusForbidden}
}

func internalError(msg string) *ServiceError {
	return &ServiceError{Code: "INTERNAL_ERROR", Message: msg, Status: http.StatusInternalServerError}
}

// ──────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────

type Service struct {
	userClient     *client.UserClient
	trainingClient *client.TrainingClient
	log            zerolog.Logger
}

func New(userClient *client.UserClient, trainingClient *client.TrainingClient, log zerolog.Logger) *Service {
	return &Service{
		userClient:     userClient,
		trainingClient: trainingClient,
		log:            log,
	}
}

// GetScreen возвращает BDUI-схему для dashboard-экрана.
func (s *Service) GetScreen(ctx context.Context, screenID, userID, userRole string) (*model.BduiSchema, error) {
	switch screenID {
	case "coach-dashboard":
		if userRole != "coach" {
			return nil, forbidden("Only coaches can access coach-dashboard")
		}
		return s.buildCoachDashboard(ctx, userID)

	case "athlete-dashboard":
		if userRole != "athlete" {
			return nil, forbidden("Only athletes can access athlete-dashboard")
		}
		return s.buildAthleteDashboard(ctx, userID)

	default:
		return nil, notFound(fmt.Sprintf("Unknown screen: %s", screenID))
	}
}

// GetTrainingDetail возвращает BDUI-схему для описания тренировки.
func (s *Service) GetTrainingDetail(ctx context.Context, assignmentID, userID, userRole string) (*model.BduiSchema, error) {
	assignment, err := s.trainingClient.GetAssignment(ctx, assignmentID, userID, userRole)
	if err != nil {
		s.log.Error().Err(err).Str("assignment_id", assignmentID).Msg("failed to get assignment")
		return nil, internalError("Failed to load training details")
	}
	if assignment == nil {
		return nil, notFound("Assignment not found")
	}

	schema := builder.BuildTrainingDetail(assignment)
	return &schema, nil
}

// ──────────────────────────────────────────────
// Coach Dashboard
// ──────────────────────────────────────────────

func (s *Service) buildCoachDashboard(ctx context.Context, userID string) (*model.BduiSchema, error) {
	var (
		profile     *model.UserProfile
		athletes    []model.AthleteInfo
		pendingReqs []model.ConnectionRequest
		reports     []model.ReportWithPlan
		assignments []model.AssignmentListItem
		mu          sync.Mutex
		errCount    int
	)

	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		p, err := s.userClient.GetUserProfile(ctx, userID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to get user profile")
			errCount++
		} else {
			profile = p
		}
	}()

	go func() {
		defer wg.Done()
		a, err := s.userClient.GetAthletes(ctx, userID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to get athletes")
			errCount++
		} else {
			athletes = a
		}
	}()

	go func() {
		defer wg.Done()
		r, err := s.userClient.GetPendingRequests(ctx, userID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to get pending requests")
			errCount++
		} else {
			pendingReqs = r
		}
	}()

	go func() {
		defer wg.Done()
		rp, err := s.trainingClient.GetRecentReports(ctx, userID, 5)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to get recent reports")
			errCount++
		} else {
			reports = rp
		}
	}()

	go func() {
		defer wg.Done()
		as, err := s.trainingClient.GetUpcomingAssignments(ctx, userID, "coach", 5)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to get upcoming assignments")
			errCount++
		} else {
			assignments = as
		}
	}()

	wg.Wait()

	if errCount == 5 {
		return nil, internalError("All upstream services unavailable")
	}

	schema := builder.BuildCoachDashboard(builder.CoachDashboardData{
		Profile:      profile,
		Athletes:     athletes,
		PendingReqs:  pendingReqs,
		RecentRpts:   reports,
		UpcomingAsns: assignments,
	})

	return &schema, nil
}

// ──────────────────────────────────────────────
// Athlete Dashboard
// ──────────────────────────────────────────────

func (s *Service) buildAthleteDashboard(ctx context.Context, userID string) (*model.BduiSchema, error) {
	var (
		profile     *model.UserProfile
		coach       *model.CoachInfo
		assignments []model.AssignmentListItem
		mu          sync.Mutex
		errCount    int
	)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		p, err := s.userClient.GetUserProfile(ctx, userID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to get user profile")
			errCount++
		} else {
			profile = p
		}
	}()

	go func() {
		defer wg.Done()
		c, err := s.userClient.GetCoach(ctx, userID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to get coach")
			errCount++
		} else {
			coach = c
		}
	}()

	go func() {
		defer wg.Done()
		as, err := s.trainingClient.GetUpcomingAssignments(ctx, userID, "athlete", 5)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			s.log.Warn().Err(err).Msg("failed to get upcoming assignments")
			errCount++
		} else {
			assignments = as
		}
	}()

	wg.Wait()

	if errCount == 3 {
		return nil, internalError("All upstream services unavailable")
	}

	schema := builder.BuildAthleteDashboard(builder.AthleteDashboardData{
		Profile:      profile,
		Coach:        coach,
		UpcomingAsns: assignments,
	})

	return &schema, nil
}
