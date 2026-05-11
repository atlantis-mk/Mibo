package library

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

type MarkScanExclusionInput struct {
	InventoryFileID uint
	Reason          string
	UserID          *uint
}

type SetScanExclusionEnabledInput struct {
	ExclusionID uint
	Enabled     bool
	UserID      *uint
}

type FilenameExclusionTargetInput struct {
	InventoryFileID uint
}

type CreateFilenameExclusionRuleInput struct {
	FilenameExclusionTargetInput
	Reason string
	UserID *uint
}

type RestoreFilenameExclusionMatchInput struct {
	RuleID          uint
	InventoryFileID uint
	UserID          *uint
}

type SetFilenameExclusionRuleEnabledInput struct {
	RuleID  uint
	Enabled bool
	UserID  *uint
}

type ListScanExclusionsInput struct {
	LibraryID uint
	Enabled   *bool
}

const scanExclusionSQLBatchSize = 500

type ScanExclusionView struct {
	database.ScanExclusion
	LibraryName string `json:"library_name"`
}

type FilenameExclusionFileView struct {
	ID                uint   `json:"id"`
	LibraryID         uint   `json:"library_id"`
	StoragePath       string `json:"storage_path"`
	StableIdentityKey string `json:"stable_identity_key,omitempty"`
	Status            string `json:"status"`
	Restored          bool   `json:"restored"`
}

type FilenameExclusionRuleView struct {
	database.FilenameExclusionRule
	LibraryName   string                      `json:"library_name"`
	AffectedCount int                         `json:"affected_count"`
	AffectedFiles []FilenameExclusionFileView `json:"affected_files"`
}

type ScanExclusionsView struct {
	ManualExclusions []ScanExclusionView         `json:"manual_exclusions"`
	FilenameRules    []FilenameExclusionRuleView `json:"filename_rules"`
}

type FilenameExclusionPreview struct {
	LibraryID          uint                        `json:"library_id"`
	LibraryName        string                      `json:"library_name"`
	StorageProvider    string                      `json:"storage_provider"`
	NormalizedFilename string                      `json:"normalized_filename"`
	AffectedCount      int                         `json:"affected_count"`
	AffectedFiles      []FilenameExclusionFileView `json:"affected_files"`
}

func (s *Service) ListScanExclusions(ctx context.Context, input ListScanExclusionsInput) ([]ScanExclusionView, error) {
	view, err := s.ListScanExclusionsView(ctx, input)
	if err != nil {
		return nil, err
	}
	return view.ManualExclusions, nil
}

func (s *Service) ListScanExclusionsView(ctx context.Context, input ListScanExclusionsInput) (ScanExclusionsView, error) {
	type scanExclusionRow struct {
		database.ScanExclusion
		LibraryName string `gorm:"column:library_name"`
	}

	query := s.db.WithContext(ctx).
		Model(&database.ScanExclusion{}).
		Select("scan_exclusions.*, libraries.name AS library_name").
		Joins("LEFT JOIN libraries ON libraries.id = scan_exclusions.library_id")
	if input.LibraryID != 0 {
		query = query.Where("scan_exclusions.library_id = ?", input.LibraryID)
	}
	if input.Enabled != nil {
		query = query.Where("scan_exclusions.enabled = ?", *input.Enabled)
	}

	var rows []scanExclusionRow
	if err := query.Order("scan_exclusions.enabled DESC, scan_exclusions.updated_at DESC, scan_exclusions.id DESC").Find(&rows).Error; err != nil {
		return ScanExclusionsView{}, err
	}
	views := make([]ScanExclusionView, 0, len(rows))
	for _, row := range rows {
		views = append(views, ScanExclusionView{ScanExclusion: row.ScanExclusion, LibraryName: row.LibraryName})
	}
	rules, err := s.ListFilenameExclusionRules(ctx, input)
	if err != nil {
		return ScanExclusionsView{}, err
	}
	return ScanExclusionsView{ManualExclusions: views, FilenameRules: rules}, nil
}

