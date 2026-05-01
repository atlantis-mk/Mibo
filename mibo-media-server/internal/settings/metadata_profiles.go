package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MetadataProviderInstance struct {
	ID                 uint                      `json:"id"`
	Name               string                    `json:"name"`
	ProviderType       string                    `json:"provider_type"`
	SystemManaged      bool                      `json:"system_managed"`
	Locked             bool                      `json:"locked"`
	Enabled            bool                      `json:"enabled"`
	AvailabilityStatus string                    `json:"availability_status"`
	FailureReason      string                    `json:"failure_reason,omitempty"`
	CooldownUntil      *time.Time                `json:"cooldown_until,omitempty"`
	Configured         bool                      `json:"configured"`
	TMDB               *MetadataProviderSettings `json:"tmdb,omitempty"`
	TVDB               *MetadataProviderSettings `json:"tvdb,omitempty"`
	MetaTube           *MetadataProviderSettings `json:"metatube,omitempty"`
}

type MetadataProfile struct {
	ID                        uint   `json:"id"`
	Name                      string `json:"name"`
	Description               string `json:"description,omitempty"`
	System                    bool   `json:"system"`
	Locked                    bool   `json:"locked"`
	SearchProviderIDs         []uint `json:"search_provider_ids"`
	DetailProviderIDs         []uint `json:"detail_provider_ids"`
	ImageProviderIDs          []uint `json:"image_provider_ids"`
	PeopleProviderIDs         []uint `json:"people_provider_ids"`
	HierarchyProviderIDs      []uint `json:"hierarchy_provider_ids"`
	PreferredMetadataLanguage string `json:"preferred_metadata_language,omitempty"`
	PreferredImageLanguage    string `json:"preferred_image_language,omitempty"`
	FallbackEnabled           bool   `json:"fallback_enabled"`
}

type LibraryMetadataProfileBinding struct {
	LibraryID                 uint   `json:"library_id"`
	MetadataProfileID         uint   `json:"metadata_profile_id"`
	PreferredMetadataLanguage string `json:"preferred_metadata_language,omitempty"`
	PreferredImageLanguage    string `json:"preferred_image_language,omitempty"`
}

type LibraryMetadataStrategy struct {
	LibraryID                 uint   `json:"library_id"`
	TemplateProfileID         uint   `json:"template_profile_id,omitempty"`
	TemplateProfileName       string `json:"template_profile_name,omitempty"`
	SearchProviderIDs         []uint `json:"search_provider_ids"`
	DetailProviderIDs         []uint `json:"detail_provider_ids"`
	ImageProviderIDs          []uint `json:"image_provider_ids"`
	PeopleProviderIDs         []uint `json:"people_provider_ids"`
	HierarchyProviderIDs      []uint `json:"hierarchy_provider_ids"`
	PreferredMetadataLanguage string `json:"preferred_metadata_language,omitempty"`
	PreferredImageLanguage    string `json:"preferred_image_language,omitempty"`
	MetadataCountryCode       string `json:"metadata_country_code,omitempty"`
}

type UpdateMetadataProviderInstanceInput struct {
	Name               string                 `json:"name"`
	ProviderType       string                 `json:"provider_type"`
	Enabled            *bool                  `json:"enabled,omitempty"`
	AvailabilityStatus string                 `json:"availability_status"`
	FailureReason      string                 `json:"failure_reason"`
	CooldownUntil      *time.Time             `json:"cooldown_until,omitempty"`
	TMDB               *MetadataProviderInput `json:"tmdb,omitempty"`
	TVDB               *MetadataProviderInput `json:"tvdb,omitempty"`
	MetaTube           *MetadataProviderInput `json:"metatube,omitempty"`
}

type UpdateMetadataProfileInput struct {
	Name                      string `json:"name"`
	Description               string `json:"description"`
	SearchProviderIDs         []uint `json:"search_provider_ids"`
	DetailProviderIDs         []uint `json:"detail_provider_ids"`
	ImageProviderIDs          []uint `json:"image_provider_ids"`
	PeopleProviderIDs         []uint `json:"people_provider_ids"`
	HierarchyProviderIDs      []uint `json:"hierarchy_provider_ids"`
	PreferredMetadataLanguage string `json:"preferred_metadata_language"`
	PreferredImageLanguage    string `json:"preferred_image_language"`
	FallbackEnabled           *bool  `json:"fallback_enabled,omitempty"`
}

type UpdateLibraryMetadataStrategyInput struct {
	TemplateProfileID         uint   `json:"template_profile_id,omitempty"`
	SearchProviderIDs         []uint `json:"search_provider_ids"`
	DetailProviderIDs         []uint `json:"detail_provider_ids"`
	ImageProviderIDs          []uint `json:"image_provider_ids"`
	PeopleProviderIDs         []uint `json:"people_provider_ids"`
	HierarchyProviderIDs      []uint `json:"hierarchy_provider_ids"`
	PreferredMetadataLanguage string `json:"preferred_metadata_language"`
	PreferredImageLanguage    string `json:"preferred_image_language"`
	MetadataCountryCode       string `json:"metadata_country_code"`
}

