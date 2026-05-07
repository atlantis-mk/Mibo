package database

import (
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"gorm.io/gorm"
)

func TestDatabaseOpenFreshCatalogDatabaseMigratesCatalogTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mibo.db")
	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	assertTablesExist(t, db, requiredFreshStartupModels())
	assertLegacyMetadataSchemaRemoved(t, db)
	assertCatalogIndexesExist(t, db)
	assertMediaStreamTechnicalColumnsExist(t, db)
}

func requiredFreshStartupModels() []any {
	return []any{
		&MediaSource{},
		&Library{},
		&LibraryPath{},
		&LibraryScanPolicy{},
		&LibraryMetadataPolicy{},
		&LibraryPlaybackPolicy{},
		&LibrarySubtitlePolicy{},
		&CatalogItem{},
		&CatalogExternalID{},
		&CatalogIdentity{},
		&MetadataSource{},
		&MetadataFieldState{},
		&ItemImage{},
		&Person{},
		&ItemPerson{},
		&Tag{},
		&ItemTag{},
		&MediaAsset{},
		&AssetItem{},
		&InventoryFile{},
		&StorageIndexEntry{},
		&StorageObservationFailure{},
		&StorageDirectoryFingerprint{},
		&ContentShapeProfile{},
		&ContentShapePlan{},
		&ContentShapeAssignment{},
		&AssetFile{},
		&MediaStream{},
		&UserItemData{},
		&ItemRollup{},
		&CatalogSearchDocument{},
		&Job{},
		&JobActiveIntent{},
		&Schedule{},
		&ScheduleRun{},
		&User{},
		&Session{},
		&SystemSetting{},
		&SearchHistory{},
		&IngestDirtyUnit{},
		&IngestCondition{},
		&IngestEvent{},
		&WorkflowRun{},
		&WorkflowTask{},
		&WorkflowTaskDependency{},
		&WorkflowTaskLease{},
		&WorkflowResourceBudget{},
		&WorkflowResourceUsage{},
	}
}

func assertTablesExist(t *testing.T, db *gorm.DB, models []any) {
	t.Helper()

	for _, model := range models {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("expected table for %T to exist", model)
		}
	}
}

