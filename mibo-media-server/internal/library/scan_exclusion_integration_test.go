package library

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func TestRunSyncLibrarySkipsAdvertisementFilesAndContinuesSiblings(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	movieDir := filepath.Join(moviesRoot, "Movie A (2024)")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "advertisement.mp4"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Movie A (2024).mkv"))
	mustWriteFixtureTextFile(t, filepath.Join(movieDir, "Movie A (2024).srt"), "subtitle")
	mustWriteFixtureFile(t, filepath.Join(movieDir, "Child", "Child Movie (2025).mkv"))
	mustWriteFixtureFile(t, filepath.Join(movieDir, "ads", "commercial.mkv"))
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)

	provider, err := svc.storage.Get("local")
	if err != nil {
		t.Fatalf("get local provider: %v", err)
	}
	result, err := svc.scanLibrary(ctx, provider, libraryRecord, libraryRecord.RootPath)
	if err != nil {
		t.Fatalf("scan library: %v", err)
	}

	configurableSkips := 0
	for source, count := range result.ExcludedFilesSkippedByReason {
		if strings.HasPrefix(source, scanExclusionSkipConfigurableRule) {
			configurableSkips += count
		}
	}
	if result.ExcludedFilesSkipped != 2 || configurableSkips != 2 {
		t.Fatalf("expected two automatic ad skips, got %#v", result)
	}
	assertCatalogCounts(t, ctx, db, 2, 3, 2, 2, 3, 2)
	assertNoInventoryFilePath(t, ctx, db, filepath.Join(movieDir, "advertisement.mp4"))
	assertNoInventoryFilePath(t, ctx, db, filepath.Join(movieDir, "ads", "commercial.mkv"))
	assertJobCount(t, ctx, db, JobKindCatalogMatchBatch, 0)
	assertJobCount(t, ctx, db, JobKindInventoryProbeBatch, 0)
}

