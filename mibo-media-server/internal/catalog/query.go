package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog/seriesplayback"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"gorm.io/gorm"
)

type CatalogLatestByLibrarySection struct {
	LibraryID   uint              `json:"library_id"`
	LibraryName string            `json:"library_name"`
	Items       []CatalogListItem `json:"items"`
}

type BrowseItemsInput struct {
	LibraryID       uint
	Query           string
	TypeFilter      string
	Genre           string
	Region          string
	Year            *int
	MinRating       *float64
	WatchedState    string
	OrganizingState string
	Sort            string
	SortDirection   string
	Limit           int
	Offset          int
	UserID          uint
}

type BrowseItemsResult struct {
	Items         []CatalogListItem `json:"items"`
	Total         int64             `json:"total"`
	Limit         int               `json:"limit"`
	Offset        int               `json:"offset"`
	HasMore       bool              `json:"has_more"`
	Sort          string            `json:"sort"`
	SortDirection string            `json:"sort_direction"`
}

type browseListEntry struct {
	Item      CatalogListItem
	TitleKey  string
	Year      *int
	CreatedAt string
	StableID  uint
}

func (s *Service) ListLibraryItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	if libraryID == 0 {
		return nil, errors.New("library id is required")
	}
	return s.listItems(ctx, &libraryID, query, typeFilter, limit)
}

func (s *Service) ListItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	var libraryFilter *uint
	if libraryID != 0 {
		libraryFilter = &libraryID
	}
	return s.listItems(ctx, libraryFilter, query, typeFilter, limit)
}

func (s *Service) SearchItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	if strings.TrimSpace(query) == "" {
		return s.ListItems(ctx, libraryID, "", typeFilter, limit)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	allowedTypes := []string{ItemTypeMovie, ItemTypeSeries}
	switch strings.ToLower(strings.TrimSpace(typeFilter)) {
	case ItemTypeMovie:
		allowedTypes = []string{ItemTypeMovie}
	case ItemTypeSeries, "show":
		allowedTypes = []string{ItemTypeSeries}
	}

	db := s.db.WithContext(ctx).
		Model(&database.CatalogSearchDocument{}).
		Where("item_type IN ?", allowedTypes)
	if libraryID != 0 {
		db = db.Where("library_id = ?", libraryID)
	}
	like := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	db = db.Where("LOWER(title) LIKE ? OR LOWER(original_title) LIKE ? OR LOWER(people_text) LIKE ? OR LOWER(tags_text) LIKE ? OR LOWER(provider_ids_text) LIKE ?", like, like, like, like, like)

	var docs []database.CatalogSearchDocument
	if err := db.Order("title asc").Order("item_id asc").Limit(limit).Find(&docs).Error; err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return []CatalogListItem{}, nil
	}
	itemIDs := make([]uint, 0, len(docs))
	for _, doc := range docs {
		itemIDs = append(itemIDs, doc.ItemID)
	}
	var items []database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", itemIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	itemByID := make(map[uint]database.CatalogItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}
	ordered := make([]database.CatalogItem, 0, len(docs))
	for _, doc := range docs {
		if item, ok := itemByID[doc.ItemID]; ok {
			ordered = append(ordered, item)
		}
	}
	return s.buildCatalogListItems(ctx, ordered)
}

func (s *Service) BrowseItems(ctx context.Context, input BrowseItemsInput) (BrowseItemsResult, error) {
	input = normalizeBrowseItemsInput(input)
	db := s.db.WithContext(ctx).
		Model(&database.CatalogItem{}).
		Where("catalog_items.deleted_at IS NULL").
		Where("catalog_items.availability_status = ?", AvailabilityAvailable)
	if input.LibraryID != 0 {
		db = db.Where("catalog_items.library_id = ?", input.LibraryID)
	}
	switch input.TypeFilter {
	case ItemTypeMovie:
		db = db.Where("catalog_items.parent_id IS NULL").Where("catalog_items.type = ?", ItemTypeMovie)
	case ItemTypeEpisode:
		db = db.Where("catalog_items.type = ?", ItemTypeEpisode)
	case ItemTypeSeries, "show":
		db = db.Where("catalog_items.parent_id IS NULL").Where("catalog_items.type = ?", ItemTypeSeries)
	default:
		db = db.Where("catalog_items.parent_id IS NULL").Where("catalog_items.type IN ?", []string{ItemTypeMovie, ItemTypeSeries})
	}
	if input.Year != nil {
		db = db.Where("catalog_items.year = ?", *input.Year)
	}
	if input.MinRating != nil {
		db = db.Where("catalog_items.community_rating IS NOT NULL AND catalog_items.community_rating >= ?", *input.MinRating)
	}
	if query := strings.TrimSpace(input.Query); query != "" {
		like := "%" + strings.ToLower(query) + "%"
		db = db.Where(`(LOWER(catalog_items.title) LIKE ?
			OR LOWER(catalog_items.original_title) LIKE ?
			OR LOWER(catalog_items.sort_title) LIKE ?
			OR EXISTS (
				SELECT 1 FROM catalog_search_documents
				WHERE catalog_search_documents.item_id = catalog_items.id
				AND (LOWER(catalog_search_documents.people_text) LIKE ?
					OR LOWER(catalog_search_documents.tags_text) LIKE ?
					OR LOWER(catalog_search_documents.provider_ids_text) LIKE ?)
			))`, like, like, like, like, like, like)
	}
	if genre := strings.TrimSpace(input.Genre); genre != "" {
		db = db.Where(`EXISTS (
			SELECT 1 FROM catalog_search_documents
			WHERE catalog_search_documents.item_id = catalog_items.id
			AND LOWER(catalog_search_documents.tags_text) LIKE ?
		)`, "%"+strings.ToLower(genre)+"%")
	}
	if region := strings.TrimSpace(input.Region); region != "" {
		db = db.Where(`EXISTS (
			SELECT 1 FROM catalog_search_documents
			WHERE catalog_search_documents.item_id = catalog_items.id
			AND LOWER(catalog_search_documents.tags_text) LIKE ?
		)`, "%"+strings.ToLower(region)+"%")
	}
	if input.WatchedState != "all" {
		db = db.Joins("LEFT JOIN user_item_data browse_user_item_data ON browse_user_item_data.item_id = catalog_items.id AND browse_user_item_data.asset_id IS NULL AND browse_user_item_data.user_id = ?", input.UserID)
		switch input.WatchedState {
		case "watched":
			db = db.Where("browse_user_item_data.completed_at IS NOT NULL")
		case "in_progress":
			db = db.Where("browse_user_item_data.completed_at IS NULL AND browse_user_item_data.position_seconds > 0")
		case "unwatched":
			db = db.Where("browse_user_item_data.id IS NULL OR (browse_user_item_data.completed_at IS NULL AND browse_user_item_data.position_seconds = 0)")
		}
	}

	var items []database.CatalogItem
	if err := applyBrowseItemsOrder(db, input).Find(&items).Error; err != nil {
		return BrowseItemsResult{}, err
	}
	mapped, err := s.buildCatalogListItems(ctx, items)
	if err != nil {
		return BrowseItemsResult{}, err
	}
	if err := s.attachOrganizingSummaries(ctx, mapped); err != nil {
		return BrowseItemsResult{}, err
	}
	entries := make([]browseListEntry, 0, len(mapped))
	for idx, item := range mapped {
		entries = append(entries, browseListEntry{Item: item, TitleKey: catalogListItemTitleKey(item), Year: item.Year, CreatedAt: items[idx].CreatedAt.Format("2006-01-02T15:04:05.000000000Z07:00"), StableID: item.ID})
	}
	discovered, err := s.discoveredBrowseEntries(ctx, input)
	if err != nil {
		return BrowseItemsResult{}, err
	}
	entries = append(entries, discovered...)
	entries = filterBrowseListEntriesByOrganizing(entries, input.OrganizingState)
	applyBrowseListEntryOrder(entries, input)
	total := int64(len(entries))
	start := input.Offset
	if start > len(entries) {
		start = len(entries)
	}
	end := start + input.Limit
	if end > len(entries) {
		end = len(entries)
	}
	paged := make([]CatalogListItem, 0, end-start)
	for _, entry := range entries[start:end] {
		paged = append(paged, entry.Item)
	}
	return BrowseItemsResult{
		Items:         paged,
		Total:         total,
		Limit:         input.Limit,
		Offset:        input.Offset,
		HasMore:       int64(input.Offset+len(paged)) < total,
		Sort:          input.Sort,
		SortDirection: input.SortDirection,
	}, nil
}

