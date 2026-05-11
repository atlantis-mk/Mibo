package catalog

import (
	"context"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) RebuildMetadataSearchDocument(ctx context.Context, metadataItemID uint) (database.MetadataSearchDocument, error) {
	doc, err := buildMetadataSearchDocument(ctx, s.db, metadataItemID)
	if err != nil {
		return database.MetadataSearchDocument{}, err
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&doc).Error; err != nil {
		return database.MetadataSearchDocument{}, err
	}
	return doc, nil
}

func buildMetadataSearchDocument(ctx context.Context, db *gorm.DB, metadataItemID uint) (database.MetadataSearchDocument, error) {
	var item database.MetadataItem
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&item).Error; err != nil {
		return database.MetadataSearchDocument{}, err
	}
	doc := database.MetadataSearchDocument{MetadataItemID: item.ID, ItemType: item.ItemType, ContentForm: item.ContentForm, Title: item.Title, OriginalTitle: item.OriginalTitle, Year: item.Year, UpdatedAt: time.Now().UTC()}
	var people []database.Person
	if err := db.WithContext(ctx).Table("people").Select("people.*").Joins("JOIN metadata_item_people ON metadata_item_people.person_id = people.id").Where("metadata_item_people.metadata_item_id = ?", item.ID).Find(&people).Error; err != nil {
		return database.MetadataSearchDocument{}, err
	}
	for _, person := range people {
		doc.PeopleText = appendSearchText(doc.PeopleText, person.Name, person.IMDBID)
	}
	var tags []database.Tag
	if err := db.WithContext(ctx).Table("tags").Select("tags.*").Joins("JOIN metadata_item_tags ON metadata_item_tags.tag_id = tags.id").Where("metadata_item_tags.metadata_item_id = ?", item.ID).Find(&tags).Error; err != nil {
		return database.MetadataSearchDocument{}, err
	}
	for _, tag := range tags {
		doc.TagsText = appendSearchText(doc.TagsText, tag.Kind, tag.Name)
	}
	var externalIDs []database.MetadataExternalID
	if err := db.WithContext(ctx).Where("metadata_item_id = ?", item.ID).Find(&externalIDs).Error; err != nil {
		return database.MetadataSearchDocument{}, err
	}
	for _, externalID := range externalIDs {
		doc.ProviderIDsText = appendSearchText(doc.ProviderIDsText, externalID.Provider, externalID.ProviderType, externalID.ExternalID)
	}
	return doc, nil
}

func appendSearchText(existing string, values ...string) string {
	parts := []string{}
	if strings.TrimSpace(existing) != "" {
		parts = append(parts, strings.TrimSpace(existing))
	}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, strings.TrimSpace(value))
		}
	}
	return strings.Join(parts, " ")
}

func (s *Service) RebuildLibrarySearchDocument(ctx context.Context, libraryID uint, metadataItemID uint) (database.LibrarySearchDocument, error) {
	doc, err := buildLibrarySearchDocument(ctx, s.db, libraryID, metadataItemID)
	if err != nil {
		return database.LibrarySearchDocument{}, err
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&doc).Error; err != nil {
		return database.LibrarySearchDocument{}, err
	}
	return doc, nil
}

func buildLibrarySearchDocument(ctx context.Context, db *gorm.DB, libraryID uint, metadataItemID uint) (database.LibrarySearchDocument, error) {
	metadataDoc, err := buildMetadataSearchDocument(ctx, db, metadataItemID)
	if err != nil {
		return database.LibrarySearchDocument{}, err
	}
	var projection database.LibraryMetadataProjection
	if err := db.WithContext(ctx).Where("library_id = ? AND metadata_item_id = ?", libraryID, metadataItemID).First(&projection).Error; err != nil {
		return database.LibrarySearchDocument{}, err
	}
	doc := database.LibrarySearchDocument{LibraryID: libraryID, MetadataItemID: metadataItemID, ItemType: projection.ItemType, Title: projection.Title, OriginalTitle: metadataDoc.OriginalTitle, PeopleText: metadataDoc.PeopleText, TagsText: metadataDoc.TagsText, ProviderIDsText: metadataDoc.ProviderIDsText, Year: projection.Year, AvailabilityStatus: projection.AvailabilityStatus, UpdatedAt: time.Now().UTC()}
	var rows []struct {
		DisplayName  string
		QualityLabel string
		StoragePath  string
	}
	if err := db.WithContext(ctx).
		Table("resources").
		Select("resources.display_name, resources.quality_label, inventory_files.storage_path").
		Joins("JOIN resource_metadata_links ON resource_metadata_links.resource_id = resources.id").
		Joins("JOIN resource_library_links ON resource_library_links.resource_id = resources.id").
		Joins("LEFT JOIN resource_files ON resource_files.resource_id = resources.id").
		Joins("LEFT JOIN inventory_files ON inventory_files.id = resource_files.inventory_file_id").
		Where("resource_metadata_links.metadata_item_id = ? AND resource_library_links.library_id = ? AND resource_library_links.deleted_at IS NULL", metadataItemID, libraryID).
		Scan(&rows).Error; err != nil {
		return database.LibrarySearchDocument{}, err
	}
	for _, row := range rows {
		doc.ResourceText = appendSearchText(doc.ResourceText, row.DisplayName, row.QualityLabel, row.StoragePath)
	}
	return doc, nil
}
