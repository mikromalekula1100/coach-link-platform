package service

import (
	"context"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/coach-link/platform/pkg/events"
	"github.com/coach-link/platform/services/user-service/internal/model"
	"github.com/coach-link/platform/services/user-service/internal/repository"
)

// ──────────────────────────────────────────────
// Error types
// ──────────────────────────────────────────────

type ServiceError struct {
	Code    string
	Message string
	Status  int // HTTP status
}

func (e *ServiceError) Error() string { return e.Message }

func forbidden(msg string) *ServiceError {
	return &ServiceError{Code: "FORBIDDEN", Message: msg, Status: 403}
}

func notFound(msg string) *ServiceError {
	return &ServiceError{Code: "NOT_FOUND", Message: msg, Status: 404}
}

func conflict(code, msg string) *ServiceError {
	return &ServiceError{Code: code, Message: msg, Status: 409}
}

func badRequest(msg string) *ServiceError {
	return &ServiceError{Code: "VALIDATION_ERROR", Message: msg, Status: 400}
}

// ──────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────

type Service struct {
	repo *repository.Repository
	js   nats.JetStreamContext
	log  zerolog.Logger
}

func New(repo *repository.Repository, js nats.JetStreamContext, log zerolog.Logger) *Service {
	return &Service{repo: repo, js: js, log: log}
}

// ──────────────────────────────────────────────
// Profiles
// ──────────────────────────────────────────────

func (s *Service) GetProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	p, err := s.repo.GetProfileByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, notFound("User profile not found")
	}
	return p, nil
}

func (s *Service) SearchUsers(ctx context.Context, query, role string, page, pageSize int) ([]model.UserProfile, int, error) {
	if len(strings.TrimSpace(query)) < 2 {
		return nil, 0, badRequest("Search query must be at least 2 characters")
	}
	return s.repo.SearchProfiles(ctx, query, role, page, pageSize)
}

// ──────────────────────────────────────────────
// Connection Requests
// ──────────────────────────────────────────────

func (s *Service) SendConnectionRequest(ctx context.Context, athleteID, coachID, athleteRole string) (*model.ConnectionRequest, error) {
	if athleteRole != "athlete" {
		return nil, forbidden("Only athletes can send connection requests")
	}

	// Verify coach exists and is a coach
	coach, err := s.repo.GetProfileByID(ctx, coachID)
	if err != nil {
		return nil, err
	}
	if coach == nil {
		return nil, notFound("Coach not found")
	}
	if coach.Role != "coach" {
		return nil, badRequest("Target user is not a coach")
	}

	cr, err := s.repo.CreateConnectionRequest(ctx, athleteID, coachID)
	if err != nil {
		switch err {
		case repository.ErrAlreadyHasCoach:
			return nil, conflict("ALREADY_HAS_COACH", "Athlete is already connected to a coach")
		case repository.ErrRequestAlreadyExists:
			return nil, conflict("REQUEST_ALREADY_EXISTS", "A pending request already exists")
		default:
			return nil, err
		}
	}

	// Fetch athlete profile for event payload
	athlete, _ := s.repo.GetProfileByID(ctx, athleteID)
	athleteLogin := ""
	athleteFullName := ""
	if athlete != nil {
		athleteLogin = athlete.Login
		athleteFullName = athlete.FullName
	}

	// Publish event
	evt := events.NewEvent(events.SubjectConnectionRequested, events.ConnectionRequestedPayload{
		RequestID:       cr.ID,
		AthleteID:       athleteID,
		AthleteFullName: athleteFullName,
		AthleteLogin:    athleteLogin,
		CoachID:         coachID,
	})
	s.publishEvent(events.SubjectConnectionRequested, evt)

	return cr, nil
}

func (s *Service) GetIncomingRequests(ctx context.Context, coachID, role, status string, page, pageSize int) ([]model.ConnectionRequest, int, error) {
	if role != "coach" {
		return nil, 0, forbidden("Only coaches can view incoming requests")
	}
	return s.repo.GetIncomingRequests(ctx, coachID, status, page, pageSize)
}

func (s *Service) GetOutgoingRequest(ctx context.Context, athleteID, role string) (*model.ConnectionRequest, error) {
	if role != "athlete" {
		return nil, forbidden("Only athletes can view outgoing requests")
	}
	cr, err := s.repo.GetOutgoingRequest(ctx, athleteID)
	if err != nil {
		return nil, err
	}
	if cr == nil {
		return nil, notFound("No pending outgoing request")
	}
	return cr, nil
}

