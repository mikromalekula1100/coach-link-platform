package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/coach-link/platform/services/training-service/internal/model"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ──────────────────────────────────────────────
// Training Plans
// ──────────────────────────────────────────────

func (r *Repository) CreatePlan(ctx context.Context, plan *model.TrainingPlan) error {
	query := `
		INSERT INTO training_plans (coach_id, title, description, scheduled_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		plan.CoachID, plan.Title, plan.Description, plan.ScheduledDate,
	).Scan(&plan.ID, &plan.CreatedAt)
}

func (r *Repository) GetPlanByID(ctx context.Context, id string) (*model.TrainingPlan, error) {
	var plan model.TrainingPlan
	query := `SELECT id, coach_id, title, description, scheduled_date, created_at FROM training_plans WHERE id = $1`
	err := r.db.GetContext(ctx, &plan, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *Repository) DeletePlan(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM training_plans WHERE id = $1`, id)
	return err
}

// ──────────────────────────────────────────────
// Training Assignments
// ──────────────────────────────────────────────

func (r *Repository) CreateAssignment(ctx context.Context, a *model.TrainingAssignment) error {
	query := `
		INSERT INTO training_assignments (plan_id, athlete_id, coach_id, athlete_full_name, athlete_login, coach_full_name, coach_login)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, assigned_at`
	return r.db.QueryRowxContext(ctx, query,
		a.PlanID, a.AthleteID, a.CoachID, a.AthleteFullName, a.AthleteLogin, a.CoachFullName, a.CoachLogin,
	).Scan(&a.ID, &a.AssignedAt)
}

