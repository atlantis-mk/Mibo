package catalog

import "context"

func (s *Service) RefreshLibraryProjection(ctx context.Context, libraryID uint, metadataItemID uint) error {
	_, err := s.RebuildLibraryMetadataProjection(ctx, libraryID, metadataItemID)
	return err
}

func (s *Service) RefreshLibraryProjectionScope(ctx context.Context, libraryID uint) error {
	return s.RebuildLibraryMetadataProjections(ctx, libraryID)
}

func (s *Service) RefreshMetadataReadModels(ctx context.Context, libraryID uint, metadataItemID uint) error {
	if err := s.RefreshLibraryProjection(ctx, libraryID, metadataItemID); err != nil {
		return err
	}
	if _, err := s.RebuildMetadataSearchDocument(ctx, metadataItemID); err != nil {
		return err
	}
	if _, err := s.RebuildLibrarySearchDocument(ctx, libraryID, metadataItemID); err != nil {
		return err
	}
	return nil
}