func (s *Service) MarkScanExclusion(ctx context.Context, input MarkScanExclusionInput) (database.ScanExclusion, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = ScanExclusionReasonAdvertisement
	}
	if !supportedScanExclusionReason(reason) {
		return database.ScanExclusion{}, errors.New("unsupported scan exclusion reason")
	}

	var result database.ScanExclusion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		file, err := scanExclusionTarget(ctx, tx, input)
		if err != nil {
			return err
		}
		if file.ID == 0 {
			return errors.New("scan exclusion target file not found")
		}

		exclusion, err := upsertScanExclusion(ctx, tx, file, reason, input.UserID)
		if err != nil {
			return err
		}
		result = exclusion

		missingAt := time.Now().UTC()
		if err := tx.WithContext(ctx).Model(&database.InventoryFile{}).Where("id = ?", file.ID).Updates(map[string]any{"status": inventory.FileStatusMissing, "missing_since": gorm.Expr("COALESCE(missing_since, ?)", missingAt), "deleted_at": nil}).Error; err != nil {
			return err
		}
		return catalog.NewService(tx).RefreshLibraryProjectionScope(ctx, file.LibraryID)
	})
	return result, err
}

func (s *Service) PreviewFilenameExclusion(ctx context.Context, input FilenameExclusionTargetInput) (FilenameExclusionPreview, error) {
	file, err := scanExclusionTarget(ctx, s.db, MarkScanExclusionInput{InventoryFileID: input.InventoryFileID})
	if err != nil {
		return FilenameExclusionPreview{}, err
	}
	if file.ID == 0 {
		return FilenameExclusionPreview{}, errors.New("filename exclusion target file not found")
	}
	files, err := s.filenameExclusionAffectedFiles(ctx, s.db, normalizedFilenameFromPath(file.StoragePath), 0)
	if err != nil {
		return FilenameExclusionPreview{}, err
	}
	var libraryName string
	_ = s.db.WithContext(ctx).Model(&database.Library{}).Select("name").Where("id = ?", file.LibraryID).Scan(&libraryName).Error
	return FilenameExclusionPreview{LibraryID: file.LibraryID, LibraryName: libraryName, StorageProvider: strings.TrimSpace(file.StorageProvider), NormalizedFilename: normalizedFilenameFromPath(file.StoragePath), AffectedCount: len(files), AffectedFiles: files}, nil
}

func (s *Service) CreateFilenameExclusionRule(ctx context.Context, input CreateFilenameExclusionRuleInput) (FilenameExclusionRuleView, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = ScanExclusionReasonAdvertisement
	}
	if !supportedScanExclusionReason(reason) {
		return FilenameExclusionRuleView{}, errors.New("unsupported scan exclusion reason")
	}
	var result database.FilenameExclusionRule
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		file, err := scanExclusionTarget(ctx, tx, MarkScanExclusionInput{InventoryFileID: input.InventoryFileID})
		if err != nil {
			return err
		}
		filename := normalizedFilenameFromPath(file.StoragePath)
		if filename == "" {
			return errors.New("filename exclusion target filename is required")
		}
		rule, err := upsertFilenameExclusionRule(ctx, tx, filename, reason, input.UserID)
		if err != nil {
			return err
		}
		result = rule
		return s.hideFilenameExclusionRuleMatches(ctx, tx, rule)
	})
	if err != nil {
		return FilenameExclusionRuleView{}, err
	}
	return s.filenameExclusionRuleView(ctx, result)
}

