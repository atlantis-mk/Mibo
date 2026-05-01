package library

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

type ScanExclusionRuleInput struct {
	LibraryID   *uint  `json:"library_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RuleType    string `json:"rule_type"`
	Value       string `json:"value"`
	Reason      string `json:"reason"`
	Enabled     bool   `json:"enabled"`
	UserID      *uint
}

type UpdateScanExclusionRuleInput struct {
	RuleID      uint
	LibraryID   *uint
	Name        string
	Description string
	RuleType    string
	Value       string
	Reason      string
	Enabled     bool
	UserID      *uint
}

type SetScanExclusionRuleEnabledInput struct {
	RuleID  uint
	Enabled bool
	UserID  *uint
}

func (s *Service) ListScanExclusionRules(ctx context.Context) ([]database.ScanExclusionRule, error) {
	if err := s.ensureDefaultScanExclusionRules(ctx); err != nil {
		return nil, err
	}
	var rules []database.ScanExclusionRule
	if err := s.db.WithContext(ctx).Order("system DESC, enabled DESC, rule_type ASC, value ASC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *Service) CreateScanExclusionRule(ctx context.Context, input ScanExclusionRuleInput) (database.ScanExclusionRule, error) {
	libraryID := normalizeScanExclusionRuleLibraryID(input.LibraryID)
	ruleInput := scanExclusionRuleInput{LibraryID: libraryID, Name: input.Name, Description: input.Description, RuleType: input.RuleType, Value: input.Value, Reason: input.Reason, Enabled: input.Enabled}
	if strings.TrimSpace(ruleInput.Reason) == "" {
		ruleInput.Reason = ScanExclusionReasonAdvertisement
	}
	if err := validateScanExclusionRuleInput(ruleInput); err != nil {
		return database.ScanExclusionRule{}, err
	}
	if err := s.validateScanExclusionRuleScope(ctx, libraryID); err != nil {
		return database.ScanExclusionRule{}, err
	}
	ruleType := strings.TrimSpace(ruleInput.RuleType)
	value := normalizeScanExclusionRuleValue(ruleInput.RuleType, ruleInput.Value)
	if err := s.ensureScanExclusionRuleUnique(ctx, 0, libraryID, ruleType, value); err != nil {
		return database.ScanExclusionRule{}, err
	}
	rule := database.ScanExclusionRule{
		Key:             userScanExclusionRuleKey(libraryID, ruleType, value),
		LibraryID:       libraryID,
		Name:            strings.TrimSpace(ruleInput.Name),
		Description:     strings.TrimSpace(ruleInput.Description),
		RuleType:        ruleType,
		Value:           value,
		Reason:          strings.TrimSpace(ruleInput.Reason),
		Enabled:         input.Enabled,
		System:          false,
		CreatedByUserID: input.UserID,
		UpdatedByUserID: input.UserID,
		DisabledAt:      disabledAtForRule(input.Enabled),
	}
	if err := s.db.WithContext(ctx).Create(&rule).Error; err != nil {
		return database.ScanExclusionRule{}, err
	}
	return rule, nil
}

func (s *Service) UpdateScanExclusionRule(ctx context.Context, input UpdateScanExclusionRuleInput) (database.ScanExclusionRule, error) {
	if input.RuleID == 0 {
		return database.ScanExclusionRule{}, errors.New("scan exclusion rule id is required")
	}
	libraryID := normalizeScanExclusionRuleLibraryID(input.LibraryID)
	ruleInput := scanExclusionRuleInput{LibraryID: libraryID, Name: input.Name, Description: input.Description, RuleType: input.RuleType, Value: input.Value, Reason: input.Reason, Enabled: input.Enabled}
	if strings.TrimSpace(ruleInput.Reason) == "" {
		ruleInput.Reason = ScanExclusionReasonAdvertisement
	}
	if err := validateScanExclusionRuleInput(ruleInput); err != nil {
		return database.ScanExclusionRule{}, err
	}
	if err := s.validateScanExclusionRuleScope(ctx, libraryID); err != nil {
		return database.ScanExclusionRule{}, err
	}
	var rule database.ScanExclusionRule
	if err := s.db.WithContext(ctx).First(&rule, input.RuleID).Error; err != nil {
		return database.ScanExclusionRule{}, err
	}
	if rule.System && libraryID != nil {
		return database.ScanExclusionRule{}, fmt.Errorf("system scan exclusion rules must remain global")
	}
	ruleType := strings.TrimSpace(ruleInput.RuleType)
	value := normalizeScanExclusionRuleValue(ruleInput.RuleType, ruleInput.Value)
	if err := s.ensureScanExclusionRuleUnique(ctx, input.RuleID, libraryID, ruleType, value); err != nil {
		return database.ScanExclusionRule{}, err
	}
	updates := map[string]any{
		"name":               strings.TrimSpace(ruleInput.Name),
		"description":        strings.TrimSpace(ruleInput.Description),
		"library_id":         libraryID,
		"rule_type":          ruleType,
		"value":              value,
		"reason":             strings.TrimSpace(ruleInput.Reason),
		"enabled":            input.Enabled,
		"updated_by_user_id": input.UserID,
		"disabled_at":        disabledAtForRule(input.Enabled),
	}
	if !rule.System {
		updates["key"] = userScanExclusionRuleKey(libraryID, ruleType, value)
	}
	if err := s.db.WithContext(ctx).Model(&database.ScanExclusionRule{}).Where("id = ?", input.RuleID).Updates(updates).Error; err != nil {
		return database.ScanExclusionRule{}, err
	}
	if err := s.db.WithContext(ctx).First(&rule, input.RuleID).Error; err != nil {
		return database.ScanExclusionRule{}, err
	}
	return rule, nil
}

func (s *Service) SetScanExclusionRuleEnabled(ctx context.Context, input SetScanExclusionRuleEnabledInput) (database.ScanExclusionRule, error) {
	if input.RuleID == 0 {
		return database.ScanExclusionRule{}, errors.New("scan exclusion rule id is required")
	}
	updates := map[string]any{"enabled": input.Enabled, "disabled_at": disabledAtForRule(input.Enabled), "updated_by_user_id": input.UserID}
	if err := s.db.WithContext(ctx).Model(&database.ScanExclusionRule{}).Where("id = ?", input.RuleID).Updates(updates).Error; err != nil {
		return database.ScanExclusionRule{}, err
	}
	var rule database.ScanExclusionRule
	if err := s.db.WithContext(ctx).First(&rule, input.RuleID).Error; err != nil {
		return database.ScanExclusionRule{}, err
	}
	return rule, nil
}

func (s *Service) DeleteScanExclusionRule(ctx context.Context, ruleID uint) error {
	if ruleID == 0 {
		return errors.New("scan exclusion rule id is required")
	}
	var rule database.ScanExclusionRule
	if err := s.db.WithContext(ctx).First(&rule, ruleID).Error; err != nil {
		return err
	}
	if rule.System {
		return errors.New("system scan exclusion rules can be disabled but not deleted")
	}
	return s.db.WithContext(ctx).Delete(&database.ScanExclusionRule{}, ruleID).Error
}

func (s *Service) ReplaceLibraryScanExclusionRules(ctx context.Context, libraryID uint, inputs []ScanExclusionRuleInput, userID *uint) ([]database.ScanExclusionRule, error) {
	if libraryID == 0 {
		return nil, errors.New("library id is required")
	}
	if err := s.validateScanExclusionRuleScope(ctx, &libraryID); err != nil {
		return nil, err
	}

	var created []database.ScanExclusionRule
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("library_id = ? AND system = ?", libraryID, false).Delete(&database.ScanExclusionRule{}).Error; err != nil {
			return err
		}
		for _, input := range inputs {
			input.LibraryID = &libraryID
			input.UserID = userID
			rule, err := s.withDB(tx).CreateScanExclusionRule(ctx, input)
			if err != nil {
				return err
			}
			created = append(created, rule)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) withDB(db *gorm.DB) *Service {
	next := *s
	next.db = db
	return &next
}

func (s *Service) validateScanExclusionRuleScope(ctx context.Context, libraryID *uint) error {
	if libraryID == nil || *libraryID == 0 {
		return nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&database.Library{}).Where("id = ?", *libraryID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("library scope does not exist")
	}
	return nil
}

func (s *Service) ensureScanExclusionRuleUnique(ctx context.Context, ruleID uint, libraryID *uint, ruleType string, value string) error {
	query := s.db.WithContext(ctx).Model(&database.ScanExclusionRule{}).Where("rule_type = ? AND value = ?", strings.TrimSpace(ruleType), normalizeScanExclusionRuleValue(ruleType, value))
	if ruleID > 0 {
		query = query.Where("id <> ?", ruleID)
	}
	if libraryID == nil || *libraryID == 0 {
		query = query.Where("library_id IS NULL")
	} else {
		query = query.Where("library_id = ?", *libraryID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("scan exclusion rule already exists in this scope")
	}
	return nil
}

func userScanExclusionRuleKey(libraryID *uint, ruleType string, value string) string {
	normalized := normalizeScanExclusionRuleValue(ruleType, value)
	if libraryID != nil && *libraryID > 0 {
		return fmt.Sprintf("user-library-%d-%s-%s", *libraryID, strings.TrimSpace(ruleType), normalized)
	}
	return fmt.Sprintf("user-global-%s-%s", strings.TrimSpace(ruleType), normalized)
}

func normalizeScanExclusionRuleLibraryID(libraryID *uint) *uint {
	if libraryID == nil || *libraryID == 0 {
		return nil
	}
	id := *libraryID
	return &id
}

func IsScanExclusionRuleNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
