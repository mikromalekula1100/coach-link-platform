package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GroupMemberInfo represents an athlete returned by the User Service internal API.
type GroupMemberInfo struct {
	AthleteID string `json:"athlete_id"`
	FullName  string `json:"full_name"`
	Login     string `json:"login"`
}

// UserClient communicates with the User Service internal API.
type UserClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewUserClient creates a new UserClient pointing at the given base URL (e.g. http://localhost:8002).
func NewUserClient(baseURL string) *UserClient {
	return &UserClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetUserByID calls GET /internal/users/{userId} on the User Service.
func (c *UserClient) GetUserByID(ctx context.Context, userID string) (*GroupMemberInfo, error) {
	url := fmt.Sprintf("%s/internal/users/%s", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-service returned status %d", resp.StatusCode)
	}

	var info GroupMemberInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &info, nil
}

// GetGroupMembers calls GET /internal/groups/{groupId}/members on the User Service.
func (c *UserClient) GetGroupMembers(ctx context.Context, groupID string) ([]GroupMemberInfo, error) {
	url := fmt.Sprintf("%s/internal/groups/%s/members", c.baseURL, groupID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("group not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-service returned status %d", resp.StatusCode)
	}

	var members []GroupMemberInfo
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return members, nil
}
