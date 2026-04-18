package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

type UserClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetUserProfile возвращает профиль пользователя.
// GET /internal/users/{userId}
func (c *UserClient) GetUserProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	reqURL := fmt.Sprintf("%s/internal/users/%s", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-service returned status %d for GetUserProfile", resp.StatusCode)
	}

	var profile model.UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &profile, nil
}

// GetAthletes возвращает список спортсменов тренера.
// Использует существующий публичный эндпоинт GET /api/v1/connections/athletes,
// передавая заголовки пользователя (X-User-ID, X-User-Role).
func (c *UserClient) GetAthletes(ctx context.Context, userID string) ([]model.AthleteInfo, error) {
	reqURL := fmt.Sprintf("%s/api/v1/connections/athletes?page_size=100", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-User-Role", "coach")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-service returned status %d for GetAthletes", resp.StatusCode)
	}

	var paginated model.PaginatedAthletes
	if err := json.NewDecoder(resp.Body).Decode(&paginated); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return paginated.Items, nil
}

// GetPendingRequests возвращает входящие заявки на подключение.
// Использует существующий публичный эндпоинт GET /api/v1/connections/requests/incoming,
// передавая заголовки пользователя.
func (c *UserClient) GetPendingRequests(ctx context.Context, userID string) ([]model.ConnectionRequest, error) {
	reqURL := fmt.Sprintf("%s/api/v1/connections/requests/incoming?status=pending&page_size=100", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-User-Role", "coach")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-service returned status %d for GetPendingRequests", resp.StatusCode)
	}

	var paginated model.PaginatedConnectionRequests
	if err := json.NewDecoder(resp.Body).Decode(&paginated); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return paginated.Items, nil
}

// GetCoach возвращает тренера спортсмена.
// Использует существующий публичный эндпоинт GET /api/v1/connections/coach,
// передавая заголовки пользователя.
func (c *UserClient) GetCoach(ctx context.Context, userID string) (*model.CoachInfo, error) {
	reqURL := fmt.Sprintf("%s/api/v1/connections/coach", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-User-Role", "athlete")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-service returned status %d for GetCoach", resp.StatusCode)
	}

	var coach model.CoachInfo
	if err := json.NewDecoder(resp.Body).Decode(&coach); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &coach, nil
}
