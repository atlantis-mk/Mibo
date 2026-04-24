package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const sessionTTL = 30 * 24 * time.Hour

type Service struct {
	db *gorm.DB
}

type LoginResult struct {
	Token     string        `json:"token"`
	ExpiresAt time.Time     `json:"expires_at"`
	User      database.User `json:"user"`
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Register(ctx context.Context, username, password string) (database.User, error) {
	normalized, err := normalizeUsername(username)
	if err != nil {
		return database.User{}, err
	}
	if err := validatePassword(password); err != nil {
		return database.User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return database.User{}, err
	}

	user := database.User{
		Username:     normalized,
		PasswordHash: string(hash),
		Role:         "user",
	}
	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		return database.User{}, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	normalized, err := normalizeUsername(username)
	if err != nil {
		return LoginResult{}, err
	}

	var user database.User
	if err := s.db.WithContext(ctx).Where("username = ?", normalized).First(&user).Error; err != nil {
		return LoginResult{}, fmt.Errorf("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, fmt.Errorf("invalid username or password")
	}

	token, err := generateToken()
	if err != nil {
		return LoginResult{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(sessionTTL)
	session := database.Session{
		UserID:     user.ID,
		TokenHash:  tokenHash(token),
		ExpiresAt:  expiresAt,
		LastUsedAt: &now,
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return LoginResult{}, err
	}

	return LoginResult{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (database.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return database.User{}, fmt.Errorf("missing session token")
	}

	var session database.Session
	if err := s.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash(token)).
		First(&session).Error; err != nil {
		return database.User{}, fmt.Errorf("invalid session token")
	}

	now := time.Now().UTC()
	if session.ExpiresAt.Before(now) {
		_ = s.db.WithContext(ctx).Delete(&database.Session{}, session.ID).Error
		return database.User{}, fmt.Errorf("session expired")
	}

	var user database.User
	if err := s.db.WithContext(ctx).First(&user, session.UserID).Error; err != nil {
		return database.User{}, err
	}

	_ = s.db.WithContext(ctx).Model(&database.Session{}).Where("id = ?", session.ID).Update("last_used_at", now).Error
	return user, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("missing session token")
	}
	return s.db.WithContext(ctx).Where("token_hash = ?", tokenHash(token)).Delete(&database.Session{}).Error
}

func normalizeUsername(username string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(username))
	if len(normalized) < 3 || len(normalized) > 128 {
		return "", fmt.Errorf("username must be between 3 and 128 characters")
	}
	return normalized, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

func generateToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
