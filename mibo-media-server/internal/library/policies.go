package library

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

type LibraryPathView struct {
	ID            uint   `json:"id"`
	LibraryID     uint   `json:"library_id"`
	MediaSourceID uint   `json:"media_source_id"`
	RootPath      string `json:"root_path"`
	DisplayName   string `json:"display_name"`
	Enabled       bool   `json:"enabled"`
}

type LibraryPoliciesView struct {
	Scan     LibraryScanPolicyView     `json:"scan"`
	Metadata LibraryMetadataPolicyView `json:"metadata"`
	Playback LibraryPlaybackPolicyView `json:"playback"`
	Subtitle LibrarySubtitlePolicyView `json:"subtitle"`
}

type LibraryScanPolicyView struct {
	ScannerEnabled             bool     `json:"scanner_enabled"`
	RealtimeMonitorEnabled     bool     `json:"realtime_monitor_enabled"`
	ScheduledRefreshEnabled    bool     `json:"scheduled_refresh_enabled"`
	RefreshIntervalHours       int      `json:"refresh_interval_hours"`
	IgnoreHiddenFiles          bool     `json:"ignore_hidden_files"`
	IgnoreFileExtensions       []string `json:"ignore_file_extensions"`
	MinFileSizeBytes           int64    `json:"min_file_size_bytes"`
	SampleIgnoreSizeBytes      int64    `json:"sample_ignore_size_bytes"`
	InventoryProbeBatchEnabled bool     `json:"inventory_probe_batch_enabled"`
	ConfigurableExclusionRules bool     `json:"configurable_exclusion_rules"`
}

type LibraryMetadataPolicyView struct {
	PreferredMetadataLanguage string `json:"preferred_metadata_language"`
	PreferredImageLanguage    string `json:"preferred_image_language"`
	MetadataCountryCode       string `json:"metadata_country_code"`
	MetadataProfileID         uint   `json:"metadata_profile_id,omitempty"`
	MetadataProfileName       string `json:"metadata_profile_name,omitempty"`
}

type LibraryPlaybackPolicyView struct {
	ResumeEnabled            bool `json:"resume_enabled"`
	MinResumePct             int  `json:"min_resume_pct"`
	MaxResumePct             int  `json:"max_resume_pct"`
	MinResumeDurationSeconds int  `json:"min_resume_duration_seconds"`
}

type LibrarySubtitlePolicyView struct {
	ExternalSidecarsEnabled        bool     `json:"external_sidecars_enabled"`
	PreferredLanguages             []string `json:"preferred_languages"`
	RequirePerfectMatch            bool     `json:"require_perfect_match"`
	SaveWithMedia                  bool     `json:"save_with_media"`
	TolerateUnavailableSubtitles   bool     `json:"tolerate_unavailable_subtitles"`
	SkipIfEmbeddedSubtitlesPresent bool     `json:"skip_if_embedded_subtitles_present"`
	SkipIfAudioTrackMatches        bool     `json:"skip_if_audio_track_matches"`
}

type LibraryPathInput struct {
	MediaSourceID uint   `json:"media_source_id"`
	RootPath      string `json:"root_path"`
	DisplayName   string `json:"display_name"`
	Enabled       *bool  `json:"enabled,omitempty"`
}

type UpdateLibraryPathInput struct {
	PathID        uint
	MediaSourceID uint   `json:"media_source_id"`
	RootPath      string `json:"root_path"`
	DisplayName   string `json:"display_name"`
	Enabled       *bool  `json:"enabled,omitempty"`
}

type EffectiveLibraryConfig struct {
	Library        database.Library
	Paths          []database.LibraryPath
	ScanPolicy     database.LibraryScanPolicy
	MetadataPolicy database.LibraryMetadataPolicy
	ProfileBinding settings.LibraryMetadataProfileBinding
	Profile        settings.MetadataProfile
	PlaybackPolicy database.LibraryPlaybackPolicy
	SubtitlePolicy database.LibrarySubtitlePolicy
}

func (c EffectiveLibraryConfig) PathsView() []LibraryPathView {
	views := make([]LibraryPathView, 0, len(c.Paths))
	for _, path := range c.Paths {
		views = append(views, LibraryPathView{ID: path.ID, LibraryID: path.LibraryID, MediaSourceID: path.MediaSourceID, RootPath: path.RootPath, DisplayName: path.DisplayName, Enabled: path.Enabled})
	}
	return views
}

