package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestRegisterCreatesFirstUserAsAdmin(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "mibo.db"),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	svc := NewService(db)
	user, err := svc.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if user.Username != "alice" {
		t.Fatalf("username = %q, want %q", user.Username, "alice")
	}
	if user.Role != "admin" {
		t.Fatalf("role = %q, want %q", user.Role, "admin")
	}

	login, err := svc.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("login registered user: %v", err)
	}
	if login.User.ID != user.ID {
		t.Fatalf("login user id = %d, want %d", login.User.ID, user.ID)
	}
}

func TestLoginSessionsListAndRevoke(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "mibo.db"),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	svc := NewService(db)
	user, err := svc.Register(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	current, err := svc.Login(ctx, "alice", "password123", LoginMetadata{
		UserAgent:  "Mozilla/5.0 (Macintosh)",
		RemoteAddr: "127.0.0.1",
		DeviceName: "Mac",
		ClientType: "Mibo Web",
	})
	if err != nil {
		t.Fatalf("login current session: %v", err)
	}
	other, err := svc.Login(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("login other session: %v", err)
	}

	sessions, err := svc.ListLoginSessions(ctx, user.ID, current.Token)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	var currentSession, otherSession LoginSession
	for _, session := range sessions {
		if session.IsCurrent {
			currentSession = session
		} else {
			otherSession = session
		}
	}
	if currentSession.ID == 0 || otherSession.ID == 0 {
		t.Fatalf("expected one current and one other session, got %#v", sessions)
	}
	if currentSession.UserAgent == "" || currentSession.DeviceName != "Mac" || currentSession.ClientType != "Mibo Web" {
		t.Fatalf("metadata was not returned for current session: %#v", currentSession)
	}

	if err := svc.RevokeLoginSession(ctx, user.ID, currentSession.ID, current.Token); !errors.Is(err, ErrCurrentSession) {
		t.Fatalf("expected current session error, got %v", err)
	}
	if _, err := svc.Authenticate(ctx, current.Token); err != nil {
		t.Fatalf("current token should still authenticate: %v", err)
	}
	if err := svc.RevokeLoginSession(ctx, user.ID, otherSession.ID, current.Token); err != nil {
		t.Fatalf("revoke other session: %v", err)
	}
	if _, err := svc.Authenticate(ctx, other.Token); err == nil {
		t.Fatal("expected revoked token to fail authentication")
	}
}

func TestRevokeLoginSessionBlocksOtherUsers(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "mibo.db"),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx := context.Background()
	svc := NewService(db)
	alice, err := svc.Register(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("register alice: %v", err)
	}
	if _, err := svc.Register(ctx, "bob", "password123"); err != nil {
		t.Fatalf("register bob: %v", err)
	}
	aliceLogin, err := svc.Login(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("login alice: %v", err)
	}
	bobLogin, err := svc.Login(ctx, "bob", "password123")
	if err != nil {
		t.Fatalf("login bob: %v", err)
	}

	var bobSession database.Session
	if err := db.Where("token_hash = ?", tokenHash(bobLogin.Token)).First(&bobSession).Error; err != nil {
		t.Fatalf("find bob session: %v", err)
	}
	if err := svc.RevokeLoginSession(ctx, alice.ID, bobSession.ID, aliceLogin.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected session not found, got %v", err)
	}
	if _, err := svc.Authenticate(ctx, bobLogin.Token); err != nil {
		t.Fatalf("bob token should still authenticate: %v", err)
	}
}

func TestRegisterCreatesAdditionalUsersAsRegularUsers(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "mibo.db"),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	svc := NewService(db)
	if _, err := svc.Register(context.Background(), "admin", "password123"); err != nil {
		t.Fatalf("register first user: %v", err)
	}
	user, err := svc.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("register second user: %v", err)
	}
	if user.Role != "user" {
		t.Fatalf("role = %q, want %q", user.Role, "user")
	}
}

func TestLoginRejectsUnknownUser(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "mibo.db"),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	svc := NewService(db)
	if _, err := svc.Login(context.Background(), "admin", "admin123"); err == nil {
		t.Fatal("expected login to fail when no users exist")
	}
}
