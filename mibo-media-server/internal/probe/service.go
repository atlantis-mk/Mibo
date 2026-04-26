package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	librarysvc "github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

const (
	StatusPending     = "pending"
	StatusReady       = "ready"
	StatusUnavailable = "unavailable"
	StatusError       = "error"
)

type Service struct {
	db      *gorm.DB
	storage *providers.Registry
	cfg     config.FFprobeConfig
	ffmpeg  config.FFmpegConfig
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Channels  int    `json:"channels"`
	Tags      struct {
		Language string `json:"language"`
		Title    string `json:"title"`
	} `json:"tags"`
}

type Track struct {
	Codec    string `json:"codec"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Channels int    `json:"channels,omitempty"`
}

func NewService(db *gorm.DB, registry *providers.Registry, cfg config.FFprobeConfig, args ...config.FFmpegConfig) *Service {
	service := &Service{db: db, storage: registry, cfg: cfg}
	if len(args) > 0 {
		service.ffmpeg = args[0]
	}
	return service
}

func (s *Service) ProbeFile(ctx context.Context, mediaFileID uint) error {
	var file database.MediaFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaFileID).
		First(&file).Error; err != nil {
		return err
	}

	provider, err := s.providerForFile(ctx, file.LibraryID)
	if err != nil {
		return s.markProbeError(ctx, file.ID, err)
	}

	target, err := s.resolveProbeTarget(ctx, provider, file.StoragePath)
	if err != nil {
		return s.markProbeError(ctx, file.ID, err)
	}

	var runtimeSeconds *int
	if !s.cfg.Enabled {
		s.tryGenerateFallbackArtwork(ctx, file, target, runtimeSeconds)
		return s.markUnavailable(ctx, file.ID, "ffprobe disabled")
	}

	probeCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	output, err := exec.CommandContext(probeCtx, s.cfg.Path, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", target).Output()
	if err != nil {
		s.tryGenerateFallbackArtwork(ctx, file, target, runtimeSeconds)
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return s.markUnavailable(ctx, file.ID, "ffprobe not found")
		}
		return s.markProbeError(ctx, file.ID, err)
	}

	var parsed ffprobeOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		s.tryGenerateFallbackArtwork(ctx, file, target, runtimeSeconds)
		return s.markProbeError(ctx, file.ID, err)
	}

	updates, runtimeSeconds, err := buildProbeUpdates(parsed)
	if err != nil {
		s.tryGenerateFallbackArtwork(ctx, file, target, runtimeSeconds)
		return s.markProbeError(ctx, file.ID, err)
	}

	if err := s.db.WithContext(ctx).
		Model(&database.MediaFile{}).
		Where("id = ?", file.ID).
		Updates(updates).Error; err != nil {
		return err
	}

	if file.MediaItemID != nil && runtimeSeconds != nil {
		if err := s.db.WithContext(ctx).
			Model(&database.MediaItem{}).
			Where("id = ?", *file.MediaItemID).
			Update("runtime_seconds", runtimeSeconds).Error; err != nil {
			return err
		}
	}
	if updates["duration_seconds"] != nil {
		if err := librarysvc.ReconcileProvisionalMediaFile(ctx, s.db, file.ID); err != nil {
			return err
		}
	}
	s.tryGenerateFallbackArtwork(ctx, file, target, runtimeSeconds)

	return nil
}

func (s *Service) ProbeInventoryFile(ctx context.Context, inventoryFileID uint) error {
	var file database.InventoryFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", inventoryFileID).
		First(&file).Error; err != nil {
		return err
	}

	provider, err := s.providerForFile(ctx, file.LibraryID)
	if err != nil {
		return s.markInventoryProbeError(ctx, file.ID, err)
	}

	target, err := s.resolveProbeTarget(ctx, provider, file.StoragePath)
	if err != nil {
		return s.markInventoryProbeError(ctx, file.ID, err)
	}

	if !s.cfg.Enabled {
		s.tryGenerateCatalogFallbackArtwork(ctx, file, target, nil)
		return s.markInventoryUnavailable(ctx, file.ID, "ffprobe disabled")
	}

	probeCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	output, err := exec.CommandContext(probeCtx, s.cfg.Path, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", target).Output()
	if err != nil {
		s.tryGenerateCatalogFallbackArtwork(ctx, file, target, nil)
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return s.markInventoryUnavailable(ctx, file.ID, "ffprobe not found")
		}
		return s.markInventoryProbeError(ctx, file.ID, err)
	}

	var parsed ffprobeOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		s.tryGenerateCatalogFallbackArtwork(ctx, file, target, nil)
		return s.markInventoryProbeError(ctx, file.ID, err)
	}

	updates, runtimeSeconds, err := buildProbeUpdates(parsed)
	if err != nil {
		s.tryGenerateCatalogFallbackArtwork(ctx, file, target, runtimeSeconds)
		return s.markInventoryProbeError(ctx, file.ID, err)
	}
	streams := buildInventoryMediaStreams(file.ID, parsed, updates)
	technicalSummaryJSON, err := buildTechnicalSummaryJSON(parsed, updates)
	if err != nil {
		return s.markInventoryProbeError(ctx, file.ID, err)
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("file_id = ?", file.ID).Delete(&database.MediaStream{}).Error; err != nil {
			return err
		}
		if len(streams) > 0 {
			if err := tx.Create(&streams).Error; err != nil {
				return err
			}
		}

		assetIDs, err := assetIDsForInventoryFile(tx, file.ID)
		if err != nil {
			return err
		}
		if len(assetIDs) > 0 {
			assetUpdates := map[string]any{
				"probe_status":           StatusReady,
				"technical_summary_json": technicalSummaryJSON,
			}
			if durationSeconds, ok := updates["duration_seconds"].(*float64); ok {
				assetUpdates["duration_seconds"] = durationSeconds
			}
			if err := tx.Model(&database.MediaAsset{}).Where("id IN ?", assetIDs).Updates(assetUpdates).Error; err != nil {
				return err
			}
		}

		if runtimeSeconds != nil {
			if err := updateCatalogRuntimeForInventoryFile(tx, file.ID, *runtimeSeconds); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return s.markInventoryProbeError(ctx, file.ID, err)
	}
	s.tryGenerateCatalogFallbackArtwork(ctx, file, target, runtimeSeconds)

	return nil
}

func (s *Service) tryGenerateFallbackArtwork(ctx context.Context, file database.MediaFile, target string, runtimeSeconds *int) {
	if err := s.generateFallbackArtwork(ctx, file, target, runtimeSeconds); err != nil {
		log.Printf("probe: fallback artwork generation failed for media_file=%d: %v", file.ID, err)
	}
}

func (s *Service) tryGenerateCatalogFallbackArtwork(ctx context.Context, file database.InventoryFile, target string, runtimeSeconds *int) {
	if err := s.generateCatalogFallbackArtwork(ctx, file, target, runtimeSeconds); err != nil {
		log.Printf("probe: catalog fallback artwork generation failed for inventory_file=%d: %v", file.ID, err)
	}
}

func buildInventoryMediaStreams(fileID uint, parsed ffprobeOutput, updates map[string]any) []database.MediaStream {
	streams := make([]database.MediaStream, 0, len(parsed.Streams))
	durationSeconds, _ := updates["duration_seconds"].(*float64)
	bitRate, _ := updates["bit_rate"].(*int64)
	for index, stream := range parsed.Streams {
		row := database.MediaStream{
			FileID:          fileID,
			StreamIndex:     index,
			StreamType:      strings.TrimSpace(stream.CodecType),
			Codec:           strings.TrimSpace(stream.CodecName),
			Language:        strings.TrimSpace(stream.Tags.Language),
			Title:           strings.TrimSpace(stream.Tags.Title),
			BitRate:         bitRate,
			DurationSeconds: durationSeconds,
		}
		if stream.Width > 0 {
			width := stream.Width
			row.Width = &width
		}
		if stream.Height > 0 {
			height := stream.Height
			row.Height = &height
		}
		if stream.Channels > 0 {
			channels := stream.Channels
			row.Channels = &channels
		}
		streams = append(streams, row)
	}
	return streams
}

func buildTechnicalSummaryJSON(parsed ffprobeOutput, updates map[string]any) (string, error) {
	audioTrackCount := 0
	subtitleTrackCount := 0
	for _, stream := range parsed.Streams {
		switch strings.TrimSpace(stream.CodecType) {
		case "audio":
			audioTrackCount++
		case "subtitle":
			subtitleTrackCount++
		}
	}

	summary := map[string]any{
		"audio_track_count":    audioTrackCount,
		"subtitle_track_count": subtitleTrackCount,
	}
	if videoCodec, ok := updates["video_codec"].(string); ok && strings.TrimSpace(videoCodec) != "" {
		summary["video_codec"] = videoCodec
	}
	if durationSeconds, ok := updates["duration_seconds"].(*float64); ok {
		summary["duration_seconds"] = *durationSeconds
	}
	if bitRate, ok := updates["bit_rate"].(*int64); ok {
		summary["bit_rate"] = *bitRate
	}
	if width, ok := updates["width"].(*int); ok {
		summary["width"] = *width
	}
	if height, ok := updates["height"].(*int); ok {
		summary["height"] = *height
	}

	encoded, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func assetIDsForInventoryFile(tx *gorm.DB, inventoryFileID uint) ([]uint, error) {
	var assetIDs []uint
	err := tx.Model(&database.AssetFile{}).
		Distinct("asset_id").
		Where("file_id = ?", inventoryFileID).
		Pluck("asset_id", &assetIDs).Error
	return assetIDs, err
}

func updateCatalogRuntimeForInventoryFile(tx *gorm.DB, inventoryFileID uint, runtimeSeconds int) error {
	subquery := tx.Table("asset_items").
		Distinct("asset_items.item_id").
		Joins("JOIN asset_files ON asset_files.asset_id = asset_items.asset_id").
		Where("asset_files.file_id = ?", inventoryFileID).
		Where("asset_items.role IN ?", []string{inventory.AssetItemRolePrimary, inventory.AssetItemRoleVersion})

	return tx.Model(&database.CatalogItem{}).
		Where("id IN (?)", subquery).
		Where("type IN ?", []string{"movie", "episode"}).
		Update("runtime_seconds", runtimeSeconds).Error
}

func (s *Service) markInventoryUnavailable(ctx context.Context, inventoryFileID uint, message string) error {
	assetIDs, err := assetIDsForInventoryFile(s.db.WithContext(ctx), inventoryFileID)
	if err != nil {
		return err
	}
	if len(assetIDs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Model(&database.MediaAsset{}).
		Where("id IN ?", assetIDs).
		Updates(map[string]any{
			"probe_status":           StatusUnavailable,
			"technical_summary_json": "",
		}).Error
}

func (s *Service) markInventoryProbeError(ctx context.Context, inventoryFileID uint, err error) error {
	message := "probe failed"
	if err != nil {
		message = err.Error()
	}
	assetIDs, queryErr := assetIDsForInventoryFile(s.db.WithContext(ctx), inventoryFileID)
	if queryErr != nil {
		return queryErr
	}
	if len(assetIDs) == 0 {
		return err
	}
	if updateErr := s.db.WithContext(ctx).
		Model(&database.MediaAsset{}).
		Where("id IN ?", assetIDs).
		Updates(map[string]any{
			"probe_status":           StatusError,
			"technical_summary_json": message,
		}).Error; updateErr != nil {
		return updateErr
	}
	return err
}

func (s *Service) resolveProbeTarget(ctx context.Context, provider storage.Provider, storagePath string) (string, error) {
	link, err := provider.Link(ctx, storage.LinkRequest{Path: storagePath})
	if err == nil && strings.TrimSpace(link.URL) != "" {
		return link.URL, nil
	}

	object, getErr := provider.Get(ctx, storage.GetRequest{Path: storagePath})
	if getErr == nil && strings.TrimSpace(object.RawURL) != "" {
		return object.RawURL, nil
	}

	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("no probe target available for %s", storagePath)
}

func (s *Service) providerForFile(ctx context.Context, libraryID uint) (storage.Provider, error) {
	var libraryRecord database.Library
	if err := s.db.WithContext(ctx).First(&libraryRecord, libraryID).Error; err != nil {
		return nil, err
	}
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, libraryRecord.MediaSourceID).Error; err != nil {
		return nil, err
	}
	return s.storage.BuildForSource(source)
}

func buildProbeUpdates(parsed ffprobeOutput) (map[string]any, *int, error) {
	audioTracks := make([]Track, 0)
	subtitleTracks := make([]Track, 0)
	var videoCodec string
	var width *int
	var height *int

	for _, stream := range parsed.Streams {
		switch stream.CodecType {
		case "video":
			if videoCodec == "" {
				videoCodec = stream.CodecName
				if stream.Width > 0 {
					width = &stream.Width
				}
				if stream.Height > 0 {
					height = &stream.Height
				}
			}
		case "audio":
			audioTracks = append(audioTracks, Track{Codec: stream.CodecName, Language: stream.Tags.Language, Title: stream.Tags.Title, Channels: stream.Channels})
		case "subtitle":
			subtitleTracks = append(subtitleTracks, Track{Codec: stream.CodecName, Language: stream.Tags.Language, Title: stream.Tags.Title})
		}
	}

	audioJSON, err := json.Marshal(audioTracks)
	if err != nil {
		return nil, nil, err
	}
	subtitleJSON, err := json.Marshal(subtitleTracks)
	if err != nil {
		return nil, nil, err
	}

	var durationSeconds *float64
	var runtimeSeconds *int
	if strings.TrimSpace(parsed.Format.Duration) != "" {
		value, err := strconv.ParseFloat(parsed.Format.Duration, 64)
		if err == nil && value > 0 {
			durationSeconds = &value
			runtime := int(value + 0.5)
			runtimeSeconds = &runtime
		}
	}

	var bitrate *int64
	if strings.TrimSpace(parsed.Format.BitRate) != "" {
		value, err := strconv.ParseInt(parsed.Format.BitRate, 10, 64)
		if err == nil && value > 0 {
			bitrate = &value
		}
	}

	return map[string]any{
		"probe_status":         StatusReady,
		"probe_error":          "",
		"duration_seconds":     durationSeconds,
		"bit_rate":             bitrate,
		"width":                width,
		"height":               height,
		"video_codec":          videoCodec,
		"audio_tracks_json":    string(audioJSON),
		"subtitle_tracks_json": string(subtitleJSON),
	}, runtimeSeconds, nil
}

func (s *Service) markUnavailable(ctx context.Context, mediaFileID uint, message string) error {
	return s.db.WithContext(ctx).
		Model(&database.MediaFile{}).
		Where("id = ?", mediaFileID).
		Updates(map[string]any{
			"probe_status": StatusUnavailable,
			"probe_error":  message,
		}).Error
}

func (s *Service) markProbeError(ctx context.Context, mediaFileID uint, err error) error {
	message := "probe failed"
	if err != nil {
		message = err.Error()
	}
	if updateErr := s.db.WithContext(ctx).
		Model(&database.MediaFile{}).
		Where("id = ?", mediaFileID).
		Updates(map[string]any{
			"probe_status": StatusError,
			"probe_error":  message,
		}).Error; updateErr != nil {
		return updateErr
	}
	return err
}
