package model

// ──────────────────────────────────────────────
// Types mirrored from training-service internal API
// ──────────────────────────────────────────────

type ReportWithPlan struct {
	ID              string   `json:"id"`
	AssignmentID    string   `json:"assignment_id"`
	AthleteID       string   `json:"athlete_id"`
	Content         string   `json:"content"`
	DurationMinutes int      `json:"duration_minutes"`
	PerceivedEffort int      `json:"perceived_effort"`
	MaxHeartRate    *int     `json:"max_heart_rate,omitempty"`
	AvgHeartRate    *int     `json:"avg_heart_rate,omitempty"`
	DistanceKm      *float64 `json:"distance_km,omitempty"`
	CreatedAt       string   `json:"created_at"`
	Title           string   `json:"title"`
	ScheduledDate   string   `json:"scheduled_date"`
}

type AthleteStats struct {
	TotalReports       int     `json:"total_reports"`
	TotalDurationMin   int     `json:"total_duration_minutes"`
	AvgDurationMin     float64 `json:"avg_duration_minutes"`
	AvgPerceivedEffort float64 `json:"avg_perceived_effort"`
	AvgHeartRate       float64 `json:"avg_heart_rate"`
	MaxHeartRateEver   *int    `json:"max_heart_rate_ever"`
	TotalDistanceKm    float64 `json:"total_distance_km"`
	TotalAssignments   int     `json:"total_assignments"`
	CompletedCount     int     `json:"completed_count"`
	CompletionRate     float64 `json:"completion_rate"`
}

type CoachOverviewStats struct {
	TotalAthletes    int `json:"total_athletes"`
	TotalAssignments int `json:"total_assignments"`
	TotalReports     int `json:"total_reports"`
}

// ──────────────────────────────────────────────
// Analytics response types
// ──────────────────────────────────────────────

type AthleteSummary struct {
	AthleteID          string  `json:"athlete_id"`
	TotalReports       int     `json:"total_reports"`
	TotalDurationMin   int     `json:"total_duration_minutes"`
	TotalDistanceKm    float64 `json:"total_distance_km"`
	AvgDurationMin     float64 `json:"avg_duration_minutes"`
	AvgPerceivedEffort float64 `json:"avg_perceived_effort"`
	AvgHeartRate       float64 `json:"avg_heart_rate"`
	MaxHeartRateEver   *int    `json:"max_heart_rate_ever"`
	CompletionRate     float64 `json:"completion_rate"`
	TotalAssignments   int     `json:"total_assignments"`
}

type ProgressPoint struct {
	PeriodStart        string  `json:"period_start"`
	PeriodEnd          string  `json:"period_end"`
	ReportCount        int     `json:"report_count"`
	TotalDurationMin   int     `json:"total_duration_minutes"`
	AvgPerceivedEffort float64 `json:"avg_perceived_effort"`
	AvgHeartRate       float64 `json:"avg_heart_rate"`
	TotalDistanceKm    float64 `json:"total_distance_km"`
}

type ProgressResponse struct {
	AthleteID string          `json:"athlete_id"`
	Period    string          `json:"period"`
	Points    []ProgressPoint `json:"points"`
}

type CoachOverview struct {
	TotalAthletes    int               `json:"total_athletes"`
	TotalAssignments int               `json:"total_assignments"`
	TotalReports     int               `json:"total_reports"`
	CompletionRate   float64           `json:"completion_rate"`
	Athletes         []AthleteOverview `json:"athletes"`
}

type AthleteOverview struct {
	AthleteID      string  `json:"athlete_id"`
	TotalReports   int     `json:"total_reports"`
	AvgEffort      float64 `json:"avg_effort"`
	CompletionRate float64 `json:"completion_rate"`
}

// ──────────────────────────────────────────────
// Error types
// ──────────────────────────────────────────────

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}
