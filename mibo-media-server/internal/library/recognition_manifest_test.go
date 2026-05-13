package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/recognition"
)

func TestApplyRecognitionFallbackPosterWritesSelectedPosterWhenMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", SortTitle: "Movie", SortKey: "movie", GovernanceStatus: database.ReviewStatePending}
	resource := database.Resource{StableResourceKey: "resource:1", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "openlist", StoragePath: "/library/Movie.mkv", ThumbnailURL: "https://image/thumb.jpg"}
	for _, row := range []any{&item, &resource, &file} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}
	for _, row := range []any{
		&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource},
		&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary},
	} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed link: %v", err)
		}
	}
	if err := svc.applyRecognitionFallbackPoster(ctx, []database.InventoryFile{file}, []uint{item.ID}); err != nil {
		t.Fatalf("apply fallback poster: %v", err)
	}
	var images []database.MetadataItemImage
	if err := db.WithContext(ctx).Where("metadata_item_id = ?", item.ID).Find(&images).Error; err != nil {
		t.Fatalf("load images: %v", err)
	}
	if len(images) != 1 || images[0].ImageType != "poster" || images[0].URL != "https://image/thumb.jpg" || !images[0].IsSelected {
		t.Fatalf("unexpected fallback poster rows: %#v", images)
	}
}

func TestPersistRecognitionManifestForFilesUsesDirectoryReductionScopePath(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	season := 1
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Show/Season 1/Show.S01E01.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Show/Season 1/Show.S01E02.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
		model := extractFilenameSignalModel(file.StoragePath)
		model.Identity.TitleCandidate = "Show"
		model.Identity.SeasonNumber = &season
		model.Identity.EpisodeNumber = intPtrForReduction(map[uint]int{1: 1, 2: 2}[file.ID])
		if err := saveInventoryFileSignals(ctx, db, inventoryFileSignalScope{LibraryID: libraryRecord.ID, StorageProvider: "local", ClassifierVersion: contentShapeSettingsFromConfig(config.Config{}).ClassifierVersion}, []inventoryFileSignalInput{{File: file, Model: model}}); err != nil {
			t.Fatalf("create signal: %v", err)
		}
	}
	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library/Show/Season 1")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	if manifest.ScopePath != "/library/Show/Season 1" {
		t.Fatalf("expected series directory scope path preserved for common parent, got %#v", manifest)
	}
}

func TestPersistRecognitionManifestForFilesPersistsKernelCandidatesAndEvidence(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	library := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{ID: 1, LibraryID: library.ID, StorageProvider: "local", StoragePath: "/library/Movie A (2024)/Movie A.2024.mkv", StableIdentityKey: "local:movie-a", ContentClass: SourceContentClassVideo, Status: "available"}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}

	manifest, err := svc.persistRecognitionManifestForFiles(ctx, library, []database.InventoryFile{file}, library.RootPath)
	if err != nil {
		t.Fatalf("persist recognition manifest: %v", err)
	}
	var candidates []database.RecognitionCandidate
	if err := db.Where("manifest_id = ?", manifest.ID).Find(&candidates).Error; err != nil {
		t.Fatalf("load candidates: %v", err)
	}
	var evidence []database.RecognitionEvidence
	if err := db.Where("manifest_id = ?", manifest.ID).Find(&evidence).Error; err != nil {
		t.Fatalf("load evidence: %v", err)
	}
	if len(candidates) == 0 || len(evidence) == 0 {
		t.Fatalf("expected kernel candidates and evidence, candidates=%#v evidence=%#v", candidates, evidence)
	}
}

func TestPersistRecognitionManifestForFilesPromotesScopePathForMovieCollection(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	yearA := 2024
	yearB := 2025
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Collection/A/A.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Collection/B/B.2025.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
		model := extractFilenameSignalModel(file.StoragePath)
		if file.ID == 1 {
			model.Identity.TitleCandidate = "A"
			model.Identity.Year = &yearA
		} else {
			model.Identity.TitleCandidate = "B"
			model.Identity.Year = &yearB
		}
		if err := saveInventoryFileSignals(ctx, db, inventoryFileSignalScope{LibraryID: libraryRecord.ID, StorageProvider: "local", ClassifierVersion: contentShapeSettingsFromConfig(config.Config{}).ClassifierVersion}, []inventoryFileSignalInput{{File: file, Model: model}}); err != nil {
			t.Fatalf("create signal: %v", err)
		}
	}
	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library/Collection/A")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	if manifest.ScopePath != "/library/Collection" {
		t.Fatalf("expected collection scope to promote to shared parent, got %#v", manifest)
	}
}