func (s *Service) discoveredBrowseEntries(ctx context.Context, input BrowseItemsInput) ([]browseListEntry, error) {
	if input.OrganizingState == "organized" {
		return nil, nil
	}
	if input.TypeFilter == ItemTypeSeries || input.TypeFilter == "show" || input.TypeFilter == ItemTypeEpisode {
		return nil, nil
	}
	if input.Genre != "" || input.Region != "" || input.Year != nil || input.MinRating != nil || input.WatchedState == "watched" || input.WatchedState == "in_progress" {
		return nil, nil
	}
	query := s.db.WithContext(ctx).
		Model(&database.InventoryFile{}).
		Where("inventory_files.deleted_at IS NULL").
		Where("inventory_files.status = ?", "available").
		Where("inventory_files.content_class = ?", "video").
		Where(`NOT EXISTS (
			SELECT 1 FROM asset_files
			JOIN media_assets ON media_assets.id = asset_files.asset_id AND media_assets.deleted_at IS NULL
			JOIN asset_items ON asset_items.asset_id = media_assets.id
			JOIN catalog_items ON catalog_items.id = asset_items.item_id AND catalog_items.deleted_at IS NULL
			WHERE asset_files.file_id = inventory_files.id
		)`)
	if input.LibraryID != 0 {
		query = query.Where("inventory_files.library_id = ?", input.LibraryID)
	}
	if input.WatchedState == "unwatched" {
		query = query.Where(`NOT EXISTS (
			SELECT 1 FROM user_item_data
			WHERE user_item_data.asset_id IN (
				SELECT asset_files.asset_id FROM asset_files WHERE asset_files.file_id = inventory_files.id
			)
			AND user_item_data.user_id = ?
			AND (user_item_data.completed_at IS NOT NULL OR user_item_data.position_seconds > 0)
		)`, input.UserID)
	}
	var files []database.InventoryFile
	if err := query.Find(&files).Error; err != nil {
		return nil, err
	}
	conditionByFileID, err := s.organizingConditionsByInventoryFile(ctx, files)
	if err != nil {
		return nil, err
	}
	thumbnailByFileID, err := s.inventoryFileThumbnailURLs(ctx, files)
	if err != nil {
		return nil, err
	}
	entries := make([]browseListEntry, 0, len(files))
	for _, file := range files {
		title := discoveredTitleFromPath(file.StoragePath)
		if input.Query != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(input.Query)) && !strings.Contains(strings.ToLower(file.StoragePath), strings.ToLower(input.Query)) {
			continue
		}
		fileID := file.ID
		state := defaultString(file.ScanState, "discovered")
		item := CatalogListItem{
			ID:                 0,
			LibraryID:          file.LibraryID,
			SourceKind:         "inventory_file",
			InventoryFileID:    &fileID,
			MaturityState:      state,
			Organizing:         state == "discovered" || state == "review_required",
			StoragePath:        strings.TrimSpace(file.StoragePath),
			Type:               ItemTypeMovie,
			Title:              title,
			SortTitle:          title,
			AvailabilityStatus: AvailabilityAvailable,
			GovernanceStatus:   GovernancePending,
		}
		if summary := buildOrganizingSummary(conditionByFileID[file.ID], true); summary != nil {
			item.OrganizingSummary = summary
			item.Organizing = summary.State == "organizing" || summary.State == "review_required" || summary.State == "failed"
		}
		if thumbnailURL := strings.TrimSpace(thumbnailByFileID[file.ID]); thumbnailURL != "" {
			item.SelectedImages = []CatalogSelectedImage{{ImageType: "poster", URL: thumbnailURL}}
		}
		entries = append(entries, browseListEntry{Item: item, TitleKey: strings.ToLower(title), CreatedAt: file.CreatedAt.Format("2006-01-02T15:04:05.000000000Z07:00"), StableID: file.ID})
	}
	return entries, nil
}

func (s *Service) inventoryFileThumbnailURLs(ctx context.Context, files []database.InventoryFile) (map[uint]string, error) {
	thumbnails := make(map[uint]string)
	if len(files) == 0 {
		return thumbnails, nil
	}
	paths := make([]string, 0, len(files))
	fileIDByKey := make(map[string]uint, len(files))
	for _, file := range files {
		if thumbnailURL := normalizeThumbnailURL(file.ThumbnailURL); thumbnailURL != "" {
			thumbnails[file.ID] = thumbnailURL
			continue
		}
		provider := strings.TrimSpace(file.StorageProvider)
		storagePath := strings.TrimSpace(file.StoragePath)
		if provider == "" || storagePath == "" {
			continue
		}
		paths = append(paths, storagePath)
		fileIDByKey[provider+"\x00"+storagePath] = file.ID
	}
	if len(paths) == 0 {
		return thumbnails, nil
	}
	var entries []database.StorageIndexEntry
	if err := s.db.WithContext(ctx).
		Where("storage_path IN ? AND observation_status = ?", paths, "present").
		Find(&entries).Error; err != nil {
		return nil, err
	}
	for _, entry := range entries {
		fileID := fileIDByKey[strings.TrimSpace(entry.StorageProvider)+"\x00"+strings.TrimSpace(entry.StoragePath)]
		if fileID == 0 {
			continue
		}
		thumbnailURL := thumbnailURLFromProviderMeta(entry.ProviderMetaJSON)
		if thumbnailURL == "" {
			continue
		}
		thumbnails[fileID] = thumbnailURL
	}
	return thumbnails, nil
}

func thumbnailURLFromProviderMeta(providerMetaJSON string) string {
	if strings.TrimSpace(providerMetaJSON) == "" {
		return ""
	}
	var meta map[string]string
	if err := json.Unmarshal([]byte(providerMetaJSON), &meta); err != nil {
		return ""
	}
	return normalizeThumbnailURL(meta["thumbnail_url"])
}

func normalizeThumbnailURL(value string) string {
	thumbnailURL := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(thumbnailURL), "http://") && !strings.HasPrefix(strings.ToLower(thumbnailURL), "https://") {
		return ""
	}
	return thumbnailURL
}