type ResolvedMetadataProviderInstance struct {
	Record      database.MetadataProviderInstance
	TMDB        config.TMDBConfig
	TVDB        config.TVDBConfig
	MetaTube    config.MetaTubeConfig
	Configured  bool
	Operational bool
}

type ResolvedLibraryMetadataProfile struct {
	Profile                   database.MetadataProfile
	Binding                   LibraryMetadataProfileBinding
	SearchProviders           []ResolvedMetadataProviderInstance
	DetailProviders           []ResolvedMetadataProviderInstance
	ImageProviders            []ResolvedMetadataProviderInstance
	PeopleProviders           []ResolvedMetadataProviderInstance
	HierarchyProviders        []ResolvedMetadataProviderInstance
	PreferredMetadataLanguage string
	PreferredImageLanguage    string
}

type MetadataExecutionFallbackSummary struct {
	Stage        string   `json:"stage"`
	Attempted    []string `json:"attempted,omitempty"`
	Selected     string   `json:"selected,omitempty"`
	UsedFallback bool     `json:"used_fallback"`
}

func (s *Service) ListMetadataProviderInstances(ctx context.Context) ([]MetadataProviderInstance, error) {
	if err := database.BackfillLibraryMetadataStrategies(s.db.WithContext(ctx)); err != nil {
		return nil, err
	}
	var records []database.MetadataProviderInstance
	if err := s.db.WithContext(ctx).Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]MetadataProviderInstance, 0, len(records))
	for _, record := range records {
		items = append(items, s.metadataProviderInstanceView(record))
	}
	return items, nil
}

func (s *Service) UpsertMetadataProviderInstance(ctx context.Context, id uint, input UpdateMetadataProviderInstanceInput) (MetadataProviderInstance, error) {
	providerType := strings.ToLower(strings.TrimSpace(input.ProviderType))
	if providerType == "" {
		providerType = database.MetadataProviderTypeTMDB
	}
	if providerType != database.MetadataProviderTypeTMDB && providerType != database.MetadataProviderTypeTVDB && providerType != database.MetadataProviderTypeMetaTube && providerType != database.MetadataProviderTypeLocalScan {
		return MetadataProviderInstance{}, fmt.Errorf("unsupported metadata provider type %q", providerType)
	}
	record := database.MetadataProviderInstance{ProviderType: providerType, Enabled: true, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable}
	if providerType == database.MetadataProviderTypeLocalScan {
		record.Name = database.BuiltInLocalScanProviderInstanceName
		record.SystemManaged = true
		record.ConfigJSON = "{}"
	}
	if id != 0 {
		if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
			return MetadataProviderInstance{}, err
		}
		providerType = record.ProviderType
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		record.Name = name
	}
	if strings.TrimSpace(record.Name) == "" {
		return MetadataProviderInstance{}, fmt.Errorf("name is required")
	}
	if record.ProviderType == database.MetadataProviderTypeLocalScan {
		record.Name = database.BuiltInLocalScanProviderInstanceName
		record.SystemManaged = true
		record.Enabled = true
		record.AvailabilityStatus = database.MetadataProviderAvailabilityAvailable
		record.FailureReason = ""
		record.CooldownUntil = nil
		record.ConfigJSON = "{}"
	}
	if id == 0 {
		var existing database.MetadataProviderInstance
		if err := s.db.WithContext(ctx).Where("name = ?", record.Name).First(&existing).Error; err == nil {
			record = existing
			id = existing.ID
			providerType = existing.ProviderType
		} else if err != nil && err != gorm.ErrRecordNotFound {
			return MetadataProviderInstance{}, err
		}
	}
	if input.Enabled != nil && record.ProviderType != database.MetadataProviderTypeLocalScan {
		record.Enabled = *input.Enabled
	}
	if status := strings.TrimSpace(input.AvailabilityStatus); status != "" && record.ProviderType != database.MetadataProviderTypeLocalScan {
		record.AvailabilityStatus = status
	}
	if record.ProviderType != database.MetadataProviderTypeLocalScan {
		record.FailureReason = strings.TrimSpace(input.FailureReason)
		record.CooldownUntil = input.CooldownUntil
		configMap, err := s.providerConfigMap(ctx, providerType, input.TMDB, input.TVDB, input.MetaTube)
		if err != nil {
			return MetadataProviderInstance{}, err
		}
		if len(configMap) > 0 {
			data, err := json.Marshal(configMap)
			if err != nil {
				return MetadataProviderInstance{}, err
			}
			record.ConfigJSON = string(data)
		}
	}
	if id == 0 {
		if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
			return MetadataProviderInstance{}, err
		}
	} else if err := s.db.WithContext(ctx).Save(&record).Error; err != nil {
		return MetadataProviderInstance{}, err
	}
	if record.Name == database.MigratedDefaultTMDBProviderInstanceName {
		if err := s.syncMigratedDefaultProfileProvider(ctx, record); err != nil {
			return MetadataProviderInstance{}, err
		}
	}
	return s.metadataProviderInstanceView(record), nil
}

