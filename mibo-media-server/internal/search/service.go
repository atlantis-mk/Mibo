package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/library"
	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	library *library.Service
}

type Result struct {
	Item         database.MediaItem `json:"item"`
	WatchedState string             `json:"watched_state"`
	Highlight    string             `json:"highlight"`
}

type HistoryEntry struct {
	ID           uint      `json:"id"`
	Query        string    `json:"query"`
	TypeFilter   string    `json:"type_filter"`
	Genre        string    `json:"genre"`
	Region       string    `json:"region"`
	Year         *int      `json:"year,omitempty"`
	MinRating    *float64  `json:"min_rating,omitempty"`
	WatchedState string    `json:"watched_state"`
	Sort         string    `json:"sort"`
	LastUsedAt   time.Time `json:"last_used_at"`
}

func NewService(args ...any) *Service {
	service := &Service{}
	for _, arg := range args {
		switch value := arg.(type) {
		case *gorm.DB:
			service.db = value
		case *library.Service:
			service.library = value
		}
	}
	return service
}

func (s *Service) Status() map[string]any {
	return map[string]any{
		"status":            "active",
		"history":           "persistent",
		"sqlite_fts5_ready": s.db != nil,
	}
}

func (s *Service) Search(ctx context.Context, userID uint, input library.BrowseMediaItemsInput) ([]Result, error) {
	if s.library == nil {
		return nil, fmt.Errorf("search service unavailable")
	}
	userRef := userID
	items, err := s.library.DiscoverMediaItems(ctx, &userRef, input)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(items))
	for _, item := range items {
		results = append(results, Result{
			Item:         item.Item,
			WatchedState: item.WatchedState,
			Highlight:    highlightForItem(item.Item, input.Query),
		})
	}
	if strings.TrimSpace(input.Query) != "" {
		if err := s.recordHistory(ctx, userID, input); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (s *Service) ReindexMediaItem(ctx context.Context, mediaItemID uint) error {
	if s.db == nil || mediaItemID == 0 {
		return nil
	}

	var item database.MediaItem
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaItemID).
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.DeleteMediaItemDocument(ctx, mediaItemID)
		}
		return err
	}

	doc := buildSearchDocument(item)
	return s.db.WithContext(ctx).Save(&doc).Error
}

func (s *Service) DeleteMediaItemDocument(ctx context.Context, mediaItemID uint) error {
	if s.db == nil || mediaItemID == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Delete(&database.SearchDocument{}, "media_item_id = ?", mediaItemID).Error
}

func (s *Service) ReindexLibrary(ctx context.Context, libraryID uint, rootPath string) error {
	if s.db == nil || libraryID == 0 {
		return nil
	}

	itemQuery := s.db.WithContext(ctx).
		Model(&database.MediaItem{}).
		Where("library_id = ? AND deleted_at IS NULL", libraryID)
	if scope := strings.TrimSpace(rootPath); scope != "" {
		itemQuery = itemQuery.Where("source_path = ? OR source_path LIKE ?", scope, scope+"/%")
	}

	var items []database.MediaItem
	if err := itemQuery.Find(&items).Error; err != nil {
		return err
	}
	for _, item := range items {
		if err := s.ReindexMediaItem(ctx, item.ID); err != nil {
			return err
		}
	}

	deleteQuery := s.db.WithContext(ctx).Table("search_documents").
		Joins("LEFT JOIN media_items ON media_items.id = search_documents.media_item_id").
		Where("search_documents.library_id = ?", libraryID).
		Where("media_items.id IS NULL OR media_items.deleted_at IS NOT NULL")
	if scope := strings.TrimSpace(rootPath); scope != "" {
		deleteQuery = deleteQuery.Where("media_items.source_path = ? OR media_items.source_path LIKE ? OR media_items.id IS NULL", scope, scope+"/%")
	}
	var staleIDs []uint
	if err := deleteQuery.Pluck("search_documents.media_item_id", &staleIDs).Error; err != nil {
		return err
	}
	if len(staleIDs) > 0 {
		return s.db.WithContext(ctx).Delete(&database.SearchDocument{}, "media_item_id IN ?", staleIDs).Error
	}
	return nil
}