func TestRecognitionManifestSeasonFolderLeadingNumericBuildsSeriesSeasonEpisodeInsteadOfMovieCollection(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Show/Season 1/01.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Show/Season 1/02.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", files); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}

	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library/Show/Season 1")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}

	repo := recognition.NewRepository(db)
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}

	seriesKey := recognition.SeriesWorkKey("Show")
	seasonKey := recognition.SeasonWorkKey("Show", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 2})

	assertCandidate := func(candidateKey string, candidateType string, candidateRole string) {
		t.Helper()
		for _, candidate := range graph.Candidates {
			if candidate.CandidateKey == candidateKey && candidate.CandidateType == candidateType && candidate.CandidateRole == candidateRole {
				return
			}
		}
		t.Fatalf("expected candidate %s type=%s role=%s in %#v", candidateKey, candidateType, candidateRole, graph.Candidates)
	}
	assertCandidate(seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertCandidate(seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertCandidate(episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertCandidate(episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)

	for _, candidate := range graph.Candidates {
		if candidate.CandidateType == recognition.CandidateTypeWork && candidate.CandidateRole == recognition.WorkKindMovie {
			t.Fatalf("did not expect movie work candidate for season folder leading numeric episodes, got %#v", graph.Candidates)
		}
	}

	seenSeriesTitleEvidence := false
	seenSeasonNumberEvidence := false
	for _, evidence := range graph.Evidence {
		if evidence.EvidenceSource != "content_shape" {
			continue
		}
		if evidence.EvidenceKey == "series_title" && evidence.EvidenceValue == "Show" {
			seenSeriesTitleEvidence = true
		}
		if evidence.EvidenceKey == "season_number" && evidence.EvidenceValue == "1" {
			seenSeasonNumberEvidence = true
		}
	}
	if !seenSeriesTitleEvidence || !seenSeasonNumberEvidence {
		t.Fatalf("expected content shape evidence for series title and season number, got %#v", graph.Evidence)
	}
}

func TestRecognitionManifestSeasonFolderLeadingNumericResourcesPointToEpisodes(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StableIdentityKey: "ep-1", StoragePath: "/library/Show/Season 1/01.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StableIdentityKey: "ep-2", StoragePath: "/library/Show/Season 1/02.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", files); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}
	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library/Show/Season 1")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	result, err := svc.resolveRecognitionManifest(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("resolve manifest: %v", err)
	}
	if len(result.ResourceIDs) != 2 {
		t.Fatalf("expected two materialized resources, got %#v", result)
	}
	var episodeItems []database.MetadataItem
	if err := db.WithContext(ctx).Where("item_type = ?", database.MetadataItemTypeEpisode).Order("index_number asc").Find(&episodeItems).Error; err != nil {
		t.Fatalf("load episode items: %v", err)
	}
	if len(episodeItems) != 2 {
		t.Fatalf("expected two episode items, got %#v", episodeItems)
	}
	var links []database.ResourceMetadataLink
	if err := db.WithContext(ctx).Order("resource_id asc, metadata_item_id asc").Find(&links).Error; err != nil {
		t.Fatalf("load resource links: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected one primary link per episode resource, got %#v", links)
	}
	episodeIDs := map[uint]struct{}{episodeItems[0].ID: {}, episodeItems[1].ID: {}}
	for _, link := range links {
		if _, ok := episodeIDs[link.MetadataItemID]; !ok {
			t.Fatalf("expected resource link to episode metadata, got %#v with episodes %#v", links, episodeItems)
		}
	}
}

func TestPersistRecognitionManifestForFilesReadsSidecarHints(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	showDir := filepath.Join(root, "Show", "Season 1")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("mkdir show dir: %v", err)
	}
	videoPath := filepath.Join(showDir, "01.mkv")
	sidecarPath := filepath.Join(showDir, "01.nfo")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	if err := os.WriteFile(sidecarPath, []byte("series_title: Show\ntitle: Episode 1\nseason_number: 1\nepisode_number: 1\ntmdb: 123\n"), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	cfg := config.Config{}
	cfg.Local.RootPath = root
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(cfg, db, providers.NewRegistry(cfg), nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: root}
	source := database.MediaSource{ID: 1, Name: "Local", Provider: "local", StorageRef: root, RootPath: root}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StableIdentityKey: "ep-1", StoragePath: videoPath, Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", []database.InventoryFile{file}); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}
	provider, err := svc.storageRegistry().BuildForSource(source)
	if err != nil {
		t.Fatalf("build provider: %v", err)
	}
	if err := svc.ensureInventorySidecarSignals(ctx, provider, libraryRecord, []database.InventoryFile{file}); err != nil {
		t.Fatalf("ensure sidecar signals: %v", err)
	}
	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, []database.InventoryFile{file}, showDir)
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	repo := recognition.NewRepository(db)
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	seenTMDB := false
	seenSeriesTitle := false
	for _, evidence := range graph.Evidence {
		if evidence.EvidenceKey == "external_id:tmdb" && evidence.EvidenceValue == "123" {
			seenTMDB = true
		}
		if evidence.EvidenceKey == "series_title" && evidence.EvidenceValue == "Show" {
			seenSeriesTitle = true
		}
	}
	if !seenTMDB || !seenSeriesTitle {
		t.Fatalf("expected sidecar hint evidence in graph, got %#v", graph.Evidence)
	}
}

