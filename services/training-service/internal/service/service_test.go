package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coach-link/platform/services/training-service/internal/client"
	"github.com/coach-link/platform/services/training-service/internal/model"
	"github.com/coach-link/platform/services/training-service/internal/service"
)

// ── Mock: EventPublisher ───────────────────────

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ string, _ []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	return &nats.PubAck{}, nil
}

// ── Mock: UserServiceClient ────────────────────
// Fields let individual tests control returned group members / errors.
// Zero value behaves like the original always-succeeds stub.

type mockUserClient struct {
	groupMembers    []client.GroupMemberInfo
	groupMembersErr error
}

func (m *mockUserClient) GetUserByID(_ context.Context, id string) (*client.GroupMemberInfo, error) {
	return &client.GroupMemberInfo{AthleteID: id, FullName: "Test User", Login: "test"}, nil
}

func (m *mockUserClient) GetGroupMembers(_ context.Context, _ string) ([]client.GroupMemberInfo, error) {
	return m.groupMembers, m.groupMembersErr
}

// ── Mock: TrainingRepository ───────────────────
// All methods return zero values by default; tests override via fields.

type mockRepo struct {
	assignment    *model.AssignmentRow
	assignmentErr error

	report    *model.TrainingReport
	reportErr error

	reportExists    bool
	reportExistsErr error

	template    *model.TrainingTemplate
	templateErr error

	updateStatusErr error
	deleteAssignErr error
	createReportErr error
	updateTmplErr   error
	deleteTmplErr   error

	// Call recorders for CreatePlan flow.
	createdPlans       int
	createdAssignments int
	assignedAthleteIDs []string
	createdTemplates   int
}

func (m *mockRepo) GetAssignmentByID(_ context.Context, _ string) (*model.AssignmentRow, error) {
	return m.assignment, m.assignmentErr
}
func (m *mockRepo) UpdateAssignmentStatus(_ context.Context, _, _ string) error {
	return m.updateStatusErr
}
func (m *mockRepo) DeleteAssignment(_ context.Context, _ string) (*model.TrainingAssignment, error) {
	if m.deleteAssignErr != nil {
		return nil, m.deleteAssignErr
	}
	return &model.TrainingAssignment{}, nil
}
func (m *mockRepo) CreateReport(_ context.Context, _ *model.TrainingReport) error {
	return m.createReportErr
}
func (m *mockRepo) GetReportByAssignmentID(_ context.Context, _ string) (*model.TrainingReport, error) {
	return m.report, m.reportErr
}
func (m *mockRepo) ReportExists(_ context.Context, _ string) (bool, error) {
	return m.reportExists, m.reportExistsErr
}
func (m *mockRepo) GetTemplateByID(_ context.Context, _ string) (*model.TrainingTemplate, error) {
	return m.template, m.templateErr
}
func (m *mockRepo) UpdateTemplate(_ context.Context, _ string, _, _ *string) error {
	return m.updateTmplErr
}
func (m *mockRepo) DeleteTemplate(_ context.Context, _ string) error {
	return m.deleteTmplErr
}

func (m *mockRepo) CreatePlan(_ context.Context, p *model.TrainingPlan) error {
	m.createdPlans++
	p.ID = "plan-1"
	return nil
}
func (m *mockRepo) GetGroupPlans(_ context.Context, _, _ string, _ bool, _, _ int) ([]model.GroupPlanRow, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) CreateAssignment(_ context.Context, a *model.TrainingAssignment) error {
	m.createdAssignments++
	a.ID = "assign-" + a.AthleteID
	m.assignedAthleteIDs = append(m.assignedAthleteIDs, a.AthleteID)
	return nil
}
func (m *mockRepo) GetCoachAssignments(_ context.Context, _ string, _ model.AssignmentFilter) ([]model.AssignmentRow, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) GetAthleteAssignments(_ context.Context, _ string, _, _ int) ([]model.AssignmentRow, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) GetArchivedAssignments(_ context.Context, _ string, _ model.AssignmentFilter) ([]model.AssignmentRow, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) GetTemplates(_ context.Context, _, _ string, _, _ int) ([]model.TrainingTemplate, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) CreateTemplate(_ context.Context, t *model.TrainingTemplate) error {
	m.createdTemplates++
	t.ID = "tmpl-1"
	return nil
}
func (m *mockRepo) GetReportsByAthleteID(_ context.Context, _, _, _ string) ([]model.ReportWithPlan, error) {
	return nil, nil
}
func (m *mockRepo) GetAthleteStats(_ context.Context, _ string) (*model.AthleteStats, error) {
	return nil, nil
}
func (m *mockRepo) GetCoachAthleteIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockRepo) GetCoachOverviewStats(_ context.Context, _ string) (*model.CoachOverviewStats, error) {
	return nil, nil
}