func filterBrowseListEntriesByOrganizing(entries []browseListEntry, organizingState string) []browseListEntry {
	switch organizingState {
	case "organized", "unorganized":
	default:
		return entries
	}
	filtered := make([]browseListEntry, 0, len(entries))
	for _, entry := range entries {
		if organizingState == "organized" && !entry.Item.Organizing {
			filtered = append(filtered, entry)
		}
		if organizingState == "unorganized" && entry.Item.Organizing {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func (s *Service) attachOrganizingSummaries(ctx context.Context, items []CatalogListItem) error {
	if len(items) == 0 {
		return nil
	}
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.ID != 0 {
			itemIDs = append(itemIDs, item.ID)
		}
	}
	if len(itemIDs) == 0 {
		return nil
	}
	conditions, err := s.organizingConditionsByCatalogItemID(ctx, itemIDs)
	if err != nil {
		return err
	}
	for idx := range items {
		summary := buildOrganizingSummary(conditions[items[idx].ID], false)
		if summary == nil {
			continue
		}
		items[idx].OrganizingSummary = summary
		items[idx].Organizing = summary.State == "organizing" || summary.State == "review_required" || summary.State == "failed"
	}
	return nil
}

func (s *Service) organizingConditionsByCatalogItemID(ctx context.Context, itemIDs []uint) (map[uint][]database.IngestCondition, error) {
	result := make(map[uint][]database.IngestCondition, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}
	var conditions []database.IngestCondition
	if err := s.db.WithContext(ctx).Where("catalog_item_id IN ?", itemIDs).Order("catalog_item_id asc, condition_type asc").Find(&conditions).Error; err != nil {
		return nil, err
	}
	for _, condition := range conditions {
		if condition.CatalogItemID == nil {
			continue
		}
		result[*condition.CatalogItemID] = append(result[*condition.CatalogItemID], condition)
	}
	return result, nil
}

func (s *Service) organizingConditionsByInventoryFile(ctx context.Context, files []database.InventoryFile) (map[uint][]database.IngestCondition, error) {
	result := make(map[uint][]database.IngestCondition, len(files))
	if len(files) == 0 {
		return result, nil
	}
	fileIDs := make([]uint, 0, len(files))
	for _, file := range files {
		fileIDs = append(fileIDs, file.ID)
	}
	var conditions []database.IngestCondition
	if err := s.db.WithContext(ctx).Where("inventory_file_id IN ?", fileIDs).Order("inventory_file_id asc, condition_type asc").Find(&conditions).Error; err != nil {
		return nil, err
	}
	for _, condition := range conditions {
		if condition.InventoryFileID == nil {
			continue
		}
		result[*condition.InventoryFileID] = append(result[*condition.InventoryFileID], condition)
	}
	return result, nil
}

func buildOrganizingSummary(conditions []database.IngestCondition, inventoryOnly bool) *CatalogOrganizingSummary {
	if len(conditions) == 0 {
		if !inventoryOnly {
			return nil
		}
		return &CatalogOrganizingSummary{State: "organizing", Stage: ingest.ConditionMaterialized, Severity: ingest.SeverityInfo, Message: "Identifying media"}
	}
	cards := make([]CatalogOrganizingCondition, 0, len(conditions))
	for _, condition := range conditions {
		cards = append(cards, CatalogOrganizingCondition{Type: condition.ConditionType, Status: condition.Status, Reason: condition.Reason, Message: condition.Message, Severity: condition.Severity})
	}
	if selected := firstConditionWithStatuses(conditions, ingest.ConditionStatusReviewRequired); selected != nil {
		return organizingSummaryFromCondition("review_required", "Review needed", selected, cards)
	}
	if selected := firstMetadataNoCandidateCondition(conditions); selected != nil {
		return organizingSummaryFromCondition("review_required", "Review needed", selected, cards)
	}
	if selected := firstConditionWithStatuses(conditions, ingest.ConditionStatusFailed); selected != nil {
		return organizingSummaryFromCondition("failed", "Organizing failed", selected, cards)
	}
	if selected := firstBlockingConditionWithStatuses(conditions, ingest.ConditionStatusRunning, ingest.ConditionStatusPending, ingest.ConditionStatusUnknown); selected != nil {
		return organizingSummaryFromCondition("organizing", organizingMessageForStage(selected.ConditionType), selected, cards)
	}
	if allReadyOrSkippedOrEnhancing(conditions) {
		return &CatalogOrganizingSummary{State: "ready", Stage: "ready", Severity: ingest.SeverityInfo, Message: "Ready", Conditions: cards}
	}
	return &CatalogOrganizingSummary{State: "partial_ready", Stage: "partial_ready", Severity: ingest.SeverityWarning, Message: "Partially ready", Conditions: cards}
}

func firstMetadataNoCandidateCondition(conditions []database.IngestCondition) *database.IngestCondition {
	for idx := range conditions {
		if conditions[idx].ConditionType == ingest.ConditionMetadataMatched && conditions[idx].Status == ingest.ConditionStatusFalse && conditions[idx].Reason == "no_candidate" {
			return &conditions[idx]
		}
	}
	return nil
}

func firstConditionWithStatuses(conditions []database.IngestCondition, statuses ...string) *database.IngestCondition {
	statusSet := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		statusSet[status] = struct{}{}
	}
	for _, conditionType := range []string{ingest.ConditionReviewRequired, ingest.ConditionMaterialized, ingest.ConditionProbed, ingest.ConditionMetadataMatched, ingest.ConditionProjectionCurrent, ingest.ConditionVisible} {
		for idx := range conditions {
			if conditions[idx].ConditionType != conditionType {
				continue
			}
			if _, ok := statusSet[conditions[idx].Status]; ok {
				return &conditions[idx]
			}
		}
	}
	return nil
}

func firstBlockingConditionWithStatuses(conditions []database.IngestCondition, statuses ...string) *database.IngestCondition {
	materialized := conditionHasStatus(conditions, ingest.ConditionMaterialized, ingest.ConditionStatusTrue)
	statusSet := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		statusSet[status] = struct{}{}
	}
	for _, conditionType := range []string{ingest.ConditionReviewRequired, ingest.ConditionMaterialized, ingest.ConditionProbed, ingest.ConditionMetadataMatched, ingest.ConditionProjectionCurrent, ingest.ConditionVisible} {
		for idx := range conditions {
			if conditions[idx].ConditionType != conditionType {
				continue
			}
			if _, ok := statusSet[conditions[idx].Status]; !ok {
				continue
			}
			if materialized && isPostMaterializeEnhancementCondition(conditions[idx].ConditionType) {
				continue
			}
			return &conditions[idx]
		}
	}
	return nil
}

func conditionHasStatus(conditions []database.IngestCondition, conditionType string, status string) bool {
	for _, condition := range conditions {
		if condition.ConditionType == conditionType && condition.Status == status {
			return true
		}
	}
	return false
}

func isPostMaterializeEnhancementCondition(conditionType string) bool {
	switch conditionType {
	case ingest.ConditionProbed, ingest.ConditionMetadataMatched, ingest.ConditionProjectionCurrent:
		return true
	default:
		return false
	}
}

func organizingSummaryFromCondition(state string, fallbackMessage string, condition *database.IngestCondition, conditions []CatalogOrganizingCondition) *CatalogOrganizingSummary {
	message := strings.TrimSpace(condition.Message)
	if message == "" {
		message = fallbackMessage
	}
	return &CatalogOrganizingSummary{State: state, Stage: condition.ConditionType, Severity: condition.Severity, Message: message, Conditions: conditions}
}

func organizingMessageForStage(stage string) string {
	switch stage {
	case ingest.ConditionMaterialized:
		return "Identifying media"
	case ingest.ConditionProbed:
		return "Analyzing video streams"
	case ingest.ConditionMetadataMatched:
		return "Matching metadata"
	case ingest.ConditionProjectionCurrent:
		return "Updating library view"
	default:
		return "Organizing media"
	}
}

func allReadyOrSkippedOrEnhancing(conditions []database.IngestCondition) bool {
	materialized := conditionHasStatus(conditions, ingest.ConditionMaterialized, ingest.ConditionStatusTrue)
	for _, condition := range conditions {
		switch condition.Status {
		case ingest.ConditionStatusTrue, ingest.ConditionStatusSkipped, ingest.ConditionStatusFalse:
			continue
		case ingest.ConditionStatusRunning, ingest.ConditionStatusPending, ingest.ConditionStatusUnknown:
			if materialized && isPostMaterializeEnhancementCondition(condition.ConditionType) {
				continue
			}
			return false
		default:
			return false
		}
	}
	return true
}

func discoveredTitleFromPath(storagePath string) string {
	base := strings.TrimSpace(path.Base(storagePath))
	if base == "." || base == "/" || base == "" {
		return strings.TrimSpace(storagePath)
	}
	name := strings.TrimSuffix(base, path.Ext(base))
	name = strings.TrimSpace(strings.NewReplacer(".", " ", "_", " ").Replace(name))
	if name == "" {
		return base
	}
	return name
}

func catalogListItemTitleKey(item CatalogListItem) string {
	if item.SortTitle != "" {
		return strings.ToLower(item.SortTitle)
	}
	return strings.ToLower(item.Title)
}

func applyBrowseListEntryOrder(entries []browseListEntry, input BrowseItemsInput) {
	desc := input.SortDirection == "desc"
	sort.SliceStable(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]
		switch input.Sort {
		case "title":
			if left.TitleKey != right.TitleKey {
				return compareBrowseString(left.TitleKey, right.TitleKey, desc)
			}
		case "year":
			if left.Year == nil || right.Year == nil {
				if left.Year == nil && right.Year != nil {
					return false
				}
				if left.Year != nil && right.Year == nil {
					return true
				}
			} else if *left.Year != *right.Year {
				if desc {
					return *left.Year > *right.Year
				}
				return *left.Year < *right.Year
			}
		default:
			if left.CreatedAt != right.CreatedAt {
				return compareBrowseString(left.CreatedAt, right.CreatedAt, desc)
			}
		}
		return left.StableID < right.StableID
	})
}

func compareBrowseString(left string, right string, desc bool) bool {
	if desc {
		return left > right
	}
	return left < right
}