func TestEnsureInventorySidecarSignalsPersistsParsedSidecars(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	showDir := filepath.Join(root, "Show", "Season 1")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("mkdir show dir: %v", err)
	}
	videoPath := filepath.Join(showDir, "01.mkv")
	sidecarPath := filepath.Join(showDir, "01.nfo")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	if err := os.WriteFile(sidecarPath, []byte("series_title: Show\ntitle: Episode 1\nseason_number: 1\nepisode_number: 1\ntmdb: 123\n"), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	cfg := config.Config{}
	cfg.Local.RootPath = root
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(cfg, db, providers.NewRegistry(cfg), nil)
	source := database.MediaSource{ID: 1, Name: "Local", Provider: "local", StorageRef: root, RootPath: root}
	libraryRecord := database.Library{ID: 1, MediaSourceID: source.ID, RootPath: root}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StableIdentityKey: "ep-1", StoragePath: videoPath, Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	provider, err := svc.storageRegistry().BuildForSource(source)
	if err != nil {
		t.Fatalf("build provider: %v", err)
	}
	if err := svc.ensureInventorySidecarSignals(ctx, provider, libraryRecord, []database.InventoryFile{file}); err != nil {
		t.Fatalf("ensure sidecar signals: %v", err)
	}
	var rows []database.InventorySidecarSignal
	if err := db.WithContext(ctx).Where("inventory_file_id = ?", file.ID).Find(&rows).Error; err != nil {
		t.Fatalf("load sidecar signals: %v", err)
	}
	if len(rows) != 1 || rows[0].SidecarPath != sidecarPath || rows[0].SeriesTitle != "Show" || rows[0].ParseStatus != "parsed" {
		t.Fatalf("unexpected sidecar signals: %#v", rows)
	}
}

func TestPersistRecognitionManifestForFilesPersistsMediaGraphRows(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StableIdentityKey: "ep-1", StoragePath: "/library/Show/Season 1/01.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StableIdentityKey: "ep-2", StoragePath: "/library/Show/Season 1/02.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", files); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}
	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library/Show/Season 1")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	var nodes []database.MediaGraphNode
	if err := db.WithContext(ctx).Where("manifest_id = ?", manifest.ID).Find(&nodes).Error; err != nil {
		t.Fatalf("load graph nodes: %v", err)
	}
	var classifications []database.MediaGraphClassification
	if err := db.WithContext(ctx).Where("manifest_id = ?", manifest.ID).Find(&classifications).Error; err != nil {
		t.Fatalf("load graph classifications: %v", err)
	}
	if len(nodes) == 0 || len(classifications) == 0 {
		t.Fatalf("expected persisted media graph rows, got nodes=%#v classifications=%#v", nodes, classifications)
	}
}

