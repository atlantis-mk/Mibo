package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		if err := ensureSQLiteDir(cfg.DSN); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, err
	}
	if cfg.Driver == "sqlite" {
		if err := configureSQLite(db); err != nil {
			return nil, err
		}
	}

	if err := db.AutoMigrate(
		&MediaSource{},
		&Library{},
		&LibraryPath{},
		&LibraryScanPolicy{},
		&LibraryMetadataPolicy{},
		&MetadataProviderInstance{},
		&MetadataProfile{},
		&LibraryMetadataStrategy{},
		&LibraryPlaybackPolicy{},
		&LibrarySubtitlePolicy{},
		&MetadataOperation{},
		&Person{},
		&ItemPerson{},
		&Tag{},
		&ItemTag{},
		&InventoryFile{},
		&InventoryFileSignal{},
		&InventorySidecarSignal{},
		&StorageIndexEntry{},
		&StorageObservationFailure{},
		&StorageDirectoryFingerprint{},
		&ContentShapeProfile{},
		&ContentShapePlan{},
		&ContentShapeAssignment{},
		&RecognitionManifest{},
		&RecognitionCandidate{},
		&RecognitionEvidence{},
		&RecognitionDecision{},
		&RecognitionConflict{},
		&MediaGraphNode{},
		&MediaGraphEdge{},
		&MediaGraphClassification{},
		&ScanExclusion{},
		&FilenameExclusionRule{},
		&FilenameExclusionRestore{},
		&ScanExclusionRule{},
		&ClassificationDecision{},
		&ClassificationRule{},
		&MediaStream{},
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
	); err != nil {
		return nil, err
	}
	if cfg.Driver == "sqlite" {
		if err := repairSQLiteConflictIndexes(db); err != nil {
			return nil, err
		}
	}

	if err := removeRetiredDevelopmentColumns(db); err != nil {
		return nil, err
	}

	if err := removeRetiredDevelopmentTables(db); err != nil {
		return nil, err
	}

	if err := validateCatalogKernelUniqueness(db); err != nil {
		return nil, err
	}

	if err := ensureCatalogKernelIndexes(db); err != nil {
		return nil, err
	}

	return db, nil
}

func BackfillIngestDirtyScopes(db *gorm.DB) error {
	var paths []LibraryPath
	if err := db.Where("enabled = ? AND deleted_at IS NULL", true).Find(&paths).Error; err != nil {
		return err
	}
	for _, pathRecord := range paths {
		if pathRecord.LibraryID == 0 || strings.TrimSpace(pathRecord.RootPath) == "" {
			continue
		}
		now := time.Now().UTC()
		unit := IngestDirtyUnit{
			DirtyKey:    fmt.Sprintf("library:%d:%s", pathRecord.LibraryID, strings.TrimSpace(pathRecord.RootPath)),
			ScopeKind:   "library",
			LibraryID:   pathRecord.LibraryID,
			RootPath:    strings.TrimSpace(pathRecord.RootPath),
			Reason:      "startup_backfill",
			Status:      "dirty",
			AvailableAt: now,
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dirty_key"}},
			DoNothing: true,
		}).Create(&unit).Error; err != nil {
			return err
		}
	}
	return nil
}

func BackfillInventoryScanState(db *gorm.DB) error {
	return db.Model(&InventoryFile{}).
		Where("scan_state = '' OR scan_state IS NULL").
		Update("scan_state", "discovered").Error
}

func configureSQLite(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	// SQLite permits only one writer. Serializing pooled connections prevents
	// concurrent worker writes from surfacing as SQLITE_BUSY during scans.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := db.Exec("PRAGMA busy_timeout = 10000").Error; err != nil {
		return err
	}
	return db.Exec("PRAGMA journal_mode = WAL").Error
}

func removeRetiredDevelopmentColumns(db *gorm.DB) error {
	retiredColumns := []struct {
		model  any
		column string
	}{
		{model: &ClassificationDecision{}, column: "asset_id"},
		{model: &ClassificationDecision{}, column: "item_id"},
		{model: &IngestDirtyUnit{}, column: "catalog_item_id"},
		{model: &IngestCondition{}, column: "catalog_item_id"},
		{model: &IngestEvent{}, column: "catalog_item_id"},
	}
	for _, retired := range retiredColumns {
		if !db.Migrator().HasColumn(retired.model, retired.column) {
			continue
		}
		if err := db.Migrator().DropColumn(retired.model, retired.column); err != nil {
			return fmt.Errorf("drop retired column %s: %w", retired.column, err)
		}
	}
	return nil
}

