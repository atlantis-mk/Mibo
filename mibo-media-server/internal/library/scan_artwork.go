package library

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

const (
	catalogScanArtworkSourceSibling          = "scanner_sibling_artwork"
	catalogScanArtworkSourceProviderThumb    = "scanner_provider_thumbnail"
	catalogScanArtworkProviderThumbnailOrder = 100
)

var catalogScanArtworkExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".webp": {},
}

func applyCatalogScanArtworkCandidates(provider storage.Provider, artifact catalogScanArtifact, object storage.Object, snapshot scanDirectorySnapshot) catalogScanArtifact {
	artifact.ImageCandidates = appendCatalogScanImageCandidates(artifact.ImageCandidates, siblingCatalogScanArtworkCandidates(provider, object, snapshot)...)
	if thumbnail := strings.TrimSpace(object.ThumbnailURL); thumbnail != "" {
		if strings.TrimSpace(artifact.ThumbnailURL) == "" {
			artifact.ThumbnailURL = thumbnail
		}
		imageType := "poster"
		if artifact.ItemType == catalog.ItemTypeEpisode {
			imageType = "still"
		}
		artifact.ImageCandidates = appendCatalogScanImageCandidates(artifact.ImageCandidates, catalogScanImageCandidate{ImageType: imageType, URL: thumbnail, Source: catalogScanArtworkSourceProviderThumb, Priority: catalogScanArtworkProviderThumbnailOrder, Provisional: true})
	}
	sort.SliceStable(artifact.ImageCandidates, func(i, j int) bool {
		if artifact.ImageCandidates[i].ImageType != artifact.ImageCandidates[j].ImageType {
			return artifact.ImageCandidates[i].ImageType < artifact.ImageCandidates[j].ImageType
		}
		return artifact.ImageCandidates[i].Priority < artifact.ImageCandidates[j].Priority
	})
	return artifact
}

func siblingCatalogScanArtworkCandidates(provider storage.Provider, object storage.Object, snapshot scanDirectorySnapshot) []catalogScanImageCandidate {
	providerName := ""
	if provider != nil {
		providerName = provider.Name()
	}
	videoBase := strings.ToLower(strings.TrimSuffix(path.Base(object.Path), path.Ext(object.Path)))
	result := make([]catalogScanImageCandidate, 0, 2)
	for _, candidate := range snapshot.Objects {
		if candidate.IsDir || !isCatalogScanArtworkFile(candidate.Path) {
			continue
		}
		base := strings.ToLower(strings.TrimSuffix(path.Base(candidate.Path), path.Ext(candidate.Path)))
		imageType, priority, ok := catalogScanArtworkDisposition(base, videoBase)
		if !ok {
			continue
		}
		artwork := catalogScanImageCandidate{ImageType: imageType, URL: httpURL(candidate.RawURL), Source: catalogScanArtworkSourceSibling, Priority: priority, Provisional: true}
		if providerName == "local" {
			artwork.Path = candidate.Path
		}
		if artwork.URL == "" && artwork.Path == "" {
			continue
		}
		result = append(result, artwork)
	}
	return result
}

func isCatalogScanArtworkFile(objectPath string) bool {
	_, ok := catalogScanArtworkExtensions[strings.ToLower(strings.TrimSpace(path.Ext(objectPath)))]
	return ok
}

func catalogScanArtworkDisposition(base string, videoBase string) (string, int, bool) {
	switch strings.TrimSpace(base) {
	case "poster":
		return "poster", 10, true
	case "cover":
		return "poster", 11, true
	case "folder":
		return "poster", 12, true
	case strings.TrimSpace(videoBase):
		return "poster", 20, true
	case "backdrop":
		return "backdrop", 10, true
	case "fanart":
		return "backdrop", 11, true
	case "background":
		return "backdrop", 12, true
	default:
		return "", 0, false
	}
}

func appendCatalogScanImageCandidates(existing []catalogScanImageCandidate, values ...catalogScanImageCandidate) []catalogScanImageCandidate {
	seen := make(map[string]struct{}, len(existing)+len(values))
	for _, value := range existing {
		if key := catalogScanImageCandidateKey(value); key != "" {
			seen[key] = struct{}{}
		}
	}
	for _, value := range values {
		key := catalogScanImageCandidateKey(value)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		existing = append(existing, value)
	}
	return existing
}