func assertCatalogIndexesExist(t *testing.T, db *gorm.DB) {
	t.Helper()

	requiredIndexes := []struct {
		model any
		name  string
	}{
		{&CatalogItem{}, "idx_catalog_items_library_type_availability_sort"},
		{&CatalogItem{}, "idx_catalog_items_parent_order"},
		{&CatalogItem{}, "idx_catalog_items_root_type_order"},
		{&CatalogSearchDocument{}, "idx_catalog_search_documents_library_type_availability_title"},
		{&CatalogExternalID{}, "idx_catalog_external_identity"},
		{&CatalogIdentity{}, "idx_catalog_identity_key"},
		{&MetadataFieldState{}, "idx_metadata_field_state_item_field"},
		{&AssetItem{}, "idx_asset_items_item_role"},
		{&AssetItem{}, "idx_asset_items_asset_item_role_segment"},
		{&AssetFile{}, "idx_asset_files_asset_part"},
		{&AssetFile{}, "idx_asset_files_asset_file_role_part"},
		{&InventoryFile{}, "idx_inventory_file_source_storage_path"},
		{&InventoryFile{}, "idx_inventory_files_library_status_path"},
		{&StorageIndexEntry{}, "idx_storage_index_identity"},
		{&StorageIndexEntry{}, "idx_storage_index_library_status_path"},
		{&StorageIndexEntry{}, "idx_storage_index_stable_identity"},
		{&StorageObservationFailure{}, "idx_storage_observation_failure_library_path"},
		{&ContentShapeProfile{}, "idx_content_shape_profile_identity"},
		{&ContentShapeProfile{}, "idx_content_shape_profiles_library_state"},
		{&ContentShapePlan{}, "idx_content_shape_plan_scope"},
		{&ContentShapePlan{}, "idx_content_shape_plans_library_state"},
		{&ContentShapeAssignment{}, "idx_content_shape_assignment_file"},
		{&ContentShapeAssignment{}, "idx_content_shape_assignments_library_state"},
		{&MediaStream{}, "idx_media_stream_file_index"},
		{&UserItemData{}, "idx_user_item_data_user_item_asset"},
		{&SystemSetting{}, "idx_system_setting_category_key"},
		{&LibraryPath{}, "idx_library_paths_library_source_path"},
		{&IngestDirtyUnit{}, "idx_ingest_dirty_units_claim"},
		{&IngestDirtyUnit{}, "idx_ingest_dirty_units_library"},
		{&IngestDirtyUnit{}, "idx_ingest_dirty_units_scope"},
		{&IngestCondition{}, "idx_ingest_conditions_unit_type"},
		{&IngestCondition{}, "idx_ingest_conditions_type_status"},
		{&IngestCondition{}, "idx_ingest_conditions_library_status"},
		{&IngestCondition{}, "idx_ingest_conditions_unit_status"},
		{&IngestCondition{}, "idx_ingest_conditions_retry_due"},
		{&IngestEvent{}, "idx_ingest_events_unit_created"},
		{&IngestEvent{}, "idx_ingest_events_library_created"},
		{&IngestEvent{}, "idx_ingest_events_condition_created"},
		{&WorkflowRun{}, "idx_workflow_runs_library_status"},
		{&WorkflowRun{}, "idx_workflow_runs_status_priority_created"},
		{&WorkflowTask{}, "idx_workflow_tasks_ready"},
		{&WorkflowTask{}, "idx_workflow_tasks_run_status"},
		{&WorkflowTask{}, "idx_workflow_tasks_library_status"},
		{&WorkflowTask{}, "idx_workflow_tasks_stage_status"},
		{&WorkflowTask{}, "idx_workflow_tasks_lease"},
		{&WorkflowTaskDependency{}, "idx_workflow_task_dependencies_pair"},
		{&WorkflowTaskDependency{}, "idx_workflow_task_dependencies_task"},
		{&WorkflowTaskDependency{}, "idx_workflow_task_dependencies_depends"},
		{&WorkflowTaskLease{}, "idx_workflow_task_leases_owner_until"},
		{&WorkflowResourceUsage{}, "idx_workflow_resource_usage_resource_task"},
		{&WorkflowResourceUsage{}, "idx_workflow_resource_usage_resource"},
	}

	for _, index := range requiredIndexes {
		if !db.Migrator().HasIndex(index.model, index.name) {
			t.Fatalf("expected index %q to exist for %T", index.name, index.model)
		}
	}
}

func assertMediaStreamTechnicalColumnsExist(t *testing.T, db *gorm.DB) {
	t.Helper()

	requiredColumns := []string{
		"profile",
		"level",
		"avg_frame_rate",
		"r_frame_rate",
		"field_order",
		"color_space",
		"bit_depth",
		"pixel_format",
		"reference_frames",
		"channel_layout",
		"sample_rate",
		"bit_rate",
	}
	for _, column := range requiredColumns {
		if !db.Migrator().HasColumn(&MediaStream{}, column) {
			t.Fatalf("expected media_streams.%s column to exist", column)
		}
	}
}

func assertLegacyMetadataSchemaRemoved(t *testing.T, db *gorm.DB) {
	t.Helper()

	if db.Migrator().HasTable("library_metadata_profile_bindings") {
		t.Fatalf("expected legacy table library_metadata_profile_bindings to be removed")
	}
	for _, check := range []struct {
		table  string
		column string
	}{
		{table: "metadata_profiles", column: "local_only"},
		{table: "library_metadata_policies", column: "tmdb_enabled"},
		{table: "library_metadata_policies", column: "tvdb_enabled"},
		{table: "library_metadata_policies", column: "provider_priority_json"},
	} {
		if db.Migrator().HasColumn(check.table, check.column) {
			t.Fatalf("expected legacy column %s.%s to be removed", check.table, check.column)
		}
	}
}
