package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type inventoryFileSignalScope struct {
	LibraryID         uint
	StorageProvider   string
	ClassifierVersion string
}

type inventoryFileSignalInput struct {
	File  database.InventoryFile
	Model filenameSignalModel
}

func inventoryFileSignalFingerprint(file database.InventoryFile, classifierVersion string) string {
	modified := ""
	if file.ModifiedAt != nil {
		modified = file.ModifiedAt.UTC().Format(time.RFC3339Nano)
	}
	parts := []string{
		"provider=" + strings.TrimSpace(file.StorageProvider),
		"path=" + strings.TrimSpace(file.StoragePath),
		"basename=" + strings.TrimSpace(path.Base(file.StoragePath)),
		fmt.Sprintf("size=%d", file.SizeBytes),
		"modified=" + modified,
		"stable=" + strings.TrimSpace(file.StableIdentityKey),
		"classifier=" + strings.TrimSpace(classifierVersion),
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func loadReusableInventoryFileSignals(ctx context.Context, db *gorm.DB, scope inventoryFileSignalScope, files []database.InventoryFile) (map[string]filenameSignalModel, map[string]database.InventoryFileSignal, error) {
	paths := make([]string, 0, len(files))
	fingerprints := make(map[string]string, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		storagePath := strings.TrimSpace(file.StoragePath)
		if storagePath == "" {
			continue
		}
		if _, ok := seen[storagePath]; ok {
			continue
		}
		seen[storagePath] = struct{}{}
		paths = append(paths, storagePath)
		fingerprints[storagePath] = inventoryFileSignalFingerprint(file, scope.ClassifierVersion)
	}
	if len(paths) == 0 {
		return nil, nil, nil
	}
	models := make(map[string]filenameSignalModel, len(paths))
	rowsByPath := make(map[string]database.InventoryFileSignal, len(paths))
	for _, batch := range chunkStrings(paths, sqliteVariableChunkSize) {
		var rows []database.InventoryFileSignal
		if err := db.WithContext(ctx).
			Where("library_id = ? AND storage_provider = ? AND classifier_version = ?", scope.LibraryID, strings.TrimSpace(scope.StorageProvider), strings.TrimSpace(scope.ClassifierVersion)).
			Where("invalidated_at IS NULL AND storage_path IN ?", batch).
			Find(&rows).Error; err != nil {
			return nil, nil, err
		}
		for _, row := range rows {
			storagePath := strings.TrimSpace(row.StoragePath)
			if storagePath == "" || strings.TrimSpace(row.FileFingerprint) != fingerprints[storagePath] {
				continue
			}
			model, ok := filenameSignalModelFromInventoryFileSignal(row)
			if !ok {
				continue
			}
			models[storagePath] = model
			rowsByPath[storagePath] = row
		}
	}
	return models, rowsByPath, nil
}

func saveInventoryFileSignals(ctx context.Context, db *gorm.DB, scope inventoryFileSignalScope, inputs []inventoryFileSignalInput) error {
	if len(inputs) == 0 {
		return nil
	}
	now := time.Now().UTC()
	rows := make([]database.InventoryFileSignal, 0, len(inputs))
	for _, input := range inputs {
		file := input.File
		storagePath := strings.TrimSpace(file.StoragePath)
		if storagePath == "" {
			continue
		}
		inventoryFileID := file.ID
		row := database.InventoryFileSignal{
			InventoryFileID:    &inventoryFileID,
			LibraryID:          scope.LibraryID,
			StorageProvider:    strings.TrimSpace(scope.StorageProvider),
			StoragePath:        storagePath,
			ParentPath:         path.Dir(storagePath),
			Basename:           path.Base(storagePath),
			Extension:          strings.ToLower(path.Ext(storagePath)),
			ClassifierVersion:  strings.TrimSpace(scope.ClassifierVersion),
			FileFingerprint:    inventoryFileSignalFingerprint(file, scope.ClassifierVersion),
			TitleCandidate:     strings.TrimSpace(firstNonEmptyString(firstScanTitle(input.Model.VideoSignal.TitleCandidates), input.Model.Identity.TitleCandidate)),
			Year:               firstNonNilInt(input.Model.VideoSignal.Year, input.Model.Identity.Year),
			SeasonNumber:       firstNonNilInt(input.Model.VideoSignal.Season, input.Model.Identity.SeasonNumber),
			EpisodeNumber:      firstNonNilInt(input.Model.VideoSignal.Episode, input.Model.Identity.EpisodeNumber),
			LeadingNumber:      firstNonNilInt(input.Model.VideoSignal.LeadingNumber, input.Model.Identity.LeadingNumber),
			EpisodeSource:      strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.EpisodeSource, input.Model.Identity.EpisodeSource)),
			Role:               strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.Role, input.Model.RoleHints.Role)),
			IsExtra:            input.Model.VideoSignal.IsExtra || input.Model.RoleHints.IsExtra,
			Quality:            strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.Quality, input.Model.ReleaseHints.Quality)),
			Codec:              strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.Codec, input.Model.ReleaseHints.Codec)),
			Audio:              strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.Audio, input.Model.ReleaseHints.Audio)),
			Subtitle:           strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.Subtitle, input.Model.ReleaseHints.Subtitle)),
			HDR:                strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.HDR, input.Model.ReleaseHints.HDR)),
			Edition:            strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.Edition, input.Model.ReleaseHints.Edition)),
			ReleaseGroup:       strings.TrimSpace(firstNonEmptyString(input.Model.VideoSignal.ReleaseGroup, input.Model.ReleaseHints.ReleaseGroup)),
			SourceTagsJSON:     mustJSON(firstNonEmptyStrings(input.Model.VideoSignal.SourceTags, input.Model.ReleaseHints.SourceTags)),
			EpisodeNumbersJSON: mustJSON(firstNonEmptyInts(input.Model.VideoSignal.EpisodeNumbers, input.Model.Identity.EpisodeNumbers)),
			TitleTokensJSON:    mustJSON(input.Model.TitleTokens),
			ModelJSON:          mustJSON(input.Model),
			EvidenceJSON:       mustJSON(input.Model.Evidence),
			InvalidatedAt:      nil,
			LastObservedAt:     now,
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "storage_provider"}, {Name: "storage_path"}, {Name: "classifier_version"}},
		DoUpdates: clause.AssignmentColumns([]string{"inventory_file_id", "library_id", "parent_path", "basename", "extension", "file_fingerprint", "title_candidate", "year", "season_number", "episode_number", "leading_number", "episode_source", "role", "is_extra", "quality", "codec", "audio", "subtitle", "hdr", "edition", "release_group", "source_tags_json", "episode_numbers_json", "title_tokens_json", "model_json", "evidence_json", "invalidated_at", "last_observed_at", "updated_at"}),
	}).CreateInBatches(&rows, sqliteNarrowWriteBatchSize).Error
}