func (s *Service) AcceptConnectionRequest(ctx context.Context, coachID, role, requestID string) (*model.ConnectionRequest, error) {
	if role != "coach" {
		return nil, forbidden("Only coaches can accept requests")
	}

	cr, err := s.repo.GetConnectionRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if cr == nil {
		return nil, notFound("Connection request not found")
	}
	if cr.CoachID != coachID {
		return nil, forbidden("This request is not addressed to you")
	}
	if cr.Status != "pending" {
		return nil, badRequest("Request is not in pending state")
	}

	// Check the athlete doesn't already have a coach (could have been accepted by another coach in the meantime)
	existingRel, err := s.repo.GetRelationByAthleteID(ctx, cr.AthleteID)
	if err != nil {
		return nil, err
	}
	if existingRel != nil {
		// Reject this request since athlete is already taken
		_ = s.repo.UpdateConnectionRequestStatus(ctx, requestID, "rejected")
		return nil, conflict("ALREADY_HAS_COACH", "Athlete is already connected to a coach")
	}

	if err := s.repo.UpdateConnectionRequestStatus(ctx, requestID, "accepted"); err != nil {
		return nil, err
	}
	if err := s.repo.CreateRelation(ctx, coachID, cr.AthleteID); err != nil {
		return nil, err
	}

	// Refresh the request
	cr, _ = s.repo.GetConnectionRequestByID(ctx, requestID)

	// Publish event
	coach, _ := s.repo.GetProfileByID(ctx, coachID)
	athlete, _ := s.repo.GetProfileByID(ctx, cr.AthleteID)
	coachFullName := ""
	athleteFullName := ""
	if coach != nil {
		coachFullName = coach.FullName
	}
	if athlete != nil {
		athleteFullName = athlete.FullName
	}

	evt := events.NewEvent(events.SubjectConnectionAccepted, events.ConnectionAcceptedPayload{
		RequestID:       requestID,
		AthleteID:       cr.AthleteID,
		AthleteFullName: athleteFullName,
		CoachID:         coachID,
		CoachFullName:   coachFullName,
	})
	s.publishEvent(events.SubjectConnectionAccepted, evt)

	return cr, nil
}

func (s *Service) RejectConnectionRequest(ctx context.Context, coachID, role, requestID string) (*model.ConnectionRequest, error) {
	if role != "coach" {
		return nil, forbidden("Only coaches can reject requests")
	}

	cr, err := s.repo.GetConnectionRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if cr == nil {
		return nil, notFound("Connection request not found")
	}
	if cr.CoachID != coachID {
		return nil, forbidden("This request is not addressed to you")
	}
	if cr.Status != "pending" {
		return nil, badRequest("Request is not in pending state")
	}

	if err := s.repo.UpdateConnectionRequestStatus(ctx, requestID, "rejected"); err != nil {
		return nil, err
	}

	// Refresh
	cr, _ = s.repo.GetConnectionRequestByID(ctx, requestID)

	// Publish event
	coach, _ := s.repo.GetProfileByID(ctx, coachID)
	coachFullName := ""
	if coach != nil {
		coachFullName = coach.FullName
	}

	evt := events.NewEvent(events.SubjectConnectionRejected, events.ConnectionRejectedPayload{
		RequestID:     requestID,
		AthleteID:     cr.AthleteID,
		CoachID:       coachID,
		CoachFullName: coachFullName,
	})
	s.publishEvent(events.SubjectConnectionRejected, evt)

	return cr, nil
}

// ──────────────────────────────────────────────
// Athletes / Coach
// ──────────────────────────────────────────────

func (s *Service) GetAthletes(ctx context.Context, coachID, role, query string, page, pageSize int) ([]model.AthleteInfo, int, error) {
	if role != "coach" {
		return nil, 0, forbidden("Only coaches can list athletes")
	}
	return s.repo.GetAthletes(ctx, coachID, query, page, pageSize)
}