func normalizeBrowseItemsInput(input BrowseItemsInput) BrowseItemsInput {
	input.Query = strings.TrimSpace(input.Query)
	input.Genre = strings.TrimSpace(input.Genre)
	input.Region = strings.TrimSpace(input.Region)
	input.TypeFilter = strings.ToLower(strings.TrimSpace(input.TypeFilter))
	input.Sort = strings.ToLower(strings.TrimSpace(input.Sort))
	input.SortDirection = strings.ToLower(strings.TrimSpace(input.SortDirection))
	input.WatchedState = strings.ToLower(strings.TrimSpace(input.WatchedState))
	input.OrganizingState = strings.ToLower(strings.TrimSpace(input.OrganizingState))
	switch input.TypeFilter {
	case ItemTypeMovie, ItemTypeSeries, "show", ItemTypeEpisode:
	default:
		input.TypeFilter = "all"
	}
	switch input.Sort {
	case "title", "year", "watch_status":
	default:
		input.Sort = "recent"
	}
	switch input.WatchedState {
	case "unwatched", "in_progress", "watched":
	default:
		input.WatchedState = "all"
	}
	switch input.OrganizingState {
	case "organized", "unorganized":
	default:
		input.OrganizingState = "all"
	}
	if input.Limit <= 0 || input.Limit > 200 {
		input.Limit = 50
	}
	if input.Offset < 0 {
		input.Offset = 0
	}
	switch input.SortDirection {
	case "asc", "desc":
	default:
		input.SortDirection = "desc"
		if input.Sort == "title" {
			input.SortDirection = "asc"
		}
	}
	return input
}

func applyBrowseItemsOrder(db *gorm.DB, input BrowseItemsInput) *gorm.DB {
	direction := "desc"
	if input.SortDirection == "asc" {
		direction = "asc"
	}
	switch input.Sort {
	case "title":
		return db.Order("COALESCE(NULLIF(catalog_items.sort_title, ''), NULLIF(catalog_items.sort_key, ''), catalog_items.title) " + direction).Order("catalog_items.id " + direction)
	case "year":
		return db.Order("catalog_items.year IS NULL asc").Order("catalog_items.year " + direction).Order("catalog_items.id " + direction)
	case "watch_status":
		if input.UserID != 0 {
			db = db.Joins("LEFT JOIN user_item_data browse_sort_user_item_data ON browse_sort_user_item_data.item_id = catalog_items.id AND browse_sort_user_item_data.asset_id IS NULL AND browse_sort_user_item_data.user_id = ?", input.UserID)
			return db.Order(`CASE
				WHEN browse_sort_user_item_data.completed_at IS NULL AND browse_sort_user_item_data.position_seconds > 0 THEN 1
				WHEN browse_sort_user_item_data.completed_at IS NOT NULL THEN 2
				ELSE 0
			END ` + direction).Order("COALESCE(NULLIF(catalog_items.sort_title, ''), NULLIF(catalog_items.sort_key, ''), catalog_items.title) asc").Order("catalog_items.id asc")
		}
	}
	return db.Order("catalog_items.created_at " + direction).Order("catalog_items.id " + direction)
}

func (s *Service) listItems(ctx context.Context, libraryID *uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	allowedTypes := []string{ItemTypeMovie, ItemTypeSeries}
	switch strings.ToLower(strings.TrimSpace(typeFilter)) {
	case ItemTypeMovie:
		allowedTypes = []string{ItemTypeMovie}
	case ItemTypeSeries, "show":
		allowedTypes = []string{ItemTypeSeries}
	}

	db := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("parent_id IS NULL").
		Where("availability_status = ?", AvailabilityAvailable).
		Where("type IN ?", allowedTypes)
	if libraryID != nil {
		db = db.Where("library_id = ?", *libraryID)
	}
	if trimmedQuery := strings.TrimSpace(query); trimmedQuery != "" {
		like := "%" + strings.ToLower(trimmedQuery) + "%"
		db = db.Where("LOWER(title) LIKE ? OR LOWER(original_title) LIKE ? OR LOWER(sort_title) LIKE ?", like, like, like)
	}

	var items []database.CatalogItem
	if err := db.Order("sort_key asc").Order("title asc").Order("id asc").Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return s.buildCatalogListItems(ctx, items)
}

func (s *Service) GetItemDetail(ctx context.Context, itemID uint) (CatalogItemDetail, error) {
	return s.GetItemDetailForUser(ctx, itemID, nil)
}

func (s *Service) GetItemDetailForUser(ctx context.Context, itemID uint, userID *uint) (CatalogItemDetail, error) {
	item, err := s.loadCatalogItem(ctx, itemID)
	if err != nil {
		return CatalogItemDetail{}, err
	}

	rollups, images, externalIDs, sources, fieldStates, err := s.loadCatalogQueryData(ctx, []uint{item.ID})
	if err != nil {
		return CatalogItemDetail{}, err
	}
	assetsByItem, err := s.loadCatalogAssetsByItem(ctx, []uint{item.ID})
	if err != nil {
		return CatalogItemDetail{}, err
	}
	tagsByItem, err := s.loadCatalogDisplayTagsByItem(ctx, []uint{item.ID})
	if err != nil {
		return CatalogItemDetail{}, err
	}
	relatedItems, err := s.loadRelatedCatalogItems(ctx, item, tagsByItem[item.ID], 12)
	if err != nil {
		return CatalogItemDetail{}, err
	}
	cast, directors, err := s.loadCatalogItemPeopleDetails(ctx, item, externalIDs[item.ID])
	if err != nil {
		return CatalogItemDetail{}, err
	}

	seasons := []CatalogSeasonDetail{}
	episodes := []CatalogEpisodeDetail{}
	if item.Type == ItemTypeSeries {
		seasons, err = s.ListSeriesSeasons(ctx, item.ID)
		if err != nil {
			return CatalogItemDetail{}, err
		}
	}
	var seriesPlaybackTarget *CatalogSeriesPlaybackTarget
	if item.Type == ItemTypeSeries {
		seriesPlaybackTarget, err = s.getSeriesPlaybackTarget(ctx, item.ID, userID)
		if err != nil {
			return CatalogItemDetail{}, err
		}
	}
	if item.Type == ItemTypeSeason {
		episodes, err = s.buildCatalogEpisodeDetailsForParent(ctx, item.ID)
		if err != nil {
			return CatalogItemDetail{}, err
		}
	}
	episodeContext, seasonID, err := s.loadEpisodeParentContext(ctx, item)
	if err != nil {
		return CatalogItemDetail{}, err
	}
	sameSeasonEpisodes, err := s.buildSameSeasonEpisodeShelf(ctx, seasonID, item.ID, userID)
	if err != nil {
		return CatalogItemDetail{}, err
	}

	return BuildCatalogItemDetail(CatalogItemDetailInput{
		Item:                 item,
		Rollup:               rollups[item.ID],
		Images:               images[item.ID],
		ExternalIDs:          externalIDs[item.ID],
		Sources:              sources[item.ID],
		FieldStates:          fieldStates[item.ID],
		Cast:                 cast,
		Directors:            directors,
		Tags:                 tagsByItem[item.ID],
		Seasons:              seasons,
		Episodes:             episodes,
		EpisodeContext:       episodeContext,
		SeriesPlaybackTarget: seriesPlaybackTarget,
		SameSeasonEpisodes:   sameSeasonEpisodes,
		Assets:               assetsByItem[item.ID],
		Related:              relatedItems,
	}), nil
}

func (s *Service) getSeriesPlaybackTarget(ctx context.Context, seriesID uint, userID *uint) (*CatalogSeriesPlaybackTarget, error) {
	target, err := seriesplayback.Select(ctx, s.db, seriesID, userID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, nil
	}
	return &CatalogSeriesPlaybackTarget{
		EpisodeItemID:   target.EpisodeID,
		AssetID:         target.AssetID,
		Title:           target.Title,
		Label:           target.Label,
		SelectionReason: target.Reason,
	}, nil
}

