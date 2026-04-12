package handler

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/training-service/internal/model"
	"github.com/coach-link/platform/services/training-service/internal/service"
)

// ──────────────────────────────────────────────
// Echo Validator
// ──────────────────────────────────────────────

type CustomValidator struct {
	validator *validator.Validate
}

func NewValidator() *CustomValidator {
	return &CustomValidator{validator: validator.New()}
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// ──────────────────────────────────────────────
// Handler
// ──────────────────────────────────────────────

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires all routes onto the Echo instance.
func RegisterRoutes(e *echo.Echo, h *Handler) {
	api := e.Group("/api/v1/training")

	// Plans
	api.POST("/plans", h.CreatePlan)
	api.GET("/groups/:groupId/plans", h.GetGroupPlans)

	// Assignments
	api.GET("/assignments", h.GetAssignments)
	api.GET("/assignments/archived", h.GetArchivedAssignments)
	api.GET("/assignments/:assignmentId", h.GetAssignment)
	api.DELETE("/assignments/:assignmentId", h.DeleteAssignment)
	api.PUT("/assignments/:assignmentId/archive", h.ArchiveAssignment)

	// Reports
	api.POST("/assignments/:assignmentId/report", h.SubmitReport)
	api.GET("/assignments/:assignmentId/report", h.GetReport)

	// Templates
	api.POST("/templates", h.CreateTemplate)
	api.GET("/templates", h.GetTemplates)
	api.GET("/templates/:templateId", h.GetTemplate)
	api.PUT("/templates/:templateId", h.UpdateTemplate)
	api.DELETE("/templates/:templateId", h.DeleteTemplate)

	// Internal API (no auth required, called by other services)
	e.GET("/internal/reports", h.InternalGetReports)
	e.GET("/internal/athletes/:athleteId/stats", h.InternalGetAthleteStats)
	e.GET("/internal/coach/:coachId/athletes", h.InternalGetCoachAthleteIDs)
	e.GET("/internal/coach/:coachId/overview", h.InternalGetCoachOverview)
}

// ──────────────────────────────────────────────
// Plans
// ──────────────────────────────────────────────

func (h *Handler) CreatePlan(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can create training plans"},
		})
	}

	var req model.CreateTrainingPlanRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: formatValidationError(err)},
		})
	}

	// Extract coach name from headers (set by gateway or upstream)
	coachFullName := c.Request().Header.Get("X-User-FullName")
	coachLogin := c.Request().Header.Get("X-User-Login")

	resp, svcErr := h.svc.CreatePlan(c.Request().Context(), userID, coachFullName, coachLogin, req)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) GetGroupPlans(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can view group plans"},
		})
	}

	groupID := c.Param("groupId")
	if groupID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "groupId is required"},
		})
	}

	activeOnly := true
	if c.QueryParam("active") == "false" {
		activeOnly = false
	}

	page, pageSize := parsePagination(c, 1, 20)

	rows, total, svcErr := h.svc.GetGroupPlans(c.Request().Context(), userID, groupID, activeOnly, page, pageSize)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	items := make([]model.GroupPlanListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, model.GroupPlanListItem{
			ID:              r.ID,
			Title:           r.Title,
			Description:     r.Description,
			ScheduledDate:   r.ScheduledDate.Format("2006-01-02"),
			CreatedAt:       r.CreatedAt.Format(time.RFC3339),
			GroupID:         r.GroupID,
			AssignmentCount: r.AssignmentCount,
		})
	}

	return c.JSON(http.StatusOK, model.PaginatedResponse{
		Items:      items,
		Pagination: buildPagination(page, pageSize, total),
	})
}

// ──────────────────────────────────────────────
// Assignments
// ──────────────────────────────────────────────

func (h *Handler) GetAssignments(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	page, pageSize := parsePagination(c, 1, 20)
	filter := model.AssignmentFilter{
		AthleteFullName: c.QueryParam("athlete_full_name"),
		AthleteLogin:    c.QueryParam("athlete_login"),
		DateFrom:        c.QueryParam("date_from"),
		DateTo:          c.QueryParam("date_to"),
		Status:          c.QueryParam("status"),
		SortBy:          c.QueryParam("sort_by"),
		Page:            page,
		PageSize:        pageSize,
	}

	rows, total, svcErr := h.svc.GetAssignments(c.Request().Context(), userID, role, filter)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	items := make([]model.AssignmentListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, toAssignmentListItem(r))
	}

	return c.JSON(http.StatusOK, model.PaginatedResponse{
		Items:      items,
		Pagination: buildPagination(page, pageSize, total),
	})
}

