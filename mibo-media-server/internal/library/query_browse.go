package library

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) GetLibrary(ctx context.Context, libraryID uint) (LibraryDetail, error) {
	var detail LibraryDetail
	if err := s.db.WithContext(ctx).First(&detail.Library, libraryID).Error; err != nil {
		return LibraryDetail{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.CatalogItem{}).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Where("parent_id IS NULL").
		Count(&detail.CatalogItemsCount).Error; err != nil {
		return LibraryDetail{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.InventoryFile{}).Where("library_id = ? AND deleted_at IS NULL", libraryID).Count(&detail.InventoryFilesCount).Error; err != nil {
		return LibraryDetail{}, err
	}
	config, err := s.EffectiveLibraryConfig(ctx, libraryID)
	if err != nil {
		return LibraryDetail{}, err
	}
	detail.Paths = config.PathsView()
	detail.Policies = config.PoliciesView()
	detail.ProbeSummary = decodeLibraryProbeSummary(detail.Library)
	detail.Collections = s.librarySourceCollections(ctx, libraryID, detail.ProbeSummary)
	return detail, nil
}

func (s *Service) librarySourceCollections(ctx context.Context, libraryID uint, probe SourceProbeSummary) []SourceCollection {
	counts := map[string]int64{}
	for className, count := range probe.Classes {
		counts[className] = int64(count)
	}
	type row struct {
		ContentClass string
		Count        int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).Model(&database.InventoryFile{}).
		Select("content_class, count(*) as count").
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Group("content_class").
		Scan(&rows).Error; err == nil {
		for _, result := range rows {
			className := strings.TrimSpace(result.ContentClass)
			if className == "" {
				className = SourceContentClassVideo
			}
			counts[className] = result.Count
		}
	}
	collections := make([]SourceCollection, 0, len(counts))
	for _, className := range []string{SourceContentClassVideo, SourceContentClassAudio, SourceContentClassText, SourceContentClassImage, SourceContentClassOther} {
		if counts[className] <= 0 {
			continue
		}
		collections = append(collections, SourceCollection{ContentClass: className, Label: sourceCollectionLabel(className), Count: counts[className]})
	}
	return collections
}

func sourceCollectionLabel(className string) string {
	switch className {
	case SourceContentClassVideo:
		return "视频"
	case SourceContentClassAudio:
		return "音乐"
	case SourceContentClassText:
		return "文本"
	case SourceContentClassImage:
		return "图片"
	default:
		return "其他"
	}
}

func decodeLibraryProbeSummary(record database.Library) SourceProbeSummary {
	var summary SourceProbeSummary
	if strings.TrimSpace(record.ProbeSummaryJSON) != "" {
		_ = json.Unmarshal([]byte(record.ProbeSummaryJSON), &summary)
	}
	if summary.Classes == nil {
		summary.Classes = emptySourceProbeClassCounts()
	}
	if summary.Status == "" {
		summary.Status = record.ProbeStatus
	}
	if summary.Status == "" {
		summary.Status = SourceProbeStatusPending
	}
	finalizeSourceProbeSummary(&summary)
	return summary
}

func ParseBrowseYear(value string) *int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	year, err := strconv.Atoi(trimmed)
	if err != nil || year <= 0 {
		return nil
	}
	return &year
}
