package catalog

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	IdentityProviderScanner = "scanner"
	IdentityProviderManual  = "manual"

	IdentityTypeMovie   = ItemTypeMovie
	IdentityTypeSeries  = ItemTypeSeries
	IdentityTypeSeason  = ItemTypeSeason
	IdentityTypeEpisode = ItemTypeEpisode
)

type IdentityInput struct {
	ItemID       uint
	Provider     string
	IdentityType string
	IdentityKey  string
	SourcePath   string
	Confidence   *float64
	EvidenceJSON string
}

func (s *Service) SetIdentity(ctx context.Context, input IdentityInput) (database.CatalogIdentity, error) {
	identity, err := normalizeIdentityInput(input)
	if err != nil {
		return database.CatalogIdentity{}, err
	}

	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "identity_type"}, {Name: "identity_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"item_id",
			"source_path",
			"confidence",
			"evidence_json",
			"updated_at",
		}),
	}).Create(&identity).Error
	if err != nil {
		return database.CatalogIdentity{}, err
	}

	return s.FindIdentity(ctx, identity.Provider, identity.IdentityType, identity.IdentityKey)
}

func (s *Service) FindIdentity(ctx context.Context, provider string, identityType string, identityKey string) (database.CatalogIdentity, error) {
	provider = strings.TrimSpace(provider)
	identityType = strings.TrimSpace(identityType)
	identityKey = strings.TrimSpace(identityKey)
	if provider == "" || identityType == "" || identityKey == "" {
		return database.CatalogIdentity{}, errors.New("provider, identity type, and identity key are required")
	}

	var identity database.CatalogIdentity
	err := s.db.WithContext(ctx).
		Where("provider = ? AND identity_type = ? AND identity_key = ?", provider, identityType, identityKey).
		First(&identity).Error
	return identity, err
}

func (s *Service) FindItemByIdentity(ctx context.Context, provider string, identityType string, identityKey string) (database.CatalogItem, database.CatalogIdentity, error) {
	identity, err := s.FindIdentity(ctx, provider, identityType, identityKey)
	if err != nil {
		return database.CatalogItem{}, database.CatalogIdentity{}, err
	}

	var item database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", identity.ItemID).First(&item).Error; err != nil {
		return database.CatalogItem{}, database.CatalogIdentity{}, err
	}
	return item, identity, nil
}

func (s *Service) ReconcileItemByIdentity(ctx context.Context, input IdentityInput) (database.CatalogItem, database.CatalogIdentity, bool, error) {
	identity, err := normalizeIdentityInput(input)
	if err != nil {
		return database.CatalogItem{}, database.CatalogIdentity{}, false, err
	}

	item, existing, err := s.FindItemByIdentity(ctx, identity.Provider, identity.IdentityType, identity.IdentityKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.CatalogItem{}, database.CatalogIdentity{}, false, nil
	}
	if err != nil {
		return database.CatalogItem{}, database.CatalogIdentity{}, false, err
	}
	return item, existing, true, nil
}

func ScannerIdentityKeyForItem(item database.CatalogItem) (string, bool) {
	path := strings.TrimSpace(item.Path)
	itemType := strings.TrimSpace(item.Type)
	if item.LibraryID == 0 || itemType == "" || path == "" {
		return "", false
	}
	return "library:" + strconv.FormatUint(uint64(item.LibraryID), 10) + ":" + itemType + ":" + path, true
}

func normalizeIdentityInput(input IdentityInput) (database.CatalogIdentity, error) {
	if input.ItemID == 0 {
		return database.CatalogIdentity{}, errors.New("item id is required")
	}
	identity := database.CatalogIdentity{
		ItemID:       input.ItemID,
		Provider:     strings.TrimSpace(input.Provider),
		IdentityType: strings.TrimSpace(input.IdentityType),
		IdentityKey:  strings.TrimSpace(input.IdentityKey),
		SourcePath:   strings.TrimSpace(input.SourcePath),
		Confidence:   input.Confidence,
		EvidenceJSON: input.EvidenceJSON,
	}
	if identity.Provider == "" || identity.IdentityType == "" || identity.IdentityKey == "" {
		return database.CatalogIdentity{}, fmt.Errorf("provider, identity type, and identity key are required")
	}
	return identity, nil
}
