package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coach-link/platform/services/auth-service/internal/config"
	"github.com/coach-link/platform/services/auth-service/internal/handler"
	"github.com/coach-link/platform/services/auth-service/internal/model"
	"github.com/coach-link/platform/services/auth-service/internal/repository"
	"github.com/coach-link/platform/services/auth-service/internal/service"
)

// These tests drive the full HTTP path (Bind → Validate → service → error
// mapping) through Echo, exactly as the running server wires it in cmd/main.go.
// The only seam is the repository, which is replaced by an in-memory mock.

// ── Mocks ──────────────────────────────────────

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ string, _ []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	return &nats.PubAck{}, nil
}

// memRepo is a minimal in-memory AuthRepository so a registered user can be
// logged in within the same test, and refresh tokens round-trip.
type memRepo struct {
	users  map[string]*model.User // keyed by login
	tokens map[string]tokenEntry  // keyed by token hash
}

type tokenEntry struct {
	userID    string
	expiresAt time.Time
}

func newMemRepo() *memRepo {
	return &memRepo{
		users:  make(map[string]*model.User),
		tokens: make(map[string]tokenEntry),
	}
}

func (m *memRepo) CreateUser(_ context.Context, user *model.User) error {
	if _, exists := m.users[user.Login]; exists {
		return repository.ErrLoginAlreadyExists
	}
	user.ID = "user-" + user.Login
	m.users[user.Login] = user
	return nil
}
func (m *memRepo) GetUserByLogin(_ context.Context, login string) (*model.User, error) {
	u, ok := m.users[login]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return u, nil
}
func (m *memRepo) GetUserByID(_ context.Context, id string) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, repository.ErrUserNotFound
}
func (m *memRepo) SaveRefreshToken(_ context.Context, userID, hash string, exp time.Time) error {
	m.tokens[hash] = tokenEntry{userID: userID, expiresAt: exp}
	return nil
}
func (m *memRepo) GetRefreshToken(_ context.Context, hash string) (string, time.Time, error) {
	e, ok := m.tokens[hash]
	if !ok {
		return "", time.Time{}, repository.ErrTokenNotFound
	}
	return e.userID, e.expiresAt, nil
}
func (m *memRepo) DeleteRefreshToken(_ context.Context, hash string) error {
	delete(m.tokens, hash)
	return nil
}

// ── Harness ────────────────────────────────────

// newServer wires Echo + validator + handler exactly like cmd/main.go, backed
// by the supplied repo, and returns it ready for httptest requests.
func newServer(repo service.AuthRepository) *echo.Echo {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 30 * 24 * time.Hour,
		BcryptCost:    4, // minimum cost → fast tests
	}
	svc := service.New(repo, cfg, &mockPublisher{})
	h := handler.New(svc)

	e := echo.New()
	e.HideBanner = true
	e.Validator = handler.NewValidator()
	h.Register(e)
	return e
}

func do(e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// ── Register: bad input ────────────────────────

func TestHandleRegister_MalformedJSON_Returns400(t *testing.T) {
	e := newServer(newMemRepo())
	rec := do(e, http.MethodPost, "/api/v1/auth/register", `{"login": "alice"`) // truncated JSON
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleRegister_InvalidRole_Returns400(t *testing.T) {
	e := newServer(newMemRepo())
	body := `{"login":"alice","email":"a@b.com","password":"password123","full_name":"Alice","role":"superadmin"}`
	rec := do(e, http.MethodPost, "/api/v1/auth/register", body)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), handler.CodeValidationError)
}