func (c EffectiveLibraryConfig) PoliciesView() LibraryPoliciesView {
	return LibraryPoliciesView{
		Scan:     LibraryScanPolicyView{ScannerEnabled: c.ScanPolicy.ScannerEnabled, RealtimeMonitorEnabled: c.ScanPolicy.RealtimeMonitorEnabled, ScheduledRefreshEnabled: c.ScanPolicy.ScheduledRefreshEnabled, RefreshIntervalHours: c.ScanPolicy.RefreshIntervalHours, IgnoreHiddenFiles: c.ScanPolicy.IgnoreHiddenFiles, IgnoreFileExtensions: stringListFromJSON(c.ScanPolicy.IgnoreFileExtensionsJSON), MinFileSizeBytes: c.ScanPolicy.MinFileSizeBytes, SampleIgnoreSizeBytes: c.ScanPolicy.SampleIgnoreSizeBytes, InventoryProbeBatchEnabled: c.ScanPolicy.InventoryProbeBatchEnabled, ConfigurableExclusionRules: c.ScanPolicy.ConfigurableExclusionRules},
		Metadata: LibraryMetadataPolicyView{PreferredMetadataLanguage: c.MetadataPolicy.PreferredMetadataLanguage, PreferredImageLanguage: c.MetadataPolicy.PreferredImageLanguage, MetadataCountryCode: c.MetadataPolicy.MetadataCountryCode, MetadataProfileID: c.ProfileBinding.MetadataProfileID, MetadataProfileName: c.Profile.Name},
		Playback: LibraryPlaybackPolicyView{ResumeEnabled: c.PlaybackPolicy.ResumeEnabled, MinResumePct: c.PlaybackPolicy.MinResumePct, MaxResumePct: c.PlaybackPolicy.MaxResumePct, MinResumeDurationSeconds: c.PlaybackPolicy.MinResumeDurationSeconds},
		Subtitle: LibrarySubtitlePolicyView{ExternalSidecarsEnabled: c.SubtitlePolicy.ExternalSidecarsEnabled, PreferredLanguages: stringListFromJSON(c.SubtitlePolicy.PreferredLanguagesJSON), RequirePerfectMatch: c.SubtitlePolicy.RequirePerfectMatch, SaveWithMedia: c.SubtitlePolicy.SaveWithMedia, TolerateUnavailableSubtitles: c.SubtitlePolicy.TolerateUnavailableSubtitles, SkipIfEmbeddedSubtitlesPresent: c.SubtitlePolicy.SkipIfEmbeddedSubtitlesPresent, SkipIfAudioTrackMatches: c.SubtitlePolicy.SkipIfAudioTrackMatches},
	}
}

func (s *Service) EffectiveLibraryConfig(ctx context.Context, libraryID uint) (EffectiveLibraryConfig, error) {
	var record database.Library
	if err := s.db.WithContext(ctx).First(&record, libraryID).Error; err != nil {
		return EffectiveLibraryConfig{}, err
	}
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), record.ID); err != nil {
		return EffectiveLibraryConfig{}, err
	}
	paths, err := s.listEffectiveLibraryPaths(ctx, record)
	if err != nil {
		return EffectiveLibraryConfig{}, err
	}
	scanPolicy, err := loadScanPolicy(ctx, s.db, record.ID)
	if err != nil {
		return EffectiveLibraryConfig{}, err
	}
	metadataPolicy, err := loadMetadataPolicy(ctx, s.db, record.ID)
	if err != nil {
		return EffectiveLibraryConfig{}, err
	}
	settingsSvc := settings.NewService(s.db, configMetadataFallback())
	strategy, err := settingsSvc.GetLibraryMetadataStrategy(ctx, record.ID)
	if err != nil {
		return EffectiveLibraryConfig{}, err
	}
	binding := settings.LibraryMetadataProfileBinding{LibraryID: record.ID, MetadataProfileID: strategy.TemplateProfileID, PreferredMetadataLanguage: strategy.PreferredMetadataLanguage, PreferredImageLanguage: strategy.PreferredImageLanguage}
	profile := settings.MetadataProfile{ID: strategy.TemplateProfileID, Name: strategy.TemplateProfileName}
	playbackPolicy, err := loadPlaybackPolicy(ctx, s.db, record.ID)
	if err != nil {
		return EffectiveLibraryConfig{}, err
	}
	subtitlePolicy, err := loadSubtitlePolicy(ctx, s.db, record.ID)
	if err != nil {
		return EffectiveLibraryConfig{}, err
	}
	return EffectiveLibraryConfig{Library: record, Paths: paths, ScanPolicy: scanPolicy, MetadataPolicy: metadataPolicy, ProfileBinding: binding, Profile: profile, PlaybackPolicy: playbackPolicy, SubtitlePolicy: subtitlePolicy}, nil
}