func filenameSignalModelFromInventoryFileSignal(row database.InventoryFileSignal) (filenameSignalModel, bool) {
	var model filenameSignalModel
	if strings.TrimSpace(row.ModelJSON) != "" {
		if err := json.Unmarshal([]byte(row.ModelJSON), &model); err == nil {
			hydrateFilenameSignalRawPathData(strings.TrimSpace(row.StoragePath), &model)
			syncFilenameSignalModel(strings.TrimSpace(row.StoragePath), &model)
			return model, true
		}
	}
	hydrateFilenameSignalRawPathData(strings.TrimSpace(row.StoragePath), &model)
	model.Identity.TitleCandidate = strings.TrimSpace(row.TitleCandidate)
	model.Identity.Year = row.Year
	model.Identity.SeasonNumber = row.SeasonNumber
	model.Identity.EpisodeNumber = row.EpisodeNumber
	model.Identity.LeadingNumber = row.LeadingNumber
	model.Identity.EpisodeSource = strings.TrimSpace(row.EpisodeSource)
	_ = json.Unmarshal([]byte(strings.TrimSpace(row.EpisodeNumbersJSON)), &model.Identity.EpisodeNumbers)
	model.RoleHints.Role = strings.TrimSpace(row.Role)
	model.RoleHints.IsExtra = row.IsExtra
	model.RoleHints.IsMain = !row.IsExtra
	model.RoleHints.IsSample = model.RoleHints.Role == "sample"
	model.RoleHints.IsTrailer = model.RoleHints.Role == "trailer"
	model.ReleaseHints.Quality = strings.TrimSpace(row.Quality)
	model.ReleaseHints.Codec = strings.TrimSpace(row.Codec)
	model.ReleaseHints.Audio = strings.TrimSpace(row.Audio)
	model.ReleaseHints.Subtitle = strings.TrimSpace(row.Subtitle)
	model.ReleaseHints.HDR = strings.TrimSpace(row.HDR)
	model.ReleaseHints.Edition = strings.TrimSpace(row.Edition)
	model.ReleaseHints.ReleaseGroup = strings.TrimSpace(row.ReleaseGroup)
	_ = json.Unmarshal([]byte(strings.TrimSpace(row.SourceTagsJSON)), &model.ReleaseHints.SourceTags)
	_ = json.Unmarshal([]byte(strings.TrimSpace(row.TitleTokensJSON)), &model.TitleTokens)
	_ = json.Unmarshal([]byte(strings.TrimSpace(row.EvidenceJSON)), &model.Evidence)
	syncFilenameSignalModel(strings.TrimSpace(row.StoragePath), &model)
	return model, strings.TrimSpace(model.Identity.TitleCandidate) != "" || model.Identity.Year != nil || model.Identity.EpisodeNumber != nil || model.Identity.LeadingNumber != nil || model.RoleHints.Role != ""
}