func removeRetiredDevelopmentTables(db *gorm.DB) error {
	retiredTables := []string{
		"recognition_shadow_runs",
		"recognition_rules",
	}
	for _, table := range retiredTables {
		if !db.Migrator().HasTable(table) {
			continue
		}
		if err := db.Migrator().DropTable(table); err != nil {
			return fmt.Errorf("drop retired table %s: %w", table, err)
		}
	}
	return nil
}

func ensureCatalogKernelIndexes(db *gorm.DB) error {
	requiredIndexes := []struct {
		model any
		name  string
	}{
		{&MetadataItem{}, "idx_metadata_items_type_status_sort"},
		{&MetadataItem{}, "idx_metadata_items_parent_order"},
		{&MetadataItem{}, "idx_metadata_items_root_type_order"},
		{&MetadataExternalID{}, "idx_metadata_external_identity"},
		{&MetadataItemFieldState{}, "idx_metadata_item_field_state_identity"},
		{&MetadataItemImage{}, "idx_metadata_item_images_selected"},
		{&MetadataItemPerson{}, "idx_metadata_item_people_identity"},
		{&MetadataItemTag{}, "idx_metadata_item_tags_identity"},
		{&Resource{}, "idx_resources_stable_resource_key"},
		{&ResourceFile{}, "idx_resource_files_resource_file_role_part"},
		{&ResourceFile{}, "idx_resource_files_resource_part"},
		{&ResourceLibraryLink{}, "idx_resource_library_link_identity"},
		{&ResourceLibraryLink{}, "idx_resource_library_links_library_status"},
		{&ResourceMetadataLink{}, "idx_resource_metadata_link_identity"},
		{&ResourceMetadataLink{}, "idx_resource_metadata_links_item_role"},
		{&LibraryMetadataProjection{}, "idx_library_metadata_projection_identity"},
		{&LibraryMetadataProjection{}, "idx_library_metadata_projections_library_type_availability_title"},
		{&LibrarySearchDocument{}, "idx_library_search_documents_library_type_availability_title"},
		{&UserMetadataData{}, "idx_user_metadata_data_identity"},
		{&UserResourceData{}, "idx_user_resource_data_identity"},
		{&InventoryFile{}, "idx_inventory_file_source_storage_path"},
		{&InventoryFile{}, "idx_inventory_files_library_status_path"},
		{&InventoryFileSignal{}, "idx_inventory_file_signal_identity"},
		{&InventoryFileSignal{}, "idx_inventory_file_signals_library_state"},
		{&InventorySidecarSignal{}, "idx_inventory_sidecar_signal_identity"},
		{&InventorySidecarSignal{}, "idx_inventory_sidecar_signals_library_state"},
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
		{&RecognitionManifest{}, "idx_recognition_manifest_key"},
		{&RecognitionManifest{}, "idx_recognition_manifest_scope"},
		{&RecognitionManifest{}, "idx_recognition_manifests_library_state"},
		{&RecognitionCandidate{}, "idx_recognition_candidate_identity"},
		{&RecognitionCandidate{}, "idx_recognition_candidates_type_state"},
		{&MediaGraphNode{}, "idx_media_graph_node_identity"},
		{&MediaGraphNode{}, "idx_media_graph_nodes_kind_state"},
		{&MediaGraphEdge{}, "idx_media_graph_edge_identity"},
		{&MediaGraphClassification{}, "idx_media_graph_classification_identity"},
		{&MediaGraphClassification{}, "idx_media_graph_classifications_state"},
		{&RecognitionEvidence{}, "idx_recognition_evidence_manifest_candidate"},
		{&RecognitionEvidence{}, "idx_recognition_evidence_kind"},
		{&RecognitionDecision{}, "idx_recognition_decisions_manifest_status"},
		{&RecognitionConflict{}, "idx_recognition_conflicts_manifest_status"},
		{&ScanExclusion{}, "idx_scan_exclusions_identity"},
		{&ScanExclusion{}, "idx_scan_exclusions_path"},
		{&FilenameExclusionRule{}, "idx_filename_exclusion_rules_normalized_filename"},
		{&FilenameExclusionRestore{}, "idx_filename_exclusion_restores_identity"},
		{&FilenameExclusionRestore{}, "idx_filename_exclusion_restores_path"},
		{&ScanExclusionRule{}, "idx_scan_exclusion_rules_enabled_type"},
		{&ClassificationDecision{}, "idx_classification_decisions_library_status"},
		{&ClassificationRule{}, "idx_classification_rules_library_enabled"},
		{&MediaStream{}, "idx_media_stream_file_index"},
		{&SystemSetting{}, "idx_system_setting_category_key"},
		{&LibraryPath{}, "idx_library_paths_library_source_path"},
		{&MetadataProviderInstance{}, "idx_metadata_provider_instances_provider_type"},
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
	}

	for _, index := range requiredIndexes {
		if db.Migrator().HasIndex(index.model, index.name) {
			continue
		}

		if err := db.Migrator().CreateIndex(index.model, index.name); err != nil {
			return fmt.Errorf("create index %s: %w", index.name, err)
		}
	}

	return nil
}