func (s *Service) loadEpisodeParentContext(ctx context.Context, item database.CatalogItem) (*CatalogEpisodeParentContext, *uint, error) {
	if item.Type != ItemTypeEpisode {
		return nil, nil, nil
	}

	var season *database.CatalogItem
	if item.ParentID != nil && *item.ParentID > 0 {
		loaded, err := s.loadCatalogItem(ctx, *item.ParentID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
		if err == nil && loaded.Type == ItemTypeSeason {
			season = &loaded
		}
	}

	var series *database.CatalogItem
	if season != nil && season.ParentID != nil && *season.ParentID > 0 {
		loaded, err := s.loadCatalogItem(ctx, *season.ParentID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
		if err == nil && loaded.Type == ItemTypeSeries {
			series = &loaded
		}
	}
	if series == nil && item.RootID != nil && *item.RootID > 0 {
		loaded, err := s.loadCatalogItem(ctx, *item.RootID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
		if err == nil && loaded.Type == ItemTypeSeries {
			series = &loaded
		}
	}

	imageIDs := make([]uint, 0, 2)
	if series != nil {
		imageIDs = append(imageIDs, series.ID)
	}
	if season != nil {
		imageIDs = append(imageIDs, season.ID)
	}
	_, images, _, _, _, err := s.loadCatalogQueryData(ctx, imageIDs)
	if err != nil {
		return nil, nil, err
	}

	var seasonID *uint
	if season != nil {
		id := season.ID
		seasonID = &id
	}
	return BuildCatalogEpisodeParentContext(series, season, selectedImagesForItem(images, series), selectedImagesForItem(images, season), item), seasonID, nil
}

func selectedImagesForItem(images map[uint][]database.ItemImage, item *database.CatalogItem) []database.ItemImage {
	if item == nil {
		return nil
	}
	return images[item.ID]
}

func (s *Service) buildSameSeasonEpisodeShelf(ctx context.Context, seasonID *uint, currentItemID uint, userID *uint) ([]CatalogEpisodeShelfItem, error) {
	if seasonID == nil || *seasonID == 0 {
		return []CatalogEpisodeShelfItem{}, nil
	}
	season, err := s.loadCatalogItem(ctx, *seasonID)
	if err != nil {
		return nil, err
	}
	if season.Type != ItemTypeSeason || season.ParentID == nil || *season.ParentID == 0 {
		return []CatalogEpisodeShelfItem{}, nil
	}
	playableEpisodeIDs, err := seriesplayback.LoadPlayableEpisodeIDs(ctx, s.db, *season.ParentID)
	if err != nil {
		return nil, err
	}
	episodes, err := s.buildCatalogEpisodeDetailsForParent(ctx, *seasonID)
	if err != nil {
		return nil, err
	}
	if len(episodes) == 0 {
		return []CatalogEpisodeShelfItem{}, nil
	}

	episodeIDs := make([]uint, 0, len(episodes))
	durationsByItem := make(map[uint]*int, len(episodes))
	for _, episode := range episodes {
		if _, ok := playableEpisodeIDs[episode.ID]; !ok {
			continue
		}
		episodeIDs = append(episodeIDs, episode.ID)
		durationsByItem[episode.ID] = episode.RuntimeSeconds
	}
	progressByItem := map[uint]*CatalogUserProgressState{}
	if userID != nil && *userID > 0 {
		progressByItem, err = s.loadCatalogUserProgressStatesByItem(ctx, *userID, episodeIDs, durationsByItem)
		if err != nil {
			return nil, err
		}
	}

	shelf := make([]CatalogEpisodeShelfItem, 0, len(episodes))
	for _, episode := range episodes {
		if _, ok := playableEpisodeIDs[episode.ID]; !ok {
			continue
		}
		shelf = append(shelf, BuildCatalogEpisodeShelfItem(CatalogEpisodeShelfItemInput{
			Episode:       episode,
			CurrentItemID: currentItemID,
			Progress:      progressByItem[episode.ID],
		}))
	}
	return shelf, nil
}

func (s *Service) loadCatalogUserProgressStatesByItem(ctx context.Context, userID uint, itemIDs []uint, durationsByItem map[uint]*int) (map[uint]*CatalogUserProgressState, error) {
	result := make(map[uint]*CatalogUserProgressState, len(itemIDs))
	if userID == 0 || len(itemIDs) == 0 {
		return result, nil
	}

	var rows []database.UserItemData
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND item_id IN ? AND asset_id IS NULL", userID, itemIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		state := catalogUserProgressState(row, durationsByItem[row.ItemID])
		result[row.ItemID] = &state
	}
	return result, nil
}

func (s *Service) loadCatalogItemPeopleDetails(ctx context.Context, item database.CatalogItem, externalIDs []database.CatalogExternalID) ([]CatalogPersonDetail, []CatalogPersonDetail, error) {
	_ = externalIDs
	return s.loadCatalogPeopleDetails(ctx, item.ID)
}

func (s *Service) loadCatalogPeopleDetails(ctx context.Context, itemID uint) ([]CatalogPersonDetail, []CatalogPersonDetail, error) {
	rows := make([]struct {
		PersonID     uint
		RelationRole string
		Character    string
		Name         string
		AvatarURL    string
	}, 0)
	if err := s.db.WithContext(ctx).
		Table("item_people").
		Select("people.id AS person_id, item_people.role AS relation_role, item_people.character, people.name, people.avatar_url").
		Joins("JOIN people ON people.id = item_people.person_id").
		Where("item_people.item_id = ?", itemID).
		Order("item_people.role asc, item_people.sort_order asc, people.name asc").
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	cast := make([]CatalogPersonDetail, 0, len(rows))
	directors := make([]CatalogPersonDetail, 0, len(rows))
	for _, row := range rows {
		person := CatalogPersonDetail{
			ID:        row.PersonID,
			Name:      strings.TrimSpace(row.Name),
			Role:      strings.TrimSpace(row.Character),
			AvatarURL: strings.TrimSpace(row.AvatarURL),
		}
		if person.Name == "" {
			continue
		}
		switch strings.TrimSpace(row.RelationRole) {
		case "director":
			directors = append(directors, person)
		default:
			cast = append(cast, person)
		}
	}
	return cast, directors, nil
}

func (s *Service) ListSeriesSeasons(ctx context.Context, seriesID uint) ([]CatalogSeasonDetail, error) {
	series, err := s.loadCatalogItem(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Type != ItemTypeSeries {
		return []CatalogSeasonDetail{}, nil
	}

	var seasons []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("parent_id = ? AND type = ? AND deleted_at IS NULL", series.ID, ItemTypeSeason).
		Order("index_number asc").Order("id asc").
		Find(&seasons).Error; err != nil {
		return nil, err
	}
	if len(seasons) == 0 {
		return []CatalogSeasonDetail{}, nil
	}

	seasonIDs := make([]uint, 0, len(seasons))
	for _, season := range seasons {
		seasonIDs = append(seasonIDs, season.ID)
	}

	rollups, images, externalIDs, sources, fieldStates, err := s.loadCatalogQueryData(ctx, seasonIDs)
	if err != nil {
		return nil, err
	}
	episodesBySeason, err := s.buildCatalogEpisodeDetailsByParent(ctx, seasonIDs)
	if err != nil {
		return nil, err
	}
	playableEpisodeIDs, err := seriesplayback.LoadPlayableEpisodeIDs(ctx, s.db, series.ID)
	if err != nil {
		return nil, err
	}

	result := make([]CatalogSeasonDetail, 0, len(seasons))
	for _, season := range seasons {
		episodes := make([]CatalogEpisodeDetail, 0, len(episodesBySeason[season.ID]))
		for _, episode := range episodesBySeason[season.ID] {
			if _, ok := playableEpisodeIDs[episode.ID]; !ok {
				continue
			}
			episodes = append(episodes, episode)
		}
		if len(episodes) == 0 {
			continue
		}
		result = append(result, BuildCatalogSeasonDetail(CatalogSeasonDetailInput{
			Item:        season,
			Rollup:      rollups[season.ID],
			Images:      images[season.ID],
			ExternalIDs: externalIDs[season.ID],
			Sources:     sources[season.ID],
			FieldStates: fieldStates[season.ID],
			Episodes:    episodes,
		}))
	}
	return result, nil
}

func (s *Service) GetGovernanceWorkspace(ctx context.Context, itemID uint) (CatalogGovernanceWorkspace, error) {
	item, err := s.loadCatalogItem(ctx, itemID)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	_, images, externalIDs, sources, fieldStates, err := s.loadCatalogQueryData(ctx, []uint{item.ID})
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	assetsByItem, err := s.loadCatalogAssetsByItem(ctx, []uint{item.ID})
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	classification, err := s.loadCatalogClassificationDecisions(ctx, item)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	classificationRules, err := s.loadCatalogClassificationRules(ctx, item.LibraryID)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}

	children, err := s.ListChildren(ctx, item.ID)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	recommendedChildren, err := s.buildCatalogListItems(ctx, children)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}

	return BuildCatalogGovernanceWorkspace(CatalogGovernanceWorkspaceInput{
		Item:                item,
		Images:              images[item.ID],
		ExternalIDs:         externalIDs[item.ID],
		Sources:             sources[item.ID],
		FieldStates:         fieldStates[item.ID],
		Assets:              assetsByItem[item.ID],
		Classification:      classification,
		ClassificationRules: classificationRules,
		RecommendedChildren: recommendedChildren,
	}), nil
}

func (s *Service) loadCatalogClassificationDecisions(ctx context.Context, item database.CatalogItem) ([]database.ClassificationDecision, error) {
	var decisions []database.ClassificationDecision
	query := s.db.WithContext(ctx).
		Where("library_id = ?", item.LibraryID).
		Where("item_id = ? OR target_key = ? OR source_path = ?", item.ID, item.Path, item.Path).
		Order("created_at desc").Order("id desc")
	if err := query.Find(&decisions).Error; err != nil {
		return nil, err
	}
	return decisions, nil
}

func (s *Service) loadCatalogClassificationRules(ctx context.Context, libraryID uint) ([]database.ClassificationRule, error) {
	var rules []database.ClassificationRule
	if err := s.db.WithContext(ctx).
		Where("library_id = ? AND enabled = ?", libraryID, true).
		Order("created_at desc").Order("id desc").
		Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *Service) ListRecentlyAdded(ctx context.Context, limit int) ([]CatalogListItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 12
	}
	var items []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Joins("JOIN libraries ON libraries.id = catalog_items.library_id AND libraries.status IN ?", []string{"active", "syncing"}).
		Where("catalog_items.deleted_at IS NULL").
		Where("catalog_items.parent_id IS NULL").
		Where("catalog_items.availability_status = ?", AvailabilityAvailable).
		Where("catalog_items.type IN ?", []string{ItemTypeMovie, ItemTypeSeries}).
		Order("catalog_items.created_at desc").Order("catalog_items.id desc").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return s.buildCatalogListItems(ctx, items)
}

