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
	"gorm.io/gorm"
)

const (
	posterArtworkKind        = "poster"
	backdropArtworkKind      = "backdrop"
	posterArtworkFilter      = "thumbnail=24,scale=720:1080:force_original_aspect_ratio=increase,crop=720:1080"
	backdropArtworkFilter    = "thumbnail=24,scale=1920:1080:force_original_aspect_ratio=increase,crop=1920:1080"
	defaultArtworkSeekOffset = 5
	maxArtworkSeekOffset     = 300
)

func (s *Service) generateCatalogFallbackArtwork(ctx context.Context, file database.InventoryFile, target string, runtimeSeconds *int) error {
	if !s.ffmpeg.Enabled || strings.TrimSpace(s.ffmpeg.Path) == "" || strings.TrimSpace(target) == "" {
		return nil
	}

	itemIDs, err := catalogItemIDsForInventoryFile(s.db.WithContext(ctx), file.ID)
	if err != nil {
		return err
	}

	var resultErr error
	for _, itemID := range itemIDs {
		if err := s.generateCatalogFallbackArtworkForItem(ctx, itemID, target, runtimeSeconds); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}
	return resultErr
}

func (s *Service) generateCatalogFallbackArtworkForItem(ctx context.Context, itemID uint, target string, runtimeSeconds *int) error {
	var images []database.ItemImage
	if err := s.db.WithContext(ctx).
		Where("item_id = ?", itemID).
		Order("image_type asc, id asc").
		Find(&images).Error; err != nil {
		return err
	}
	if catalogItemHasNonGeneratedArtwork(itemID, images) {
		return nil
	}

	posterPath := s.catalogArtworkPath(itemID, posterArtworkKind)
	backdropPath := s.catalogArtworkPath(itemID, backdropArtworkKind)
	posterURL := generatedCatalogArtworkURL(itemID, posterArtworkKind)
	backdropURL := generatedCatalogArtworkURL(itemID, backdropArtworkKind)
	needsPoster := shouldGenerateCatalogArtwork(images, posterArtworkKind, posterURL, posterPath)
	needsBackdrop := shouldGenerateCatalogArtwork(images, backdropArtworkKind, backdropURL, backdropPath)
	if !needsPoster && !needsBackdrop {
		return nil
	}

	if err := os.MkdirAll(s.catalogArtworkDir(itemID), 0o755); err != nil {
		return err
	}

	var resultErr error
	if needsPoster {
		if err := s.extractArtwork(ctx, target, runtimeSeconds, posterArtworkFilter, posterPath); err != nil {
			resultErr = errors.Join(resultErr, err)
		} else if err := s.upsertGeneratedCatalogArtwork(ctx, itemID, posterArtworkKind, posterURL); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}
	if needsBackdrop {
		if err := s.extractArtwork(ctx, target, runtimeSeconds, backdropArtworkFilter, backdropPath); err != nil {
			resultErr = errors.Join(resultErr, err)
		} else if err := s.upsertGeneratedCatalogArtwork(ctx, itemID, backdropArtworkKind, backdropURL); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}

	return resultErr
}

func (s *Service) upsertGeneratedCatalogArtwork(ctx context.Context, itemID uint, kind string, url string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var image database.ItemImage
		err := tx.WithContext(ctx).
			Where("item_id = ? AND image_type = ? AND url = ?", itemID, kind, url).
			First(&image).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			image = database.ItemImage{
				ItemID:     itemID,
				ImageType:  kind,
				URL:        url,
				IsSelected: true,
			}
			if err := tx.WithContext(ctx).Create(&image).Error; err != nil {
				return err
			}
		} else if !image.IsSelected {
			if err := tx.WithContext(ctx).
				Model(&database.ItemImage{}).
				Where("id = ?", image.ID).
				Updates(map[string]any{"is_selected": true}).Error; err != nil {
				return err
			}
		}

		return tx.WithContext(ctx).
			Model(&database.ItemImage{}).
			Where("item_id = ? AND image_type = ? AND id <> ?", itemID, kind, image.ID).
			Update("is_selected", false).Error
	})
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

func generatedCatalogArtworkURL(itemID uint, kind string) string {
	return fmt.Sprintf("/api/v1/items/%d/artwork/%s", itemID, kind)
}

func (s *Service) catalogArtworkDir(itemID uint) string {
	return filepath.Join(s.artworkRootPath(), "catalog", fmt.Sprintf("%d", itemID))
}

func (s *Service) catalogArtworkPath(itemID uint, kind string) string {
	return filepath.Join(s.catalogArtworkDir(itemID), kind+".jpg")
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

func catalogItemIDsForInventoryFile(tx *gorm.DB, inventoryFileID uint) ([]uint, error) {
	var itemIDs []uint
	err := tx.Model(&database.AssetItem{}).
		Distinct("asset_items.item_id").
		Joins("JOIN asset_files ON asset_files.asset_id = asset_items.asset_id").
		Joins("JOIN catalog_items ON catalog_items.id = asset_items.item_id").
		Where("asset_files.file_id = ?", inventoryFileID).
		Where("asset_items.role IN ?", []string{"primary", "version"}).
		Where("catalog_items.type IN ?", []string{"movie", "episode"}).
		Pluck("asset_items.item_id", &itemIDs).Error
	return itemIDs, err
}

func catalogItemHasNonGeneratedArtwork(itemID uint, images []database.ItemImage) bool {
	for _, image := range images {
		if trimmedURL := strings.TrimSpace(image.URL); trimmedURL != "" && !isGeneratedCatalogArtworkURL(itemID, trimmedURL) {
			return true
		}
	}
	return false
}

func shouldGenerateCatalogArtwork(images []database.ItemImage, kind string, generatedURL string, outputPath string) bool {
	for _, image := range images {
		if strings.TrimSpace(image.ImageType) != kind || strings.TrimSpace(image.URL) != generatedURL {
			continue
		}
		if _, err := os.Stat(outputPath); err == nil {
			return false
		}
		break
	}
	return true
}

func isGeneratedCatalogArtworkURL(itemID uint, rawURL string) bool {
	prefix := fmt.Sprintf("/api/v1/items/%d/artwork/", itemID)
	return strings.HasPrefix(strings.TrimSpace(rawURL), prefix)
}