func (s *Service) syncMigratedDefaultProfileProvider(ctx context.Context, provider database.MetadataProviderInstance) error {
	var profile database.MetadataProfile
	if err := s.db.WithContext(ctx).Where("name = ?", database.MigratedDefaultOnlineProfileName).First(&profile).Error; err != nil {
		return err
	}
	providerIDs := marshalUintList([]uint{provider.ID})
	resolved := s.resolveTMDBProviderInstanceConfig(provider)
	return s.db.WithContext(ctx).Model(&database.MetadataProfile{}).Where("id = ?", profile.ID).Updates(map[string]any{
		"search_providers_json":       providerIDs,
		"detail_providers_json":       providerIDs,
		"image_providers_json":        providerIDs,
		"people_providers_json":       providerIDs,
		"hierarchy_providers_json":    providerIDs,
		"preferred_metadata_language": strings.TrimSpace(resolved.Language),
	}).Error
}

func (s *Service) ListMetadataProfiles(ctx context.Context) ([]MetadataProfile, error) {
	if err := database.BackfillLibraryMetadataStrategies(s.db.WithContext(ctx)); err != nil {
		return nil, err
	}
	var records []database.MetadataProfile
	if err := s.db.WithContext(ctx).Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]MetadataProfile, 0, len(records))
	for _, record := range records {
		if isLegacyLocalOnlyProfile(record) {
			continue
		}
		items = append(items, metadataProfileView(record))
	}
	return items, nil
}

func (s *Service) UpsertMetadataProfile(ctx context.Context, id uint, input UpdateMetadataProfileInput) (MetadataProfile, error) {
	record := database.MetadataProfile{FallbackEnabled: true}
	if id != 0 {
		if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
			return MetadataProfile{}, err
		}
		if isLockedMetadataProfile(record) {
			return MetadataProfile{}, fmt.Errorf("metadata profile %q is system-managed and cannot be edited", record.Name)
		}
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		record.Name = name
	}
	if strings.TrimSpace(record.Name) == "" {
		return MetadataProfile{}, fmt.Errorf("name is required")
	}
	if err := s.validateStageProviderIDs(ctx, "search", input.SearchProviderIDs); err != nil {
		return MetadataProfile{}, err
	}
	if err := s.validateStageProviderIDs(ctx, "detail", input.DetailProviderIDs); err != nil {
		return MetadataProfile{}, err
	}
	if err := s.validateStageProviderIDs(ctx, "image", input.ImageProviderIDs); err != nil {
		return MetadataProfile{}, err
	}
	if err := s.validateStageProviderIDs(ctx, "people", input.PeopleProviderIDs); err != nil {
		return MetadataProfile{}, err
	}
	if err := s.validateStageProviderIDs(ctx, "hierarchy", input.HierarchyProviderIDs); err != nil {
		return MetadataProfile{}, err
	}
	record.Description = strings.TrimSpace(input.Description)
	record.SearchProvidersJSON = marshalUintList(input.SearchProviderIDs)
	record.DetailProvidersJSON = marshalUintList(input.DetailProviderIDs)
	record.ImageProvidersJSON = marshalUintList(input.ImageProviderIDs)
	record.PeopleProvidersJSON = marshalUintList(input.PeopleProviderIDs)
	record.HierarchyProvidersJSON = marshalUintList(input.HierarchyProviderIDs)
	record.PreferredMetadataLanguage = strings.TrimSpace(input.PreferredMetadataLanguage)
	record.PreferredImageLanguage = strings.TrimSpace(input.PreferredImageLanguage)
	if input.FallbackEnabled != nil {
		record.FallbackEnabled = *input.FallbackEnabled
	}
	if id == 0 {
		if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
			return MetadataProfile{}, err
		}
	} else if err := s.db.WithContext(ctx).Save(&record).Error; err != nil {
		return MetadataProfile{}, err
	}
	return metadataProfileView(record), nil
}

func (s *Service) GetLibraryMetadataStrategy(ctx context.Context, libraryID uint) (LibraryMetadataStrategy, error) {
	if err := database.EnsureLibraryMetadataStrategy(s.db.WithContext(ctx), libraryID); err != nil {
		return LibraryMetadataStrategy{}, err
	}
	var record database.LibraryMetadataStrategy
	if err := s.db.WithContext(ctx).Where("library_id = ?", libraryID).First(&record).Error; err != nil {
		return LibraryMetadataStrategy{}, err
	}
	templateProfileID := derefUint(record.MetadataProfileID)
	var templateName string
	if record.MetadataProfileID != nil && *record.MetadataProfileID != 0 {
		var profile database.MetadataProfile
		if err := s.db.WithContext(ctx).First(&profile, *record.MetadataProfileID).Error; err == nil {
			if isLegacyLocalOnlyProfile(profile) {
				templateProfileID = 0
			} else {
				templateName = strings.TrimSpace(profile.Name)
			}
		}
	}
	return LibraryMetadataStrategy{LibraryID: record.LibraryID, TemplateProfileID: templateProfileID, TemplateProfileName: templateName, SearchProviderIDs: unmarshalUintList(record.SearchProvidersJSON), DetailProviderIDs: unmarshalUintList(record.DetailProvidersJSON), ImageProviderIDs: unmarshalUintList(record.ImageProvidersJSON), PeopleProviderIDs: unmarshalUintList(record.PeopleProvidersJSON), HierarchyProviderIDs: unmarshalUintList(record.HierarchyProvidersJSON), PreferredMetadataLanguage: strings.TrimSpace(record.PreferredMetadataLanguage), PreferredImageLanguage: strings.TrimSpace(record.PreferredImageLanguage), MetadataCountryCode: strings.TrimSpace(record.MetadataCountryCode)}, nil
}

