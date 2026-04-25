package service_test

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coach-link/platform/services/user-service/internal/model"
	"github.com/coach-link/platform/services/user-service/internal/repository"
	"github.com/coach-link/platform/services/user-service/internal/service"
)

// ── Mock: UserEventPublisher ───────────────────

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ string, _ []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	return &nats.PubAck{}, nil
}

// ── Mock: UserRepository ───────────────────────
// All methods return zero/nil by default; tests override specific fields.

type mockRepo struct {
	profile         *model.UserProfile
	profileErr      error
	profiles        []model.UserProfile
	searchTotal     int
	searchErr       error
	connRequest     *model.ConnectionRequest
	connRequestErr  error
	hasRelation     bool
	hasRelationErr  error
	group           *model.TrainingGroup
	groupErr        error
	updateGroupErr  error
	deleteGroupErr  error
}

func (m *mockRepo) GetProfileByID(_ context.Context, _ string) (*model.UserProfile, error) {
	return m.profile, m.profileErr
}
func (m *mockRepo) SearchProfiles(_ context.Context, _, _ string, _, _ int) ([]model.UserProfile, int, error) {
	return m.profiles, m.searchTotal, m.searchErr
}
func (m *mockRepo) CreateConnectionRequest(_ context.Context, _, _ string) (*model.ConnectionRequest, error) {
	return m.connRequest, m.connRequestErr
}
func (m *mockRepo) GetConnectionRequestByID(_ context.Context, _ string) (*model.ConnectionRequest, error) {
	return m.connRequest, m.connRequestErr
}
func (m *mockRepo) GetIncomingRequests(_ context.Context, _, _ string, _, _ int) ([]model.ConnectionRequest, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) GetOutgoingRequest(_ context.Context, _ string) (*model.ConnectionRequest, error) {
	return nil, nil
}
func (m *mockRepo) UpdateConnectionRequestStatus(_ context.Context, _, _ string) error { return nil }
func (m *mockRepo) CreateRelation(_ context.Context, _, _ string) error                { return nil }
func (m *mockRepo) GetRelationByAthleteID(_ context.Context, _ string) (*model.CoachAthleteRelation, error) {
	return nil, nil
}
func (m *mockRepo) GetAthletes(_ context.Context, _, _ string, _, _ int) ([]model.AthleteInfo, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) DeleteRelation(_ context.Context, _, _ string) error { return nil }
func (m *mockRepo) HasRelation(_ context.Context, _, _ string) (bool, error) {
	return m.hasRelation, m.hasRelationErr
}
func (m *mockRepo) CreateGroup(_ context.Context, _, _ string) (*model.TrainingGroup, error) {
	return m.group, m.groupErr
}
func (m *mockRepo) GetGroupByID(_ context.Context, _ string) (*model.TrainingGroup, error) {
	return m.group, m.groupErr
}
func (m *mockRepo) GetCoachGroups(_ context.Context, _ string, _, _ int) ([]model.TrainingGroupSummary, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) GetAthleteGroups(_ context.Context, _ string, _, _ int) ([]model.TrainingGroupSummary, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) UpdateGroup(_ context.Context, _, _ string) error   { return m.updateGroupErr }
func (m *mockRepo) DeleteGroup(_ context.Context, _ string) error      { return m.deleteGroupErr }
func (m *mockRepo) AddGroupMember(_ context.Context, _, _ string) error { return nil }
func (m *mockRepo) RemoveGroupMember(_ context.Context, _, _ string) error { return nil }
func (m *mockRepo) GetGroupMembers(_ context.Context, _, _ string) ([]model.GroupMember, error) {
	return nil, nil
}
func (m *mockRepo) GetGroupMemberIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockRepo) CreateProfile(_ context.Context, _ model.UserProfile) error { return nil }

// ── Helpers ────────────────────────────────────

func newSvc(repo service.UserRepository) *service.Service {
	return service.New(repo, &mockPublisher{}, zerolog.Nop())
}

// ── GetProfile ─────────────────────────────────

func TestGetProfile_NotFound_Returns404(t *testing.T) {
	svc := newSvc(&mockRepo{profile: nil})

	_, err := svc.GetProfile(context.Background(), "user-99")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 404, se.Status)
}

