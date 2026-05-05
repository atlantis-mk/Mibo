package metadata

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

func TestApplyCatalogGovernanceFieldOperationRecordsManualApply(t *testing.T) {
	ctx := context.Background()
	db, settingsSvc := newGovernanceMetadataTestServices(t)
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Unmatched", Path: "/movies/unmatched.mkv", SortKey: "Unmatched", GovernanceStatus: catalog.GovernanceUnmatched})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.MetadataOperation{Operation: OperationTypeMatch, OriginItemID: item.ID, TargetItemID: item.ID, LibraryID: item.LibraryID, Status: OperationStatusNoCandidate, GovernanceStatus: catalog.GovernanceUnmatched, StartedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("seed no candidate operation: %v", err)
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).ApplyCatalogGovernanceFieldOperation(ctx, item.ID, ApplyGovernanceFieldInput{FieldKey: "title", Value: "Manual Title"})
	if err != nil {
		t.Fatalf("apply governance field operation: %v", err)
	}
	if operation.Operation != OperationTypeManualApply || operation.Status != OperationStatusApplied || operation.GovernanceStatus != catalog.GovernanceManual {
		t.Fatalf("unexpected operation: %#v", operation)
	}
	if len(operation.AppliedFields) != 2 || operation.AppliedFields[0].FieldKey != "title" || operation.AppliedFields[1].FieldKey != "governance_status" {
		t.Fatalf("unexpected applied fields: %#v", operation.AppliedFields)
	}

	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if stored.Title != "Manual Title" || stored.GovernanceStatus != catalog.GovernanceManual {
		t.Fatalf("unexpected stored item: %#v", stored)
	}
	var recorded database.MetadataOperation
	if err := db.WithContext(ctx).Where("target_item_id = ? AND operation = ?", item.ID, OperationTypeManualApply).Order("id desc").First(&recorded).Error; err != nil {
		t.Fatalf("load recorded operation: %v", err)
	}
	if recorded.Status != OperationStatusApplied || recorded.GovernanceStatus != catalog.GovernanceManual {
		t.Fatalf("unexpected recorded operation: %#v", recorded)
	}
}

func TestApplyCatalogGovernanceFieldOperationSkippedLockedField(t *testing.T) {
	ctx := context.Background()
	db, settingsSvc := newGovernanceMetadataTestServices(t)
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Locked", Path: "/movies/locked.mkv", SortKey: "Locked", GovernanceStatus: catalog.GovernanceUnmatched})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if _, _, err := catalog.NewService(db).ApplyField(ctx, catalog.ApplyFieldInput{ItemID: item.ID, FieldKey: "title", Value: "Locked", Lock: true, Force: true}); err != nil {
		t.Fatalf("pre-lock title: %v", err)
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).ApplyCatalogGovernanceFieldOperation(ctx, item.ID, ApplyGovernanceFieldInput{FieldKey: "title", Value: "Ignored"})
	if err != nil {
		t.Fatalf("apply locked field operation: %v", err)
	}
	if operation.Status != OperationStatusSkipped || len(operation.SkippedFields) != 1 || operation.SkippedFields[0].Reason != "locked" {
		t.Fatalf("unexpected skipped operation: %#v", operation)
	}
	var stored database.CatalogItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("reload catalog item: %v", err)
	}
	if stored.Title != "Locked" || stored.GovernanceStatus != catalog.GovernanceUnmatched {
		t.Fatalf("unexpected stored item after skipped edit: %#v", stored)
	}
}

func TestApplyCatalogGovernanceFieldOperationAcceptsClassificationReview(t *testing.T) {
	ctx := context.Background()
	db, settingsSvc := newGovernanceMetadataTestServices(t)
	fileID := uint(42)
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Unmatched", Path: "/movies/unmatched.mkv", SortKey: "Unmatched", GovernanceStatus: catalog.GovernanceUnmatched})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	decision := database.ClassificationDecision{LibraryID: item.LibraryID, InventoryFileID: &fileID, ItemID: &item.ID, SourcePath: item.Path, DecisionType: "movie", TargetKey: item.Path, Status: "review_required"}
	if err := db.WithContext(ctx).Create(&decision).Error; err != nil {
		t.Fatalf("create classification decision: %v", err)
	}

	_, err = NewService(db, config.MetadataConfig{}, settingsSvc).ApplyCatalogGovernanceFieldOperation(ctx, item.ID, ApplyGovernanceFieldInput{FieldKey: "title", Value: "Manual Title"})
	if err != nil {
		t.Fatalf("apply governance field operation: %v", err)
	}

	var stored database.ClassificationDecision
	if err := db.WithContext(ctx).First(&stored, decision.ID).Error; err != nil {
		t.Fatalf("reload classification decision: %v", err)
	}
	if stored.Status != "accepted" || stored.ResolvedAt == nil {
		t.Fatalf("expected classification decision to be accepted, got %#v", stored)
	}
}

