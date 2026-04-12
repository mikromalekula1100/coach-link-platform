package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/ai-service/internal/model"
	"github.com/coach-link/platform/services/ai-service/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires all routes onto the Echo instance.
func RegisterRoutes(e *echo.Echo, h *Handler) {
	api := e.Group("/api/v1/ai")
	api.POST("/athletes/:athleteId/recommendations", h.GetRecommendations)
	api.POST("/athletes/:athleteId/analysis", h.GetAnalysis)
	api.POST("/coach/summary", h.GetCoachSummary)
}

// ──────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────

// GetRecommendations generates AI training recommendations for an athlete.
func (h *Handler) GetRecommendations(c echo.Context) error {
	if _, err := extractCoach(c); err != nil {
		return err
	}

	athleteID := c.Param("athleteId")
	if athleteID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "athleteId is required"},
		})
	}

	var req model.AIRequest
	// Body is optional; ignore bind errors for empty body
	_ = c.Bind(&req)

	resp, err := h.svc.GenerateRecommendations(c.Request().Context(), athleteID, req.Context)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetAnalysis generates AI training analysis for an athlete.
func (h *Handler) GetAnalysis(c echo.Context) error {
	if _, err := extractCoach(c); err != nil {
		return err
	}

	athleteID := c.Param("athleteId")
	if athleteID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "athleteId is required"},
		})
	}

	var req model.AIRequest
	_ = c.Bind(&req)

	resp, err := h.svc.GenerateAnalysis(c.Request().Context(), athleteID, req.Context)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetCoachSummary generates a digest of all athletes' reports for the coach.
func (h *Handler) GetCoachSummary(c echo.Context) error {
	coachID, err := extractCoach(c)
	if err != nil {
		return err
	}

	var req model.SummaryRequest
	_ = c.Bind(&req)

	resp, svcErr := h.svc.GenerateSummary(c.Request().Context(), coachID, req)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

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