func (s *Service) RestoreFilenameExclusionMatch(ctx context.Context, input RestoreFilenameExclusionMatchInput) (database.FilenameExclusionRestore, error) {
	if input.RuleID == 0 || input.InventoryFileID == 0 {
		return database.FilenameExclusionRestore{}, errors.New("rule id and inventory file id are required")
	}
	var restore database.FilenameExclusionRestore
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rule database.FilenameExclusionRule
		if err := tx.WithContext(ctx).First(&rule, input.RuleID).Error; err != nil {
			return err
		}
		var file database.InventoryFile
		if err := tx.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", input.InventoryFileID).First(&file).Error; err != nil {
			return err
		}
		if normalizedFilenameFromPath(file.StoragePath) != rule.NormalizedFilename {
			return errors.New("inventory file does not match this filename exclusion rule")
		}
		query := tx.WithContext(ctx).Where("rule_id = ?", rule.ID)
		if strings.TrimSpace(file.StableIdentityKey) != "" {
			query = query.Where("stable_identity_key = ?", strings.TrimSpace(file.StableIdentityKey))
		} else {
			query = query.Where("storage_path = ?", normalizePath(file.StoragePath))
		}
		err := query.Order("id asc").First(&restore).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			restore = database.FilenameExclusionRestore{RuleID: rule.ID, StableIdentityKey: strings.TrimSpace(file.StableIdentityKey), StoragePath: normalizePath(file.StoragePath), CreatedByUserID: input.UserID}
			return tx.WithContext(ctx).Create(&restore).Error
		}
		return err
	})
	return restore, err
}

func (s *Service) SetFilenameExclusionRuleEnabled(ctx context.Context, input SetFilenameExclusionRuleEnabledInput) (FilenameExclusionRuleView, error) {
	if input.RuleID == 0 {
		return FilenameExclusionRuleView{}, errors.New("filename exclusion rule id is required")
	}
	var rule database.FilenameExclusionRule
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"enabled": input.Enabled, "updated_by_user_id": input.UserID}
		if input.Enabled {
			updates["disabled_at"] = nil
			updates["disabled_by_user_id"] = nil
		} else {
			now := time.Now().UTC()
			updates["disabled_at"] = &now
			updates["disabled_by_user_id"] = input.UserID
		}
		if err := tx.WithContext(ctx).Model(&database.FilenameExclusionRule{}).Where("id = ?", input.RuleID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).First(&rule, input.RuleID).Error; err != nil {
			return err
		}
		if input.Enabled {
			return s.hideFilenameExclusionRuleMatches(ctx, tx, rule)
		}
		return nil
	})
	if err != nil {
		return FilenameExclusionRuleView{}, err
	}
	return s.filenameExclusionRuleView(ctx, rule)
}

func (s *Service) SetScanExclusionEnabled(ctx context.Context, input SetScanExclusionEnabledInput) (database.ScanExclusion, error) {
	if input.ExclusionID == 0 {
		return database.ScanExclusion{}, errors.New("scan exclusion id is required")
	}
	updates := map[string]any{"enabled": input.Enabled}
	if input.Enabled {
		updates["disabled_at"] = nil
		updates["disabled_by_user_id"] = nil
	} else {
		now := time.Now().UTC()
		updates["disabled_at"] = &now
		updates["disabled_by_user_id"] = input.UserID
	}
	if err := s.db.WithContext(ctx).Model(&database.ScanExclusion{}).Where("id = ?", input.ExclusionID).Updates(updates).Error; err != nil {
		return database.ScanExclusion{}, err
	}
	var exclusion database.ScanExclusion
	if err := s.db.WithContext(ctx).First(&exclusion, input.ExclusionID).Error; err != nil {
		return database.ScanExclusion{}, err
	}
	return exclusion, nil
}

func scanExclusionTarget(ctx context.Context, tx *gorm.DB, input MarkScanExclusionInput) (database.InventoryFile, error) {
	if input.InventoryFileID == 0 {
		return database.InventoryFile{}, errors.New("inventory file id is required")
	}
	return scanExclusionTargetFromFile(ctx, tx, input.InventoryFileID)
}

