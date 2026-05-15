package recognition

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestRepositoryUpsertAndLoadManifestGraph(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	manifest, err := repo.UpsertManifest(ctx, ManifestScope{ManifestKey: "manifest:local:/library", LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test", Fingerprint: "fp"})
	if err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}
	if manifest.ID == 0 {
		t.Fatalf("expected manifest id")
	}
	candidate := database.RecognitionCandidate{ManifestID: manifest.ID, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, ReviewState: database.ReviewStatePending, CanonicalKey: "work:movie:movie:2024"}
	if err := repo.SaveCandidates(ctx, []database.RecognitionCandidate{candidate}); err != nil {
		t.Fatalf("save candidate: %v", err)
	}
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	if len(graph.Candidates) != 1 || graph.Candidates[0].CandidateKey != candidate.CandidateKey {
		t.Fatalf("expected candidate in graph, got %#v", graph.Candidates)
	}
}

func TestRepositoryUpsertManifestGeneratesKeyFromScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	first, err := repo.UpsertManifest(ctx, ManifestScope{LibraryID: 1, StorageProvider: "openlist", RootPath: "/library", ScopePath: "/library/A", ClassifierVersion: "test", Fingerprint: "fp-a"})
	if err != nil {
		t.Fatalf("upsert first manifest: %v", err)
	}
	second, err := repo.UpsertManifest(ctx, ManifestScope{LibraryID: 1, StorageProvider: "openlist", RootPath: "/library", ScopePath: "/library/B", ClassifierVersion: "test", Fingerprint: "fp-b"})
	if err != nil {
		t.Fatalf("upsert second manifest: %v", err)
	}
	if first.ManifestKey == "" || second.ManifestKey == "" || first.ManifestKey == second.ManifestKey || first.ID == second.ID {
		t.Fatalf("expected distinct generated manifest keys, first=%#v second=%#v", first, second)
	}
}

func TestRepositoryReplaceCandidatesAndEvidenceRemovesStaleRows(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	manifest, err := repo.UpsertManifest(ctx, ManifestScope{ManifestKey: "manifest:replace-candidates", LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test", Fingerprint: "fp"})
	if err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}
	fileID := uint(1)
	if err := repo.ReplaceCandidatesAndEvidence(ctx, manifest.ID,
		[]database.RecognitionCandidate{{ManifestID: manifest.ID, CandidateKey: "work:movie:old", CandidateType: CandidateTypeWork, ReviewState: database.ReviewStatePending}},
		[]database.RecognitionEvidence{{ManifestID: manifest.ID, InventoryFileID: &fileID, EvidenceKind: "kind", EvidenceSource: "source", EvidenceKey: "old", EvidenceValue: "old"}},
	); err != nil {
		t.Fatalf("save initial rows: %v", err)
	}
	if err := repo.ReplaceCandidatesAndEvidence(ctx, manifest.ID,
		[]database.RecognitionCandidate{{ManifestID: manifest.ID, CandidateKey: "work:movie:new", CandidateType: CandidateTypeWork, ReviewState: database.ReviewStatePending}},
		[]database.RecognitionEvidence{{ManifestID: manifest.ID, InventoryFileID: &fileID, EvidenceKind: "kind", EvidenceSource: "source", EvidenceKey: "new", EvidenceValue: "new"}},
	); err != nil {
		t.Fatalf("replace rows: %v", err)
	}
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	if len(graph.Candidates) != 1 || graph.Candidates[0].CandidateKey != "work:movie:new" {
		t.Fatalf("expected only replacement candidate, got %#v", graph.Candidates)
	}
	if len(graph.Evidence) != 1 || graph.Evidence[0].EvidenceKey != "new" {
		t.Fatalf("expected only replacement evidence, got %#v", graph.Evidence)
	}
}

func TestRepositorySupersedeManifest(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	manifest, err := repo.UpsertManifest(ctx, ManifestScope{ManifestKey: "manifest:local:/old", LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/old", ClassifierVersion: "test", Fingerprint: "fp"})
	if err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}
	if err := repo.SupersedeManifest(ctx, manifest.ID, manifest.ObservedAt); err != nil {
		t.Fatalf("supersede manifest: %v", err)
	}
	loaded, found, err := repo.LoadManifestByKey(ctx, manifest.ManifestKey)
	if err != nil || !found {
		t.Fatalf("load manifest found=%v err=%v", found, err)
	}
	if loaded.Status != "superseded" || loaded.SupersededAt == nil {
		t.Fatalf("expected superseded manifest, got %#v", loaded)
	}
}


func TestRepositoryReplaceEvidenceForInventoryFiles(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	manifest, err := repo.UpsertManifest(ctx, ManifestScope{ManifestKey: "manifest:replace-evidence", LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test", Fingerprint: "fp"})
	if err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}
	fileID := uint(7)
	initial := []database.RecognitionEvidence{
		{ManifestID: manifest.ID, InventoryFileID: &fileID, EvidenceKind: "file_signal", EvidenceSource: "inventory_file_signal", EvidenceKey: "title", EvidenceValue: "Old"},
		{ManifestID: manifest.ID, InventoryFileID: &fileID, EvidenceKind: "file_signal", EvidenceSource: "inventory_file_signal", EvidenceKey: "year", EvidenceValue: "2024"},
	}
	if err := repo.SaveEvidence(ctx, initial); err != nil {
		t.Fatalf("save evidence: %v", err)
	}
	replacement := []database.RecognitionEvidence{
		{ManifestID: manifest.ID, InventoryFileID: &fileID, EvidenceKind: "file_signal", EvidenceSource: "inventory_file_signal", EvidenceKey: "title", EvidenceValue: "New"},
	}
	if err := repo.ReplaceEvidenceForInventoryFiles(ctx, manifest.ID, []uint{fileID}, replacement); err != nil {
		t.Fatalf("replace evidence: %v", err)
	}
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	if len(graph.Evidence) != 1 || graph.Evidence[0].EvidenceValue != "New" {
		t.Fatalf("expected replacement evidence, got %#v", graph.Evidence)
	}
}

func TestRepositoryDeleteLibraryManifests(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	manifestA, err := repo.UpsertManifest(ctx, ManifestScope{ManifestKey: "manifest:library:1:a", LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library/A", ClassifierVersion: "test", Fingerprint: "fp-a"})
	if err != nil {
		t.Fatalf("upsert manifest A: %v", err)
	}
	manifestB, err := repo.UpsertManifest(ctx, ManifestScope{ManifestKey: "manifest:library:1:b", LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library/B", ClassifierVersion: "test", Fingerprint: "fp-b"})
	if err != nil {
		t.Fatalf("upsert manifest B: %v", err)
	}
	if err := repo.SaveCandidates(ctx, []database.RecognitionCandidate{
		{ManifestID: manifestA.ID, CandidateKey: "work:movie:a", CandidateType: CandidateTypeWork, ReviewState: database.ReviewStatePending},
		{ManifestID: manifestB.ID, CandidateKey: "work:movie:b", CandidateType: CandidateTypeWork, ReviewState: database.ReviewStatePending},
	}); err != nil {
		t.Fatalf("save candidates: %v", err)
	}
	if err := repo.DeleteLibraryManifests(ctx, 1); err != nil {
		t.Fatalf("delete library manifests: %v", err)
	}
}