func (s *Service) UpdateLibraryMetadataStrategy(ctx context.Context, libraryID uint, input UpdateLibraryMetadataStrategyInput) (LibraryMetadataStrategy, error) {
	if libraryID == 0 {
		return LibraryMetadataStrategy{}, fmt.Errorf("library id is required")
	}
	if err := database.EnsureLibraryMetadataStrategy(s.db.WithContext(ctx), libraryID); err != nil {
		return LibraryMetadataStrategy{}, err
	}
	resolved, err := s.resolveStrategyInput(ctx, input.TemplateProfileID, input)
	if err != nil {
		return LibraryMetadataStrategy{}, err
	}
	record := database.LibraryMetadataStrategy{LibraryID: libraryID, MetadataProfileID: uintPtrOrNil(resolved.TemplateProfileID), SearchProvidersJSON: marshalUintList(resolved.SearchProviderIDs), DetailProvidersJSON: marshalUintList(resolved.DetailProviderIDs), ImageProvidersJSON: marshalUintList(resolved.ImageProviderIDs), PeopleProvidersJSON: marshalUintList(resolved.PeopleProviderIDs), HierarchyProvidersJSON: marshalUintList(resolved.HierarchyProviderIDs), PreferredMetadataLanguage: strings.TrimSpace(resolved.PreferredMetadataLanguage), PreferredImageLanguage: strings.TrimSpace(resolved.PreferredImageLanguage), MetadataCountryCode: strings.TrimSpace(resolved.MetadataCountryCode)}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "library_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"metadata_profile_id", "search_providers_json", "detail_providers_json", "image_providers_json", "people_providers_json", "hierarchy_providers_json", "preferred_metadata_language", "preferred_image_language", "metadata_country_code", "updated_at"}),
	}).Create(&record).Error; err != nil {
		return LibraryMetadataStrategy{}, err
	}
	return s.GetLibraryMetadataStrategy(ctx, libraryID)
}

func (s *Service) ResolveLibraryMetadataProfile(ctx context.Context, libraryID uint) (ResolvedLibraryMetadataProfile, error) {
	if err := database.EnsureLibraryMetadataStrategy(s.db.WithContext(ctx), libraryID); err != nil {
		return ResolvedLibraryMetadataProfile{}, err
	}
	var strategy database.LibraryMetadataStrategy
	if err := s.db.WithContext(ctx).Where("library_id = ?", libraryID).First(&strategy).Error; err != nil {
		return ResolvedLibraryMetadataProfile{}, err
	}
	binding := LibraryMetadataProfileBinding{LibraryID: libraryID, MetadataProfileID: derefUint(strategy.MetadataProfileID), PreferredMetadataLanguage: strategy.PreferredMetadataLanguage, PreferredImageLanguage: strategy.PreferredImageLanguage}
	profile := database.MetadataProfile{}
	if strategy.MetadataProfileID != nil && *strategy.MetadataProfileID != 0 {
		if err := s.db.WithContext(ctx).First(&profile, *strategy.MetadataProfileID).Error; err != nil {
			return ResolvedLibraryMetadataProfile{}, err
		}
	}
	resolved := ResolvedLibraryMetadataProfile{
		Profile:                   profile,
		Binding:                   binding,
		PreferredMetadataLanguage: strings.TrimSpace(firstNonEmpty(strategy.PreferredMetadataLanguage, profile.PreferredMetadataLanguage)),
		PreferredImageLanguage:    strings.TrimSpace(firstNonEmpty(strategy.PreferredImageLanguage, profile.PreferredImageLanguage)),
	}
	var err error
	if resolved.SearchProviders, err = s.resolveProviderStage(ctx, strategy.SearchProvidersJSON); err != nil {
		return ResolvedLibraryMetadataProfile{}, err
	}
	if resolved.DetailProviders, err = s.resolveDetailProviderStage(ctx, strategy.DetailProvidersJSON); err != nil {
		return ResolvedLibraryMetadataProfile{}, err
	}
	if resolved.ImageProviders, err = s.resolveProviderStage(ctx, strategy.ImageProvidersJSON); err != nil {
		return ResolvedLibraryMetadataProfile{}, err
	}
	if resolved.PeopleProviders, err = s.resolveProviderStage(ctx, strategy.PeopleProvidersJSON); err != nil {
		return ResolvedLibraryMetadataProfile{}, err
	}
	if resolved.HierarchyProviders, err = s.resolveProviderStage(ctx, strategy.HierarchyProvidersJSON); err != nil {
		return ResolvedLibraryMetadataProfile{}, err
	}
	return resolved, nil
}

