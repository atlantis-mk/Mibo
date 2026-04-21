package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	storage *providers.Registry
}

type PlaybackSource struct {
	MediaItemID    uint            `json:"media_item_id"`
	MediaFileID    uint            `json:"media_file_id"`
	Title          string          `json:"title"`
	Type           string          `json:"type"`
	Container      string          `json:"container"`
	URL            string          `json:"url"`
	Direct         bool            `json:"direct"`
	SizeBytes      int64           `json:"size_bytes"`
	RuntimeSeconds *int            `json:"runtime_seconds,omitempty"`
	VideoCodec     string          `json:"video_codec,omitempty"`
	Width          *int            `json:"width,omitempty"`
	Height         *int            `json:"height,omitempty"`
	AudioTracks    []Track         `json:"audio_tracks,omitempty"`
	SubtitleTracks []Track         `json:"subtitle_tracks,omitempty"`
	Checks         []PlaybackCheck `json:"checks"`
	Playable       bool            `json:"playable"`
	Decision       PlaybackDecision `json:"decision"`
}

type FileLink struct {
	MediaFileID uint            `json:"media_file_id"`
	StoragePath string          `json:"storage_path"`
	URL         string          `json:"url"`
	Checks      []PlaybackCheck `json:"checks"`
	Playable    bool            `json:"playable"`
}

type PlaybackCheck struct {
	Code    string `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Track struct {
	Codec    string `json:"codec"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Channels int    `json:"channels,omitempty"`
}

func NewService(db *gorm.DB, registry *providers.Registry) *Service {
	return &Service{db: db, storage: registry}
}

func (s *Service) Status() string {
	return "active"
}

func (s *Service) GetPlaybackSource(ctx context.Context, req PlaybackRequest) (PlaybackSource, error) {
	item, files, err := s.loadMediaItemFiles(ctx, req.MediaItemID)
	if err != nil {
		return PlaybackSource{}, err
	}

	selected, selectedBy, err := selectPlaybackFile(files, req.PreferredFileID, req.ClientProfile)
	if err != nil {
		return PlaybackSource{}, err
	}

	fileLink, err := s.GetFileLink(ctx, selected.ID)
	if err != nil {
		return PlaybackSource{}, err
	}

	audioTracks, err := parseTrackList(selected.AudioTracksJSON)
	if err != nil {
		return PlaybackSource{}, err
	}
	subtitleTracks, err := parseTrackList(selected.SubtitleTracksJSON)
	if err != nil {
		return PlaybackSource{}, err
	}

	checks := append([]PlaybackCheck{}, fileLink.Checks...)
	checks = append(checks, buildMediaInfoCheck(selected))
	directDecision := assessDirectPlay(selected, req.ClientProfile)
	if !fileLink.Playable {
		directDecision.direct = false
		directDecision.reasons = append([]DecisionReason{{
			Code:     "source_unavailable",
			Category: "availability",
			Message:  "media source is unavailable",
		}}, directDecision.reasons...)
	}

	base := PlaybackSource{
		MediaItemID:    item.ID,
		MediaFileID:    selected.ID,
		Title:          item.Title,
		Type:           item.Type,
		Container:      selected.Container,
		SizeBytes:      selected.SizeBytes,
		RuntimeSeconds: item.RuntimeSeconds,
		VideoCodec:     selected.VideoCodec,
		Width:          selected.Width,
		Height:         selected.Height,
		AudioTracks:    audioTracks,
		SubtitleTracks: subtitleTracks,
		Checks:         checks,
	}

	if directDecision.direct {
		base.URL = fileLink.URL
		base.Direct = true
		base.Playable = true
		base.Decision = PlaybackDecision{
			Kind:          "direct",
			ClientProfile: req.ClientProfile,
			SelectedBy:    selectedBy,
			Reasons:       directDecision.reasons,
		}
		return base, nil
	}

	if req.AllowHLSFallback && fileLink.Playable {
		base.Container = "m3u8"
		base.URL = fmt.Sprintf("/api/v1/media-files/%d/hls/index.m3u8", selected.ID)
		base.Direct = false
		base.Playable = true
		base.Decision = PlaybackDecision{
			Kind:          "fallback",
			ClientProfile: req.ClientProfile,
			SelectedBy:    selectedBy,
			FallbackKind:  "hls",
			Reasons: append(append([]DecisionReason{}, directDecision.reasons...), DecisionReason{
				Code:     "hls_fallback_selected",
				Category: "fallback",
				Message:  "switched to HLS fallback for this client profile",
			}),
		}
		return base, nil
	}

	base.Direct = false
	base.Playable = false
	base.URL = ""
	base.Decision = PlaybackDecision{
		Kind:          "unplayable",
		ClientProfile: req.ClientProfile,
		SelectedBy:    selectedBy,
		Reasons: append(append([]DecisionReason{}, directDecision.reasons...), DecisionReason{
			Code:     "no_supported_playback_path",
			Category: "fallback",
			Message:  "no supported playback path is available for this client profile",
		}),
	}
	return base, nil
}

