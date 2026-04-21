package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
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

func NewService(db *gorm.DB, registry *providers.Registry, cfg config.FFprobeConfig) *Service {
	return &Service{db: db, storage: registry, cfg: cfg}
}

func (s *Service) ProbeFile(ctx context.Context, mediaFileID uint) error {
	var file database.MediaFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", mediaFileID).
		First(&file).Error; err != nil {
		return err
	}

	if !s.cfg.Enabled {
		return s.markUnavailable(ctx, file.ID, "ffprobe disabled")
	}

	provider, err := s.providerForFile(ctx, file.LibraryID)
	if err != nil {
		return s.markProbeError(ctx, file.ID, err)
	}

	target, err := s.resolveProbeTarget(ctx, provider, file.StoragePath)
	if err != nil {
		return s.markProbeError(ctx, file.ID, err)
	}

	probeCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	output, err := exec.CommandContext(probeCtx, s.cfg.Path, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", target).Output()
	if err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return s.markUnavailable(ctx, file.ID, "ffprobe not found")
		}
		return s.markProbeError(ctx, file.ID, err)
	}

	var parsed ffprobeOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return s.markProbeError(ctx, file.ID, err)
	}

	updates, runtimeSeconds, err := buildProbeUpdates(parsed)
	if err != nil {
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

	return nil
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
