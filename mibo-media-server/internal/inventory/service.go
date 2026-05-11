package inventory

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	bulkLookupChunkSize    = 400
	bulkFileWriteBatchSize = 50
	bulkLinkWriteBatchSize = 100
)

const (
	AssetTypeMain    = "main"
	AssetTypeVersion = "version"
	AssetTypeExtra   = "extra"
	AssetTypeTrailer = "trailer"
	AssetTypeSample  = "sample"

	AssetStatusAvailable = "available"
	AssetStatusMissing   = "missing"

	AssetItemRolePrimary          = "primary"
	AssetItemRoleVersion          = "version"
	AssetItemRoleMultiEpisodePart = "multi_episode_part"
	AssetItemRoleExtra            = "extra"
	AssetItemRoleTrailer          = "trailer"

	FileStatusAvailable         = "available"
	FileStatusMissing           = "missing"
	FileRoleSource              = "source"
	FileRoleSubtitle            = "subtitle"
	FileScanStateDiscovered     = "discovered"
	FileScanStateClassified     = "classified"
	FileScanStateEnriched       = "enriched"
	FileScanStateReviewRequired = "review_required"

	MediaStreamTypeSubtitle                 = "subtitle"
	MediaStreamDispositionExternalScanner   = "scanner"
	MediaStreamDispositionManagedByScanner  = "scanner"
	MediaStreamDispositionExternalAvailable = true
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type UpsertFileInput struct {
	LibraryID         uint
	MediaSourceID     uint
	StorageProvider   string
	StoragePath       string
	StableIdentityKey string
	HashesJSON        string
	ThumbnailURL      string
	SizeBytes         int64
	ModifiedAt        *time.Time
	Container         string
	ContentClass      string
	Status            string
	ScanState         string
}

type UpsertInventoryFileInput struct {
	MediaSourceID     uint
	LibraryID         uint
	StorageProvider   string
	StoragePath       string
	StableIdentityKey string
	HashesJSON        string
	ThumbnailURL      string
	SizeBytes         int64
	ModifiedAt        *time.Time
	Container         string
	ContentClass      string
	Status            string
	ScanState         string
}

type BulkUpsertFilesResult struct {
	FilesByStoragePath map[string]database.InventoryFile
	FilesBySourcePath  map[string]database.InventoryFile
}

type UpsertResourceInput struct {
	StableResourceKey    string
	ResourceType         string
	ResourceShape        string
	DisplayName          string
	Edition              string
	QualityLabel         string
	DurationSeconds      *float64
	Status               string
	ProbeStatus          string
	TechnicalSummaryJSON string
}

type LinkResourceFileInput struct {
	ResourceID      uint
	InventoryFileID uint
	Role            string
	PartIndex       int
}

type AttachResourceLibraryInput struct {
	ResourceID   uint
	LibraryID    uint
	Status       string
	ObservedAt   *time.Time
	EvidenceJSON string
	ReviewState  string
}

type LinkResourceMetadataInput struct {
	ResourceID     uint
	MetadataItemID uint
	Role           string
	SegmentIndex   int
	StartSeconds   *float64
	EndSeconds     *float64
	Confidence     *float64
	EvidenceJSON   string
	Source         string
	ReviewState    string
}

func (s *Service) UpsertFile(ctx context.Context, input UpsertFileInput) (database.InventoryFile, error) {
	if input.LibraryID == 0 && input.MediaSourceID == 0 {
		return database.InventoryFile{}, errors.New("library id or media source id is required")
	}
	if strings.TrimSpace(input.StorageProvider) == "" || strings.TrimSpace(input.StoragePath) == "" {
		return database.InventoryFile{}, errors.New("storage provider and path are required")
	}

	file := database.InventoryFile{
		LibraryID:         input.LibraryID,
		MediaSourceID:     input.MediaSourceID,
		StorageProvider:   strings.TrimSpace(input.StorageProvider),
		StoragePath:       strings.TrimSpace(input.StoragePath),
		StableIdentityKey: strings.TrimSpace(input.StableIdentityKey),
		HashesJSON:        input.HashesJSON,
		ThumbnailURL:      strings.TrimSpace(input.ThumbnailURL),
		SizeBytes:         input.SizeBytes,
		ModifiedAt:        input.ModifiedAt,
		Container:         strings.TrimSpace(input.Container),
		ContentClass:      defaultString(input.ContentClass, "video"),
		Status:            defaultString(input.Status, FileStatusAvailable),
		ScanState:         defaultString(input.ScanState, FileScanStateDiscovered),
	}
	if file.Status == FileStatusMissing {
		now := time.Now().UTC()
		file.MissingSince = &now
	}
	updateColumns := []string{"library_id", "media_source_id", "stable_identity_key", "hashes_json", "thumbnail_url", "size_bytes", "modified_at", "container", "content_class", "status", "scan_state", "updated_at"}
	if file.Status == FileStatusAvailable {
		updateColumns = append(updateColumns, "missing_since")
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "media_source_id"}, {Name: "storage_provider"}, {Name: "storage_path"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(&file).Error
	if err != nil {
		return database.InventoryFile{}, err
	}

	err = s.db.WithContext(ctx).Where("media_source_id = ? AND storage_provider = ? AND storage_path = ?", file.MediaSourceID, file.StorageProvider, file.StoragePath).First(&file).Error
	return file, err
}

func (s *Service) UpsertInventoryFile(ctx context.Context, input UpsertInventoryFileInput) (database.InventoryFile, error) {
	if input.MediaSourceID == 0 {
		return database.InventoryFile{}, errors.New("media source id is required")
	}
	return s.UpsertFile(ctx, UpsertFileInput{
		LibraryID:         input.LibraryID,
		MediaSourceID:     input.MediaSourceID,
		StorageProvider:   input.StorageProvider,
		StoragePath:       input.StoragePath,
		StableIdentityKey: input.StableIdentityKey,
		HashesJSON:        input.HashesJSON,
		ThumbnailURL:      input.ThumbnailURL,
		SizeBytes:         input.SizeBytes,
		ModifiedAt:        input.ModifiedAt,
		Container:         input.Container,
		ContentClass:      input.ContentClass,
		Status:            input.Status,
		ScanState:         input.ScanState,
	})
}

func (s *Service) BulkUpsertFiles(ctx context.Context, inputs []UpsertFileInput) (BulkUpsertFilesResult, error) {
	if len(inputs) == 0 {
		return BulkUpsertFilesResult{FilesByStoragePath: map[string]database.InventoryFile{}, FilesBySourcePath: map[string]database.InventoryFile{}}, nil
	}
	files := make([]database.InventoryFile, 0, len(inputs))
	lookupPathsByProvider := make(map[string][]string)
	seenPairs := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if input.LibraryID == 0 && input.MediaSourceID == 0 {
			return BulkUpsertFilesResult{}, errors.New("library id or media source id is required")
		}
		provider := strings.TrimSpace(input.StorageProvider)
		storagePath := strings.TrimSpace(input.StoragePath)
		if provider == "" || storagePath == "" {
			return BulkUpsertFilesResult{}, errors.New("storage provider and path are required")
		}
		file := database.InventoryFile{
			LibraryID:         input.LibraryID,
			MediaSourceID:     input.MediaSourceID,
			StorageProvider:   provider,
			StoragePath:       storagePath,
			StableIdentityKey: strings.TrimSpace(input.StableIdentityKey),
			HashesJSON:        input.HashesJSON,
			ThumbnailURL:      strings.TrimSpace(input.ThumbnailURL),
			SizeBytes:         input.SizeBytes,
			ModifiedAt:        input.ModifiedAt,
			Container:         strings.TrimSpace(input.Container),
			ContentClass:      defaultString(input.ContentClass, "video"),
			Status:            defaultString(input.Status, FileStatusAvailable),
			ScanState:         defaultString(input.ScanState, FileScanStateDiscovered),
		}
		if file.Status == FileStatusMissing {
			now := time.Now().UTC()
			file.MissingSince = &now
		}
		files = append(files, file)
		pairKey := inventoryFileLookupKey(file.MediaSourceID, provider, storagePath)
		if _, ok := seenPairs[pairKey]; ok {
			continue
		}
		seenPairs[pairKey] = struct{}{}
		lookupPathsByProvider[inventoryFileProviderLookupKey(file.MediaSourceID, provider)] = append(lookupPathsByProvider[inventoryFileProviderLookupKey(file.MediaSourceID, provider)], storagePath)
	}
	updateColumns := []string{"library_id", "media_source_id", "stable_identity_key", "hashes_json", "thumbnail_url", "size_bytes", "modified_at", "container", "content_class", "status", "scan_state", "missing_since", "updated_at"}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "media_source_id"}, {Name: "storage_provider"}, {Name: "storage_path"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).CreateInBatches(&files, bulkFileWriteBatchSize).Error; err != nil {
		return BulkUpsertFilesResult{}, err
	}
	var stored []database.InventoryFile
	for providerKey, lookupPaths := range lookupPathsByProvider {
		mediaSourceID, provider := parseInventoryFileProviderLookupKey(providerKey)
		for _, pathBatch := range chunkStrings(lookupPaths, bulkLookupChunkSize) {
			var partial []database.InventoryFile
			if err := s.db.WithContext(ctx).
				Where("media_source_id = ? AND storage_provider = ? AND storage_path IN ?", mediaSourceID, provider, pathBatch).
				Find(&partial).Error; err != nil {
				return BulkUpsertFilesResult{}, err
			}
			stored = append(stored, partial...)
		}
	}
	result := BulkUpsertFilesResult{FilesByStoragePath: make(map[string]database.InventoryFile, len(stored)), FilesBySourcePath: make(map[string]database.InventoryFile, len(stored))}
	for _, file := range stored {
		result.FilesByStoragePath[file.StorageProvider+"\x00"+file.StoragePath] = file
		result.FilesBySourcePath[inventoryFileLookupKey(file.MediaSourceID, file.StorageProvider, file.StoragePath)] = file
	}
	return result, nil
}