func (s *Service) GetFileLink(ctx context.Context, mediaFileID uint) (FileLink, error) {
	var file database.MediaFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaFileID).
		First(&file).Error; err != nil {
		return FileLink{}, err
	}

	provider, err := s.providerForMediaFile(ctx, file.ID)
	if err != nil {
		return FileLink{}, err
	}

	checks := []PlaybackCheck{}
	object, err := provider.Get(ctx, storage.GetRequest{Path: file.StoragePath})
	if err != nil {
		checks = append(checks, PlaybackCheck{Code: "file_exists", Status: "fail", Message: err.Error()})
		return FileLink{MediaFileID: file.ID, StoragePath: file.StoragePath, Checks: checks, Playable: false}, nil
	}
	if object.IsDir {
		checks = append(checks, PlaybackCheck{Code: "file_exists", Status: "fail", Message: "selected media file is a directory"})
		return FileLink{MediaFileID: file.ID, StoragePath: file.StoragePath, Checks: checks, Playable: false}, nil
	}
	checks = append(checks, PlaybackCheck{Code: "file_exists", Status: "pass", Message: "media file resolved"})
	checks = append(checks, PlaybackCheck{Code: "file_access", Status: "pass", Message: "media stream endpoint available"})

	return FileLink{
		MediaFileID: file.ID,
		StoragePath: file.StoragePath,
		URL:         fmt.Sprintf("/api/v1/media-files/%d/stream", file.ID),
		Checks:      checks,
		Playable:    isPlayable(checks),
	}, nil
}

func (s *Service) providerForMediaFile(ctx context.Context, mediaFileID uint) (storage.Provider, error) {
	var file database.MediaFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaFileID).
		First(&file).Error; err != nil {
		return nil, err
	}

	var libraryRecord database.Library
	if err := s.db.WithContext(ctx).First(&libraryRecord, file.LibraryID).Error; err != nil {
		return nil, err
	}

	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, libraryRecord.MediaSourceID).Error; err != nil {
		return nil, err
	}

	return s.storage.BuildForSource(source)
}

func (s *Service) loadMediaItemFiles(ctx context.Context, mediaItemID uint) (database.MediaItem, []database.MediaFile, error) {
	var item database.MediaItem
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaItemID).
		First(&item).Error; err != nil {
		return database.MediaItem{}, nil, err
	}

	var files []database.MediaFile
	if err := s.db.WithContext(ctx).
		Where("media_item_id = ? AND deleted_at IS NULL", mediaItemID).
		Find(&files).Error; err != nil {
		return database.MediaItem{}, nil, err
	}
	if len(files) == 0 {
		return database.MediaItem{}, nil, fmt.Errorf("no playable files found for media item %d", mediaItemID)
	}

	return item, files, nil
}

func selectPlaybackFile(files []database.MediaFile, preferredFileID uint, clientProfile ClientProfile) (database.MediaFile, string, error) {
	if preferredFileID != 0 {
		for _, file := range files {
			if file.ID == preferredFileID {
				return file, "preferred_file", nil
			}
		}
		return database.MediaFile{}, "", fmt.Errorf("preferred media file %d is not available", preferredFileID)
	}

	sorted := append([]database.MediaFile(nil), files...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := scoreFile(sorted[i], clientProfile)
		right := scoreFile(sorted[j], clientProfile)
		if left != right {
			return left > right
		}
		leftPixels := resolutionPixels(sorted[i])
		rightPixels := resolutionPixels(sorted[j])
		if leftPixels != rightPixels {
			return leftPixels > rightPixels
		}
		leftBitrate := int64Value(sorted[i].BitRate)
		rightBitrate := int64Value(sorted[j].BitRate)
		if leftBitrate != rightBitrate {
			return leftBitrate > rightBitrate
		}
		if sorted[i].SizeBytes != sorted[j].SizeBytes {
			return sorted[i].SizeBytes > sorted[j].SizeBytes
		}
		return sorted[i].ID < sorted[j].ID
	})

	return sorted[0], "profile_filter_then_rank", nil
}

func scoreFile(file database.MediaFile, clientProfile ClientProfile) int {
	decision := assessDirectPlay(file, clientProfile)
	score := 0
	if decision.direct {
		score += 100
		if file.ProbeStatus == probe.StatusReady {
			score += 20
		} else {
			score += 10
		}
	} else if file.ProbeStatus == probe.StatusReady {
		score += 5
	}
	if file.VideoCodec != "" {
		score += 3
	}
	if file.Width != nil && file.Height != nil {
		score += 2
	}
	return score
}

