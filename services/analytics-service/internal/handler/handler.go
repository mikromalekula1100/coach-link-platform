package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/analytics-service/internal/client"
	"github.com/coach-link/platform/services/analytics-service/internal/model"
	"github.com/coach-link/platform/services/analytics-service/internal/service"
)

type Handler struct {
	svc        *service.Service
	userClient *client.UserClient
}

func New(svc *service.Service, userClient *client.UserClient) *Handler {
	return &Handler{svc: svc, userClient: userClient}
}

// RegisterRoutes wires all routes onto the Echo instance.
func RegisterRoutes(e *echo.Echo, h *Handler) {
	api := e.Group("/api/v1/analytics")

	// Coach endpoints
	api.GET("/athletes/:athleteId/summary", h.GetAthleteSummary)
	api.GET("/athletes/:athleteId/progress", h.GetAthleteProgress)
	api.GET("/overview", h.GetCoachOverview)

	// Athlete endpoints (uses X-User-ID)
	api.GET("/me/summary", h.GetMySummary)
	api.GET("/me/progress", h.GetMyProgress)

	// Internal endpoints for AI service
	e.GET("/internal/analytics/athletes/:athleteId/summary", h.InternalGetAthleteSummary)
	e.GET("/internal/analytics/athletes/:athleteId/reports", h.InternalGetAthleteReports)
}

// ──────────────────────────────────────────────
// Coach endpoints
// ──────────────────────────────────────────────

// GetAthleteSummary returns an aggregated summary for a specific athlete.
func (h *Handler) GetAthleteSummary(c echo.Context) error {
	coachID, err := extractCoach(c)
	if err != nil {
		return err
	}

	athleteID := c.Param("athleteId")
	if athleteID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "athleteId is required"},
		})
	}

	if err := h.guardCoachAthlete(c, coachID, athleteID); err != nil {
		return err
	}

	summary, svcErr := h.svc.GetAthleteSummary(c.Request().Context(), athleteID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, summary)
}

// GetAthleteProgress returns time-bucketed progress data for a specific athlete.
func (h *Handler) GetAthleteProgress(c echo.Context) error {
	coachID, err := extractCoach(c)
	if err != nil {
		return err
	}

	athleteID := c.Param("athleteId")
	if athleteID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "athleteId is required"},
		})
	}

	if err := h.guardCoachAthlete(c, coachID, athleteID); err != nil {
		return err
	}

	period := c.QueryParam("period")
	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")

	progress, svcErr := h.svc.GetAthleteProgress(c.Request().Context(), athleteID, period, dateFrom, dateTo)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, progress)
}

// GetCoachOverview returns an overview of all athletes for the coach.
func (h *Handler) GetCoachOverview(c echo.Context) error {
	coachID, err := extractCoach(c)
	if err != nil {
		return err
	}

	overview, svcErr := h.svc.GetCoachOverview(c.Request().Context(), coachID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, overview)
}

// ──────────────────────────────────────────────
// Athlete endpoints
// ──────────────────────────────────────────────

// GetMySummary returns an aggregated summary for the authenticated athlete.
func (h *Handler) GetMySummary(c echo.Context) error {
	userID, err := extractUserID(c)
	if err != nil {
		return err
	}

	summary, svcErr := h.svc.GetAthleteSummary(c.Request().Context(), userID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, summary)
}

// GetMyProgress returns time-bucketed progress data for the authenticated athlete.
func (h *Handler) GetMyProgress(c echo.Context) error {
	userID, err := extractUserID(c)
	if err != nil {
		return err
	}

	period := c.QueryParam("period")
	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")

	progress, svcErr := h.svc.GetAthleteProgress(c.Request().Context(), userID, period, dateFrom, dateTo)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, progress)
}

// ──────────────────────────────────────────────
// Internal endpoints (no auth)
// ──────────────────────────────────────────────

// InternalGetAthleteSummary returns athlete summary without auth checks.
func (h *Handler) InternalGetAthleteSummary(c echo.Context) error {
	athleteID := c.Param("athleteId")
	if athleteID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "athleteId is required"},
		})
	}

	summary, err := h.svc.GetAthleteSummary(c.Request().Context(), athleteID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, summary)
}

// InternalGetAthleteReports returns raw reports for an athlete without auth checks.
func (h *Handler) InternalGetAthleteReports(c echo.Context) error {
	athleteID := c.Param("athleteId")
	if athleteID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "athleteId is required"},
		})
	}

	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")

	reports, err := h.svc.GetReports(c.Request().Context(), athleteID, dateFrom, dateTo)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, reports)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

// guardCoachAthlete verifies that athleteID is connected to coachID; returns a
// JSON error response and a non-nil error if the check fails or the athlete
// does not belong to this coach.
func (h *Handler) guardCoachAthlete(c echo.Context, coachID, athleteID string) error {
	ok, err := h.userClient.BelongsToCoach(c.Request().Context(), coachID, athleteID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "INTERNAL_ERROR", Message: "failed to verify athlete ownership"},
		})
	}
	if !ok {
		return c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "athlete does not belong to this coach"},
		})
	}
	return nil
}

func extractUserID(c echo.Context) (string, error) {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return "", c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "UNAUTHORIZED", Message: "Missing user identification headers"},
		})
	}
	return userID, nil
}

func extractCoach(c echo.Context) (string, error) {
	userID := c.Request().Header.Get("X-User-ID")
	role := c.Request().Header.Get("X-User-Role")
	if userID == "" {
		return "", c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "UNAUTHORIZED", Message: "Missing user identification headers"},
		})
	}
	if role != "coach" {
		return "", c.JSON(http.StatusForbidden, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "Only coaches can access this endpoint"},
		})
	}
	return userID, nil
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
