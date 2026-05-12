package library

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

func uniqueCatalogScanTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(strings.TrimPrefix(tag, "#"))
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

func (s *Service) cleanupMissingCatalog(ctx context.Context, libraryID uint, rootPath string, seen map[string]struct{}, skippedDirectories []string) error {
	var dirtyMissingFileIDs []uint
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var files []database.InventoryFile
		fileQuery := tx.WithContext(ctx).
			Where("library_id = ? AND deleted_at IS NULL", libraryID)
		fileQuery = applyScopedPathFilter(fileQuery, "storage_path", rootPath)
		fileQuery = applySkippedDirectoryFilter(fileQuery, "storage_path", skippedDirectories)
		if err := fileQuery.Order("id asc").Find(&files).Error; err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}

		missingFileIDs := make([]uint, 0)
		for _, file := range files {
			if _, ok := seen[file.StoragePath]; ok {
				continue
			}
			missingFileIDs = append(missingFileIDs, file.ID)
		}
		if len(missingFileIDs) > 0 {
			missingAt := time.Now().UTC()
			for _, batch := range chunkUints(missingFileIDs, sqliteVariableChunkSize) {
				if err := tx.WithContext(ctx).
					Model(&database.InventoryFile{}).
					Where("id IN ?", batch).
					Updates(map[string]any{"status": inventory.FileStatusMissing, "missing_since": gorm.Expr("COALESCE(missing_since, ?)", missingAt), "deleted_at": nil}).Error; err != nil {
					return err
				}
			}
			dirtyMissingFileIDs = append(dirtyMissingFileIDs, missingFileIDs...)
		}

		return nil
	}); err != nil {
		return err
	}
	for _, fileID := range dirtyMissingFileIDs {
		s.markInventoryFileDirty(ctx, fileID, "scanner_missing")
	}
	s.markProjectionLibraryDirty(ctx, libraryID, rootPath, "scanner_missing")
	return nil
}

type inventoryFileStableIdentityReuseInput struct {
	LibraryID         uint
	StorageProvider   string
	StableIdentityKey string
	StoragePath       string
	HashesJSON        string
	ThumbnailURL      string
	SizeBytes         int64
	ModifiedAt        *time.Time
	Container         string
	ContentClass      string
	Status            string
	ScanState         string
}

func loadInventoryFilesByStableIdentity(ctx context.Context, db *gorm.DB, libraryID uint, storageProvider string, stableIdentityKeys []string) ([]database.InventoryFile, error) {
	keys := make([]string, 0, len(stableIdentityKeys))
	seen := make(map[string]struct{}, len(stableIdentityKeys))
	for _, key := range stableIdentityKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		keys = append(keys, trimmed)
	}
	if len(keys) == 0 {
		return nil, nil
	}
	files := make([]database.InventoryFile, 0, len(keys))
	for _, batch := range chunkStrings(keys, sqliteVariableChunkSize) {
		var partial []database.InventoryFile
		if err := db.WithContext(ctx).
			Where("library_id = ? AND storage_provider = ? AND stable_identity_key IN ? AND deleted_at IS NULL", libraryID, strings.TrimSpace(storageProvider), batch).
			Order("id asc").
			Find(&partial).Error; err != nil {
			return nil, err
		}
		files = append(files, partial...)
	}
	return files, nil
}

func reuseInventoryFileByStableIdentity(ctx context.Context, db *gorm.DB, input inventoryFileStableIdentityReuseInput) (database.InventoryFile, bool, error) {
	stableIdentityKey := strings.TrimSpace(input.StableIdentityKey)
	if stableIdentityKey == "" {
		return database.InventoryFile{}, false, nil
	}
	files, err := loadInventoryFilesByStableIdentity(ctx, db, input.LibraryID, input.StorageProvider, []string{stableIdentityKey})
	if err != nil {
		return database.InventoryFile{}, false, err
	}
	if len(files) == 0 {
		return database.InventoryFile{}, false, nil
	}
	file := files[0]
	if err := applyInventoryFileStableIdentityReuseUpdate(ctx, db, file.ID, input); err != nil {
		return database.InventoryFile{}, false, err
	}
	var refreshed database.InventoryFile
	if err := db.WithContext(ctx).First(&refreshed, file.ID).Error; err != nil {
		return database.InventoryFile{}, false, err
	}
	return refreshed, true, nil
}

func applyInventoryFileStableIdentityReuseUpdate(ctx context.Context, db *gorm.DB, fileID uint, input inventoryFileStableIdentityReuseInput) error {
	updates := map[string]any{
		"storage_path":  strings.TrimSpace(input.StoragePath),
		"hashes_json":   strings.TrimSpace(input.HashesJSON),
		"thumbnail_url": strings.TrimSpace(input.ThumbnailURL),
		"size_bytes":    input.SizeBytes,
		"modified_at":   input.ModifiedAt,
		"container":     strings.TrimSpace(input.Container),
		"content_class": strings.TrimSpace(input.ContentClass),
		"status":        strings.TrimSpace(input.Status),
		"scan_state":    strings.TrimSpace(input.ScanState),
		"missing_since": gorm.Expr("NULL"),
		"deleted_at":    gorm.Expr("NULL"),
	}
	return db.WithContext(ctx).
		Model(&database.InventoryFile{}).
		Where("id = ?", fileID).
		Select("storage_path", "hashes_json", "thumbnail_url", "size_bytes", "modified_at", "container", "content_class", "status", "scan_state", "missing_since", "deleted_at").
		Updates(updates).Error
}