// ── Helpers ────────────────────────────────────

func newSvc(repo service.TrainingRepository) *service.Service {
	return service.New(repo, &mockPublisher{}, zerolog.Nop(), &mockUserClient{})
}

// newSvcWithClient lets CreatePlan tests inject a user client that returns
// specific group members.
func newSvcWithClient(repo service.TrainingRepository, uc service.UserServiceClient) *service.Service {
	return service.New(repo, &mockPublisher{}, zerolog.Nop(), uc)
}

func assignedRow(coachID, athleteID string, scheduledDate time.Time) *model.AssignmentRow {
	return &model.AssignmentRow{
		ID:            "assign-1",
		CoachID:       coachID,
		AthleteID:     athleteID,
		Status:        "assigned",
		ScheduledDate: scheduledDate,
		Title:         "Test Plan",
	}
}

func completedRow(coachID, athleteID string) *model.AssignmentRow {
	row := assignedRow(coachID, athleteID, time.Now())
	row.Status = "completed"
	return row
}

// ── ComputeIsOverdue (pure function) ──────────

func TestComputeIsOverdue_AssignedAndPastDeadline_ReturnsTrue(t *testing.T) {
	past := time.Now().UTC().Add(-48 * time.Hour)
	assert.True(t, service.ComputeIsOverdue("assigned", past))
}

func TestComputeIsOverdue_AssignedButFutureDate_ReturnsFalse(t *testing.T) {
	future := time.Now().UTC().Add(24 * time.Hour)
	assert.False(t, service.ComputeIsOverdue("assigned", future))
}

func TestComputeIsOverdue_CompletedStatus_ReturnsFalse(t *testing.T) {
	past := time.Now().UTC().Add(-48 * time.Hour)
	assert.False(t, service.ComputeIsOverdue("completed", past))
}

func TestComputeIsOverdue_ArchivedStatus_ReturnsFalse(t *testing.T) {
	past := time.Now().UTC().Add(-48 * time.Hour)
	assert.False(t, service.ComputeIsOverdue("archived", past))
}

func TestComputeIsOverdue_JustAtDeadline_ReturnsFalse(t *testing.T) {
	// scheduled_date + exactly 24h = now → NOT overdue yet
	notYet := time.Now().UTC().Add(-24 * time.Hour).Add(time.Minute)
	assert.False(t, service.ComputeIsOverdue("assigned", notYet))
}

// ── GetAssignment ──────────────────────────────

