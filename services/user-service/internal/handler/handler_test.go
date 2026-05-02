package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coach-link/platform/services/user-service/internal/handler"
	"github.com/coach-link/platform/services/user-service/internal/model"
	"github.com/coach-link/platform/services/user-service/internal/service"
)

// These tests drive the full HTTP path through Echo (header auth → Bind →
// validation → service → error mapping), exactly as cmd/main.go wires it.
// The repository is the only seam, replaced by a configurable mock.

// ── Mocks ──────────────────────────────────────

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ string, _ []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	return &nats.PubAck{}, nil
}

// mockRepo returns zero/nil by default; tests override specific fields.
type mockRepo struct {
	profile *model.UserProfile
	group   *model.TrainingGroup
	groupErr error
}

func (m *mockRepo) GetProfileByID(_ context.Context, _ string) (*model.UserProfile, error) {
	return m.profile, nil
}
func (m *mockRepo) SearchProfiles(_ context.Context, _, _ string, _, _ int) ([]model.UserProfile, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) CreateConnectionRequest(_ context.Context, _, _ string) (*model.ConnectionRequest, error) {
	return nil, nil
}
func (m *mockRepo) GetConnectionRequestByID(_ context.Context, _ string) (*model.ConnectionRequest, error) {
	return nil, nil
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
	return false, nil
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
func (m *mockRepo) UpdateGroup(_ context.Context, _, _ string) error        { return nil }
func (m *mockRepo) DeleteGroup(_ context.Context, _ string) error           { return nil }
func (m *mockRepo) AddGroupMember(_ context.Context, _, _ string) error     { return nil }
func (m *mockRepo) RemoveGroupMember(_ context.Context, _, _ string) error  { return nil }
func (m *mockRepo) GetGroupMembers(_ context.Context, _, _ string) ([]model.GroupMember, error) {
	return nil, nil
}
func (m *mockRepo) GetGroupMemberIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockRepo) CreateProfile(_ context.Context, _ model.UserProfile) error { return nil }

// ── Harness ────────────────────────────────────

func newServer(repo service.UserRepository) *echo.Echo {
	svc := service.New(repo, &mockPublisher{}, zerolog.Nop())
	h := handler.New(svc)
	e := echo.New()
	e.HideBanner = true
	handler.RegisterRoutes(e, h)
	return e
}

type reqOpts struct {
	userID string
	role   string
	body   string
}

func do(e *echo.Echo, method, path string, o reqOpts) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(o.body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if o.userID != "" {
		req.Header.Set("X-User-ID", o.userID)
	}
	if o.role != "" {
		req.Header.Set("X-User-Role", o.role)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// ── Auth header enforcement ────────────────────

func TestSearchUsers_MissingAuthHeaders_Returns401(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodGet, "/api/v1/users/search?q=ivan", reqOpts{})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateGroup_MissingAuthHeaders_Returns401(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodPost, "/api/v1/groups", reqOpts{body: `{"name":"X"}`})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ── SearchUsers ────────────────────────────────

func TestSearchUsers_EmptyQuery_Returns400(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodGet, "/api/v1/users/search?q=", reqOpts{userID: "u1", role: "coach"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSearchUsers_TooShortQuery_Returns400(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodGet, "/api/v1/users/search?q=a", reqOpts{userID: "u1", role: "coach"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSearchUsers_ValidQuery_Returns200(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodGet, "/api/v1/users/search?q=ivan&role=athlete", reqOpts{userID: "u1", role: "coach"})
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ── CreateGroup ────────────────────────────────

func TestCreateGroup_EmptyName_Returns400(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodPost, "/api/v1/groups", reqOpts{userID: "coach-1", role: "coach", body: `{"name":""}`})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "name is required")
}

func TestCreateGroup_MalformedJSON_Returns400(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodPost, "/api/v1/groups", reqOpts{userID: "coach-1", role: "coach", body: `{"name":`})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateGroup_AthleteRole_Returns403(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodPost, "/api/v1/groups", reqOpts{userID: "u1", role: "athlete", body: `{"name":"Sprint"}`})
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCreateGroup_Coach_Returns201(t *testing.T) {
	group := &model.TrainingGroup{ID: "g1", Name: "Sprint Group"}
	e := newServer(&mockRepo{group: group})
	rec := do(e, http.MethodPost, "/api/v1/groups", reqOpts{userID: "coach-1", role: "coach", body: `{"name":"Sprint Group"}`})
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "Sprint Group")
}

// ── SendConnectionRequest ──────────────────────

func TestSendConnectionRequest_MissingCoachID_Returns400(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodPost, "/api/v1/connections/request", reqOpts{userID: "athlete-1", role: "athlete", body: `{}`})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "coach_id is required")
}

func TestSendConnectionRequest_CoachRole_Returns403(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodPost, "/api/v1/connections/request", reqOpts{userID: "coach-1", role: "coach", body: `{"coach_id":"coach-2"}`})
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// ── DeleteGroup ────────────────────────────────

func TestDeleteGroup_AthleteRole_Returns403(t *testing.T) {
	e := newServer(&mockRepo{})
	rec := do(e, http.MethodDelete, "/api/v1/groups/g1", reqOpts{userID: "u1", role: "athlete"})
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteGroup_NotFound_Returns404(t *testing.T) {
	e := newServer(&mockRepo{group: nil})
	rec := do(e, http.MethodDelete, "/api/v1/groups/missing", reqOpts{userID: "coach-1", role: "coach"})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
