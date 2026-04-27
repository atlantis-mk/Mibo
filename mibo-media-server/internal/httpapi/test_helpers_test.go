package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func createAuthHeader(t *testing.T, ctx context.Context, authSvc *auth.Service) string {
	t.Helper()
	username := "test-user"
	password := "test-password-123"
	if _, err := authSvc.Register(ctx, username, password); err != nil {
		t.Fatalf("register test user: %v", err)
	}
	login, err := authSvc.Login(ctx, username, password)
	if err != nil {
		t.Fatalf("login test user: %v", err)
	}
	return "Bearer " + login.Token
}

func createAdminAuthHeader(t *testing.T, ctx context.Context, db *gorm.DB, authSvc *auth.Service) string {
	t.Helper()
	username := "admin-user"
	password := "test-password-123"
	user, err := authSvc.Register(ctx, username, password)
	if err != nil {
		t.Fatalf("register admin user: %v", err)
	}
	if err := db.WithContext(ctx).Model(&database.User{}).Where("id = ?", user.ID).Update("role", "admin").Error; err != nil {
		t.Fatalf("promote admin user: %v", err)
	}
	login, err := authSvc.Login(ctx, username, password)
	if err != nil {
		t.Fatalf("login admin user: %v", err)
	}
	return "Bearer " + login.Token
}

func timePtr(value time.Time) *time.Time {
	return &value
}
