package catalog

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProjectionRefreshRequest struct {
	ItemID    uint   `json:"item_id,omitempty"`
	LibraryID uint   `json:"library_id,omitempty"`
	RootPath  string `json:"root_path,omitempty"`
}

type ItemProjectionRefreshPayload struct {
	ItemID uint `json:"item_id"`
}

type LibraryProjectionRefreshPayload struct {
	LibraryID uint   `json:"library_id"`
	RootPath  string `json:"root_path"`
}

func (s *Service) RefreshItemProjection(ctx context.Context, itemID uint) error {
	if itemID == 0 {
		return errors.New("item id is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{ItemID: itemID})
	})
}

func (s *Service) RefreshLibraryProjection(ctx context.Context, libraryID uint, rootPath string) error {
	if libraryID == 0 {
		return errors.New("library id is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.refreshProjectionWithDB(ctx, tx, ProjectionRefreshRequest{LibraryID: libraryID, RootPath: strings.TrimSpace(rootPath)})
	})
}

func (s *Service) refreshProjectionWithDB(ctx context.Context, db *gorm.DB, request ProjectionRefreshRequest) error {
	items, targetItemID, targetDocIDs, targetRollupIDs, err := s.loadProjectionScope(ctx, db, request)
	if err != nil {
		return err
	}
	if len(targetDocIDs) == 0 && len(targetRollupIDs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	if len(targetRollupIDs) > 0 {
		if err := db.WithContext(ctx).Where("item_id IN ?", targetRollupIDs).Delete(&database.ItemRollup{}).Error; err != nil {
			return err
		}
	}
	if len(targetDocIDs) > 0 {
		if err := db.WithContext(ctx).Where("item_id IN ?", targetDocIDs).Delete(&database.CatalogSearchDocument{}).Error; err != nil {
			return err
		}
	}

	rollups := buildItemRollups(items, targetRollupIDs, now)
	if len(rollups) > 0 {
		if err := db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&rollups).Error; err != nil {
			return err
		}
	}

	docs, err := s.buildCatalogSearchDocuments(ctx, db.WithContext(ctx), items, targetDocIDs, now)
	if err != nil {
		return err
	}
	if len(docs) > 0 {
		if err := db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&docs).Error; err != nil {
			return err
		}
	}

	_ = targetItemID
	return nil
}

func (s *Service) loadProjectionScope(ctx context.Context, db *gorm.DB, request ProjectionRefreshRequest) ([]database.CatalogItem, uint, []uint, []uint, error) {
	if request.ItemID != 0 {
		var item database.CatalogItem
		err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", request.ItemID).First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, nil, nil, nil
		}
		if err != nil {
			return nil, 0, nil, nil, err
		}

		query := db.WithContext(ctx).Where("library_id = ? AND deleted_at IS NULL", item.LibraryID)
		if item.RootID != nil {
			query = query.Where("id = ? OR root_id = ?", item.ID, *item.RootID)
		} else {
			query = query.Where("id = ?", item.ID)
		}
		var items []database.CatalogItem
		if err := query.Order("id asc").Find(&items).Error; err != nil {
			return nil, 0, nil, nil, err
		}
		rollupIDs := projectionRollupIDs(items, []uint{item.ID})
		return items, item.ID, []uint{item.ID}, rollupIDs, nil
	}

	var items []database.CatalogItem
	if err := db.WithContext(ctx).Where("library_id = ? AND deleted_at IS NULL", request.LibraryID).Order("id asc").Find(&items).Error; err != nil {
		return nil, 0, nil, nil, err
	}
	if len(items) == 0 {
		return nil, 0, nil, nil, nil
	}

	scopedIDs := make([]uint, 0, len(items))
	normalizedRootPath := strings.TrimSpace(request.RootPath)
	for _, item := range items {
		if normalizedRootPath == "" || item.Path == normalizedRootPath || strings.HasPrefix(item.Path, normalizedRootPath+"/") {
			scopedIDs = append(scopedIDs, item.ID)
		}
	}
	if len(scopedIDs) == 0 {
		return items, 0, nil, nil, nil
	}

	return items, 0, scopedIDs, projectionRollupIDs(items, scopedIDs), nil
}