func (s *Service) ListLatestByLibrary(ctx context.Context, limit int) ([]CatalogLatestByLibrarySection, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	var libraries []database.Library
	if err := s.db.WithContext(ctx).
		Where("status IN ?", []string{"active", "syncing"}).
		Order("name asc").
		Find(&libraries).Error; err != nil {
		return nil, err
	}
	sections := make([]CatalogLatestByLibrarySection, 0, len(libraries))
	for _, library := range libraries {
		var items []database.CatalogItem
		if err := s.db.WithContext(ctx).
			Where("library_id = ? AND deleted_at IS NULL", library.ID).
			Where("parent_id IS NULL").
			Where("availability_status = ?", AvailabilityAvailable).
			Where("type IN ?", []string{ItemTypeMovie, ItemTypeSeries}).
			Order("created_at desc").Order("id desc").
			Limit(limit).
			Find(&items).Error; err != nil {
			return nil, err
		}
		if len(items) == 0 {
			continue
		}
		mapped, err := s.buildCatalogListItems(ctx, items)
		if err != nil {
			return nil, err
		}
		sections = append(sections, CatalogLatestByLibrarySection{
			LibraryID:   library.ID,
			LibraryName: library.Name,
			Items:       mapped,
		})
	}
	return sections, nil
}

func (s *Service) loadCatalogItem(ctx context.Context, itemID uint) (database.CatalogItem, error) {
	var item database.CatalogItem
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", itemID).First(&item).Error
	return item, err
}

func (s *Service) IsGovernanceTargetAllowed(ctx context.Context, workspaceItemID uint, targetItemID uint) (bool, error) {
	if workspaceItemID == 0 || targetItemID == 0 {
		return false, nil
	}
	item, err := s.loadCatalogItem(ctx, targetItemID)
	if err != nil {
		return false, err
	}
	for {
		if item.ID == workspaceItemID {
			return true, nil
		}
		if item.ParentID == nil || *item.ParentID == 0 {
			return false, nil
		}
		item, err = s.loadCatalogItem(ctx, *item.ParentID)
		if err != nil {
			return false, err
		}
	}
}