func TestApplyCatalogGovernanceFieldOperationQueuesClassificationReviewRefresh(t *testing.T) {
	ctx := context.Background()
	db, settingsSvc := newGovernanceMetadataTestServices(t)
	if err := db.WithContext(ctx).Create(&database.MediaSource{ID: 1, Name: "Local", Provider: "local", RootPath: "/media"}).Error; err != nil {
		t.Fatalf("create media source: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.Library{ID: 1, Name: "Movies", Type: "movies", MediaSourceID: 1, RootPath: "/media"}).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/media/Movie.mkv", ContentClass: "video", Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Unmatched", Path: file.StoragePath, SortKey: "Unmatched", GovernanceStatus: catalog.GovernanceUnmatched})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ClassificationDecision{LibraryID: item.LibraryID, InventoryFileID: &file.ID, ItemID: &item.ID, SourcePath: item.Path, DecisionType: "movie", TargetKey: item.Path, Status: "review_required"}).Error; err != nil {
		t.Fatalf("create classification decision: %v", err)
	}
	ingestSvc := ingest.NewService(db)

	_, err = NewService(db, config.MetadataConfig{}, settingsSvc, ingestSvc).ApplyCatalogGovernanceFieldOperation(ctx, item.ID, ApplyGovernanceFieldInput{FieldKey: "title", Value: "Manual Title"})
	if err != nil {
		t.Fatalf("apply governance field operation: %v", err)
	}

	var dirty database.IngestDirtyUnit
	if err := db.WithContext(ctx).Where("dirty_key = ?", "inventory_file:1").First(&dirty).Error; err != nil {
		t.Fatalf("load dirty unit: %v", err)
	}
	if dirty.Reason != "classification_metadata_confirmed" || dirty.Status != ingest.DirtyStatusDirty {
		t.Fatalf("unexpected dirty unit: %#v", dirty)
	}
}

func TestApplyCatalogGovernanceClassificationCorrectionCopiesImages(t *testing.T) {
	ctx := context.Background()
	db, settingsSvc := newGovernanceMetadataTestServices(t)
	series, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeries, Title: "Show", Path: "/media/Show", SortKey: "Show", AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceManual})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonIndex := 1
	season, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Path: "/media/Show/Season 01", Title: "Season 1", SortKey: "Season 1", IndexNumber: &seasonIndex, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceManual})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeIndex := 1
	episode, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Path: "/media/Show/Season 01/E01.mp4", Title: "Episode 1", SortKey: "Episode 1", ParentIndexNumber: &seasonIndex, IndexNumber: &episodeIndex, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceManual})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if err := db.WithContext(ctx).Create([]database.ItemImage{{ItemID: series.ID, ImageType: "poster", URL: "https://example.test/show-poster.jpg", IsSelected: true}, {ItemID: series.ID, ImageType: "backdrop", URL: "https://example.test/show-backdrop.jpg", IsSelected: true}, {ItemID: episode.ID, ImageType: "still", URL: "https://example.test/episode-still.jpg", IsSelected: true}}).Error; err != nil {
		t.Fatalf("seed images: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/media/Show/Season 01/E01.mp4", ContentClass: "video", Status: inventory.FileStatusAvailable}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	asset := database.MediaAsset{LibraryID: 1, AssetType: inventory.AssetTypeMain, Status: inventory.AssetStatusAvailable, ProbeStatus: "ready"}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: inventory.FileRoleSource}).Error; err != nil {
		t.Fatalf("link asset file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: episode.ID, Role: inventory.AssetItemRolePrimary, Source: "scanner"}).Error; err != nil {
		t.Fatalf("link asset item: %v", err)
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).ApplyCatalogGovernanceClassificationCorrectionOperation(ctx, episode.ID, ApplyGovernanceClassificationCorrectionInput{Action: "movie_versions", RootPath: "/media/Show/Season 01", Title: "Movie"})
	if err != nil {
		t.Fatalf("apply correction: %v", err)
	}

	var images []database.ItemImage
	if err := db.WithContext(ctx).Where("item_id = ? AND is_selected = ?", operation.TargetItemID, true).Find(&images).Error; err != nil {
		t.Fatalf("load movie images: %v", err)
	}
	selected := make(map[string]string, len(images))
	for _, image := range images {
		selected[image.ImageType] = image.URL
	}
	if selected["poster"] != "https://example.test/show-poster.jpg" {
		t.Fatalf("expected migrated movie poster, got %#v", images)
	}
	if selected["backdrop"] != "https://example.test/show-backdrop.jpg" {
		t.Fatalf("expected migrated movie backdrop, got %#v", images)
	}
}