func (s *Service) metadataProviderInstanceView(record database.MetadataProviderInstance) MetadataProviderInstance {
	view := MetadataProviderInstance{ID: record.ID, Name: record.Name, ProviderType: record.ProviderType, SystemManaged: record.SystemManaged, Locked: record.SystemManaged, Enabled: record.Enabled, AvailabilityStatus: record.AvailabilityStatus, FailureReason: record.FailureReason, CooldownUntil: record.CooldownUntil}
	switch record.ProviderType {
	case database.MetadataProviderTypeTMDB:
		cfg := s.resolveTMDBProviderInstanceConfig(record)
		view.Configured = strings.TrimSpace(cfg.APIKey) != ""
		view.TMDB = &MetadataProviderSettings{Configured: view.Configured, APIKeyMasked: view.Configured, BaseURL: cfg.BaseURL, ImageBaseURL: cfg.ImageBaseURL, Language: cfg.Language, Timeout: cfg.Timeout.String(), Source: "database", Implementation: "active"}
	case database.MetadataProviderTypeTVDB:
		cfg := s.resolveTVDBProviderInstanceConfig(record)
		view.Configured = strings.TrimSpace(cfg.APIKey) != ""
		view.TVDB = &MetadataProviderSettings{Configured: view.Configured, APIKeyMasked: view.Configured, BaseURL: cfg.BaseURL, Language: cfg.Language, Timeout: cfg.Timeout.String(), Source: "database", Implementation: "configured"}
	case database.MetadataProviderTypeMetaTube:
		cfg := s.resolveMetaTubeProviderInstanceConfig(record)
		view.Configured = strings.TrimSpace(cfg.BaseURL) != ""
		view.MetaTube = &MetadataProviderSettings{Configured: view.Configured, APIKeyMasked: strings.TrimSpace(cfg.Token) != "", BaseURL: cfg.BaseURL, Timeout: cfg.Timeout.String(), Source: "database", Implementation: "active", UpstreamProviderFilter: cfg.UpstreamProviderFilter, FallbackEnabled: cfg.FallbackEnabled}
	case database.MetadataProviderTypeLocalScan:
		view.Configured = true
	}
	return view
}

func metadataProfileView(record database.MetadataProfile) MetadataProfile {
	return MetadataProfile{ID: record.ID, Name: record.Name, Description: record.Description, System: isSystemMetadataProfile(record), Locked: isLockedMetadataProfile(record), SearchProviderIDs: unmarshalUintList(record.SearchProvidersJSON), DetailProviderIDs: unmarshalUintList(record.DetailProvidersJSON), ImageProviderIDs: unmarshalUintList(record.ImageProvidersJSON), PeopleProviderIDs: unmarshalUintList(record.PeopleProvidersJSON), HierarchyProviderIDs: unmarshalUintList(record.HierarchyProvidersJSON), PreferredMetadataLanguage: strings.TrimSpace(record.PreferredMetadataLanguage), PreferredImageLanguage: strings.TrimSpace(record.PreferredImageLanguage), FallbackEnabled: record.FallbackEnabled}
}

func isSystemMetadataProfile(record database.MetadataProfile) bool {
	return false
}

func isLockedMetadataProfile(record database.MetadataProfile) bool {
	return isSystemMetadataProfile(record)
}

func isLegacyLocalOnlyProfile(record database.MetadataProfile) bool {
	return strings.TrimSpace(record.Name) == database.MigratedDefaultLocalProfileName
}

func (s *Service) resolveProviderStage(ctx context.Context, raw string) ([]ResolvedMetadataProviderInstance, error) {
	ids := unmarshalUintList(raw)
	return s.resolveProviderIDs(ctx, ids)
}

func (s *Service) resolveDetailProviderStage(ctx context.Context, raw string) ([]ResolvedMetadataProviderInstance, error) {
	ids, err := s.withLocalScanDetailFallback(ctx, unmarshalUintList(raw))
	if err != nil {
		return nil, err
	}
	return s.resolveProviderIDs(ctx, ids)
}

func (s *Service) resolveProviderIDs(ctx context.Context, ids []uint) ([]ResolvedMetadataProviderInstance, error) {
	resolved := make([]ResolvedMetadataProviderInstance, 0, len(ids))
	for _, id := range ids {
		provider, err := s.ResolveMetadataProviderInstance(ctx, id)
		if err != nil {
			return nil, err
		}
		if provider.Operational {
			resolved = append(resolved, provider)
		}
	}
	return resolved, nil
}

func (s *Service) withLocalScanDetailFallback(ctx context.Context, ids []uint) ([]uint, error) {
	localScanID, err := s.localScanProviderID(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]uint, 0, len(ids)+1)
	seen := map[uint]struct{}{}
	for _, id := range ids {
		if id == 0 || id == localScanID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return append(result, localScanID), nil
}

