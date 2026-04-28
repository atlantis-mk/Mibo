package probe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	slashpath "path"
	"path/filepath"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

const (
	posterArtworkKind        = "poster"
	backdropArtworkKind      = "backdrop"
	stillArtworkKind         = "still"
	posterArtworkFilter      = "thumbnail=24,scale=720:1080:force_original_aspect_ratio=increase,crop=720:1080"
	backdropArtworkFilter    = "thumbnail=24,scale=1920:1080:force_original_aspect_ratio=increase,crop=1920:1080"
	defaultArtworkSeekOffset = 5
	maxArtworkSeekOffset     = 300
)

var artworkFileExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}

func (s *Service) generateCatalogFallbackArtwork(ctx context.Context, file database.InventoryFile, provider storage.Provider, target string, runtimeSeconds *int) error {
	if provider == nil || strings.TrimSpace(file.StoragePath) == "" {
		return nil
	}

	itemIDs, err := catalogItemIDsForInventoryFile(s.db.WithContext(ctx), file.ID)
	if err != nil {
		return err
	}

	var resultErr error
	for _, itemID := range itemIDs {
		if err := s.generateCatalogFallbackArtworkForItem(ctx, itemID, file.StoragePath, provider, target, runtimeSeconds); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}
	return resultErr
}

func (s *Service) generateCatalogFallbackArtworkForItem(ctx context.Context, itemID uint, storagePath string, provider storage.Provider, target string, runtimeSeconds *int) error {
	itemType, err := catalogItemType(s.db.WithContext(ctx), itemID)
	if err != nil {
		return err
	}
	var images []database.ItemImage
	if err := s.db.WithContext(ctx).
		Where("item_id = ?", itemID).
		Order("image_type asc, id asc").
		Find(&images).Error; err != nil {
		return err
	}
	thumbnailKind := providerThumbnailArtworkKind(itemType)
	posterPath := s.catalogArtworkPath(itemID, posterArtworkKind)
	backdropPath := s.catalogArtworkPath(itemID, backdropArtworkKind)
	posterURL := generatedCatalogArtworkURL(itemID, posterArtworkKind)
	backdropURL := generatedCatalogArtworkURL(itemID, backdropArtworkKind)
	posterHasNonGeneratedArtwork := catalogItemHasNonGeneratedArtwork(itemID, images, posterArtworkKind)
	thumbnailHasNonGeneratedArtwork := catalogItemHasNonGeneratedArtwork(itemID, images, thumbnailKind)
	needsPoster := itemType != "episode" && !posterHasNonGeneratedArtwork && shouldGenerateCatalogArtwork(images, posterArtworkKind, posterURL, posterPath)
	needsBackdrop := !catalogItemHasNonGeneratedArtwork(itemID, images, backdropArtworkKind) && shouldGenerateCatalogArtwork(images, backdropArtworkKind, backdropURL, backdropPath)
	if !needsPoster && !needsBackdrop && thumbnailHasNonGeneratedArtwork {
		return nil
	}

	mediaObject, mediaObjectOK := getStorageObject(ctx, provider, storagePath)
	localPoster, localBackdrop, err := s.applySiblingCatalogArtwork(ctx, itemID, storagePath, provider, mediaObject.Related, !posterHasNonGeneratedArtwork, needsBackdrop)
	if err != nil {
		return err
	}
	needsPoster = needsPoster && !localPoster
	needsBackdrop = needsBackdrop && !localBackdrop
	if !thumbnailHasNonGeneratedArtwork && (thumbnailKind != posterArtworkKind || !localPoster) {
		thumbnailApplied, err := s.applyProviderThumbnailCatalogArtwork(ctx, itemID, storagePath, provider, mediaObject, mediaObjectOK, thumbnailKind)
		if err != nil {
			return err
		}
		if thumbnailKind == posterArtworkKind {
			needsPoster = needsPoster && !thumbnailApplied
		}
	}
	if !needsPoster && !needsBackdrop {
		return nil
	}
	if !s.ffmpeg.Enabled || strings.TrimSpace(s.ffmpeg.Path) == "" || strings.TrimSpace(target) == "" {
		return nil
	}

	if err := os.MkdirAll(s.catalogArtworkDir(itemID), 0o755); err != nil {
		return err
	}

	var resultErr error
	if needsPoster {
		if err := s.extractArtwork(ctx, target, runtimeSeconds, posterArtworkFilter, posterPath); err != nil {
			resultErr = errors.Join(resultErr, err)
		} else if err := s.upsertSelectedCatalogArtwork(ctx, itemID, posterArtworkKind, posterURL); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}
	if needsBackdrop {
		if err := s.extractArtwork(ctx, target, runtimeSeconds, backdropArtworkFilter, backdropPath); err != nil {
			resultErr = errors.Join(resultErr, err)
		} else if err := s.upsertSelectedCatalogArtwork(ctx, itemID, backdropArtworkKind, backdropURL); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}

	return resultErr
}