func TestMarkScanExclusionCreatesRecordAndHidesScannerAsset(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	adPath := filepath.Join(moviesRoot, "Movie A (2024)", "promo.mp4")
	mustWriteFixtureFile(t, adPath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	var file database.InventoryFile
	if err := db.WithContext(ctx).Where("storage_path = ?", adPath).First(&file).Error; err != nil {
		t.Fatalf("load scanned file: %v", err)
	}
	exclusion, err := svc.MarkScanExclusion(ctx, MarkScanExclusionInput{InventoryFileID: file.ID, Reason: ScanExclusionReasonAdvertisement})
	if err != nil {
		t.Fatalf("mark scan exclusion: %v", err)
	}
	if exclusion.Reason != ScanExclusionReasonAdvertisement || !exclusion.Enabled || exclusion.StoragePath != adPath {
		t.Fatalf("unexpected exclusion: %#v", exclusion)
	}

	var reloadedFile database.InventoryFile
	if err := db.WithContext(ctx).First(&reloadedFile, file.ID).Error; err != nil {
		t.Fatalf("reload file: %v", err)
	}
	if reloadedFile.Status != inventory.FileStatusMissing {
		t.Fatalf("expected source file to be hidden as missing, got %q", reloadedFile.Status)
	}
	var assetItems int64
	if err := db.WithContext(ctx).Model(&database.AssetItem{}).Count(&assetItems).Error; err != nil {
		t.Fatalf("count asset item links: %v", err)
	}
	if assetItems != 0 {
		t.Fatalf("expected scanner asset links to be removed, got %d", assetItems)
	}
	var item database.CatalogItem
	if err := db.WithContext(ctx).First(&item).Error; err != nil {
		t.Fatalf("load catalog item: %v", err)
	}
	if item.AvailabilityStatus != catalog.AvailabilityMissing {
		t.Fatalf("expected catalog item to become missing, got %q", item.AvailabilityStatus)
	}
	if _, err := svc.SetScanExclusionEnabled(ctx, SetScanExclusionEnabledInput{ExclusionID: exclusion.ID, Enabled: false}); err != nil {
		t.Fatalf("disable exclusion: %v", err)
	}
	var disabled database.ScanExclusion
	if err := db.WithContext(ctx).First(&disabled, exclusion.ID).Error; err != nil {
		t.Fatalf("reload disabled exclusion: %v", err)
	}
	if disabled.Enabled || disabled.DisabledAt == nil {
		t.Fatalf("expected disabled exclusion with audit timestamp, got %#v", disabled)
	}
}

func TestUserMarkedScanExclusionPreventsReimport(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, svc, libraryRecord := newIdentityScanService(t)
	provider := &stableIdentityProvider{objects: [][]storage.Object{{
		{Name: "MovieA.2024.mkv", Path: "/library/MovieA.2024.mkv", Size: 2048, StableIdentity: "stable-movie-a"},
	}}}
	if _, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library"); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	var file database.InventoryFile
	if err := db.WithContext(ctx).Where("stable_identity_key = ?", "stable-movie-a").First(&file).Error; err != nil {
		t.Fatalf("load file: %v", err)
	}
	if _, err := svc.MarkScanExclusion(ctx, MarkScanExclusionInput{InventoryFileID: file.ID, Reason: ScanExclusionReasonAdvertisement}); err != nil {
		t.Fatalf("mark exclusion: %v", err)
	}

	result, err := svc.scanLibrary(ctx, provider, libraryRecord, "/library")
	if err != nil {
		t.Fatalf("rescan: %v", err)
	}
	if result.ExcludedFilesSkipped != 1 || result.ExcludedFilesSkippedByReason[scanExclusionSkipUserExclusion] != 1 {
		t.Fatalf("expected user exclusion skip on rescan, got %#v", result)
	}
	assertJobCount(t, ctx, db, JobKindCatalogMatchBatch, 0)
	assertJobCount(t, ctx, db, JobKindInventoryProbeBatch, 0)
}

func TestMarkScanExclusionFromItemAllowsNonScannerAssetLink(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	ctx, db, svc := newDirectScanHarness(t, rootPath)
	moviesRoot := filepath.Join(rootPath, "movies")
	moviePath := filepath.Join(moviesRoot, "Movie A (2024).mkv")
	mustWriteFixtureFile(t, moviePath)
	libraryRecord := createDirectScanLibrary(t, ctx, svc, "Movies", "movies", moviesRoot)
	if err := svc.RunSyncLibrary(ctx, newSyncLibraryJobPayload(libraryRecord.ID, libraryRecord.RootPath)); err != nil {
		t.Fatalf("run sync: %v", err)
	}

	if err := db.WithContext(ctx).Model(&database.AssetItem{}).Where("1 = 1").Update("source", "metadata_match").Error; err != nil {
		t.Fatalf("update asset item source: %v", err)
	}
	var item database.CatalogItem
	if err := db.WithContext(ctx).First(&item).Error; err != nil {
		t.Fatalf("load catalog item: %v", err)
	}

	exclusion, err := svc.MarkScanExclusion(ctx, MarkScanExclusionInput{ItemID: item.ID, Reason: ScanExclusionReasonAdvertisement})
	if err != nil {
		t.Fatalf("mark scan exclusion from non-scanner item link: %v", err)
	}
	if exclusion.StoragePath != moviePath || !exclusion.Enabled {
		t.Fatalf("unexpected exclusion: %#v", exclusion)
	}
	var assetItems int64
	if err := db.WithContext(ctx).Model(&database.AssetItem{}).Count(&assetItems).Error; err != nil {
		t.Fatalf("count asset item links: %v", err)
	}
	if assetItems != 0 {
		t.Fatalf("expected non-scanner asset links to be removed, got %d", assetItems)
	}
}

func assertNoInventoryFilePath(t *testing.T, ctx context.Context, db *gorm.DB, filePath string) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Model(&database.InventoryFile{}).Where("storage_path = ?", filePath).Count(&count).Error; err != nil {
		t.Fatalf("count inventory file path %s: %v", filePath, err)
	}
	if count != 0 {
		t.Fatalf("expected no inventory file for %s, got %d", filePath, count)
	}
}

func assertJobCount(t *testing.T, ctx context.Context, db *gorm.DB, kind string, expected int64) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Model(&database.Job{}).Where("kind = ?", kind).Count(&count).Error; err != nil {
		t.Fatalf("count jobs %s: %v", kind, err)
	}
	if count != expected {
		t.Fatalf("expected %d %s jobs, got %d", expected, kind, count)
	}
}