func (s *Service) localScanProviderID(ctx context.Context) (uint, error) {
	if err := database.BackfillMetadataProfiles(s.db.WithContext(ctx)); err != nil {
		return 0, err
	}
	var record database.MetadataProviderInstance
	if err := s.db.WithContext(ctx).Where("name = ?", database.BuiltInLocalScanProviderInstanceName).First(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (s *Service) ResolveMetadataProviderInstance(ctx context.Context, id uint) (ResolvedMetadataProviderInstance, error) {
	var record database.MetadataProviderInstance
	if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return ResolvedMetadataProviderInstance{}, err
	}
	resolved := ResolvedMetadataProviderInstance{Record: record}
	switch record.ProviderType {
	case database.MetadataProviderTypeTMDB:
		cfg := s.resolveTMDBProviderInstanceConfig(record)
		resolved.TMDB = cfg
		resolved.Configured = strings.TrimSpace(cfg.APIKey) != ""
	case database.MetadataProviderTypeTVDB:
		cfg := s.resolveTVDBProviderInstanceConfig(record)
		resolved.TVDB = cfg
		resolved.Configured = strings.TrimSpace(cfg.APIKey) != ""
	case database.MetadataProviderTypeMetaTube:
		cfg := s.resolveMetaTubeProviderInstanceConfig(record)
		resolved.MetaTube = cfg
		resolved.Configured = strings.TrimSpace(cfg.BaseURL) != ""
	case database.MetadataProviderTypeLocalScan:
		resolved.Configured = true
	default:
		return ResolvedMetadataProviderInstance{}, fmt.Errorf("unsupported metadata provider type %q", record.ProviderType)
	}
	resolved.Operational = record.Enabled && resolved.Configured && providerAvailabilityActive(record) && providerTypeHasExecutionSupport(record.ProviderType)
	return resolved, nil
}

func (s *Service) resolveTMDBProviderInstanceConfig(record database.MetadataProviderInstance) config.TMDBConfig {
	resolved := s.fallback.TMDB
	var values map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(record.ConfigJSON)), &values); err == nil {
		applyStringOverride(&resolved.APIKey, values[tmdbAPIKeyKey])
		applyStringOverride(&resolved.BaseURL, values[tmdbBaseURLKey])
		applyStringOverride(&resolved.ImageBaseURL, values[tmdbImageBaseURLKey])
		applyStringOverride(&resolved.Language, values[tmdbLanguageKey])
		applyDurationOverride(&resolved.Timeout, values[tmdbTimeoutKey])
	}
	return resolved
}

func (s *Service) resolveTVDBProviderInstanceConfig(record database.MetadataProviderInstance) config.TVDBConfig {
	resolved := s.fallback.TVDB
	var values map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(record.ConfigJSON)), &values); err == nil {
		applyStringOverride(&resolved.APIKey, values[tvdbAPIKeyKey])
		applyStringOverride(&resolved.BaseURL, values[tvdbBaseURLKey])
		applyStringOverride(&resolved.Language, values[tvdbLanguageKey])
		applyDurationOverride(&resolved.Timeout, values[tvdbTimeoutKey])
	}
	return resolved
}

func (s *Service) resolveMetaTubeProviderInstanceConfig(record database.MetadataProviderInstance) config.MetaTubeConfig {
	resolved := s.fallback.MetaTube
	var values map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(record.ConfigJSON)), &values); err == nil {
		applyStringOverride(&resolved.Token, values[metatubeTokenKey])
		applyStringOverride(&resolved.BaseURL, values[metatubeBaseURLKey])
		applyStringOverride(&resolved.UpstreamProviderFilter, values[metatubeUpstreamProviderFilterKey])
		if value := strings.TrimSpace(values[metatubeFallbackEnabledKey]); value != "" {
			if parsed, err := strconv.ParseBool(value); err == nil {
				resolved.FallbackEnabled = parsed
			}
		}
		applyDurationOverride(&resolved.Timeout, values[metatubeTimeoutKey])
	}
	resolved.BaseURL = strings.TrimRight(strings.TrimSpace(resolved.BaseURL), "/")
	return resolved
}

func providerAvailabilityActive(record database.MetadataProviderInstance) bool {
	status := strings.TrimSpace(record.AvailabilityStatus)
	if status == "" || status == database.MetadataProviderAvailabilityAvailable {
		return record.CooldownUntil == nil || record.CooldownUntil.Before(time.Now().UTC())
	}
	if status == database.MetadataProviderAvailabilityCooldown {
		return record.CooldownUntil != nil && record.CooldownUntil.Before(time.Now().UTC())
	}
	return false
}

