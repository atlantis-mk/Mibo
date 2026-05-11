package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) recordNormalizedMetadataItemProviderSource(ctx context.Context, metadataItemID uint, plan MetadataExecutionPlan, detail NormalizedMetadataDetail, confidence float64) (database.MetadataItemSource, error) {
	payload, err := json.Marshal(map[string]any{"provider": detail.Provider, "provider_type": detail.ProviderType, "external_id": detail.ExternalID, "confidence": confidence, "matched_title": detail.Title})
	if err != nil {
		return database.MetadataItemSource{}, err
	}
	var providerInstanceID *uint
	providerInstanceName := ""
	for _, provider := range plan.DetailProviders {
		if provider.Record.ProviderType == detail.Provider {
			providerInstanceID = &provider.Record.ID
			providerInstanceName = provider.Record.Name
			break
		}
	}
	source := database.MetadataItemSource{MetadataItemID: metadataItemID, SourceType: catalog.SourceTypeProvider, SourceName: strings.TrimSpace(detail.Provider), Language: strings.TrimSpace(plan.PreferredMetadataLanguage), ExternalID: strings.TrimSpace(detail.ExternalID), TriggeringLibraryID: &plan.LibraryID, MetadataProfileID: plan.MetadataProfileID, MetadataProfileName: strings.TrimSpace(plan.MetadataProfileName), ProviderInstanceID: providerInstanceID, ProviderInstanceName: providerInstanceName, FallbackSummaryJSON: marshalOperationJSON(map[string]any{"preferred_image_language": plan.PreferredImageLanguage}), PayloadJSON: string(payload), Confidence: &confidence, FetchedAt: time.Now().UTC()}
	if err := s.db.WithContext(ctx).Create(&source).Error; err != nil {
		return database.MetadataItemSource{}, err
	}
	return source, nil
}

func (s *Service) applyMetadataItemFieldChanges(ctx context.Context, changes []MetadataFieldChange) ([]MetadataAppliedField, []MetadataSkippedField, error) {
	applied := make([]MetadataAppliedField, 0, len(changes))
	skipped := make([]MetadataSkippedField, 0)
	for _, change := range changes {
		fieldKey := strings.TrimSpace(change.FieldKey)
		if change.ItemID == 0 || fieldKey == "" {
			return nil, nil, fmt.Errorf("metadata item field change requires item id and field key")
		}
		locked, err := s.metadataItemFieldLocked(ctx, change.ItemID, fieldKey)
		if err != nil {
			return nil, nil, err
		}
		if locked && !change.Force && change.ApplyMode != FieldApplyModeManual {
			skipped = append(skipped, MetadataSkippedField{ItemID: change.ItemID, FieldKey: fieldKey, Reason: "locked"})
			continue
		}
		valueJSON, err := json.Marshal(change.Value)
		if err != nil {
			return nil, nil, err
		}
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := applyMetadataItemCanonicalField(ctx, tx, change.ItemID, fieldKey, change.Value); err != nil {
				return err
			}
			state := database.MetadataItemFieldState{MetadataItemID: change.ItemID, FieldKey: fieldKey, SourceID: change.SourceID, ValueJSON: string(valueJSON)}
			return tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "metadata_item_id"}, {Name: "field_key"}, {Name: "locale"}}, DoUpdates: clause.AssignmentColumns([]string{"source_id", "value_json", "updated_at"})}).Create(&state).Error
		}); err != nil {
			return nil, nil, err
		}
		applied = append(applied, MetadataAppliedField{ItemID: change.ItemID, FieldKey: fieldKey, SourceID: change.SourceID, ApplyMode: change.ApplyMode, Confidence: change.Confidence})
	}
	return applied, skipped, nil
}

func normalizedDetailFieldChanges(itemID uint, detail NormalizedMetadataDetail, sourceID *uint, applyMode string, confidence *float64) []MetadataFieldChange {
	changes := make([]MetadataFieldChange, 0, 8)
	add := func(fieldKey string, value any) {
		if value == nil {
			return
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			return
		}
		changes = append(changes, MetadataFieldChange{ItemID: itemID, FieldKey: fieldKey, Value: value, SourceID: sourceID, ApplyMode: applyMode, Confidence: confidence})
	}
	add("title", detail.Title)
	add("sort_title", detail.Title)
	add("original_title", detail.OriginalTitle)
	add("overview", detail.Overview)
	if detail.Year != nil {
		add("year", *detail.Year)
	}
	if detail.RuntimeSeconds != nil {
		add("runtime_seconds", *detail.RuntimeSeconds)
	}
	if detail.CommunityRating != nil {
		add("community_rating", *detail.CommunityRating)
	}
	add("official_rating", detail.OfficialRating)
	add("series_status", detail.SeriesStatus)
	if detail.ReleaseDate != "" {
		if parsed := parseProviderDate(detail.ReleaseDate); parsed != nil {
			add("release_date", *parsed)
		}
	}
	if detail.FirstAirDate != "" {
		if parsed := parseProviderDate(detail.FirstAirDate); parsed != nil {
			add("first_air_date", *parsed)
		}
	}
	return changes
}

func (s *Service) metadataItemFieldLocked(ctx context.Context, metadataItemID uint, fieldKey string) (bool, error) {
	var state database.MetadataItemFieldState
	err := s.db.WithContext(ctx).Where("metadata_item_id = ? AND field_key = ? AND locale = ?", metadataItemID, fieldKey, "").First(&state).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return state.IsLocked, nil
}

