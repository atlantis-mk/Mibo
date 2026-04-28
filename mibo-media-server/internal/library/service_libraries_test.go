package library

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestDeleteLibraryRemovesCatalogInventoryAndJobRecords(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrap database: %v", err)
	}
	defer sqlDB.Close()

	source := database.MediaSource{Name: "Local", Provider: "local", StorageRef: "local", RootPath: "/media"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	library := database.Library{Name: "Delete Me", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/delete", Status: "active", ScannerEnabled: true}
	otherLibrary := database.Library{Name: "Keep Me", Type: "movies", MediaSourceID: source.ID, RootPath: "/media/keep", Status: "active", ScannerEnabled: true}
	if err := db.Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.Create(&otherLibrary).Error; err != nil {
		t.Fatalf("create other library: %v", err)
	}

	item := database.CatalogItem{LibraryID: library.ID, Type: "movie", Title: "Gone", SortKey: "gone", AvailabilityStatus: "available", GovernanceStatus: "matched"}
	otherItem := database.CatalogItem{LibraryID: otherLibrary.ID, Type: "movie", Title: "Stay", SortKey: "stay", AvailabilityStatus: "available", GovernanceStatus: "matched"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.Create(&otherItem).Error; err != nil {
		t.Fatalf("create other item: %v", err)
	}
	inventoryFile := database.InventoryFile{LibraryID: library.ID, StorageProvider: "local", StoragePath: "/media/delete/movie.mkv", Status: "available"}
	otherInventoryFile := database.InventoryFile{LibraryID: otherLibrary.ID, StorageProvider: "local", StoragePath: "/media/keep/movie.mkv", Status: "available"}
	if err := db.Create(&inventoryFile).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := db.Create(&otherInventoryFile).Error; err != nil {
		t.Fatalf("create other inventory file: %v", err)
	}
	asset := database.MediaAsset{LibraryID: library.ID, AssetType: "main", Status: "available", ProbeStatus: "completed"}
	otherAsset := database.MediaAsset{LibraryID: otherLibrary.ID, AssetType: "main", Status: "available", ProbeStatus: "completed"}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.Create(&otherAsset).Error; err != nil {
		t.Fatalf("create other asset: %v", err)
	}
	person := database.Person{Name: "Deleted Person", SortName: "deleted person"}
	otherPerson := database.Person{Name: "Kept Person", SortName: "kept person"}
	tag := database.Tag{Kind: "genre", Name: "Deleted"}
	otherTag := database.Tag{Kind: "genre", Name: "Kept"}
	if err := db.Create(&person).Error; err != nil {
		t.Fatalf("create person: %v", err)
	}
	if err := db.Create(&otherPerson).Error; err != nil {
		t.Fatalf("create other person: %v", err)
	}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if err := db.Create(&otherTag).Error; err != nil {
		t.Fatalf("create other tag: %v", err)
	}
	seed := []any{
		&database.AssetFile{AssetID: asset.ID, FileID: inventoryFile.ID, Role: "source"},
		&database.AssetFile{AssetID: otherAsset.ID, FileID: otherInventoryFile.ID, Role: "source"},
		&database.AssetItem{AssetID: asset.ID, ItemID: item.ID, Role: "primary"},
		&database.AssetItem{AssetID: otherAsset.ID, ItemID: otherItem.ID, Role: "primary"},
		&database.MediaStream{FileID: inventoryFile.ID, StreamIndex: 0, StreamType: "video"},
		&database.MediaStream{FileID: otherInventoryFile.ID, StreamIndex: 0, StreamType: "video"},
		&database.MediaStream{FileID: 999999, StreamIndex: 0, StreamType: "video"},
		&database.CatalogExternalID{ItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "deleted"},
		&database.MetadataSource{ItemID: item.ID, SourceType: "remote", SourceName: "tmdb", FetchedAt: time.Now()},
		&database.MetadataFieldState{ItemID: item.ID, FieldKey: "title", ValueJSON: `"Gone"`},
		&database.ItemImage{ItemID: item.ID, ImageType: "poster", URL: "https://example.test/poster.jpg"},
		&database.ItemPerson{ItemID: item.ID, PersonID: person.ID, Role: "cast"},
		&database.ItemPerson{ItemID: otherItem.ID, PersonID: otherPerson.ID, Role: "cast"},
		&database.ItemTag{ItemID: item.ID, TagID: tag.ID},
		&database.ItemTag{ItemID: otherItem.ID, TagID: otherTag.ID},
		&database.ItemRollup{ItemID: item.ID, UpdatedAt: time.Now()},
		&database.CatalogSearchDocument{ItemID: item.ID, LibraryID: library.ID, ItemType: "movie", Title: "Gone", AvailabilityStatus: "available"},
		&database.User{Username: "user", PasswordHash: "hash", Role: "admin"},
	}
	for _, record := range seed {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
	if err := db.Create(&database.UserItemData{UserID: 1, ItemID: item.ID, AssetID: &asset.ID}).Error; err != nil {
		t.Fatalf("create user item data: %v", err)
	}
	schedule := database.Schedule{Name: "scan", Kind: "scan", ScopeKind: "library", LibraryID: &library.ID, FrequencyKind: "daily", TimeOfDay: "03:00", Enabled: true}
	if err := db.Create(&schedule).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	if err := db.Create(&database.ScheduleRun{ScheduleID: schedule.ID, Status: "completed"}).Error; err != nil {
		t.Fatalf("create schedule run: %v", err)
	}
	job := database.Job{Kind: JobKindSyncLibrary, Status: "queued", PayloadJSON: fmt.Sprintf(`{"library_id":%d,"root_path":"/media/delete"}`, library.ID), AvailableAt: time.Now()}
	probeJob := database.Job{Kind: JobKindProbeInventoryFile, Status: "queued", PayloadJSON: fmt.Sprintf(`{"inventory_file_id":%d}`, inventoryFile.ID), AvailableAt: time.Now()}
	otherJob := database.Job{Kind: JobKindSyncLibrary, Status: "queued", PayloadJSON: fmt.Sprintf(`{"library_id":%d,"root_path":"/media/keep"}`, otherLibrary.ID), AvailableAt: time.Now()}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := db.Create(&probeJob).Error; err != nil {
		t.Fatalf("create probe job: %v", err)
	}
	if err := db.Create(&otherJob).Error; err != nil {
		t.Fatalf("create other job: %v", err)
	}
	if err := db.Create(&database.JobActiveIntent{IntentKey: "sync", Kind: job.Kind, JobID: job.ID}).Error; err != nil {
		t.Fatalf("create active intent: %v", err)
	}
	if err := db.Exec(`CREATE TABLE media_items (id integer PRIMARY KEY AUTOINCREMENT, library_id integer NOT NULL, type text NOT NULL, title text NOT NULL, source_path text NOT NULL, status text NOT NULL, created_at datetime, updated_at datetime)`).Error; err != nil {
		t.Fatalf("create legacy media_items table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE media_files (id integer PRIMARY KEY AUTOINCREMENT, library_id integer NOT NULL, media_item_id integer, storage_path text NOT NULL, created_at datetime, updated_at datetime)`).Error; err != nil {
		t.Fatalf("create legacy media_files table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE playback_progresses (id integer PRIMARY KEY AUTOINCREMENT, user_id integer NOT NULL, media_item_id integer NOT NULL, media_file_id integer, created_at datetime, updated_at datetime)`).Error; err != nil {
		t.Fatalf("create legacy playback_progresses table: %v", err)
	}
	if err := db.Exec(`INSERT INTO media_items (id, library_id, type, title, source_path, status, created_at, updated_at) VALUES (101, ?, 'movie', 'Legacy Gone', '/media/delete/movie.mkv', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, library.ID).Error; err != nil {
		t.Fatalf("create legacy item: %v", err)
	}
	if err := db.Exec(`INSERT INTO media_files (id, library_id, media_item_id, storage_path, created_at, updated_at) VALUES (201, ?, 101, '/media/delete/movie.mkv', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, library.ID).Error; err != nil {
		t.Fatalf("create legacy file: %v", err)
	}
	if err := db.Exec(`INSERT INTO playback_progresses (user_id, media_item_id, media_file_id, created_at, updated_at) VALUES (1, 101, 201, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatalf("create legacy progress: %v", err)
	}

	svc := NewService(config.Config{}, db, nil, nil)
	if err := svc.DeleteLibrary(context.Background(), library.ID); err != nil {
		t.Fatalf("delete library: %v", err)
	}

	assertRawTableCount(t, db, "libraries", "id = ?", 0, library.ID)
	assertRawTableCount(t, db, "catalog_items", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "inventory_files", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "media_assets", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "asset_items", "item_id = ? OR asset_id = ?", 0, item.ID, asset.ID)
	assertRawTableCount(t, db, "asset_files", "file_id = ? OR asset_id = ?", 0, inventoryFile.ID, asset.ID)
	assertRawTableCount(t, db, "media_streams", "file_id = ?", 0, inventoryFile.ID)
	assertRawTableCount(t, db, "media_streams", "file_id = 999999", 0)
	assertRawTableCount(t, db, "catalog_external_ids", "item_id = ?", 0, item.ID)
	assertRawTableCount(t, db, "metadata_sources", "item_id = ?", 0, item.ID)
	assertRawTableCount(t, db, "metadata_field_states", "item_id = ?", 0, item.ID)
	assertRawTableCount(t, db, "item_images", "item_id = ?", 0, item.ID)
	assertRawTableCount(t, db, "item_people", "item_id = ?", 0, item.ID)
	assertRawTableCount(t, db, "item_tags", "item_id = ?", 0, item.ID)
	assertRawTableCount(t, db, "item_rollups", "item_id = ?", 0, item.ID)
	assertRawTableCount(t, db, "catalog_search_documents", "item_id = ? OR library_id = ?", 0, item.ID, library.ID)
	assertRawTableCount(t, db, "user_item_data", "item_id = ? OR asset_id = ?", 0, item.ID, asset.ID)
	assertRawTableCount(t, db, "schedules", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "schedule_runs", "schedule_id = ?", 0, schedule.ID)
	assertRawTableCount(t, db, "jobs", "id IN (?, ?)", 0, job.ID, probeJob.ID)
	assertRawTableCount(t, db, "job_active_intents", "job_id = ?", 0, job.ID)
	assertRawTableCount(t, db, "media_items", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "media_files", "library_id = ?", 0, library.ID)
	assertRawTableCount(t, db, "playback_progresses", "media_item_id = 101 OR media_file_id = 201", 0)
	assertRawTableCount(t, db, "people", "id = ?", 0, person.ID)
	assertRawTableCount(t, db, "tags", "id = ?", 0, tag.ID)

	assertRawTableCount(t, db, "libraries", "id = ?", 1, otherLibrary.ID)
	assertRawTableCount(t, db, "catalog_items", "id = ?", 1, otherItem.ID)
	assertRawTableCount(t, db, "inventory_files", "id = ?", 1, otherInventoryFile.ID)
	assertRawTableCount(t, db, "media_assets", "id = ?", 1, otherAsset.ID)
	assertRawTableCount(t, db, "jobs", "id = ?", 1, otherJob.ID)
	assertRawTableCount(t, db, "people", "id = ?", 1, otherPerson.ID)
	assertRawTableCount(t, db, "tags", "id = ?", 1, otherTag.ID)
}

func assertRawTableCount(t *testing.T, db *gorm.DB, table, where string, expected int64, args ...any) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(where, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("%s count = %d, want %d", table, count, expected)
	}
}
