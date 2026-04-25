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

func TestLegacyBackfillReportQueries(t *testing.T) {
	svc, ctx := newTestService(t)

	runs, err := svc.ListLegacyBackfillRuns(ctx)
	if err != nil {
		t.Fatalf("list empty runs: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected empty run list, got %#v", runs)
	}

	olderLibraryID := uint(3)
	olderRun, err := svc.CreateLegacyBackfillRun(ctx, CreateLegacyBackfillRunInput{
		Scope:             LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &olderLibraryID},
		TriggeredByUserID: 41,
	})
	if err != nil {
		t.Fatalf("create older run: %v", err)
	}
	if _, err := svc.recordLegacyBackfillEntry(ctx, olderRun.ID, LegacyBackfillEntry{EntryType: LegacyBackfillEntryTypeSuccess, LibraryID: &olderLibraryID, Message: "older success"}); err != nil {
		t.Fatalf("record older entry: %v", err)
	}
	if _, err := svc.finalizeLegacyBackfillRun(ctx, olderRun.ID, LegacyBackfillStatusCompleted, ""); err != nil {
		t.Fatalf("finalize older run: %v", err)
	}

	libraryID := uint(7)
	run, err := svc.CreateLegacyBackfillRun(ctx, CreateLegacyBackfillRunInput{
		Scope:             LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &libraryID},
		TriggeredByUserID: 42,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	legacyItemTwenty := uint(20)
	legacyItemTen := uint(10)
	legacyItemNine := uint(9)
	legacyItemThirty := uint(30)
	legacyFileTwo := uint(2)
	legacyFileNinetyNine := uint(99)

	entries := []LegacyBackfillEntry{
		{EntryType: LegacyBackfillEntryTypeSuccess, LibraryID: &libraryID, LegacyMediaItemID: &legacyItemTwenty, Message: "success later"},
		{EntryType: LegacyBackfillEntryTypeConflict, LibraryID: &libraryID, LegacyMediaItemID: &legacyItemNine, Message: "conflict"},
		{EntryType: LegacyBackfillEntryTypeOrphanFile, LibraryID: &libraryID, LegacyMediaFileID: &legacyFileNinetyNine, Message: "orphan"},
		{EntryType: LegacyBackfillEntryTypeSkipped, LibraryID: &libraryID, LegacyMediaItemID: &legacyItemThirty, Message: "skipped"},
		{EntryType: LegacyBackfillEntryTypeSuccess, LibraryID: &libraryID, LegacyMediaItemID: &legacyItemTen, Message: "success earlier"},
		{EntryType: LegacyBackfillEntryTypeDuplicateEpisodeCandidate, LibraryID: &libraryID, LegacyMediaItemID: &legacyItemTwenty, LegacyMediaFileID: &legacyFileTwo, Message: "duplicate"},
	}
	for _, entry := range entries {
		if _, err := svc.recordLegacyBackfillEntry(ctx, run.ID, entry); err != nil {
			t.Fatalf("record entry %q: %v", entry.EntryType, err)
		}
	}
	if _, err := svc.finalizeLegacyBackfillRun(ctx, run.ID, LegacyBackfillStatusCompleted, ""); err != nil {
		t.Fatalf("finalize run: %v", err)
	}

	runs, err = svc.ListLegacyBackfillRuns(ctx)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %#v", runs)
	}
	if runs[0].ID != run.ID || runs[1].ID != olderRun.ID {
		t.Fatalf("expected newest-first runs, got %#v", runs)
	}
	if runs[0].SuccessCount != 2 || runs[0].SkippedCount != 1 || runs[0].ConflictCount != 1 || runs[0].OrphanFileCount != 1 || runs[0].DuplicateEpisodeCandidateCount != 1 {
		t.Fatalf("unexpected aggregated counts for latest run: %#v", runs[0])
	}
	if runs[1].SuccessCount != 1 {
		t.Fatalf("expected older run success count to persist, got %#v", runs[1])
	}

	report, err := svc.GetLegacyBackfillRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if len(report.Entries) != 6 {
		t.Fatalf("expected 6 report entries, got %#v", report.Entries)
	}

	orderedTypes := []string{
		LegacyBackfillEntryTypeConflict,
		LegacyBackfillEntryTypeDuplicateEpisodeCandidate,
		LegacyBackfillEntryTypeOrphanFile,
		LegacyBackfillEntryTypeSkipped,
		LegacyBackfillEntryTypeSuccess,
		LegacyBackfillEntryTypeSuccess,
	}
	orderedLegacyItemIDs := []uint{9, 20, 0, 30, 10, 20}
	for index, entry := range report.Entries {
		if entry.EntryType != orderedTypes[index] {
			t.Fatalf("entry %d type mismatch: want %q got %q", index, orderedTypes[index], entry.EntryType)
		}
		var gotLegacyItemID uint
		if entry.LegacyMediaItemID != nil {
			gotLegacyItemID = *entry.LegacyMediaItemID
		}
		if gotLegacyItemID != orderedLegacyItemIDs[index] {
			t.Fatalf("entry %d legacy item mismatch: want %d got %d", index, orderedLegacyItemIDs[index], gotLegacyItemID)
		}
	}
}