func (s *Service) buildCatalogListItems(ctx context.Context, items []database.CatalogItem) ([]CatalogListItem, error) {
	if len(items) == 0 {
		return []CatalogListItem{}, nil
	}
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	rollups, images, externalIDs, _, _, err := s.loadCatalogQueryData(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	result := make([]CatalogListItem, 0, len(items))
	for _, item := range items {
		result = append(result, BuildCatalogListItem(CatalogListItemInput{
			Item:        item,
			Rollup:      rollups[item.ID],
			Images:      images[item.ID],
			ExternalIDs: externalIDs[item.ID],
		}))
	}
	return result, nil
}

func (s *Service) loadCatalogDisplayTagsByItem(ctx context.Context, itemIDs []uint) (map[uint][]CatalogTagDetail, error) {
	result := make(map[uint][]CatalogTagDetail, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		ItemID uint
		Kind   string
		Name   string
	}
	if err := s.db.WithContext(ctx).
		Table("item_tags").
		Select("item_tags.item_id, tags.kind, tags.name").
		Joins("JOIN tags ON tags.id = item_tags.tag_id").
		Where("item_tags.item_id IN ?", itemIDs).
		Order("item_tags.item_id asc, CASE WHEN LOWER(tags.kind) = 'genre' THEN 0 ELSE 1 END asc, tags.kind asc, tags.name asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	seen := make(map[uint]map[string]struct{}, len(itemIDs))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		kind := strings.TrimSpace(row.Kind)
		key := strings.ToLower(kind) + "\x00" + strings.ToLower(name)
		if seen[row.ItemID] == nil {
			seen[row.ItemID] = make(map[string]struct{})
		}
		if _, ok := seen[row.ItemID][key]; ok {
			continue
		}
		seen[row.ItemID][key] = struct{}{}
		result[row.ItemID] = append(result[row.ItemID], CatalogTagDetail{Kind: kind, Name: name})
	}
	return result, nil
}

func (s *Service) loadRelatedCatalogItems(ctx context.Context, item database.CatalogItem, tags []CatalogTagDetail, limit int) ([]CatalogListItem, error) {
	if limit <= 0 || limit > 24 {
		limit = 12
	}
	if item.ID == 0 || item.LibraryID == 0 {
		return []CatalogListItem{}, nil
	}

	items, err := s.findRelatedItemsByTags(ctx, item, tags, limit)
	if err != nil {
		return nil, err
	}
	if len(items) < limit {
		fallback, err := s.findRelatedItemsByLibrary(ctx, item, limit-len(items), relatedItemIDSet(items, item.ID))
		if err != nil {
			return nil, err
		}
		items = append(items, fallback...)
	}
	return s.buildCatalogListItems(ctx, items)
}

func (s *Service) findRelatedItemsByTags(ctx context.Context, item database.CatalogItem, tags []CatalogTagDetail, limit int) ([]database.CatalogItem, error) {
	tagNames := relatedTagNames(tags)
	if len(tagNames) == 0 {
		return []database.CatalogItem{}, nil
	}
	var items []database.CatalogItem
	err := s.db.WithContext(ctx).
		Model(&database.CatalogItem{}).
		Select("catalog_items.*").
		Joins("JOIN item_tags ON item_tags.item_id = catalog_items.id").
		Joins("JOIN tags ON tags.id = item_tags.tag_id").
		Where("catalog_items.deleted_at IS NULL").
		Where("catalog_items.library_id = ?", item.LibraryID).
		Where("catalog_items.id <> ?", item.ID).
		Where("catalog_items.parent_id IS NULL").
		Where("catalog_items.type IN ?", []string{ItemTypeMovie, ItemTypeSeries}).
		Where("LOWER(tags.name) IN ?", tagNames).
		Group("catalog_items.id").
		Order("COUNT(tags.id) desc").
		Order("catalog_items.year desc").
		Order("catalog_items.sort_key asc").
		Order("catalog_items.title asc").
		Order("catalog_items.id asc").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func (s *Service) findRelatedItemsByLibrary(ctx context.Context, item database.CatalogItem, limit int, excluded map[uint]struct{}) ([]database.CatalogItem, error) {
	if limit <= 0 {
		return []database.CatalogItem{}, nil
	}
	excludedIDs := make([]uint, 0, len(excluded))
	for id := range excluded {
		excludedIDs = append(excludedIDs, id)
	}
	sort.Slice(excludedIDs, func(i, j int) bool { return excludedIDs[i] < excludedIDs[j] })
	var items []database.CatalogItem
	query := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("library_id = ?", item.LibraryID).
		Where("parent_id IS NULL").
		Where("type IN ?", []string{ItemTypeMovie, ItemTypeSeries}).
		Where("id NOT IN ?", excludedIDs).
		Order("year desc").
		Order("sort_key asc").
		Order("title asc").
		Order("id asc").
		Limit(limit)
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func relatedTagNames(tags []CatalogTagDetail) []string {
	if len(tags) == 0 {
		return nil
	}
	preferred := make([]string, 0, len(tags))
	fallback := make([]string, 0, len(tags))
	seenPreferred := make(map[string]struct{}, len(tags))
	seenFallback := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		name := strings.ToLower(strings.TrimSpace(tag.Name))
		if name == "" {
			continue
		}
		if _, ok := seenFallback[name]; !ok {
			fallback = append(fallback, name)
			seenFallback[name] = struct{}{}
		}
		if strings.EqualFold(strings.TrimSpace(tag.Kind), "genre") {
			if _, ok := seenPreferred[name]; ok {
				continue
			}
			preferred = append(preferred, name)
			seenPreferred[name] = struct{}{}
		}
	}
	if len(preferred) > 0 {
		return preferred
	}
	return fallback
}

func relatedItemIDSet(items []database.CatalogItem, currentID uint) map[uint]struct{} {
	ids := make(map[uint]struct{}, len(items)+1)
	ids[currentID] = struct{}{}
	for _, item := range items {
		ids[item.ID] = struct{}{}
	}
	return ids
}

func (s *Service) buildCatalogEpisodeDetailsForParent(ctx context.Context, parentID uint) ([]CatalogEpisodeDetail, error) {
	byParent, err := s.buildCatalogEpisodeDetailsByParent(ctx, []uint{parentID})
	if err != nil {
		return nil, err
	}
	return byParent[parentID], nil
}

func (s *Service) buildCatalogEpisodeDetailsByParent(ctx context.Context, parentIDs []uint) (map[uint][]CatalogEpisodeDetail, error) {
	if len(parentIDs) == 0 {
		return map[uint][]CatalogEpisodeDetail{}, nil
	}
	var episodes []database.CatalogItem
	if err := s.db.WithContext(ctx).
		Where("parent_id IN ? AND type = ? AND deleted_at IS NULL", parentIDs, ItemTypeEpisode).
		Order("parent_id asc").Order("index_number asc").Order("id asc").
		Find(&episodes).Error; err != nil {
		return nil, err
	}
	if len(episodes) == 0 {
		return map[uint][]CatalogEpisodeDetail{}, nil
	}
	itemIDs := make([]uint, 0, len(episodes))
	for _, episode := range episodes {
		itemIDs = append(itemIDs, episode.ID)
	}
	_, images, externalIDs, sources, fieldStates, err := s.loadCatalogQueryData(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	assetsByItem, err := s.loadCatalogAssetsByItem(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uint][]CatalogEpisodeDetail, len(parentIDs))
	for _, episode := range episodes {
		if episode.ParentID == nil {
			continue
		}
		result[*episode.ParentID] = append(result[*episode.ParentID], BuildCatalogEpisodeDetail(CatalogEpisodeDetailInput{
			Item:        episode,
			Images:      images[episode.ID],
			ExternalIDs: externalIDs[episode.ID],
			Sources:     sources[episode.ID],
			FieldStates: fieldStates[episode.ID],
			Assets:      assetsByItem[episode.ID],
		}))
	}
	return result, nil
}

func (s *Service) loadCatalogQueryData(ctx context.Context, itemIDs []uint) (map[uint]*database.ItemRollup, map[uint][]database.ItemImage, map[uint][]database.CatalogExternalID, map[uint][]database.MetadataSource, map[uint][]database.MetadataFieldState, error) {
	rollups := make(map[uint]*database.ItemRollup, len(itemIDs))
	images := make(map[uint][]database.ItemImage, len(itemIDs))
	externalIDs := make(map[uint][]database.CatalogExternalID, len(itemIDs))
	sources := make(map[uint][]database.MetadataSource, len(itemIDs))
	fieldStates := make(map[uint][]database.MetadataFieldState, len(itemIDs))
	if len(itemIDs) == 0 {
		return rollups, images, externalIDs, sources, fieldStates, nil
	}

	var rollupRows []database.ItemRollup
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Find(&rollupRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range rollupRows {
		rowCopy := row
		rollups[row.ItemID] = &rowCopy
	}

	var imageRows []database.ItemImage
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, sort_order asc, id asc").Find(&imageRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range imageRows {
		images[row.ItemID] = append(images[row.ItemID], row)
	}

	var externalIDRows []database.CatalogExternalID
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, is_primary desc, provider asc, provider_type asc, id asc").Find(&externalIDRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range externalIDRows {
		externalIDs[row.ItemID] = append(externalIDs[row.ItemID], row)
	}

	var sourceRows []database.MetadataSource
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, fetched_at desc, id desc").Find(&sourceRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range sourceRows {
		sources[row.ItemID] = append(sources[row.ItemID], row)
	}

	var fieldStateRows []database.MetadataFieldState
	if err := s.db.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, field_key asc, id asc").Find(&fieldStateRows).Error; err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, row := range fieldStateRows {
		fieldStates[row.ItemID] = append(fieldStates[row.ItemID], row)
	}

	return rollups, images, externalIDs, sources, fieldStates, nil
}