func (r *Repository) GetAssignmentByID(ctx context.Context, id string) (*model.AssignmentRow, error) {
	var row model.AssignmentRow
	query := `
		SELECT
			a.id, a.plan_id, a.athlete_id, a.coach_id,
			a.athlete_full_name, a.athlete_login,
			a.coach_full_name, a.coach_login,
			a.status, a.assigned_at, a.completed_at, a.archived_at,
			p.title, p.description, p.scheduled_date,
			EXISTS(SELECT 1 FROM training_reports tr WHERE tr.assignment_id = a.id) AS has_report
		FROM training_assignments a
		JOIN training_plans p ON p.id = a.plan_id
		WHERE a.id = $1`
	err := r.db.GetContext(ctx, &row, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) GetCoachAssignments(ctx context.Context, coachID string, filter model.AssignmentFilter) ([]model.AssignmentRow, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("a.coach_id = $%d", argIdx))
	args = append(args, coachID)
	argIdx++

	conditions = append(conditions, "a.status != 'archived'")

	if filter.AthleteFullName != "" {
		conditions = append(conditions, fmt.Sprintf("a.athlete_full_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.AthleteFullName+"%")
		argIdx++
	}
	if filter.AthleteLogin != "" {
		conditions = append(conditions, fmt.Sprintf("a.athlete_login ILIKE $%d", argIdx))
		args = append(args, "%"+filter.AthleteLogin+"%")
		argIdx++
	}
	if filter.DateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("p.scheduled_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		conditions = append(conditions, fmt.Sprintf("p.scheduled_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM training_assignments a
		JOIN training_plans p ON p.id = a.plan_id
		WHERE %s`, where)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Order
	orderClause := "p.scheduled_date DESC, a.assigned_at DESC"
	switch filter.SortBy {
	case "date_asc":
		orderClause = "p.scheduled_date ASC, a.assigned_at ASC"
	case "date_desc":
		orderClause = "p.scheduled_date DESC, a.assigned_at DESC"
	case "name_asc":
		orderClause = "a.athlete_full_name ASC"
	case "name_desc":
		orderClause = "a.athlete_full_name DESC"
	}

	offset := (filter.Page - 1) * filter.PageSize

	dataQuery := fmt.Sprintf(`
		SELECT
			a.id, a.plan_id, a.athlete_id, a.coach_id,
			a.athlete_full_name, a.athlete_login,
			a.coach_full_name, a.coach_login,
			a.status, a.assigned_at, a.completed_at, a.archived_at,
			p.title, p.description, p.scheduled_date,
			EXISTS(SELECT 1 FROM training_reports tr WHERE tr.assignment_id = a.id) AS has_report
		FROM training_assignments a
		JOIN training_plans p ON p.id = a.plan_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, where, orderClause, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)

	var rows []model.AssignmentRow
	if err := r.db.SelectContext(ctx, &rows, dataQuery, args...); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) GetAthleteAssignments(ctx context.Context, athleteID string, page, pageSize int) ([]model.AssignmentRow, int, error) {
	where := "a.athlete_id = $1 AND a.status = 'assigned'"

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM training_assignments a
		JOIN training_plans p ON p.id = a.plan_id
		WHERE %s`, where)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, athleteID); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize

	dataQuery := fmt.Sprintf(`
		SELECT
			a.id, a.plan_id, a.athlete_id, a.coach_id,
			a.athlete_full_name, a.athlete_login,
			a.coach_full_name, a.coach_login,
			a.status, a.assigned_at, a.completed_at, a.archived_at,
			p.title, p.description, p.scheduled_date,
			EXISTS(SELECT 1 FROM training_reports tr WHERE tr.assignment_id = a.id) AS has_report
		FROM training_assignments a
		JOIN training_plans p ON p.id = a.plan_id
		WHERE %s
		ORDER BY p.scheduled_date ASC, a.assigned_at ASC
		LIMIT $2 OFFSET $3`, where)

	var rows []model.AssignmentRow
	if err := r.db.SelectContext(ctx, &rows, dataQuery, athleteID, pageSize, offset); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) GetArchivedAssignments(ctx context.Context, coachID string, filter model.AssignmentFilter) ([]model.AssignmentRow, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("a.coach_id = $%d", argIdx))
	args = append(args, coachID)
	argIdx++

	conditions = append(conditions, "a.status = 'archived'")

	if filter.AthleteFullName != "" {
		conditions = append(conditions, fmt.Sprintf("a.athlete_full_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.AthleteFullName+"%")
		argIdx++
	}
	if filter.AthleteLogin != "" {
		conditions = append(conditions, fmt.Sprintf("a.athlete_login ILIKE $%d", argIdx))
		args = append(args, "%"+filter.AthleteLogin+"%")
		argIdx++
	}
	if filter.DateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("p.scheduled_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		conditions = append(conditions, fmt.Sprintf("p.scheduled_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM training_assignments a
		JOIN training_plans p ON p.id = a.plan_id
		WHERE %s`, where)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	orderClause := "a.archived_at DESC"
	switch filter.SortBy {
	case "date_asc":
		orderClause = "p.scheduled_date ASC"
	case "date_desc":
		orderClause = "p.scheduled_date DESC"
	case "name_asc":
		orderClause = "a.athlete_full_name ASC"
	case "name_desc":
		orderClause = "a.athlete_full_name DESC"
	}

	offset := (filter.Page - 1) * filter.PageSize

	dataQuery := fmt.Sprintf(`
		SELECT
			a.id, a.plan_id, a.athlete_id, a.coach_id,
			a.athlete_full_name, a.athlete_login,
			a.coach_full_name, a.coach_login,
			a.status, a.assigned_at, a.completed_at, a.archived_at,
			p.title, p.description, p.scheduled_date,
			EXISTS(SELECT 1 FROM training_reports tr WHERE tr.assignment_id = a.id) AS has_report
		FROM training_assignments a
		JOIN training_plans p ON p.id = a.plan_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, where, orderClause, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)

	var rows []model.AssignmentRow
	if err := r.db.SelectContext(ctx, &rows, dataQuery, args...); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) UpdateAssignmentStatus(ctx context.Context, id, status string) error {
	var query string
	now := time.Now().UTC()

	switch status {
	case "completed":
		query = `UPDATE training_assignments SET status = $1, completed_at = $2 WHERE id = $3`
	case "archived":
		query = `UPDATE training_assignments SET status = $1, archived_at = $2 WHERE id = $3`
	default:
		query = `UPDATE training_assignments SET status = $1 WHERE id = $2`
		_, err := r.db.ExecContext(ctx, query, status, id)
		return err
	}

	_, err := r.db.ExecContext(ctx, query, status, now, id)
	return err
}

func (r *Repository) DeleteAssignment(ctx context.Context, id string) (*model.TrainingAssignment, error) {
	var a model.TrainingAssignment
	query := `
		DELETE FROM training_assignments WHERE id = $1
		RETURNING id, plan_id, athlete_id, coach_id, athlete_full_name, athlete_login,
		          coach_full_name, coach_login, status, assigned_at, completed_at, archived_at`
	err := r.db.QueryRowxContext(ctx, query, id).StructScan(&a)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ──────────────────────────────────────────────
// Training Reports
// ──────────────────────────────────────────────

func (r *Repository) CreateReport(ctx context.Context, report *model.TrainingReport) error {
	query := `
		INSERT INTO training_reports (assignment_id, athlete_id, content, duration_minutes, perceived_effort, max_heart_rate, avg_heart_rate, distance_km)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		report.AssignmentID, report.AthleteID, report.Content,
		report.DurationMinutes, report.PerceivedEffort,
		report.MaxHeartRate, report.AvgHeartRate, report.DistanceKm,
	).Scan(&report.ID, &report.CreatedAt)
}

func (r *Repository) GetReportByAssignmentID(ctx context.Context, assignmentID string) (*model.TrainingReport, error) {
	var report model.TrainingReport
	query := `
		SELECT id, assignment_id, athlete_id, content, duration_minutes, perceived_effort,
		       max_heart_rate, avg_heart_rate, distance_km, created_at
		FROM training_reports
		WHERE assignment_id = $1`
	err := r.db.GetContext(ctx, &report, query, assignmentID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *Repository) ReportExists(ctx context.Context, assignmentID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM training_reports WHERE assignment_id = $1)`
	err := r.db.GetContext(ctx, &exists, query, assignmentID)
	return exists, err
}

// ──────────────────────────────────────────────
// Training Templates
// ──────────────────────────────────────────────

func (r *Repository) CreateTemplate(ctx context.Context, t *model.TrainingTemplate) error {
	query := `
		INSERT INTO training_templates (coach_id, title, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		t.CoachID, t.Title, t.Description,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *Repository) GetTemplateByID(ctx context.Context, id string) (*model.TrainingTemplate, error) {
	var t model.TrainingTemplate
	query := `SELECT id, coach_id, title, description, created_at, updated_at FROM training_templates WHERE id = $1`
	err := r.db.GetContext(ctx, &t, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) GetTemplates(ctx context.Context, coachID, query string, page, pageSize int) ([]model.TrainingTemplate, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("coach_id = $%d", argIdx))
	args = append(args, coachID)
	argIdx++

	if query != "" {
		conditions = append(conditions, fmt.Sprintf("title ILIKE $%d", argIdx))
		args = append(args, "%"+query+"%")
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM training_templates WHERE %s`, where)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize

	dataQuery := fmt.Sprintf(`
		SELECT id, coach_id, title, description, created_at, updated_at
		FROM training_templates
		WHERE %s
		ORDER BY updated_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	var templates []model.TrainingTemplate
	if err := r.db.SelectContext(ctx, &templates, dataQuery, args...); err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

func (r *Repository) UpdateTemplate(ctx context.Context, id string, title, description *string) error {
	var setClauses []string
	var args []interface{}
	argIdx := 1

	if title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *title)
		argIdx++
	}
	if description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *description)
		argIdx++
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++

	args = append(args, id)

	query := fmt.Sprintf(`UPDATE training_templates SET %s WHERE id = $%d`,
		strings.Join(setClauses, ", "), argIdx)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) DeleteTemplate(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM training_templates WHERE id = $1`, id)
	return err
}

// ──────────────────────────────────────────────
// Internal API queries
// ──────────────────────────────────────────────

func (r *Repository) GetReportsByAthleteID(ctx context.Context, athleteID, dateFrom, dateTo string) ([]model.ReportWithPlan, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tr.athlete_id = $%d", argIdx))
	args = append(args, athleteID)
	argIdx++

	if dateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("p.scheduled_date >= $%d", argIdx))
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		conditions = append(conditions, fmt.Sprintf("p.scheduled_date <= $%d", argIdx))
		args = append(args, dateTo)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT
			tr.id, tr.assignment_id, tr.athlete_id, tr.content,
			tr.duration_minutes, tr.perceived_effort,
			tr.max_heart_rate, tr.avg_heart_rate, tr.distance_km,
			tr.created_at,
			p.title, p.scheduled_date
		FROM training_reports tr
		JOIN training_assignments a ON a.id = tr.assignment_id
		JOIN training_plans p ON p.id = a.plan_id
		WHERE %s
		ORDER BY p.scheduled_date DESC`, where)

	var reports []model.ReportWithPlan
	if err := r.db.SelectContext(ctx, &reports, query, args...); err != nil {
		return nil, err
	}
	return reports, nil
}

