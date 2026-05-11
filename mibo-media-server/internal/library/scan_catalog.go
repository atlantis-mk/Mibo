package library

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
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

func buildCatalogScanEvidencePayload(artifact catalogScanArtifact, episodeNumbers []int) string {
	payload := map[string]any{
		"storage_path":        strings.TrimSpace(artifact.SourcePath),
		"stable_identity_key": strings.TrimSpace(artifact.StableIdentityKey),
		"provider_name":       strings.TrimSpace(artifact.ProviderName),
		"hashes_json":         strings.TrimSpace(artifact.HashesJSON),
		"detected_title":      strings.TrimSpace(artifact.Title),
	}
	if strings.TrimSpace(artifact.ObjectType) != "" {
		payload["object_type"] = strings.TrimSpace(artifact.ObjectType)
	}
	if len(artifact.ProviderMeta) > 0 {
		payload["provider_metadata"] = artifact.ProviderMeta
	}
	if strings.TrimSpace(artifact.NormalizationVersion) != "" {
		payload["normalization_version"] = strings.TrimSpace(artifact.NormalizationVersion)
	}
	if len(artifact.RemovedTokens) > 0 {
		payload["removed_tokens"] = artifact.RemovedTokens
	}
	if hints := filenameReleaseHintsPayload(artifact.FilenameSignals.ReleaseHints); len(hints) > 0 {
		payload["filename_release_hints"] = hints
	}
	if evidence := filenameEvidencePayload(artifact.FilenameSignals.Evidence); len(evidence) > 0 {
		payload["filename_signal_evidence"] = evidence
	}
	if tags := uniqueCatalogScanTags(artifact.Tags); len(tags) > 0 {
		payload["hashtag_tags"] = tags
	}
	if len(artifact.SubtitleSidecars) > 0 {
		payload["subtitle_sidecars"] = sidecarEvidencePayload(artifact.SubtitleSidecars)
	}
	if len(artifact.MetadataSidecars) > 0 {
		payload["metadata_sidecars"] = metadataSidecarEvidencePayload(artifact.MetadataSidecars)
	}
	if len(artifact.ExternalIDs) > 0 {
		payload["external_ids"] = externalIDEvidencePayload(artifact.ExternalIDs)
	}
	if len(artifact.Decisions) > 0 {
		payload["resolver_decisions"] = resolverDecisionEvidencePayload(artifact.Decisions)
	}
	if len(artifact.ContentShapeProfile) > 0 {
		payload["content_shape_profile"] = artifact.ContentShapeProfile
	}
	if len(artifact.ContentShapePlan) > 0 {
		payload["content_shape_plan"] = artifact.ContentShapePlan
	}
	if len(artifact.ContentShapeAssignment) > 0 {
		payload["content_shape_assignment"] = artifact.ContentShapeAssignment
	}
	if strings.TrimSpace(artifact.SeriesTitle) != "" {
		payload["series_title"] = strings.TrimSpace(artifact.SeriesTitle)
	}
	if artifact.SeasonNumber != nil {
		payload["season_number"] = *artifact.SeasonNumber
	}
	if len(episodeNumbers) > 0 {
		payload["episode_numbers"] = episodeNumbers
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
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

func externalIDEvidencePayload(externalIDs []catalogScanExternalID) []map[string]any {
	items := make([]map[string]any, 0, len(externalIDs))
	for _, externalID := range externalIDs {
		items = append(items, map[string]any{
			"provider":      strings.TrimSpace(externalID.Provider),
			"provider_type": strings.TrimSpace(externalID.ProviderType),
			"external_id":   strings.TrimSpace(externalID.ExternalID),
		})
	}
	return items
}

func sidecarEvidencePayload(sidecars []catalogScanSidecar) []map[string]any {
	items := make([]map[string]any, 0, len(sidecars))
	for _, sidecar := range sidecars {
		item := map[string]any{
			"path":               strings.TrimSpace(sidecar.Path),
			"extension":          strings.TrimSpace(sidecar.Extension),
			"association_source": strings.TrimSpace(sidecar.AssociationSource),
		}
		items = append(items, item)
	}
	return items
}

func metadataSidecarEvidencePayload(sidecars []catalogScanMetadataSidecar) []map[string]any {
	items := make([]map[string]any, 0, len(sidecars))
	for _, sidecar := range sidecars {
		item := map[string]any{
			"path":               strings.TrimSpace(sidecar.Path),
			"extension":          strings.TrimSpace(sidecar.Extension),
			"association_source": strings.TrimSpace(sidecar.AssociationSource),
			"parse_status":       strings.TrimSpace(sidecar.ParseStatus),
		}
		if hints := metadataHintsPayload(sidecar.Hints); len(hints) > 0 {
			item["hints"] = hints
		}
		if len(sidecar.ExternalIDs) > 0 {
			item["external_ids"] = sidecar.ExternalIDs
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

func metadataHintsPayload(hints catalogScanMetadataHints) map[string]any {
	payload := make(map[string]any)
	if strings.TrimSpace(hints.Title) != "" {
		payload["title"] = strings.TrimSpace(hints.Title)
	}
	if strings.TrimSpace(hints.OriginalTitle) != "" {
		payload["original_title"] = strings.TrimSpace(hints.OriginalTitle)
	}
	if hints.Year != nil {
		payload["year"] = *hints.Year
	}
	if strings.TrimSpace(hints.MediaType) != "" {
		payload["media_type"] = strings.TrimSpace(hints.MediaType)
	}
	if strings.TrimSpace(hints.SeriesTitle) != "" {
		payload["series_title"] = strings.TrimSpace(hints.SeriesTitle)
	}
	if hints.SeasonNumber != nil {
		payload["season_number"] = *hints.SeasonNumber
	}
	if hints.EpisodeNumber != nil {
		payload["episode_number"] = *hints.EpisodeNumber
	}
	return payload
}

func defaultCatalogTitle(title string, fallbackPath string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	base := strings.TrimSuffix(path.Base(strings.TrimSpace(fallbackPath)), path.Ext(strings.TrimSpace(fallbackPath)))
	if strings.TrimSpace(base) != "" && base != "." && base != "/" {
		return cleanTitle(base)
	}
	return strings.TrimSpace(fallbackPath)
}

func defaultCatalogSortKey(sortKey string, fallback string) string {
	if strings.TrimSpace(sortKey) != "" {
		return strings.TrimSpace(sortKey)
	}
	return defaultCatalogTitle(fallback, fallback)
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

func canonicalEpisodeItemPath(seasonPath string, episodeNumber int) string {
	return fmt.Sprintf("%s/episode-%04d", seasonPath, episodeNumber)
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
