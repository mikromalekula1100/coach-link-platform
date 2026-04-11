package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAuthRegisterCoachSuccess(t *testing.T) {
	login := uniqueLogin("authtest-coach")
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{
		"login":     login,
		"email":     uniqueEmail("authtest-coach"),
		"password":  defaultPassword,
		"full_name": "Auth Test Coach",
		"role":      "coach",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var resp AuthResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Greater(t, resp.ExpiresIn, 0)
	assert.Equal(t, login, resp.User.Login)
	assert.Equal(t, "coach", resp.User.Role)
	assert.Equal(t, "Auth Test Coach", resp.User.FullName)
	assert.NotEmpty(t, resp.User.ID)
}

func testAuthRegisterAthleteSuccess(t *testing.T) {
	login := uniqueLogin("authtest-athlete")
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{
		"login":     login,
		"email":     uniqueEmail("authtest-athlete"),
		"password":  defaultPassword,
		"full_name": "Auth Test Athlete",
		"role":      "athlete",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, data)

	var resp AuthResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.Equal(t, "athlete", resp.User.Role)
	assert.NotEmpty(t, resp.AccessToken)
}

func testAuthRegisterDuplicateLogin(t *testing.T) {
	// coach1Login was registered in TestMain
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{
		"login":     coach1Login,
		"email":     uniqueEmail("dup"),
		"password":  defaultPassword,
		"full_name": "Duplicate User",
		"role":      "coach",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusConflict, status, data)

	svcErr, err := parseServiceError(data)
	require.NoError(t, err)
	assert.Equal(t, "LOGIN_ALREADY_EXISTS", svcErr.Error.Code)
}

func testAuthRegisterLoginTooShort(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{
		"login":     "ab",
		"email":     uniqueEmail("short"),
		"password":  defaultPassword,
		"full_name": "Short Login",
		"role":      "coach",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testAuthRegisterLoginInvalidChars(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{
		"login":     "bad login!",
		"email":     uniqueEmail("invalid"),
		"password":  defaultPassword,
		"full_name": "Invalid Login",
		"role":      "coach",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testAuthRegisterInvalidEmail(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{
		"login":     uniqueLogin("bademail"),
		"email":     "not-an-email",
		"password":  defaultPassword,
		"full_name": "Bad Email",
		"role":      "coach",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testAuthRegisterPasswordTooShort(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{
		"login":     uniqueLogin("shortpw"),
		"email":     uniqueEmail("shortpw"),
		"password":  "short",
		"full_name": "Short Password",
		"role":      "coach",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testAuthRegisterInvalidRole(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{
		"login":     uniqueLogin("badrole"),
		"email":     uniqueEmail("badrole"),
		"password":  defaultPassword,
		"full_name": "Bad Role",
		"role":      "admin",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testAuthRegisterEmptyBody(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/register", map[string]string{}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, data)
}

func testAuthLoginSuccess(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/login", map[string]string{
		"login":    coach1Login,
		"password": defaultPassword,
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp AuthResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, coach1ID, resp.User.ID)
}

func testAuthLoginWrongPassword(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/login", map[string]string{
		"login":    coach1Login,
		"password": "wrong-password-here",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, data)
}

func testAuthLoginNonexistentUser(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/login", map[string]string{
		"login":    "nonexistent-user-zzz",
		"password": defaultPassword,
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, data)
}

func testAuthRefreshSuccess(t *testing.T) {
	// First login to get a fresh refresh token
	_, loginData, err := client.Post("/api/v1/auth/login", map[string]string{
		"login":    athlete1Login,
		"password": defaultPassword,
	}, "")
	require.NoError(t, err)
	var loginResp AuthResponse
	require.NoError(t, json.Unmarshal(loginData, &loginResp))

	status, data, err := client.Post("/api/v1/auth/refresh", map[string]string{
		"refresh_token": loginResp.RefreshToken,
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, data)

	var resp AuthResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}

func testAuthRefreshInvalidToken(t *testing.T) {
	status, data, err := client.Post("/api/v1/auth/refresh", map[string]string{
		"refresh_token": "invalid-token-abc-123",
	}, "")
	require.NoError(t, err)
	requireStatus(t, http.StatusUnauthorized, status, data)
}