func (r *Repository) GetAthleteStats(ctx context.Context, athleteID string) (*model.AthleteStats, error) {
	var stats model.AthleteStats

	// Report aggregates
	reportQuery := `
		SELECT
			COUNT(*) AS total_reports,
			COALESCE(SUM(duration_minutes), 0) AS total_duration_minutes,
			COALESCE(AVG(duration_minutes), 0) AS avg_duration_minutes,
			COALESCE(AVG(perceived_effort), 0) AS avg_perceived_effort,
			COALESCE(AVG(avg_heart_rate) FILTER (WHERE avg_heart_rate IS NOT NULL), 0) AS avg_heart_rate,
			MAX(max_heart_rate) AS max_heart_rate_ever,
			COALESCE(SUM(distance_km), 0) AS total_distance_km
		FROM training_reports
		WHERE athlete_id = $1`
	if err := r.db.GetContext(ctx, &stats, reportQuery, athleteID); err != nil {
		return nil, err
	}

	// Assignment counts
	assignmentQuery := `
		SELECT
			COUNT(*) AS total_assignments,
			COUNT(*) FILTER (WHERE status IN ('completed', 'archived')) AS completed_count
		FROM training_assignments
		WHERE athlete_id = $1`
	if err := r.db.GetContext(ctx, &stats, assignmentQuery, athleteID); err != nil {
		return nil, err
	}

	if stats.TotalAssignments > 0 {
		stats.CompletionRate = float64(stats.CompletedCount) / float64(stats.TotalAssignments)
	}

	return &stats, nil
}

func (r *Repository) GetCoachAthleteIDs(ctx context.Context, coachID string) ([]string, error) {
	var ids []string
	query := `SELECT DISTINCT athlete_id FROM training_assignments WHERE coach_id = $1`
	if err := r.db.SelectContext(ctx, &ids, query, coachID); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repository) GetCoachOverviewStats(ctx context.Context, coachID string) (*model.CoachOverviewStats, error) {
	var stats model.CoachOverviewStats

	query := `
		SELECT
			COUNT(DISTINCT a.athlete_id) AS total_athletes,
			COUNT(DISTINCT a.id) AS total_assignments,
			COUNT(DISTINCT tr.id) AS total_reports
		FROM training_assignments a
		LEFT JOIN training_reports tr ON tr.assignment_id = a.id
		WHERE a.coach_id = $1`

	if err := r.db.QueryRowxContext(ctx, query, coachID).Scan(
		&stats.TotalAthletes,
		&stats.TotalAssignments,
		&stats.TotalReports,
	); err != nil {
		return nil, err
	}

	return &stats, nil
}