func TestHandleRegister_ShortPassword_Returns400(t *testing.T) {
	e := newServer(newMemRepo())
	body := `{"login":"alice","email":"a@b.com","password":"short","full_name":"Alice","role":"athlete"}`
	rec := do(e, http.MethodPost, "/api/v1/auth/register", body)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleRegister_MissingFields_Returns400(t *testing.T) {
	e := newServer(newMemRepo())
	rec := do(e, http.MethodPost, "/api/v1/auth/register", `{}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleRegister_Valid_Returns201(t *testing.T) {
	e := newServer(newMemRepo())
	body := `{"login":"alice","email":"alice@example.com","password":"password123","full_name":"Alice","role":"athlete"}`
	rec := do(e, http.MethodPost, "/api/v1/auth/register", body)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "access_token")
}

func TestHandleRegister_DuplicateLogin_Returns409(t *testing.T) {
	e := newServer(newMemRepo())
	body := `{"login":"alice","email":"alice@example.com","password":"password123","full_name":"Alice","role":"athlete"}`

	require.Equal(t, http.StatusCreated, do(e, http.MethodPost, "/api/v1/auth/register", body).Code)

	rec := do(e, http.MethodPost, "/api/v1/auth/register", body) // same login again
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), handler.CodeLoginAlreadyExists)
}

// ── Login ──────────────────────────────────────

func TestHandleLogin_NonexistentUser_Returns401(t *testing.T) {
	e := newServer(newMemRepo())
	rec := do(e, http.MethodPost, "/api/v1/auth/login", `{"login":"ghost","password":"whatever"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), handler.CodeInvalidCredentials)
}

func TestHandleLogin_WrongPassword_Returns401(t *testing.T) {
	e := newServer(newMemRepo())
	reg := `{"login":"bob","email":"bob@example.com","password":"correct-password","full_name":"Bob","role":"coach"}`
	require.Equal(t, http.StatusCreated, do(e, http.MethodPost, "/api/v1/auth/register", reg).Code)

	rec := do(e, http.MethodPost, "/api/v1/auth/login", `{"login":"bob","password":"wrong-password"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), handler.CodeInvalidCredentials)
}

func TestHandleLogin_MissingPassword_Returns400(t *testing.T) {
	e := newServer(newMemRepo())
	rec := do(e, http.MethodPost, "/api/v1/auth/login", `{"login":"bob"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleLogin_Valid_Returns200(t *testing.T) {
	e := newServer(newMemRepo())
	reg := `{"login":"carol","email":"carol@example.com","password":"mypassword","full_name":"Carol","role":"athlete"}`
	require.Equal(t, http.StatusCreated, do(e, http.MethodPost, "/api/v1/auth/register", reg).Code)

	rec := do(e, http.MethodPost, "/api/v1/auth/login", `{"login":"carol","password":"mypassword"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "access_token")
}

// ── Refresh ────────────────────────────────────

func TestHandleRefresh_UnknownToken_Returns401(t *testing.T) {
	e := newServer(newMemRepo())
	rec := do(e, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"00000000-0000-0000-0000-000000000000"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), handler.CodeTokenInvalid)
}

func TestHandleRefresh_MissingToken_Returns400(t *testing.T) {
	e := newServer(newMemRepo())
	rec := do(e, http.MethodPost, "/api/v1/auth/refresh", `{}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleRefresh_Valid_RotatesToken(t *testing.T) {
	repo := newMemRepo()
	e := newServer(repo)
	reg := `{"login":"dave","email":"dave@example.com","password":"password123","full_name":"Dave","role":"coach"}`
	regRec := do(e, http.MethodPost, "/api/v1/auth/register", reg)
	require.Equal(t, http.StatusCreated, regRec.Code)

	var auth model.AuthResponse
	require.NoError(t, decodeJSON(regRec.Body.String(), &auth))
	require.NotEmpty(t, auth.RefreshToken)

	rec := do(e, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"`+auth.RefreshToken+`"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "access_token")

	// Old refresh token must no longer work (single-use rotation).
	reuse := do(e, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"`+auth.RefreshToken+`"}`)
	assert.Equal(t, http.StatusUnauthorized, reuse.Code)
}

func decodeJSON(body string, v interface{}) error {
	return json.Unmarshal([]byte(body), v)
}
