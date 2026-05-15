package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
)

type HomeContentSection struct {
	Key   string            `json:"key"`
	Title string            `json:"title"`
	Items []CatalogListItem `json:"items"`
}

type HomeMediaOverview struct {
	Sections []HomeMediaSectionSummary `json:"sections"`
}

type HomeMediaSectionSummary struct {
	Key   string            `json:"key"`
	Title string            `json:"title"`
	Count int               `json:"count"`
	Items []CatalogListItem `json:"items"`
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
	Watched   bool
	InProgress bool
	LastPlayedAt string
}

func (s *Service) ListLibraryItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	if libraryID == 0 {
		return nil, errors.New("library id is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	projected, err := s.ListLibraryProjectionItems(ctx, libraryID, query, typeFilter, limit)
	if err != nil {
		return nil, err
	}
	if err := s.attachOrganizingSummaries(ctx, projected); err != nil {
		return nil, err
	}
	discovered, err := s.discoveredBrowseEntries(ctx, BrowseItemsInput{LibraryID: libraryID, Query: query, TypeFilter: typeFilter, Limit: limit})
	if err != nil {
		return nil, err
	}
	if len(discovered) == 0 {
		return projected, nil
	}
	entries := make([]browseListEntry, 0, len(projected)+len(discovered))
	for _, item := range projected {
		entries = append(entries, browseListEntry{Item: item, TitleKey: catalogListItemTitleKey(item), Year: item.Year, StableID: item.MetadataItemID})
	}
	entries = append(entries, discovered...)
	applyBrowseListEntryOrder(entries, BrowseItemsInput{Sort: "title", SortDirection: "asc"})
	capacity := len(entries)
	if capacity > limit {
		capacity = limit
	}
	items := make([]CatalogListItem, 0, capacity)
	for _, entry := range entries {
		items = append(items, entry.Item)
		if len(items) == limit {
			break
		}
	}
	return items, nil
}

func (s *Service) ListItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	return s.SearchProjectionItems(ctx, libraryID, query, typeFilter, limit)
}

func (s *Service) SearchItems(ctx context.Context, libraryID uint, query string, typeFilter string, limit int) ([]CatalogListItem, error) {
	return s.SearchProjectionItems(ctx, libraryID, query, typeFilter, limit)
}

func (s *Service) BrowseItems(ctx context.Context, input BrowseItemsInput) (BrowseItemsResult, error) {
	input = normalizeBrowseItemsInput(input)
	entries, err := s.browseEntries(ctx, input)
	if err != nil {
		return BrowseItemsResult{}, err
	}
	return buildBrowseItemsResult(entries, int64(len(entries)), input), nil
}

func (s *Service) browseEntries(ctx context.Context, input BrowseItemsInput) ([]browseListEntry, error) {
	projectedEntries, err := s.projectedBrowseEntries(ctx, input)
	if err != nil {
		return nil, err
	}
	discoveredEntries, err := s.discoveredBrowseEntries(ctx, input)
	if err != nil {
		return nil, err
	}
	entries := make([]browseListEntry, 0, len(projectedEntries)+len(discoveredEntries))
	entries = append(entries, projectedEntries...)
	entries = append(entries, discoveredEntries...)
	entries = filterBrowseListEntriesByOrganizing(entries, input.OrganizingState)
	if err := s.attachBrowseUserState(ctx, input.UserID, entries); err != nil {
		return nil, err
	}
	entries = filterBrowseListEntriesByWatchedState(entries, input.WatchedState)
	applyBrowseListEntryOrder(entries, input)
	return entries, nil
}

