package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestNewDoesNotSeedUsers(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    filepath.Join(t.TempDir(), "mibo.db"),
		},
		Storage: config.StorageConfig{Provider: "local"},
		Local:   config.LocalStorageConfig{RootPath: t.TempDir()},
	}

	app, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	sqlDB, err := app.db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	defer sqlDB.Close()

	var userCount int64
	if err := app.db.WithContext(context.Background()).Model(&database.User{}).Count(&userCount).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("user count = %d, want 0", userCount)
	}
}