func (s *Service) loadCatalogAssetsByItem(ctx context.Context, itemIDs []uint) (map[uint][]CatalogAssetDetail, error) {
	result := make(map[uint][]CatalogAssetDetail, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}

	var links []database.AssetItem
	if err := s.db.WithContext(ctx).
		Where("item_id IN ?", itemIDs).
		Order("item_id asc, role asc, segment_index asc, id asc").
		Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return result, nil
	}

	assetIDs := make([]uint, 0, len(links))
	assetIDSet := make(map[uint]struct{}, len(links))
	linksByItem := make(map[uint][]database.AssetItem, len(itemIDs))
	for _, link := range links {
		linksByItem[link.ItemID] = append(linksByItem[link.ItemID], link)
		if _, ok := assetIDSet[link.AssetID]; ok {
			continue
		}
		assetIDSet[link.AssetID] = struct{}{}
		assetIDs = append(assetIDs, link.AssetID)
	}

	var assetLinks []database.AssetItem
	if err := s.db.WithContext(ctx).
		Where("asset_id IN ?", assetIDs).
		Order("asset_id asc, role asc, segment_index asc, id asc").
		Find(&assetLinks).Error; err != nil {
		return nil, err
	}
	linksByAsset := make(map[uint][]database.AssetItem, len(assetLinks))
	for _, link := range assetLinks {
		linksByAsset[link.AssetID] = append(linksByAsset[link.AssetID], link)
	}

	var assets []database.MediaAsset
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", assetIDs).Order("id asc").Find(&assets).Error; err != nil {
		return nil, err
	}
	assetByID := make(map[uint]database.MediaAsset, len(assets))
	for _, asset := range assets {
		assetByID[asset.ID] = asset
	}

	var assetFileRows []database.AssetFile
	if err := s.db.WithContext(ctx).Where("asset_id IN ?", assetIDs).Order("asset_id asc, part_index asc, id asc").Find(&assetFileRows).Error; err != nil {
		return nil, err
	}
	fileIDsByAsset := make(map[uint][]uint, len(assetIDs))
	assetFilesByAsset := make(map[uint][]database.AssetFile, len(assetIDs))
	fileIDSet := make(map[uint]struct{}, len(assetFileRows))
	fileIDs := make([]uint, 0, len(assetFileRows))
	for _, row := range assetFileRows {
		fileIDsByAsset[row.AssetID] = append(fileIDsByAsset[row.AssetID], row.FileID)
		assetFilesByAsset[row.AssetID] = append(assetFilesByAsset[row.AssetID], row)
		if _, ok := fileIDSet[row.FileID]; ok {
			continue
		}
		fileIDSet[row.FileID] = struct{}{}
		fileIDs = append(fileIDs, row.FileID)
	}

	inventoryFilesByID := make(map[uint]database.InventoryFile, len(fileIDs))
	streamsByFileID := make(map[uint][]database.MediaStream, len(fileIDs))
	if len(fileIDs) > 0 {
		var inventoryFiles []database.InventoryFile
		if err := s.db.WithContext(ctx).Where("id IN ?", fileIDs).Order("id asc").Find(&inventoryFiles).Error; err != nil {
			return nil, err
		}
		for _, file := range inventoryFiles {
			inventoryFilesByID[file.ID] = file
		}

		var streams []database.MediaStream
		if err := s.db.WithContext(ctx).Where("file_id IN ?", fileIDs).Order("file_id asc, stream_index asc").Find(&streams).Error; err != nil {
			return nil, err
		}
		for _, stream := range streams {
			streamsByFileID[stream.FileID] = append(streamsByFileID[stream.FileID], stream)
		}
	}

	for itemID, itemLinks := range linksByItem {
		assetDetails := make([]CatalogAssetDetail, 0, len(itemLinks))
		seenAssets := make(map[uint]struct{}, len(itemLinks))
		for _, link := range itemLinks {
			if _, ok := seenAssets[link.AssetID]; ok {
				continue
			}
			asset, ok := assetByID[link.AssetID]
			if !ok {
				continue
			}
			fileSummaries, streamSummaries := buildCatalogAssetFileAndStreamSummaries(assetFilesByAsset[link.AssetID], inventoryFilesByID, streamsByFileID)
			assetDetails = append(assetDetails, BuildCatalogAssetDetail(CatalogAssetDetailInput{Asset: asset, Links: linksByAsset[link.AssetID], FileIDs: fileIDsByAsset[link.AssetID], Files: fileSummaries, Streams: streamSummaries}))
			seenAssets[link.AssetID] = struct{}{}
		}
		sort.SliceStable(assetDetails, func(i, j int) bool {
			if assetDetails[i].Status != assetDetails[j].Status {
				return assetDetails[i].Status < assetDetails[j].Status
			}
			return assetDetails[i].ID < assetDetails[j].ID
		})
		result[itemID] = assetDetails
	}
	return result, nil
}

func buildCatalogAssetFileAndStreamSummaries(assetFiles []database.AssetFile, inventoryFilesByID map[uint]database.InventoryFile, streamsByFileID map[uint][]database.MediaStream) ([]CatalogAssetFileSummary, []CatalogMediaStreamSummary) {
	if len(assetFiles) == 0 {
		return []CatalogAssetFileSummary{}, []CatalogMediaStreamSummary{}
	}
	fileSummaries := make([]CatalogAssetFileSummary, 0, len(assetFiles))
	streamSummaries := make([]CatalogMediaStreamSummary, 0)
	for _, assetFile := range assetFiles {
		file := inventoryFilesByID[assetFile.FileID]
		fileSummaries = append(fileSummaries, CatalogAssetFileSummary{
			FileID:              assetFile.FileID,
			Role:                strings.TrimSpace(assetFile.Role),
			PartIndex:           assetFile.PartIndex,
			StorageProvider:     strings.TrimSpace(file.StorageProvider),
			StoragePath:         strings.TrimSpace(file.StoragePath),
			StableIdentity:      strings.TrimSpace(file.StableIdentityKey),
			SizeBytes:           file.SizeBytes,
			Container:           strings.TrimSpace(file.Container),
			Status:              normalizeAvailabilityStatus(file.Status),
			ModifiedAt:          file.ModifiedAt,
			ProviderDiagnostics: buildCatalogProviderDiagnostics(file),
		})
		for _, stream := range streamsByFileID[assetFile.FileID] {
			streamSummaries = append(streamSummaries, buildCatalogMediaStreamSummary(stream, file))
		}
	}
	return fileSummaries, streamSummaries
}

func buildCatalogProviderDiagnostics(file database.InventoryFile) *CatalogProviderDiagnostics {
	storageProvider := strings.TrimSpace(file.StorageProvider)
	hashKeys := catalogHashKeys(file.HashesJSON)
	if storageProvider == "" && len(hashKeys) == 0 {
		return nil
	}
	diagnostics := &CatalogProviderDiagnostics{
		StorageProvider:   storageProvider,
		AvailableHashKeys: hashKeys,
	}
	if strings.TrimSpace(file.StableIdentityKey) != "" {
		diagnostics.MetadataIndicators = append(diagnostics.MetadataIndicators, "stable_identity")
	}
	if strings.TrimSpace(file.HashesJSON) != "" {
		diagnostics.MetadataIndicators = append(diagnostics.MetadataIndicators, "hash_info")
	}
	if file.ModifiedAt != nil {
		diagnostics.MetadataIndicators = append(diagnostics.MetadataIndicators, "modified")
	}
	return diagnostics
}

func catalogHashKeys(raw string) []string {
	decoded, ok := decodeCatalogJSONValue(raw)
	if !ok {
		return nil
	}
	values, ok := decoded.(map[string]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || !isCatalogScalarJSONValue(value) {
			continue
		}
		keys = append(keys, trimmedKey)
	}
	sort.Strings(keys)
	return keys
}

func buildCatalogMediaStreamSummary(stream database.MediaStream, file database.InventoryFile) CatalogMediaStreamSummary {
	defaultDisposition, forcedDisposition, externalDisposition, hearingImpairedDisposition := catalogMediaStreamDispositionFlags(stream.DispositionJSON)
	summary := CatalogMediaStreamSummary{
		FileID:          stream.FileID,
		StreamIndex:     stream.StreamIndex,
		StreamType:      strings.TrimSpace(stream.StreamType),
		Codec:           strings.TrimSpace(stream.Codec),
		Profile:         strings.TrimSpace(stream.Profile),
		Level:           stream.Level,
		Language:        strings.TrimSpace(stream.Language),
		Title:           strings.TrimSpace(stream.Title),
		Width:           stream.Width,
		Height:          stream.Height,
		AvgFrameRate:    strings.TrimSpace(stream.AvgFrameRate),
		RFrameRate:      strings.TrimSpace(stream.RFrameRate),
		FieldOrder:      strings.TrimSpace(stream.FieldOrder),
		ColorSpace:      strings.TrimSpace(stream.ColorSpace),
		BitDepth:        stream.BitDepth,
		PixelFormat:     strings.TrimSpace(stream.PixelFormat),
		ReferenceFrames: stream.ReferenceFrames,
		Channels:        stream.Channels,
		ChannelLayout:   strings.TrimSpace(stream.ChannelLayout),
		SampleRate:      stream.SampleRate,
		BitRate:         stream.BitRate,
		DurationSeconds: stream.DurationSeconds,
		Default:         defaultDisposition,
		Forced:          forcedDisposition,
		HearingImpaired: hearingImpairedDisposition,
		External:        externalDisposition,
	}
	if externalDisposition && strings.EqualFold(strings.TrimSpace(stream.StreamType), "subtitle") {
		available := normalizeAvailabilityStatus(file.Status) == AvailabilityAvailable && file.DeletedAt == nil
		summary.Available = &available
		if available {
			summary.URL = fmt.Sprintf("/api/v1/inventory-files/%d/stream", stream.FileID)
		}
	}
	return summary
}

func catalogMediaStreamDispositionFlags(raw string) (bool, bool, bool, bool) {
	decoded, ok := decodeCatalogJSONValue(raw)
	if !ok {
		return false, false, false, false
	}
	values, ok := decoded.(map[string]any)
	if !ok {
		return false, false, false, false
	}
	return catalogJSONBool(values["default"]), catalogJSONBool(values["forced"]), catalogJSONBool(values["external"]), catalogJSONBool(values["hearing_impaired"])
}

func catalogJSONBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes":
			return true
		}
	}
	return false
}
