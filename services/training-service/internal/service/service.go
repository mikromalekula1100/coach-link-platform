package service

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/coach-link/platform/pkg/events"
	"github.com/coach-link/platform/services/training-service/internal/client"
	"github.com/coach-link/platform/services/training-service/internal/model"
	"github.com/coach-link/platform/services/training-service/internal/repository"
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

func forbidden(msg string) *ServiceError {
	return &ServiceError{Code: "FORBIDDEN", Message: msg, Status: 403}
}

func notFound(msg string) *ServiceError {
	return &ServiceError{Code: "NOT_FOUND", Message: msg, Status: 404}
}

func badRequest(msg string) *ServiceError {
	return &ServiceError{Code: "VALIDATION_ERROR", Message: msg, Status: 400}
}

func conflictError(code, msg string) *ServiceError {
	return &ServiceError{Code: code, Message: msg, Status: 409}
}

// IsServiceError checks if an error is a ServiceError and returns it.
func IsServiceError(err error) (*ServiceError, bool) {
	if se, ok := err.(*ServiceError); ok {
		return se, true
	}
	return nil, false
}

// ──────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────

type Service struct {
	repo       *repository.Repository
	js         nats.JetStreamContext
	log        zerolog.Logger
	userClient *client.UserClient
}

func New(repo *repository.Repository, js nats.JetStreamContext, log zerolog.Logger, userClient *client.UserClient) *Service {
	return &Service{repo: repo, js: js, log: log, userClient: userClient}
}

// ──────────────────────────────────────────────
// Training Plans
// ──────────────────────────────────────────────