func (s *Service) GetCoach(ctx context.Context, athleteID, role string) (*model.CoachInfo, error) {
	if role != "athlete" {
		return nil, forbidden("Only athletes can view their coach")
	}

	rel, err := s.repo.GetRelationByAthleteID(ctx, athleteID)
	if err != nil {
		return nil, err
	}
	if rel == nil {
		return nil, notFound("Not connected to a coach")
	}

	coach, err := s.repo.GetProfileByID(ctx, rel.CoachID)
	if err != nil {
		return nil, err
	}
	if coach == nil {
		return nil, notFound("Coach profile not found")
	}

	return &model.CoachInfo{
		ID:          coach.ID,
		Login:       coach.Login,
		FullName:    coach.FullName,
		ConnectedAt: rel.CreatedAt,
	}, nil
}

func (s *Service) RemoveAthlete(ctx context.Context, coachID, role, athleteID string) error {
	if role != "coach" {
		return forbidden("Only coaches can remove athletes")
	}
	err := s.repo.DeleteRelation(ctx, coachID, athleteID)
	if err == repository.ErrNotFound {
		return notFound("Athlete relation not found")
	}
	return err
}

// ──────────────────────────────────────────────
// Training Groups
// ──────────────────────────────────────────────

func (s *Service) CreateGroup(ctx context.Context, coachID, role, name string) (*model.TrainingGroup, error) {
	if role != "coach" {
		return nil, forbidden("Only coaches can create groups")
	}
	return s.repo.CreateGroup(ctx, coachID, name)
}

func (s *Service) GetGroups(ctx context.Context, userID, role string, page, pageSize int) ([]model.TrainingGroupSummary, int, error) {
	if role == "coach" {
		return s.repo.GetCoachGroups(ctx, userID, page, pageSize)
	}
	return s.repo.GetAthleteGroups(ctx, userID, page, pageSize)
}

func (s *Service) GetGroup(ctx context.Context, userID, role, groupID, query string) (*model.TrainingGroupDetail, error) {
	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, notFound("Group not found")
	}

	// Access check
	if role == "coach" && group.CoachID != userID {
		return nil, forbidden("This is not your group")
	}
	if role == "athlete" {
		// Verify athlete is a member
		members, err := s.repo.GetGroupMemberIDs(ctx, groupID)
		if err != nil {
			return nil, err
		}
		found := false
		for _, mid := range members {
			if mid == userID {
				found = true
				break
			}
		}
		if !found {
			return nil, forbidden("You are not a member of this group")
		}
	}

	groupMembers, err := s.repo.GetGroupMembers(ctx, groupID, query)
	if err != nil {
		return nil, err
	}

	memberResponses := make([]model.GroupMemberResponse, 0, len(groupMembers))
	for _, m := range groupMembers {
		memberResponses = append(memberResponses, model.GroupMemberResponse{
			AthleteID: m.AthleteID,
			Login:     m.Login,
			FullName:  m.FullName,
			AddedAt:   m.AddedAt,
		})
	}

	return &model.TrainingGroupDetail{
		ID:        group.ID,
		Name:      group.Name,
		Members:   memberResponses,
		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}, nil
}

func (s *Service) UpdateGroup(ctx context.Context, coachID, role, groupID, name string) (*model.TrainingGroup, error) {
	if role != "coach" {
		return nil, forbidden("Only coaches can update groups")
	}

	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, notFound("Group not found")
	}
	if group.CoachID != coachID {
		return nil, forbidden("This is not your group")
	}

	if err := s.repo.UpdateGroup(ctx, groupID, name); err != nil {
		return nil, err
	}

	return s.repo.GetGroupByID(ctx, groupID)
}

func (s *Service) DeleteGroup(ctx context.Context, coachID, role, groupID string) error {
	if role != "coach" {
		return forbidden("Only coaches can delete groups")
	}

	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return notFound("Group not found")
	}
	if group.CoachID != coachID {
		return forbidden("This is not your group")
	}

	return s.repo.DeleteGroup(ctx, groupID)
}

