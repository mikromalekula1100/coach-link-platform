package model

// ──────────────────────────────────────────────
// Types mirrored from analytics-service
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

type ReportWithPlan struct {
	ID              string   `json:"id"`
	Content         string   `json:"content"`
	DurationMinutes int      `json:"duration_minutes"`
	PerceivedEffort int      `json:"perceived_effort"`
	MaxHeartRate    *int     `json:"max_heart_rate,omitempty"`
	AvgHeartRate    *int     `json:"avg_heart_rate,omitempty"`
	DistanceKm      *float64 `json:"distance_km,omitempty"`
	Title           string   `json:"title"`
	ScheduledDate   string   `json:"scheduled_date"`
}

// ──────────────────────────────────────────────
// AI request / response
// ──────────────────────────────────────────────

type AIRequest struct {
	Context string `json:"context,omitempty"`
}

type AIResponse struct {
	AthleteID   string `json:"athlete_id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	GeneratedAt string `json:"generated_at"`
	Model       string `json:"model"`
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
