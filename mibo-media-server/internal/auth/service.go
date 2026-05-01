package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const sessionTTL = 30 * 24 * time.Hour

var (
	ErrDuplicateUsername = errors.New("username already exists")
	ErrInvalidRole       = errors.New("role must be user or admin")
	ErrCurrentSession    = errors.New("current session must be signed out with logout")
	ErrSessionNotFound   = errors.New("session not found")
)

type Service struct {
	db *gorm.DB
}

type LoginResult struct {
	Token     string        `json:"token"`
	ExpiresAt time.Time     `json:"expires_at"`
	User      database.User `json:"user"`
}

type LoginMetadata struct {
	UserAgent  string
	RemoteAddr string
	DeviceName string
	ClientType string
}

type LoginSession struct {
	ID         uint       `json:"id"`
	UserAgent  string     `json:"user_agent"`
	RemoteAddr string     `json:"remote_addr"`
	DeviceName string     `json:"device_name"`
	ClientType string     `json:"client_type"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	IsCurrent  bool       `json:"is_current"`
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

	var user database.User
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var userCount int64
		if err := tx.Model(&database.User{}).Count(&userCount).Error; err != nil {
			return err
		}

		role := "user"
		if userCount == 0 {
			role = "admin"
		}

		user = database.User{
			Username:     normalized,
			PasswordHash: string(hash),
			Role:         role,
		}
		return tx.Create(&user).Error
	})
	if err != nil {
		return database.User{}, err
	}
	return user, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]database.User, error) {
	var users []database.User
	if err := s.db.WithContext(ctx).Order("username ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Service) CreateUser(ctx context.Context, username, password, role string) (database.User, error) {
	normalized, err := normalizeUsername(username)
	if err != nil {
		return database.User{}, err
	}
	if err := validatePassword(password); err != nil {
		return database.User{}, err
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "user" && role != "admin" {
		return database.User{}, ErrInvalidRole
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return database.User{}, err
	}

	user := database.User{
		Username:     normalized,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		if isDuplicateUsernameError(err) {
			return database.User{}, ErrDuplicateUsername
		}
		return database.User{}, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, username, password string, metadata ...LoginMetadata) (LoginResult, error) {
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
	if len(metadata) > 0 {
		applyLoginMetadata(&session, metadata[0])
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

func (s *Service) ListLoginSessions(ctx context.Context, userID uint, currentToken string) ([]LoginSession, error) {
	currentHash := tokenHash(strings.TrimSpace(currentToken))
	var sessions []database.Session
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_used_at IS NULL ASC, last_used_at DESC, created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	result := make([]LoginSession, 0, len(sessions))
	for _, session := range sessions {
		result = append(result, LoginSession{
			ID:         session.ID,
			UserAgent:  stringValue(session.UserAgent),
			RemoteAddr: stringValue(session.RemoteAddr),
			DeviceName: stringValue(session.DeviceName),
			ClientType: stringValue(session.ClientType),
			ExpiresAt:  session.ExpiresAt,
			LastUsedAt: session.LastUsedAt,
			CreatedAt:  session.CreatedAt,
			UpdatedAt:  session.UpdatedAt,
			IsCurrent:  session.TokenHash == currentHash,
		})
	}
	return result, nil
}

func (s *Service) RevokeLoginSession(ctx context.Context, userID, sessionID uint, currentToken string) error {
	currentHash := tokenHash(strings.TrimSpace(currentToken))
	var session database.Session
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", sessionID, userID).
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return err
	}
	if session.TokenHash == currentHash {
		return ErrCurrentSession
	}
	return s.db.WithContext(ctx).Delete(&database.Session{}, session.ID).Error
}

func (s *Service) RevokeOtherLoginSessions(ctx context.Context, userID uint, currentToken string) error {
	currentHash := tokenHash(strings.TrimSpace(currentToken))
	return s.db.WithContext(ctx).
		Where("user_id = ? AND token_hash <> ?", userID, currentHash).
		Delete(&database.Session{}).Error
}

func applyLoginMetadata(session *database.Session, metadata LoginMetadata) {
	session.UserAgent = optionalString(metadata.UserAgent)
	session.RemoteAddr = optionalString(metadata.RemoteAddr)
	session.DeviceName = optionalString(metadata.DeviceName)
	session.ClientType = optionalString(metadata.ClientType)
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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

func isDuplicateUsernameError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") && strings.Contains(message, "users") && strings.Contains(message, "username")
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