func EnsureBuiltInMetadataProviders(db *gorm.DB) error {
	return ensureBuiltInLocalScanProviderInstance(db)
}

func EnsureLibraryPolicyDefaults(db *gorm.DB, libraryID uint) error {
	if libraryID == 0 {
		return nil
	}
	var count int64
	if err := db.Model(&LibraryScanPolicy{}).Where("library_id = ?", libraryID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := db.Create(&LibraryScanPolicy{LibraryID: libraryID, ScannerEnabled: true, RealtimeMonitorEnabled: true, ScheduledRefreshEnabled: true, RefreshIntervalHours: 24, IgnoreHiddenFiles: true, IgnoreFileExtensionsJSON: "[]", InventoryProbeBatchEnabled: true, ConfigurableExclusionRules: true}).Error; err != nil {
			return err
		}
	}
	count = 0
	if err := db.Model(&LibraryMetadataPolicy{}).Where("library_id = ?", libraryID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := db.Create(&LibraryMetadataPolicy{LibraryID: libraryID, LocalMetadataEnabled: true}).Error; err != nil {
			return err
		}
	}
	count = 0
	if err := db.Model(&LibraryPlaybackPolicy{}).Where("library_id = ?", libraryID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := db.Create(&LibraryPlaybackPolicy{LibraryID: libraryID, ResumeEnabled: true, MinResumePct: 5, MaxResumePct: 90, MinResumeDurationSeconds: 300}).Error; err != nil {
			return err
		}
	}
	count = 0
	if err := db.Model(&LibrarySubtitlePolicy{}).Where("library_id = ?", libraryID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return db.Create(&LibrarySubtitlePolicy{LibraryID: libraryID, ExternalSidecarsEnabled: true, PreferredLanguagesJSON: "[]", TolerateUnavailableSubtitles: true}).Error
	}
	return nil
}

func ensureBuiltInLocalScanProviderInstance(db *gorm.DB) error {
	record := MetadataProviderInstance{
		Name:               BuiltInLocalScanProviderInstanceName,
		ProviderType:       MetadataProviderTypeLocalScan,
		Enabled:            true,
		AvailabilityStatus: MetadataProviderAvailabilityAvailable,
		ConfigJSON:         "{}",
		SystemManaged:      true,
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider_type", "enabled", "availability_status", "config_json", "system_managed", "updated_at"}),
	}).Create(&record).Error
}

func EnsureLibraryMetadataStrategy(db *gorm.DB, libraryID uint) error {
	if libraryID == 0 {
		return nil
	}
	if err := ensureBuiltInLocalScanProviderInstance(db); err != nil {
		return err
	}
	var count int64
	if err := db.Model(&LibraryMetadataStrategy{}).Where("library_id = ?", libraryID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	localScan := MetadataProviderInstance{}
	if err := db.Where("name = ?", BuiltInLocalScanProviderInstanceName).First(&localScan).Error; err != nil {
		return err
	}
	strategy := LibraryMetadataStrategy{
		LibraryID:           libraryID,
		DetailProvidersJSON: mustJSON([]uint{localScan.ID}),
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "library_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"metadata_profile_id", "search_providers_json", "detail_providers_json", "image_providers_json", "people_providers_json", "hierarchy_providers_json", "preferred_metadata_language", "preferred_image_language", "metadata_country_code", "updated_at"}),
	}).Create(&strategy).Error
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(data)
}