func projectionRollupIDs(items []database.CatalogItem, scopedIDs []uint) []uint {
	itemByID := make(map[uint]database.CatalogItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}
	ids := make(map[uint]struct{}, len(scopedIDs))
	for _, id := range scopedIDs {
		current, ok := itemByID[id]
		if !ok {
			continue
		}
		ids[current.ID] = struct{}{}
		for current.ParentID != nil {
			parentID := *current.ParentID
			ids[parentID] = struct{}{}
			parent, ok := itemByID[parentID]
			if !ok {
				break
			}
			current = parent
		}
	}
	return sortedProjectionIDs(ids)
}

func buildItemRollups(items []database.CatalogItem, targetIDs []uint, updatedAt time.Time) []database.ItemRollup {
	if len(targetIDs) == 0 {
		return nil
	}
	childrenByParent := make(map[uint][]database.CatalogItem)
	itemByID := make(map[uint]database.CatalogItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
		if item.ParentID != nil {
			childrenByParent[*item.ParentID] = append(childrenByParent[*item.ParentID], item)
		}
	}

	rollups := make([]database.ItemRollup, 0, len(targetIDs))
	for _, itemID := range targetIDs {
		item, ok := itemByID[itemID]
		if !ok {
			continue
		}
		descendants := collectProjectionDescendants(childrenByParent, item.ID)
		rollup := database.ItemRollup{ItemID: item.ID, UpdatedAt: updatedAt}
		rollup.ChildCount = len(descendants)
		for _, descendant := range descendants {
			if len(childrenByParent[descendant.ID]) > 0 {
				continue
			}
			switch strings.TrimSpace(descendant.AvailabilityStatus) {
			case AvailabilityAvailable:
				rollup.AvailableCount++
			case AvailabilityUnaired:
				rollup.UnairedCount++
			case AvailabilityMissing, AvailabilityNoLocalMedia:
				rollup.MissingCount++
			}
			rollup.LatestAirDate = maxCatalogTime(rollup.LatestAirDate, descendant.FirstAirDate, descendant.ReleaseDate)
			rollup.LatestAddedAt = maxCatalogTime(rollup.LatestAddedAt, &descendant.CreatedAt)
		}
		if rollup.ChildCount == 0 {
			rollup.LatestAirDate = maxCatalogTime(rollup.LatestAirDate, item.FirstAirDate, item.ReleaseDate)
			rollup.LatestAddedAt = maxCatalogTime(rollup.LatestAddedAt, &item.CreatedAt)
		}
		rollups = append(rollups, rollup)
	}
	return rollups
}

func (s *Service) buildCatalogSearchDocuments(ctx context.Context, tx *gorm.DB, items []database.CatalogItem, targetIDs []uint, updatedAt time.Time) ([]database.CatalogSearchDocument, error) {
	if len(targetIDs) == 0 {
		return nil, nil
	}
	itemByID := make(map[uint]database.CatalogItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}

	externalIDsByItem, err := s.loadExternalIDsByItem(ctx, tx, targetIDs)
	if err != nil {
		return nil, err
	}
	peopleByItem, err := s.loadPeopleByItem(ctx, tx, targetIDs)
	if err != nil {
		return nil, err
	}
	tagsByItem, err := s.loadTagsByItem(ctx, tx, targetIDs)
	if err != nil {
		return nil, err
	}

	docs := make([]database.CatalogSearchDocument, 0, len(targetIDs))
	for _, itemID := range targetIDs {
		item, ok := itemByID[itemID]
		if !ok {
			continue
		}
		docs = append(docs, database.CatalogSearchDocument{
			ItemID:             item.ID,
			LibraryID:          item.LibraryID,
			ItemType:           normalizeCatalogType(item.Type),
			Title:              strings.TrimSpace(item.Title),
			OriginalTitle:      strings.TrimSpace(item.OriginalTitle),
			PeopleText:         joinProjectionText(peopleByItem[item.ID]),
			TagsText:           joinProjectionText(tagsByItem[item.ID]),
			ProviderIDsText:    joinProjectionText(externalIDsByItem[item.ID]),
			Year:               item.Year,
			OfficialRating:     strings.TrimSpace(item.OfficialRating),
			AvailabilityStatus: normalizeAvailabilityStatus(item.AvailabilityStatus),
			UpdatedAt:          updatedAt,
		})
	}
	return docs, nil
}

