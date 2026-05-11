package playback

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/providers"
	"gorm.io/gorm"
)

type playbackDecisionFixture struct {
	t        *testing.T
	db       *gorm.DB
	service  *Service
	library  database.Library
	rootPath string
}

func newPlaybackDecisionFixture(t *testing.T) *playbackDecisionFixture {
	t.Helper()
	rootPath := t.TempDir()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(rootPath, "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	cfg := config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}}
	registry := providers.NewRegistry(cfg)
	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: rootPath, RootPath: rootPath}
	if err := db.WithContext(context.Background()).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: rootPath, Status: "active", ScannerEnabled: true}
	if err := db.WithContext(context.Background()).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	return &playbackDecisionFixture{t: t, db: db, service: NewService(db, registry), library: libraryRecord, rootPath: rootPath}
}

func hasDecisionReasonCode(reasons []DecisionReason, want string) bool {
	for _, reason := range reasons {
		if reason.Code == want {
			return true
		}
	}
	return false
}
