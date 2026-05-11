package playback

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/probe"
	"gorm.io/gorm"
)

type resourcePlaybackCandidate struct {
	Resource  database.Resource
	File      database.InventoryFile
	Files     map[uint]database.InventoryFile
	Streams   []database.MediaStream
	Link      database.ResourceMetadataLink
	LibraryID uint
}

func (s *Service) getResourcePlaybackSource(ctx context.Context, req PlaybackRequest) (PlaybackSource, bool, error) {
	var metadataItem database.MetadataItem
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.MetadataItemID).First(&metadataItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return PlaybackSource{}, false, nil
		}
		return PlaybackSource{}, false, err
	}
	candidates, err := s.loadResourcePlaybackCandidates(ctx, req.MetadataItemID, req.LibraryID)
	if err != nil {
		return PlaybackSource{}, true, err
	}
	if len(candidates) == 0 {
		return PlaybackSource{MetadataItemID: metadataItem.ID, Title: metadataItem.Title, Type: metadataItem.ItemType, Playable: false, Decision: PlaybackDecision{Kind: "unplayable", ClientProfile: req.ClientProfile, SelectedBy: "no_resource", Reasons: []DecisionReason{{Code: "no_available_resource", Category: "availability", Message: "no playable resource is linked to this metadata item"}}}}, true, nil
	}
	selected, selectedBy, ok := s.selectResourcePlaybackCandidate(ctx, candidates, req)
	if !ok {
		return PlaybackSource{MetadataItemID: metadataItem.ID, Title: metadataItem.Title, Type: metadataItem.ItemType, Playable: false, Decision: PlaybackDecision{Kind: "unplayable", ClientProfile: req.ClientProfile, SelectedBy: "resource_filter", Reasons: []DecisionReason{{Code: "resource_not_available", Category: "availability", Message: "requested resource is not linked to this metadata item"}}}}, true, nil
	}
	fileLink, err := s.GetInventoryFileLink(ctx, selected.File.ID)
	if err != nil {
		return PlaybackSource{}, true, fmt.Errorf("load inventory file link: %w", err)
	}
	pseudoFile, audioTracks, subtitleTracks := inventoryCandidateMediaInfo(playbackFileCandidate{ID: selected.Resource.ID, Status: selected.Resource.Status, Kind: selected.Link.Role, ProbeStatus: selected.Resource.ProbeStatus, QualityLabel: selected.Resource.QualityLabel, Edition: selected.Resource.Edition, File: selected.File, Files: selected.Files, Streams: selected.Streams})
	subtitlePolicy, err := s.subtitlePolicy(ctx, selected.LibraryID)
	if err == nil {
		subtitleTracks = s.applySubtitlePolicy(s.enrichExternalSubtitleTracks(ctx, subtitleTracks), subtitlePolicy)
	}
	checks := append([]PlaybackCheck{}, fileLink.Checks...)
	checks = append(checks, buildMediaInfoCheck(pseudoFile))
	directDecision := assessDirectPlay(pseudoFile, req.ClientProfile)
	if !fileLink.Playable {
		directDecision.direct = false
		directDecision.reasons = append([]DecisionReason{{Code: "source_unavailable", Category: "availability", Message: "media source is unavailable"}}, directDecision.reasons...)
	}
	source := PlaybackSource{MetadataItemID: metadataItem.ID, ResourceID: selected.Resource.ID, FileID: selected.File.ID, Title: metadataItem.Title, Type: metadataItem.ItemType, Container: selected.File.Container, URL: fileLink.URL, Direct: fileLink.Playable, SizeBytes: selected.File.SizeBytes, RuntimeSeconds: metadataItem.RuntimeSeconds, SegmentIndex: selected.Link.SegmentIndex, StartSeconds: selected.Link.StartSeconds, EndSeconds: selected.Link.EndSeconds, QualityLabel: selected.Resource.QualityLabel, Edition: selected.Resource.Edition, VideoCodec: pseudoFile.VideoCodec, Width: pseudoFile.Width, Height: pseudoFile.Height, AudioTracks: audioTracks, SubtitleTracks: subtitleTracks, Checks: checks, Playable: fileLink.Playable}
	if source.Playable {
		source.Decision = PlaybackDecision{Kind: "direct", ClientProfile: req.ClientProfile, SelectedBy: selectedBy, Reasons: directDecision.reasons}
		return source, true, nil
	}
	source.URL = ""
	source.Direct = false
	source.Decision = PlaybackDecision{Kind: "unplayable", ClientProfile: req.ClientProfile, SelectedBy: selectedBy, Reasons: append(append([]DecisionReason{}, directDecision.reasons...), DecisionReason{Code: "no_supported_playback_path", Category: "fallback", Message: "no supported playback path is available for this resource"})}
	return source, true, nil
}

