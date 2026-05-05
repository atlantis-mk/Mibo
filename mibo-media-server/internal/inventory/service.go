package inventory

import (
	"context"
	"errors"
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

type CreateAssetInput struct {
	LibraryID            uint
	AssetType            string
	DisplayName          string
	Edition              string
	QualityLabel         string
	DurationSeconds      *float64
	Status               string
	ProbeStatus          string
	TechnicalSummaryJSON string
}

type UpsertFileInput struct {
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
}

type LinkAssetItemInput struct {
	AssetID      uint
	ItemID       uint
	Role         string
	SegmentIndex int
	StartSeconds *float64
	EndSeconds   *float64
	Confidence   *float64
	Source       string
}

type LinkAssetFileInput struct {
	AssetID   uint
	FileID    uint
	Role      string
	PartIndex int
}

func (s *Service) CreateAsset(ctx context.Context, input CreateAssetInput) (database.MediaAsset, error) {
	if input.LibraryID == 0 {
		return database.MediaAsset{}, errors.New("library id is required")
	}
	asset := database.MediaAsset{
		LibraryID:            input.LibraryID,
		AssetType:            defaultString(input.AssetType, AssetTypeMain),
		DisplayName:          strings.TrimSpace(input.DisplayName),
		Edition:              strings.TrimSpace(input.Edition),
		QualityLabel:         strings.TrimSpace(input.QualityLabel),
		DurationSeconds:      input.DurationSeconds,
		Status:               defaultString(input.Status, AssetStatusAvailable),
		ProbeStatus:          defaultString(input.ProbeStatus, "pending"),
		TechnicalSummaryJSON: input.TechnicalSummaryJSON,
	}
	return asset, s.db.WithContext(ctx).Create(&asset).Error
}

func (s *Service) UpsertFile(ctx context.Context, input UpsertFileInput) (database.InventoryFile, error) {
	if input.LibraryID == 0 {
		return database.InventoryFile{}, errors.New("library id is required")
	}
	if strings.TrimSpace(input.StorageProvider) == "" || strings.TrimSpace(input.StoragePath) == "" {
		return database.InventoryFile{}, errors.New("storage provider and path are required")
	}

	file := database.InventoryFile{
		LibraryID:         input.LibraryID,
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
	updateColumns := []string{"library_id", "stable_identity_key", "hashes_json", "thumbnail_url", "size_bytes", "modified_at", "container", "content_class", "status", "scan_state", "updated_at"}
	if file.Status == FileStatusAvailable {
		updateColumns = append(updateColumns, "missing_since")
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "storage_provider"}, {Name: "storage_path"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(&file).Error
	if err != nil {
		return database.InventoryFile{}, err
	}

	err = s.db.WithContext(ctx).Where("storage_provider = ? AND storage_path = ?", file.StorageProvider, file.StoragePath).First(&file).Error
	return file, err
}

func (s *Service) BulkUpsertFiles(ctx context.Context, inputs []UpsertFileInput) (BulkUpsertFilesResult, error) {
	if len(inputs) == 0 {
		return BulkUpsertFilesResult{FilesByStoragePath: map[string]database.InventoryFile{}}, nil
	}
	files := make([]database.InventoryFile, 0, len(inputs))
	lookupPathsByProvider := make(map[string][]string)
	seenPairs := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if input.LibraryID == 0 {
			return BulkUpsertFilesResult{}, errors.New("library id is required")
		}
		provider := strings.TrimSpace(input.StorageProvider)
		storagePath := strings.TrimSpace(input.StoragePath)
		if provider == "" || storagePath == "" {
			return BulkUpsertFilesResult{}, errors.New("storage provider and path are required")
		}
		file := database.InventoryFile{
			LibraryID:         input.LibraryID,
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
		pairKey := provider + "\x00" + storagePath
		if _, ok := seenPairs[pairKey]; ok {
			continue
		}
		seenPairs[pairKey] = struct{}{}
		lookupPathsByProvider[provider] = append(lookupPathsByProvider[provider], storagePath)
	}
	updateColumns := []string{"library_id", "stable_identity_key", "hashes_json", "thumbnail_url", "size_bytes", "modified_at", "container", "content_class", "status", "scan_state", "missing_since", "updated_at"}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "storage_provider"}, {Name: "storage_path"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).CreateInBatches(&files, bulkFileWriteBatchSize).Error; err != nil {
		return BulkUpsertFilesResult{}, err
	}
	var stored []database.InventoryFile
	for provider, lookupPaths := range lookupPathsByProvider {
		for _, pathBatch := range chunkStrings(lookupPaths, bulkLookupChunkSize) {
			var partial []database.InventoryFile
			if err := s.db.WithContext(ctx).
				Where("storage_provider = ? AND storage_path IN ?", provider, pathBatch).
				Find(&partial).Error; err != nil {
				return BulkUpsertFilesResult{}, err
			}
			stored = append(stored, partial...)
		}
	}
	result := BulkUpsertFilesResult{FilesByStoragePath: make(map[string]database.InventoryFile, len(stored))}
	for _, file := range stored {
		result.FilesByStoragePath[file.StorageProvider+"\x00"+file.StoragePath] = file
	}
	return result, nil
}

func (s *Service) LinkAssetToItem(ctx context.Context, input LinkAssetItemInput) (database.AssetItem, error) {
	if input.AssetID == 0 || input.ItemID == 0 {
		return database.AssetItem{}, errors.New("asset id and item id are required")
	}
	link := database.AssetItem{
		AssetID:      input.AssetID,
		ItemID:       input.ItemID,
		Role:         defaultString(input.Role, AssetItemRolePrimary),
		SegmentIndex: input.SegmentIndex,
		StartSeconds: input.StartSeconds,
		EndSeconds:   input.EndSeconds,
		Confidence:   input.Confidence,
		Source:       strings.TrimSpace(input.Source),
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "asset_id"}, {Name: "item_id"}, {Name: "role"}, {Name: "segment_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"start_seconds", "end_seconds", "confidence", "source", "updated_at"}),
	}).Create(&link).Error
	return link, err
}

func (s *Service) BulkLinkAssetToItems(ctx context.Context, inputs []LinkAssetItemInput) error {
	if len(inputs) == 0 {
		return nil
	}
	links := make([]database.AssetItem, 0, len(inputs))
	for _, input := range inputs {
		if input.AssetID == 0 || input.ItemID == 0 {
			return errors.New("asset id and item id are required")
		}
		links = append(links, database.AssetItem{
			AssetID:      input.AssetID,
			ItemID:       input.ItemID,
			Role:         defaultString(input.Role, AssetItemRolePrimary),
			SegmentIndex: input.SegmentIndex,
			StartSeconds: input.StartSeconds,
			EndSeconds:   input.EndSeconds,
			Confidence:   input.Confidence,
			Source:       strings.TrimSpace(input.Source),
		})
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "asset_id"}, {Name: "item_id"}, {Name: "role"}, {Name: "segment_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"start_seconds", "end_seconds", "confidence", "source", "updated_at"}),
	}).CreateInBatches(&links, bulkLinkWriteBatchSize).Error
}

