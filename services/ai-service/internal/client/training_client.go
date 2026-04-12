package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coach-link/platform/services/ai-service/internal/model"
)

// TrainingClient communicates with the Training Service internal API.
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

// GetCoachAthleteIDs returns all unique athlete IDs for the given coach.
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
		return nil, fmt.Errorf("training-service returned status %d", resp.StatusCode)
	}

	var ids []string
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return ids, nil
}

// GetReportsByAthlete fetches reports for a specific athlete within an optional date range.
func (c *TrainingClient) GetReportsByAthlete(ctx context.Context, athleteID, dateFrom, dateTo string) ([]model.ReportWithPlan, error) {
	reqURL := fmt.Sprintf("%s/internal/reports?athlete_id=%s", c.baseURL, athleteID)
	if dateFrom != "" {
		reqURL += "&date_from=" + dateFrom
	}
	if dateTo != "" {
		reqURL += "&date_to=" + dateTo
	}

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
		return nil, fmt.Errorf("training-service returned status %d", resp.StatusCode)
	}

	var reports []model.ReportWithPlan
	if err := json.NewDecoder(resp.Body).Decode(&reports); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return reports, nil
}