func configMetadataFallback() config.MetadataConfig {
	return config.MetadataConfig{}
}

func (s *Service) listEffectiveLibraryPaths(ctx context.Context, record database.Library) ([]database.LibraryPath, error) {
	var paths []database.LibraryPath
	if err := s.db.WithContext(ctx).Where("library_id = ? AND enabled = ?", record.ID, true).Order("id asc").Find(&paths).Error; err != nil {
		return nil, err
	}
	if len(paths) > 0 {
		return paths, nil
	}
	if record.MediaSourceID == 0 || strings.TrimSpace(record.RootPath) == "" {
		return nil, nil
	}
	return []database.LibraryPath{{LibraryID: record.ID, MediaSourceID: record.MediaSourceID, RootPath: record.RootPath, DisplayName: record.Name, Enabled: true}}, nil
}

func loadScanPolicy(ctx context.Context, db *gorm.DB, libraryID uint) (database.LibraryScanPolicy, error) {
	policy := defaultScanPolicy(libraryID)
	if err := db.WithContext(ctx).Where("library_id = ?", libraryID).First(&policy).Error; err != nil && err != gorm.ErrRecordNotFound {
		return database.LibraryScanPolicy{}, err
	}
	return policy, nil
}

func loadMetadataPolicy(ctx context.Context, db *gorm.DB, libraryID uint) (database.LibraryMetadataPolicy, error) {
	policy := defaultMetadataPolicy(libraryID)
	if err := db.WithContext(ctx).Where("library_id = ?", libraryID).First(&policy).Error; err != nil && err != gorm.ErrRecordNotFound {
		return database.LibraryMetadataPolicy{}, err
	}
	policy.LocalMetadataEnabled = true
	return policy, nil
}

func loadPlaybackPolicy(ctx context.Context, db *gorm.DB, libraryID uint) (database.LibraryPlaybackPolicy, error) {
	policy := defaultPlaybackPolicy(libraryID)
	if err := db.WithContext(ctx).Where("library_id = ?", libraryID).First(&policy).Error; err != nil && err != gorm.ErrRecordNotFound {
		return database.LibraryPlaybackPolicy{}, err
	}
	return policy, nil
}

func loadSubtitlePolicy(ctx context.Context, db *gorm.DB, libraryID uint) (database.LibrarySubtitlePolicy, error) {
	policy := defaultSubtitlePolicy(libraryID)
	if err := db.WithContext(ctx).Where("library_id = ?", libraryID).First(&policy).Error; err != nil && err != gorm.ErrRecordNotFound {
		return database.LibrarySubtitlePolicy{}, err
	}
	return policy, nil
}

func defaultScanPolicy(libraryID uint) database.LibraryScanPolicy {
	return database.LibraryScanPolicy{LibraryID: libraryID, ScannerEnabled: true, RealtimeMonitorEnabled: true, ScheduledRefreshEnabled: true, RefreshIntervalHours: 24, IgnoreHiddenFiles: true, IgnoreFileExtensionsJSON: "[]", InventoryProbeBatchEnabled: true, ConfigurableExclusionRules: true}
}

func defaultMetadataPolicy(libraryID uint) database.LibraryMetadataPolicy {
	return database.LibraryMetadataPolicy{LibraryID: libraryID, LocalMetadataEnabled: true}
}

func defaultPlaybackPolicy(libraryID uint) database.LibraryPlaybackPolicy {
	return database.LibraryPlaybackPolicy{LibraryID: libraryID, ResumeEnabled: true, MinResumePct: 5, MaxResumePct: 90, MinResumeDurationSeconds: 300}
}

func defaultSubtitlePolicy(libraryID uint) database.LibrarySubtitlePolicy {
	return database.LibrarySubtitlePolicy{LibraryID: libraryID, ExternalSidecarsEnabled: true, PreferredLanguagesJSON: "[]", TolerateUnavailableSubtitles: true}
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func stringListFromJSON(value string) []string {
	var parsed []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &parsed); err != nil {
		return []string{}
	}
	return parsed
}

func jsonStringList(values []string) string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return mustJSON(cleaned)
}