func applyMetadataItemCanonicalField(ctx context.Context, tx *gorm.DB, metadataItemID uint, fieldKey string, value any) error {
	updates := map[string]any{}
	switch fieldKey {
	case "title", "sort_title", "original_title", "overview", "year", "runtime_seconds", "community_rating", "official_rating", "series_status", "release_date", "first_air_date", "last_air_date", "governance_status":
		updates[fieldKey] = value
	}
	if len(updates) == 0 {
		return nil
	}
	updates["last_canonicalized_at"] = time.Now().UTC()
	return tx.WithContext(ctx).Model(&database.MetadataItem{}).Where("id = ?", metadataItemID).Updates(updates).Error
}

func (s *Service) applyNormalizedMetadataItemExternalIDs(ctx context.Context, metadataItemID uint, externalIDs []NormalizedMetadataExternalID, source string, defaultConfidence *float64) error {
	for _, externalID := range externalIDs {
		provider := strings.TrimSpace(externalID.Provider)
		providerType := strings.TrimSpace(externalID.ProviderType)
		value := strings.TrimSpace(externalID.ExternalID)
		if metadataItemID == 0 || provider == "" || providerType == "" || value == "" {
			continue
		}
		confidence := externalID.Confidence
		if confidence == nil {
			confidence = defaultConfidence
		}
		row := database.MetadataExternalID{MetadataItemID: metadataItemID, Provider: provider, ProviderType: providerType, ExternalID: value, IsPrimary: externalID.IsPrimary, Source: source, Confidence: confidence}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "provider"}, {Name: "provider_type"}, {Name: "external_id"}}, DoUpdates: clause.AssignmentColumns([]string{"metadata_item_id", "is_primary", "source", "confidence", "updated_at"})}).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) applyNormalizedMetadataItemImages(ctx context.Context, metadataItemID uint, images []NormalizedMetadataImage, forceSelectImages bool, sourceID *uint) error {
	for _, image := range images {
		if metadataItemID == 0 || strings.TrimSpace(image.URL) == "" || strings.TrimSpace(image.ImageType) == "" {
			continue
		}
		selected := image.Selected || forceSelectImages
		if selected {
			if err := s.db.WithContext(ctx).Model(&database.MetadataItemImage{}).Where("metadata_item_id = ? AND image_type = ?", metadataItemID, strings.TrimSpace(image.ImageType)).Update("is_selected", false).Error; err != nil {
				return err
			}
		}
		row := database.MetadataItemImage{MetadataItemID: metadataItemID, ImageType: strings.TrimSpace(image.ImageType), URL: strings.TrimSpace(image.URL), SourceID: sourceID, Language: strings.TrimSpace(image.Language), Width: image.Width, Height: image.Height, IsSelected: selected, SortOrder: image.SortOrder}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) applyNormalizedMetadataItemPeople(ctx context.Context, metadataItemID uint, people []NormalizedMetadataPerson, sourceID *uint) error {
	if metadataItemID == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("metadata_item_id = ?", metadataItemID).Delete(&database.MetadataItemPerson{}).Error; err != nil {
			return err
		}
		for _, person := range people {
			name := strings.TrimSpace(person.Name)
			role := strings.TrimSpace(person.Role)
			if name == "" {
				continue
			}
			if role == "" {
				role = "actor"
			}
			personRow := database.Person{Name: name, SortName: strings.ToLower(name), AvatarURL: strings.TrimSpace(person.AvatarURL), TMDBPersonID: person.TMDBPersonID, IMDBID: strings.TrimSpace(person.IMDBID)}
			if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoUpdates: clause.AssignmentColumns([]string{"avatar_url", "tmdb_person_id", "imdb_id", "updated_at"})}).Create(&personRow).Error; err != nil {
				return err
			}
			if personRow.ID == 0 {
				if err := tx.WithContext(ctx).Where("name = ?", name).First(&personRow).Error; err != nil {
					return err
				}
			}
			link := database.MetadataItemPerson{MetadataItemID: metadataItemID, PersonID: personRow.ID, Role: role, Character: strings.TrimSpace(person.Character), SortOrder: person.SortOrder, SourceID: sourceID}
			if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "metadata_item_id"}, {Name: "person_id"}, {Name: "role"}}, DoUpdates: clause.AssignmentColumns([]string{"character", "sort_order", "source_id", "updated_at"})}).Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) applyNormalizedMetadataItemTags(ctx context.Context, metadataItemID uint, tags []NormalizedMetadataTag, sourceID *uint) error {
	if metadataItemID == 0 {
		return nil
	}
	byKind := map[string][]string{}
	for _, tag := range tags {
		kind := strings.ToLower(strings.TrimSpace(tag.Kind))
		name := strings.TrimSpace(tag.Name)
		if kind == "" || name == "" || kind != "genre" && kind != "keyword" {
			continue
		}
		byKind[kind] = appendUniqueStringFold(byKind[kind], name)
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for kind, names := range byKind {
			if err := tx.WithContext(ctx).Where("metadata_item_id = ? AND tag_id IN (SELECT id FROM tags WHERE kind = ?)", metadataItemID, kind).Delete(&database.MetadataItemTag{}).Error; err != nil {
				return err
			}
			for _, name := range names {
				tag := database.Tag{Kind: kind, Name: name}
				if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "kind"}, {Name: "name"}}, DoNothing: true}).Create(&tag).Error; err != nil {
					return err
				}
				if tag.ID == 0 {
					if err := tx.WithContext(ctx).Where("kind = ? AND name = ?", kind, name).First(&tag).Error; err != nil {
						return err
					}
				}
				link := database.MetadataItemTag{MetadataItemID: metadataItemID, TagID: tag.ID, SourceID: sourceID}
				if err := tx.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "metadata_item_id"}, {Name: "tag_id"}}, DoUpdates: clause.AssignmentColumns([]string{"source_id", "updated_at"})}).Create(&link).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
