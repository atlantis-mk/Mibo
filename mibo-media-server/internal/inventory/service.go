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

	FileStatusAvailable = "available"
	FileStatusMissing   = "missing"
	FileRoleSource      = "source"
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
	SizeBytes         int64
	ModifiedAt        *time.Time
	Container         string
	Status            string
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
		SizeBytes:         input.SizeBytes,
		ModifiedAt:        input.ModifiedAt,
		Container:         strings.TrimSpace(input.Container),
		Status:            defaultString(input.Status, FileStatusAvailable),
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "storage_provider"}, {Name: "storage_path"}},
		DoUpdates: clause.AssignmentColumns([]string{"library_id", "stable_identity_key", "hashes_json", "size_bytes", "modified_at", "container", "status", "updated_at"}),
	}).Create(&file).Error
	if err != nil {
		return database.InventoryFile{}, err
	}

	err = s.db.WithContext(ctx).Where("storage_provider = ? AND storage_path = ?", file.StorageProvider, file.StoragePath).First(&file).Error
	return file, err
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

func (s *Service) ListAssetItems(ctx context.Context, assetID uint) ([]database.AssetItem, error) {
	var links []database.AssetItem
	err := s.db.WithContext(ctx).
		Where("asset_id = ?", assetID).
		Order("segment_index asc").
		Order("id asc").
		Find(&links).Error
	return links, err
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