func (s *Service) ListLibraryPaths(ctx context.Context, libraryID uint) ([]LibraryPathView, error) {
	config, err := s.EffectiveLibraryConfig(ctx, libraryID)
	if err != nil {
		return nil, err
	}
	return config.PathsView(), nil
}

func (s *Service) LibraryPolicies(ctx context.Context, libraryID uint) (LibraryPoliciesView, error) {
	config, err := s.EffectiveLibraryConfig(ctx, libraryID)
	if err != nil {
		return LibraryPoliciesView{}, err
	}
	return config.PoliciesView(), nil
}

func (s *Service) UpdateLibraryScanPolicy(ctx context.Context, libraryID uint, input LibraryScanPolicyView) (LibraryScanPolicyView, error) {
	if _, err := s.EffectiveLibraryConfig(ctx, libraryID); err != nil {
		return LibraryScanPolicyView{}, err
	}
	updates := database.LibraryScanPolicy{LibraryID: libraryID, ScannerEnabled: input.ScannerEnabled, RealtimeMonitorEnabled: input.RealtimeMonitorEnabled, ScheduledRefreshEnabled: input.ScheduledRefreshEnabled, RefreshIntervalHours: input.RefreshIntervalHours, IgnoreHiddenFiles: input.IgnoreHiddenFiles, IgnoreFileExtensionsJSON: jsonStringList(input.IgnoreFileExtensions), MinFileSizeBytes: input.MinFileSizeBytes, SampleIgnoreSizeBytes: input.SampleIgnoreSizeBytes, InventoryProbeBatchEnabled: input.InventoryProbeBatchEnabled, ConfigurableExclusionRules: input.ConfigurableExclusionRules}
	if updates.RefreshIntervalHours <= 0 {
		updates.RefreshIntervalHours = 24
	}
	if updates.MinFileSizeBytes < 0 {
		updates.MinFileSizeBytes = 0
	}
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), libraryID); err != nil {
		return LibraryScanPolicyView{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.LibraryScanPolicy{}).Where("library_id = ?", libraryID).Updates(map[string]any{"scanner_enabled": updates.ScannerEnabled, "realtime_monitor_enabled": updates.RealtimeMonitorEnabled, "scheduled_refresh_enabled": updates.ScheduledRefreshEnabled, "refresh_interval_hours": updates.RefreshIntervalHours, "ignore_hidden_files": updates.IgnoreHiddenFiles, "ignore_file_extensions_json": updates.IgnoreFileExtensionsJSON, "min_file_size_bytes": updates.MinFileSizeBytes, "sample_ignore_size_bytes": updates.SampleIgnoreSizeBytes, "inventory_probe_batch_enabled": updates.InventoryProbeBatchEnabled, "configurable_exclusion_rules": updates.ConfigurableExclusionRules}).Error; err != nil {
		return LibraryScanPolicyView{}, err
	}
	return LibraryScanPolicyView{ScannerEnabled: updates.ScannerEnabled, RealtimeMonitorEnabled: updates.RealtimeMonitorEnabled, ScheduledRefreshEnabled: updates.ScheduledRefreshEnabled, RefreshIntervalHours: updates.RefreshIntervalHours, IgnoreHiddenFiles: updates.IgnoreHiddenFiles, IgnoreFileExtensions: input.IgnoreFileExtensions, MinFileSizeBytes: updates.MinFileSizeBytes, SampleIgnoreSizeBytes: updates.SampleIgnoreSizeBytes, InventoryProbeBatchEnabled: updates.InventoryProbeBatchEnabled, ConfigurableExclusionRules: updates.ConfigurableExclusionRules}, nil
}