func (s *Service) AddGroupMember(ctx context.Context, coachID, role, groupID, athleteID string) (*model.GroupMemberResponse, error) {
	if role != "coach" {
		return nil, forbidden("Only coaches can add group members")
	}

	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, notFound("Group not found")
	}
	if group.CoachID != coachID {
		return nil, forbidden("This is not your group")
	}

	// Verify athlete is coach's athlete
	has, err := s.repo.HasRelation(ctx, coachID, athleteID)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, forbidden("Athlete is not connected to you")
	}

	if err := s.repo.AddGroupMember(ctx, groupID, athleteID); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, conflict("ALREADY_IN_GROUP", "Athlete is already in this group")
		}
		return nil, err
	}

	// Fetch athlete info for the response
	athlete, err := s.repo.GetProfileByID(ctx, athleteID)
	if err != nil {
		return nil, err
	}

	// Fetch added_at — get the group members and find this athlete
	members, err := s.repo.GetGroupMembers(ctx, groupID, "")
	if err != nil {
		return nil, err
	}
	var resp model.GroupMemberResponse
	for _, m := range members {
		if m.AthleteID == athleteID {
			resp = model.GroupMemberResponse{
				AthleteID: m.AthleteID,
				Login:     m.Login,
				FullName:  m.FullName,
				AddedAt:   m.AddedAt,
			}
			break
		}
	}
	if resp.AthleteID == "" && athlete != nil {
		resp = model.GroupMemberResponse{
			AthleteID: athlete.ID,
			Login:     athlete.Login,
			FullName:  athlete.FullName,
		}
	}

	// Publish event
	evt := events.NewEvent(events.SubjectGroupAthleteAdded, events.GroupAthleteAddedPayload{
		GroupID:   groupID,
		GroupName: group.Name,
		AthleteID: athleteID,
		CoachID:   coachID,
	})
	s.publishEvent(events.SubjectGroupAthleteAdded, evt)

	return &resp, nil
}

func (s *Service) RemoveGroupMember(ctx context.Context, coachID, role, groupID, athleteID string) error {
	if role != "coach" {
		return forbidden("Only coaches can remove group members")
	}

	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return notFound("Group not found")
	}
	if group.CoachID != coachID {
		return forbidden("This is not your group")
	}

	if err := s.repo.RemoveGroupMember(ctx, groupID, athleteID); err != nil {
		if err == repository.ErrNotFound {
			return notFound("Athlete not found in group")
		}
		return err
	}

	// Publish event
	evt := events.NewEvent(events.SubjectGroupAthleteRemoved, events.GroupAthleteRemovedPayload{
		GroupID:   groupID,
		GroupName: group.Name,
		AthleteID: athleteID,
		CoachID:   coachID,
	})
	s.publishEvent(events.SubjectGroupAthleteRemoved, evt)

	return nil
}

func (s *Service) GetGroupMemberIDs(ctx context.Context, groupID string) ([]model.InternalGroupMember, error) {
	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, notFound("Group not found")
	}

	members, err := s.repo.GetGroupMembers(ctx, groupID, "")
	if err != nil {
		return nil, err
	}

	result := make([]model.InternalGroupMember, 0, len(members))
	for _, m := range members {
		result = append(result, model.InternalGroupMember{
			AthleteID: m.AthleteID,
			FullName:  m.FullName,
			Login:     m.Login,
		})
	}
	return result, nil
}

// EnrichConnectionRequest fetches athlete and coach profiles to build a rich response.
func (s *Service) EnrichConnectionRequest(ctx context.Context, cr *model.ConnectionRequest) (*model.ConnectionRequestResponse, error) {
	athlete, err := s.repo.GetProfileByID(ctx, cr.AthleteID)
	if err != nil {
		return nil, err
	}
	coach, err := s.repo.GetProfileByID(ctx, cr.CoachID)
	if err != nil {
		return nil, err
	}

	resp := &model.ConnectionRequestResponse{
		ID:        cr.ID,
		Status:    cr.Status,
		CreatedAt: cr.CreatedAt,
	}
	if cr.UpdatedAt != cr.CreatedAt {
		resp.UpdatedAt = &cr.UpdatedAt
	}
	if athlete != nil {
		resp.Athlete = model.UserBrief{ID: athlete.ID, Login: athlete.Login, FullName: athlete.FullName}
	} else {
		resp.Athlete = model.UserBrief{ID: cr.AthleteID}
	}
	if coach != nil {
		resp.Coach = model.UserBrief{ID: coach.ID, Login: coach.Login, FullName: coach.FullName}
	} else {
		resp.Coach = model.UserBrief{ID: cr.CoachID}
	}
	return resp, nil
}

// ──────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────

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

// IsServiceError checks if an error is a ServiceError and returns it.
func IsServiceError(err error) (*ServiceError, bool) {
	if se, ok := err.(*ServiceError); ok {
		return se, true
	}
	return nil, false
}

