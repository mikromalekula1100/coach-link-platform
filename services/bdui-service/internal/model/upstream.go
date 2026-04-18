package model

import "time"

// --- Данные от user-service ---

// UserProfile — профиль пользователя (GET /internal/users/{userId}).
type UserProfile struct {
	AthleteID string `json:"athlete_id"`
	FullName  string `json:"full_name"`
	Login     string `json:"login"`
}

// AthleteInfo — спортсмен тренера (GET /internal/connections/athletes).
type AthleteInfo struct {
	ID          string    `json:"id"`
	Login       string    `json:"login"`
	FullName    string    `json:"full_name"`
	ConnectedAt time.Time `json:"connected_at"`
}

// UserBrief — краткая информация о пользователе (поле в ConnectionRequest).
type UserBrief struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
}

// ConnectionRequest — входящая заявка (ответ GET /api/v1/connections/requests/incoming).
type ConnectionRequest struct {
	ID        string    `json:"id"`
	Athlete   UserBrief `json:"athlete"`
	Coach     UserBrief `json:"coach"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// CoachInfo — тренер спортсмена (GET /internal/connections/coach).
type CoachInfo struct {
	ID          string    `json:"id"`
	Login       string    `json:"login"`
	FullName    string    `json:"full_name"`
	ConnectedAt time.Time `json:"connected_at"`
}

// --- Данные от training-service ---

// ReportWithPlan — отчёт с данными плана (GET /internal/reports).
type ReportWithPlan struct {
	ID              string    `json:"id"`
	AssignmentID    string    `json:"assignment_id"`
	AthleteID       string    `json:"athlete_id"`
	Content         string    `json:"content"`
	DurationMinutes int       `json:"duration_minutes"`
	PerceivedEffort int       `json:"perceived_effort"`
	MaxHeartRate    *int      `json:"max_heart_rate,omitempty"`
	AvgHeartRate    *int      `json:"avg_heart_rate,omitempty"`
	DistanceKm      *float64  `json:"distance_km,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	Title           string    `json:"title"`
	ScheduledDate   time.Time `json:"scheduled_date"`
}

// AssignmentListItem — задание из списка.
type AssignmentListItem struct {
	ID              string  `json:"id"`
	PlanID          string  `json:"plan_id"`
	Title           string  `json:"title"`
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

// AssignmentDetail — детальная информация о задании.
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

// Pagination — метаданные пагинации.
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// PaginatedAthletes — ответ GET /api/v1/connections/athletes.
type PaginatedAthletes struct {
	Items      []AthleteInfo `json:"items"`
	Pagination Pagination    `json:"pagination"`
}

// PaginatedConnectionRequests — ответ GET /api/v1/connections/requests/incoming.
type PaginatedConnectionRequests struct {
	Items      []ConnectionRequest `json:"items"`
	Pagination Pagination          `json:"pagination"`
}
