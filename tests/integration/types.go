package integration

// AuthResponse mirrors auth-service AuthResponse.
type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int         `json:"expires_in"`
	User         UserProfile `json:"user"`
}

// UserProfile from auth-service response.
type UserProfile struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// ServiceErrorResponse is the nested error format from backend services.
type ServiceErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error code and message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GatewayErrorResponse is the flat error format from the API gateway.
type GatewayErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Pagination used across paginated responses.
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// UserProfileResponse from user-service /users/me.
type UserProfileResponse struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// UserBrief is nested in ConnectionRequestResponse.
type UserBrief struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
}

// ConnectionRequestResponse from user-service.
type ConnectionRequestResponse struct {
	ID        string    `json:"id"`
	Athlete   UserBrief `json:"athlete"`
	Coach     UserBrief `json:"coach"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt *string   `json:"updated_at,omitempty"`
}

// AthleteInfo from GET /connections/athletes.
type AthleteInfo struct {
	ID          string `json:"id"`
	Login       string `json:"login"`
	FullName    string `json:"full_name"`
	ConnectedAt string `json:"connected_at"`
}

// CoachInfo from GET /connections/coach.
type CoachInfo struct {
	ID          string `json:"id"`
	Login       string `json:"login"`
	FullName    string `json:"full_name"`
	ConnectedAt string `json:"connected_at"`
}

// TrainingGroupSummary from GET /groups.
type TrainingGroupSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MembersCount int    `json:"members_count"`
	CreatedAt    string `json:"created_at"`
}

// GroupMemberResponse from group detail.
type GroupMemberResponse struct {
	AthleteID string `json:"athlete_id"`
	Login     string `json:"login"`
	FullName  string `json:"full_name"`
	AddedAt   string `json:"added_at"`
}

// TrainingGroupDetail from GET /groups/:id.
type TrainingGroupDetail struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Members   []GroupMemberResponse `json:"members"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
}

// CreateTrainingPlanResponse from POST /training/plans.
type CreateTrainingPlanResponse struct {
	Plan        PlanResponse              `json:"plan"`
	Assignments []AssignmentBriefResponse `json:"assignments"`
	Template    *TemplateResponse         `json:"template,omitempty"`
}

// PlanResponse within CreateTrainingPlanResponse.
type PlanResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	ScheduledDate string `json:"scheduled_date"`
	CreatedAt     string `json:"created_at"`
}

// AssignmentBriefResponse within CreateTrainingPlanResponse.
type AssignmentBriefResponse struct {
	ID              string `json:"id"`
	AthleteID       string `json:"athlete_id"`
	AthleteFullName string `json:"athlete_full_name"`
	AthleteLogin    string `json:"athlete_login"`
}

// AssignmentListItem from GET /training/assignments.
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

// AssignmentDetail from GET /training/assignments/:id.
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

// TrainingReportResponse from report endpoints.
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

// TemplateResponse from template endpoints.
type TemplateResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// NotificationResponse single notification.
type NotificationResponse struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Body      *string                `json:"body,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	IsRead    bool                   `json:"is_read"`
	CreatedAt string                 `json:"created_at"`
}

// NotificationsListResponse from GET /notifications.
type NotificationsListResponse struct {
	Items       []NotificationResponse `json:"items"`
	Pagination  Pagination             `json:"pagination"`
	UnreadCount int                    `json:"unread_count"`
}

// PaginatedSearchResponse for user search results.
type PaginatedSearchResponse struct {
	Items      []UserProfileResponse `json:"items"`
	Pagination Pagination            `json:"pagination"`
}

// PaginatedConnectionRequests for incoming/outgoing requests.
type PaginatedConnectionRequests struct {
	Items      []ConnectionRequestResponse `json:"items"`
	Pagination Pagination                  `json:"pagination"`
}

// PaginatedAthletes for GET /connections/athletes.
type PaginatedAthletes struct {
	Items      []AthleteInfo `json:"items"`
	Pagination Pagination    `json:"pagination"`
}

// PaginatedGroups for GET /groups.
type PaginatedGroups struct {
	Items      []TrainingGroupSummary `json:"items"`
	Pagination Pagination             `json:"pagination"`
}

// PaginatedAssignments for GET /training/assignments.
type PaginatedAssignments struct {
	Items      []AssignmentListItem `json:"items"`
	Pagination Pagination           `json:"pagination"`
}

// PaginatedTemplates for GET /training/templates.
type PaginatedTemplates struct {
	Items      []TemplateResponse `json:"items"`
	Pagination Pagination         `json:"pagination"`
}