func (s *Service) UpsertResource(ctx context.Context, input UpsertResourceInput) (database.Resource, error) {
	stableKey := strings.TrimSpace(input.StableResourceKey)
	if stableKey == "" {
		return database.Resource{}, errors.New("stable resource key is required")
	}
	resource := database.Resource{
		StableResourceKey:    stableKey,
		ResourceType:         database.NormalizeResourceType(input.ResourceType),
		ResourceShape:        database.NormalizeResourceShape(input.ResourceShape),
		DisplayName:          strings.TrimSpace(input.DisplayName),
		Edition:              strings.TrimSpace(input.Edition),
		QualityLabel:         strings.TrimSpace(input.QualityLabel),
		DurationSeconds:      input.DurationSeconds,
		Status:               defaultString(input.Status, AssetStatusAvailable),
		ProbeStatus:          defaultString(input.ProbeStatus, "pending"),
		TechnicalSummaryJSON: input.TechnicalSummaryJSON,
	}
	if resource.Status == AssetStatusMissing {
		now := time.Now().UTC()
		resource.MissingSince = &now
	}
	updateColumns := []string{"resource_type", "resource_shape", "display_name", "edition", "quality_label", "duration_seconds", "status", "probe_status", "technical_summary_json", "updated_at"}
	if resource.Status == AssetStatusAvailable {
		updateColumns = append(updateColumns, "missing_since")
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "stable_resource_key"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(&resource).Error; err != nil {
		return database.Resource{}, err
	}
	if err := s.db.WithContext(ctx).Where("stable_resource_key = ?", stableKey).First(&resource).Error; err != nil {
		return database.Resource{}, err
	}
	return resource, nil
}

func (s *Service) LinkResourceToFile(ctx context.Context, input LinkResourceFileInput) (database.ResourceFile, error) {
	if input.ResourceID == 0 || input.InventoryFileID == 0 {
		return database.ResourceFile{}, errors.New("resource id and inventory file id are required")
	}
	link := database.ResourceFile{
		ResourceID:      input.ResourceID,
		InventoryFileID: input.InventoryFileID,
		Role:            database.NormalizeResourceFileRole(input.Role),
		PartIndex:       input.PartIndex,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "resource_id"}, {Name: "inventory_file_id"}, {Name: "role"}, {Name: "part_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(&link).Error; err != nil {
		return database.ResourceFile{}, err
	}
	if err := s.db.WithContext(ctx).
		Where("resource_id = ? AND inventory_file_id = ? AND role = ? AND part_index = ?", link.ResourceID, link.InventoryFileID, link.Role, link.PartIndex).
		First(&link).Error; err != nil {
		return database.ResourceFile{}, err
	}
	return link, nil
}

func (s *Service) BulkLinkResourceToFiles(ctx context.Context, inputs []LinkResourceFileInput) error {
	if len(inputs) == 0 {
		return nil
	}
	links := make([]database.ResourceFile, 0, len(inputs))
	for _, input := range inputs {
		if input.ResourceID == 0 || input.InventoryFileID == 0 {
			return errors.New("resource id and inventory file id are required")
		}
		links = append(links, database.ResourceFile{
			ResourceID:      input.ResourceID,
			InventoryFileID: input.InventoryFileID,
			Role:            database.NormalizeResourceFileRole(input.Role),
			PartIndex:       input.PartIndex,
		})
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "resource_id"}, {Name: "inventory_file_id"}, {Name: "role"}, {Name: "part_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).CreateInBatches(&links, bulkLinkWriteBatchSize).Error
}

func (s *Service) AttachResourceToLibrary(ctx context.Context, input AttachResourceLibraryInput) (database.ResourceLibraryLink, error) {
	if input.ResourceID == 0 || input.LibraryID == 0 {
		return database.ResourceLibraryLink{}, errors.New("resource id and library id are required")
	}
	observedAt := time.Now().UTC()
	if input.ObservedAt != nil && !input.ObservedAt.IsZero() {
		observedAt = input.ObservedAt.UTC()
	}
	status := defaultString(input.Status, AssetStatusAvailable)
	link := database.ResourceLibraryLink{
		ResourceID:   input.ResourceID,
		LibraryID:    input.LibraryID,
		Status:       status,
		FirstSeenAt:  observedAt,
		LastSeenAt:   observedAt,
		EvidenceJSON: input.EvidenceJSON,
		ReviewState:  database.NormalizeReviewState(input.ReviewState),
	}
	if status == AssetStatusMissing {
		link.MissingSince = &observedAt
	}
	updates := map[string]any{
		"status":        link.Status,
		"last_seen_at":  link.LastSeenAt,
		"evidence_json": link.EvidenceJSON,
		"review_state":  link.ReviewState,
		"deleted_at":    nil,
		"updated_at":    observedAt,
	}
	if status == AssetStatusMissing {
		updates["missing_since"] = link.MissingSince
	} else {
		updates["missing_since"] = nil
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "resource_id"}, {Name: "library_id"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(&link).Error; err != nil {
		return database.ResourceLibraryLink{}, err
	}
	if err := s.db.WithContext(ctx).
		Where("resource_id = ? AND library_id = ?", input.ResourceID, input.LibraryID).
		First(&link).Error; err != nil {
		return database.ResourceLibraryLink{}, err
	}
	return link, nil
}

func (s *Service) MarkResourceLibraryMissing(ctx context.Context, resourceID uint, libraryID uint, missingAt time.Time) (database.ResourceLibraryLink, error) {
	if resourceID == 0 || libraryID == 0 {
		return database.ResourceLibraryLink{}, errors.New("resource id and library id are required")
	}
	if missingAt.IsZero() {
		missingAt = time.Now().UTC()
	}
	if err := s.db.WithContext(ctx).Model(&database.ResourceLibraryLink{}).
		Where("resource_id = ? AND library_id = ?", resourceID, libraryID).
		Updates(map[string]any{"status": AssetStatusMissing, "missing_since": missingAt.UTC(), "updated_at": time.Now().UTC()}).Error; err != nil {
		return database.ResourceLibraryLink{}, err
	}
	var link database.ResourceLibraryLink
	if err := s.db.WithContext(ctx).Where("resource_id = ? AND library_id = ?", resourceID, libraryID).First(&link).Error; err != nil {
		return database.ResourceLibraryLink{}, err
	}
	return link, nil
}

func (s *Service) LinkResourceToMetadata(ctx context.Context, input LinkResourceMetadataInput) (database.ResourceMetadataLink, error) {
	if input.ResourceID == 0 || input.MetadataItemID == 0 {
		return database.ResourceMetadataLink{}, errors.New("resource id and metadata item id are required")
	}
	link := database.ResourceMetadataLink{
		ResourceID:     input.ResourceID,
		MetadataItemID: input.MetadataItemID,
		Role:           database.NormalizeResourceLinkRole(input.Role),
		SegmentIndex:   input.SegmentIndex,
		StartSeconds:   input.StartSeconds,
		EndSeconds:     input.EndSeconds,
		Confidence:     input.Confidence,
		EvidenceJSON:   input.EvidenceJSON,
		Source:         strings.TrimSpace(input.Source),
		ReviewState:    database.NormalizeReviewState(input.ReviewState),
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "resource_id"}, {Name: "metadata_item_id"}, {Name: "role"}, {Name: "segment_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"start_seconds", "end_seconds", "confidence", "evidence_json", "source", "review_state", "updated_at"}),
	}).Create(&link).Error; err != nil {
		return database.ResourceMetadataLink{}, err
	}
	if err := s.db.WithContext(ctx).
		Where("resource_id = ? AND metadata_item_id = ? AND role = ? AND segment_index = ?", link.ResourceID, link.MetadataItemID, link.Role, link.SegmentIndex).
		First(&link).Error; err != nil {
		return database.ResourceMetadataLink{}, err
	}
	return link, nil
}

func (s *Service) UnlinkResourceFromMetadata(ctx context.Context, resourceID uint, metadataItemID uint, role string, segmentIndex int) error {
	if resourceID == 0 || metadataItemID == 0 {
		return errors.New("resource id and metadata item id are required")
	}
	return s.db.WithContext(ctx).
		Where("resource_id = ? AND metadata_item_id = ? AND role = ? AND segment_index = ?", resourceID, metadataItemID, database.NormalizeResourceLinkRole(role), segmentIndex).
		Delete(&database.ResourceMetadataLink{}).Error
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func chunkStrings(values []string, size int) [][]string {
	if len(values) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(values)
	}
	chunks := make([][]string, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func inventoryFileLookupKey(mediaSourceID uint, provider string, storagePath string) string {
	return strconv.FormatUint(uint64(mediaSourceID), 10) + "\x00" + strings.TrimSpace(provider) + "\x00" + strings.TrimSpace(storagePath)
}

func inventoryFileProviderLookupKey(mediaSourceID uint, provider string) string {
	return strconv.FormatUint(uint64(mediaSourceID), 10) + "\x00" + strings.TrimSpace(provider)
}

func parseInventoryFileProviderLookupKey(value string) (uint, string) {
	parts := strings.SplitN(value, "\x00", 2)
	if len(parts) != 2 {
		return 0, strings.TrimSpace(value)
	}
	parsed, _ := strconv.ParseUint(parts[0], 10, 64)
	return uint(parsed), parts[1]
}
