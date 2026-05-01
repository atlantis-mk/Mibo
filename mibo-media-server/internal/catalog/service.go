package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ItemTypeMovie      = "movie"
	ItemTypeSeries     = "series"
	ItemTypeSeason     = "season"
	ItemTypeEpisode    = "episode"
	ItemTypeExtra      = "extra"
	ItemTypeCollection = "collection"

	DisplayOrderAired = "aired"

	AvailabilityAvailable    = "available"
	AvailabilityMissing      = "missing"
	AvailabilityUnaired      = "unaired"
	AvailabilityNoLocalMedia = "no_local_media"

	GovernancePending     = "pending"
	GovernanceMatched     = "matched"
	GovernanceNeedsReview = "needs_review"
	GovernanceLocked      = "locked"
	GovernanceManual      = "manual"
	GovernanceUnmatched   = "unmatched"

	SourceTypeProvider  = "provider"
	SourceTypeLocalFile = "local_file"
	SourceTypeManual    = "manual"
	SourceTypeNFO       = "nfo"
)

type Service struct {
	db                     *gorm.DB
	personProfileRefresher PersonProfileRefresher
}

type PersonProfileRefresher interface {
	RefreshCatalogPersonProfile(ctx context.Context, personID uint) error
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) SetPersonProfileRefresher(refresher PersonProfileRefresher) {
	s.personProfileRefresher = refresher
}

type CreateItemInput struct {
	LibraryID          uint
	Type               string
	ParentID           *uint
	Path               string
	SortKey            string
	DisplayOrder       string
	IndexNumber        *int
	IndexNumberEnd     *int
	ParentIndexNumber  *int
	AbsoluteNumber     *int
	Title              string
	OriginalTitle      string
	SortTitle          string
	Overview           string
	ReleaseDate        *time.Time
	FirstAirDate       *time.Time
	LastAirDate        *time.Time
	Year               *int
	EndYear            *int
	RuntimeSeconds     *int
	CommunityRating    *float64
	OfficialRating     string
	SeriesStatus       string
	AvailabilityStatus string
	GovernanceStatus   string
}

type MetadataSourceInput struct {
	ItemID               uint
	SourceType           string
	SourceName           string
	Language             string
	ExternalID           string
	MetadataProfileID    *uint
	MetadataProfileName  string
	ProviderInstanceID   *uint
	ProviderInstanceName string
	FallbackSummaryJSON  string
	PayloadJSON          string
	Confidence           *float64
	FetchedAt            time.Time
	ExpiresAt            *time.Time
}

type ExternalIDInput struct {
	ItemID       uint
	Provider     string
	ProviderType string
	ExternalID   string
	IsPrimary    bool
	Source       string
	Confidence   *float64
}

type ApplyFieldInput struct {
	ItemID         uint
	FieldKey       string
	Value          any
	SourceID       *uint
	Lock           bool
	LockReason     string
	EditedByUserID *uint
	Force          bool
}

type CorrectEpisodeNumberingInput struct {
	EpisodeID        uint
	SeasonNumber     int
	EpisodeNumber    int
	EpisodeNumberEnd *int
}

