package auth

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestRegisterCreatesRegularUser(t *testing.T) {
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
	if user.Role != "user" {
		t.Fatalf("role = %q, want %q", user.Role, "user")
	}

	login, err := svc.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("login registered user: %v", err)
	}
	if login.User.ID != user.ID {
		t.Fatalf("login user id = %d, want %d", login.User.ID, user.ID)
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
