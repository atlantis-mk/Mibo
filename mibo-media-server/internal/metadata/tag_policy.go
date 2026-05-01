package metadata

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) applyNormalizedTags(ctx context.Context, itemID uint, tags []NormalizedMetadataTag, sourceID *uint) error {
	if itemID == 0 {
		return nil
	}
	byKind := map[string][]string{}
	for _, tag := range tags {
		kind := strings.ToLower(strings.TrimSpace(tag.Kind))
		name := strings.TrimSpace(tag.Name)
		if kind == "" || name == "" {
			continue
		}
		if kind != "genre" && kind != "keyword" {
			continue
		}
		byKind[kind] = appendUniqueStringFold(byKind[kind], name)
	}
	if len(byKind) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for kind, names := range byKind {
			if err := tx.WithContext(ctx).
				Where("item_id = ? AND tag_id IN (SELECT id FROM tags WHERE kind = ?)", itemID, kind).
				Delete(&database.ItemTag{}).Error; err != nil {
				return err
			}
			for _, name := range names {
				tag := database.Tag{Kind: kind, Name: name}
				if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "kind"}, {Name: "name"}},
					DoNothing: true,
				}).Create(&tag).Error; err != nil {
					return err
				}
				if tag.ID == 0 {
					if err := tx.WithContext(ctx).Where("kind = ? AND name = ?", kind, name).First(&tag).Error; err != nil {
						return err
					}
				}
				itemTag := database.ItemTag{ItemID: itemID, TagID: tag.ID, SourceID: sourceID}
				if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "item_id"}, {Name: "tag_id"}},
					DoUpdates: clause.AssignmentColumns([]string{"source_id", "updated_at"}),
				}).Create(&itemTag).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func appendUniqueStringFold(values []string, value string) []string {
	key := strings.ToLower(strings.TrimSpace(value))
	for _, existing := range values {
		if strings.ToLower(strings.TrimSpace(existing)) == key {
			return values
		}
	}
	return append(values, strings.TrimSpace(value))
}

func appliedTagFields(itemID uint, tags []NormalizedMetadataTag, sourceID *uint, applyMode string, confidence *float64) []MetadataAppliedField {
	seen := map[string]struct{}{}
	result := []MetadataAppliedField{}
	for _, tag := range tags {
		kind := strings.ToLower(strings.TrimSpace(tag.Kind))
		if kind != "genre" && kind != "keyword" {
			continue
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		result = append(result, MetadataAppliedField{ItemID: itemID, FieldKey: "tags." + kind, SourceID: sourceID, ApplyMode: applyMode, Confidence: confidence})
	}
	return result
}
