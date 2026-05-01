package library

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

const (
	ScanExclusionReasonAdvertisement = "advertisement"
	ScanExclusionReasonUnwanted      = "unwanted"
	ScanExclusionReasonDuplicate     = "duplicate"
	ScanExclusionReasonWrongImport   = "wrong_import"
	ScanExclusionReasonOther         = "other"

	ScanExclusionRuleTypeFilenameToken    = "filename_token"
	ScanExclusionRuleTypeDirectorySegment = "directory_segment"
	ScanExclusionRuleTypePathPattern      = "path_pattern"

	scanExclusionSkipUserExclusion    = "user_exclusion"
	scanExclusionSkipFilenameRule     = "filename_rule"
	scanExclusionSkipConfigurableRule = "configurable_rule"
)

type scanExclusionDecision struct {
	Excluded bool
	Reason   string
	Source   string
}

type scanExclusionRuleInput struct {
	Key         string
	LibraryID   *uint
	Name        string
	Description string
	RuleType    string
	Value       string
	Reason      string
	Enabled     bool
	System      bool
}

func (s *Service) scanExclusionDecision(ctx context.Context, library database.Library, providerName string, object storage.Object) (scanExclusionDecision, error) {
	rules, err := s.enabledScanExclusionRules(ctx, library.ID)
	if err != nil {
		return scanExclusionDecision{}, err
	}
	return s.scanExclusionDecisionWithRules(ctx, library, providerName, object, rules)
}

func (s *Service) scanExclusionDecisionWithRules(ctx context.Context, library database.Library, providerName string, object storage.Object, rules []database.ScanExclusionRule) (scanExclusionDecision, error) {
	provider := strings.TrimSpace(providerName)
	if provider == "" {
		provider = strings.TrimSpace(object.Provider)
	}
	if provider == "" {
		provider = "local"
	}
	storagePath := normalizePath(object.Path)
	stableIdentity := strings.TrimSpace(object.StableIdentity)

	if allowed, err := s.hasFilenameRestoreException(ctx, library.ID, provider, storagePath, stableIdentity); err != nil {
		return scanExclusionDecision{}, err
	} else if allowed {
		return scanExclusionDecision{}, nil
	}

	var exclusion database.ScanExclusion
	query := s.db.WithContext(ctx).
		Where("storage_provider = ? AND enabled = ?", provider, true).
		Where("library_id = ? OR NOT EXISTS (SELECT 1 FROM libraries WHERE libraries.id = scan_exclusions.library_id)", library.ID)
	if stableIdentity != "" {
		query = query.Where("stable_identity_key = ? OR storage_path = ?", stableIdentity, storagePath)
	} else {
		query = query.Where("storage_path = ?", storagePath)
	}
	err := query.Order(fmt.Sprintf("scan_exclusions.library_id = %d DESC", library.ID)).Order("stable_identity_key <> '' DESC, id asc").First(&exclusion).Error
	if err == nil {
		return scanExclusionDecision{Excluded: true, Reason: strings.TrimSpace(exclusion.Reason), Source: scanExclusionSkipUserExclusion}, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return scanExclusionDecision{}, err
	}
	if rule, ok, err := s.matchingFilenameExclusionRule(ctx, storagePath); err != nil {
		return scanExclusionDecision{}, err
	} else if ok {
		return scanExclusionDecision{Excluded: true, Reason: strings.TrimSpace(rule.Reason), Source: scanExclusionFilenameRuleSource(rule)}, nil
	}

	if rule, ok := matchingScanExclusionRule(storagePath, rules); ok {
		return scanExclusionDecision{Excluded: true, Reason: strings.TrimSpace(rule.Reason), Source: scanExclusionRuleSource(rule)}, nil
	}
	return scanExclusionDecision{}, nil
}

func normalizedFilenameFromPath(value string) string {
	base := strings.TrimSpace(path.Base(normalizePath(value)))
	if base == "." || base == "/" {
		return ""
	}
	return strings.ToLower(base)
}

func normalizedFilenamesEqual(left, right string) bool {
	return normalizedFilenameFromPath(left) == normalizedFilenameFromPath(right)
}

