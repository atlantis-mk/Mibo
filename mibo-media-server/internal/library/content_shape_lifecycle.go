package library

import (
	"context"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func markContentShapeScopeDeleted(ctx context.Context, tx *gorm.DB, now time.Time, where string, args ...any) error {
	updates := map[string]any{"deleted_scope": true, "invalidated_at": now}
	for _, model := range []any{&database.ContentShapeAssignment{}, &database.ContentShapePlan{}, &database.ContentShapeProfile{}} {
		if err := tx.WithContext(ctx).Model(model).Where(where, args...).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}
