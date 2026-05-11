package database

import (
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"gorm.io/gorm"
)

func TestCatalogKernelTablesAreMigrated(t *testing.T) {
	db := openCatalogTestDB(t)

	for _, model := range requiredFreshStartupModels() {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("expected table for %T to exist", model)
		}
	}
}

func TestDatabaseOpenMigratesCatalogIndexes(t *testing.T) {
	db := openCatalogTestDB(t)

	requiredIndexes := []struct {
		model any
		name  string
	}{
		{&InventoryFile{}, "idx_inventory_file_source_storage_path"},
		{&InventoryFile{}, "idx_inventory_files_library_status_path"},
		{&MediaStream{}, "idx_media_stream_file_index"},
		{&SystemSetting{}, "idx_system_setting_category_key"},
		{&LibraryPath{}, "idx_library_paths_library_source_path"},
	}

	for _, index := range requiredIndexes {
		if !db.Migrator().HasIndex(index.model, index.name) {
			t.Fatalf("expected index %q to exist for %T", index.name, index.model)
		}
	}
}

func TestMetadataResourceGraphTablesAreMigrated(t *testing.T) {
	db := openCatalogTestDB(t)

	models := []any{
		&MetadataItem{},
		&MetadataExternalID{},
		&MetadataItemSource{},
		&MetadataItemFieldState{},
		&MetadataItemImage{},
		&MetadataItemPerson{},
		&MetadataItemTag{},
		&Resource{},
		&ResourceFile{},
		&ResourceLibraryLink{},
		&ResourceMetadataLink{},
		&LibraryMetadataProjection{},
		&MetadataSearchDocument{},
		&LibrarySearchDocument{},
		&UserMetadataData{},
		&UserResourceData{},
	}

	for _, model := range models {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("expected table for %T to exist", model)
		}
	}
}

func TestDatabaseOpenMigratesMetadataResourceIndexes(t *testing.T) {
	db := openCatalogTestDB(t)

	requiredIndexes := []struct {
		model any
		name  string
	}{
		{&MetadataItem{}, "idx_metadata_items_type_status_sort"},
		{&MetadataItem{}, "idx_metadata_items_parent_order"},
		{&MetadataItem{}, "idx_metadata_items_root_type_order"},
		{&MetadataExternalID{}, "idx_metadata_external_identity"},
		{&MetadataItemFieldState{}, "idx_metadata_item_field_state_identity"},
		{&Resource{}, "idx_resources_stable_resource_key"},
		{&ResourceFile{}, "idx_resource_files_resource_file_role_part"},
		{&ResourceFile{}, "idx_resource_files_resource_part"},
		{&ResourceLibraryLink{}, "idx_resource_library_link_identity"},
		{&ResourceMetadataLink{}, "idx_resource_metadata_link_identity"},
		{&LibraryMetadataProjection{}, "idx_library_metadata_projection_identity"},
		{&LibraryMetadataProjection{}, "idx_library_metadata_projections_library_type_availability_title"},
		{&LibrarySearchDocument{}, "idx_library_search_documents_library_type_availability_title"},
		{&UserMetadataData{}, "idx_user_metadata_data_identity"},
		{&UserResourceData{}, "idx_user_resource_data_identity"},
	}

	for _, index := range requiredIndexes {
		if !db.Migrator().HasIndex(index.model, index.name) {
			t.Fatalf("expected index %q to exist for %T", index.name, index.model)
		}
	}
}

func TestMetadataResourceGraphUniqueConstraints(t *testing.T) {
	db := openCatalogTestDB(t)

	item := MetadataItem{ItemType: MetadataItemTypeMovie, ContentForm: MetadataContentFormStandard, Title: "Movie", GovernanceStatus: ReviewStatePending}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	if item.ID == 0 {
		t.Fatal("expected metadata item id")
	}
	if err := db.Create(&MetadataItem{ItemType: MetadataItemTypeEpisode, ContentForm: MetadataContentFormAnime, ParentID: &item.ID, RootID: &item.ID, ParentIndexNumber: intPtr(1), IndexNumber: intPtr(2), Title: "Episode", GovernanceStatus: ReviewStatePending}).Error; err != nil {
		t.Fatalf("create hierarchical metadata item: %v", err)
	}

	externalID := MetadataExternalID{MetadataItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "123"}
	if err := db.Create(&externalID).Error; err != nil {
		t.Fatalf("create metadata external id: %v", err)
	}
	duplicateExternalID := MetadataExternalID{MetadataItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "123"}
	if err := db.Create(&duplicateExternalID).Error; err == nil {
		t.Fatal("expected duplicate metadata external id to fail")
	}

	resource := Resource{ResourceType: ResourceTypePlayable, ResourceShape: ResourceShapeSingleFile, StableResourceKey: "resource:movie:123", Status: "available"}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	file := InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/media/movie.mkv", ContentClass: "video", Status: "available", ScanState: "discovered"}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	resourceFile := ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: ResourceFileRoleSource, PartIndex: 0}
	if err := db.Create(&resourceFile).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	duplicateResourceFile := ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: ResourceFileRoleSource, PartIndex: 0}
	if err := db.Create(&duplicateResourceFile).Error; err == nil {
		t.Fatal("expected duplicate resource file link to fail")
	}

	libraryLink := ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 1, Status: "available", FirstSeenAt: item.CreatedAt, LastSeenAt: item.CreatedAt, ReviewState: ReviewStateAccepted}
	if err := db.Create(&libraryLink).Error; err != nil {
		t.Fatalf("create resource library link: %v", err)
	}
	duplicateLibraryLink := ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 1, Status: "available", FirstSeenAt: item.CreatedAt, LastSeenAt: item.CreatedAt, ReviewState: ReviewStateAccepted}
	if err := db.Create(&duplicateLibraryLink).Error; err == nil {
		t.Fatal("expected duplicate resource library link to fail")
	}

	projection := LibraryMetadataProjection{LibraryID: 1, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, AvailabilityStatus: ProjectionAvailabilityAvailable, LastProjectedAt: item.CreatedAt}
	if err := db.Create(&projection).Error; err != nil {
		t.Fatalf("create library metadata projection: %v", err)
	}
	duplicateProjection := LibraryMetadataProjection{LibraryID: 1, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, AvailabilityStatus: ProjectionAvailabilityAvailable, LastProjectedAt: item.CreatedAt}
	if err := db.Create(&duplicateProjection).Error; err == nil {
		t.Fatal("expected duplicate library metadata projection to fail")
	}
}

func openCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	return db
}

func intPtr(value int) *int {
	return &value
}