func (s *Service) CreatePlan(ctx context.Context, coachID, coachFullName, coachLogin string, req model.CreateTrainingPlanRequest) (*model.CreateTrainingPlanResponse, error) {
	// Validate: at least one of athlete_ids or group_id must be provided
	if len(req.AthleteIDs) == 0 && req.GroupID == nil {
		return nil, badRequest("At least one of athlete_ids or group_id must be provided")
	}

	// Fetch coach's full_name if not provided by gateway (X-User-FullName is not in JWT)
	if coachFullName == "" {
		if info, err := s.userClient.GetUserByID(ctx, coachID); err == nil {
			coachFullName = info.FullName
			if coachLogin == "" {
				coachLogin = info.Login
			}
		} else {
			s.log.Warn().Err(err).Str("coach_id", coachID).Msg("could not fetch coach profile")
		}
	}

	// Parse scheduled_date
	scheduledDate, err := time.Parse("2006-01-02", req.ScheduledDate)
	if err != nil {
		return nil, badRequest("Invalid scheduled_date format, expected YYYY-MM-DD")
	}

	title := req.Title
	description := req.Description

	// If template_id is provided, load the template and use its title/description as defaults
	if req.TemplateID != nil && *req.TemplateID != "" {
		tmpl, err := s.repo.GetTemplateByID(ctx, *req.TemplateID)
		if err != nil {
			return nil, err
		}
		if tmpl == nil {
			return nil, notFound("Template not found")
		}
		if tmpl.CoachID != coachID {
			return nil, forbidden("Template does not belong to you")
		}
		// Use template values only if request fields are empty
		if title == "" {
			title = tmpl.Title
		}
		if description == "" {
			description = tmpl.Description
		}
	}

	// Collect athlete info: merge athlete_ids with group members
	type athleteInfo struct {
		ID       string
		FullName string
		Login    string
	}
	athleteMap := make(map[string]athleteInfo)

	// If group_id provided, fetch members from User Service
	if req.GroupID != nil && *req.GroupID != "" {
		members, err := s.userClient.GetGroupMembers(ctx, *req.GroupID)
		if err != nil {
			s.log.Error().Err(err).Str("group_id", *req.GroupID).Msg("failed to get group members")
			return nil, badRequest("Failed to retrieve group members: " + err.Error())
		}
		for _, m := range members {
			athleteMap[m.AthleteID] = athleteInfo{
				ID:       m.AthleteID,
				FullName: m.FullName,
				Login:    m.Login,
			}
		}
	}

	// Fetch profile for any individual athlete_id not already resolved from group members
	for _, aid := range req.AthleteIDs {
		if _, exists := athleteMap[aid]; !exists {
			info, err := s.userClient.GetUserByID(ctx, aid)
			if err != nil {
				s.log.Warn().Err(err).Str("athlete_id", aid).Msg("could not fetch athlete profile, using empty name")
				athleteMap[aid] = athleteInfo{ID: aid, FullName: "", Login: ""}
			} else {
				athleteMap[aid] = athleteInfo{ID: aid, FullName: info.FullName, Login: info.Login}
			}
		}
	}

	if len(athleteMap) == 0 {
		return nil, badRequest("No athletes resolved for the training plan")
	}

	// Create the plan
	plan := &model.TrainingPlan{
		CoachID:       coachID,
		Title:         title,
		Description:   description,
		ScheduledDate: scheduledDate,
	}
	if err := s.repo.CreatePlan(ctx, plan); err != nil {
		return nil, err
	}

	// Create assignments for each athlete
	var assignmentResponses []model.AssignmentBriefResponse
	for _, info := range athleteMap {
		assignment := &model.TrainingAssignment{
			PlanID:          plan.ID,
			AthleteID:       info.ID,
			CoachID:         coachID,
			AthleteFullName: info.FullName,
			AthleteLogin:    info.Login,
			CoachFullName:   coachFullName,
			CoachLogin:      coachLogin,
		}
		if err := s.repo.CreateAssignment(ctx, assignment); err != nil {
			s.log.Error().Err(err).Str("athlete_id", info.ID).Msg("failed to create assignment")
			continue
		}

		assignmentResponses = append(assignmentResponses, model.AssignmentBriefResponse{
			ID:              assignment.ID,
			AthleteID:       info.ID,
			AthleteFullName: info.FullName,
			AthleteLogin:    info.Login,
		})

		// Publish training.assigned event
		evt := events.NewEvent(events.SubjectTrainingAssigned, events.TrainingAssignedPayload{
			AssignmentID:  assignment.ID,
			AthleteID:     info.ID,
			CoachID:       coachID,
			CoachFullName: coachFullName,
			Title:         title,
			ScheduledDate: req.ScheduledDate,
		})
		s.publishEvent(events.SubjectTrainingAssigned, evt)
	}

	resp := &model.CreateTrainingPlanResponse{
		Plan: model.PlanResponse{
			ID:            plan.ID,
			Title:         plan.Title,
			ScheduledDate: plan.ScheduledDate.Format("2006-01-02"),
			CreatedAt:     plan.CreatedAt.Format(time.RFC3339),
		},
		Assignments: assignmentResponses,
	}

	// If save_as_template, create template
	if req.SaveAsTemplate {
		tmpl := &model.TrainingTemplate{
			CoachID:     coachID,
			Title:       title,
			Description: description,
		}
		if err := s.repo.CreateTemplate(ctx, tmpl); err != nil {
			s.log.Error().Err(err).Msg("failed to save training template")
		} else {
			tmplResp := &model.TrainingTemplateResponse{
				ID:          tmpl.ID,
				Title:       tmpl.Title,
				Description: tmpl.Description,
				CreatedAt:   tmpl.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   tmpl.UpdatedAt.Format(time.RFC3339),
			}
			resp.Template = tmplResp
		}
	}

	return resp, nil
}

// ──────────────────────────────────────────────
// Assignments
// ──────────────────────────────────────────────

func (s *Service) GetAssignments(ctx context.Context, userID, role string, filter model.AssignmentFilter) ([]model.AssignmentRow, int, error) {
	if role == "coach" {
		return s.repo.GetCoachAssignments(ctx, userID, filter)
	}
	return s.repo.GetAthleteAssignments(ctx, userID, filter.Page, filter.PageSize)
}

