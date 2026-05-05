package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog/seriesplayback"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
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
	ItemID         uint             `json:"item_id,omitempty"`
	AssetID        uint             `json:"asset_id,omitempty"`
	FileID         uint             `json:"file_id,omitempty"`
	Title          string           `json:"title"`
	Type           string           `json:"type"`
	Container      string           `json:"container"`
	URL            string           `json:"url"`
	Direct         bool             `json:"direct"`
	SizeBytes      int64            `json:"size_bytes"`
	RuntimeSeconds *int             `json:"runtime_seconds,omitempty"`
	QualityLabel   string           `json:"quality_label,omitempty"`
	Edition        string           `json:"edition,omitempty"`
	VideoCodec     string           `json:"video_codec,omitempty"`
	Width          *int             `json:"width,omitempty"`
	Height         *int             `json:"height,omitempty"`
	AudioTracks    []Track          `json:"audio_tracks,omitempty"`
	SubtitleTracks []Track          `json:"subtitle_tracks,omitempty"`
	Checks         []PlaybackCheck  `json:"checks"`
	Playable       bool             `json:"playable"`
	Decision       PlaybackDecision `json:"decision"`
}

type FileLink struct {
	FileID      uint            `json:"file_id,omitempty"`
	AssetID     uint            `json:"asset_id,omitempty"`
	StoragePath string          `json:"storage_path"`
	URL         string          `json:"url"`
	Checks      []PlaybackCheck `json:"checks"`
	Playable    bool            `json:"playable"`
}

type mediaInfo struct {
	Container   string
	ProbeStatus string
	VideoCodec  string
	BitRate     *int64
	Width       *int
	Height      *int
}

