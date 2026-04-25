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
)

const JobKindLegacyBackfill = "catalog_backfill_legacy"

const (
	LegacyBackfillScopeAll     = "all"
	LegacyBackfillScopeLibrary = "library"
)

const (
	LegacyBackfillStatusQueued    = "queued"
	LegacyBackfillStatusRunning   = "running"
	LegacyBackfillStatusCompleted = "completed"
	LegacyBackfillStatusFailed    = "failed"
)

const (
	LegacyBackfillEntryTypeSuccess                   = "success"
	LegacyBackfillEntryTypeSkipped                   = "skipped"
	LegacyBackfillEntryTypeConflict                  = "conflict"
	LegacyBackfillEntryTypeOrphanFile                = "orphan_file"
	LegacyBackfillEntryTypeDuplicateEpisodeCandidate = "duplicate_episode_candidate"
)

type LegacyBackfillPayload struct {
	RunID             uint                `json:"run_id"`
	Scope             LegacyBackfillScope `json:"scope"`
	TriggeredByUserID uint                `json:"triggered_by_user_id"`
}

type CreateLegacyBackfillRunInput struct {
	Scope             LegacyBackfillScope `json:"scope"`
	TriggeredByUserID uint                `json:"triggered_by_user_id"`
}

type LegacyBackfillScope struct {
	Kind      string `json:"kind"`
	LibraryID *uint  `json:"library_id,omitempty"`
}

type LegacyBackfillRun struct {
	ID                             uint                  `json:"id"`
	Scope                          LegacyBackfillScope   `json:"scope"`
	Status                         string                `json:"status"`
	TriggeredByUserID              uint                  `json:"triggered_by_user_id"`
	FatalError                     string                `json:"fatal_error,omitempty"`
	SuccessCount                   int                   `json:"success_count"`
	SkippedCount                   int                   `json:"skipped_count"`
	ConflictCount                  int                   `json:"conflict_count"`
	OrphanFileCount                int                   `json:"orphan_file_count"`
	DuplicateEpisodeCandidateCount int                   `json:"duplicate_episode_candidate_count"`
	StartedAt                      *time.Time            `json:"started_at,omitempty"`
	FinishedAt                     *time.Time            `json:"finished_at,omitempty"`
	CreatedAt                      time.Time             `json:"created_at"`
	UpdatedAt                      time.Time             `json:"updated_at"`
	Entries                        []LegacyBackfillEntry `json:"entries,omitempty"`
}