func TestApplyCatalogGovernanceClassificationCorrectionCreatesIndependentMovies(t *testing.T) {
	ctx := context.Background()
	db, settingsSvc := newGovernanceMetadataTestServices(t)
	series, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeries, Title: "Show", Path: "/media/Show", SortKey: "Show", AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceManual})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonIndex := 1
	season, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeSeason, ParentID: &series.ID, Path: "/media/Show/Season 01", Title: "Season 1", SortKey: "Season 1", IndexNumber: &seasonIndex, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceManual})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	for index, name := range []string{"Movie A.mp4", "Movie B.mp4"} {
		episodeIndex := index + 1
		episode, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeEpisode, ParentID: &season.ID, Path: "/media/Show/Season 01/" + name, Title: name, SortKey: name, ParentIndexNumber: &seasonIndex, IndexNumber: &episodeIndex, AvailabilityStatus: catalog.AvailabilityAvailable, GovernanceStatus: catalog.GovernanceManual})
		if err != nil {
			t.Fatalf("create episode: %v", err)
		}
		file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/media/Show/Season 01/" + name, ContentClass: "video", Status: inventory.FileStatusAvailable}
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
		asset := database.MediaAsset{LibraryID: 1, AssetType: inventory.AssetTypeMain, Status: inventory.AssetStatusAvailable, ProbeStatus: "ready"}
		if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
			t.Fatalf("create asset: %v", err)
		}
		if err := db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: inventory.FileRoleSource}).Error; err != nil {
			t.Fatalf("link asset file: %v", err)
		}
		if err := db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: episode.ID, Role: inventory.AssetItemRolePrimary, Source: "scanner"}).Error; err != nil {
			t.Fatalf("link asset item: %v", err)
		}
	}

	operation, err := NewService(db, config.MetadataConfig{}, settingsSvc).ApplyCatalogGovernanceClassificationCorrectionOperation(ctx, series.ID, ApplyGovernanceClassificationCorrectionInput{Action: "independent_movies", RootPath: "/media/Show/Season 01"})
	if err != nil {
		t.Fatalf("apply correction: %v", err)
	}
	if len(operation.AffectedScope.ItemIDs) != 6 {
		t.Fatalf("expected two movies plus original series scope, got %#v", operation.AffectedScope.ItemIDs)
	}

	var movies []database.CatalogItem
	if err := db.WithContext(ctx).Where("type = ? AND deleted_at IS NULL", catalog.ItemTypeMovie).Order("path asc").Find(&movies).Error; err != nil {
		t.Fatalf("load movies: %v", err)
	}
	if len(movies) != 2 {
		t.Fatalf("expected two independent movies, got %#v", movies)
	}
	for _, movie := range movies {
		var link database.AssetItem
		if err := db.WithContext(ctx).Where("item_id = ? AND role = ?", movie.ID, inventory.AssetItemRolePrimary).First(&link).Error; err != nil {
			t.Fatalf("load movie asset link: %v", err)
		}
	}
}

func newGovernanceMetadataTestServices(t *testing.T) (*gorm.DB, *settings.Service) {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{})
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}
	if _, err := settingsSvc.UpdateLibraryMetadataStrategy(ctx, 1, settings.UpdateLibraryMetadataStrategyInput{}); err != nil {
		t.Fatalf("update metadata strategy: %v", err)
	}
	return db, settingsSvc
}
