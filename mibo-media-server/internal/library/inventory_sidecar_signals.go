package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/recognition"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type inventorySidecarSignalInput struct {
	File              database.InventoryFile
	SidecarPath       string
	Extension         string
	AssociationSource string
	Hint              parsedSidecarMetadata
	ParseStatus       string
	ParseError        string
}

func inventorySidecarSignalFingerprint(file database.InventoryFile, sidecarPath string) string {
	modified := ""
	if file.ModifiedAt != nil {
		modified = file.ModifiedAt.UTC().Format(time.RFC3339Nano)
	}
	parts := []string{
		"provider=" + strings.TrimSpace(file.StorageProvider),
		"video=" + strings.TrimSpace(file.StoragePath),
		"sidecar=" + strings.TrimSpace(sidecarPath),
		"size=" + stringInt64(file.SizeBytes),
		"modified=" + modified,
		"stable=" + strings.TrimSpace(file.StableIdentityKey),
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func loadInventorySidecarSignals(ctx context.Context, db *gorm.DB, libraryID uint, storageProvider string, files []database.InventoryFile) (map[uint][]database.InventorySidecarSignal, error) {
	if db == nil || libraryID == 0 || len(files) == 0 {
		return nil, nil
	}
	fileIDs := make([]uint, 0, len(files))
	for _, file := range files {
		if file.ID != 0 {
			fileIDs = append(fileIDs, file.ID)
		}
	}
	if len(fileIDs) == 0 {
		return nil, nil
	}
	var rows []database.InventorySidecarSignal
	if err := db.WithContext(ctx).
		Where("library_id = ? AND storage_provider = ? AND inventory_file_id IN ? AND invalidated_at IS NULL", libraryID, strings.TrimSpace(storageProvider), fileIDs).
		Order("id asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	grouped := make(map[uint][]database.InventorySidecarSignal)
	for _, row := range rows {
		if row.InventoryFileID == nil || *row.InventoryFileID == 0 {
			continue
		}
		grouped[*row.InventoryFileID] = append(grouped[*row.InventoryFileID], row)
	}
	if len(grouped) == 0 {
		return nil, nil
	}
	return grouped, nil
}

func saveInventorySidecarSignals(ctx context.Context, db *gorm.DB, libraryID uint, storageProvider string, inputs []inventorySidecarSignalInput) error {
	if db == nil || libraryID == 0 || len(inputs) == 0 {
		return nil
	}
	now := time.Now().UTC()
	rows := make([]database.InventorySidecarSignal, 0, len(inputs))
	for _, input := range inputs {
		if input.File.ID == 0 || strings.TrimSpace(input.File.StoragePath) == "" || strings.TrimSpace(input.SidecarPath) == "" {
			continue
		}
		inventoryFileID := input.File.ID
		row := database.InventorySidecarSignal{
			InventoryFileID:   &inventoryFileID,
			LibraryID:         libraryID,
			StorageProvider:   strings.TrimSpace(storageProvider),
			VideoStoragePath:  strings.TrimSpace(input.File.StoragePath),
			SidecarPath:       strings.TrimSpace(input.SidecarPath),
			ParentPath:        path.Dir(strings.TrimSpace(input.SidecarPath)),
			Extension:         strings.TrimSpace(input.Extension),
			AssociationSource: strings.TrimSpace(input.AssociationSource),
			ParseStatus:       strings.TrimSpace(input.ParseStatus),
			MediaType:         strings.TrimSpace(input.Hint.MediaType),
			Title:             strings.TrimSpace(input.Hint.Title),
			OriginalTitle:     strings.TrimSpace(input.Hint.OriginalTitle),
			Year:              input.Hint.Year,
			SeriesTitle:       strings.TrimSpace(input.Hint.SeriesTitle),
			SeasonNumber:      input.Hint.SeasonNumber,
			EpisodeNumber:     input.Hint.EpisodeNumber,
			ExternalIDsJSON:   mustJSON(input.Hint.ExternalIDs),
			FieldsJSON:        mustJSON(input.Hint.Fields),
			ParseError:        strings.TrimSpace(input.ParseError),
			FileFingerprint:   inventorySidecarSignalFingerprint(input.File, input.SidecarPath),
			LastObservedAt:    now,
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "storage_provider"}, {Name: "video_storage_path"}, {Name: "sidecar_path"}},
		DoUpdates: clause.AssignmentColumns([]string{"inventory_file_id", "library_id", "parent_path", "extension", "association_source", "parse_status", "media_type", "title", "original_title", "year", "series_title", "season_number", "episode_number", "external_ids_json", "fields_json", "parse_error", "file_fingerprint", "invalidated_at", "last_observed_at", "updated_at"}),
	}).CreateInBatches(&rows, sqliteNarrowWriteBatchSize).Error
}

func sidecarHintsFromSignals(rows []database.InventorySidecarSignal) []recognition.SidecarHint {
	if len(rows) == 0 {
		return nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].SidecarPath < rows[j].SidecarPath })
	hints := make([]recognition.SidecarHint, 0, len(rows))
	for _, row := range rows {
		hint := recognition.SidecarHint{
			Path:          strings.TrimSpace(row.SidecarPath),
			Extension:     strings.TrimSpace(row.Extension),
			ParseStatus:   strings.TrimSpace(row.ParseStatus),
			MediaType:     strings.TrimSpace(row.MediaType),
			Title:         strings.TrimSpace(row.Title),
			OriginalTitle: strings.TrimSpace(row.OriginalTitle),
			Year:          row.Year,
			SeriesTitle:   strings.TrimSpace(row.SeriesTitle),
			SeasonNumber:  row.SeasonNumber,
			EpisodeNumber: row.EpisodeNumber,
			ExternalIDs:   map[string]string{},
			Fields:        map[string]any{},
		}
		_ = json.Unmarshal([]byte(strings.TrimSpace(row.ExternalIDsJSON)), &hint.ExternalIDs)
		_ = json.Unmarshal([]byte(strings.TrimSpace(row.FieldsJSON)), &hint.Fields)
		hints = append(hints, hint)
	}
	return hints
}

func stringInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}