func scanExclusionFilenameRuleSource(rule database.FilenameExclusionRule) string {
	if rule.ID == 0 {
		return scanExclusionSkipFilenameRule
	}
	return fmt.Sprintf("%s:%d", scanExclusionSkipFilenameRule, rule.ID)
}

func (s *Service) matchingFilenameExclusionRule(ctx context.Context, storagePath string) (database.FilenameExclusionRule, bool, error) {
	filename := normalizedFilenameFromPath(storagePath)
	if filename == "" {
		return database.FilenameExclusionRule{}, false, nil
	}
	var rule database.FilenameExclusionRule
	err := s.db.WithContext(ctx).
		Where("normalized_filename = ? AND enabled = ?", filename, true).
		Order("id asc").
		First(&rule).Error
	if err == nil {
		return rule, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return database.FilenameExclusionRule{}, false, nil
	}
	return database.FilenameExclusionRule{}, false, err
}

func (s *Service) hasFilenameRestoreException(ctx context.Context, libraryID uint, provider string, storagePath string, stableIdentity string) (bool, error) {
	rule, ok, err := s.matchingFilenameExclusionRule(ctx, storagePath)
	if err != nil || !ok {
		return false, err
	}
	query := s.db.WithContext(ctx).Model(&database.FilenameExclusionRestore{}).Where("rule_id = ?", rule.ID)
	if strings.TrimSpace(stableIdentity) != "" {
		query = query.Where("stable_identity_key = ? OR storage_path = ?", strings.TrimSpace(stableIdentity), normalizePath(storagePath))
	} else {
		query = query.Where("storage_path = ?", normalizePath(storagePath))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func hasExplicitAdvertisementPathMarker(value string) bool {
	_, ok := matchingScanExclusionRule(value, defaultAdvertisementScanExclusionRules())
	return ok
}

func isAdvertisementDirectorySegment(value string) bool {
	return scanExclusionDirectorySegmentMatches(value, defaultAdvertisementValues())
}

func hasAdvertisementToken(value string) bool {
	return scanExclusionFilenameTokenMatches(value, defaultAdvertisementValues())
}

func normalizedPathTokens(value string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.TrimSpace(value)), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	})
}

func supportedScanExclusionReason(reason string) bool {
	switch strings.TrimSpace(reason) {
	case ScanExclusionReasonAdvertisement, ScanExclusionReasonUnwanted, ScanExclusionReasonDuplicate, ScanExclusionReasonWrongImport, ScanExclusionReasonOther:
		return true
	default:
		return false
	}
}

func supportedScanExclusionRuleType(ruleType string) bool {
	switch strings.TrimSpace(ruleType) {
	case ScanExclusionRuleTypeFilenameToken, ScanExclusionRuleTypeDirectorySegment, ScanExclusionRuleTypePathPattern:
		return true
	default:
		return false
	}
}

func validateScanExclusionRuleInput(input scanExclusionRuleInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("rule name is required")
	}
	ruleType := strings.TrimSpace(input.RuleType)
	if !supportedScanExclusionRuleType(ruleType) {
		return fmt.Errorf("unsupported scan exclusion rule type")
	}
	value := normalizeScanExclusionRuleValue(ruleType, input.Value)
	if value == "" {
		return fmt.Errorf("rule value is required")
	}
	if !supportedScanExclusionReason(input.Reason) {
		return fmt.Errorf("unsupported scan exclusion reason")
	}
	if ruleType == ScanExclusionRuleTypePathPattern {
		if value == "*" || value == "/*" || value == "/**" || value == "**" {
			return fmt.Errorf("path pattern is too broad")
		}
		if _, err := path.Match(value, "/example/video.mkv"); err != nil {
			return fmt.Errorf("invalid path pattern: %w", err)
		}
	}
	return nil
}

func normalizeScanExclusionRuleValue(ruleType string, value string) string {
	trimmed := strings.TrimSpace(value)
	switch strings.TrimSpace(ruleType) {
	case ScanExclusionRuleTypeFilenameToken, ScanExclusionRuleTypeDirectorySegment:
		return strings.ToLower(trimmed)
	default:
		return trimmed
	}
}