func TestPersistRecognitionManifestPersistsSidecarAttachmentGraphRows(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	video := database.InventoryFile{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StableIdentityKey: "movie", StoragePath: "/library/Movie/Movie.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"}
	sidecar := database.InventoryFile{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StableIdentityKey: "poster", StoragePath: "/library/Movie/poster.jpg", Container: "jpg", ContentClass: SourceContentClassImage, Status: "available"}
	for _, file := range []database.InventoryFile{video, sidecar} {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", []database.InventoryFile{video}); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}
	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, []database.InventoryFile{video}, "/library/Movie")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	if err := saveInventorySidecarSignals(ctx, db, libraryRecord.ID, "local", []inventorySidecarSignalInput{{File: video, SidecarPath: sidecar.StoragePath, Extension: ".jpg", AssociationSource: "basename", ParseStatus: "parsed"}}); err != nil {
		t.Fatalf("save sidecar signal: %v", err)
	}
	manifest, err = svc.persistRecognitionManifestForFiles(ctx, libraryRecord, []database.InventoryFile{video}, "/library/Movie")
	if err != nil {
		t.Fatalf("persist manifest with sidecar: %v", err)
	}
	var nodes []database.MediaGraphNode
	if err := db.WithContext(ctx).Where("manifest_id = ? AND node_kind = ?", manifest.ID, "attachment").Find(&nodes).Error; err != nil {
		t.Fatalf("load attachment nodes: %v", err)
	}
	foundPoster := false
	for _, node := range nodes {
		if strings.Contains(node.PayloadJSON, `"role":"poster"`) && node.InventoryFileID != nil && *node.InventoryFileID == sidecar.ID {
			foundPoster = true
		}
	}
	if !foundPoster {
		t.Fatalf("expected poster attachment graph node with inventory file id %d, got %#v", sidecar.ID, nodes)
	}
	var playableCount int64
	if err := db.WithContext(ctx).Model(&database.RecognitionCandidate{}).Where("manifest_id = ? AND candidate_type = ? AND primary_inventory_id = ?", manifest.ID, recognition.CandidateTypePlayableResource, sidecar.ID).Count(&playableCount).Error; err != nil {
		t.Fatalf("count sidecar playable candidates: %v", err)
	}
	if playableCount != 0 {
		t.Fatalf("did not expect sidecar attachment to materialize as playable resource")
	}
}

func TestRecognitionManifestSingleMovieFolderBuildsMovieAndResource(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Movie (2024)/Movie.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", []database.InventoryFile{file}); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}

	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, []database.InventoryFile{file}, "/library/Movie (2024)")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	repo := recognition.NewRepository(db)
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}

	movieKey := recognition.MovieWorkKey(recognition.MovieWorkInput{Title: "Movie", Year: intPtrForReduction(2024)})
	foundMovie := false
	foundResource := false
	for _, candidate := range graph.Candidates {
		if candidate.CandidateKey == movieKey && candidate.CandidateType == recognition.CandidateTypeWork && candidate.CandidateRole == recognition.WorkKindMovie {
			foundMovie = true
		}
		if candidate.CandidateType == recognition.CandidateTypePlayableResource && candidate.ParentCandidateKey == movieKey {
			foundResource = true
		}
	}
	if !foundMovie || !foundResource {
		t.Fatalf("expected single movie folder to build movie/resource candidates, got %#v", graph.Candidates)
	}
}

func TestRecognitionManifestMovieVersionFolderBuildsOneMovieWithVariants(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Movie/Movie.2024.1080p.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Movie/Movie.2024.2160p.Directors.Cut.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", files); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}

	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library/Movie")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	repo := recognition.NewRepository(db)
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}

	movieCandidates := 0
	variantCandidates := 0
	editionCandidates := 0
	for _, candidate := range graph.Candidates {
		switch candidate.CandidateType {
		case recognition.CandidateTypeWork:
			if candidate.CandidateRole == recognition.WorkKindMovie {
				movieCandidates++
			}
		case recognition.CandidateTypeVariant:
			variantCandidates++
		case recognition.CandidateTypeEdition:
			editionCandidates++
		}
	}
	if movieCandidates != 1 || variantCandidates < 2 || editionCandidates < 1 {
		t.Fatalf("expected one movie plus grouped variants/edition, got movie=%d variant=%d edition=%d candidates=%#v", movieCandidates, variantCandidates, editionCandidates, graph.Candidates)
	}
}

func TestRecognitionManifestIndependentMovieCollectionBuildsSeparateMovies(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Collection/A/A.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Collection/B/B.2025.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", files); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}

	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library/Collection")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	repo := recognition.NewRepository(db)
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}

	movieKeys := map[string]struct{}{}
	for _, candidate := range graph.Candidates {
		if candidate.CandidateType == recognition.CandidateTypeWork && candidate.CandidateRole == recognition.WorkKindMovie {
			movieKeys[candidate.CandidateKey] = struct{}{}
		}
	}
	if len(movieKeys) != 2 {
		t.Fatalf("expected two independent movie work candidates, got %#v", graph.Candidates)
	}
}