type LegacyBackfillEntry struct {
	ID                uint            `json:"id"`
	RunID             uint            `json:"run_id,omitempty"`
	EntryType         string          `json:"entry_type"`
	LibraryID         *uint           `json:"library_id,omitempty"`
	LegacyMediaItemID *uint           `json:"legacy_media_item_id,omitempty"`
	LegacyMediaFileID *uint           `json:"legacy_media_file_id,omitempty"`
	CatalogItemID     *uint           `json:"catalog_item_id,omitempty"`
	AssetID           *uint           `json:"asset_id,omitempty"`
	InventoryFileID   *uint           `json:"inventory_file_id,omitempty"`
	StoragePath       string          `json:"storage_path,omitempty"`
	Title             string          `json:"title,omitempty"`
	Message           string          `json:"message,omitempty"`
	Details           json.RawMessage `json:"details,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (s *Service) createLegacyBackfillRun(ctx context.Context, scope LegacyBackfillScope, triggeredByUserID uint) (database.CatalogMigrationRun, error) {
	if triggeredByUserID == 0 {
		return database.CatalogMigrationRun{}, errors.New("triggered by user id is required")
	}

	normalizedScope, err := normalizeLegacyBackfillScope(scope)
	if err != nil {
		return database.CatalogMigrationRun{}, err
	}

	run := database.CatalogMigrationRun{
		ScopeKind:         normalizedScope.Kind,
		LibraryID:         normalizedScope.LibraryID,
		Status:            LegacyBackfillStatusQueued,
		TriggeredByUserID: triggeredByUserID,
	}

	if err := s.db.WithContext(ctx).Create(&run).Error; err != nil {
		return database.CatalogMigrationRun{}, err
	}

	return run, nil
}

func (s *Service) CreateLegacyBackfillRun(ctx context.Context, input CreateLegacyBackfillRunInput) (LegacyBackfillRun, error) {
	run, err := s.createLegacyBackfillRun(ctx, input.Scope, input.TriggeredByUserID)
	if err != nil {
		return LegacyBackfillRun{}, err
	}
	return legacyBackfillRunFromModel(run), nil
}

func (s *Service) ListLegacyBackfillRuns(ctx context.Context) ([]LegacyBackfillRun, error) {
	var runs []database.CatalogMigrationRun
	if err := s.db.WithContext(ctx).
		Order("created_at desc").
		Order("id desc").
		Find(&runs).Error; err != nil {
		return nil, err
	}

	result := make([]LegacyBackfillRun, 0, len(runs))
	for _, run := range runs {
		result = append(result, legacyBackfillRunFromModel(run))
	}
	return result, nil
}

func (s *Service) GetLegacyBackfillRun(ctx context.Context, runID uint) (LegacyBackfillRun, error) {
	if runID == 0 {
		return LegacyBackfillRun{}, errors.New("run id is required")
	}

	var run database.CatalogMigrationRun
	if err := s.db.WithContext(ctx).First(&run, runID).Error; err != nil {
		return LegacyBackfillRun{}, err
	}

	var entries []database.CatalogMigrationEntry
	if err := s.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("entry_type asc").
		Order("library_id asc").
		Order("legacy_media_item_id asc").
		Order("legacy_media_file_id asc").
		Order("id asc").
		Find(&entries).Error; err != nil {
		return LegacyBackfillRun{}, err
	}

	report := legacyBackfillRunFromModel(run)
	report.Entries = make([]LegacyBackfillEntry, 0, len(entries))
	for _, entry := range entries {
		report.Entries = append(report.Entries, legacyBackfillEntryFromModel(entry))
	}
	return report, nil
}

func (s *Service) recordLegacyBackfillEntry(ctx context.Context, runID uint, entry LegacyBackfillEntry) (database.CatalogMigrationEntry, error) {
	if runID == 0 {
		return database.CatalogMigrationEntry{}, errors.New("run id is required")
	}
	entryType, err := normalizeLegacyBackfillEntryType(entry.EntryType)
	if err != nil {
		return database.CatalogMigrationEntry{}, err
	}
	if len(entry.Details) > 0 && !json.Valid(entry.Details) {
		return database.CatalogMigrationEntry{}, errors.New("details must be valid json")
	}

	model := database.CatalogMigrationEntry{
		RunID:             runID,
		EntryType:         entryType,
		LibraryID:         entry.LibraryID,
		LegacyMediaItemID: entry.LegacyMediaItemID,
		LegacyMediaFileID: entry.LegacyMediaFileID,
		CatalogItemID:     entry.CatalogItemID,
		AssetID:           entry.AssetID,
		InventoryFileID:   entry.InventoryFileID,
		StoragePath:       strings.TrimSpace(entry.StoragePath),
		Title:             strings.TrimSpace(entry.Title),
		Message:           strings.TrimSpace(entry.Message),
		DetailsJSON:       strings.TrimSpace(string(entry.Details)),
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run database.CatalogMigrationRun
		if err := tx.First(&run, runID).Error; err != nil {
			return err
		}
		return tx.Create(&model).Error
	}); err != nil {
		return database.CatalogMigrationEntry{}, err
	}

	return model, nil
}

func (s *Service) finalizeLegacyBackfillRun(ctx context.Context, runID uint, status string, fatalError string) (database.CatalogMigrationRun, error) {
	if runID == 0 {
		return database.CatalogMigrationRun{}, errors.New("run id is required")
	}
	normalizedStatus, err := normalizeLegacyBackfillStatus(status)
	if err != nil {
		return database.CatalogMigrationRun{}, err
	}

	now := time.Now().UTC()
	var run database.CatalogMigrationRun
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&run, runID).Error; err != nil {
			return err
		}

		counts, err := loadLegacyBackfillCounts(ctx, tx, runID)
		if err != nil {
			return err
		}

		updates := map[string]any{
			"status":                            normalizedStatus,
			"fatal_error":                       strings.TrimSpace(fatalError),
			"success_count":                     counts[LegacyBackfillEntryTypeSuccess],
			"skipped_count":                     counts[LegacyBackfillEntryTypeSkipped],
			"conflict_count":                    counts[LegacyBackfillEntryTypeConflict],
			"orphan_file_count":                 counts[LegacyBackfillEntryTypeOrphanFile],
			"duplicate_episode_candidate_count": counts[LegacyBackfillEntryTypeDuplicateEpisodeCandidate],
			"finished_at":                       now,
		}
		if run.StartedAt == nil {
			updates["started_at"] = now
		}

		if err := tx.Model(&database.CatalogMigrationRun{}).Where("id = ?", runID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&run, runID).Error
	})
	if err != nil {
		return database.CatalogMigrationRun{}, err
	}

	return run, nil
}

func loadLegacyBackfillCounts(ctx context.Context, db *gorm.DB, runID uint) (map[string]int, error) {
	rows := make([]struct {
		EntryType string
		Count     int
	}, 0)
	if err := db.WithContext(ctx).
		Model(&database.CatalogMigrationEntry{}).
		Select("entry_type, count(*) as count").
		Where("run_id = ?", runID).
		Group("entry_type").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := map[string]int{
		LegacyBackfillEntryTypeSuccess:                   0,
		LegacyBackfillEntryTypeSkipped:                   0,
		LegacyBackfillEntryTypeConflict:                  0,
		LegacyBackfillEntryTypeOrphanFile:                0,
		LegacyBackfillEntryTypeDuplicateEpisodeCandidate: 0,
	}
	for _, row := range rows {
		counts[row.EntryType] = row.Count
	}
	return counts, nil
}

func normalizeLegacyBackfillScope(scope LegacyBackfillScope) (LegacyBackfillScope, error) {
	normalized := LegacyBackfillScope{Kind: strings.TrimSpace(scope.Kind), LibraryID: scope.LibraryID}
	switch normalized.Kind {
	case LegacyBackfillScopeAll:
		normalized.LibraryID = nil
		return normalized, nil
	case LegacyBackfillScopeLibrary:
		if normalized.LibraryID == nil || *normalized.LibraryID == 0 {
			return LegacyBackfillScope{}, errors.New("library scope requires library id")
		}
		return normalized, nil
	default:
		return LegacyBackfillScope{}, fmt.Errorf("unsupported backfill scope %q", scope.Kind)
	}
}

func normalizeLegacyBackfillStatus(status string) (string, error) {
	switch strings.TrimSpace(status) {
	case LegacyBackfillStatusQueued, LegacyBackfillStatusRunning, LegacyBackfillStatusCompleted, LegacyBackfillStatusFailed:
		return strings.TrimSpace(status), nil
	default:
		return "", fmt.Errorf("unsupported backfill status %q", status)
	}
}

func normalizeLegacyBackfillEntryType(entryType string) (string, error) {
	switch strings.TrimSpace(entryType) {
	case LegacyBackfillEntryTypeSuccess,
		LegacyBackfillEntryTypeSkipped,
		LegacyBackfillEntryTypeConflict,
		LegacyBackfillEntryTypeOrphanFile,
		LegacyBackfillEntryTypeDuplicateEpisodeCandidate:
		return strings.TrimSpace(entryType), nil
	default:
		return "", fmt.Errorf("unsupported backfill entry type %q", entryType)
	}
}

func legacyBackfillRunFromModel(run database.CatalogMigrationRun) LegacyBackfillRun {
	return LegacyBackfillRun{
		ID:     run.ID,
		Scope:  LegacyBackfillScope{Kind: run.ScopeKind, LibraryID: run.LibraryID},
		Status: run.Status,

		TriggeredByUserID:              run.TriggeredByUserID,
		FatalError:                     run.FatalError,
		SuccessCount:                   run.SuccessCount,
		SkippedCount:                   run.SkippedCount,
		ConflictCount:                  run.ConflictCount,
		OrphanFileCount:                run.OrphanFileCount,
		DuplicateEpisodeCandidateCount: run.DuplicateEpisodeCandidateCount,
		StartedAt:                      run.StartedAt,
		FinishedAt:                     run.FinishedAt,
		CreatedAt:                      run.CreatedAt,
		UpdatedAt:                      run.UpdatedAt,
	}
}

func legacyBackfillEntryFromModel(entry database.CatalogMigrationEntry) LegacyBackfillEntry {
	result := LegacyBackfillEntry{
		ID:                entry.ID,
		RunID:             entry.RunID,
		EntryType:         entry.EntryType,
		LibraryID:         entry.LibraryID,
		LegacyMediaItemID: entry.LegacyMediaItemID,
		LegacyMediaFileID: entry.LegacyMediaFileID,
		CatalogItemID:     entry.CatalogItemID,
		AssetID:           entry.AssetID,
		InventoryFileID:   entry.InventoryFileID,
		StoragePath:       entry.StoragePath,
		Title:             entry.Title,
		Message:           entry.Message,
		CreatedAt:         entry.CreatedAt,
		UpdatedAt:         entry.UpdatedAt,
	}
	if details := strings.TrimSpace(entry.DetailsJSON); details != "" {
		result.Details = json.RawMessage(details)
	}
	return result
}