func (s *Service) enabledScanExclusionRules(ctx context.Context, libraryID uint) ([]database.ScanExclusionRule, error) {
	if err := s.ensureDefaultScanExclusionRules(ctx); err != nil {
		return nil, err
	}
	var rules []database.ScanExclusionRule
	query := s.db.WithContext(ctx).Where("enabled = ?", true)
	if libraryID > 0 {
		query = query.Where("library_id IS NULL OR library_id = ?", libraryID)
	} else {
		query = query.Where("library_id IS NULL")
	}
	if err := query.Order("system DESC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *Service) ensureDefaultScanExclusionRules(ctx context.Context) error {
	defaults := defaultAdvertisementScanExclusionRules()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, rule := range defaults {
			var existing database.ScanExclusionRule
			err := tx.Where("key = ?", rule.Key).First(&existing).Error
			if err == nil {
				continue
			}
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if err := tx.Create(&rule).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func defaultAdvertisementScanExclusionRules() []database.ScanExclusionRule {
	values := defaultAdvertisementValues()
	rules := make([]database.ScanExclusionRule, 0, len(values)*2)
	for _, value := range values {
		rules = append(rules, database.ScanExclusionRule{Key: "system-ad-filename-token-" + value, Name: "Advertisement filename token: " + value, RuleType: ScanExclusionRuleTypeFilenameToken, Value: value, Reason: ScanExclusionReasonAdvertisement, Enabled: true, System: true})
		rules = append(rules, database.ScanExclusionRule{Key: "system-ad-directory-segment-" + value, Name: "Advertisement directory segment: " + value, RuleType: ScanExclusionRuleTypeDirectorySegment, Value: value, Reason: ScanExclusionReasonAdvertisement, Enabled: true, System: true})
	}
	return rules
}

func defaultAdvertisementValues() []string {
	return []string{"ad", "ads", "advert", "adverts", "advertisement", "advertisements", "commercial", "commercials", "广告"}
}

func matchingScanExclusionRule(storagePath string, rules []database.ScanExclusionRule) (database.ScanExclusionRule, bool) {
	trimmed := strings.TrimSpace(storagePath)
	if trimmed == "" {
		return database.ScanExclusionRule{}, false
	}
	base := strings.TrimSuffix(path.Base(trimmed), path.Ext(trimmed))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		value := normalizeScanExclusionRuleValue(rule.RuleType, rule.Value)
		switch strings.TrimSpace(rule.RuleType) {
		case ScanExclusionRuleTypeFilenameToken:
			if scanExclusionFilenameTokenMatches(base, []string{value}) {
				return rule, true
			}
		case ScanExclusionRuleTypeDirectorySegment:
			for _, segment := range strings.Split(path.Dir(trimmed), "/") {
				if scanExclusionDirectorySegmentMatches(segment, []string{value}) {
					return rule, true
				}
			}
		case ScanExclusionRuleTypePathPattern:
			if ok, _ := path.Match(value, trimmed); ok {
				return rule, true
			}
		}
	}
	return database.ScanExclusionRule{}, false
}

func scanExclusionRuleSource(rule database.ScanExclusionRule) string {
	if rule.ID == 0 {
		return scanExclusionSkipConfigurableRule
	}
	return fmt.Sprintf("%s:%d", scanExclusionSkipConfigurableRule, rule.ID)
}

func scanExclusionFilenameTokenMatches(value string, candidates []string) bool {
	tokens := normalizedPathTokens(value)
	for idx, token := range tokens {
		for _, candidate := range candidates {
			candidate = strings.ToLower(strings.TrimSpace(candidate))
			if candidate == "" || token != candidate {
				continue
			}
			if candidate == "ad" || candidate == "ads" {
				if len(tokens) == 1 || idx > 0 {
					return true
				}
				continue
			}
			return true
		}
	}
	return false
}

func scanExclusionDirectorySegmentMatches(value string, candidates []string) bool {
	segment := strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range candidates {
		if segment == strings.ToLower(strings.TrimSpace(candidate)) && segment != "" {
			return true
		}
	}
	return false
}

func disabledAtForRule(enabled bool) *time.Time {
	if enabled {
		return nil
	}
	now := time.Now().UTC()
	return &now
}

func escapeSQLLikePattern(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch r {
		case '\\', '%', '_':
			builder.WriteRune('\\')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