func (s *Service) providerConfigMap(ctx context.Context, providerType string, tmdbInput *MetadataProviderInput, tvdbInput *MetadataProviderInput, metatubeInput *MetadataProviderInput) (map[string]string, error) {
	switch providerType {
	case database.MetadataProviderTypeTMDB:
		baseCfg, _, err := s.ResolveTMDBConfig(ctx)
		if err != nil {
			return nil, err
		}
		configMap := map[string]string{}
		if tmdbInput == nil {
			if strings.TrimSpace(baseCfg.APIKey) != "" {
				configMap[tmdbAPIKeyKey] = baseCfg.APIKey
			}
			if strings.TrimSpace(baseCfg.BaseURL) != "" {
				configMap[tmdbBaseURLKey] = baseCfg.BaseURL
			}
			if strings.TrimSpace(baseCfg.ImageBaseURL) != "" {
				configMap[tmdbImageBaseURLKey] = baseCfg.ImageBaseURL
			}
			if strings.TrimSpace(baseCfg.Language) != "" {
				configMap[tmdbLanguageKey] = baseCfg.Language
			}
			if baseCfg.Timeout > 0 {
				configMap[tmdbTimeoutKey] = baseCfg.Timeout.String()
			}
			return configMap, nil
		}
		if strings.TrimSpace(tmdbInput.APIKey) != "" {
			configMap[tmdbAPIKeyKey] = strings.TrimSpace(tmdbInput.APIKey)
		} else if !tmdbInput.ClearAPIKey && strings.TrimSpace(baseCfg.APIKey) != "" {
			configMap[tmdbAPIKeyKey] = baseCfg.APIKey
		}
		configMap[tmdbBaseURLKey] = firstNonEmpty(tmdbInput.BaseURL, baseCfg.BaseURL)
		configMap[tmdbImageBaseURLKey] = firstNonEmpty(tmdbInput.ImageBaseURL, baseCfg.ImageBaseURL)
		configMap[tmdbLanguageKey] = firstNonEmpty(tmdbInput.Language, baseCfg.Language)
		configMap[tmdbTimeoutKey] = firstNonEmpty(tmdbInput.Timeout, baseCfg.Timeout.String())
		return configMap, nil
	case database.MetadataProviderTypeTVDB:
		baseCfg := s.fallback.TVDB
		configMap := map[string]string{}
		input := tvdbInput
		if input == nil {
			if strings.TrimSpace(baseCfg.APIKey) != "" {
				configMap[tvdbAPIKeyKey] = baseCfg.APIKey
			}
			if strings.TrimSpace(baseCfg.BaseURL) != "" {
				configMap[tvdbBaseURLKey] = baseCfg.BaseURL
			}
			if strings.TrimSpace(baseCfg.Language) != "" {
				configMap[tvdbLanguageKey] = baseCfg.Language
			}
			if baseCfg.Timeout > 0 {
				configMap[tvdbTimeoutKey] = baseCfg.Timeout.String()
			}
			return configMap, nil
		}
		if strings.TrimSpace(input.APIKey) != "" {
			configMap[tvdbAPIKeyKey] = strings.TrimSpace(input.APIKey)
		} else if !input.ClearAPIKey && strings.TrimSpace(baseCfg.APIKey) != "" {
			configMap[tvdbAPIKeyKey] = baseCfg.APIKey
		}
		configMap[tvdbBaseURLKey] = firstNonEmpty(input.BaseURL, baseCfg.BaseURL)
		configMap[tvdbLanguageKey] = firstNonEmpty(input.Language, baseCfg.Language)
		configMap[tvdbTimeoutKey] = firstNonEmpty(input.Timeout, baseCfg.Timeout.String())
		return configMap, nil
	case database.MetadataProviderTypeMetaTube:
		baseCfg := s.fallback.MetaTube
		configMap := map[string]string{}
		input := metatubeInput
		if input == nil {
			if strings.TrimSpace(baseCfg.Token) != "" {
				configMap[metatubeTokenKey] = baseCfg.Token
			}
			if strings.TrimSpace(baseCfg.BaseURL) != "" {
				configMap[metatubeBaseURLKey] = strings.TrimRight(strings.TrimSpace(baseCfg.BaseURL), "/")
			}
			if strings.TrimSpace(baseCfg.UpstreamProviderFilter) != "" {
				configMap[metatubeUpstreamProviderFilterKey] = strings.TrimSpace(baseCfg.UpstreamProviderFilter)
			}
			configMap[metatubeFallbackEnabledKey] = strconv.FormatBool(baseCfg.FallbackEnabled)
			if baseCfg.Timeout > 0 {
				configMap[metatubeTimeoutKey] = baseCfg.Timeout.String()
			}
			return configMap, nil
		}
		if strings.TrimSpace(input.APIKey) != "" {
			configMap[metatubeTokenKey] = strings.TrimSpace(input.APIKey)
		} else if !input.ClearAPIKey && strings.TrimSpace(baseCfg.Token) != "" {
			configMap[metatubeTokenKey] = baseCfg.Token
		}
		configMap[metatubeBaseURLKey] = strings.TrimRight(firstNonEmpty(input.BaseURL, baseCfg.BaseURL), "/")
		configMap[metatubeUpstreamProviderFilterKey] = firstNonEmpty(input.UpstreamProviderFilter, baseCfg.UpstreamProviderFilter)
		fallbackEnabled := baseCfg.FallbackEnabled
		if input.FallbackEnabled != nil {
			fallbackEnabled = *input.FallbackEnabled
		}
		configMap[metatubeFallbackEnabledKey] = strconv.FormatBool(fallbackEnabled)
		configMap[metatubeTimeoutKey] = firstNonEmpty(input.Timeout, baseCfg.Timeout.String())
		return configMap, nil
	default:
		return nil, fmt.Errorf("unsupported metadata provider type %q", providerType)
	}
}

