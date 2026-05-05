package listener

import (
	"context"
	"errors"
	"testing"

	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/storageindex"
)

func TestDiagnosticsReportsObserverStateAndFailures(t *testing.T) {
	t.Parallel()

	svc, _, record := newListenerIntegrationService(t)
	ctx := context.Background()
	if _, err := svc.index.UpsertPresent(ctx, storageindex.ObservationInput{LibraryID: record.ID, StorageProvider: "local", StoragePath: record.RootPath}); err != nil {
		t.Fatalf("upsert observation: %v", err)
	}
	if _, err := svc.index.RecordFailure(ctx, storageindex.FailureInput{LibraryID: record.ID, StorageProvider: "local", StoragePath: record.RootPath, Reason: "list_failed", Error: errors.New("permission denied")}); err != nil {
		t.Fatalf("record failure: %v", err)
	}
	if _, err := svc.library.QueueTargetedRefresh(ctx, record.ID, record.RootPath, library.WorkflowReasonTargetedRefresh); err != nil {
		t.Fatalf("create pending workflow: %v", err)
	}

	diagnostics, err := svc.Diagnostics(ctx)
	if err != nil {
		t.Fatalf("diagnostics: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", diagnostics)
	}
	diag := diagnostics[0]
	if diag.ObserverMode != "local_watcher_with_reconcile" || !diag.ObserverEnabled {
		t.Fatalf("unexpected observer state: %#v", diag)
	}
	if diag.LastSuccessfulObservation == nil || diag.LastReconcileAt == nil {
		t.Fatalf("expected observation timestamps, got %#v", diag)
	}
	if diag.PendingPlanCount != 1 {
		t.Fatalf("expected one pending plan, got %#v", diag)
	}
	if diag.RecentFailureSummary != "list_failed: permission denied" {
		t.Fatalf("unexpected failure summary: %#v", diag)
	}
}
