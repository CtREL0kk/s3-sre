package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"myS3/internal/auth"
	"myS3/internal/domain"
	"myS3/internal/repository/refresh"
	"myS3/internal/repository/user"
)

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type AuthService struct {
	users      *user.UserRepo
	refreshes  *refresh.RefreshRepo
	manager    *auth.Manager
	refreshTTL time.Duration
}

func NewAuthService(
	users *user.UserRepo,
	refreshes *refresh.RefreshRepo,
	manager *auth.Manager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:      users,
		refreshes:  refreshes,
		manager:    manager,
		refreshTTL: refreshTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	u := &domain.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return domain.User{}, err
	}
	return *u, nil
}

func (s *AuthService) Login(ctx context.Context, login, password string) (Tokens, error) {
	u, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Tokens{}, domain.ErrInvalidCredentials
		}
		return Tokens{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, domain.ErrInvalidCredentials
	}

	return s.issueTokens(ctx, u)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	rt, err := s.refreshes.GetByHash(ctx, hashToken(refreshToken))
	if err != nil {
		return Tokens{}, err
	}
	if rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		return Tokens{}, domain.ErrInvalidToken
	}
	u, err := s.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return Tokens{}, err
	}
	if err := s.refreshes.Revoke(ctx, rt.ID); err != nil {
		return Tokens{}, err
	}
	return s.issueTokens(ctx, u)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	rt, err := s.refreshes.GetByHash(ctx, hashToken(refreshToken))
	if err != nil {
		return err
	}
	if rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		return domain.ErrInvalidToken
	}
	return s.refreshes.Revoke(ctx, rt.ID)
}

func (s *AuthService) issueTokens(ctx context.Context, u *domain.User) (Tokens, error) {
	access, _, err := s.manager.GenerateAccessToken(u.ID, u.IsAdmin)
	if err != nil {
		return Tokens{}, err
	}

	refresh, refreshHash, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}

	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: refreshHash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}
	if err := s.refreshes.Create(ctx, rt); err != nil {
		return Tokens{}, err
	}

	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func generateRefreshToken() (token, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	token = hex.EncodeToString(buf)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
