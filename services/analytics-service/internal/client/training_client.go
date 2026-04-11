package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coach-link/platform/services/analytics-service/internal/model"
)

// TrainingClient communicates with the Training Service internal API.
type TrainingClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewTrainingClient creates a new TrainingClient pointing at the given base URL.
func NewTrainingClient(baseURL string) *TrainingClient {
	return &TrainingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetReports fetches reports for an athlete, optionally filtered by date range.
func (c *TrainingClient) GetReports(ctx context.Context, athleteID, dateFrom, dateTo string) ([]model.ReportWithPlan, error) {
	params := url.Values{}
	params.Set("athlete_id", athleteID)
	if dateFrom != "" {
		params.Set("date_from", dateFrom)
	}
	if dateTo != "" {
		params.Set("date_to", dateTo)
	}

	reqURL := fmt.Sprintf("%s/internal/reports?%s", c.baseURL, params.Encode())

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
		return nil, fmt.Errorf("training-service returned status %d for GetReports", resp.StatusCode)
	}

	var reports []model.ReportWithPlan
	if err := json.NewDecoder(resp.Body).Decode(&reports); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return reports, nil
}

// GetAthleteStats fetches pre-aggregated stats for an athlete.
func (c *TrainingClient) GetAthleteStats(ctx context.Context, athleteID string) (*model.AthleteStats, error) {
	reqURL := fmt.Sprintf("%s/internal/athletes/%s/stats", c.baseURL, athleteID)

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
		return nil, fmt.Errorf("training-service returned status %d for GetAthleteStats", resp.StatusCode)
	}

	var stats model.AthleteStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &stats, nil
}

// GetCoachAthleteIDs returns athlete IDs for a coach.
func (c *TrainingClient) GetCoachAthleteIDs(ctx context.Context, coachID string) ([]string, error) {
	reqURL := fmt.Sprintf("%s/internal/coach/%s/athletes", c.baseURL, coachID)

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
		return nil, fmt.Errorf("training-service returned status %d for GetCoachAthleteIDs", resp.StatusCode)
	}

	var ids []string
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return ids, nil
}

// GetCoachOverview returns coach's aggregate stats.
func (c *TrainingClient) GetCoachOverview(ctx context.Context, coachID string) (*model.CoachOverviewStats, error) {
	reqURL := fmt.Sprintf("%s/internal/coach/%s/overview", c.baseURL, coachID)

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
		return nil, fmt.Errorf("training-service returned status %d for GetCoachOverview", resp.StatusCode)
	}

	var stats model.CoachOverviewStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &stats, nil
}