func (h *Handler) GetArchivedAssignments(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can view archived assignments"},
		})
	}

	page, pageSize := parsePagination(c, 1, 20)
	filter := model.AssignmentFilter{
		AthleteFullName: c.QueryParam("athlete_full_name"),
		AthleteLogin:    c.QueryParam("athlete_login"),
		DateFrom:        c.QueryParam("date_from"),
		DateTo:          c.QueryParam("date_to"),
		SortBy:          c.QueryParam("sort_by"),
		Page:            page,
		PageSize:        pageSize,
	}

	rows, total, svcErr := h.svc.GetArchivedAssignments(c.Request().Context(), userID, filter)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	items := make([]model.AssignmentListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, toAssignmentListItem(r))
	}

	return c.JSON(http.StatusOK, model.PaginatedResponse{
		Items:      items,
		Pagination: buildPagination(page, pageSize, total),
	})
}

func (h *Handler) GetAssignment(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	assignmentID := c.Param("assignmentId")
	row, svcErr := h.svc.GetAssignment(c.Request().Context(), userID, role, assignmentID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, toAssignmentDetail(*row))
}

func (h *Handler) DeleteAssignment(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can delete assignments"},
		})
	}

	assignmentID := c.Param("assignmentId")
	if svcErr := h.svc.DeleteAssignment(c.Request().Context(), userID, assignmentID); svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ArchiveAssignment(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can archive assignments"},
		})
	}

	assignmentID := c.Param("assignmentId")
	if svcErr := h.svc.ArchiveAssignment(c.Request().Context(), userID, assignmentID); svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "archived"})
}

// ──────────────────────────────────────────────
// Reports
// ──────────────────────────────────────────────

func (h *Handler) SubmitReport(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "athlete" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only athletes can submit reports"},
		})
	}

	assignmentID := c.Param("assignmentId")

	var req model.CreateReportRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: formatValidationError(err)},
		})
	}

	report, svcErr := h.svc.SubmitReport(c.Request().Context(), userID, assignmentID, req)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	// To build the response, we need athlete name info from the assignment
	row, _ := h.svc.GetAssignment(c.Request().Context(), userID, role, assignmentID)
	athleteFullName := ""
	athleteLogin := ""
	if row != nil {
		athleteFullName = row.AthleteFullName
		athleteLogin = row.AthleteLogin
	}

	return c.JSON(http.StatusCreated, model.TrainingReportResponse{
		ID:              report.ID,
		AssignmentID:    report.AssignmentID,
		AthleteID:       report.AthleteID,
		AthleteFullName: athleteFullName,
		AthleteLogin:    athleteLogin,
		Content:         report.Content,
		DurationMinutes: report.DurationMinutes,
		PerceivedEffort: report.PerceivedEffort,
		MaxHeartRate:    report.MaxHeartRate,
		AvgHeartRate:    report.AvgHeartRate,
		DistanceKm:      report.DistanceKm,
		CreatedAt:       report.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) GetReport(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	assignmentID := c.Param("assignmentId")
	report, assignment, svcErr := h.svc.GetReport(c.Request().Context(), userID, role, assignmentID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, model.TrainingReportResponse{
		ID:              report.ID,
		AssignmentID:    report.AssignmentID,
		AthleteID:       report.AthleteID,
		AthleteFullName: assignment.AthleteFullName,
		AthleteLogin:    assignment.AthleteLogin,
		Content:         report.Content,
		DurationMinutes: report.DurationMinutes,
		PerceivedEffort: report.PerceivedEffort,
		MaxHeartRate:    report.MaxHeartRate,
		AvgHeartRate:    report.AvgHeartRate,
		DistanceKm:      report.DistanceKm,
		CreatedAt:       report.CreatedAt.Format(time.RFC3339),
	})
}

// ──────────────────────────────────────────────
// Templates
// ──────────────────────────────────────────────

func (h *Handler) CreateTemplate(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can create templates"},
		})
	}

	var req model.CreateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: formatValidationError(err)},
		})
	}

	tmpl, svcErr := h.svc.CreateTemplate(c.Request().Context(), userID, req)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusCreated, toTemplateResponse(tmpl))
}

func (h *Handler) GetTemplates(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can view templates"},
		})
	}

	query := c.QueryParam("q")
	page, pageSize := parsePagination(c, 1, 20)

	templates, total, svcErr := h.svc.GetTemplates(c.Request().Context(), userID, query, page, pageSize)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	items := make([]model.TrainingTemplateResponse, 0, len(templates))
	for _, t := range templates {
		items = append(items, toTemplateResponse(&t))
	}

	return c.JSON(http.StatusOK, model.PaginatedResponse{
		Items:      items,
		Pagination: buildPagination(page, pageSize, total),
	})
}

func (h *Handler) GetTemplate(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can view templates"},
		})
	}

	templateID := c.Param("templateId")
	tmpl, svcErr := h.svc.GetTemplate(c.Request().Context(), userID, templateID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, toTemplateResponse(tmpl))
}

func (h *Handler) UpdateTemplate(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can update templates"},
		})
	}

	templateID := c.Param("templateId")

	var req model.UpdateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}

	tmpl, svcErr := h.svc.UpdateTemplate(c.Request().Context(), userID, templateID, req)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, toTemplateResponse(tmpl))
}

