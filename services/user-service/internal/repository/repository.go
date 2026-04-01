package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/coach-link/platform/services/user-service/internal/model"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ──────────────────────────────────────────────
// User Profiles
// ──────────────────────────────────────────────

func (r *Repository) CreateProfile(ctx context.Context, p model.UserProfile) error {
	const q = `
		INSERT INTO user_profiles (id, login, email, full_name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.Login, p.Email, p.FullName, p.Role, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *Repository) GetProfileByID(ctx context.Context, id string) (*model.UserProfile, error) {
	var p model.UserProfile
	const q = `SELECT id, login, email, full_name, role, created_at, updated_at FROM user_profiles WHERE id = $1`
	if err := r.db.GetContext(ctx, &p, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) SearchProfiles(ctx context.Context, query, role string, page, pageSize int) ([]model.UserProfile, int, error) {
	offset := (page - 1) * pageSize
	likePattern := "%" + query + "%"

	var args []interface{}
	where := "WHERE (full_name ILIKE $1 OR login ILIKE $1)"
	args = append(args, likePattern)

	argIdx := 2
	if role != "" {
		where += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, role)
		argIdx++
	}

	// Count total
	var total int
	countQ := "SELECT COUNT(*) FROM user_profiles " + where
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, err
	}

	// Fetch page
	dataQ := fmt.Sprintf(
		"SELECT id, login, email, full_name, role, created_at, updated_at FROM user_profiles %s ORDER BY full_name ASC LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1,
	)
	args = append(args, pageSize, offset)

	var profiles []model.UserProfile
	if err := r.db.SelectContext(ctx, &profiles, dataQ, args...); err != nil {
		return nil, 0, err
	}
	return profiles, total, nil
}

// ──────────────────────────────────────────────
// Connection Requests
// ──────────────────────────────────────────────

func (r *Repository) CreateConnectionRequest(ctx context.Context, athleteID, coachID string) (*model.ConnectionRequest, error) {
	// Check athlete doesn't already have a coach
	var relCount int
	if err := r.db.GetContext(ctx, &relCount, "SELECT COUNT(*) FROM coach_athlete_relations WHERE athlete_id = $1", athleteID); err != nil {
		return nil, err
	}
	if relCount > 0 {
		return nil, ErrAlreadyHasCoach
	}

	// Check no pending request to any coach exists
	var pendingCount int
	if err := r.db.GetContext(ctx, &pendingCount,
		"SELECT COUNT(*) FROM connection_requests WHERE athlete_id = $1 AND status = 'pending'", athleteID); err != nil {
		return nil, err
	}
	if pendingCount > 0 {
		return nil, ErrRequestAlreadyExists
	}

	var cr model.ConnectionRequest
	const q = `
		INSERT INTO connection_requests (athlete_id, coach_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, athlete_id, coach_id, status, created_at, updated_at`
	if err := r.db.GetContext(ctx, &cr, q, athleteID, coachID); err != nil {
		return nil, err
	}
	return &cr, nil
}

func (r *Repository) GetConnectionRequestByID(ctx context.Context, id string) (*model.ConnectionRequest, error) {
	var cr model.ConnectionRequest
	const q = `SELECT id, athlete_id, coach_id, status, created_at, updated_at FROM connection_requests WHERE id = $1`
	if err := r.db.GetContext(ctx, &cr, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &cr, nil
}

func (r *Repository) GetIncomingRequests(ctx context.Context, coachID, status string, page, pageSize int) ([]model.ConnectionRequest, int, error) {
	offset := (page - 1) * pageSize

	var args []interface{}
	where := "WHERE coach_id = $1"
	args = append(args, coachID)

	argIdx := 2
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	var total int
	countQ := "SELECT COUNT(*) FROM connection_requests " + where
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, err
	}

	dataQ := fmt.Sprintf(
		"SELECT id, athlete_id, coach_id, status, created_at, updated_at FROM connection_requests %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1,
	)
	args = append(args, pageSize, offset)

	var reqs []model.ConnectionRequest
	if err := r.db.SelectContext(ctx, &reqs, dataQ, args...); err != nil {
		return nil, 0, err
	}
	return reqs, total, nil
}

func (r *Repository) GetOutgoingRequest(ctx context.Context, athleteID string) (*model.ConnectionRequest, error) {
	var cr model.ConnectionRequest
	const q = `SELECT id, athlete_id, coach_id, status, created_at, updated_at
	           FROM connection_requests WHERE athlete_id = $1 AND status = 'pending'
	           ORDER BY created_at DESC LIMIT 1`
	if err := r.db.GetContext(ctx, &cr, q, athleteID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &cr, nil
}

func (r *Repository) UpdateConnectionRequestStatus(ctx context.Context, id, status string) error {
	const q = `UPDATE connection_requests SET status = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, q, status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ──────────────────────────────────────────────
// Coach-Athlete Relations
// ──────────────────────────────────────────────

func (r *Repository) CreateRelation(ctx context.Context, coachID, athleteID string) error {
	const q = `INSERT INTO coach_athlete_relations (coach_id, athlete_id) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, q, coachID, athleteID)
	return err
}

func (r *Repository) GetRelationByAthleteID(ctx context.Context, athleteID string) (*model.CoachAthleteRelation, error) {
	var rel model.CoachAthleteRelation
	const q = `SELECT id, coach_id, athlete_id, created_at FROM coach_athlete_relations WHERE athlete_id = $1`
	if err := r.db.GetContext(ctx, &rel, q, athleteID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rel, nil
}

func (r *Repository) GetAthletes(ctx context.Context, coachID, query string, page, pageSize int) ([]model.AthleteInfo, int, error) {
	offset := (page - 1) * pageSize

	var args []interface{}
	args = append(args, coachID)
	argIdx := 2

	where := "WHERE r.coach_id = $1"
	if query != "" {
		where += fmt.Sprintf(" AND (p.full_name ILIKE $%d OR p.login ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+query+"%")
		argIdx++
	}

	var total int
	countQ := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM coach_athlete_relations r
		JOIN user_profiles p ON p.id = r.athlete_id
		%s`, where)
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, err
	}

	dataQ := fmt.Sprintf(`
		SELECT p.id, p.login, p.full_name, r.created_at AS connected_at
		FROM coach_athlete_relations r
		JOIN user_profiles p ON p.id = r.athlete_id
		%s
		ORDER BY p.full_name ASC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	var athletes []model.AthleteInfo
	if err := r.db.SelectContext(ctx, &athletes, dataQ, args...); err != nil {
		return nil, 0, err
	}
	return athletes, total, nil
}

func (r *Repository) DeleteRelation(ctx context.Context, coachID, athleteID string) error {
	const q = `DELETE FROM coach_athlete_relations WHERE coach_id = $1 AND athlete_id = $2`
	res, err := r.db.ExecContext(ctx, q, coachID, athleteID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) HasRelation(ctx context.Context, coachID, athleteID string) (bool, error) {
	var count int
	const q = `SELECT COUNT(*) FROM coach_athlete_relations WHERE coach_id = $1 AND athlete_id = $2`
	if err := r.db.GetContext(ctx, &count, q, coachID, athleteID); err != nil {
		return false, err
	}
	return count > 0, nil
}

// ──────────────────────────────────────────────
// Training Groups
// ──────────────────────────────────────────────

func (r *Repository) CreateGroup(ctx context.Context, coachID, name string) (*model.TrainingGroup, error) {
	var g model.TrainingGroup
	const q = `
		INSERT INTO training_groups (coach_id, name)
		VALUES ($1, $2)
		RETURNING id, coach_id, name, created_at, updated_at`
	if err := r.db.GetContext(ctx, &g, q, coachID, name); err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repository) GetGroupByID(ctx context.Context, id string) (*model.TrainingGroup, error) {
	var g model.TrainingGroup
	const q = `SELECT id, coach_id, name, created_at, updated_at FROM training_groups WHERE id = $1`
	if err := r.db.GetContext(ctx, &g, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func (r *Repository) GetCoachGroups(ctx context.Context, coachID string, page, pageSize int) ([]model.TrainingGroupSummary, int, error) {
	offset := (page - 1) * pageSize

	var total int
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM training_groups WHERE coach_id = $1", coachID); err != nil {
		return nil, 0, err
	}

	const q = `
		SELECT g.id, g.name, COALESCE(m.cnt, 0) AS members_count, g.created_at
		FROM training_groups g
		LEFT JOIN (
			SELECT group_id, COUNT(*) AS cnt FROM training_group_members GROUP BY group_id
		) m ON m.group_id = g.id
		WHERE g.coach_id = $1
		ORDER BY g.name ASC
		LIMIT $2 OFFSET $3`

	var groups []model.TrainingGroupSummary
	if err := r.db.SelectContext(ctx, &groups, q, coachID, pageSize, offset); err != nil {
		return nil, 0, err
	}
	return groups, total, nil
}

func (r *Repository) GetAthleteGroups(ctx context.Context, athleteID string, page, pageSize int) ([]model.TrainingGroupSummary, int, error) {
	offset := (page - 1) * pageSize

	var total int
	countQ := `
		SELECT COUNT(*)
		FROM training_groups g
		JOIN training_group_members gm ON gm.group_id = g.id
		WHERE gm.athlete_id = $1`
	if err := r.db.GetContext(ctx, &total, countQ, athleteID); err != nil {
		return nil, 0, err
	}

	const q = `
		SELECT g.id, g.name, COALESCE(m.cnt, 0) AS members_count, g.created_at
		FROM training_groups g
		JOIN training_group_members gm ON gm.group_id = g.id
		LEFT JOIN (
			SELECT group_id, COUNT(*) AS cnt FROM training_group_members GROUP BY group_id
		) m ON m.group_id = g.id
		WHERE gm.athlete_id = $1
		ORDER BY g.name ASC
		LIMIT $2 OFFSET $3`

	var groups []model.TrainingGroupSummary
	if err := r.db.SelectContext(ctx, &groups, q, athleteID, pageSize, offset); err != nil {
		return nil, 0, err
	}
	return groups, total, nil
}

func (r *Repository) UpdateGroup(ctx context.Context, id, name string) error {
	const q = `UPDATE training_groups SET name = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, q, name, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteGroup(ctx context.Context, id string) error {
	const q = `DELETE FROM training_groups WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ──────────────────────────────────────────────
// Group Members
// ──────────────────────────────────────────────

func (r *Repository) AddGroupMember(ctx context.Context, groupID, athleteID string) error {
	const q = `INSERT INTO training_group_members (group_id, athlete_id) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, q, groupID, athleteID)
	if err != nil {
		// unique violation → already in group
		return err
	}
	return nil
}

func (r *Repository) RemoveGroupMember(ctx context.Context, groupID, athleteID string) error {
	const q = `DELETE FROM training_group_members WHERE group_id = $1 AND athlete_id = $2`
	res, err := r.db.ExecContext(ctx, q, groupID, athleteID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetGroupMembers(ctx context.Context, groupID, query string) ([]model.GroupMember, error) {
	var args []interface{}
	args = append(args, groupID)
	argIdx := 2

	where := "WHERE gm.group_id = $1"
	if query != "" {
		where += fmt.Sprintf(" AND (p.full_name ILIKE $%d OR p.login ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+query+"%")
	}

	q := fmt.Sprintf(`
		SELECT gm.group_id, gm.athlete_id, p.login, p.full_name, gm.added_at
		FROM training_group_members gm
		JOIN user_profiles p ON p.id = gm.athlete_id
		%s
		ORDER BY p.full_name ASC`, where)

	var members []model.GroupMember
	if err := r.db.SelectContext(ctx, &members, q, args...); err != nil {
		return nil, err
	}
	return members, nil
}

func (r *Repository) GetGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	var ids []string
	const q = `SELECT athlete_id FROM training_group_members WHERE group_id = $1`
	if err := r.db.SelectContext(ctx, &ids, q, groupID); err != nil {
		return nil, err
	}
	return ids, nil
}

// ──────────────────────────────────────────────
// Sentinel errors
// ──────────────────────────────────────────────

type repoError string

func (e repoError) Error() string { return string(e) }

const (
	ErrAlreadyHasCoach      repoError = "athlete already has a coach"
	ErrRequestAlreadyExists repoError = "a pending connection request already exists"
	ErrNotFound             repoError = "not found"
	ErrAlreadyInGroup       repoError = "athlete is already in group"
)
