package probe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
)

const (
	posterArtworkKind        = "poster"
	backdropArtworkKind      = "backdrop"
	posterArtworkFilter      = "thumbnail=24,scale=720:1080:force_original_aspect_ratio=increase,crop=720:1080"
	backdropArtworkFilter    = "thumbnail=24,scale=1920:1080:force_original_aspect_ratio=increase,crop=1920:1080"
	defaultArtworkSeekOffset = 5
	maxArtworkSeekOffset     = 300
)

func (s *Service) generateFallbackArtwork(ctx context.Context, file database.MediaFile, target string, runtimeSeconds *int) error {
	if file.MediaItemID == nil || *file.MediaItemID == 0 {
		return nil
	}
	if !s.ffmpeg.Enabled || strings.TrimSpace(s.ffmpeg.Path) == "" || strings.TrimSpace(target) == "" {
		return nil
	}

	var item database.MediaItem
	if err := s.db.WithContext(ctx).
		Select("id", "poster_url", "backdrop_url").
		Where("id = ? AND deleted_at IS NULL", *file.MediaItemID).
		First(&item).Error; err != nil {
		return err
	}

	posterURL := generatedArtworkURL(item.ID, posterArtworkKind)
	backdropURL := generatedArtworkURL(item.ID, backdropArtworkKind)
	needsPoster := shouldGenerateArtwork(item.PosterURL, posterURL)
	needsBackdrop := shouldGenerateArtwork(item.BackdropURL, backdropURL)
	if !needsPoster && !needsBackdrop {
		return nil
	}

	if err := os.MkdirAll(s.artworkDir(item.ID), 0o755); err != nil {
		return err
	}

	updates := make(map[string]any)
	var resultErr error
	if needsPoster {
		posterPath := s.artworkPath(item.ID, posterArtworkKind)
		if err := s.extractArtwork(ctx, target, runtimeSeconds, posterArtworkFilter, posterPath); err != nil {
			resultErr = errors.Join(resultErr, err)
		} else {
			updates["poster_url"] = posterURL
		}
	}
	if needsBackdrop {
		backdropPath := s.artworkPath(item.ID, backdropArtworkKind)
		if err := s.extractArtwork(ctx, target, runtimeSeconds, backdropArtworkFilter, backdropPath); err != nil {
			resultErr = errors.Join(resultErr, err)
		} else {
			updates["backdrop_url"] = backdropURL
		}
	}
	if len(updates) == 0 {
		return resultErr
	}
	if err := s.db.WithContext(ctx).Model(&database.MediaItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return errors.Join(resultErr, err)
	}
	return resultErr
}

func (s *Service) extractArtwork(ctx context.Context, target string, runtimeSeconds *int, filter, outputPath string) error {
	commandCtx := ctx
	var cancel context.CancelFunc
	if s.ffmpeg.Timeout > 0 {
		commandCtx, cancel = context.WithTimeout(ctx, s.ffmpeg.Timeout)
		defer cancel()
	}

	args := []string{
		"-y",
		"-nostdin",
		"-ss", artworkSeekOffset(runtimeSeconds),
		"-i", target,
		"-frames:v", "1",
		"-an",
		"-sn",
		"-vf", filter,
		"-q:v", "2",
		outputPath,
	}
	output, err := exec.CommandContext(commandCtx, s.ffmpeg.Path, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg artwork extraction failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func generatedArtworkURL(mediaItemID uint, kind string) string {
	return fmt.Sprintf("/api/v1/media-items/%d/artwork/%s", mediaItemID, kind)
}

func shouldGenerateArtwork(current, generated string) bool {
	trimmed := strings.TrimSpace(current)
	return trimmed == "" || trimmed == generated
}

func (s *Service) artworkDir(mediaItemID uint) string {
	return filepath.Join(s.artworkRootPath(), fmt.Sprintf("%d", mediaItemID))
}

func (s *Service) artworkPath(mediaItemID uint, kind string) string {
	return filepath.Join(s.artworkDir(mediaItemID), kind+".jpg")
}

func (s *Service) artworkRootPath() string {
	trimmed := strings.TrimSpace(s.ffmpeg.ArtworkRootPath)
	if trimmed != "" {
		return trimmed
	}
	return filepath.Join("tmp", "artwork")
}

func artworkSeekOffset(runtimeSeconds *int) string {
	seconds := defaultArtworkSeekOffset
	if runtimeSeconds != nil && *runtimeSeconds > 0 {
		seconds = *runtimeSeconds / 3
		if seconds > maxArtworkSeekOffset {
			seconds = maxArtworkSeekOffset
		}
		if seconds < 0 {
			seconds = 0
		}
	}
	return fmt.Sprintf("%d", seconds)
}
