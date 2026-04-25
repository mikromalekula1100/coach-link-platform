package handler

import (
	"math"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/user-service/internal/model"
	"github.com/coach-link/platform/services/user-service/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires all routes onto the Echo instance.
func RegisterRoutes(e *echo.Echo, h *Handler) {
	api := e.Group("/api/v1")

	// Users
	api.GET("/users/me", h.GetMe)
	api.GET("/users/search", h.SearchUsers)

	// Connections
	api.POST("/connections/request", h.SendConnectionRequest)
	api.GET("/connections/requests/incoming", h.GetIncomingRequests)
	api.GET("/connections/requests/outgoing", h.GetOutgoingRequest)
	api.PUT("/connections/requests/:requestId/accept", h.AcceptConnectionRequest)
	api.PUT("/connections/requests/:requestId/reject", h.RejectConnectionRequest)
	api.GET("/connections/athletes", h.GetAthletes)
	api.GET("/connections/coach", h.GetCoach)
	api.DELETE("/connections/athletes/:athleteId", h.RemoveAthlete)

	// Groups
	api.POST("/groups", h.CreateGroup)
	api.GET("/groups", h.GetGroups)
	api.GET("/groups/:groupId", h.GetGroup)
	api.PUT("/groups/:groupId", h.UpdateGroup)
	api.DELETE("/groups/:groupId", h.DeleteGroup)
	api.POST("/groups/:groupId/members", h.AddGroupMember)
	api.DELETE("/groups/:groupId/members/:athleteId", h.RemoveGroupMember)

	// Internal API (no auth)
	internal := e.Group("/internal")
	internal.GET("/groups/:groupId/members", h.InternalGetGroupMembers)
	internal.GET("/users/:userId", h.InternalGetUser)
	internal.GET("/coach/:coachId/has-athlete/:athleteId", h.InternalCheckCoachAthlete)
}

// ──────────────────────────────────────────────
// Users
// ──────────────────────────────────────────────

func (h *Handler) GetMe(c echo.Context) error {
	userID, _, err := extractUser(c)
	if err != nil {
		return err
	}

	profile, svcErr := h.svc.GetProfile(c.Request().Context(), userID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, toUserProfileResponse(profile))
}

func (h *Handler) SearchUsers(c echo.Context) error {
	_, _, err := extractUser(c)
	if err != nil {
		return err
	}

	query := c.QueryParam("q")
	role := c.QueryParam("role")
	page, pageSize := parsePagination(c, 1, 20)

	profiles, total, svcErr := h.svc.SearchUsers(c.Request().Context(), query, role, page, pageSize)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	items := make([]model.UserProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, toUserProfileResponse(&p))
	}

	return c.JSON(http.StatusOK, model.PaginatedResponse{
		Items:      items,
		Pagination: buildPagination(page, pageSize, total),
	})
}

// ──────────────────────────────────────────────
// Connection Requests
// ──────────────────────────────────────────────

func (h *Handler) SendConnectionRequest(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	var req model.ConnectionRequestCreate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}
	if req.CoachID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "coach_id is required"},
		})
	}

	cr, svcErr := h.svc.SendConnectionRequest(c.Request().Context(), userID, req.CoachID, role)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	resp, svcErr := h.svc.EnrichConnectionRequest(c.Request().Context(), cr)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) GetIncomingRequests(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	status := c.QueryParam("status")
	if status == "" {
		status = "pending"
	}
	page, pageSize := parsePagination(c, 1, 20)

	reqs, total, svcErr := h.svc.GetIncomingRequests(c.Request().Context(), userID, role, status, page, pageSize)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	items := make([]model.ConnectionRequestResponse, 0, len(reqs))
	for _, cr := range reqs {
		enriched, enrichErr := h.svc.EnrichConnectionRequest(c.Request().Context(), &cr)
		if enrichErr != nil {
			return handleError(c, enrichErr)
		}
		items = append(items, *enriched)
	}

	return c.JSON(http.StatusOK, model.PaginatedResponse{
		Items:      items,
		Pagination: buildPagination(page, pageSize, total),
	})
}