func buildBrowseItemsResult(entries []browseListEntry, total int64, input BrowseItemsInput) BrowseItemsResult {
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
	}
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
			SELECT 1 FROM recognition_candidates
			WHERE recognition_candidates.primary_inventory_id = inventory_files.id
			AND recognition_candidates.candidate_type = ?
			AND recognition_candidates.superseded_at IS NULL
		)`, "supplemental").
		Where(`NOT EXISTS (
			SELECT 1 FROM resource_files
			JOIN resource_metadata_links ON resource_metadata_links.resource_id = resource_files.resource_id
			JOIN metadata_items ON metadata_items.id = resource_metadata_links.metadata_item_id AND metadata_items.deleted_at IS NULL
			WHERE resource_files.inventory_file_id = inventory_files.id
		)`)
	if input.LibraryID != 0 {
		query = query.Where("inventory_files.library_id = ?", input.LibraryID)
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

func (s *Service) projectedBrowseEntries(ctx context.Context, input BrowseItemsInput) ([]browseListEntry, error) {
	allowedTypes := []string{database.MetadataItemTypeMovie, database.MetadataItemTypeSeries}
	switch input.TypeFilter {
	case ItemTypeMovie:
		allowedTypes = []string{database.MetadataItemTypeMovie}
	case ItemTypeSeries, "show":
		allowedTypes = []string{database.MetadataItemTypeSeries}
	case ItemTypeEpisode:
		allowedTypes = []string{database.MetadataItemTypeEpisode}
	}
	db := s.db.WithContext(ctx).
		Table("library_metadata_projections AS p").
		Select("p.*").
		Joins("JOIN metadata_items AS m ON m.id = p.metadata_item_id AND m.deleted_at IS NULL").
		Where("p.hidden = ?", false).
		Where("p.availability_status = ?", database.ProjectionAvailabilityAvailable).
		Where("p.item_type IN ?", allowedTypes)
	if input.LibraryID != 0 {
		db = db.Where("p.library_id = ?", input.LibraryID)
	}
	if input.Query != "" {
		like := "%" + strings.ToLower(input.Query) + "%"
		if input.LibraryID != 0 {
			db = db.Joins("LEFT JOIN library_search_documents AS d ON d.library_id = p.library_id AND d.metadata_item_id = p.metadata_item_id").Where("LOWER(p.title) LIKE ? OR LOWER(m.original_title) LIKE ? OR LOWER(COALESCE(d.people_text, '')) LIKE ? OR LOWER(COALESCE(d.tags_text, '')) LIKE ? OR LOWER(COALESCE(d.provider_ids_text, '')) LIKE ? OR LOWER(COALESCE(d.resource_text, '')) LIKE ?", like, like, like, like, like, like)
		} else {
			db = db.Joins("LEFT JOIN metadata_search_documents AS d ON d.metadata_item_id = p.metadata_item_id").Where("LOWER(p.title) LIKE ? OR LOWER(m.original_title) LIKE ? OR LOWER(COALESCE(d.people_text, '')) LIKE ? OR LOWER(COALESCE(d.tags_text, '')) LIKE ? OR LOWER(COALESCE(d.provider_ids_text, '')) LIKE ?", like, like, like, like, like)
		}
	}
	if input.Year != nil {
		db = db.Where("m.year = ?", *input.Year)
	}
	if input.MinRating != nil {
		db = db.Where("m.community_rating >= ?", *input.MinRating)
	}
	if input.Genre != "" {
		db = db.Where(`EXISTS (
			SELECT 1 FROM metadata_item_tags mit
			JOIN tags t ON t.id = mit.tag_id
			WHERE mit.metadata_item_id = p.metadata_item_id
			AND LOWER(t.kind) = 'genre'
			AND LOWER(t.name) = ?
		)`, strings.ToLower(input.Genre))
	}
	var rawProjections []database.LibraryMetadataProjection
	if err := db.Order("p.library_id asc").Order("p.metadata_item_id asc").Find(&rawProjections).Error; err != nil {
		return nil, err
	}
	projections := rawProjections
	if input.LibraryID == 0 {
		projections = dedupeBrowseProjections(rawProjections)
	}
	items, err := s.buildProjectionListItems(ctx, projections)
	if err != nil {
		return nil, err
	}
	if err := s.attachOrganizingSummaries(ctx, items); err != nil {
		return nil, err
	}
	projectionByItemID := make(map[uint]database.LibraryMetadataProjection, len(projections))
	for _, projection := range projections {
		projectionByItemID[projection.MetadataItemID] = projection
	}
	entries := make([]browseListEntry, 0, len(items))
	for _, item := range items {
		projection, ok := projectionByItemID[item.MetadataItemID]
		if !ok {
			continue
		}
		createdAt := projection.UpdatedAt.Format("2006-01-02T15:04:05.000000000Z07:00")
		if projection.LatestAddedAt != nil {
			createdAt = projection.LatestAddedAt.Format("2006-01-02T15:04:05.000000000Z07:00")
		}
		entries = append(entries, browseListEntry{
			Item: item,
			TitleKey: catalogListItemTitleKey(item),
			Year: item.Year,
			CreatedAt: createdAt,
			StableID: item.MetadataItemID,
		})
	}
	return entries, nil
}

func dedupeBrowseProjections(projections []database.LibraryMetadataProjection) []database.LibraryMetadataProjection {
	result := make([]database.LibraryMetadataProjection, 0, len(projections))
	seen := make(map[uint]struct{}, len(projections))
	for _, projection := range projections {
		if _, ok := seen[projection.MetadataItemID]; ok {
			continue
		}
		seen[projection.MetadataItemID] = struct{}{}
		result = append(result, projection)
	}
	return result
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

func filterBrowseListEntriesByWatchedState(entries []browseListEntry, watchedState string) []browseListEntry {
	switch watchedState {
	case "unwatched", "in_progress", "watched":
	default:
		return entries
	}
	filtered := make([]browseListEntry, 0, len(entries))
	for _, entry := range entries {
		switch watchedState {
		case "unwatched":
			if !entry.Watched && !entry.InProgress {
				filtered = append(filtered, entry)
			}
		case "in_progress":
			if entry.InProgress {
				filtered = append(filtered, entry)
			}
		case "watched":
			if entry.Watched {
				filtered = append(filtered, entry)
			}
		}
	}
	return filtered
}

func (s *Service) attachBrowseUserState(ctx context.Context, userID uint, entries []browseListEntry) error {
	if userID == 0 || len(entries) == 0 {
		return nil
	}
	metadataIDs := make([]uint, 0, len(entries))
	for _, entry := range entries {
		if entry.Item.MetadataItemID != 0 {
			metadataIDs = append(metadataIDs, entry.Item.MetadataItemID)
		}
	}
	if len(metadataIDs) == 0 {
		return nil
	}
	var rows []database.UserMetadataData
	if err := s.db.WithContext(ctx).Where("user_id = ? AND metadata_item_id IN ?", userID, metadataIDs).Find(&rows).Error; err != nil {
		return err
	}
	stateByMetadataID := make(map[uint]database.UserMetadataData, len(rows))
	for _, row := range rows {
		stateByMetadataID[row.MetadataItemID] = row
	}
	for idx := range entries {
		row, ok := stateByMetadataID[entries[idx].Item.MetadataItemID]
		if !ok {
			continue
		}
		entries[idx].Watched = row.CompletedAt != nil
		entries[idx].InProgress = row.CompletedAt == nil && (row.PositionSeconds > 0 || row.LastPlayedAt != nil)
		if row.LastPlayedAt != nil {
			entries[idx].LastPlayedAt = row.LastPlayedAt.Format("2006-01-02T15:04:05.000000000Z07:00")
		}
	}
	return nil
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
	conditions, err := s.organizingConditionsByMetadataItemID(ctx, itemIDs)
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

func (s *Service) organizingConditionsByMetadataItemID(ctx context.Context, itemIDs []uint) (map[uint][]database.IngestCondition, error) {
	result := make(map[uint][]database.IngestCondition, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}
	var conditions []database.IngestCondition
	if err := s.db.WithContext(ctx).Where("metadata_item_id IN ?", itemIDs).Order("metadata_item_id asc, condition_type asc").Find(&conditions).Error; err != nil {
		return nil, err
	}
	for _, condition := range conditions {
		if condition.MetadataItemID == nil {
			continue
		}
		result[*condition.MetadataItemID] = append(result[*condition.MetadataItemID], condition)
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
		case "watch_status":
			leftRank := browseWatchStatusRank(left)
			rightRank := browseWatchStatusRank(right)
			if leftRank != rightRank {
				if desc {
					return leftRank > rightRank
				}
				return leftRank < rightRank
			}
			if left.LastPlayedAt != right.LastPlayedAt {
				return compareBrowseString(left.LastPlayedAt, right.LastPlayedAt, desc)
			}
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

func browseWatchStatusRank(entry browseListEntry) int {
	if entry.Watched {
		return 2
	}
	if entry.InProgress {
		return 1
	}
	return 0
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

func (s *Service) GetGovernanceWorkspace(ctx context.Context, metadataItemID uint, libraryID uint) (CatalogGovernanceWorkspace, error) {
	if metadataItemID == 0 {
		return CatalogGovernanceWorkspace{}, errors.New("metadata item id is required")
	}
	var item database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", metadataItemID).First(&item).Error; err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	projection := database.LibraryMetadataProjection{LibraryID: libraryID, MetadataItemID: item.ID, AvailabilityStatus: database.ProjectionAvailabilityUnavailable}
	if libraryID != 0 {
		_ = s.db.WithContext(ctx).Where("library_id = ? AND metadata_item_id = ?", libraryID, item.ID).First(&projection).Error
	}
	imagesByItem, err := s.loadMetadataItemImages(ctx, []uint{item.ID})
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	identitiesByItem, err := s.loadMetadataExternalIdentities(ctx, []uint{item.ID})
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	resources, err := s.loadMetadataResourceDetails(ctx, item.ID, libraryID)
	if err != nil {
		return CatalogGovernanceWorkspace{}, err
	}
	return CatalogGovernanceWorkspace{
		MetadataItemID:      item.ID,
		LibraryID:           projection.LibraryID,
		Type:                metadataItemTypeToCatalogType(item.ItemType),
		Title:               strings.TrimSpace(item.Title),
		AvailabilityStatus:  metadataProjectionAvailability(projection),
		GovernanceStatus:    strings.TrimSpace(item.GovernanceStatus),
		SelectedImages:      ensureCatalogSelectedImages(selectedMetadataImages(imagesByItem[item.ID])),
		ImageCandidates:     ensureCatalogSelectedImages(imagesByItem[item.ID]),
		ExternalIdentities:  ensureCatalogExternalIdentities(identitiesByItem[item.ID]),
		SourceEvidence:      []CatalogSourceEvidence{},
		FieldStates:         []CatalogFieldState{},
		Resources:           ensureCatalogResourceDetails(resources),
		Classification:      []CatalogClassificationDecision{},
		ClassificationRules: []CatalogClassificationRuleSummary{},
		RecommendedChildren: []CatalogListItem{},
	}, nil
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
	var projections []database.LibraryMetadataProjection
	if err := s.db.WithContext(ctx).
		Where("hidden = ?", false).
		Where("availability_status = ?", database.ProjectionAvailabilityAvailable).
		Where("item_type IN ?", []string{database.MetadataItemTypeMovie, database.MetadataItemTypeSeries}).
		Order("latest_added_at desc").Order("metadata_item_id desc").
		Limit(limit).
		Find(&projections).Error; err != nil {
		return nil, err
	}
	return s.buildProjectionListItems(ctx, projections)
}

func (s *Service) ListHomeContentSections(ctx context.Context, limit int) ([]HomeContentSection, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	definitions := []struct {
		key      string
		title    string
		itemType string
	}{
		{key: "movies", title: "电影", itemType: database.MetadataItemTypeMovie},
		{key: "series", title: "剧集", itemType: database.MetadataItemTypeSeries},
	}
	sections := make([]HomeContentSection, 0, len(definitions))
	for _, definition := range definitions {
		items, err := s.listLatestProjectionItemsByType(ctx, definition.itemType, limit)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			continue
		}
		sections = append(sections, HomeContentSection{Key: definition.key, Title: definition.title, Items: items})
	}
	return sections, nil
}

func (s *Service) ListHomeMediaOverview(ctx context.Context, previewLimit int) (HomeMediaOverview, error) {
	if previewLimit <= 0 || previewLimit > 20 {
		previewLimit = 4
	}
	definitions := []struct {
		key      string
		title    string
		itemType string
	}{
		{key: "movies", title: "电影", itemType: database.MetadataItemTypeMovie},
		{key: "series", title: "剧集", itemType: database.MetadataItemTypeSeries},
	}
	sections := make([]HomeMediaSectionSummary, 0, len(definitions))
	for _, definition := range definitions {
		count, err := s.countProjectionItemsByType(ctx, definition.itemType)
		if err != nil {
			return HomeMediaOverview{}, err
		}
		items, err := s.listLatestProjectionItemsByType(ctx, definition.itemType, previewLimit)
		if err != nil {
			return HomeMediaOverview{}, err
		}
		sections = append(sections, HomeMediaSectionSummary{Key: definition.key, Title: definition.title, Count: count, Items: items})
	}
	return HomeMediaOverview{Sections: sections}, nil
}

func (s *Service) listLatestProjectionItemsByType(ctx context.Context, itemType string, limit int) ([]CatalogListItem, error) {
	if strings.TrimSpace(itemType) == "" || limit <= 0 {
		return []CatalogListItem{}, nil
	}
	var projections []database.LibraryMetadataProjection
	if err := s.db.WithContext(ctx).
		Where("hidden = ?", false).
		Where("availability_status = ?", database.ProjectionAvailabilityAvailable).
		Where("item_type = ?", itemType).
		Order("latest_added_at desc").Order("metadata_item_id desc").
		Limit(limit).
		Find(&projections).Error; err != nil {
		return nil, err
	}
	return s.buildProjectionListItems(ctx, projections)
}

func (s *Service) countProjectionItemsByType(ctx context.Context, itemType string) (int, error) {
	if strings.TrimSpace(itemType) == "" {
		return 0, nil
	}
	var count int64
	if err := s.db.WithContext(ctx).
		Model(&database.LibraryMetadataProjection{}).
		Where("hidden = ?", false).
		Where("availability_status = ?", database.ProjectionAvailabilityAvailable).
		Where("item_type = ?", itemType).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *Service) IsGovernanceTargetAllowed(ctx context.Context, workspaceItemID uint, targetItemID uint) (bool, error) {
	if workspaceItemID == 0 || targetItemID == 0 {
		return false, nil
	}
	var item database.MetadataItem
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", targetItemID).First(&item).Error
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
		if err = s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *item.ParentID).First(&item).Error; err != nil {
			return false, err
		}
	}
}

func buildCatalogResourceFileAndStreamSummaries(resourceFiles []database.ResourceFile, inventoryFilesByID map[uint]database.InventoryFile, streamsByFileID map[uint][]database.MediaStream) ([]CatalogResourceFileSummary, []CatalogMediaStreamSummary) {
	if len(resourceFiles) == 0 {
		return []CatalogResourceFileSummary{}, []CatalogMediaStreamSummary{}
	}
	fileSummaries := make([]CatalogResourceFileSummary, 0, len(resourceFiles))
	streamSummaries := make([]CatalogMediaStreamSummary, 0)
	for _, resourceFile := range resourceFiles {
		file := inventoryFilesByID[resourceFile.InventoryFileID]
		fileSummaries = append(fileSummaries, CatalogResourceFileSummary{
			FileID:              resourceFile.InventoryFileID,
			Role:                strings.TrimSpace(resourceFile.Role),
			PartIndex:           resourceFile.PartIndex,
			StorageProvider:     strings.TrimSpace(file.StorageProvider),
			StoragePath:         strings.TrimSpace(file.StoragePath),
			StableIdentity:      strings.TrimSpace(file.StableIdentityKey),
			SizeBytes:           file.SizeBytes,
			Container:           strings.TrimSpace(file.Container),
			Status:              normalizeAvailabilityStatus(file.Status),
			ModifiedAt:          file.ModifiedAt,
			ProviderDiagnostics: buildCatalogProviderDiagnostics(file),
		})
		for _, stream := range streamsByFileID[resourceFile.InventoryFileID] {
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
