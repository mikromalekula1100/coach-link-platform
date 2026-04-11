package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	defaultBaseURL  = "http://api-gateway:8080"
	defaultPassword = "password123"
	natsSyncDelay   = 3 * time.Second
	notifyTimeout   = 5 * time.Second
	notifyPollDelay = 200 * time.Millisecond
)

var (
	runSuffix string
	client    *APIClient
)

func getBaseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return defaultBaseURL
}

// APIClient wraps http.Client with JSON helpers.
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *APIClient) DoJSON(method, path string, body interface{}, token string) (int, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("read body: %w", err)
	}

	return resp.StatusCode, respBody, nil
}

func (c *APIClient) Get(path, token string) (int, []byte, error) {
	return c.DoJSON(http.MethodGet, path, nil, token)
}

func (c *APIClient) Post(path string, body interface{}, token string) (int, []byte, error) {
	return c.DoJSON(http.MethodPost, path, body, token)
}

func (c *APIClient) Put(path string, body interface{}, token string) (int, []byte, error) {
	return c.DoJSON(http.MethodPut, path, body, token)
}

func (c *APIClient) Delete(path, token string) (int, []byte, error) {
	return c.DoJSON(http.MethodDelete, path, nil, token)
}

// uniqueLogin returns a login name unique to this test run.
func uniqueLogin(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, runSuffix)
}

// uniqueEmail returns an email unique to this test run.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@test.coachlink.dev", prefix, runSuffix)
}

// registerUser calls POST /api/v1/auth/register and returns parsed AuthResponse.
func registerUser(login, email, fullName, role string) (*AuthResponse, int, error) {
	body := map[string]string{
		"login":     login,
		"email":     email,
		"password":  defaultPassword,
		"full_name": fullName,
		"role":      role,
	}
	status, data, err := client.Post("/api/v1/auth/register", body, "")
	if err != nil {
		return nil, 0, err
	}
	if status != http.StatusCreated {
		return nil, status, fmt.Errorf("register returned %d: %s", status, string(data))
	}
	var resp AuthResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, status, fmt.Errorf("unmarshal: %w", err)
	}
	return &resp, status, nil
}

// loginUser calls POST /api/v1/auth/login and returns parsed AuthResponse.
func loginUser(login, password string) (*AuthResponse, int, error) {
	body := map[string]string{
		"login":    login,
		"password": password,
	}
	status, data, err := client.Post("/api/v1/auth/login", body, "")
	if err != nil {
		return nil, 0, err
	}
	if status != http.StatusOK {
		return nil, status, fmt.Errorf("login returned %d: %s", status, string(data))
	}
	var resp AuthResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, status, fmt.Errorf("unmarshal: %w", err)
	}
	return &resp, status, nil
}

// waitForNotifications polls GET /notifications until unread_count >= minCount or timeout.
func waitForNotifications(t *testing.T, token string, minCount int, timeout time.Duration) *NotificationsListResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastResult NotificationsListResponse
	for {
		status, data, err := client.Get("/api/v1/notifications", token)
		if err == nil && status == http.StatusOK {
			if err := json.Unmarshal(data, &lastResult); err == nil {
				if lastResult.UnreadCount >= minCount {
					return &lastResult
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d notifications, got %d", minCount, lastResult.UnreadCount)
		}
		time.Sleep(notifyPollDelay)
	}
}

// parseJSON unmarshals JSON into the given type.
func parseJSON[T any](data []byte) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	return v, err
}

// parseServiceError tries to parse the nested error format from services.
func parseServiceError(data []byte) (ServiceErrorResponse, error) {
	var v ServiceErrorResponse
	err := json.Unmarshal(data, &v)
	return v, err
}