func (s *Service) UpdateLibraryMetadataPolicy(ctx context.Context, libraryID uint, input LibraryMetadataPolicyView) (LibraryMetadataPolicyView, error) {
	if _, err := s.EffectiveLibraryConfig(ctx, libraryID); err != nil {
		return LibraryMetadataPolicyView{}, err
	}
	updates := database.LibraryMetadataPolicy{LibraryID: libraryID, PreferredMetadataLanguage: strings.TrimSpace(input.PreferredMetadataLanguage), PreferredImageLanguage: strings.TrimSpace(input.PreferredImageLanguage), MetadataCountryCode: strings.TrimSpace(input.MetadataCountryCode), LocalMetadataEnabled: true}
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), libraryID); err != nil {
		return LibraryMetadataPolicyView{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.LibraryMetadataPolicy{}).Where("library_id = ?", libraryID).Updates(map[string]any{"preferred_metadata_language": updates.PreferredMetadataLanguage, "preferred_image_language": updates.PreferredImageLanguage, "metadata_country_code": updates.MetadataCountryCode, "local_metadata_enabled": true}).Error; err != nil {
		return LibraryMetadataPolicyView{}, err
	}
	settingsSvc := settings.NewService(s.db, configMetadataFallback())
	strategy, err := settingsSvc.GetLibraryMetadataStrategy(ctx, libraryID)
	if err != nil {
		return LibraryMetadataPolicyView{}, err
	}
	if _, err := settingsSvc.UpdateLibraryMetadataStrategy(ctx, libraryID, settings.UpdateLibraryMetadataStrategyInput{TemplateProfileID: firstNonZero(input.MetadataProfileID, strategy.TemplateProfileID), SearchProviderIDs: strategy.SearchProviderIDs, DetailProviderIDs: strategy.DetailProviderIDs, ImageProviderIDs: strategy.ImageProviderIDs, PeopleProviderIDs: strategy.PeopleProviderIDs, HierarchyProviderIDs: strategy.HierarchyProviderIDs, PreferredMetadataLanguage: updates.PreferredMetadataLanguage, PreferredImageLanguage: updates.PreferredImageLanguage, MetadataCountryCode: updates.MetadataCountryCode}); err != nil {
		return LibraryMetadataPolicyView{}, err
	}
	return input, nil
}

func firstNonZero(values ...uint) uint {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func (s *Service) UpdateLibraryPlaybackPolicy(ctx context.Context, libraryID uint, input LibraryPlaybackPolicyView) (LibraryPlaybackPolicyView, error) {
	if _, err := s.EffectiveLibraryConfig(ctx, libraryID); err != nil {
		return LibraryPlaybackPolicyView{}, err
	}
	updates := database.LibraryPlaybackPolicy{LibraryID: libraryID, ResumeEnabled: input.ResumeEnabled, MinResumePct: input.MinResumePct, MaxResumePct: input.MaxResumePct, MinResumeDurationSeconds: input.MinResumeDurationSeconds}
	if updates.MinResumePct < 0 {
		updates.MinResumePct = 0
	}
	if updates.MaxResumePct <= 0 || updates.MaxResumePct > 100 {
		updates.MaxResumePct = 90
	}
	if updates.MinResumeDurationSeconds < 0 {
		updates.MinResumeDurationSeconds = 0
	}
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), libraryID); err != nil {
		return LibraryPlaybackPolicyView{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.LibraryPlaybackPolicy{}).Where("library_id = ?", libraryID).Updates(map[string]any{"resume_enabled": updates.ResumeEnabled, "min_resume_pct": updates.MinResumePct, "max_resume_pct": updates.MaxResumePct, "min_resume_duration_seconds": updates.MinResumeDurationSeconds}).Error; err != nil {
		return LibraryPlaybackPolicyView{}, err
	}
	return LibraryPlaybackPolicyView{ResumeEnabled: updates.ResumeEnabled, MinResumePct: updates.MinResumePct, MaxResumePct: updates.MaxResumePct, MinResumeDurationSeconds: updates.MinResumeDurationSeconds}, nil
}

func (s *Service) UpdateLibrarySubtitlePolicy(ctx context.Context, libraryID uint, input LibrarySubtitlePolicyView) (LibrarySubtitlePolicyView, error) {
	if _, err := s.EffectiveLibraryConfig(ctx, libraryID); err != nil {
		return LibrarySubtitlePolicyView{}, err
	}
	updates := database.LibrarySubtitlePolicy{LibraryID: libraryID, ExternalSidecarsEnabled: input.ExternalSidecarsEnabled, PreferredLanguagesJSON: jsonStringList(input.PreferredLanguages), RequirePerfectMatch: input.RequirePerfectMatch, SaveWithMedia: input.SaveWithMedia, TolerateUnavailableSubtitles: input.TolerateUnavailableSubtitles, SkipIfEmbeddedSubtitlesPresent: input.SkipIfEmbeddedSubtitlesPresent, SkipIfAudioTrackMatches: input.SkipIfAudioTrackMatches}
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), libraryID); err != nil {
		return LibrarySubtitlePolicyView{}, err
	}
	if err := s.db.WithContext(ctx).Model(&database.LibrarySubtitlePolicy{}).Where("library_id = ?", libraryID).Updates(map[string]any{"external_sidecars_enabled": updates.ExternalSidecarsEnabled, "preferred_languages_json": updates.PreferredLanguagesJSON, "require_perfect_match": updates.RequirePerfectMatch, "save_with_media": updates.SaveWithMedia, "tolerate_unavailable_subtitles": updates.TolerateUnavailableSubtitles, "skip_if_embedded_subtitles_present": updates.SkipIfEmbeddedSubtitlesPresent, "skip_if_audio_track_matches": updates.SkipIfAudioTrackMatches}).Error; err != nil {
		return LibrarySubtitlePolicyView{}, err
	}
	return input, nil
}

