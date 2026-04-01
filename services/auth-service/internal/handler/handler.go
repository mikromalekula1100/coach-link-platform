package handler

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/auth-service/internal/model"
	"github.com/coach-link/platform/services/auth-service/internal/repository"
	"github.com/coach-link/platform/services/auth-service/internal/service"
)

const (
	CodeValidationError   = "VALIDATION_ERROR"
	CodeInvalidLogin      = "INVALID_LOGIN_FORMAT"
	CodeLoginAlreadyExists = "LOGIN_ALREADY_EXISTS"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeTokenExpired       = "TOKEN_EXPIRED"
	CodeTokenInvalid       = "INVALID_TOKEN"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeBadRequest         = "BAD_REQUEST"
)

type CustomValidator struct {
	validator *validator.Validate
}

func NewValidator() *CustomValidator {
	return &CustomValidator{validator: validator.New()}
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(e *echo.Echo) {
	g := e.Group("/api/v1/auth")
	g.POST("/register", h.HandleRegister)
	g.POST("/login", h.HandleLogin)
	g.POST("/refresh", h.HandleRefresh)
	g.POST("/logout", h.HandleLogout)
}

func (h *Handler) HandleRegister(c echo.Context) error {
	var req model.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, CodeBadRequest, "Invalid request body", nil)
	}

	if err := c.Validate(&req); err != nil {
		return sendValidationError(c, err)
	}

	resp, err := h.svc.Register(c.Request().Context(), req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) HandleLogin(c echo.Context) error {
	var req model.LoginRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, CodeBadRequest, "Invalid request body", nil)
	}

	if err := c.Validate(&req); err != nil {
		return sendValidationError(c, err)
	}

	resp, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) HandleRefresh(c echo.Context) error {
	var req model.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, CodeBadRequest, "Invalid request body", nil)
	}

	if err := c.Validate(&req); err != nil {
		return sendValidationError(c, err)
	}

	resp, err := h.svc.Refresh(c.Request().Context(), req)
	if err != nil {
		return handleServiceError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) HandleLogout(c echo.Context) error {
	var req model.LogoutRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, CodeBadRequest, "Invalid request body", nil)
	}

	if err := c.Validate(&req); err != nil {
		return sendValidationError(c, err)
	}

	if err := h.svc.Logout(c.Request().Context(), req); err != nil {
		return handleServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func handleServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidLogin):
		return sendError(c, http.StatusBadRequest, CodeInvalidLogin, err.Error(), nil)
	case errors.Is(err, repository.ErrLoginAlreadyExists):
		return sendError(c, http.StatusConflict, CodeLoginAlreadyExists, "A user with this login already exists", nil)
	case errors.Is(err, service.ErrInvalidCredentials):
		return sendError(c, http.StatusUnauthorized, CodeInvalidCredentials, "Invalid login or password", nil)
	case errors.Is(err, service.ErrTokenExpired):
		return sendError(c, http.StatusUnauthorized, CodeTokenExpired, "Refresh token has expired", nil)
	case errors.Is(err, service.ErrTokenInvalid):
		return sendError(c, http.StatusUnauthorized, CodeTokenInvalid, "Invalid refresh token", nil)
	default:
		return sendError(c, http.StatusInternalServerError, CodeInternalError, "Internal server error", nil)
	}
}

func sendValidationError(c echo.Context, err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		details := make([]map[string]string, 0, len(ve))
		for _, fe := range ve {
			details = append(details, map[string]string{
				"field":   fe.Field(),
				"tag":     fe.Tag(),
				"value":   fe.Param(),
				"message": formatValidationMessage(fe),
			})
		}
		return sendError(c, http.StatusBadRequest, CodeValidationError, "Validation failed", details)
	}
	return sendError(c, http.StatusBadRequest, CodeValidationError, err.Error(), nil)
}

func sendError(c echo.Context, status int, code, message string, details interface{}) error {
	return c.JSON(status, model.ErrorResponse{
		Error: model.ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func formatValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "email":
		return fe.Field() + " must be a valid email address"
	case "min":
		return fe.Field() + " must be at least " + fe.Param() + " characters"
	case "max":
		return fe.Field() + " must be at most " + fe.Param() + " characters"
	case "oneof":
		return fe.Field() + " must be one of: " + fe.Param()
	default:
		return fe.Field() + " failed validation: " + fe.Tag()
	}
}
