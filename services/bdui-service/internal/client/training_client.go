package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

type TrainingClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTrainingClient(baseURL string) *TrainingClient {
	return &TrainingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetRecentReports возвращает последние отчёты спортсменов тренера.
// GET /internal/reports/coach?coach_id=&limit=
func (c *TrainingClient) GetRecentReports(ctx context.Context, coachID string, limit int) ([]model.ReportWithPlan, error) {
	params := url.Values{}
	params.Set("coach_id", coachID)
	params.Set("limit", fmt.Sprintf("%d", limit))
	reqURL := fmt.Sprintf("%s/internal/reports/coach?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to training-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("training-service returned status %d for GetRecentReports", resp.StatusCode)
	}

	var reports []model.ReportWithPlan
	if err := json.NewDecoder(resp.Body).Decode(&reports); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return reports, nil
}

// GetUpcomingAssignments возвращает ближайшие задания.
// GET /internal/assignments/upcoming?user_id=&role=&limit=
func (c *TrainingClient) GetUpcomingAssignments(ctx context.Context, userID, role string, limit int) ([]model.AssignmentListItem, error) {
	params := url.Values{}
	params.Set("user_id", userID)
	params.Set("role", role)
	params.Set("limit", fmt.Sprintf("%d", limit))
	reqURL := fmt.Sprintf("%s/internal/assignments/upcoming?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to training-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("training-service returned status %d for GetUpcomingAssignments", resp.StatusCode)
	}

	var assignments []model.AssignmentListItem
	if err := json.NewDecoder(resp.Body).Decode(&assignments); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return assignments, nil
}

// GetAssignment возвращает задание по ID.
// Использует существующий публичный эндпоинт GET /api/v1/training/assignments/{assignmentId},
// передавая заголовки пользователя.
func (c *TrainingClient) GetAssignment(ctx context.Context, assignmentID, userID, userRole string) (*model.AssignmentDetail, error) {
	reqURL := fmt.Sprintf("%s/api/v1/training/assignments/%s", c.baseURL, assignmentID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-User-Role", userRole)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to training-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("training-service returned status %d for GetAssignment", resp.StatusCode)
	}

	var assignment model.AssignmentDetail
	if err := json.NewDecoder(resp.Body).Decode(&assignment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &assignment, nil
}