func TestRecognitionManifestTrailerAndSampleBecomeSupplementals(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Movie/Movie.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Movie/trailer.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 3, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Movie/sample.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", files); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}

	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library/Movie")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	repo := recognition.NewRepository(db)
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}

	roles := map[string]bool{"trailer": false, "sample": false}
	for _, candidate := range graph.Candidates {
		if candidate.CandidateType == recognition.CandidateTypeSupplemental {
			roles[candidate.CandidateRole] = true
		}
	}
	if !roles["trailer"] || !roles["sample"] {
		t.Fatalf("expected trailer and sample supplemental candidates, got %#v", graph.Candidates)
	}
}

func TestRecognitionManifestExplicitEpisodePatternsBuildSeriesHierarchy(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Show/Season 1/Show.S01E02.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/灵笼/第一季/第03集.mp4", Container: "mp4", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 3, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/灵笼/第一季/第04集.mp4", Container: "mp4", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", files); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}

	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, []database.InventoryFile{files[0]}, "/library/Show/Season 1")
	if err != nil {
		t.Fatalf("persist show manifest: %v", err)
	}
	repo := recognition.NewRepository(db)
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load show graph: %v", err)
	}
	wantEpisodeKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 2})
	found := false
	for _, candidate := range graph.Candidates {
		if candidate.CandidateKey == wantEpisodeKey && candidate.CandidateType == recognition.CandidateTypeEpisode {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected explicit S01E02 episode candidate, got %#v", graph.Candidates)
	}

	_ = files[1]
	_ = files[2]
}

func TestRecognitionResolveMarksAmbiguousInventoryFileReviewRequired(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{
		ID:              1,
		LibraryID:       libraryRecord.ID,
		MediaSourceID:   libraryRecord.MediaSourceID,
		StorageProvider: "local",
		StoragePath:     "/library/Ambiguous/001.mkv",
		Container:       "mkv",
		ContentClass:    SourceContentClassVideo,
		Status:          "available",
		ScanState:       inventory.FileScanStateDiscovered,
	}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := svc.ensureInventoryFileSignals(ctx, libraryRecord.ID, "local", []database.InventoryFile{file}); err != nil {
		t.Fatalf("ensure signals: %v", err)
	}
	manifest, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, []database.InventoryFile{file}, "/library/Ambiguous")
	if err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	repo := recognition.NewRepository(db)
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	if err := repo.SaveDecisions(ctx, []database.RecognitionDecision{{
		ManifestID:   manifest.ID,
		CandidateID:  &graph.Candidates[0].ID,
		DecisionType: "resolver_outcome",
		Outcome:      recognition.DecisionOutcomeReviewRequired,
		TargetKind:   graph.Candidates[0].CandidateType,
		TargetKey:    graph.Candidates[0].CandidateKey,
	}}); err != nil {
		t.Fatalf("save review decision: %v", err)
	}
	if err := svc.markReviewRequiredInventoryFromManifest(ctx, manifest.ID); err != nil {
		t.Fatalf("mark review required: %v", err)
	}
	var refreshed database.InventoryFile
	if err := db.WithContext(ctx).First(&refreshed, file.ID).Error; err != nil {
		t.Fatalf("load refreshed file: %v", err)
	}
	if refreshed.ScanState != inventory.FileScanStateReviewRequired {
		t.Fatalf("expected scan_state review_required, got %#v", refreshed)
	}
}

