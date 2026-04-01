package model

import "time"

// ──────────────────────────────────────────────
// Database models (sqlx)
// ──────────────────────────────────────────────

type UserProfile struct {
	ID        string    `db:"id"`
	Login     string    `db:"login"`
	Email     string    `db:"email"`
	FullName  string    `db:"full_name"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ConnectionRequest struct {
	ID        string    `db:"id"`
	AthleteID string    `db:"athlete_id"`
	CoachID   string    `db:"coach_id"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type CoachAthleteRelation struct {
	ID        string    `db:"id"`
	CoachID   string    `db:"coach_id"`
	AthleteID string    `db:"athlete_id"`
	CreatedAt time.Time `db:"created_at"`
}

type TrainingGroup struct {
	ID        string    `db:"id"`
	CoachID   string    `db:"coach_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type GroupMember struct {
	GroupID   string    `db:"group_id"`
	AthleteID string   `db:"athlete_id"`
	Login     string    `db:"login"`
	FullName  string    `db:"full_name"`
	AddedAt   time.Time `db:"added_at"`
}

// ──────────────────────────────────────────────
// DTOs – JSON request bodies
// ──────────────────────────────────────────────

type ConnectionRequestCreate struct {
	CoachID string `json:"coach_id" validate:"required,uuid"`
}

type CreateGroupRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type UpdateGroupRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type AddGroupMemberRequest struct {
	AthleteID string `json:"athlete_id" validate:"required,uuid"`
}

// ──────────────────────────────────────────────
// DTOs – JSON response bodies
// ──────────────────────────────────────────────

type UserProfileResponse struct {
	ID        string    `json:"id"`
	Login     string    `json:"login"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type UserBrief struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
}

type ConnectionRequestResponse struct {
	ID        string     `json:"id"`
	Athlete   UserBrief  `json:"athlete"`
	Coach     UserBrief  `json:"coach"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type TrainingGroupSummary struct {
	ID           string    `json:"id"            db:"id"`
	Name         string    `json:"name"          db:"name"`
	MembersCount int       `json:"members_count" db:"members_count"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
}

type TrainingGroupDetail struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Members   []GroupMemberResponse `json:"members"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

type GroupMemberResponse struct {
	AthleteID string    `json:"athlete_id"`
	Login     string    `json:"login"`
	FullName  string    `json:"full_name"`
	AddedAt   time.Time `json:"added_at"`
}

type AthleteInfo struct {
	ID          string    `json:"id"           db:"id"`
	Login       string    `json:"login"        db:"login"`
	FullName    string    `json:"full_name"    db:"full_name"`
	ConnectedAt time.Time `json:"connected_at" db:"connected_at"`
}

type CoachInfo struct {
	ID          string    `json:"id"`
	Login       string    `json:"login"`
	FullName    string    `json:"full_name"`
	ConnectedAt time.Time `json:"connected_at"`
}

// ──────────────────────────────────────────────
// Pagination
// ──────────────────────────────────────────────

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

// ──────────────────────────────────────────────
// Error response
// ──────────────────────────────────────────────

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ──────────────────────────────────────────────
// Internal API response
// ──────────────────────────────────────────────

type InternalGroupMember struct {
	AthleteID string `json:"athlete_id" db:"athlete_id"`
	FullName  string `json:"full_name"  db:"full_name"`
	Login     string `json:"login"      db:"login"`
}
