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
		&CatalogItem{},
		&CatalogExternalID{},
		&CatalogIdentity{},
		&MetadataSource{},
		&MetadataFieldState{},
		&MetadataOperation{},
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
		&ScanExclusion{},
		&FilenameExclusionRule{},
		&FilenameExclusionRestore{},
		&ScanExclusionRule{},
		&ClassificationDecision{},
		&ClassificationRule{},
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
	); err != nil {
		return nil, err
	}

	if err := validateCatalogKernelUniqueness(db); err != nil {
		return nil, err
	}

	if err := ensureCatalogKernelIndexes(db); err != nil {
		return nil, err
	}
	if err := BackfillLibraryPathsAndPolicies(db); err != nil {
		return nil, err
	}
	if err := BackfillMetadataProfiles(db); err != nil {
		return nil, err
	}
	if err := BackfillLibraryMetadataStrategies(db); err != nil {
		return nil, err
	}
	if err := cleanupLegacyMetadataSchema(db); err != nil {
		return nil, err
	}
	if err := BackfillMissingSince(db); err != nil {
		return nil, err
	}
	if err := BackfillInventoryScanState(db); err != nil {
		return nil, err
	}
	if err := BackfillIngestDirtyScopes(db); err != nil {
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

func BackfillMissingSince(db *gorm.DB) error {
	nowExpr := gorm.Expr("updated_at")
	if err := db.Model(&InventoryFile{}).
		Where("status = ? AND missing_since IS NULL", "missing").
		Update("missing_since", nowExpr).Error; err != nil {
		return err
	}
	if err := db.Model(&MediaAsset{}).
		Where("status = ? AND missing_since IS NULL", "missing").
		Update("missing_since", nowExpr).Error; err != nil {
		return err
	}
	return db.Model(&CatalogItem{}).
		Where("availability_status = ? AND missing_since IS NULL", "missing").
		Update("missing_since", nowExpr).Error
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

func ensureCatalogKernelIndexes(db *gorm.DB) error {
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
		{&InventoryFile{}, "idx_inventory_file_storage_path"},
		{&InventoryFile{}, "idx_inventory_files_library_status_path"},
		{&StorageIndexEntry{}, "idx_storage_index_identity"},
		{&StorageIndexEntry{}, "idx_storage_index_library_status_path"},
		{&StorageIndexEntry{}, "idx_storage_index_stable_identity"},
		{&StorageObservationFailure{}, "idx_storage_observation_failure_library_path"},
		{&ScanExclusion{}, "idx_scan_exclusions_identity"},
		{&ScanExclusion{}, "idx_scan_exclusions_path"},
		{&FilenameExclusionRule{}, "idx_filename_exclusion_rules_normalized_filename"},
		{&FilenameExclusionRestore{}, "idx_filename_exclusion_restores_identity"},
		{&FilenameExclusionRestore{}, "idx_filename_exclusion_restores_path"},
		{&ScanExclusionRule{}, "idx_scan_exclusion_rules_enabled_type"},
		{&ClassificationDecision{}, "idx_classification_decisions_library_status"},
		{&ClassificationRule{}, "idx_classification_rules_library_enabled"},
		{&MediaStream{}, "idx_media_stream_file_index"},
		{&UserItemData{}, "idx_user_item_data_user_item_asset"},
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

func BackfillLibraryPathsAndPolicies(db *gorm.DB) error {
	var libraries []Library
	if err := db.Find(&libraries).Error; err != nil {
		return err
	}
	for _, library := range libraries {
		if err := backfillLibraryPath(db, library); err != nil {
			return err
		}
		if err := EnsureLibraryPolicyDefaults(db, library.ID); err != nil {
			return err
		}
	}
	return nil
}

func BackfillMetadataProfiles(db *gorm.DB) error {
	if err := ensureMigratedMetadataProfiles(db); err != nil {
		return err
	}
	if err := ensureBuiltInLocalScanProviderInstance(db); err != nil {
		return err
	}
	return nil
}

func BackfillLibraryMetadataStrategies(db *gorm.DB) error {
	if err := ensureBuiltInLocalScanProviderInstance(db); err != nil {
		return err
	}
	var libraries []Library
	if err := db.Find(&libraries).Error; err != nil {
		return err
	}
	for _, library := range libraries {
		if err := EnsureLibraryMetadataStrategy(db, library.ID); err != nil {
			return err
		}
	}
	return nil
}

func backfillLibraryPath(db *gorm.DB, library Library) error {
	if library.ID == 0 || library.MediaSourceID == 0 || strings.TrimSpace(library.RootPath) == "" {
		return nil
	}
	var count int64
	if err := db.Model(&LibraryPath{}).Where("library_id = ?", library.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	path := LibraryPath{LibraryID: library.ID, MediaSourceID: library.MediaSourceID, RootPath: library.RootPath, DisplayName: library.Name, Enabled: true}
	return db.Create(&path).Error
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

func ensureMigratedMetadataProfiles(db *gorm.DB) error {
	instanceID, err := ensureMigratedTMDBProviderInstance(db)
	if err != nil {
		return err
	}
	providerList := []uint{}
	if instanceID != 0 {
		providerList = append(providerList, instanceID)
	}
	profiles := []MetadataProfile{
		{
			Name:                      MigratedDefaultOnlineProfileName,
			Description:               "Migrated default metadata profile backed by the legacy TMDB configuration.",
			SearchProvidersJSON:       mustJSON(providerList),
			DetailProvidersJSON:       mustJSON(providerList),
			ImageProvidersJSON:        mustJSON(providerList),
			PeopleProvidersJSON:       mustJSON(providerList),
			HierarchyProvidersJSON:    mustJSON(providerList),
			FallbackEnabled:           true,
			PreferredMetadataLanguage: strings.TrimSpace(loadSystemSettingValue(db, metadataCategory, tmdbLanguageKey)),
		},
		{
			Name:            MigratedDefaultLocalProfileName,
			FallbackEnabled: false,
		},
	}
	for _, profile := range profiles {
		record := profile
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"description", "search_providers_json", "detail_providers_json", "image_providers_json", "people_providers_json", "hierarchy_providers_json", "preferred_metadata_language", "preferred_image_language", "fallback_enabled", "updated_at"}),
		}).Create(&record).Error; err != nil {
			return err
		}
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
	policy := LibraryMetadataPolicy{LibraryID: libraryID, LocalMetadataEnabled: true}
	if err := db.Where("library_id = ?", libraryID).First(&policy).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	legacyBinding := legacyLibraryMetadataProfileBinding{LibraryID: libraryID}
	if db.Migrator().HasTable("library_metadata_profile_bindings") {
		if err := db.Table("library_metadata_profile_bindings").Where("library_id = ?", libraryID).First(&legacyBinding).Error; err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
	}
	if legacyBinding.ForceLocalOnly {
		legacyBinding.MetadataProfileID = legacyLocalOnlyProfileID(db)
	}
	profile := MetadataProfile{}
	if legacyBinding.MetadataProfileID != 0 {
		if err := db.First(&profile, legacyBinding.MetadataProfileID).Error; err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
	}
	if profile.ID == 0 {
		profile = legacyDefaultProfileForPolicy(db, loadLegacyMetadataPolicy(db, libraryID, policy))
		if profile.ID != 0 {
			legacyBinding.MetadataProfileID = profile.ID
		}
	}
	localScanID, tmdbIDs, err := resolveBackfillStrategyProviderIDs(db)
	if err != nil {
		return err
	}
	legacyPolicy := loadLegacyMetadataPolicy(db, libraryID, policy)
	searchProviders, detailProviders, imageProviders, peopleProviders, hierarchyProviders := backfillStrategyProviders(profile, legacyPolicy, localScanID, tmdbIDs)
	strategy := LibraryMetadataStrategy{
		LibraryID:                 libraryID,
		MetadataProfileID:         uintPtrOrNil(legacyBinding.MetadataProfileID),
		SearchProvidersJSON:       mustJSON(searchProviders),
		DetailProvidersJSON:       mustJSON(detailProviders),
		ImageProvidersJSON:        mustJSON(imageProviders),
		PeopleProvidersJSON:       mustJSON(peopleProviders),
		HierarchyProvidersJSON:    mustJSON(hierarchyProviders),
		PreferredMetadataLanguage: strings.TrimSpace(firstNonEmpty(legacyBinding.PreferredMetadataLanguage, profile.PreferredMetadataLanguage, policy.PreferredMetadataLanguage)),
		PreferredImageLanguage:    strings.TrimSpace(firstNonEmpty(legacyBinding.PreferredImageLanguage, profile.PreferredImageLanguage, policy.PreferredImageLanguage)),
		MetadataCountryCode:       strings.TrimSpace(policy.MetadataCountryCode),
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "library_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"metadata_profile_id", "search_providers_json", "detail_providers_json", "image_providers_json", "people_providers_json", "hierarchy_providers_json", "preferred_metadata_language", "preferred_image_language", "metadata_country_code", "updated_at"}),
	}).Create(&strategy).Error
}

type legacyLibraryMetadataPolicy struct {
	TMDBEnabled bool
}

type legacyLibraryMetadataProfileBinding struct {
	LibraryID                 uint
	MetadataProfileID         uint
	PreferredMetadataLanguage string
	PreferredImageLanguage    string
	ForceLocalOnly            bool
}

func loadLegacyMetadataPolicy(db *gorm.DB, libraryID uint, fallback LibraryMetadataPolicy) legacyLibraryMetadataPolicy {
	legacy := legacyLibraryMetadataPolicy{TMDBEnabled: true}
	if !db.Migrator().HasColumn("library_metadata_policies", "tmdb_enabled") {
		return legacy
	}
	if err := db.Table("library_metadata_policies").Where("library_id = ?", libraryID).Take(&legacy).Error; err == nil {
		return legacy
	}
	return legacy
}

func cleanupLegacyMetadataSchema(db *gorm.DB) error {
	if db.Migrator().HasTable("library_metadata_profile_bindings") {
		if err := db.Migrator().DropTable("library_metadata_profile_bindings"); err != nil {
			return err
		}
	}
	for _, column := range []string{"local_only"} {
		if db.Migrator().HasColumn("metadata_profiles", column) {
			if err := dropLegacyColumnIndexes(db, "metadata_profiles", column); err != nil {
				return err
			}
			if err := dropLegacyColumn(db, "metadata_profiles", column); err != nil {
				return err
			}
		}
	}
	for _, column := range []string{"tmdb_enabled", "tvdb_enabled", "provider_priority_json"} {
		if db.Migrator().HasColumn("library_metadata_policies", column) {
			if err := dropLegacyColumnIndexes(db, "library_metadata_policies", column); err != nil {
				return err
			}
			if err := dropLegacyColumn(db, "library_metadata_policies", column); err != nil {
				return err
			}
		}
	}
	return nil
}

func dropLegacyColumnIndexes(db *gorm.DB, table, column string) error {
	for _, index := range legacyColumnIndexes(table, column) {
		if err := db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %q", index)).Error; err != nil {
			return err
		}
	}
	return nil
}

