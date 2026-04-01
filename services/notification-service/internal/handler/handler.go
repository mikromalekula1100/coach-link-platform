package handler

import (
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/notification-service/internal/model"
	"github.com/coach-link/platform/services/notification-service/internal/service"
)

var validate = validator.New()

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires all routes onto the Echo instance.
func RegisterRoutes(e *echo.Echo, h *Handler) {
	api := e.Group("/api/v1")

	api.GET("/notifications", h.GetNotifications)
	api.PUT("/notifications/:notificationId/read", h.MarkRead)
	api.PUT("/notifications/read-all", h.MarkAllRead)
	api.POST("/notifications/device-token", h.RegisterDeviceToken)
}

// ──────────────────────────────────────────────
// GET /api/v1/notifications
// ──────────────────────────────────────────────

func (h *Handler) GetNotifications(c echo.Context) error {
	userID, err := extractUserID(c)
	if err != nil {
		return err
	}

	page, pageSize := parsePagination(c, 1, 20)

	var isRead *bool
	if v := c.QueryParam("is_read"); v != "" {
		b, parseErr := strconv.ParseBool(v)
		if parseErr == nil {
			isRead = &b
		}
	}

	resp, svcErr := h.svc.GetNotifications(c.Request().Context(), userID, isRead, page, pageSize)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────
// PUT /api/v1/notifications/:notificationId/read
// ──────────────────────────────────────────────

func (h *Handler) MarkRead(c echo.Context) error {
	userID, err := extractUserID(c)
	if err != nil {
		return err
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "notificationId is required"},
		})
	}

	resp, svcErr := h.svc.MarkRead(c.Request().Context(), userID, notificationID)
	if svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────
// PUT /api/v1/notifications/read-all
// ──────────────────────────────────────────────

func (h *Handler) MarkAllRead(c echo.Context) error {
	userID, err := extractUserID(c)
	if err != nil {
		return err
	}

	if svcErr := h.svc.MarkAllRead(c.Request().Context(), userID); svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.NoContent(http.StatusNoContent)
}

// ──────────────────────────────────────────────
// POST /api/v1/notifications/device-token
// ──────────────────────────────────────────────

func (h *Handler) RegisterDeviceToken(c echo.Context) error {
	userID, err := extractUserID(c)
	if err != nil {
		return err
	}

	var req model.RegisterDeviceTokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "Invalid request body"},
		})
	}

	if err := validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "VALIDATION_ERROR", Message: "fcm_token is required"},
		})
	}

	deviceInfo := ""
	if req.DeviceInfo != nil {
		deviceInfo = *req.DeviceInfo
	}

	if svcErr := h.svc.RegisterDeviceToken(c.Request().Context(), userID, req.FCMToken, deviceInfo); svcErr != nil {
		return handleError(c, svcErr)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func extractUserID(c echo.Context) (string, error) {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return "", c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error: model.ErrorDetail{Code: "UNAUTHORIZED", Message: "Missing user identification headers"},
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