func getStorageObject(ctx context.Context, provider storage.Provider, storagePath string) (storage.Object, bool) {
	object, err := provider.Get(ctx, storage.GetRequest{Path: storagePath})
	if err != nil {
		return storage.Object{}, false
	}
	return object, true
}

func (s *Service) applyProviderThumbnailCatalogArtwork(ctx context.Context, itemID uint, storagePath string, provider storage.Provider, mediaObject storage.Object, mediaObjectOK bool, kind string) (bool, error) {
	object := mediaObject
	if !mediaObjectOK {
		var ok bool
		object, ok = getStorageObject(ctx, provider, storagePath)
		if !ok {
			return false, nil
		}
	}
	thumbnailURL := strings.TrimSpace(object.ThumbnailURL)
	if thumbnailURL == "" {
		return false, nil
	}
	return true, s.upsertSelectedCatalogArtwork(ctx, itemID, kind, thumbnailURL)
}

func (s *Service) applySiblingCatalogArtwork(ctx context.Context, itemID uint, storagePath string, provider storage.Provider, related []storage.Object, needsPoster bool, needsBackdrop bool) (bool, bool, error) {
	var posterApplied bool
	var backdropApplied bool
	if needsPoster {
		applied, err := s.applySiblingCatalogArtworkKind(ctx, itemID, storagePath, provider, related, posterArtworkKind)
		if err != nil {
			return false, false, err
		}
		posterApplied = applied
	}
	if needsBackdrop {
		applied, err := s.applySiblingCatalogArtworkKind(ctx, itemID, storagePath, provider, related, backdropArtworkKind)
		if err != nil {
			return posterApplied, false, err
		}
		backdropApplied = applied
	}
	return posterApplied, backdropApplied, nil
}

func (s *Service) applySiblingCatalogArtworkKind(ctx context.Context, itemID uint, storagePath string, provider storage.Provider, related []storage.Object, kind string) (bool, error) {
	object, ok := findSiblingArtwork(ctx, provider, storagePath, related, kind)
	if !ok {
		return false, nil
	}

	if provider.Name() == "local" {
		ext := normalizedArtworkExtension(object.Path)
		if ext == "" {
			return false, nil
		}
		outputPath := s.catalogArtworkPathWithExt(itemID, kind, ext)
		if err := copyLocalArtwork(object.Path, outputPath); err != nil {
			return false, err
		}
		return true, s.upsertSelectedCatalogArtwork(ctx, itemID, kind, generatedCatalogArtworkURL(itemID, kind))
	}

	link, err := provider.Link(ctx, storage.LinkRequest{Path: object.Path})
	if err == nil && strings.TrimSpace(link.URL) != "" {
		return true, s.upsertSelectedCatalogArtwork(ctx, itemID, kind, link.URL)
	}
	if strings.TrimSpace(object.RawURL) != "" && strings.HasPrefix(strings.TrimSpace(object.RawURL), "http") {
		return true, s.upsertSelectedCatalogArtwork(ctx, itemID, kind, object.RawURL)
	}
	return false, nil
}

