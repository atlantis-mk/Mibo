package catalog

import (
	"github.com/atlan/mibo-media-server/internal/database"
	"testing"
)

func TestLegacyBackfillRun(t *testing.T) {
	t.Run("create run persists durable scope and operator fields", func(t *testing.T) {
		svc, ctx := newTestService(t)
		libraryID := uint(42)
		triggeredByUserID := uint(7)

		run, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{
			Kind:      LegacyBackfillScopeLibrary,
			LibraryID: &libraryID,
		}, triggeredByUserID)
		if err != nil {
			t.Fatalf("create run: %v", err)
		}

		if run.ScopeKind != LegacyBackfillScopeLibrary {
			t.Fatalf("expected scope %q, got %q", LegacyBackfillScopeLibrary, run.ScopeKind)
		}
		if run.LibraryID == nil || *run.LibraryID != libraryID {
			t.Fatalf("expected library id %d, got %#v", libraryID, run.LibraryID)
		}
		if run.Status != LegacyBackfillStatusQueued {
			t.Fatalf("expected queued status, got %q", run.Status)
		}
		if run.TriggeredByUserID != triggeredByUserID {
			t.Fatalf("expected triggered_by_user_id %d, got %d", triggeredByUserID, run.TriggeredByUserID)
		}
		if run.CreatedAt.IsZero() || run.UpdatedAt.IsZero() {
			t.Fatalf("expected created and updated timestamps, got %#v", run)
		}

		var persisted database.CatalogMigrationRun
		if err := svc.db.WithContext(ctx).First(&persisted, run.ID).Error; err != nil {
			t.Fatalf("reload run: %v", err)
		}
		if persisted.ScopeKind != run.ScopeKind || persisted.Status != run.Status || persisted.TriggeredByUserID != run.TriggeredByUserID {
			t.Fatalf("unexpected persisted run: %#v", persisted)
		}
	})

	t.Run("entries must use one allowed report classification", func(t *testing.T) {
		svc, ctx := newTestService(t)
		triggeredByUserID := uint(9)

		run, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{Kind: LegacyBackfillScopeAll}, triggeredByUserID)
		if err != nil {
			t.Fatalf("create run: %v", err)
		}

		legacyMediaItemID := uint(123)
		entry, err := svc.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{
			EntryType:         LegacyBackfillEntryTypeSuccess,
			LegacyMediaItemID: &legacyMediaItemID,
			Title:             "Movie A",
			Message:           "mapped",
		})
		if err != nil {
			t.Fatalf("record success entry: %v", err)
		}
		if entry.EntryType != LegacyBackfillEntryTypeSuccess {
			t.Fatalf("expected success entry type, got %q", entry.EntryType)
		}

		if _, err := svc.recordLegacyBackfillEntry(ctx, run.ID, LegacyBackfillEntry{EntryType: "unknown"}); err == nil {
			t.Fatalf("expected invalid entry type to fail")
		}
	})
}