func legacyColumnIndexes(table, column string) []string {
	switch table {
	case "metadata_profiles":
		if column == "local_only" {
			return []string{"idx_metadata_profiles_local_only"}
		}
	case "library_metadata_policies":
		switch column {
		case "tmdb_enabled":
			return []string{"idx_library_metadata_policies_tmdb_enabled"}
		case "tvdb_enabled":
			return []string{"idx_library_metadata_policies_tvdb_enabled"}
		case "provider_priority_json":
			return []string{"idx_library_metadata_policies_provider_priority_json"}
		}
	}
	return nil
}

func dropLegacyColumn(db *gorm.DB, table, column string) error {
	if !isKnownLegacyMetadataColumn(table, column) {
		return fmt.Errorf("refusing to drop unknown legacy metadata column %s.%s", table, column)
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE %q DROP COLUMN %q", table, column)).Error
}

func isKnownLegacyMetadataColumn(table, column string) bool {
	switch table {
	case "metadata_profiles":
		return column == "local_only"
	case "library_metadata_policies":
		switch column {
		case "tmdb_enabled", "tvdb_enabled", "provider_priority_json":
			return true
		}
	}
	return false
}

func legacyLocalOnlyProfileID(db *gorm.DB) uint {
	var profile MetadataProfile
	if err := db.Where("name = ?", MigratedDefaultLocalProfileName).First(&profile).Error; err != nil {
		return 0
	}
	return profile.ID
}

func legacyDefaultProfileForPolicy(db *gorm.DB, policy legacyLibraryMetadataPolicy) MetadataProfile {
	profileName := MigratedDefaultOnlineProfileName
	if !policy.TMDBEnabled {
		profileName = MigratedDefaultLocalProfileName
	}
	var profile MetadataProfile
	if err := db.Where("name = ?", profileName).First(&profile).Error; err != nil {
		return MetadataProfile{}
	}
	return profile
}

func resolveBackfillStrategyProviderIDs(db *gorm.DB) (uint, []uint, error) {
	var localScan MetadataProviderInstance
	if err := db.Where("name = ?", BuiltInLocalScanProviderInstanceName).First(&localScan).Error; err != nil {
		return 0, nil, err
	}
	var tmdbProviders []MetadataProviderInstance
	if err := db.Where("provider_type = ? AND enabled = ?", MetadataProviderTypeTMDB, true).Order("id asc").Find(&tmdbProviders).Error; err != nil {
		return 0, nil, err
	}
	ids := make([]uint, 0, len(tmdbProviders))
	for _, provider := range tmdbProviders {
		ids = append(ids, provider.ID)
	}
	return localScan.ID, ids, nil
}

func backfillStrategyProviders(profile MetadataProfile, policy legacyLibraryMetadataPolicy, localScanID uint, tmdbIDs []uint) ([]uint, []uint, []uint, []uint, []uint) {
	searchProviders := filterKnownProviderIDs(unmarshalUintList(profile.SearchProvidersJSON), tmdbIDs)
	detailProviders := filterKnownProviderIDs(unmarshalUintList(profile.DetailProvidersJSON), tmdbIDs)
	imageProviders := filterKnownProviderIDs(unmarshalUintList(profile.ImageProvidersJSON), tmdbIDs)
	peopleProviders := filterKnownProviderIDs(unmarshalUintList(profile.PeopleProvidersJSON), tmdbIDs)
	hierarchyProviders := filterKnownProviderIDs(unmarshalUintList(profile.HierarchyProvidersJSON), tmdbIDs)
	if strings.TrimSpace(profile.Name) == MigratedDefaultLocalProfileName || !policy.TMDBEnabled {
		searchProviders = []uint{}
		imageProviders = []uint{}
		peopleProviders = []uint{}
		hierarchyProviders = []uint{}
		detailProviders = []uint{localScanID}
		return searchProviders, detailProviders, imageProviders, peopleProviders, hierarchyProviders
	}
	if len(searchProviders) == 0 {
		searchProviders = append([]uint(nil), tmdbIDs...)
	}
	if len(detailProviders) == 0 {
		detailProviders = append([]uint(nil), tmdbIDs...)
	}
	if len(imageProviders) == 0 {
		imageProviders = append([]uint(nil), tmdbIDs...)
	}
	if len(peopleProviders) == 0 {
		peopleProviders = append([]uint(nil), tmdbIDs...)
	}
	if len(hierarchyProviders) == 0 {
		hierarchyProviders = append([]uint(nil), tmdbIDs...)
	}
	return searchProviders, detailProviders, imageProviders, peopleProviders, hierarchyProviders
}

func filterKnownProviderIDs(candidate []uint, allowed []uint) []uint {
	if len(candidate) == 0 || len(allowed) == 0 {
		return []uint{}
	}
	allowedSet := make(map[uint]struct{}, len(allowed))
	for _, id := range allowed {
		allowedSet[id] = struct{}{}
	}
	filtered := make([]uint, 0, len(candidate))
	for _, id := range candidate {
		if _, ok := allowedSet[id]; ok {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func uintPtrOrNil(value uint) *uint {
	if value == 0 {
		return nil
	}
	copy := value
	return &copy
}

func unmarshalUintList(raw string) []uint {
	var values []uint
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return []uint{}
	}
	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func ensureMigratedTMDBProviderInstance(db *gorm.DB) (uint, error) {
	config := map[string]string{}
	for _, key := range []string{tmdbAPIKeyKey, tmdbBaseURLKey, tmdbImageBaseURLKey, tmdbLanguageKey, tmdbTimeoutKey} {
		if value := strings.TrimSpace(loadSystemSettingValue(db, metadataCategory, key)); value != "" {
			config[key] = value
		}
	}
	var existing MetadataProviderInstance
	if err := db.Where("name = ?", MigratedDefaultTMDBProviderInstanceName).First(&existing).Error; err == nil {
		if len(config) == 0 {
			if strings.TrimSpace(existing.ConfigJSON) == "" || strings.TrimSpace(existing.ConfigJSON) == "{}" {
				return 0, nil
			}
			return existing.ID, nil
		}
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return 0, err
	}
	record := MetadataProviderInstance{
		Name:               MigratedDefaultTMDBProviderInstanceName,
		ProviderType:       MetadataProviderTypeTMDB,
		Enabled:            true,
		AvailabilityStatus: MetadataProviderAvailabilityAvailable,
		ConfigJSON:         mustJSON(config),
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider_type", "enabled", "availability_status", "config_json", "updated_at"}),
	}).Create(&record).Error; err != nil {
		return 0, err
	}
	var stored MetadataProviderInstance
	if err := db.Where("name = ?", MigratedDefaultTMDBProviderInstanceName).First(&stored).Error; err != nil {
		return 0, err
	}
	return stored.ID, nil
}

func loadSystemSettingValue(db *gorm.DB, category, key string) string {
	var value string
	_ = db.Model(&SystemSetting{}).Where("category = ? AND key = ?", category, key).Select("value").Scan(&value).Error
	return strings.TrimSpace(value)
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
			label:      "catalog external identity",
			table:      "catalog_external_ids",
			columns:    []string{"provider", "provider_type", "external_id"},
			groupBy:    "provider, provider_type, external_id",
			selectExpr: "provider || '|' || provider_type || '|' || external_id",
		},
		{
			label:      "catalog identity",
			table:      "catalog_identities",
			columns:    []string{"provider", "identity_type", "identity_key"},
			groupBy:    "provider, identity_type, identity_key",
			selectExpr: "provider || '|' || identity_type || '|' || identity_key",
		},
		{
			label:      "metadata field state",
			table:      "metadata_field_states",
			columns:    []string{"item_id", "field_key"},
			groupBy:    "item_id, field_key",
			selectExpr: "CAST(item_id AS TEXT) || '|' || field_key",
		},
		{
			label:      "asset item link",
			table:      "asset_items",
			columns:    []string{"asset_id", "item_id", "role", "segment_index"},
			groupBy:    "asset_id, item_id, role, segment_index",
			selectExpr: "CAST(asset_id AS TEXT) || '|' || CAST(item_id AS TEXT) || '|' || role || '|' || CAST(segment_index AS TEXT)",
		},
		{
			label:      "asset file link",
			table:      "asset_files",
			columns:    []string{"asset_id", "file_id", "role", "part_index"},
			groupBy:    "asset_id, file_id, role, part_index",
			selectExpr: "CAST(asset_id AS TEXT) || '|' || CAST(file_id AS TEXT) || '|' || role || '|' || CAST(part_index AS TEXT)",
		},
		{
			label:      "inventory file storage path",
			table:      "inventory_files",
			columns:    []string{"storage_provider", "storage_path"},
			where:      "deleted_at IS NULL",
			groupBy:    "storage_provider, storage_path",
			selectExpr: "storage_provider || '|' || storage_path",
		},
		{
			label:      "media stream index",
			table:      "media_streams",
			columns:    []string{"file_id", "stream_index"},
			groupBy:    "file_id, stream_index",
			selectExpr: "CAST(file_id AS TEXT) || '|' || CAST(stream_index AS TEXT)",
		},
		{
			label:      "user item data identity",
			table:      "user_item_data",
			columns:    []string{"user_id", "item_id", "asset_id"},
			groupBy:    "user_id, item_id, asset_id",
			selectExpr: "CAST(user_id AS TEXT) || '|' || CAST(item_id AS TEXT) || '|' || COALESCE(CAST(asset_id AS TEXT), 'null')",
		},
		{
			label:      "system setting category key",
			table:      "system_settings",
			columns:    []string{"category", "key"},
			groupBy:    "category, key",
			selectExpr: "category || '|' || key",
		},
	}

	for _, check := range checks {
		if !db.Migrator().HasTable(check.table) {
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