func (s *Service) loadResourcePlaybackCandidates(ctx context.Context, metadataItemID uint, libraryID uint) ([]resourcePlaybackCandidate, error) {
	var links []database.ResourceMetadataLink
	query := s.db.WithContext(ctx).Where("metadata_item_id = ?", metadataItemID).Order("role asc, segment_index asc, id asc")
	if err := query.Find(&links).Error; err != nil {
		return nil, err
	}
	result := make([]resourcePlaybackCandidate, 0, len(links))
	for _, link := range links {
		var resource database.Resource
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", link.ResourceID).First(&resource).Error; err != nil {
			continue
		}
		if strings.TrimSpace(resource.Status) != "available" {
			continue
		}
		var libraryLink database.ResourceLibraryLink
		libraryQuery := s.db.WithContext(ctx).Where("resource_id = ? AND deleted_at IS NULL", resource.ID)
		if libraryID != 0 {
			libraryQuery = libraryQuery.Where("library_id = ?", libraryID)
		}
		if err := libraryQuery.Order("library_id asc").First(&libraryLink).Error; err != nil {
			continue
		}
		var resourceFiles []database.ResourceFile
		if err := s.db.WithContext(ctx).Where("resource_id = ? AND role IN ?", resource.ID, []string{database.ResourceFileRoleSource, database.ResourceFileRoleSubtitle}).Order("role asc, part_index asc, id asc").Find(&resourceFiles).Error; err != nil {
			return nil, err
		}
		fileIDs := make([]uint, 0, len(resourceFiles))
		firstFileID := uint(0)
		for _, file := range resourceFiles {
			fileIDs = append(fileIDs, file.InventoryFileID)
			if file.Role == database.ResourceFileRoleSource && firstFileID == 0 {
				firstFileID = file.InventoryFileID
			}
		}
		if firstFileID == 0 {
			continue
		}
		var files []database.InventoryFile
		if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", fileIDs).Find(&files).Error; err != nil {
			return nil, err
		}
		filesByID := map[uint]database.InventoryFile{}
		for _, file := range files {
			filesByID[file.ID] = file
		}
		firstFile, ok := filesByID[firstFileID]
		if !ok {
			continue
		}
		var streams []database.MediaStream
		if len(fileIDs) > 0 {
			if err := s.db.WithContext(ctx).Where("file_id IN ?", fileIDs).Order("file_id asc, stream_index asc").Find(&streams).Error; err != nil {
				return nil, err
			}
		}
		result = append(result, resourcePlaybackCandidate{Resource: resource, File: firstFile, Files: filesByID, Streams: streams, Link: link, LibraryID: libraryLink.LibraryID})
	}
	return result, nil
}

func (s *Service) selectResourcePlaybackCandidate(ctx context.Context, candidates []resourcePlaybackCandidate, req PlaybackRequest) (resourcePlaybackCandidate, string, bool) {
	if req.ResourceID != 0 {
		for _, candidate := range candidates {
			if candidate.Resource.ID == req.ResourceID {
				return candidate, "preferred_resource", true
			}
		}
		return resourcePlaybackCandidate{}, "preferred_resource", false
	}
	if len(candidates) == 0 {
		return resourcePlaybackCandidate{}, "no_resource", false
	}
	if req.UserID != nil {
		if candidate, ok := s.selectUserPreferredResource(ctx, candidates, *req.UserID, req.MetadataItemID); ok {
			return candidate, "user_resource_progress", true
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := resourcePlaybackRank(candidates[i])
		right := resourcePlaybackRank(candidates[j])
		if left != right {
			return left > right
		}
		return candidates[i].Resource.ID < candidates[j].Resource.ID
	})
	return candidates[0], "resource_rank", true
}

func (s *Service) selectUserPreferredResource(ctx context.Context, candidates []resourcePlaybackCandidate, userID uint, metadataItemID uint) (resourcePlaybackCandidate, bool) {
	var metadataData database.UserMetadataData
	if err := s.db.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", userID, metadataItemID).First(&metadataData).Error; err == nil {
		if metadataData.PreferredResourceID != nil {
			for _, candidate := range candidates {
				if candidate.Resource.ID == *metadataData.PreferredResourceID {
					return candidate, true
				}
			}
		}
	} else if err != gorm.ErrRecordNotFound {
		return resourcePlaybackCandidate{}, false
	}
	var rows []database.UserResourceData
	if err := s.db.WithContext(ctx).Where("user_id = ? AND metadata_item_id = ?", userID, metadataItemID).Order("last_played_at desc, updated_at desc").Find(&rows).Error; err != nil {
		return resourcePlaybackCandidate{}, false
	}
	byResourceID := make(map[uint]resourcePlaybackCandidate, len(candidates))
	for _, candidate := range candidates {
		byResourceID[candidate.Resource.ID] = candidate
	}
	for _, row := range rows {
		if candidate, ok := byResourceID[row.ResourceID]; ok {
			return candidate, true
		}
	}
	return resourcePlaybackCandidate{}, false
}

func resourcePlaybackRank(candidate resourcePlaybackCandidate) int {
	score := 0
	if strings.TrimSpace(candidate.Resource.Status) == "available" {
		score += 100
	}
	switch strings.TrimSpace(candidate.Link.Role) {
	case database.ResourceLinkRolePrimary:
		score += 30
	case database.ResourceLinkRoleVersion:
		score += 20
	}
	if strings.TrimSpace(candidate.Resource.ProbeStatus) == probe.StatusReady {
		score += 10
	}
	pseudo, _, _ := inventoryCandidateMediaInfo(playbackFileCandidate{ID: candidate.Resource.ID, Status: candidate.Resource.Status, Kind: candidate.Link.Role, ProbeStatus: candidate.Resource.ProbeStatus, QualityLabel: candidate.Resource.QualityLabel, Edition: candidate.Resource.Edition, File: candidate.File, Files: candidate.Files, Streams: candidate.Streams})
	if assessDirectPlay(pseudo, ClientProfileWeb).direct {
		score += 5
	}
	score += resolutionPixels(pseudo)
	return score
}
