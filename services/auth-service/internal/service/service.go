package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/coach-link/platform/pkg/events"
	"github.com/coach-link/platform/services/auth-service/internal/config"
	"github.com/coach-link/platform/services/auth-service/internal/model"
	"github.com/coach-link/platform/services/auth-service/internal/repository"
)

var (
	ErrInvalidLogin      = errors.New("login must contain only letters, digits, and hyphens")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired       = errors.New("refresh token expired")
	ErrTokenInvalid       = errors.New("invalid refresh token")
)

var loginRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

type Service struct {
	repo *repository.Repository
	cfg  *config.Config
	js   nats.JetStreamContext
}

func New(repo *repository.Repository, cfg *config.Config, js nats.JetStreamContext) *Service {
	return &Service{
		repo: repo,
		cfg:  cfg,
		js:   js,
	}
}

func (s *Service) Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
	if !loginRegex.MatchString(req.Login) {
		return nil, ErrInvalidLogin
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Login:        req.Login,
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Role:         req.Role,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	s.publishUserRegistered(user)

	resp, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return resp, nil
}

func (s *Service) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.repo.GetUserByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	resp, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return resp, nil
}

func (s *Service) Refresh(ctx context.Context, req model.RefreshRequest) (*model.AuthResponse, error) {
	tokenHash := hashToken(req.RefreshToken)

	userID, expiresAt, err := s.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil, ErrTokenInvalid
		}
		return nil, err
	}

	if time.Now().UTC().After(expiresAt) {
		_ = s.repo.DeleteRefreshToken(ctx, tokenHash)
		return nil, ErrTokenExpired
	}

	if err := s.repo.DeleteRefreshToken(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("delete old refresh token: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return resp, nil
}

func (s *Service) Logout(ctx context.Context, req model.LogoutRequest) error {
	tokenHash := hashToken(req.RefreshToken)
	return s.repo.DeleteRefreshToken(ctx, tokenHash)
}

func (s *Service) generateTokens(ctx context.Context, user *model.User) (*model.AuthResponse, error) {
	now := time.Now().UTC()
	accessExp := now.Add(s.cfg.JWTAccessTTL)

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"login": user.Login,
		"role":  user.Role,
		"exp":   accessExp.Unix(),
		"iat":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken := uuid.New().String()
	refreshHash := hashToken(refreshToken)
	refreshExp := now.Add(s.cfg.JWTRefreshTTL)

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshHash, refreshExp); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.JWTAccessTTL.Seconds()),
		User:         model.UserToProfile(user),
	}, nil
}

func (s *Service) publishUserRegistered(user *model.User) {
	evt := events.NewEvent(events.SubjectUserRegistered, events.UserRegisteredPayload{
		UserID:   user.ID,
		Login:    user.Login,
		FullName: user.FullName,
		Email:    user.Email,
		Role:     user.Role,
	})

	data, err := evt.Marshal()
	if err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("failed to marshal user.registered event")
		return
	}

	if _, err := s.js.Publish(events.SubjectUserRegistered, data); err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("failed to publish user.registered event")
	}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