func hydrateFilenameSignalRawPathData(storagePath string, model *filenameSignalModel) {
	if model == nil {
		return
	}
	cleanPath := path.Clean(strings.TrimSpace(storagePath))
	if cleanPath == "" || cleanPath == "." {
		return
	}
	if strings.TrimSpace(model.RawPathData.Path) == "" {
		model.RawPathData.Path = cleanPath
	}
	if strings.TrimSpace(model.RawPathData.Directory) == "" {
		model.RawPathData.Directory = path.Dir(cleanPath)
	}
	if strings.TrimSpace(model.RawPathData.Basename) == "" {
		model.RawPathData.Basename = path.Base(cleanPath)
	}
	if strings.TrimSpace(model.RawPathData.Extension) == "" {
		model.RawPathData.Extension = path.Ext(cleanPath)
	}
	if len(model.RawPathData.Segments) == 0 {
		model.RawPathData.Segments = strings.Split(strings.Trim(cleanPath, "/"), "/")
	}
}

func hydrateFilenameTokenCacheFromSignals(cache *filenameTokenProfileCache, models map[string]filenameSignalModel) {
	if cache == nil || len(models) == 0 {
		return
	}
	if cache.profilesByPath == nil {
		cache.profilesByPath = make(map[string]filenameSignalModel, len(models))
	}
	keys := make([]string, 0, len(models))
	for storagePath := range models {
		keys = append(keys, storagePath)
	}
	sort.Strings(keys)
	for _, storagePath := range keys {
		cache.profilesByPath[strings.TrimSpace(storagePath)] = models[storagePath]
	}
}

func firstNonEmptyStrings(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return append([]string(nil), value...)
		}
	}
	return nil
}

func firstNonEmptyInts(values ...[]int) []int {
	for _, value := range values {
		if len(value) > 0 {
			return append([]int(nil), value...)
		}
	}
	return nil
}
