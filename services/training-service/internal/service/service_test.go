package service_test

import (
	"context"
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

type mockUserClient struct{}

func (m *mockUserClient) GetUserByID(_ context.Context, _ string) (*client.GroupMemberInfo, error) {
	return &client.GroupMemberInfo{FullName: "Test User", Login: "test"}, nil
}

func (m *mockUserClient) GetGroupMembers(_ context.Context, _ string) ([]client.GroupMemberInfo, error) {
	return nil, nil
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

// Unused in these tests — no-op implementations.
func (m *mockRepo) CreatePlan(_ context.Context, _ *model.TrainingPlan) error { return nil }
func (m *mockRepo) GetGroupPlans(_ context.Context, _, _ string, _ bool, _, _ int) ([]model.GroupPlanRow, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) CreateAssignment(_ context.Context, _ *model.TrainingAssignment) error { return nil }
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
func (m *mockRepo) CreateTemplate(_ context.Context, _ *model.TrainingTemplate) error { return nil }
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