func (s *Service) ListHistory(ctx context.Context, userID uint, limit int) ([]HistoryEntry, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	var rows []database.SearchHistory
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("last_used_at desc, id desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	entries := make([]HistoryEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, HistoryEntry{
			ID:           row.ID,
			Query:        row.Query,
			TypeFilter:   row.TypeFilter,
			Genre:        row.Genre,
			Region:       row.Region,
			Year:         row.Year,
			MinRating:    row.MinRating,
			WatchedState: row.WatchedState,
			Sort:         row.Sort,
			LastUsedAt:   row.LastUsedAt,
		})
	}
	return entries, nil
}

func (s *Service) recordHistory(ctx context.Context, userID uint, input library.BrowseMediaItemsInput) error {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil
	}
	var existing database.SearchHistory
	lookup := s.db.WithContext(ctx).Where(
		"user_id = ? AND query = ? AND type_filter = ? AND genre = ? AND region = ? AND watched_state = ? AND sort = ? AND deleted_at IS NULL",
		userID,
		query,
		string(input.TypeFilter),
		input.Genre,
		input.Region,
		string(input.Watched),
		string(input.Sort),
	)
	if input.Year == nil {
		lookup = lookup.Where("year IS NULL")
	} else {
		lookup = lookup.Where("year = ?", *input.Year)
	}
	if input.MinRating == nil {
		lookup = lookup.Where("min_rating IS NULL")
	} else {
		lookup = lookup.Where("min_rating = ?", *input.MinRating)
	}
	err := lookup.First(&existing).Error
	now := time.Now().UTC()
	if err == nil {
		existing.LastUsedAt = now
		return s.db.WithContext(ctx).Save(&existing).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	entry := database.SearchHistory{
		UserID:       userID,
		Query:        query,
		TypeFilter:   string(input.TypeFilter),
		Genre:        input.Genre,
		Region:       input.Region,
		Year:         input.Year,
		MinRating:    input.MinRating,
		WatchedState: string(input.Watched),
		Sort:         string(input.Sort),
		LastUsedAt:   now,
	}
	return s.db.WithContext(ctx).Create(&entry).Error
}

func highlightForItem(item database.MediaItem, query string) string {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return ""
	}
	fields := []string{item.Title, item.OriginalTitle, item.SeriesTitle, item.CastJSON, item.DirectorsJSON}
	for _, field := range fields {
		if field == "" {
			continue
		}
		lower := strings.ToLower(field)
		idx := strings.Index(lower, needle)
		if idx == -1 {
			continue
		}
		start := max(0, idx-18)
		end := min(len(field), idx+len(query)+18)
		return strings.TrimSpace(field[start:end])
	}
	return ""
}

type searchPerson struct {
	Name string `json:"name"`
}

func buildSearchDocument(item database.MediaItem) database.SearchDocument {
	return database.SearchDocument{
		MediaItemID:         item.ID,
		LibraryID:           item.LibraryID,
		MediaType:           searchDocumentMediaType(item),
		Title:               strings.TrimSpace(item.Title),
		OriginalTitle:       strings.TrimSpace(item.OriginalTitle),
		SeriesTitle:         strings.TrimSpace(item.SeriesTitle),
		Overview:            strings.TrimSpace(item.Overview),
		SearchPeopleText:    strings.Join(searchPeopleTokens(item), " "),
		SearchGenresText:    strings.Join(searchStringTokens(item.GenresJSON), " "),
		SearchCountriesText: strings.Join(searchStringTokens(item.RegionsJSON), " "),
		Year:                item.Year,
		VoteAverage:         item.VoteAverage,
		UpdatedAt:           time.Now().UTC(),
	}
}

func searchDocumentMediaType(item database.MediaItem) string {
	if strings.EqualFold(item.Type, "movie") {
		return "movie"
	}
	return string(library.BrowseTypeFilterShow)
}

func searchPeopleTokens(item database.MediaItem) []string {
	people := append(searchPeopleJSON(item.CastJSON), searchPeopleJSON(item.DirectorsJSON)...)
	return dedupeSearchTokens(people)
}

func searchPeopleJSON(raw string) []string {
	var people []searchPerson
	if err := json.Unmarshal([]byte(raw), &people); err != nil {
		return nil
	}
	names := make([]string, 0, len(people))
	for _, person := range people {
		if name := strings.TrimSpace(person.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func searchStringTokens(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return dedupeSearchTokens(values)
}

func dedupeSearchTokens(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
