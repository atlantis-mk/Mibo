package metadata

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestApplyMetadataFieldChangesReportsLockedSkips(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	catalogSvc := catalog.NewService(db)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Original", Path: "/movies/original.mkv", SortKey: "Original"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	if _, applied, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "title", Value: "Locked", Lock: true, LockReason: "manual"}); err != nil || !applied {
		t.Fatalf("lock title: applied=%v err=%v", applied, err)
	}
	svc := NewService(db, config.MetadataConfig{}, nil)
	applied, skipped, err := svc.applyMetadataFieldChanges(ctx, []MetadataFieldChange{
		{ItemID: item.ID, FieldKey: "title", Value: "Provider Title", ApplyMode: FieldApplyModeAutomated},
		{ItemID: item.ID, FieldKey: "overview", Value: "Provider Overview", ApplyMode: FieldApplyModeAutomated},
	})
	if err != nil {
		t.Fatalf("apply metadata field changes: %v", err)
	}
	if len(applied) != 1 || applied[0].FieldKey != "overview" {
		t.Fatalf("unexpected applied fields: %#v", applied)
	}
	if len(skipped) != 1 || skipped[0].FieldKey != "title" || skipped[0].Reason != "locked" {
		t.Fatalf("unexpected skipped fields: %#v", skipped)
	}
	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if stored.Title != "Locked" || stored.Overview != "Provider Overview" {
		t.Fatalf("unexpected stored item after field policy: %#v", stored)
	}
}
