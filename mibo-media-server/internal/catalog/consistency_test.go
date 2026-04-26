package catalog

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCatalogConsistencyReportAndRebuild(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Show A", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 2
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	asset := database.MediaAsset{LibraryID: 1, AssetType: "main", Status: AvailabilityAvailable, ProbeStatus: "ready"}
	if err := svc.db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: episode.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("create asset item: %v", err)
	}

	if err := svc.db.WithContext(ctx).Where("item_id IN ?", []uint{series.ID, season.ID, episode.ID}).Delete(&database.ItemRollup{}).Error; err != nil {
		t.Fatalf("delete rollups: %v", err)
	}
	if err := svc.db.WithContext(ctx).Where("item_id IN ?", []uint{series.ID, season.ID, episode.ID}).Delete(&database.CatalogSearchDocument{}).Error; err != nil {
		t.Fatalf("delete search docs: %v", err)
	}
	if err := svc.db.WithContext(ctx).Model(&database.CatalogItem{}).Where("id = ?", episode.ID).Update("availability_status", AvailabilityMissing).Error; err != nil {
		t.Fatalf("set mismatched availability: %v", err)
	}
	if err := svc.db.Exec("DROP INDEX idx_catalog_external_identity").Error; err != nil {
		t.Fatalf("drop external identity unique index: %v", err)
	}
	if err := svc.db.Exec("DROP INDEX idx_inventory_file_storage_path").Error; err != nil {
		t.Fatalf("drop inventory file unique index: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.CatalogExternalID{ItemID: series.ID, Provider: "tmdb", ProviderType: "tv", ExternalID: "tv:777"}).Error; err != nil {
		t.Fatalf("create first external id: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.CatalogExternalID{ItemID: season.ID, Provider: "tmdb", ProviderType: "tv", ExternalID: "tv:777"}).Error; err != nil {
		t.Fatalf("create duplicate external id: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/library/duplicate.mkv", Status: "available"}).Error; err != nil {
		t.Fatalf("create first duplicate inventory file: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/library/duplicate.mkv", Status: "available"}).Error; err != nil {
		t.Fatalf("create second duplicate inventory file: %v", err)
	}

	report, err := svc.CheckConsistency(ctx, nil)
	if err != nil {
		t.Fatalf("check consistency: %v", err)
	}
	if report.MissingRollupCount == 0 || report.MissingSearchDocumentCount == 0 || report.AvailabilityMismatchCount == 0 || report.AssetFileLinkGapCount == 0 {
		t.Fatalf("expected consistency gaps in report, got %#v", report)
	}
	if report.DuplicateExternalIDCount == 0 || report.DuplicateInventoryPathCount == 0 {
		t.Fatalf("expected duplicate identity and inventory path counts in report, got %#v", report)
	}

	result, err := svc.RebuildDerivedData(ctx, nil)
	if err != nil {
		t.Fatalf("rebuild derived data: %v", err)
	}
	if result.ItemsUpdated == 0 || result.ProjectionsRebuilt == 0 {
		t.Fatalf("expected rebuild result to record work, got %#v", result)
	}

	postReport, err := svc.CheckConsistency(ctx, nil)
	if err != nil {
		t.Fatalf("re-check consistency: %v", err)
	}
	if postReport.MissingRollupCount != 0 || postReport.MissingSearchDocumentCount != 0 || postReport.AvailabilityMismatchCount != 0 {
		t.Fatalf("expected rebuilt catalog to clear projection and availability mismatches, got %#v", postReport)
	}
}
