package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/ai-service/internal/client"
	"github.com/coach-link/platform/services/ai-service/internal/model"
	"github.com/coach-link/platform/services/ai-service/internal/service"
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
	api := e.Group("/api/v1/ai")
	api.POST("/athletes/:athleteId/recommendations", h.GetRecommendations)
}

// GetRecommendations generates AI training recommendations for an athlete.
func (h *Handler) GetRecommendations(c echo.Context) error {
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

	var req model.AIRequest
	// Body is optional; ignore bind errors for empty body
	_ = c.Bind(&req)

	resp, err := h.svc.GenerateRecommendations(c.Request().Context(), athleteID, req.Context)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

// guardCoachAthlete verifies that athleteID is connected to coachID.
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
