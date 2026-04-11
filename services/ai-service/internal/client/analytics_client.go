package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coach-link/platform/services/ai-service/internal/model"
)

// AnalyticsClient communicates with the Analytics Service internal API.
type AnalyticsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAnalyticsClient creates a new AnalyticsClient pointing at the given base URL.
func NewAnalyticsClient(baseURL string) *AnalyticsClient {
	return &AnalyticsClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAthleteSummary fetches aggregated summary for an athlete.
func (c *AnalyticsClient) GetAthleteSummary(ctx context.Context, athleteID string) (*model.AthleteSummary, error) {
	reqURL := fmt.Sprintf("%s/internal/analytics/athletes/%s/summary", c.baseURL, athleteID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to analytics-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("analytics-service returned status %d for GetAthleteSummary", resp.StatusCode)
	}

	var summary model.AthleteSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &summary, nil
}

// GetAthleteReports fetches reports with plan info for an athlete.
func (c *AnalyticsClient) GetAthleteReports(ctx context.Context, athleteID string) ([]model.ReportWithPlan, error) {
	reqURL := fmt.Sprintf("%s/internal/analytics/athletes/%s/reports", c.baseURL, athleteID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to analytics-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("analytics-service returned status %d for GetAthleteReports", resp.StatusCode)
	}

	var reports []model.ReportWithPlan
	if err := json.NewDecoder(resp.Body).Decode(&reports); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return reports, nil
}