func (h *Handler) DeleteTemplate(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}
	if role != "coach" {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can delete templates"},
		})
	}

	templateID := c.Param("templateId")
	if svcErr := h.svc.DeleteTemplate(c.Request().Context(), userID, templateID); svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.NoContent(http.StatusNoContent)
}

// ──────────────────────────────────────────────
// Internal API
// ──────────────────────────────────────────────

func (h *Handler) InternalGetReports(c echo.Context) error {
	athleteID := c.QueryParam("athlete_id")
	if athleteID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "athlete_id query parameter is required"},
		})
	}

	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")

	reports, err := h.svc.GetReportsByAthleteID(c.Request().Context(), athleteID, dateFrom, dateTo)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, reports)
}

func (h *Handler) InternalGetAthleteStats(c echo.Context) error {
	athleteID := c.Param("athleteId")

	stats, err := h.svc.GetAthleteStats(c.Request().Context(), athleteID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, stats)
}

func (h *Handler) InternalGetCoachAthleteIDs(c echo.Context) error {
	coachID := c.Param("coachId")

	ids, err := h.svc.GetCoachAthleteIDs(c.Request().Context(), coachID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, ids)
}

func (h *Handler) InternalGetCoachOverview(c echo.Context) error {
	coachID := c.Param("coachId")

	stats, err := h.svc.GetCoachOverviewStats(c.Request().Context(), coachID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, stats)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func extractUser(c echo.Context) (string, string, error) {
	userID := c.Request().Header.Get("X-User-ID")
	role := c.Request().Header.Get("X-User-Role")
	if userID == "" || role == "" {
		return "", "", c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "UNAUTHORIZED", Message: "Missing user identification headers"},
		})
	}
	return userID, role, nil
}

func handleError(c echo.Context, err error) error {
	if se, ok := service.IsServiceError(err); ok {
		return c.JSON(se.Status, model.ErrorResponse{
			Error: model.ErrorDetail{Code: se.Code, Message: se.Message},
		})
	}
	return c.JSON(http.StatusInternalServerError, model.ErrorResponse{
		Error: model.ErrorDetail{Code: "INTERNAL_ERROR", Message: "Internal server error"},
	})
}

func parsePagination(c echo.Context, defaultPage, defaultPageSize int) (int, int) {
	page := defaultPage
	pageSize := defaultPageSize

	if v := c.QueryParam("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p >= 1 {
			page = p
		}
	}
	if v := c.QueryParam("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps >= 1 && ps <= 100 {
			pageSize = ps
		}
	}
	return page, pageSize
}

func buildPagination(page, pageSize, total int) model.Pagination {
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return model.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}
}

func toAssignmentListItem(r model.AssignmentRow) model.AssignmentListItem {
	item := model.AssignmentListItem{
		ID:              r.ID,
		PlanID:          r.PlanID,
		Title:           r.Title,
		ScheduledDate:   r.ScheduledDate.Format("2006-01-02"),
		Status:          r.Status,
		IsOverdue:       service.ComputeIsOverdue(r.Status, r.ScheduledDate),
		HasReport:       r.HasReport,
		AssignedAt:      r.AssignedAt.Format(time.RFC3339),
		AthleteID:       r.AthleteID,
		AthleteFullName: r.AthleteFullName,
		AthleteLogin:    r.AthleteLogin,
		CoachFullName:   r.CoachFullName,
		CoachLogin:      r.CoachLogin,
	}
	if r.CompletedAt != nil {
		t := r.CompletedAt.Format(time.RFC3339)
		item.CompletedAt = &t
	}
	return item
}

func toAssignmentDetail(r model.AssignmentRow) model.AssignmentDetail {
	detail := model.AssignmentDetail{
		ID:              r.ID,
		PlanID:          r.PlanID,
		Title:           r.Title,
		Description:     r.Description,
		ScheduledDate:   r.ScheduledDate.Format("2006-01-02"),
		Status:          r.Status,
		IsOverdue:       service.ComputeIsOverdue(r.Status, r.ScheduledDate),
		HasReport:       r.HasReport,
		AssignedAt:      r.AssignedAt.Format(time.RFC3339),
		AthleteID:       r.AthleteID,
		AthleteFullName: r.AthleteFullName,
		AthleteLogin:    r.AthleteLogin,
		CoachFullName:   r.CoachFullName,
		CoachLogin:      r.CoachLogin,
	}
	if r.CompletedAt != nil {
		t := r.CompletedAt.Format(time.RFC3339)
		detail.CompletedAt = &t
	}
	return detail
}

func toTemplateResponse(t *model.TrainingTemplate) model.TrainingTemplateResponse {
	return model.TrainingTemplateResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
}

func formatValidationError(err error) string {
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			switch fe.Tag() {
			case "required":
				return fe.Field() + " is required"
			case "min":
				return fe.Field() + " must be at least " + fe.Param()
			case "max":
				return fe.Field() + " must be at most " + fe.Param()
			default:
				return fe.Field() + " is invalid"
			}
		}
	}
	return "Validation failed"
}
