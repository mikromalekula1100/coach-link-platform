package client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// UserClient communicates with the User Service internal API.
type UserClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// BelongsToCoach returns true if the athlete is connected to the given coach.
func (c *UserClient) BelongsToCoach(ctx context.Context, coachID, athleteID string) (bool, error) {
	reqURL := fmt.Sprintf("%s/internal/coach/%s/has-athlete/%s", c.baseURL, coachID, athleteID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request to user-service: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("user-service returned unexpected status %d", resp.StatusCode)
	}
}