func (s *Service) GetAssignment(ctx context.Context, userID, role, assignmentID string) (*model.AssignmentRow, error) {
	row, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, notFound("Assignment not found")
	}

	// Access check
	if role == "coach" && row.CoachID != userID {
		return nil, forbidden("Assignment does not belong to you")
	}
	if role == "athlete" && row.AthleteID != userID {
		return nil, forbidden("Assignment is not assigned to you")
	}

	return row, nil
}

func (s *Service) DeleteAssignment(ctx context.Context, coachID, assignmentID string) error {
	// Verify ownership first
	row, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return err
	}
	if row == nil {
		return notFound("Assignment not found")
	}
	if row.CoachID != coachID {
		return forbidden("Assignment does not belong to you")
	}

	deleted, err := s.repo.DeleteAssignment(ctx, assignmentID)
	if err != nil {
		return err
	}
	if deleted == nil {
		return notFound("Assignment not found")
	}

	// Publish training.deleted event
	evt := events.NewEvent(events.SubjectTrainingDeleted, events.TrainingDeletedPayload{
		AssignmentID: deleted.ID,
		AthleteID:    deleted.AthleteID,
		CoachID:      deleted.CoachID,
		Title:        row.Title,
	})
	s.publishEvent(events.SubjectTrainingDeleted, evt)

	return nil
}

func (s *Service) ArchiveAssignment(ctx context.Context, coachID, assignmentID string) error {
	row, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return err
	}
	if row == nil {
		return notFound("Assignment not found")
	}
	if row.CoachID != coachID {
		return forbidden("Assignment does not belong to you")
	}
	if row.Status != "completed" {
		return &ServiceError{Code: "ASSIGNMENT_NOT_COMPLETED", Message: "Only completed assignments can be archived", Status: 400}
	}

	return s.repo.UpdateAssignmentStatus(ctx, assignmentID, "archived")
}

func (s *Service) GetArchivedAssignments(ctx context.Context, coachID string, filter model.AssignmentFilter) ([]model.AssignmentRow, int, error) {
	return s.repo.GetArchivedAssignments(ctx, coachID, filter)
}

// ──────────────────────────────────────────────
// Reports
// ──────────────────────────────────────────────

func (s *Service) SubmitReport(ctx context.Context, athleteID, assignmentID string, req model.CreateReportRequest) (*model.TrainingReport, error) {
	// Verify the assignment exists and belongs to this athlete
	row, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, notFound("Assignment not found")
	}
	if row.AthleteID != athleteID {
		return nil, forbidden("Assignment is not assigned to you")
	}
	if row.Status != "assigned" {
		return nil, &ServiceError{Code: "ASSIGNMENT_NOT_ASSIGNED", Message: "Cannot submit report for this assignment status", Status: 400}
	}

	// Check if report already exists
	exists, err := s.repo.ReportExists(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, conflictError("REPORT_ALREADY_EXISTS", "A report has already been submitted for this assignment")
	}

	// Create the report
	report := &model.TrainingReport{
		AssignmentID:    assignmentID,
		AthleteID:       athleteID,
		Content:         req.Content,
		DurationMinutes: req.DurationMinutes,
		PerceivedEffort: req.PerceivedEffort,
		MaxHeartRate:    req.MaxHeartRate,
		AvgHeartRate:    req.AvgHeartRate,
		DistanceKm:      req.DistanceKm,
	}
	if err := s.repo.CreateReport(ctx, report); err != nil {
		return nil, err
	}

	// Update assignment status to completed
	if err := s.repo.UpdateAssignmentStatus(ctx, assignmentID, "completed"); err != nil {
		s.log.Error().Err(err).Str("assignment_id", assignmentID).Msg("failed to update assignment status after report")
	}

	// Publish report.submitted event
	evt := events.NewEvent(events.SubjectReportSubmitted, events.ReportSubmittedPayload{
		AssignmentID:    assignmentID,
		AthleteID:       athleteID,
		AthleteFullName: row.AthleteFullName,
		CoachID:         row.CoachID,
		Title:           row.Title,
	})
	s.publishEvent(events.SubjectReportSubmitted, evt)

	return report, nil
}

