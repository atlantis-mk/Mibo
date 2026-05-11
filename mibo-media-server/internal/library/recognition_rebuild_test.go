package library

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestResetRecognitionLibraryStateClearsLibraryRecognitionAndProjectionState(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)

	manifest := database.RecognitionManifest{ManifestKey: "manifest:library:7", LibraryID: 7, StorageProvider: "local", RootPath: "/library", ScopePath: "/library/Show", ClassifierVersion: "test", Fingerprint: "fp", Status: "pending"}
	if err := db.WithContext(ctx).Create(&manifest).Error; err != nil {
		t.Fatalf("create manifest: %v", err)
	}
	candidate := database.RecognitionCandidate{ManifestID: manifest.ID, CandidateKey: "work:series:show", CandidateType: "work", CandidateRole: "series", ReviewState: database.ReviewStatePending}
	if err := db.WithContext(ctx).Create(&candidate).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}
	evidence := database.RecognitionEvidence{ManifestID: manifest.ID, EvidenceKind: "directory_context", EvidenceSource: "content_shape", EvidenceKey: "series_title", EvidenceValue: "Show"}
	if err := db.WithContext(ctx).Create(&evidence).Error; err != nil {
		t.Fatalf("create evidence: %v", err)
	}
	decision := database.RecognitionDecision{ManifestID: manifest.ID, CandidateID: &candidate.ID, DecisionType: "resolver_outcome", Outcome: "accepted", TargetKind: "work", TargetKey: candidate.CandidateKey}
	if err := db.WithContext(ctx).Create(&decision).Error; err != nil {
		t.Fatalf("create decision: %v", err)
	}

	series := database.MetadataItem{ItemType: database.MetadataItemTypeSeries, ContentForm: database.MetadataContentFormStandard, Title: "Show", SortTitle: "Show", SortKey: "work:series:show", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&series).Error; err != nil {
		t.Fatalf("create series: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:show", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 7, Status: "available"}).Error; err != nil {
		t.Fatalf("create resource library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: 1, Role: database.ResourceFileRoleSource}).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: series.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create resource metadata link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: series.ID, ItemType: series.ItemType, Title: series.Title}).Error; err != nil {
		t.Fatalf("create projection: %v", err)
	}

	if err := svc.ResetRecognitionLibraryState(ctx, 7); err != nil {
		t.Fatalf("reset library recognition state: %v", err)
	}

	assertZeroCount := func(model any, where string, args ...any) {
		t.Helper()
		var count int64
		if err := db.WithContext(ctx).Model(model).Where(where, args...).Count(&count).Error; err != nil {
			t.Fatalf("count %T: %v", model, err)
		}
		if count != 0 {
			t.Fatalf("expected zero rows for %T, got %d", model, count)
		}
	}

	assertZeroCount(&database.RecognitionManifest{}, "library_id = ?", 7)
	assertZeroCount(&database.RecognitionCandidate{}, "manifest_id = ?", manifest.ID)
	assertZeroCount(&database.RecognitionEvidence{}, "manifest_id = ?", manifest.ID)
	assertZeroCount(&database.RecognitionDecision{}, "manifest_id = ?", manifest.ID)
	assertZeroCount(&database.LibraryMetadataProjection{}, "library_id = ?", 7)
	assertZeroCount(&database.ResourceLibraryLink{}, "library_id = ?", 7)
	assertZeroCount(&database.ResourceMetadataLink{}, "resource_id = ?", resource.ID)
	assertZeroCount(&database.ResourceFile{}, "resource_id = ?", resource.ID)
	assertZeroCount(&database.Resource{}, "id = ?", resource.ID)
	assertZeroCount(&database.MetadataItem{}, "id = ?", series.ID)
}

func TestResetRecognitionLibraryStateKeepsOtherLibraryData(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)

	otherManifest := database.RecognitionManifest{ManifestKey: "manifest:library:8", LibraryID: 8, StorageProvider: "local", RootPath: "/other", ScopePath: "/other", ClassifierVersion: "test", Fingerprint: "fp", Status: "pending"}
	if err := db.WithContext(ctx).Create(&otherManifest).Error; err != nil {
		t.Fatalf("create other manifest: %v", err)
	}
	otherItem := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Other", SortTitle: "Other", SortKey: "work:movie:other", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&otherItem).Error; err != nil {
		t.Fatalf("create other item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 8, MetadataItemID: otherItem.ID, ItemType: otherItem.ItemType, Title: otherItem.Title}).Error; err != nil {
		t.Fatalf("create other projection: %v", err)
	}

	if err := svc.ResetRecognitionLibraryState(ctx, 7); err != nil {
		t.Fatalf("reset library recognition state: %v", err)
	}

	var manifestCount int64
	if err := db.WithContext(ctx).Model(&database.RecognitionManifest{}).Where("library_id = ?", 8).Count(&manifestCount).Error; err != nil {
		t.Fatalf("count other manifests: %v", err)
	}
	if manifestCount != 1 {
		t.Fatalf("expected other library manifest preserved, got %d", manifestCount)
	}
	var projectionCount int64
	if err := db.WithContext(ctx).Model(&database.LibraryMetadataProjection{}).Where("library_id = ?", 8).Count(&projectionCount).Error; err != nil {
		t.Fatalf("count other projections: %v", err)
	}
	if projectionCount != 1 {
		t.Fatalf("expected other library projection preserved, got %d", projectionCount)
	}
}
