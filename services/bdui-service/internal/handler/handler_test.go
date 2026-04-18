package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
	"github.com/coach-link/platform/services/bdui-service/internal/service"
)

// mockService реализует serviceProvider для тестов.
type mockService struct {
	getScreen         func(ctx context.Context, screenID, userID, role string) (*model.BduiSchema, error)
	getTrainingDetail func(ctx context.Context, assignmentID, userID, role string) (*model.BduiSchema, error)
}

func (m *mockService) GetScreen(ctx context.Context, screenID, userID, role string) (*model.BduiSchema, error) {
	return m.getScreen(ctx, screenID, userID, role)
}

func (m *mockService) GetTrainingDetail(ctx context.Context, assignmentID, userID, role string) (*model.BduiSchema, error) {
	return m.getTrainingDetail(ctx, assignmentID, userID, role)
}

func setupEcho(h *Handler) *echo.Echo {
	e := echo.New()
	RegisterRoutes(e, h)
	return e
}

func fakeSchema(screenID string) *model.BduiSchema {
	return &model.BduiSchema{ScreenID: screenID, Version: "1.0.0"}
}

// --- GetScreen ---

func TestGetScreen_MissingUserID(t *testing.T) {
	h := New(&service.Service{})
	e := setupEcho(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bdui/screens/coach-dashboard", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	var body model.ErrorResponse
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Error.Code != "UNAUTHORIZED" {
		t.Errorf("error code = %q, want UNAUTHORIZED", body.Error.Code)
	}
}

func TestGetScreen_TrainingDetailPrefix_Returns400(t *testing.T) {
	svc := &mockService{
		getScreen: func(_ context.Context, _, _, _ string) (*model.BduiSchema, error) {
			t.Fatal("GetScreen should not be called for training-detail path")
			return nil, nil
		},
	}
	e := setupEcho(&Handler{svc: svc})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bdui/screens/training-detail-foo", nil)
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestGetScreen_Success(t *testing.T) {
	svc := &mockService{
		getScreen: func(_ context.Context, screenID, userID, role string) (*model.BduiSchema, error) {
			if screenID != "coach-dashboard" || userID != "u1" || role != "coach" {
				t.Errorf("unexpected args: screenID=%q userID=%q role=%q", screenID, userID, role)
			}
			return fakeSchema("coach-dashboard"), nil
		},
	}
	e := setupEcho(&Handler{svc: svc})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bdui/screens/coach-dashboard", nil)
	req.Header.Set("X-User-ID", "u1")
	req.Header.Set("X-User-Role", "coach")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var schema model.BduiSchema
	_ = json.NewDecoder(rec.Body).Decode(&schema)
	if schema.ScreenID != "coach-dashboard" {
		t.Errorf("screen_id = %q, want coach-dashboard", schema.ScreenID)
	}
}

func TestGetScreen_ServiceError_Forbidden(t *testing.T) {
	svc := &mockService{
		getScreen: func(_ context.Context, _, _, _ string) (*model.BduiSchema, error) {
			return nil, &service.ServiceError{Code: "FORBIDDEN", Message: "access denied", Status: http.StatusForbidden}
		},
	}
	e := setupEcho(&Handler{svc: svc})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bdui/screens/coach-dashboard", nil)
	req.Header.Set("X-User-ID", "u1")
	req.Header.Set("X-User-Role", "athlete")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	var body model.ErrorResponse
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Error.Code != "FORBIDDEN" {
		t.Errorf("error code = %q, want FORBIDDEN", body.Error.Code)
	}
}

func TestGetScreen_ServiceError_NotFound(t *testing.T) {
	svc := &mockService{
		getScreen: func(_ context.Context, _, _, _ string) (*model.BduiSchema, error) {
			return nil, &service.ServiceError{Code: "NOT_FOUND", Message: "unknown screen", Status: http.StatusNotFound}
		},
	}
	e := setupEcho(&Handler{svc: svc})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bdui/screens/unknown-screen", nil)
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// --- GetTrainingDetail ---

func TestGetTrainingDetail_MissingUserID(t *testing.T) {
	e := setupEcho(&Handler{svc: &mockService{}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bdui/screens/training-detail/asn-1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestGetTrainingDetail_Success(t *testing.T) {
	svc := &mockService{
		getTrainingDetail: func(_ context.Context, assignmentID, userID, role string) (*model.BduiSchema, error) {
			if assignmentID != "asn-42" || userID != "u99" || role != "athlete" {
				t.Errorf("unexpected args: assignmentID=%q userID=%q role=%q", assignmentID, userID, role)
			}
			return fakeSchema("training-detail/asn-42"), nil
		},
	}
	e := setupEcho(&Handler{svc: svc})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bdui/screens/training-detail/asn-42", nil)
	req.Header.Set("X-User-ID", "u99")
	req.Header.Set("X-User-Role", "athlete")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var schema model.BduiSchema
	_ = json.NewDecoder(rec.Body).Decode(&schema)
	if schema.ScreenID != "training-detail/asn-42" {
		t.Errorf("screen_id = %q", schema.ScreenID)
	}
}

func TestGetTrainingDetail_NotFound(t *testing.T) {
	svc := &mockService{
		getTrainingDetail: func(_ context.Context, _, _, _ string) (*model.BduiSchema, error) {
			return nil, &service.ServiceError{Code: "NOT_FOUND", Message: "assignment not found", Status: http.StatusNotFound}
		},
	}
	e := setupEcho(&Handler{svc: svc})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bdui/screens/training-detail/missing", nil)
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