func TestGetAssignment_NotFound_Returns404(t *testing.T) {
	svc := newSvc(&mockRepo{assignment: nil})

	_, err := svc.GetAssignment(context.Background(), "coach-1", "coach", "assign-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 404, se.Status)
}

func TestGetAssignment_WrongCoach_Returns403(t *testing.T) {
	row := assignedRow("other-coach", "athlete-1", time.Now())
	svc := newSvc(&mockRepo{assignment: row})

	_, err := svc.GetAssignment(context.Background(), "coach-1", "coach", "assign-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestGetAssignment_WrongAthlete_Returns403(t *testing.T) {
	row := assignedRow("coach-1", "other-athlete", time.Now())
	svc := newSvc(&mockRepo{assignment: row})

	_, err := svc.GetAssignment(context.Background(), "athlete-1", "athlete", "assign-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestGetAssignment_CoachOwnsAssignment_ReturnsRow(t *testing.T) {
	row := assignedRow("coach-1", "athlete-1", time.Now())
	svc := newSvc(&mockRepo{assignment: row})

	result, err := svc.GetAssignment(context.Background(), "coach-1", "coach", "assign-1")
	require.NoError(t, err)
	assert.Equal(t, "assign-1", result.ID)
}

// ── ArchiveAssignment ──────────────────────────

func TestArchiveAssignment_NotCompleted_Returns400(t *testing.T) {
	row := assignedRow("coach-1", "athlete-1", time.Now())
	svc := newSvc(&mockRepo{assignment: row})

	err := svc.ArchiveAssignment(context.Background(), "coach-1", "assign-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
	assert.Equal(t, "ASSIGNMENT_NOT_COMPLETED", se.Code)
}

func TestArchiveAssignment_WrongCoach_Returns403(t *testing.T) {
	row := completedRow("other-coach", "athlete-1")
	svc := newSvc(&mockRepo{assignment: row})

	err := svc.ArchiveAssignment(context.Background(), "coach-1", "assign-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestArchiveAssignment_Success(t *testing.T) {
	row := completedRow("coach-1", "athlete-1")
	svc := newSvc(&mockRepo{assignment: row})

	err := svc.ArchiveAssignment(context.Background(), "coach-1", "assign-1")
	require.NoError(t, err)
}

// ── SubmitReport ───────────────────────────────

func TestSubmitReport_WrongAthlete_Returns403(t *testing.T) {
	row := assignedRow("coach-1", "other-athlete", time.Now())
	svc := newSvc(&mockRepo{assignment: row})

	_, err := svc.SubmitReport(context.Background(), "athlete-1", "assign-1", model.CreateReportRequest{
		Content: "done", DurationMinutes: 60, PerceivedEffort: 7,
	})
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestSubmitReport_AlreadyExists_Returns409(t *testing.T) {
	row := assignedRow("coach-1", "athlete-1", time.Now())
	svc := newSvc(&mockRepo{assignment: row, reportExists: true})

	_, err := svc.SubmitReport(context.Background(), "athlete-1", "assign-1", model.CreateReportRequest{
		Content: "done", DurationMinutes: 60, PerceivedEffort: 7,
	})
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 409, se.Status)
	assert.Equal(t, "REPORT_ALREADY_EXISTS", se.Code)
}

func TestSubmitReport_AssignmentNotFound_Returns404(t *testing.T) {
	svc := newSvc(&mockRepo{assignment: nil})

	_, err := svc.SubmitReport(context.Background(), "athlete-1", "assign-1", model.CreateReportRequest{
		Content: "done", DurationMinutes: 60, PerceivedEffort: 7,
	})
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 404, se.Status)
}

func TestSubmitReport_NonAssignedStatus_Returns400(t *testing.T) {
	row := completedRow("coach-1", "athlete-1")
	svc := newSvc(&mockRepo{assignment: row})

	_, err := svc.SubmitReport(context.Background(), "athlete-1", "assign-1", model.CreateReportRequest{
		Content: "done", DurationMinutes: 60, PerceivedEffort: 7,
	})
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

// ── Template ownership ─────────────────────────

func TestGetTemplate_WrongCoach_Returns403(t *testing.T) {
	tmpl := &model.TrainingTemplate{ID: "t1", CoachID: "other-coach", Title: "Plan A"}
	svc := newSvc(&mockRepo{template: tmpl})

	_, err := svc.GetTemplate(context.Background(), "coach-1", "t1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestDeleteTemplate_WrongCoach_Returns403(t *testing.T) {
	tmpl := &model.TrainingTemplate{ID: "t1", CoachID: "other-coach", Title: "Plan A"}
	svc := newSvc(&mockRepo{template: tmpl})

	err := svc.DeleteTemplate(context.Background(), "coach-1", "t1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestUpdateTemplate_NoFields_Returns400(t *testing.T) {
	tmpl := &model.TrainingTemplate{ID: "t1", CoachID: "coach-1", Title: "Plan A"}
	svc := newSvc(&mockRepo{template: tmpl})

	_, err := svc.UpdateTemplate(context.Background(), "coach-1", "t1", model.UpdateTemplateRequest{})
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

func TestUpdateTemplate_NotFound_Returns404(t *testing.T) {
	svc := newSvc(&mockRepo{template: nil})

	title := "New Title"
	_, err := svc.UpdateTemplate(context.Background(), "coach-1", "t1", model.UpdateTemplateRequest{Title: &title})
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 404, se.Status)
}

func TestDeleteAssignment_WrongCoach_Returns403(t *testing.T) {
	row := assignedRow("other-coach", "athlete-1", time.Now())
	svc := newSvc(&mockRepo{assignment: row})

	err := svc.DeleteAssignment(context.Background(), "coach-1", "assign-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}

func TestDeleteAssignment_NotFound_Returns404(t *testing.T) {
	svc := newSvc(&mockRepo{assignment: nil})

	err := svc.DeleteAssignment(context.Background(), "coach-1", "assign-1")
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 404, se.Status)
}

// ── CreatePlan ─────────────────────────────────

func planReq(date string, athleteIDs []string) model.CreateTrainingPlanRequest {
	return model.CreateTrainingPlanRequest{
		Title:         "Interval Session",
		Description:   "8x400m",
		ScheduledDate: date,
		AthleteIDs:    athleteIDs,
	}
}

func TestCreatePlan_NoAthletesNoGroup_Returns400(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach", planReq("2026-07-01", nil))
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

func TestCreatePlan_InvalidDate_Returns400(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach", planReq("01-07-2026", []string{"a1"}))
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

func TestCreatePlan_MultipleAthletes_CreatesAssignmentEach(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(repo)

	resp, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach",
		planReq("2026-07-01", []string{"a1", "a2", "a3"}))
	require.NoError(t, err)

	assert.Equal(t, 1, repo.createdPlans, "exactly one plan created")
	assert.Equal(t, 3, repo.createdAssignments, "one assignment per athlete")
	assert.ElementsMatch(t, []string{"a1", "a2", "a3"}, repo.assignedAthleteIDs)
	assert.Len(t, resp.Assignments, 3)
}

func TestCreatePlan_DuplicateAthleteIDs_Deduplicated(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(repo)

	_, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach",
		planReq("2026-07-01", []string{"a1", "a1", "a2"}))
	require.NoError(t, err)

	// athleteMap de-duplicates by ID, so 2 distinct assignments, not 3.
	assert.Equal(t, 2, repo.createdAssignments)
	assert.ElementsMatch(t, []string{"a1", "a2"}, repo.assignedAthleteIDs)
}

func TestCreatePlan_GroupExpandsToMembers(t *testing.T) {
	repo := &mockRepo{}
	uc := &mockUserClient{groupMembers: []client.GroupMemberInfo{
		{AthleteID: "m1", FullName: "Member One", Login: "m1"},
		{AthleteID: "m2", FullName: "Member Two", Login: "m2"},
	}}
	svc := newSvcWithClient(repo, uc)

	groupID := "group-1"
	req := model.CreateTrainingPlanRequest{
		Title: "Group Plan", Description: "tempo run",
		ScheduledDate: "2026-07-01", GroupID: &groupID,
	}

	_, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach", req)
	require.NoError(t, err)

	assert.Equal(t, 2, repo.createdAssignments, "one assignment per group member")
	assert.ElementsMatch(t, []string{"m1", "m2"}, repo.assignedAthleteIDs)
}

func TestCreatePlan_GroupMembersFetchFails_Returns400(t *testing.T) {
	repo := &mockRepo{}
	uc := &mockUserClient{groupMembersErr: errors.New("user service unavailable")}
	svc := newSvcWithClient(repo, uc)

	groupID := "group-1"
	req := model.CreateTrainingPlanRequest{
		Title: "Group Plan", Description: "tempo run",
		ScheduledDate: "2026-07-01", GroupID: &groupID,
	}

	_, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach", req)
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 400, se.Status)
}

func TestCreatePlan_SaveAsTemplate_CreatesTemplate(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(repo)

	req := planReq("2026-07-01", []string{"a1"})
	req.SaveAsTemplate = true

	resp, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach", req)
	require.NoError(t, err)

	assert.Equal(t, 1, repo.createdTemplates, "template persisted when save_as_template is true")
	require.NotNil(t, resp.Template)
	assert.Equal(t, "Interval Session", resp.Template.Title)
}

func TestCreatePlan_NoTemplate_WhenFlagFalse(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(repo)

	resp, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach",
		planReq("2026-07-01", []string{"a1"}))
	require.NoError(t, err)

	assert.Equal(t, 0, repo.createdTemplates)
	assert.Nil(t, resp.Template)
}

func TestCreatePlan_TemplateWrongCoach_Returns403(t *testing.T) {
	tmpl := &model.TrainingTemplate{ID: "t1", CoachID: "other-coach", Title: "X", Description: "Y"}
	repo := &mockRepo{template: tmpl}
	svc := newSvc(repo)

	templateID := "t1"
	req := model.CreateTrainingPlanRequest{
		Title: "From Template", Description: "desc",
		ScheduledDate: "2026-07-01", AthleteIDs: []string{"a1"}, TemplateID: &templateID,
	}

	_, err := svc.CreatePlan(context.Background(), "coach-1", "Coach", "coach", req)
	require.Error(t, err)

	se, ok := service.IsServiceError(err)
	require.True(t, ok)
	assert.Equal(t, 403, se.Status)
}