func upsertFilenameExclusionRule(ctx context.Context, tx *gorm.DB, filename string, reason string, userID *uint) (database.FilenameExclusionRule, error) {
	var rule database.FilenameExclusionRule
	err := tx.WithContext(ctx).Where("normalized_filename = ?", filename).Order("id asc").First(&rule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rule = database.FilenameExclusionRule{NormalizedFilename: filename, Reason: reason, Enabled: true, CreatedByUserID: userID, UpdatedByUserID: userID}
		return rule, tx.WithContext(ctx).Create(&rule).Error
	}
	if err != nil {
		return database.FilenameExclusionRule{}, err
	}
	updates := map[string]any{"reason": reason, "enabled": true, "updated_by_user_id": userID, "disabled_at": nil, "disabled_by_user_id": nil}
	if err := tx.WithContext(ctx).Model(&database.FilenameExclusionRule{}).Where("id = ?", rule.ID).Updates(updates).Error; err != nil {
		return database.FilenameExclusionRule{}, err
	}
	if err := tx.WithContext(ctx).First(&rule, rule.ID).Error; err != nil {
		return database.FilenameExclusionRule{}, err
	}
	return rule, nil
}

func (s *Service) hideFilenameExclusionRuleMatches(ctx context.Context, tx *gorm.DB, rule database.FilenameExclusionRule) error {
	files, err := s.filenameExclusionAffectedFiles(ctx, tx, rule.NormalizedFilename, rule.ID)
	if err != nil {
		return err
	}
	var fileIDs []uint
	for _, file := range files {
		if !file.Restored {
			fileIDs = append(fileIDs, file.ID)
		}
	}
	if len(fileIDs) == 0 {
		return nil
	}
	libraryIDs := make(map[uint]struct{})
	for _, file := range files {
		if !file.Restored && file.LibraryID != 0 {
			libraryIDs[file.LibraryID] = struct{}{}
		}
	}
	if err := updateInventoryFilesMissingInBatches(ctx, tx, fileIDs); err != nil {
		return err
	}
	for libraryID := range libraryIDs {
		if err := catalog.NewService(tx).RefreshLibraryProjectionScope(ctx, libraryID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListFilenameExclusionRules(ctx context.Context, input ListScanExclusionsInput) ([]FilenameExclusionRuleView, error) {
	query := s.db.WithContext(ctx).Model(&database.FilenameExclusionRule{})
	if input.Enabled != nil {
		query = query.Where("filename_exclusion_rules.enabled = ?", *input.Enabled)
	}
	var rows []database.FilenameExclusionRule
	if err := query.Order("filename_exclusion_rules.enabled DESC, filename_exclusion_rules.updated_at DESC, filename_exclusion_rules.id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	views := make([]FilenameExclusionRuleView, 0, len(rows))
	for _, row := range rows {
		view, err := s.filenameExclusionRuleView(ctx, row)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) filenameExclusionRuleView(ctx context.Context, rule database.FilenameExclusionRule) (FilenameExclusionRuleView, error) {
	files, err := s.filenameExclusionAffectedFiles(ctx, s.db, rule.NormalizedFilename, rule.ID)
	if err != nil {
		return FilenameExclusionRuleView{}, err
	}
	return FilenameExclusionRuleView{FilenameExclusionRule: rule, AffectedCount: len(files), AffectedFiles: files}, nil
}

func (s *Service) filenameExclusionAffectedFiles(ctx context.Context, tx *gorm.DB, filename string, ruleID uint) ([]FilenameExclusionFileView, error) {
	var rows []database.InventoryFile
	query := tx.WithContext(ctx).Where("deleted_at IS NULL")
	if err := query.Order("storage_path asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	restored := map[uint]bool{}
	var restores []database.FilenameExclusionRestore
	if ruleID != 0 {
		if err := tx.WithContext(ctx).Where("rule_id = ?", ruleID).Find(&restores).Error; err != nil {
			return nil, err
		}
	}
	for _, restore := range restores {
		for _, file := range rows {
			if filenameRestoreMatchesFile(restore, file) {
				restored[file.ID] = true
			}
		}
	}
	files := make([]FilenameExclusionFileView, 0)
	for _, file := range rows {
		if normalizedFilenameFromPath(file.StoragePath) != filename {
			continue
		}
		files = append(files, FilenameExclusionFileView{ID: file.ID, LibraryID: file.LibraryID, StoragePath: normalizePath(file.StoragePath), StableIdentityKey: strings.TrimSpace(file.StableIdentityKey), Status: file.Status, Restored: restored[file.ID]})
	}
	return files, nil
}

func filenameRestoreMatchesFile(restore database.FilenameExclusionRestore, file database.InventoryFile) bool {
	if strings.TrimSpace(restore.StableIdentityKey) != "" && strings.TrimSpace(restore.StableIdentityKey) == strings.TrimSpace(file.StableIdentityKey) {
		return true
	}
	return normalizePath(restore.StoragePath) == normalizePath(file.StoragePath)
}

func scanExclusionTargetFromFile(ctx context.Context, tx *gorm.DB, fileID uint) (database.InventoryFile, error) {
	var file database.InventoryFile
	if err := tx.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", fileID).First(&file).Error; err != nil {
		return database.InventoryFile{}, err
	}
	return file, nil
}

func updateInventoryFilesMissingInBatches(ctx context.Context, tx *gorm.DB, fileIDs []uint) error {
	missingAt := time.Now().UTC()
	return forEachUintBatch(fileIDs, func(batch []uint) error {
		return tx.WithContext(ctx).Model(&database.InventoryFile{}).Where("id IN ?", batch).Updates(map[string]any{"status": inventory.FileStatusMissing, "missing_since": gorm.Expr("COALESCE(missing_since, ?)", missingAt), "deleted_at": nil}).Error
	})
}

func upsertScanExclusion(ctx context.Context, tx *gorm.DB, file database.InventoryFile, reason string, userID *uint) (database.ScanExclusion, error) {
	var exclusion database.ScanExclusion
	query := tx.WithContext(ctx).Where("library_id = ? AND storage_provider = ?", file.LibraryID, strings.TrimSpace(file.StorageProvider))
	if strings.TrimSpace(file.StableIdentityKey) != "" {
		query = query.Where("stable_identity_key = ?", strings.TrimSpace(file.StableIdentityKey))
	} else {
		query = query.Where("storage_path = ?", normalizePath(file.StoragePath))
	}
	err := query.Order("id asc").First(&exclusion).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		exclusion = database.ScanExclusion{LibraryID: file.LibraryID, StorageProvider: strings.TrimSpace(file.StorageProvider), StableIdentityKey: strings.TrimSpace(file.StableIdentityKey), StoragePath: normalizePath(file.StoragePath), Reason: reason, Enabled: true, CreatedByUserID: userID}
		return exclusion, tx.WithContext(ctx).Create(&exclusion).Error
	}
	if err != nil {
		return database.ScanExclusion{}, err
	}
	updates := map[string]any{"stable_identity_key": strings.TrimSpace(file.StableIdentityKey), "storage_path": normalizePath(file.StoragePath), "reason": reason, "enabled": true, "disabled_at": nil, "disabled_by_user_id": nil}
	if err := tx.WithContext(ctx).Model(&database.ScanExclusion{}).Where("id = ?", exclusion.ID).Updates(updates).Error; err != nil {
		return database.ScanExclusion{}, err
	}
	if err := tx.WithContext(ctx).First(&exclusion, exclusion.ID).Error; err != nil {
		return database.ScanExclusion{}, err
	}
	return exclusion, nil
}

func forEachUintBatch(values []uint, fn func([]uint) error) error {
	for start := 0; start < len(values); start += scanExclusionSQLBatchSize {
		end := start + scanExclusionSQLBatchSize
		if end > len(values) {
			end = len(values)
		}
		if err := fn(values[start:end]); err != nil {
			return err
		}
	}
	return nil
}