func (s *Service) GetReport(ctx context.Context, userID, role, assignmentID string) (*model.TrainingReport, *model.TrainingAssignment, error) {
	// Verify assignment access
	row, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return nil, nil, err
	}
	if row == nil {
		return nil, nil, notFound("Assignment not found")
	}
	if role == "coach" && row.CoachID != userID {
		return nil, nil, forbidden("Assignment does not belong to you")
	}
	if role == "athlete" && row.AthleteID != userID {
		return nil, nil, forbidden("Assignment is not assigned to you")
	}

	report, err := s.repo.GetReportByAssignmentID(ctx, assignmentID)
	if err != nil {
		return nil, nil, err
	}
	if report == nil {
		return nil, nil, notFound("Report not found for this assignment")
	}

	// Build a TrainingAssignment from the row for name info
	assignment := &model.TrainingAssignment{
		AthleteFullName: row.AthleteFullName,
		AthleteLogin:    row.AthleteLogin,
	}

	return report, assignment, nil
}

// ──────────────────────────────────────────────
// Templates
// ──────────────────────────────────────────────

func (s *Service) CreateTemplate(ctx context.Context, coachID string, req model.CreateTemplateRequest) (*model.TrainingTemplate, error) {
	tmpl := &model.TrainingTemplate{
		CoachID:     coachID,
		Title:       req.Title,
		Description: req.Description,
	}
	if err := s.repo.CreateTemplate(ctx, tmpl); err != nil {
		return nil, err
	}
	return tmpl, nil
}

func (s *Service) GetTemplates(ctx context.Context, coachID, query string, page, pageSize int) ([]model.TrainingTemplate, int, error) {
	return s.repo.GetTemplates(ctx, coachID, query, page, pageSize)
}

func (s *Service) GetTemplate(ctx context.Context, coachID, templateID string) (*model.TrainingTemplate, error) {
	tmpl, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, notFound("Template not found")
	}
	if tmpl.CoachID != coachID {
		return nil, forbidden("Template does not belong to you")
	}
	return tmpl, nil
}

func (s *Service) UpdateTemplate(ctx context.Context, coachID, templateID string, req model.UpdateTemplateRequest) (*model.TrainingTemplate, error) {
	tmpl, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, notFound("Template not found")
	}
	if tmpl.CoachID != coachID {
		return nil, forbidden("Template does not belong to you")
	}

	if req.Title == nil && req.Description == nil {
		return nil, badRequest("At least one of title or description must be provided")
	}

	if err := s.repo.UpdateTemplate(ctx, templateID, req.Title, req.Description); err != nil {
		return nil, err
	}

	return s.repo.GetTemplateByID(ctx, templateID)
}

func (s *Service) DeleteTemplate(ctx context.Context, coachID, templateID string) error {
	tmpl, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return err
	}
	if tmpl == nil {
		return notFound("Template not found")
	}
	if tmpl.CoachID != coachID {
		return forbidden("Template does not belong to you")
	}

	return s.repo.DeleteTemplate(ctx, templateID)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func ComputeIsOverdue(status string, scheduledDate time.Time) bool {
	if status != "assigned" {
		return false
	}
	// Overdue if scheduled_date + 1 day is before now (i.e., the day after scheduled_date has passed)
	deadline := scheduledDate.Add(24 * time.Hour)
	return time.Now().UTC().After(deadline)
}

func (s *Service) publishEvent(subject string, evt events.Event) {
	data, err := evt.Marshal()
	if err != nil {
		s.log.Error().Err(err).Str("subject", subject).Msg("failed to marshal event")
		return
	}
	if _, err := s.js.Publish(subject, data); err != nil {
		s.log.Error().Err(err).Str("subject", subject).Msg("failed to publish event")
	} else {
		s.log.Info().Str("subject", subject).Str("event_id", evt.EventID).Msg("event published")
	}
}