const (
	metadataCategory    = "metadata"
	tmdbAPIKeyKey       = "tmdb_api_key"
	tmdbBaseURLKey      = "tmdb_base_url"
	tmdbImageBaseURLKey = "tmdb_image_base_url"
	tmdbLanguageKey     = "tmdb_language"
	tmdbTimeoutKey      = "tmdb_timeout"
)

func repairSQLiteConflictIndexes(db *gorm.DB) error {
	checks := []struct {
		model           any
		indexName       string
		expectedColumns []string
		expectUnique    bool
	}{
		{model: &ContentShapePlan{}, indexName: "idx_content_shape_plan_scope", expectedColumns: []string{"library_id", "storage_provider", "root_path", "directory_path", "classifier_version"}, expectUnique: true},
	}
	for _, check := range checks {
		if err := ensureSQLiteIndexDefinition(db, check.model, check.indexName, check.expectedColumns, check.expectUnique); err != nil {
			return err
		}
	}
	return nil
}

func ensureSQLiteIndexDefinition(db *gorm.DB, model any, indexName string, expectedColumns []string, expectUnique bool) error {
	tableName, err := sqliteTableNameForModel(db, model)
	if err != nil {
		return err
	}
	exists, unique, columns, err := sqliteIndexDefinition(db, tableName, indexName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if unique == expectUnique && sameStringSlice(columns, expectedColumns) {
		return nil
	}
	if err := db.Migrator().DropIndex(model, indexName); err != nil {
		return fmt.Errorf("drop stale sqlite index %s: %w", indexName, err)
	}
	if err := db.Migrator().CreateIndex(model, indexName); err != nil {
		return fmt.Errorf("recreate sqlite index %s: %w", indexName, err)
	}
	exists, unique, columns, err = sqliteIndexDefinition(db, tableName, indexName)
	if err != nil {
		return err
	}
	if !exists || unique != expectUnique || !sameStringSlice(columns, expectedColumns) {
		return fmt.Errorf("sqlite index %s has unexpected definition after repair: unique=%t columns=%v", indexName, unique, columns)
	}
	return nil
}

func sqliteTableNameForModel(db *gorm.DB, model any) (string, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return "", err
	}
	if stmt.Schema == nil || strings.TrimSpace(stmt.Schema.Table) == "" {
		return "", fmt.Errorf("resolve sqlite table name for %T", model)
	}
	return stmt.Schema.Table, nil
}

func sqliteIndexDefinition(db *gorm.DB, tableName string, indexName string) (bool, bool, []string, error) {
	type sqliteIndexListRow struct {
		Name   string `gorm:"column:name"`
		Unique int    `gorm:"column:unique"`
	}
	var indexes []sqliteIndexListRow
	if err := db.Raw(fmt.Sprintf("PRAGMA index_list(%s)", sqliteQuotedIdentifier(tableName))).Scan(&indexes).Error; err != nil {
		return false, false, nil, fmt.Errorf("list sqlite indexes for %s: %w", tableName, err)
	}
	for _, index := range indexes {
		if index.Name != indexName {
			continue
		}
		type sqliteIndexInfoRow struct {
			Seqno int    `gorm:"column:seqno"`
			Name  string `gorm:"column:name"`
		}
		var infoRows []sqliteIndexInfoRow
		if err := db.Raw(fmt.Sprintf("PRAGMA index_info(%s)", sqliteQuotedIdentifier(indexName))).Scan(&infoRows).Error; err != nil {
			return false, false, nil, fmt.Errorf("describe sqlite index %s: %w", indexName, err)
		}
		columns := make([]string, 0, len(infoRows))
		for _, row := range infoRows {
			columns = append(columns, row.Name)
		}
		return true, index.Unique == 1, columns, nil
	}
	return false, false, nil, nil
}

func sqliteQuotedIdentifier(value string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(value), `"`, `""`) + `"`
}