func TestApplyRecognitionFallbackPosterWritesPerMetadataItem(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	itemA := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie A", SortTitle: "Movie A", SortKey: "movie-a", GovernanceStatus: database.ReviewStatePending}
	itemB := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie B", SortTitle: "Movie B", SortKey: "movie-b", GovernanceStatus: database.ReviewStatePending}
	resourceA := database.Resource{StableResourceKey: "resource:a", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	resourceB := database.Resource{StableResourceKey: "resource:b", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	fileA := database.InventoryFile{ID: 11, LibraryID: 1, StorageProvider: "openlist", StoragePath: "/library/A.mkv", ThumbnailURL: "https://image/a.jpg"}
	fileB := database.InventoryFile{ID: 12, LibraryID: 1, StorageProvider: "openlist", StoragePath: "/library/B.mkv", ThumbnailURL: "https://image/b.jpg"}
	for _, row := range []any{&itemA, &itemB, &resourceA, &resourceB, &fileA, &fileB} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}
	for _, row := range []any{
		&database.ResourceFile{ResourceID: resourceA.ID, InventoryFileID: fileA.ID, Role: database.ResourceFileRoleSource},
		&database.ResourceFile{ResourceID: resourceB.ID, InventoryFileID: fileB.ID, Role: database.ResourceFileRoleSource},
		&database.ResourceMetadataLink{ResourceID: resourceA.ID, MetadataItemID: itemA.ID, Role: database.ResourceLinkRolePrimary},
		&database.ResourceMetadataLink{ResourceID: resourceB.ID, MetadataItemID: itemB.ID, Role: database.ResourceLinkRolePrimary},
	} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed link: %v", err)
		}
	}
	if err := svc.applyRecognitionFallbackPoster(ctx, []database.InventoryFile{fileA, fileB}, []uint{itemA.ID, itemB.ID}); err != nil {
		t.Fatalf("apply fallback posters: %v", err)
	}
	var images []database.MetadataItemImage
	if err := db.WithContext(ctx).Order("metadata_item_id asc").Find(&images).Error; err != nil {
		t.Fatalf("load images: %v", err)
	}
	if len(images) != 2 || images[0].MetadataItemID != itemA.ID || images[0].URL != "https://image/a.jpg" || images[1].MetadataItemID != itemB.ID || images[1].URL != "https://image/b.jpg" {
		t.Fatalf("unexpected per-item fallback posters: %#v", images)
	}
}

func TestApplyRecognitionFallbackPosterSkipsWhenPosterExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", SortTitle: "Movie", SortKey: "movie", GovernanceStatus: database.ReviewStatePending}
	resource := database.Resource{StableResourceKey: "resource:1", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "openlist", StoragePath: "/library/Movie.mkv", ThumbnailURL: "https://image/thumb.jpg"}
	for _, row := range []any{&item, &resource, &file} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}
	for _, row := range []any{
		&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource},
		&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary},
	} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed link: %v", err)
		}
	}
	if err := db.WithContext(ctx).Create(&database.MetadataItemImage{MetadataItemID: item.ID, ImageType: "poster", URL: "https://image/existing.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create existing image: %v", err)
	}
	if err := svc.applyRecognitionFallbackPoster(ctx, []database.InventoryFile{file}, []uint{item.ID}); err != nil {
		t.Fatalf("apply fallback poster: %v", err)
	}
	var images []database.MetadataItemImage
	if err := db.WithContext(ctx).Where("metadata_item_id = ?", item.ID).Order("id asc").Find(&images).Error; err != nil {
		t.Fatalf("load images: %v", err)
	}
	if len(images) != 1 || images[0].URL != "https://image/existing.jpg" {
		t.Fatalf("expected existing poster to remain untouched, got %#v", images)
	}
}

func TestBackfillRecognitionFallbackPostersCreatesMissingPosters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", SortTitle: "Movie", SortKey: "movie", GovernanceStatus: database.ReviewStatePending}
	resource := database.Resource{StableResourceKey: "resource:1", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	file := database.InventoryFile{ID: 21, LibraryID: 1, StorageProvider: "openlist", StoragePath: "/library/Movie.mkv", ThumbnailURL: "https://image/thumb.jpg"}
	for _, row := range []any{&item, &resource, &file} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}
	for _, row := range []any{
		&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource},
		&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary},
	} {
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			t.Fatalf("seed link: %v", err)
		}
	}
	created, err := svc.BackfillRecognitionFallbackPosters(ctx)
	if err != nil {
		t.Fatalf("backfill fallback posters: %v", err)
	}
	if created != 1 {
		t.Fatalf("expected one created poster, got %d", created)
	}
	var images []database.MetadataItemImage
	if err := db.WithContext(ctx).Where("metadata_item_id = ?", item.ID).Find(&images).Error; err != nil {
		t.Fatalf("load images: %v", err)
	}
	if len(images) != 1 || images[0].URL != "https://image/thumb.jpg" || !images[0].IsSelected {
		t.Fatalf("unexpected backfilled poster rows: %#v", images)
	}
}