func (s *Service) CreateItem(ctx context.Context, input CreateItemInput) (database.CatalogItem, error) {
	if input.LibraryID == 0 {
		return database.CatalogItem{}, errors.New("library id is required")
	}
	if strings.TrimSpace(input.Type) == "" {
		return database.CatalogItem{}, errors.New("item type is required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return database.CatalogItem{}, errors.New("title is required")
	}

	item := database.CatalogItem{
		LibraryID:           input.LibraryID,
		Type:                strings.TrimSpace(input.Type),
		ParentID:            input.ParentID,
		Path:                strings.TrimSpace(input.Path),
		SortKey:             strings.TrimSpace(input.SortKey),
		DisplayOrder:        defaultString(input.DisplayOrder, DisplayOrderAired),
		IndexNumber:         input.IndexNumber,
		IndexNumberEnd:      input.IndexNumberEnd,
		ParentIndexNumber:   input.ParentIndexNumber,
		AbsoluteNumber:      input.AbsoluteNumber,
		Title:               strings.TrimSpace(input.Title),
		OriginalTitle:       strings.TrimSpace(input.OriginalTitle),
		SortTitle:           strings.TrimSpace(input.SortTitle),
		Overview:            input.Overview,
		ReleaseDate:         input.ReleaseDate,
		FirstAirDate:        input.FirstAirDate,
		LastAirDate:         input.LastAirDate,
		Year:                input.Year,
		EndYear:             input.EndYear,
		RuntimeSeconds:      input.RuntimeSeconds,
		CommunityRating:     input.CommunityRating,
		OfficialRating:      strings.TrimSpace(input.OfficialRating),
		SeriesStatus:        strings.TrimSpace(input.SeriesStatus),
		AvailabilityStatus:  defaultString(input.AvailabilityStatus, AvailabilityNoLocalMedia),
		GovernanceStatus:    defaultString(input.GovernanceStatus, GovernancePending),
		CanonicalVersion:    1,
		LastCanonicalizedAt: timePtr(time.Now().UTC()),
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if item.ParentID != nil {
			var parent database.CatalogItem
			if err := tx.First(&parent, *item.ParentID).Error; err != nil {
				return fmt.Errorf("load parent item: %w", err)
			}
			if parent.RootID != nil {
				item.RootID = parent.RootID
			} else {
				item.RootID = &parent.ID
			}
		}

		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if item.RootID == nil {
			item.RootID = &item.ID
			if err := tx.Model(&item).Update("root_id", item.ID).Error; err != nil {
				return err
			}
		}
		return s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: item.ID})
	})
	return item, err
}

func (s *Service) ListChildren(ctx context.Context, parentID uint) ([]database.CatalogItem, error) {
	var items []database.CatalogItem
	err := s.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("parent_index_number asc").
		Order("index_number asc").
		Order("sort_key asc").
		Order("title asc").
		Order("id asc").
		Find(&items).Error
	return items, err
}

func (s *Service) RecordMetadataSource(ctx context.Context, input MetadataSourceInput) (database.MetadataSource, error) {
	if input.ItemID == 0 {
		return database.MetadataSource{}, errors.New("item id is required")
	}
	if strings.TrimSpace(input.SourceType) == "" {
		return database.MetadataSource{}, errors.New("source type is required")
	}
	if strings.TrimSpace(input.SourceName) == "" {
		return database.MetadataSource{}, errors.New("source name is required")
	}
	if input.FetchedAt.IsZero() {
		input.FetchedAt = time.Now().UTC()
	}

	source := database.MetadataSource{
		ItemID:               input.ItemID,
		SourceType:           strings.TrimSpace(input.SourceType),
		SourceName:           strings.TrimSpace(input.SourceName),
		Language:             strings.TrimSpace(input.Language),
		ExternalID:           strings.TrimSpace(input.ExternalID),
		MetadataProfileID:    input.MetadataProfileID,
		MetadataProfileName:  strings.TrimSpace(input.MetadataProfileName),
		ProviderInstanceID:   input.ProviderInstanceID,
		ProviderInstanceName: strings.TrimSpace(input.ProviderInstanceName),
		FallbackSummaryJSON:  strings.TrimSpace(input.FallbackSummaryJSON),
		PayloadJSON:          input.PayloadJSON,
		Confidence:           input.Confidence,
		FetchedAt:            input.FetchedAt,
		ExpiresAt:            input.ExpiresAt,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&source).Error; err != nil {
			return err
		}
		return s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: input.ItemID})
	})
	return source, err
}