func sameStringSlice(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func validateCatalogKernelUniqueness(db *gorm.DB) error {
	checks := []struct {
		label      string
		table      string
		where      string
		columns    []string
		groupBy    string
		selectExpr string
	}{
		{
			label:      "inventory file storage path",
			table:      "inventory_files",
			columns:    []string{"media_source_id", "storage_provider", "storage_path"},
			where:      "deleted_at IS NULL",
			groupBy:    "media_source_id, storage_provider, storage_path",
			selectExpr: "CAST(media_source_id AS TEXT) || '|' || storage_provider || '|' || storage_path",
		},
		{
			label:      "media stream index",
			table:      "media_streams",
			columns:    []string{"file_id", "stream_index"},
			groupBy:    "file_id, stream_index",
			selectExpr: "CAST(file_id AS TEXT) || '|' || CAST(stream_index AS TEXT)",
		},
		{
			label:      "system setting category key",
			table:      "system_settings",
			columns:    []string{"category", "key"},
			groupBy:    "category, key",
			selectExpr: "category || '|' || key",
		},
		{
			label:      "metadata external identity",
			table:      "metadata_external_ids",
			columns:    []string{"provider", "provider_type", "external_id"},
			groupBy:    "provider, provider_type, external_id",
			selectExpr: "provider || '|' || provider_type || '|' || external_id",
		},
		{
			label:      "metadata item field state",
			table:      "metadata_item_field_states",
			columns:    []string{"metadata_item_id", "field_key", "locale"},
			groupBy:    "metadata_item_id, field_key, locale",
			selectExpr: "CAST(metadata_item_id AS TEXT) || '|' || field_key || '|' || locale",
		},
		{
			label:      "resource file link",
			table:      "resource_files",
			columns:    []string{"resource_id", "inventory_file_id", "role", "part_index"},
			groupBy:    "resource_id, inventory_file_id, role, part_index",
			selectExpr: "CAST(resource_id AS TEXT) || '|' || CAST(inventory_file_id AS TEXT) || '|' || role || '|' || CAST(part_index AS TEXT)",
		},
		{
			label:      "resource library link",
			table:      "resource_library_links",
			columns:    []string{"resource_id", "library_id"},
			where:      "deleted_at IS NULL",
			groupBy:    "resource_id, library_id",
			selectExpr: "CAST(resource_id AS TEXT) || '|' || CAST(library_id AS TEXT)",
		},
		{
			label:      "resource metadata link",
			table:      "resource_metadata_links",
			columns:    []string{"resource_id", "metadata_item_id", "role", "segment_index"},
			groupBy:    "resource_id, metadata_item_id, role, segment_index",
			selectExpr: "CAST(resource_id AS TEXT) || '|' || CAST(metadata_item_id AS TEXT) || '|' || role || '|' || CAST(segment_index AS TEXT)",
		},
		{
			label:      "library metadata projection",
			table:      "library_metadata_projections",
			columns:    []string{"library_id", "metadata_item_id"},
			groupBy:    "library_id, metadata_item_id",
			selectExpr: "CAST(library_id AS TEXT) || '|' || CAST(metadata_item_id AS TEXT)",
		},
		{
			label:      "user metadata data identity",
			table:      "user_metadata_data",
			columns:    []string{"user_id", "metadata_item_id"},
			groupBy:    "user_id, metadata_item_id",
			selectExpr: "CAST(user_id AS TEXT) || '|' || CAST(metadata_item_id AS TEXT)",
		},
		{
			label:      "user resource data identity",
			table:      "user_resource_data",
			columns:    []string{"user_id", "resource_id", "metadata_item_id"},
			groupBy:    "user_id, resource_id, metadata_item_id",
			selectExpr: "CAST(user_id AS TEXT) || '|' || CAST(resource_id AS TEXT) || '|' || CAST(metadata_item_id AS TEXT)",
		},
	}

	for _, check := range checks {
		if !db.Migrator().HasTable(check.table) {
			continue
		}
		missingColumn := false
		for _, column := range check.columns {
			if !db.Migrator().HasColumn(check.table, column) {
				missingColumn = true
				break
			}
		}
		if missingColumn {
			continue
		}
		var duplicates []string
		query := db.Table(check.table).Select(check.selectExpr)
		if strings.TrimSpace(check.where) != "" {
			query = query.Where(check.where)
		}
		if err := query.Group(check.groupBy).Having("COUNT(*) > 1").Limit(3).Scan(&duplicates).Error; err != nil {
			return fmt.Errorf("check %s duplicates: %w", check.label, err)
		}
		if len(duplicates) > 0 {
			return fmt.Errorf("duplicate-prone %s rows block startup; sample keys: %s", check.label, strings.Join(duplicates, ", "))
		}
	}

	return nil
}

func ensureSQLiteDir(dsn string) error {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" || trimmed == ":memory:" || strings.HasPrefix(trimmed, "file:") {
		return nil
	}

	dir := filepath.Dir(trimmed)
	if dir == "." || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}