func (s *Service) AddLibraryPath(ctx context.Context, libraryID uint, input LibraryPathInput) (LibraryPathView, error) {
	source, err := s.validateLibraryPathInput(ctx, input.MediaSourceID, input.RootPath)
	if err != nil {
		return LibraryPathView{}, err
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	pathRecord := database.LibraryPath{LibraryID: libraryID, MediaSourceID: source.ID, RootPath: normalizePathForProvider(source.Provider, input.RootPath), DisplayName: strings.TrimSpace(input.DisplayName), Enabled: enabled}
	if err := s.db.WithContext(ctx).Create(&pathRecord).Error; err != nil {
		return LibraryPathView{}, err
	}
	if !enabled {
		if err := s.db.WithContext(ctx).Model(&database.LibraryPath{}).Where("id = ?", pathRecord.ID).Update("enabled", false).Error; err != nil {
			return LibraryPathView{}, err
		}
		pathRecord.Enabled = false
	}
	return LibraryPathView{ID: pathRecord.ID, LibraryID: pathRecord.LibraryID, MediaSourceID: pathRecord.MediaSourceID, RootPath: pathRecord.RootPath, DisplayName: pathRecord.DisplayName, Enabled: pathRecord.Enabled}, nil
}

func (s *Service) UpdateLibraryPath(ctx context.Context, libraryID uint, input UpdateLibraryPathInput) (LibraryPathView, error) {
	if input.PathID == 0 {
		return LibraryPathView{}, fmt.Errorf("path id is required")
	}
	var pathRecord database.LibraryPath
	if err := s.db.WithContext(ctx).Where("library_id = ? AND id = ?", libraryID, input.PathID).First(&pathRecord).Error; err != nil {
		return LibraryPathView{}, err
	}
	updates := map[string]any{}
	if input.MediaSourceID != 0 || strings.TrimSpace(input.RootPath) != "" {
		mediaSourceID := pathRecord.MediaSourceID
		if input.MediaSourceID != 0 {
			mediaSourceID = input.MediaSourceID
		}
		rootPath := pathRecord.RootPath
		if strings.TrimSpace(input.RootPath) != "" {
			rootPath = input.RootPath
		}
		source, err := s.validateLibraryPathInput(ctx, mediaSourceID, rootPath)
		if err != nil {
			return LibraryPathView{}, err
		}
		updates["media_source_id"] = source.ID
		updates["root_path"] = normalizePathForProvider(source.Provider, rootPath)
	}
	if strings.TrimSpace(input.DisplayName) != "" {
		updates["display_name"] = strings.TrimSpace(input.DisplayName)
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&database.LibraryPath{}).Where("library_id = ? AND id = ?", libraryID, input.PathID).Updates(updates).Error; err != nil {
			return LibraryPathView{}, err
		}
	}
	if err := s.db.WithContext(ctx).Where("library_id = ? AND id = ?", libraryID, input.PathID).First(&pathRecord).Error; err != nil {
		return LibraryPathView{}, err
	}
	return LibraryPathView{ID: pathRecord.ID, LibraryID: pathRecord.LibraryID, MediaSourceID: pathRecord.MediaSourceID, RootPath: pathRecord.RootPath, DisplayName: pathRecord.DisplayName, Enabled: pathRecord.Enabled}, nil
}

func (s *Service) validateLibraryPathInput(ctx context.Context, mediaSourceID uint, rootPath string) (database.MediaSource, error) {
	if mediaSourceID == 0 {
		return database.MediaSource{}, fmt.Errorf("media_source_id is required")
	}
	if strings.TrimSpace(rootPath) == "" {
		return database.MediaSource{}, fmt.Errorf("root_path is required")
	}
	source, provider, err := s.providerForSource(ctx, mediaSourceID)
	if err != nil {
		return database.MediaSource{}, err
	}
	normalized := normalizePathForProvider(source.Provider, rootPath)
	if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: normalized}); err != nil {
		return database.MediaSource{}, fmt.Errorf("resolve library path %s: %w", normalized, err)
	}
	return source, nil
}