func (s *Service) upsertSelectedCatalogArtwork(ctx context.Context, itemID uint, kind string, url string) error {
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

func (s *Service) catalogArtworkPathWithExt(itemID uint, kind string, ext string) string {
	return filepath.Join(s.catalogArtworkDir(itemID), kind+ext)
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

func catalogItemType(tx *gorm.DB, itemID uint) (string, error) {
	var item database.CatalogItem
	if err := tx.Select("type").First(&item, itemID).Error; err != nil {
		return "", err
	}
	return strings.TrimSpace(item.Type), nil
}

func providerThumbnailArtworkKind(itemType string) string {
	if strings.TrimSpace(itemType) == "episode" {
		return stillArtworkKind
	}
	return posterArtworkKind
}

func catalogItemHasNonGeneratedArtwork(itemID uint, images []database.ItemImage, kind string) bool {
	for _, image := range images {
		if !image.IsSelected || strings.TrimSpace(image.ImageType) != kind {
			continue
		}
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
		if generatedArtworkFileExists(outputPath) {
			return false
		}
		break
	}
	return true
}

func generatedArtworkFileExists(defaultPath string) bool {
	if _, err := os.Stat(defaultPath); err == nil {
		return true
	}
	base := strings.TrimSuffix(defaultPath, filepath.Ext(defaultPath))
	for _, ext := range artworkFileExtensions {
		if _, err := os.Stat(base + ext); err == nil {
			return true
		}
	}
	return false
}

func findSiblingArtwork(ctx context.Context, provider storage.Provider, storagePath string, related []storage.Object, kind string) (storage.Object, bool) {
	for _, candidatePath := range siblingArtworkCandidatePaths(provider.Name(), storagePath, kind) {
		if object, ok := findRelatedArtwork(related, candidatePath); ok {
			return object, true
		}
		object, err := provider.Get(ctx, storage.GetRequest{Path: candidatePath})
		if err == nil && !object.IsDir {
			return object, true
		}
	}
	return storage.Object{}, false
}

func findRelatedArtwork(related []storage.Object, candidatePath string) (storage.Object, bool) {
	for _, object := range related {
		if object.IsDir || strings.TrimSpace(object.Path) != candidatePath {
			continue
		}
		return object, true
	}
	return storage.Object{}, false
}

func siblingArtworkCandidatePaths(providerName string, storagePath string, kind string) []string {
	names := siblingArtworkNames(storagePath, kind, providerName == "local")
	paths := make([]string, 0, len(names)*len(artworkFileExtensions))
	if providerName == "local" {
		dir := filepath.Dir(storagePath)
		for _, name := range names {
			for _, ext := range artworkFileExtensions {
				paths = append(paths, filepath.Join(dir, name+ext))
			}
		}
		return paths
	}
	dir := slashpath.Dir(storagePath)
	for _, name := range names {
		for _, ext := range artworkFileExtensions {
			paths = append(paths, slashpath.Join(dir, name+ext))
		}
	}
	return paths
}

func siblingArtworkNames(storagePath string, kind string, localPath bool) []string {
	baseName := slashpath.Base(storagePath)
	if localPath {
		baseName = filepath.Base(storagePath)
	}
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	switch kind {
	case posterArtworkKind:
		return []string{baseName + "-poster", baseName + "-cover", "poster", "cover", "folder"}
	case backdropArtworkKind:
		return []string{baseName + "-backdrop", baseName + "-background", baseName + "-fanart", "backdrop", "background", "fanart"}
	default:
		return nil
	}
}

func normalizedArtworkExtension(path string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(path)))
	for _, supported := range artworkFileExtensions {
		if ext == supported {
			return ext
		}
	}
	return ""
}

func copyLocalArtwork(sourcePath string, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer target.Close()
	_, err = io.Copy(target, source)
	return err
}

func isGeneratedCatalogArtworkURL(itemID uint, rawURL string) bool {
	prefix := fmt.Sprintf("/api/v1/items/%d/artwork/", itemID)
	return strings.HasPrefix(strings.TrimSpace(rawURL), prefix)
}
