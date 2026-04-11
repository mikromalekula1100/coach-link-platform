package model

import "time"

// ──────────────────────────────────────────────
// Database models (sqlx)
// ──────────────────────────────────────────────

type TrainingPlan struct {
	ID            string    `db:"id"`
	CoachID       string    `db:"coach_id"`
	Title         string    `db:"title"`
	Description   string    `db:"description"`
	ScheduledDate time.Time `db:"scheduled_date"`
	CreatedAt     time.Time `db:"created_at"`
}

type TrainingAssignment struct {
	ID               string     `db:"id"`
	PlanID           string     `db:"plan_id"`
	AthleteID        string     `db:"athlete_id"`
	CoachID          string     `db:"coach_id"`
	AthleteFullName  string     `db:"athlete_full_name"`
	AthleteLogin     string     `db:"athlete_login"`
	CoachFullName    string     `db:"coach_full_name"`
	CoachLogin       string     `db:"coach_login"`
	Status           string     `db:"status"`
	AssignedAt       time.Time  `db:"assigned_at"`
	CompletedAt      *time.Time `db:"completed_at"`
	ArchivedAt       *time.Time `db:"archived_at"`
}

type TrainingReport struct {
	ID              string    `db:"id"`
	AssignmentID    string    `db:"assignment_id"`
	AthleteID       string    `db:"athlete_id"`
	Content         string    `db:"content"`
	DurationMinutes int       `db:"duration_minutes"`
	PerceivedEffort int       `db:"perceived_effort"`
	MaxHeartRate    *int      `db:"max_heart_rate"`
	AvgHeartRate    *int      `db:"avg_heart_rate"`
	DistanceKm      *float64  `db:"distance_km"`
	CreatedAt       time.Time `db:"created_at"`
}

type TrainingTemplate struct {
	ID          string    `db:"id"`
	CoachID     string    `db:"coach_id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// AssignmentRow is a flat struct returned by repository JOIN queries.
type AssignmentRow struct {
	// Assignment fields
	ID              string     `db:"id"`
	PlanID          string     `db:"plan_id"`
	AthleteID       string     `db:"athlete_id"`
	CoachID         string     `db:"coach_id"`
	AthleteFullName string     `db:"athlete_full_name"`
	AthleteLogin    string     `db:"athlete_login"`
	CoachFullName   string     `db:"coach_full_name"`
	CoachLogin      string     `db:"coach_login"`
	Status          string     `db:"status"`
	AssignedAt      time.Time  `db:"assigned_at"`
	CompletedAt     *time.Time `db:"completed_at"`
	ArchivedAt      *time.Time `db:"archived_at"`
	// Joined from training_plans
	Title         string    `db:"title"`
	Description   string    `db:"description"`
	ScheduledDate time.Time `db:"scheduled_date"`
	// Computed
	HasReport bool `db:"has_report"`
}

// ──────────────────────────────────────────────
// DTOs – JSON request bodies
// ──────────────────────────────────────────────

type CreateTrainingPlanRequest struct {
	Title          string   `json:"title" validate:"required,min=1,max=255"`
	Description    string   `json:"description" validate:"required,min=1"`
	ScheduledDate  string   `json:"scheduled_date" validate:"required"`
	AthleteIDs     []string `json:"athlete_ids,omitempty"`
	GroupID        *string  `json:"group_id,omitempty"`
	TemplateID     *string  `json:"template_id,omitempty"`
	SaveAsTemplate bool     `json:"save_as_template,omitempty"`
}

type CreateReportRequest struct {
	Content         string   `json:"content" validate:"required,min=1"`
	DurationMinutes int      `json:"duration_minutes" validate:"required,min=1,max=1440"`
	PerceivedEffort int      `json:"perceived_effort" validate:"min=0,max=10"`
	MaxHeartRate    *int     `json:"max_heart_rate,omitempty"`
	AvgHeartRate    *int     `json:"avg_heart_rate,omitempty"`
	DistanceKm      *float64 `json:"distance_km,omitempty"`
}

type CreateTemplateRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"required,min=1"`
}

type UpdateTemplateRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

type AssignmentFilter struct {
	AthleteFullName string
	AthleteLogin    string
	DateFrom        string
	DateTo          string
	Status          string
	SortBy          string
	Page            int
	PageSize        int
}

// ──────────────────────────────────────────────
// DTOs – JSON response bodies
// ──────────────────────────────────────────────

type CreateTrainingPlanResponse struct {
	Plan        PlanResponse                `json:"plan"`
	Assignments []AssignmentBriefResponse   `json:"assignments"`
	Template    *TrainingTemplateResponse   `json:"template,omitempty"`
}

type PlanResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	ScheduledDate string `json:"scheduled_date"`
	CreatedAt     string `json:"created_at"`
}

type AssignmentBriefResponse struct {
	ID              string `json:"id"`
	AthleteID       string `json:"athlete_id"`
	AthleteFullName string `json:"athlete_full_name"`
	AthleteLogin    string `json:"athlete_login"`
}

type AssignmentListItem struct {
	ID              string     `json:"id"`
	PlanID          string     `json:"plan_id"`
	Title           string     `json:"title"`
	ScheduledDate   string     `json:"scheduled_date"`
	Status          string     `json:"status"`
	IsOverdue       bool       `json:"is_overdue"`
	HasReport       bool       `json:"has_report"`
	AssignedAt      string     `json:"assigned_at"`
	CompletedAt     *string    `json:"completed_at,omitempty"`
	AthleteID       string     `json:"athlete_id"`
	AthleteFullName string     `json:"athlete_full_name"`
	AthleteLogin    string     `json:"athlete_login"`
	CoachFullName   string     `json:"coach_full_name"`
	CoachLogin      string     `json:"coach_login"`
}

type AssignmentDetail struct {
	ID              string  `json:"id"`
	PlanID          string  `json:"plan_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	ScheduledDate   string  `json:"scheduled_date"`
	Status          string  `json:"status"`
	IsOverdue       bool    `json:"is_overdue"`
	HasReport       bool    `json:"has_report"`
	AssignedAt      string  `json:"assigned_at"`
	CompletedAt     *string `json:"completed_at,omitempty"`
	AthleteID       string  `json:"athlete_id"`
	AthleteFullName string  `json:"athlete_full_name"`
	AthleteLogin    string  `json:"athlete_login"`
	CoachFullName   string  `json:"coach_full_name"`
	CoachLogin      string  `json:"coach_login"`
}

type TrainingReportResponse struct {
	ID              string   `json:"id"`
	AssignmentID    string   `json:"assignment_id"`
	AthleteID       string   `json:"athlete_id"`
	AthleteFullName string   `json:"athlete_full_name"`
	AthleteLogin    string   `json:"athlete_login"`
	Content         string   `json:"content"`
	DurationMinutes int      `json:"duration_minutes"`
	PerceivedEffort int      `json:"perceived_effort"`
	MaxHeartRate    *int     `json:"max_heart_rate,omitempty"`
	AvgHeartRate    *int     `json:"avg_heart_rate,omitempty"`
	DistanceKm      *float64 `json:"distance_km,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

type TrainingTemplateResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
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
// Internal API response types
// ──────────────────────────────────────────────

type ReportWithPlan struct {
	ID              string    `json:"id" db:"id"`
	AssignmentID    string    `json:"assignment_id" db:"assignment_id"`
	AthleteID       string    `json:"athlete_id" db:"athlete_id"`
	Content         string    `json:"content" db:"content"`
	DurationMinutes int       `json:"duration_minutes" db:"duration_minutes"`
	PerceivedEffort int       `json:"perceived_effort" db:"perceived_effort"`
	MaxHeartRate    *int      `json:"max_heart_rate,omitempty" db:"max_heart_rate"`
	AvgHeartRate    *int      `json:"avg_heart_rate,omitempty" db:"avg_heart_rate"`
	DistanceKm      *float64  `json:"distance_km,omitempty" db:"distance_km"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	Title           string    `json:"title" db:"title"`
	ScheduledDate   time.Time `json:"scheduled_date" db:"scheduled_date"`
}

type AthleteStats struct {
	TotalReports       int     `json:"total_reports" db:"total_reports"`
	TotalDurationMin   int     `json:"total_duration_minutes" db:"total_duration_minutes"`
	AvgDurationMin     float64 `json:"avg_duration_minutes" db:"avg_duration_minutes"`
	AvgPerceivedEffort float64 `json:"avg_perceived_effort" db:"avg_perceived_effort"`
	AvgHeartRate       float64 `json:"avg_heart_rate" db:"avg_heart_rate"`
	MaxHeartRateEver   *int    `json:"max_heart_rate_ever" db:"max_heart_rate_ever"`
	TotalDistanceKm    float64 `json:"total_distance_km" db:"total_distance_km"`
	TotalAssignments   int     `json:"total_assignments" db:"total_assignments"`
	CompletedCount     int     `json:"completed_count" db:"completed_count"`
	CompletionRate     float64 `json:"completion_rate"`
}

type CoachOverviewStats struct {
	TotalAthletes    int `json:"total_athletes"`
	TotalAssignments int `json:"total_assignments" db:"total_assignments"`
	TotalReports     int `json:"total_reports" db:"total_reports"`
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