func filenameReleaseHintsPayload(hints filenameReleaseHints) map[string]any {
	payload := make(map[string]any)
	if strings.TrimSpace(hints.Quality) != "" {
		payload["quality"] = strings.TrimSpace(hints.Quality)
	}
	if len(hints.SourceTags) > 0 {
		payload["source_tags"] = append([]string(nil), hints.SourceTags...)
	}
	if strings.TrimSpace(hints.Codec) != "" {
		payload["codec"] = strings.TrimSpace(hints.Codec)
	}
	if strings.TrimSpace(hints.Audio) != "" {
		payload["audio"] = strings.TrimSpace(hints.Audio)
	}
	if strings.TrimSpace(hints.Subtitle) != "" {
		payload["subtitle"] = strings.TrimSpace(hints.Subtitle)
	}
	if strings.TrimSpace(hints.HDR) != "" {
		payload["hdr"] = strings.TrimSpace(hints.HDR)
	}
	if strings.TrimSpace(hints.Edition) != "" {
		payload["edition"] = strings.TrimSpace(hints.Edition)
	}
	if strings.TrimSpace(hints.ReleaseGroup) != "" {
		payload["release_group"] = strings.TrimSpace(hints.ReleaseGroup)
	}
	return payload
}

func filenameEvidencePayload(evidence []filenameEvidenceSummary) []map[string]any {
	items := make([]map[string]any, 0, len(evidence))
	for _, summary := range evidence {
		item := map[string]any{
			"kind":   strings.TrimSpace(summary.Kind),
			"source": strings.TrimSpace(summary.Source),
			"value":  strings.TrimSpace(summary.Value),
		}
		if strings.TrimSpace(summary.Reason) != "" {
			item["reason"] = strings.TrimSpace(summary.Reason)
		}
		if item["kind"] == "" || item["source"] == "" || item["value"] == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func resolverDecisionEvidencePayload(decisions []scanDecision) []map[string]any {
	items := make([]map[string]any, 0, len(decisions))
	for _, decision := range decisions {
		item := map[string]any{
			"type":        strings.TrimSpace(decision.Type),
			"target_kind": strings.TrimSpace(decision.TargetKind),
			"target_key":  strings.TrimSpace(decision.TargetKey),
			"reason":      strings.TrimSpace(decision.Reason),
		}
		if strings.TrimSpace(decision.Role) != "" {
			item["role"] = strings.TrimSpace(decision.Role)
		}
		if strings.TrimSpace(decision.CandidateType) != "" {
			item["candidate_type"] = strings.TrimSpace(decision.CandidateType)
		}
		if strings.TrimSpace(decision.Status) != "" {
			item["status"] = strings.TrimSpace(decision.Status)
		}
		if decision.Confidence != nil {
			item["confidence"] = *decision.Confidence
		}
		if alternatives := resolverDecisionAlternativesPayload(decision.Alternatives); len(alternatives) > 0 {
			item["alternatives"] = alternatives
		}
		if evidence := resolverDecisionEvidenceItemsPayload(decision.Evidence); len(evidence) > 0 {
			item["evidence"] = evidence
		}
		if len(decision.EvidenceRefs) > 0 {
			item["evidence_refs"] = append([]string(nil), decision.EvidenceRefs...)
		}
		if len(decision.Warnings) > 0 {
			item["warnings"] = append([]string(nil), decision.Warnings...)
		}
		if !decision.CreatedAt.IsZero() {
			item["created_at"] = decision.CreatedAt.UTC().Format(time.RFC3339Nano)
		}
		items = append(items, item)
	}
	return items
}

func resolverDecisionAlternativesPayload(alternatives []scanDecisionAlternative) []map[string]any {
	items := make([]map[string]any, 0, len(alternatives))
	for _, alternative := range alternatives {
		item := map[string]any{
			"type":        strings.TrimSpace(alternative.Type),
			"role":        strings.TrimSpace(alternative.Role),
			"target_kind": strings.TrimSpace(alternative.TargetKind),
			"target_key":  strings.TrimSpace(alternative.TargetKey),
			"reason":      strings.TrimSpace(alternative.Reason),
		}
		if alternative.Confidence != nil {
			item["confidence"] = *alternative.Confidence
		}
		items = append(items, item)
	}
	return items
}

func resolverDecisionEvidenceItemsPayload(evidence []scanDecisionEvidence) []map[string]any {
	items := make([]map[string]any, 0, len(evidence))
	for _, itemEvidence := range evidence {
		item := map[string]any{
			"kind":   strings.TrimSpace(itemEvidence.Kind),
			"source": strings.TrimSpace(itemEvidence.Source),
			"value":  strings.TrimSpace(itemEvidence.Value),
		}
		if itemEvidence.Weight != nil {
			item["weight"] = *itemEvidence.Weight
		}
		items = append(items, item)
	}
	return items
}

func canonicalSeriesPath(seriesTitle string) string {
	cleaned := strings.TrimSpace(cleanTitle(seriesTitle))
	if cleaned == "" {
		return "series"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(cleaned) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		return "series"
	}
	return normalized
}

func applyScopedPathFilter(query *gorm.DB, column string, rootPath string) *gorm.DB {
	scope := strings.TrimRight(strings.TrimSpace(rootPath), "/")
	if scope == "" {
		return query
	}
	return query.Where(column+" = ? OR "+column+" LIKE ?", scope, scope+"/%")
}

func applySkippedDirectoryFilter(query *gorm.DB, column string, skippedDirectories []string) *gorm.DB {
	for _, skipped := range skippedDirectories {
		scope := strings.TrimRight(strings.TrimSpace(skipped), "/")
		if scope == "" {
			continue
		}
		query = query.Where(column+" <> ? AND "+column+" NOT LIKE ?", scope, scope+"/%")
	}
	return query
}