func (s *Service) loadExternalIDsByItem(ctx context.Context, tx *gorm.DB, itemIDs []uint) (map[uint][]string, error) {
	var rows []database.CatalogExternalID
	if err := tx.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("item_id asc, provider asc, provider_type asc, external_id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	byItem := make(map[uint][]string, len(itemIDs))
	for _, row := range rows {
		parts := []string{strings.TrimSpace(row.Provider), strings.TrimSpace(row.ProviderType), strings.TrimSpace(row.ExternalID)}
		byItem[row.ItemID] = append(byItem[row.ItemID], strings.Join(parts, ":"))
	}
	return byItem, nil
}

func (s *Service) loadPeopleByItem(ctx context.Context, tx *gorm.DB, itemIDs []uint) (map[uint][]string, error) {
	var rows []struct {
		ItemID uint
		Name   string
	}
	if err := tx.WithContext(ctx).
		Table("item_people").
		Select("item_people.item_id, people.name").
		Joins("JOIN people ON people.id = item_people.person_id").
		Where("item_people.item_id IN ?", itemIDs).
		Order("item_people.item_id asc, item_people.sort_order asc, people.name asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	byItem := make(map[uint][]string, len(itemIDs))
	for _, row := range rows {
		byItem[row.ItemID] = append(byItem[row.ItemID], strings.TrimSpace(row.Name))
	}
	return byItem, nil
}

func (s *Service) loadTagsByItem(ctx context.Context, tx *gorm.DB, itemIDs []uint) (map[uint][]string, error) {
	var rows []struct {
		ItemID uint
		Name   string
	}
	if err := tx.WithContext(ctx).
		Table("item_tags").
		Select("item_tags.item_id, tags.name").
		Joins("JOIN tags ON tags.id = item_tags.tag_id").
		Where("item_tags.item_id IN ?", itemIDs).
		Order("item_tags.item_id asc, tags.kind asc, tags.name asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	byItem := make(map[uint][]string, len(itemIDs))
	for _, row := range rows {
		byItem[row.ItemID] = append(byItem[row.ItemID], strings.TrimSpace(row.Name))
	}
	return byItem, nil
}

func collectProjectionDescendants(childrenByParent map[uint][]database.CatalogItem, itemID uint) []database.CatalogItem {
	children := childrenByParent[itemID]
	if len(children) == 0 {
		return nil
	}
	result := make([]database.CatalogItem, 0, len(children))
	for _, child := range children {
		result = append(result, child)
		result = append(result, collectProjectionDescendants(childrenByParent, child.ID)...)
	}
	return result
}

func joinProjectionText(values []string) string {
	if len(values) == 0 {
		return ""
	}
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		unique[trimmed] = struct{}{}
	}
	sorted := make([]string, 0, len(unique))
	for value := range unique {
		sorted = append(sorted, value)
	}
	sort.Strings(sorted)
	return strings.Join(sorted, " ")
}

func sortedProjectionIDs(values map[uint]struct{}) []uint {
	ids := make([]uint, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func maxCatalogTime(current *time.Time, candidates ...*time.Time) *time.Time {
	best := current
	for _, candidate := range candidates {
		if candidate == nil || candidate.IsZero() {
			continue
		}
		if best == nil || candidate.After(*best) {
			value := candidate.UTC()
			best = &value
		}
	}
	return best
}