func (h *Handler) GetOutgoingRequest(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	cr, svcErr := h.svc.GetOutgoingRequest(c.Request().Context(), userID, role)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	resp, svcErr := h.svc.EnrichConnectionRequest(c.Request().Context(), cr)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) AcceptConnectionRequest(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	requestID := c.Param("requestId")
	cr, svcErr := h.svc.AcceptConnectionRequest(c.Request().Context(), userID, role, requestID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	resp, svcErr := h.svc.EnrichConnectionRequest(c.Request().Context(), cr)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) RejectConnectionRequest(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	requestID := c.Param("requestId")
	cr, svcErr := h.svc.RejectConnectionRequest(c.Request().Context(), userID, role, requestID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	resp, svcErr := h.svc.EnrichConnectionRequest(c.Request().Context(), cr)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────
// Athletes / Coach
// ──────────────────────────────────────────────

func (h *Handler) GetAthletes(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	query := c.QueryParam("q")
	page, pageSize := parsePagination(c, 1, 50)

	athletes, total, svcErr := h.svc.GetAthletes(c.Request().Context(), userID, role, query, page, pageSize)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, model.PaginatedResponse{
		Items:      athletes,
		Pagination: buildPagination(page, pageSize, total),
	})
}

func (h *Handler) GetCoach(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	coach, svcErr := h.svc.GetCoach(c.Request().Context(), userID, role)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, coach)
}

func (h *Handler) RemoveAthlete(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	athleteID := c.Param("athleteId")
	if svcErr := h.svc.RemoveAthlete(c.Request().Context(), userID, role, athleteID); svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.NoContent(http.StatusNoContent)
}

// ──────────────────────────────────────────────
// Groups
// ──────────────────────────────────────────────

func (h *Handler) CreateGroup(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	var req model.CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "name is required"},
		})
	}
	if len(req.Name) > 255 {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "name must be at most 255 characters"},
		})
	}

	group, svcErr := h.svc.CreateGroup(c.Request().Context(), userID, role, req.Name)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusCreated, toTrainingGroupResponse(group))
}

func (h *Handler) GetGroups(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	page, pageSize := parsePagination(c, 1, 20)

	groups, total, svcErr := h.svc.GetGroups(c.Request().Context(), userID, role, page, pageSize)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, model.PaginatedResponse{
		Items:      groups,
		Pagination: buildPagination(page, pageSize, total),
	})
}

func (h *Handler) GetGroup(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	groupID := c.Param("groupId")
	query := c.QueryParam("q")

	detail, svcErr := h.svc.GetGroup(c.Request().Context(), userID, role, groupID, query)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, detail)
}

func (h *Handler) UpdateGroup(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	groupID := c.Param("groupId")

	var req model.UpdateGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "name is required"},
		})
	}
	if len(req.Name) > 255 {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "name must be at most 255 characters"},
		})
	}

	group, svcErr := h.svc.UpdateGroup(c.Request().Context(), userID, role, groupID, req.Name)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, toTrainingGroupResponse(group))
}

func (h *Handler) DeleteGroup(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	groupID := c.Param("groupId")
	if svcErr := h.svc.DeleteGroup(c.Request().Context(), userID, role, groupID); svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) AddGroupMember(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	groupID := c.Param("groupId")

	var req model.AddGroupMemberRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}
	if req.AthleteID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "athlete_id is required"},
		})
	}

	member, svcErr := h.svc.AddGroupMember(c.Request().Context(), userID, role, groupID, req.AthleteID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusCreated, member)
}

func (h *Handler) RemoveGroupMember(c echo.Context) error {
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	groupID := c.Param("groupId")
	athleteID := c.Param("athleteId")

	if svcErr := h.svc.RemoveGroupMember(c.Request().Context(), userID, role, groupID, athleteID); svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.NoContent(http.StatusNoContent)
}

// ──────────────────────────────────────────────
// Internal API
// ──────────────────────────────────────────────

func (h *Handler) InternalGetUser(c echo.Context) error {
	userID := c.Param("userId")

	profile, svcErr := h.svc.GetProfile(c.Request().Context(), userID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, model.InternalGroupMember{
		AthleteID: profile.ID,
		FullName:  profile.FullName,
		Login:     profile.Login,
	})
}

func (h *Handler) InternalGetGroupMembers(c echo.Context) error {
	groupID := c.Param("groupId")

	members, svcErr := h.svc.GetGroupMemberIDs(c.Request().Context(), groupID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, members)
}

// InternalCheckCoachAthlete returns 200 if the athlete is connected to the coach, 404 otherwise.
func (h *Handler) InternalCheckCoachAthlete(c echo.Context) error {
	coachID := c.Param("coachId")
	athleteID := c.Param("athleteId")

	has, err := h.svc.HasCoachAthleteRelation(c.Request().Context(), coachID, athleteID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "INTERNAL_ERROR", Message: "failed to check relation"},
		})
	}
	if !has {
		return c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "NOT_FOUND", Message: "athlete is not connected to this coach"},
		})
	}
	return c.NoContent(http.StatusOK)
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

func toUserProfileResponse(p *model.UserProfile) model.UserProfileResponse {
	return model.UserProfileResponse{
		ID:        p.ID,
		Login:     p.Login,
		Email:     p.Email,
		FullName:  p.FullName,
		Role:      p.Role,
		CreatedAt: p.CreatedAt,
	}
}

func toTrainingGroupResponse(g *model.TrainingGroup) map[string]interface{} {
	return map[string]interface{}{
		"id":         g.ID,
		"name":       g.Name,
		"created_at": g.CreatedAt,
		"updated_at": g.UpdatedAt,
	}
}