func catalogScanSeriesFallbackImageCandidates(candidates []catalogScanImageCandidate) []catalogScanImageCandidate {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.ImageType) != "still" {
			continue
		}
		locator := firstNonEmptyString(candidate.URL, candidate.Path)
		if locator == "" {
			continue
		}
		poster := candidate
		poster.ImageType = "poster"
		backdrop := candidate
		backdrop.ImageType = "backdrop"
		return []catalogScanImageCandidate{poster, backdrop}
	}
	return nil
}

func catalogScanImageCandidateKey(value catalogScanImageCandidate) string {
	imageType := strings.TrimSpace(value.ImageType)
	locator := firstNonEmptyString(value.URL, value.Path)
	if imageType == "" || locator == "" {
		return ""
	}
	return imageType + "\x00" + locator
}

func (s *Service) applyCatalogScanImages(ctx context.Context, tx *gorm.DB, itemID uint, candidates []catalogScanImageCandidate, sourceID *uint) error {
	if itemID == 0 || len(candidates) == 0 {
		return nil
	}
	for _, candidate := range candidates {
		imageType := strings.TrimSpace(candidate.ImageType)
		if imageType == "" {
			continue
		}
		url, err := s.catalogScanImageURL(itemID, candidate)
		if err != nil {
			return err
		}
		if strings.TrimSpace(url) == "" {
			continue
		}
		selected, err := catalogScanImageShouldSelect(ctx, tx, itemID, imageType)
		if err != nil {
			return err
		}
		if selected {
			if err := tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ?", itemID, imageType).Update("is_selected", false).Error; err != nil {
				return err
			}
		}
		if err := upsertCatalogScanImage(ctx, tx, database.ItemImage{ItemID: itemID, ImageType: imageType, URL: url, SourceID: sourceID, IsSelected: selected, SortOrder: candidate.Priority}); err != nil {
			return err
		}
	}
	return nil
}

func upsertCatalogScanImage(ctx context.Context, tx *gorm.DB, image database.ItemImage) error {
	var existing database.ItemImage
	err := tx.WithContext(ctx).Where("item_id = ? AND image_type = ? AND url = ?", image.ItemID, image.ImageType, image.URL).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.WithContext(ctx).Create(&image).Error
	}
	updates := map[string]any{"source_id": image.SourceID, "sort_order": image.SortOrder}
	if image.IsSelected {
		updates["is_selected"] = true
	}
	return tx.WithContext(ctx).Model(&database.ItemImage{}).Where("id = ?", existing.ID).Updates(updates).Error
}

func (s *Service) catalogScanImageURL(itemID uint, candidate catalogScanImageCandidate) (string, error) {
	if trimmed := strings.TrimSpace(candidate.URL); trimmed != "" {
		return trimmed, nil
	}
	if strings.TrimSpace(candidate.Path) == "" || candidate.Source != catalogScanArtworkSourceSibling {
		return "", nil
	}
	imageType := strings.TrimSpace(candidate.ImageType)
	if imageType != "poster" && imageType != "backdrop" {
		return "", nil
	}
	outputPath := filepath.Join(s.catalogScanArtworkDir(itemID), imageType+strings.ToLower(filepath.Ext(candidate.Path)))
	if err := copyCatalogScanArtwork(candidate.Path, outputPath); err != nil {
		return "", err
	}
	return "/api/v1/items/" + strconv.FormatUint(uint64(itemID), 10) + "/artwork/" + imageType, nil
}

func (s *Service) catalogScanArtworkDir(itemID uint) string {
	root := strings.TrimSpace(s.cfg.FFmpeg.ArtworkRootPath)
	if root == "" {
		root = filepath.Join("tmp", "artwork")
	}
	return filepath.Join(root, "catalog", strconv.FormatUint(uint64(itemID), 10))
}

func copyCatalogScanArtwork(sourcePath string, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	input, err := os.Open(filepath.Clean(sourcePath))
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.Create(filepath.Clean(outputPath))
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	return err
}

func catalogScanImageShouldSelect(ctx context.Context, tx *gorm.DB, itemID uint, imageType string) (bool, error) {
	var count int64
	if err := tx.WithContext(ctx).Model(&database.ItemImage{}).Where("item_id = ? AND image_type = ? AND is_selected = ?", itemID, imageType, true).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func httpURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
