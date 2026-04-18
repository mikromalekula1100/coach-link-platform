package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
	"github.com/coach-link/platform/services/bdui-service/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func RegisterRoutes(e *echo.Echo, h *Handler) {
	api := e.Group("/api/v1/bdui")

	api.GET("/screens/training-detail/:assignmentId", h.GetTrainingDetail)
	api.GET("/screens/:screenId", h.GetScreen)
}

// GetScreen — BDUI-схема для dashboard-экранов.
func (h *Handler) GetScreen(c echo.Context) error {
	screenID := c.Param("screenId")
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	if strings.HasPrefix(screenID, "training-detail") {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{
				Code:    "INVALID_SCREEN",
				Message: "Use /screens/training-detail/:assignmentId for training details",
			},
		})
	}

	schema, svcErr := h.svc.GetScreen(c.Request().Context(), screenID, userID, role)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, schema)
}

// GetTrainingDetail — BDUI-схема для описания тренировки.
func (h *Handler) GetTrainingDetail(c echo.Context) error {
	assignmentID := c.Param("assignmentId")
	userID, role, err := extractUser(c)
	if err != nil {
		return err
	}

	schema, svcErr := h.svc.GetTrainingDetail(c.Request().Context(), assignmentID, userID, role)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, schema)
}

func extractUser(c echo.Context) (string, string, error) {
	userID := c.Request().Header.Get("X-User-ID")
	role := c.Request().Header.Get("X-User-Role")
	if userID == "" {
		return "", "", c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "UNAUTHORIZED", Message: "Missing X-User-ID header"},
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