func TestGetProfile_Success(t *testing.T) {
	p := &model.UserProfile{ID: "user-1", Login: "alice", Role: "athlete"}
	svc := newSvc(&mockRepo{profile: p})

	result, err := svc.GetProfile(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "alice", result.Login)
}

// ── SearchUsers ────────────────────────────────

func TestSearchUsers_QueryTooShort_Returns400(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, _, err := svc.SearchUsers(context.Background(), "a", "", 1, 20)
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

func TestSearchUsers_EmptyQuery_Returns400(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, _, err := svc.SearchUsers(context.Background(), "", "", 1, 20)
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

func TestSearchUsers_ValidQuery_CallsRepo(t *testing.T) {
	profiles := []model.UserProfile{
		{ID: "u1", Login: "ivan123", FullName: "Иван Петров", Role: "athlete"},
	}
	svc := newSvc(&mockRepo{profiles: profiles, searchTotal: 1})

	results, total, err := svc.SearchUsers(context.Background(), "ivan", "athlete", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, results, 1)
	assert.Equal(t, "ivan123", results[0].Login)
}

// ── SendConnectionRequest ──────────────────────

func TestSendConnectionRequest_NonAthlete_Returns403(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, err := svc.SendConnectionRequest(context.Background(), "coach-1", "coach-2", "coach")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestSendConnectionRequest_CoachNotFound_Returns404(t *testing.T) {
	svc := newSvc(&mockRepo{profile: nil})

	_, err := svc.SendConnectionRequest(context.Background(), "athlete-1", "nonexistent-coach", "athlete")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 404, se.Status)
}

func TestSendConnectionRequest_TargetIsNotCoach_Returns400(t *testing.T) {
	notACoach := &model.UserProfile{ID: "user-2", Role: "athlete"}
	svc := newSvc(&mockRepo{profile: notACoach})

	_, err := svc.SendConnectionRequest(context.Background(), "athlete-1", "user-2", "athlete")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

// ── HasCoachAthleteRelation ────────────────────

func TestHasCoachAthleteRelation_ExistingRelation_ReturnsTrue(t *testing.T) {
	svc := newSvc(&mockRepo{hasRelation: true})

	has, err := svc.HasCoachAthleteRelation(context.Background(), "coach-1", "athlete-1")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestHasCoachAthleteRelation_NoRelation_ReturnsFalse(t *testing.T) {
	svc := newSvc(&mockRepo{hasRelation: false})

	has, err := svc.HasCoachAthleteRelation(context.Background(), "coach-1", "athlete-99")
	require.NoError(t, err)
	assert.False(t, has)
}

// ── CreateGroup ────────────────────────────────

func TestCreateGroup_NonCoach_Returns403(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, err := svc.CreateGroup(context.Background(), "user-1", "athlete", "Sprint Group")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestCreateGroup_CoachSuccess(t *testing.T) {
	group := &model.TrainingGroup{ID: "g1", Name: "Sprint Group"}
	svc := newSvc(&mockRepo{group: group})

	result, err := svc.CreateGroup(context.Background(), "coach-1", "coach", "Sprint Group")
	require.NoError(t, err)
	assert.Equal(t, "Sprint Group", result.Name)
}

// ── DeleteGroup ────────────────────────────────

func TestDeleteGroup_NonCoach_Returns403(t *testing.T) {
	svc := newSvc(&mockRepo{})

	err := svc.DeleteGroup(context.Background(), "user-1", "athlete", "g1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestDeleteGroup_GroupNotFound_Returns404(t *testing.T) {
	svc := newSvc(&mockRepo{group: nil})

	err := svc.DeleteGroup(context.Background(), "coach-1", "coach", "nonexistent")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 404, se.Status)
}

func TestDeleteGroup_WrongCoach_Returns403(t *testing.T) {
	group := &model.TrainingGroup{ID: "g1", CoachID: "other-coach", Name: "Group"}
	svc := newSvc(&mockRepo{group: group})

	err := svc.DeleteGroup(context.Background(), "coach-1", "coach", "g1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}
