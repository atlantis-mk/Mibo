package workflow

const (
	ResourceDBWrite      = "db_write"
	ResourceLocalDiskIO  = "local_disk_io"
	ResourceOpenListHTTP = "openlist_http"
	ResourceFFprobe      = "ffprobe"
	ResourceMetadataAPI  = "metadata_api"
	ResourceCPUHeavy     = "cpu_heavy"
)

const (
	TaskTypeDiscoverStorage     = "discover_storage"
	TaskTypeScanLibraryPath     = "scan_library_path"
	TaskTypeMaterializeCatalog  = "materialize_catalog"
	TaskTypeRefreshProjection   = "refresh_projection"
	TaskTypeProbeInventory      = "probe_inventory"
	TaskTypeProbeInventoryFile  = "probe_inventory_file"
	TaskTypeMatchMetadata       = "match_metadata"
	TaskTypeApplyStorageRefresh = "apply_storage_refresh"
)

const (
	StageScan           = "scan"
	StageMaterialize    = "materialize"
	StageProjection     = "projection"
	StageProbe          = "probe"
	StageMetadataMatch  = "metadata_match"
	StageCleanup        = "cleanup"
	StageStorageRefresh = "storage_refresh"
)

type TaskTypeDefinition struct {
	TaskType  string
	Stage     string
	Resources map[string]int
}

var defaultTaskTypes = map[string]TaskTypeDefinition{
	TaskTypeDiscoverStorage:     {TaskType: TaskTypeDiscoverStorage, Stage: StageScan, Resources: map[string]int{ResourceDBWrite: 1, ResourceLocalDiskIO: 1}},
	TaskTypeScanLibraryPath:     {TaskType: TaskTypeScanLibraryPath, Stage: StageScan, Resources: map[string]int{ResourceDBWrite: 1, ResourceLocalDiskIO: 1}},
	TaskTypeMaterializeCatalog:  {TaskType: TaskTypeMaterializeCatalog, Stage: StageMaterialize, Resources: map[string]int{ResourceDBWrite: 1, ResourceCPUHeavy: 1}},
	TaskTypeRefreshProjection:   {TaskType: TaskTypeRefreshProjection, Stage: StageProjection, Resources: map[string]int{ResourceDBWrite: 1}},
	TaskTypeProbeInventory:      {TaskType: TaskTypeProbeInventory, Stage: StageProbe, Resources: map[string]int{ResourceFFprobe: 1, ResourceLocalDiskIO: 1, ResourceCPUHeavy: 1, ResourceDBWrite: 1}},
	TaskTypeProbeInventoryFile:  {TaskType: TaskTypeProbeInventoryFile, Stage: StageProbe, Resources: map[string]int{ResourceFFprobe: 1, ResourceLocalDiskIO: 1, ResourceCPUHeavy: 1, ResourceDBWrite: 1}},
	TaskTypeMatchMetadata:       {TaskType: TaskTypeMatchMetadata, Stage: StageMetadataMatch, Resources: map[string]int{ResourceMetadataAPI: 1, ResourceDBWrite: 1}},
	TaskTypeApplyStorageRefresh: {TaskType: TaskTypeApplyStorageRefresh, Stage: StageStorageRefresh, Resources: map[string]int{ResourceDBWrite: 1}},
}

func DefaultTaskTypeDefinitions() map[string]TaskTypeDefinition {
	definitions := make(map[string]TaskTypeDefinition, len(defaultTaskTypes))
	for key, definition := range defaultTaskTypes {
		resources := make(map[string]int, len(definition.Resources))
		for resourceKey, units := range definition.Resources {
			resources[resourceKey] = units
		}
		definition.Resources = resources
		definitions[key] = definition
	}
	return definitions
}

func DefaultSQLiteResourceBudgets() map[string]int {
	return map[string]int{ResourceDBWrite: 1, ResourceLocalDiskIO: 2, ResourceOpenListHTTP: 2, ResourceFFprobe: 2, ResourceMetadataAPI: 1, ResourceCPUHeavy: 2}
}

func DefaultServerResourceBudgets() map[string]int {
	return map[string]int{ResourceDBWrite: 2, ResourceLocalDiskIO: 3, ResourceOpenListHTTP: 4, ResourceFFprobe: 4, ResourceMetadataAPI: 2, ResourceCPUHeavy: 3}
}
