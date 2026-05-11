package library

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
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