func (s *Service) validateStageProviderIDs(ctx context.Context, stage string, ids []uint) error {
	seen := map[uint]struct{}{}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		var record database.MetadataProviderInstance
		if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
			return err
		}
		if !record.Enabled {
			return fmt.Errorf("provider instance %q is disabled", record.Name)
		}
		if !providerSupportsStage(record.ProviderType, stage) {
			return fmt.Errorf("provider instance %q does not support metadata stage %q", record.Name, stage)
		}
	}
	return nil
}

func marshalUintList(values []uint) string {
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func unmarshalUintList(raw string) []uint {
	var values []uint
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return []uint{}
	}
	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func providerSupportsStage(providerType string, stage string) bool {
	switch providerType {
	case database.MetadataProviderTypeTMDB:
		return stage == "search" || stage == "detail" || stage == "image" || stage == "people" || stage == "hierarchy"
	case database.MetadataProviderTypeTVDB:
		return false
	case database.MetadataProviderTypeMetaTube:
		return stage == "search" || stage == "detail" || stage == "image" || stage == "people"
	case database.MetadataProviderTypeLocalScan:
		return stage == "detail"
	default:
		return false
	}
}

func providerTypeHasExecutionSupport(providerType string) bool {
	return providerType == database.MetadataProviderTypeTMDB || providerType == database.MetadataProviderTypeMetaTube || providerType == database.MetadataProviderTypeLocalScan
}

func (s *Service) resolveStrategyInput(ctx context.Context, templateProfileID uint, input UpdateLibraryMetadataStrategyInput) (UpdateLibraryMetadataStrategyInput, error) {
	resolved := input
	if templateProfileID == 0 {
		templateProfileID = input.TemplateProfileID
	}
	if templateProfileID != 0 {
		var profile database.MetadataProfile
		if err := s.db.WithContext(ctx).First(&profile, templateProfileID).Error; err != nil {
			return UpdateLibraryMetadataStrategyInput{}, err
		}
		resolved.TemplateProfileID = templateProfileID
		if len(resolved.SearchProviderIDs) == 0 {
			resolved.SearchProviderIDs = unmarshalUintList(profile.SearchProvidersJSON)
		}
		if len(resolved.DetailProviderIDs) == 0 {
			resolved.DetailProviderIDs = unmarshalUintList(profile.DetailProvidersJSON)
		}
		if len(resolved.ImageProviderIDs) == 0 {
			resolved.ImageProviderIDs = unmarshalUintList(profile.ImageProvidersJSON)
		}
		if len(resolved.PeopleProviderIDs) == 0 {
			resolved.PeopleProviderIDs = unmarshalUintList(profile.PeopleProvidersJSON)
		}
		if len(resolved.HierarchyProviderIDs) == 0 {
			resolved.HierarchyProviderIDs = unmarshalUintList(profile.HierarchyProvidersJSON)
		}
		resolved.PreferredMetadataLanguage = firstNonEmpty(resolved.PreferredMetadataLanguage, profile.PreferredMetadataLanguage)
		resolved.PreferredImageLanguage = firstNonEmpty(resolved.PreferredImageLanguage, profile.PreferredImageLanguage)
	}
	detailProviderIDs, err := s.withLocalScanDetailFallback(ctx, resolved.DetailProviderIDs)
	if err != nil {
		return UpdateLibraryMetadataStrategyInput{}, err
	}
	resolved.DetailProviderIDs = detailProviderIDs
	if err := s.validateStageProviderIDs(ctx, "search", resolved.SearchProviderIDs); err != nil {
		return UpdateLibraryMetadataStrategyInput{}, err
	}
	if err := s.validateStageProviderIDs(ctx, "detail", resolved.DetailProviderIDs); err != nil {
		return UpdateLibraryMetadataStrategyInput{}, err
	}
	if err := s.validateStageProviderIDs(ctx, "image", resolved.ImageProviderIDs); err != nil {
		return UpdateLibraryMetadataStrategyInput{}, err
	}
	if err := s.validateStageProviderIDs(ctx, "people", resolved.PeopleProviderIDs); err != nil {
		return UpdateLibraryMetadataStrategyInput{}, err
	}
	if err := s.validateStageProviderIDs(ctx, "hierarchy", resolved.HierarchyProviderIDs); err != nil {
		return UpdateLibraryMetadataStrategyInput{}, err
	}
	return resolved, nil
}

func derefUint(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func uintPtrOrNil(value uint) *uint {
	if value == 0 {
		return nil
	}
	copy := value
	return &copy
}