func (s *Service) LinkAssetToFile(ctx context.Context, input LinkAssetFileInput) (database.AssetFile, error) {
	if input.AssetID == 0 || input.FileID == 0 {
		return database.AssetFile{}, errors.New("asset id and file id are required")
	}
	link := database.AssetFile{
		AssetID:   input.AssetID,
		FileID:    input.FileID,
		Role:      defaultString(input.Role, FileRoleSource),
		PartIndex: input.PartIndex,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "asset_id"}, {Name: "file_id"}, {Name: "role"}, {Name: "part_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(&link).Error
	return link, err
}

func (s *Service) BulkLinkAssetToFiles(ctx context.Context, inputs []LinkAssetFileInput) error {
	if len(inputs) == 0 {
		return nil
	}
	links := make([]database.AssetFile, 0, len(inputs))
	for _, input := range inputs {
		if input.AssetID == 0 || input.FileID == 0 {
			return errors.New("asset id and file id are required")
		}
		links = append(links, database.AssetFile{
			AssetID:   input.AssetID,
			FileID:    input.FileID,
			Role:      defaultString(input.Role, FileRoleSource),
			PartIndex: input.PartIndex,
		})
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "asset_id"}, {Name: "file_id"}, {Name: "role"}, {Name: "part_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).CreateInBatches(&links, bulkLinkWriteBatchSize).Error
}

func (s *Service) UnlinkAssetFromItem(ctx context.Context, assetID uint, itemID uint) error {
	if assetID == 0 || itemID == 0 {
		return errors.New("asset id and item id are required")
	}
	return s.db.WithContext(ctx).
		Where("asset_id = ? AND item_id = ?", assetID, itemID).
		Delete(&database.AssetItem{}).Error
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
