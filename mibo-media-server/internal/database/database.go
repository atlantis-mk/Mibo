package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		if err := ensureSQLiteDir(cfg.DSN); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&MediaSource{},
		&Library{},
		&MediaItem{},
		&MediaFile{},
		&TVSeasonMetadataCache{},
		&TVEpisodeMetadataCache{},
		&Job{},
		&Schedule{},
		&ScheduleRun{},
		&User{},
		&Session{},
		&PlaybackProgress{},
		&SystemSetting{},
		&SearchHistory{},
		&SearchDocument{},
	); err != nil {
		return nil, err
	}

	return db, nil
}

func ensureSQLiteDir(dsn string) error {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" || trimmed == ":memory:" || strings.HasPrefix(trimmed, "file:") {
		return nil
	}

	dir := filepath.Dir(trimmed)
	if dir == "." || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}