func (s *Service) SetExternalID(ctx context.Context, input ExternalIDInput) (database.CatalogExternalID, error) {
	if input.ItemID == 0 {
		return database.CatalogExternalID{}, errors.New("item id is required")
	}
	if strings.TrimSpace(input.Provider) == "" || strings.TrimSpace(input.ProviderType) == "" || strings.TrimSpace(input.ExternalID) == "" {
		return database.CatalogExternalID{}, errors.New("provider, provider type, and external id are required")
	}

	externalID := database.CatalogExternalID{
		ItemID:       input.ItemID,
		Provider:     strings.TrimSpace(input.Provider),
		ProviderType: strings.TrimSpace(input.ProviderType),
		ExternalID:   strings.TrimSpace(input.ExternalID),
		IsPrimary:    input.IsPrimary,
		Source:       strings.TrimSpace(input.Source),
		Confidence:   input.Confidence,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "provider_type"}, {Name: "external_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"item_id", "is_primary", "source", "confidence", "updated_at"}),
		}).Create(&externalID).Error; err != nil {
			return err
		}
		return s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: input.ItemID})
	})
	return externalID, err
}

func (s *Service) ApplyField(ctx context.Context, input ApplyFieldInput) (database.MetadataFieldState, bool, error) {
	if input.ItemID == 0 {
		return database.MetadataFieldState{}, false, errors.New("item id is required")
	}
	input.FieldKey = strings.TrimSpace(input.FieldKey)
	if input.FieldKey == "" {
		return database.MetadataFieldState{}, false, errors.New("field key is required")
	}

	valueJSON, err := json.Marshal(input.Value)
	if err != nil {
		return database.MetadataFieldState{}, false, fmt.Errorf("marshal field value: %w", err)
	}

	var state database.MetadataFieldState
	applied := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("item_id = ? AND field_key = ?", input.ItemID, input.FieldKey).First(&state).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && state.IsLocked && !input.Force {
			return nil
		}

		now := time.Now().UTC()
		state.ItemID = input.ItemID
		state.FieldKey = input.FieldKey
		state.SourceID = input.SourceID
		state.ValueJSON = string(valueJSON)
		state.IsLocked = input.Lock
		state.LockReason = strings.TrimSpace(input.LockReason)
		state.EditedByUserID = input.EditedByUserID
		if input.EditedByUserID != nil {
			state.EditedAt = &now
		}

		if state.ID == 0 {
			if err := tx.Create(&state).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&state).Error; err != nil {
			return err
		}

		updates, err := catalogItemUpdate(input.FieldKey, input.Value, now)
		if err != nil {
			return err
		}
		if len(updates) > 0 {
			if err := tx.Model(&database.CatalogItem{}).Where("id = ?", input.ItemID).Updates(updates).Error; err != nil {
				return err
			}
		}
		applied = true
		return s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: input.ItemID})
	})
	return state, applied, err
}

