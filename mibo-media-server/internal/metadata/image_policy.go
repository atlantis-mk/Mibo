package metadata

import "context"

func (s *Service) applyNormalizedImages(ctx context.Context, itemID uint, images []NormalizedMetadataImage, forceSelectImages bool, sourceID *uint) error {
	for _, image := range images {
		if image.URL == "" || image.ImageType == "" {
			continue
		}
		if err := s.upsertCatalogImageCandidate(ctx, itemID, image.ImageType, image.URL, image.Language, image.SortOrder, image.Selected, forceSelectImages, sourceID); err != nil {
			return err
		}
	}
	return nil
}
