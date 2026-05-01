package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type MetadataFieldChange struct {
	ItemID     uint
	FieldKey   string
	Value      any
	SourceID   *uint
	Confidence *float64
	ApplyMode  string
	Force      bool
}

func (s *Service) applyMetadataFieldChanges(ctx context.Context, changes []MetadataFieldChange) ([]MetadataAppliedField, []MetadataSkippedField, error) {
	applied := make([]MetadataAppliedField, 0, len(changes))
	skipped := make([]MetadataSkippedField, 0)
	catalogSvc := catalog.NewService(s.db)
	for _, change := range changes {
		fieldKey := strings.TrimSpace(change.FieldKey)
		if change.ItemID == 0 || fieldKey == "" {
			return nil, nil, fmt.Errorf("metadata field change requires item id and field key")
		}
		locked, err := s.fieldLocked(ctx, change.ItemID, fieldKey)
		if err != nil {
			return nil, nil, err
		}
		if locked && !change.Force && change.ApplyMode != FieldApplyModeManual {
			skipped = append(skipped, MetadataSkippedField{ItemID: change.ItemID, FieldKey: fieldKey, Reason: "locked"})
			continue
		}
		_, didApply, err := catalogSvc.ApplyField(ctx, catalog.ApplyFieldInput{ItemID: change.ItemID, FieldKey: fieldKey, Value: change.Value, SourceID: change.SourceID, Force: change.Force || change.ApplyMode == FieldApplyModeManual})
		if err != nil {
			return nil, nil, err
		}
		if !didApply {
			skipped = append(skipped, MetadataSkippedField{ItemID: change.ItemID, FieldKey: fieldKey, Reason: "not_applied"})
			continue
		}
		if change.SourceID != nil {
			if err := s.db.WithContext(ctx).Model(&database.MetadataFieldState{}).Where("item_id = ? AND field_key = ?", change.ItemID, fieldKey).Update("source_id", *change.SourceID).Error; err != nil {
				return nil, nil, err
			}
		}
		applied = append(applied, MetadataAppliedField{ItemID: change.ItemID, FieldKey: fieldKey, SourceID: change.SourceID, ApplyMode: change.ApplyMode, Confidence: change.Confidence})
	}
	return applied, skipped, nil
}

func (s *Service) fieldLocked(ctx context.Context, itemID uint, fieldKey string) (bool, error) {
	var state database.MetadataFieldState
	err := s.db.WithContext(ctx).Where("item_id = ? AND field_key = ?", itemID, fieldKey).First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !state.IsLocked {
		return false, nil
	}
	var existing any
	_ = json.Unmarshal([]byte(state.ValueJSON), &existing)
	return true, nil
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
	if detail.LastAirDate != "" {
		if parsed := parseProviderDate(detail.LastAirDate); parsed != nil {
			add("last_air_date", *parsed)
		}
	}
	return changes
}