func (s *Service) CorrectEpisodeNumbering(ctx context.Context, input CorrectEpisodeNumberingInput) (database.CatalogItem, error) {
	if input.EpisodeID == 0 {
		return database.CatalogItem{}, errors.New("episode id is required")
	}
	if input.SeasonNumber < 0 || input.EpisodeNumber <= 0 {
		return database.CatalogItem{}, errors.New("season_number and episode_number are required")
	}

	var updated database.CatalogItem
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var episode database.CatalogItem
		if err := tx.Where("id = ? AND deleted_at IS NULL", input.EpisodeID).First(&episode).Error; err != nil {
			return err
		}
		if episode.Type != ItemTypeEpisode {
			return errors.New("item must be an episode")
		}
		if episode.RootID == nil || *episode.RootID == 0 {
			return errors.New("episode lacks series root")
		}

		var series database.CatalogItem
		if err := tx.Where("id = ? AND type = ? AND deleted_at IS NULL", *episode.RootID, ItemTypeSeries).First(&series).Error; err != nil {
			return err
		}

		var season database.CatalogItem
		err := tx.Where("parent_id = ? AND type = ? AND index_number = ? AND deleted_at IS NULL", series.ID, ItemTypeSeason, input.SeasonNumber).First(&season).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			seasonNumber := input.SeasonNumber
			season = database.CatalogItem{
				LibraryID:           series.LibraryID,
				Type:                ItemTypeSeason,
				ParentID:            &series.ID,
				RootID:              &series.ID,
				Path:                strings.TrimRight(series.Path, "/") + fmt.Sprintf("/Season %02d", input.SeasonNumber),
				SortKey:             fmt.Sprintf("%s S%02d", strings.TrimSpace(series.Title), input.SeasonNumber),
				DisplayOrder:        DisplayOrderAired,
				IndexNumber:         &seasonNumber,
				ParentIndexNumber:   &seasonNumber,
				Title:               fmt.Sprintf("Season %d", input.SeasonNumber),
				AvailabilityStatus:  AvailabilityNoLocalMedia,
				GovernanceStatus:    GovernancePending,
				CanonicalVersion:    1,
				LastCanonicalizedAt: timePtr(time.Now().UTC()),
			}
			if err := tx.Create(&season).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		var conflict database.CatalogItem
		err = tx.Where("parent_id = ? AND type = ? AND index_number = ? AND id <> ? AND deleted_at IS NULL", season.ID, ItemTypeEpisode, input.EpisodeNumber, episode.ID).First(&conflict).Error
		if err == nil {
			return fmt.Errorf("target episode slot already occupied by item %d", conflict.ID)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		updates := map[string]any{
			"parent_id":             season.ID,
			"root_id":               series.ID,
			"parent_index_number":   input.SeasonNumber,
			"index_number":          input.EpisodeNumber,
			"index_number_end":      input.EpisodeNumberEnd,
			"last_canonicalized_at": time.Now().UTC(),
		}
		if err := tx.Model(&database.CatalogItem{}).Where("id = ?", episode.ID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", episode.ID).First(&updated).Error; err != nil {
			return err
		}
		for _, itemID := range []uint{episode.ID, season.ID, series.ID} {
			if err := s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: itemID}); err != nil {
				return err
			}
		}
		if episode.ParentID != nil && *episode.ParentID != season.ID {
			if err := s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: *episode.ParentID}); err != nil {
				return err
			}
		}
		return nil
	})
	return updated, err
}

func catalogItemUpdate(fieldKey string, value any, now time.Time) (map[string]any, error) {
	updates := map[string]any{"last_canonicalized_at": now}
	switch fieldKey {
	case "title", "original_title", "sort_title", "overview", "availability_status", "governance_status", "series_status", "official_rating":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("field %s requires a string value", fieldKey)
		}
		updates[fieldKey] = text
	case "year", "runtime_seconds":
		intValue, ok := asInt(value)
		if !ok {
			return nil, fmt.Errorf("field %s requires an integer value", fieldKey)
		}
		updates[fieldKey] = intValue
	case "community_rating":
		floatValue, ok := asFloat(value)
		if !ok {
			return nil, fmt.Errorf("field %s requires a numeric value", fieldKey)
		}
		updates[fieldKey] = floatValue
	case "release_date", "first_air_date", "last_air_date":
		dateValue, ok := asTime(value)
		if !ok {
			return nil, fmt.Errorf("field %s requires a date value", fieldKey)
		}
		updates[fieldKey] = dateValue
	default:
		return updates, nil
	}
	return updates, nil
}

func asFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	}
	return 0, false
}

func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if typed == float64(int(typed)) {
			return int(typed), true
		}
	}
	return 0, false
}

func asTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed.UTC(), true
	case *time.Time:
		if typed == nil {
			return time.Time{}, false
		}
		return typed.UTC(), true
	case string:
		parsed := parseCatalogDate(typed)
		if parsed == nil {
			return time.Time{}, false
		}
		return *parsed, true
	default:
		return time.Time{}, false
	}
}

func parseCatalogDate(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func timePtr(value time.Time) *time.Time {
	return &value
}
