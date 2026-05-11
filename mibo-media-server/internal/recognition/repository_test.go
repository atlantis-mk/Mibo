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

func TestRepositoryLoadEnabledRulesAppliesPathScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repo := NewRepository(db)
	_, err = repo.UpsertRule(ctx, database.RecognitionRule{RuleKey: "scoped", LibraryID: 1, StorageProvider: "local", ScopePath: "/library/Movie", RuleType: "recognition_correction", CandidateType: CandidateTypeWork, Action: RuleActionAccept, Enabled: true, Priority: 10})
	if err != nil {
		t.Fatalf("upsert rule: %v", err)
	}
	matched, err := repo.LoadEnabledRules(ctx, 1, "local", "/library/Movie/File.mkv")
	if err != nil {
		t.Fatalf("load matched rules: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected scoped rule match, got %#v", matched)
	}
	unmatched, err := repo.LoadEnabledRules(ctx, 1, "local", "/library/Other/File.mkv")
	if err != nil {
		t.Fatalf("load unmatched rules: %v", err)
	}
	if len(unmatched) != 0 {
		t.Fatalf("expected no out-of-scope rules, got %#v", unmatched)
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

func TestRepositoryLoadLibraryGraphsAndDeleteLibraryManifests(t *testing.T) {
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
	graphs, err := repo.LoadLibraryGraphs(ctx, 1)
	if err != nil {
		t.Fatalf("load library graphs: %v", err)
	}
	if len(graphs) != 2 {
		t.Fatalf("expected two graphs, got %#v", graphs)
	}
	if err := repo.DeleteLibraryManifests(ctx, 1); err != nil {
		t.Fatalf("delete library manifests: %v", err)
	}
	graphs, err = repo.LoadLibraryGraphs(ctx, 1)
	if err != nil {
		t.Fatalf("load graphs after delete: %v", err)
	}
	if len(graphs) != 0 {
		t.Fatalf("expected no graphs after delete, got %#v", graphs)
	}
}