type directPlayAssessment struct {
	direct  bool
	reasons []DecisionReason
}

func assessDirectPlay(file database.MediaFile, clientProfile ClientProfile) directPlayAssessment {
	container := normalizeContainer(file.Container)
	codec := strings.ToLower(strings.TrimSpace(file.VideoCodec))

	if file.ProbeStatus == probe.StatusReady {
		if supportsDirectPlay(clientProfile, container, codec) {
			return directPlayAssessment{direct: true, reasons: []DecisionReason{{
				Code:     "direct_profile_match",
				Category: "profile",
				Message:  "media format is directly playable for this client profile",
			}}}
		}

		reasons := []DecisionReason{}
		if !supportsContainer(clientProfile, container) {
			reasons = append(reasons, DecisionReason{
				Code:     "direct_play_unsupported_container",
				Category: "profile",
				Message:  fmt.Sprintf("container %s is not directly supported for this client profile", fallbackValue(container, "unknown")),
			})
		}
		if !supportsCodec(clientProfile, container, codec) {
			reasons = append(reasons, DecisionReason{
				Code:     "direct_play_unsupported_video_codec",
				Category: "profile",
				Message:  fmt.Sprintf("video codec %s is not directly supported for this client profile", fallbackValue(codec, "unknown")),
			})
		}
		if len(reasons) == 0 {
			reasons = append(reasons, DecisionReason{
				Code:     "direct_play_not_supported",
				Category: "profile",
				Message:  "media format is not directly supported for this client profile",
			})
		}
		return directPlayAssessment{reasons: reasons}
	}

	if isOptimisticContainer(clientProfile, container) {
		return directPlayAssessment{direct: true, reasons: []DecisionReason{{
			Code:     "probe_missing_assumed_compatible",
			Category: "probe",
			Message:  "probe data is unavailable, but this common container is allowed for direct playback",
		}}}
	}

	return directPlayAssessment{reasons: []DecisionReason{{
		Code:     "probe_missing_unknown_container",
		Category: "probe",
		Message:  "probe data is unavailable and direct compatibility cannot be confirmed for this container",
	}}}
}

func supportsDirectPlay(clientProfile ClientProfile, container, codec string) bool {
	if !supportsContainer(clientProfile, container) {
		return false
	}
	return supportsCodec(clientProfile, container, codec)
}

func supportsContainer(clientProfile ClientProfile, container string) bool {
	switch clientProfile {
	case ClientProfileWeb:
		return container == "mp4" || container == "m4v" || container == "webm"
	case ClientProfileMobile, ClientProfileTV:
		return container == "mp4" || container == "m4v"
	default:
		return false
	}
}

func supportsCodec(clientProfile ClientProfile, container, codec string) bool {
	switch clientProfile {
	case ClientProfileWeb:
		return (isMP4Family(container) && codec == "h264") || (container == "webm" && codec == "vp9")
	case ClientProfileMobile, ClientProfileTV:
		return isMP4Family(container) && codec == "h264"
	default:
		return false
	}
}

func isOptimisticContainer(clientProfile ClientProfile, container string) bool {
	switch clientProfile {
	case ClientProfileWeb:
		return container == "mp4" || container == "m4v" || container == "webm"
	case ClientProfileMobile, ClientProfileTV:
		return container == "mp4" || container == "m4v"
	default:
		return false
	}
}

func normalizeContainer(container string) string {
	value := strings.ToLower(strings.TrimSpace(container))
	return strings.TrimPrefix(value, ".")
}

func isMP4Family(container string) bool {
	return container == "mp4" || container == "m4v"
}

func resolutionPixels(file database.MediaFile) int {
	if file.Width == nil || file.Height == nil {
		return 0
	}
	return *file.Width * *file.Height
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func fallbackValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func buildMediaInfoCheck(file database.MediaFile) PlaybackCheck {
	switch file.ProbeStatus {
	case probe.StatusReady:
		return PlaybackCheck{Code: "media_info", Status: "pass", Message: "media technical info available"}
	case probe.StatusUnavailable:
		return PlaybackCheck{Code: "media_info", Status: "warn", Message: "technical info unavailable"}
	case probe.StatusError:
		return PlaybackCheck{Code: "media_info", Status: "warn", Message: "technical info probe failed"}
	default:
		return PlaybackCheck{Code: "media_info", Status: "warn", Message: "technical info pending"}
	}
}

func isPlayable(checks []PlaybackCheck) bool {
	for _, check := range checks {
		if check.Status == "fail" {
			return false
		}
	}
	return true
}

func parseTrackList(input string) ([]Track, error) {
	if input == "" {
		return nil, nil
	}
	var tracks []Track
	if err := json.Unmarshal([]byte(input), &tracks); err != nil {
		return nil, err
	}
	return tracks, nil
}