type PlaybackCheck struct {
	Code    string `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Track struct {
	Codec     string `json:"codec"`
	Language  string `json:"language"`
	Title     string `json:"title"`
	Channels  int    `json:"channels,omitempty"`
	FileID    uint   `json:"file_id,omitempty"`
	URL       string `json:"url,omitempty"`
	External  bool   `json:"external,omitempty"`
	Available *bool  `json:"available,omitempty"`
}

func NewService(db *gorm.DB, registry *providers.Registry) *Service {
	return &Service{db: db, storage: registry}
}

func (s *Service) Status() string {
	return "active"
}

func (s *Service) GetPlaybackSource(ctx context.Context, req PlaybackRequest) (PlaybackSource, error) {
	return s.getCatalogPlaybackSource(ctx, req)
}

func (s *Service) GetInventoryFilePlaybackSource(ctx context.Context, fileID uint, clientProfile ClientProfile) (PlaybackSource, error) {
	var file database.InventoryFile
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", fileID).First(&file).Error; err != nil {
		return PlaybackSource{}, err
	}
	fileLink, err := s.GetInventoryFileLink(ctx, file.ID)
	if err != nil {
		return PlaybackSource{}, fmt.Errorf("load inventory file link: %w", err)
	}
	streams, err := s.loadMediaStreamsForFile(ctx, file.ID)
	if err != nil {
		return PlaybackSource{}, err
	}
	pseudoFile, audioTracks, subtitleTracks := inventoryFileMediaInfo(file, streams)
	checks := append([]PlaybackCheck{}, fileLink.Checks...)
	checks = append(checks, buildMediaInfoCheck(pseudoFile))
	directDecision := assessDirectPlay(pseudoFile, clientProfile)
	if !fileLink.Playable {
		directDecision.direct = false
		directDecision.reasons = append([]DecisionReason{{Code: "source_unavailable", Category: "availability", Message: "media source is unavailable"}}, directDecision.reasons...)
	}
	source := PlaybackSource{
		FileID:         file.ID,
		Title:          titleFromInventoryPath(file.StoragePath),
		Type:           "inventory_file",
		Container:      file.Container,
		URL:            fileLink.URL,
		Direct:         fileLink.Playable,
		SizeBytes:      file.SizeBytes,
		VideoCodec:     pseudoFile.VideoCodec,
		Width:          pseudoFile.Width,
		Height:         pseudoFile.Height,
		AudioTracks:    audioTracks,
		SubtitleTracks: subtitleTracks,
		Checks:         checks,
		Playable:       fileLink.Playable,
	}
	if source.Playable {
		source.Decision = PlaybackDecision{Kind: "direct", ClientProfile: clientProfile, SelectedBy: "inventory_file", Reasons: directDecision.reasons}
		return source, nil
	}
	source.URL = ""
	source.Direct = false
	source.Decision = PlaybackDecision{Kind: "unplayable", ClientProfile: clientProfile, SelectedBy: "inventory_file", Reasons: append(append([]DecisionReason{}, directDecision.reasons...), DecisionReason{Code: "no_supported_playback_path", Category: "fallback", Message: "no supported playback path is available for this inventory file"})}
	return source, nil
}

type catalogPlaybackCandidate struct {
	Asset   database.MediaAsset
	File    database.InventoryFile
	Files   map[uint]database.InventoryFile
	Streams []database.MediaStream
}

func (s *Service) getCatalogPlaybackSource(ctx context.Context, req PlaybackRequest) (PlaybackSource, error) {
	var item database.CatalogItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.ItemID).First(&item).Error; err != nil {
		return PlaybackSource{}, err
	}
	if item.Type == "series" {
		target, err := seriesplayback.Select(ctx, s.db, item.ID, req.UserID)
		if err != nil {
			return PlaybackSource{}, err
		}
		if target == nil {
			return PlaybackSource{
				ItemID:   item.ID,
				Title:    item.Title,
				Type:     item.Type,
				Playable: false,
				Decision: PlaybackDecision{Kind: "unplayable", ClientProfile: req.ClientProfile, SelectedBy: "series_target", Reasons: []DecisionReason{{Code: "series_has_no_playable_episode", Category: "availability", Message: "series has no locally playable episode target"}}},
			}, nil
		}
		if target.AssetID != nil && req.AssetID == 0 {
			req.AssetID = *target.AssetID
		}
		req.ItemID = target.EpisodeID
		item = database.CatalogItem{}
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.ItemID).First(&item).Error; err != nil {
			return PlaybackSource{}, fmt.Errorf("load resolved series episode: %w", err)
		}
	}
	candidates, err := s.loadCatalogPlaybackCandidates(ctx, req.ItemID)
	if err != nil {
		return PlaybackSource{}, fmt.Errorf("load catalog playback candidates: %w", err)
	}
	if len(candidates) == 0 {
		return PlaybackSource{
			ItemID:   item.ID,
			Title:    item.Title,
			Type:     item.Type,
			Playable: false,
			Decision: PlaybackDecision{Kind: "unplayable", ClientProfile: req.ClientProfile, SelectedBy: "no_asset", Reasons: []DecisionReason{{Code: "no_available_asset", Category: "availability", Message: "no playable asset is linked to this catalog item"}}},
		}, nil
	}

	selected, selectedBy, ok := selectCatalogPlaybackCandidate(candidates, req.AssetID)
	if !ok {
		return PlaybackSource{
			ItemID:   item.ID,
			Title:    item.Title,
			Type:     item.Type,
			Playable: false,
			Decision: PlaybackDecision{Kind: "unplayable", ClientProfile: req.ClientProfile, SelectedBy: "asset_filter", Reasons: []DecisionReason{{Code: "asset_not_available", Category: "availability", Message: "requested asset is not linked to this catalog item"}}},
		}, nil
	}

	fileLink, err := s.GetInventoryFileLink(ctx, selected.File.ID)
	if err != nil {
		return PlaybackSource{}, fmt.Errorf("load inventory file link: %w", err)
	}
	pseudoFile, audioTracks, subtitleTracks := inventoryCandidateMediaInfo(selected)
	subtitlePolicy, err := s.subtitlePolicy(ctx, item.LibraryID)
	if err != nil {
		return PlaybackSource{}, err
	}
	subtitleTracks = s.applySubtitlePolicy(s.enrichExternalSubtitleTracks(ctx, subtitleTracks), subtitlePolicy)
	checks := append([]PlaybackCheck{}, fileLink.Checks...)
	checks = append(checks, buildMediaInfoCheck(pseudoFile))
	directDecision := assessDirectPlay(pseudoFile, req.ClientProfile)
	if !fileLink.Playable {
		directDecision.direct = false
		directDecision.reasons = append([]DecisionReason{{Code: "source_unavailable", Category: "availability", Message: "media source is unavailable"}}, directDecision.reasons...)
	}

	base := PlaybackSource{
		ItemID:         item.ID,
		AssetID:        selected.Asset.ID,
		FileID:         selected.File.ID,
		Title:          item.Title,
		Type:           item.Type,
		Container:      selected.File.Container,
		URL:            fileLink.URL,
		Direct:         fileLink.Playable,
		SizeBytes:      selected.File.SizeBytes,
		RuntimeSeconds: item.RuntimeSeconds,
		QualityLabel:   selected.Asset.QualityLabel,
		Edition:        selected.Asset.Edition,
		VideoCodec:     pseudoFile.VideoCodec,
		Width:          pseudoFile.Width,
		Height:         pseudoFile.Height,
		AudioTracks:    audioTracks,
		SubtitleTracks: subtitleTracks,
		Checks:         checks,
		Playable:       fileLink.Playable,
	}
	if base.Playable {
		base.Decision = PlaybackDecision{Kind: "direct", ClientProfile: req.ClientProfile, SelectedBy: selectedBy, Reasons: directDecision.reasons}
		return base, nil
	}
	base.URL = ""
	base.Direct = false
	base.Decision = PlaybackDecision{Kind: "unplayable", ClientProfile: req.ClientProfile, SelectedBy: selectedBy, Reasons: append(append([]DecisionReason{}, directDecision.reasons...), DecisionReason{Code: "no_supported_playback_path", Category: "fallback", Message: "no supported playback path is available for this asset"})}
	return base, nil
}

func (s *Service) GetAssetLink(ctx context.Context, assetID uint) (FileLink, error) {
	var link database.AssetFile
	if err := s.db.WithContext(ctx).
		Where("asset_id = ? AND role = ?", assetID, "source").
		Order("part_index asc, id asc").
		First(&link).Error; err != nil {
		return FileLink{}, err
	}
	fileLink, err := s.GetInventoryFileLink(ctx, link.FileID)
	if err != nil {
		return FileLink{}, err
	}
	fileLink.AssetID = assetID
	return fileLink, nil
}

func (s *Service) GetInventoryFileLink(ctx context.Context, fileID uint) (FileLink, error) {
	var file database.InventoryFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", fileID).
		First(&file).Error; err != nil {
		return FileLink{}, err
	}

	provider, err := s.providerForInventoryFile(ctx, file.ID)
	if err != nil {
		return FileLink{}, err
	}

	checks := []PlaybackCheck{}
	object, err := provider.Get(ctx, storage.GetRequest{Path: file.StoragePath})
	if err != nil {
		checks = append(checks, PlaybackCheck{Code: "file_exists", Status: "fail", Message: err.Error()})
		return FileLink{FileID: file.ID, StoragePath: file.StoragePath, Checks: checks, Playable: false}, nil
	}
	if object.IsDir {
		checks = append(checks, PlaybackCheck{Code: "file_exists", Status: "fail", Message: "selected inventory file is a directory"})
		return FileLink{FileID: file.ID, StoragePath: file.StoragePath, Checks: checks, Playable: false}, nil
	}
	checks = append(checks, PlaybackCheck{Code: "file_exists", Status: "pass", Message: "inventory file resolved"})
	checks = append(checks, PlaybackCheck{Code: "file_access", Status: "pass", Message: "inventory stream endpoint available"})
	return FileLink{FileID: file.ID, StoragePath: file.StoragePath, URL: fmt.Sprintf("/api/v1/inventory-files/%d/stream", file.ID), Checks: checks, Playable: isPlayable(checks)}, nil
}

func (s *Service) providerForInventoryFile(ctx context.Context, fileID uint) (storage.Provider, error) {
	var file database.InventoryFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", fileID).
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

func (s *Service) loadCatalogPlaybackCandidates(ctx context.Context, itemID uint) ([]catalogPlaybackCandidate, error) {
	var links []database.AssetItem
	if err := s.db.WithContext(ctx).
		Where("item_id = ?", itemID).
		Order("role asc, segment_index asc, id asc").
		Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}
	assetIDs := make([]uint, 0, len(links))
	seen := map[uint]struct{}{}
	for _, link := range links {
		if _, ok := seen[link.AssetID]; ok {
			continue
		}
		seen[link.AssetID] = struct{}{}
		assetIDs = append(assetIDs, link.AssetID)
	}
	var assets []database.MediaAsset
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", assetIDs).Find(&assets).Error; err != nil {
		return nil, err
	}
	assetByID := make(map[uint]database.MediaAsset, len(assets))
	for _, asset := range assets {
		assetByID[asset.ID] = asset
	}
	var assetFiles []database.AssetFile
	if err := s.db.WithContext(ctx).
		Where("asset_id IN ? AND role IN ?", assetIDs, []string{inventory.FileRoleSource, inventory.FileRoleSubtitle}).
		Order("asset_id asc, role asc, part_index asc, id asc").
		Find(&assetFiles).Error; err != nil {
		return nil, err
	}
	fileIDs := make([]uint, 0, len(assetFiles))
	firstFileByAsset := make(map[uint]uint, len(assetFiles))
	fileIDsByAsset := make(map[uint][]uint, len(assetFiles))
	fileIDSet := make(map[uint]struct{}, len(assetFiles))
	for _, link := range assetFiles {
		if link.Role == inventory.FileRoleSource {
			if _, ok := firstFileByAsset[link.AssetID]; !ok {
				firstFileByAsset[link.AssetID] = link.FileID
			}
		}
		fileIDsByAsset[link.AssetID] = append(fileIDsByAsset[link.AssetID], link.FileID)
		if _, ok := fileIDSet[link.FileID]; !ok {
			fileIDSet[link.FileID] = struct{}{}
			fileIDs = append(fileIDs, link.FileID)
		}
	}
	var files []database.InventoryFile
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", fileIDs).Find(&files).Error; err != nil {
		return nil, err
	}
	fileByID := make(map[uint]database.InventoryFile, len(files))
	for _, file := range files {
		fileByID[file.ID] = file
	}
	var streams []database.MediaStream
	if len(fileIDs) > 0 {
		if err := s.db.WithContext(ctx).Where("file_id IN ?", fileIDs).Order("file_id asc, stream_index asc").Find(&streams).Error; err != nil {
			return nil, err
		}
	}
	streamsByFile := make(map[uint][]database.MediaStream, len(fileIDs))
	for _, stream := range streams {
		streamsByFile[stream.FileID] = append(streamsByFile[stream.FileID], stream)
	}
	result := make([]catalogPlaybackCandidate, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		asset, ok := assetByID[assetID]
		if !ok {
			continue
		}
		fileID, ok := firstFileByAsset[assetID]
		if !ok {
			continue
		}
		file, ok := fileByID[fileID]
		if !ok {
			continue
		}
		candidateFiles := make(map[uint]database.InventoryFile)
		candidateStreams := make([]database.MediaStream, 0)
		for _, linkedFileID := range fileIDsByAsset[assetID] {
			linkedFile, ok := fileByID[linkedFileID]
			if !ok {
				continue
			}
			candidateFiles[linkedFileID] = linkedFile
			candidateStreams = append(candidateStreams, streamsByFile[linkedFileID]...)
		}
		result = append(result, catalogPlaybackCandidate{Asset: asset, File: file, Files: candidateFiles, Streams: candidateStreams})
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := catalogPlaybackRank(result[i])
		right := catalogPlaybackRank(result[j])
		if left != right {
			return left > right
		}
		return result[i].Asset.ID < result[j].Asset.ID
	})
	return result, nil
}

func selectCatalogPlaybackCandidate(candidates []catalogPlaybackCandidate, preferredAssetID uint) (catalogPlaybackCandidate, string, bool) {
	if preferredAssetID != 0 {
		for _, candidate := range candidates {
			if candidate.Asset.ID == preferredAssetID {
				return candidate, "preferred_asset", true
			}
		}
		return catalogPlaybackCandidate{}, "preferred_asset", false
	}
	if len(candidates) == 0 {
		return catalogPlaybackCandidate{}, "no_asset", false
	}
	return candidates[0], "asset_rank", true
}

func catalogPlaybackRank(candidate catalogPlaybackCandidate) int {
	score := 0
	if strings.TrimSpace(candidate.Asset.Status) == "available" {
		score += 100
	}
	if strings.TrimSpace(candidate.Asset.AssetType) == "main" {
		score += 20
	}
	if strings.TrimSpace(candidate.Asset.ProbeStatus) == probe.StatusReady {
		score += 10
	}
	pseudo, _, _ := inventoryCandidateMediaInfo(candidate)
	if assessDirectPlay(pseudo, ClientProfileWeb).direct {
		score += 5
	}
	score += resolutionPixels(pseudo)
	return score
}

func inventoryCandidateMediaInfo(candidate catalogPlaybackCandidate) (mediaInfo, []Track, []Track) {
	pseudo := mediaInfo{Container: candidate.File.Container, ProbeStatus: candidate.Asset.ProbeStatus}
	audioTracks := make([]Track, 0)
	subtitleTracks := make([]Track, 0)
	for _, stream := range candidate.Streams {
		switch strings.ToLower(strings.TrimSpace(stream.StreamType)) {
		case "video":
			if pseudo.VideoCodec == "" {
				pseudo.VideoCodec = strings.TrimSpace(stream.Codec)
				pseudo.Width = stream.Width
				pseudo.Height = stream.Height
				pseudo.BitRate = stream.BitRate
			}
		case "audio":
			audioTracks = append(audioTracks, Track{Codec: strings.TrimSpace(stream.Codec), Language: strings.TrimSpace(stream.Language), Title: strings.TrimSpace(stream.Title), Channels: intValue(stream.Channels)})
		case "subtitle":
			subtitleTracks = append(subtitleTracks, buildSubtitleTrack(candidate, stream))
		}
	}
	return pseudo, audioTracks, subtitleTracks
}

func inventoryFileMediaInfo(file database.InventoryFile, streams []database.MediaStream) (mediaInfo, []Track, []Track) {
	return inventoryCandidateMediaInfo(catalogPlaybackCandidate{Asset: database.MediaAsset{ProbeStatus: probe.StatusPending}, File: file, Streams: streams})
}

func (s *Service) loadMediaStreamsForFile(ctx context.Context, fileID uint) ([]database.MediaStream, error) {
	var streams []database.MediaStream
	if err := s.db.WithContext(ctx).Where("file_id = ?", fileID).Order("stream_index asc").Find(&streams).Error; err != nil {
		return nil, err
	}
	return streams, nil
}

func titleFromInventoryPath(storagePath string) string {
	trimmed := strings.TrimSpace(storagePath)
	if trimmed == "" {
		return "未整理媒体"
	}
	parts := strings.Split(trimmed, "/")
	base := strings.TrimSpace(parts[len(parts)-1])
	if base == "" {
		return trimmed
	}
	if dot := strings.LastIndex(base, "."); dot > 0 {
		base = base[:dot]
	}
	base = strings.NewReplacer(".", " ", "_", " ").Replace(base)
	return strings.TrimSpace(base)
}

func buildSubtitleTrack(candidate catalogPlaybackCandidate, stream database.MediaStream) Track {
	track := Track{Codec: strings.TrimSpace(stream.Codec), Language: strings.TrimSpace(stream.Language), Title: strings.TrimSpace(stream.Title), Channels: intValue(stream.Channels)}
	if !playbackStreamDispositionBool(stream.DispositionJSON, "external") {
		return track
	}
	file, ok := candidate.Files[stream.FileID]
	if !ok {
		available := false
		track.FileID = stream.FileID
		track.External = true
		track.Available = &available
		return track
	}
	available := strings.TrimSpace(file.Status) == inventory.FileStatusAvailable && file.DeletedAt == nil
	track.FileID = stream.FileID
	track.External = true
	track.Available = &available
	if available {
		track.URL = fmt.Sprintf("/api/v1/inventory-files/%d/stream", stream.FileID)
	}
	return track
}

func (s *Service) enrichExternalSubtitleTracks(ctx context.Context, tracks []Track) []Track {
	for i := range tracks {
		if !tracks[i].External || tracks[i].FileID == 0 {
			continue
		}
		link, err := s.GetInventoryFileLink(ctx, tracks[i].FileID)
		available := err == nil && link.Playable
		tracks[i].Available = &available
		if available {
			tracks[i].URL = link.URL
		} else {
			tracks[i].URL = ""
		}
	}
	return tracks
}

func (s *Service) subtitlePolicy(ctx context.Context, libraryID uint) (database.LibrarySubtitlePolicy, error) {
	if err := database.EnsureLibraryPolicyDefaults(s.db.WithContext(ctx), libraryID); err != nil {
		return database.LibrarySubtitlePolicy{}, err
	}
	var policy database.LibrarySubtitlePolicy
	if err := s.db.WithContext(ctx).Where("library_id = ?", libraryID).First(&policy).Error; err != nil {
		return database.LibrarySubtitlePolicy{}, err
	}
	return policy, nil
}

func (s *Service) applySubtitlePolicy(tracks []Track, policy database.LibrarySubtitlePolicy) []Track {
	if !policy.ExternalSidecarsEnabled {
		filtered := tracks[:0]
		for _, track := range tracks {
			if !track.External {
				filtered = append(filtered, track)
			}
		}
		tracks = filtered
	}
	if !policy.TolerateUnavailableSubtitles {
		filtered := tracks[:0]
		for _, track := range tracks {
			if track.Available != nil && !*track.Available {
				continue
			}
			filtered = append(filtered, track)
		}
		tracks = filtered
	}
	preferred := stringListFromJSON(policy.PreferredLanguagesJSON)
	if len(preferred) == 0 {
		return tracks
	}
	preferredSet := map[string]struct{}{}
	for _, language := range preferred {
		preferredSet[strings.ToLower(strings.TrimSpace(language))] = struct{}{}
	}
	filtered := make([]Track, 0, len(tracks))
	for _, track := range tracks {
		language := strings.ToLower(strings.TrimSpace(track.Language))
		if _, ok := preferredSet[language]; ok {
			filtered = append(filtered, track)
		}
	}
	if len(filtered) == 0 {
		return tracks
	}
	return filtered
}

func stringListFromJSON(value string) []string {
	var parsed []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &parsed); err != nil {
		return nil
	}
	return parsed
}

func playbackStreamDispositionBool(raw string, key string) bool {
	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return false
	}
	switch value := values[key].(type) {
	case bool:
		return value
	case float64:
		return value != 0
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true") || strings.TrimSpace(value) == "1"
	default:
		return false
	}
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

type directPlayAssessment struct {
	direct  bool
	reasons []DecisionReason
}

func assessDirectPlay(file mediaInfo, clientProfile ClientProfile) directPlayAssessment {
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

func resolutionPixels(file mediaInfo) int {
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

func buildMediaInfoCheck(file mediaInfo) PlaybackCheck {
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
