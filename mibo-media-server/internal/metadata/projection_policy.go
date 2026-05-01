package metadata

import (
	"context"

	"github.com/atlan/mibo-media-server/internal/catalog"
)

func (s *Service) refreshMetadataOperationProjectionScope(ctx context.Context, scope MetadataAffectedScope) error {
	catalogSvc := catalog.NewService(s.db)
	if scope.RootID != nil && *scope.RootID != 0 {
		return catalogSvc.RefreshItemProjection(ctx, *scope.RootID)
	}
	seen := map[uint]struct{}{}
	for _, itemID := range scope.ItemIDs {
		if itemID == 0 {
			continue
		}
		if _, ok := seen[itemID]; ok {
			continue
		}
		seen[itemID] = struct{}{}
		if err := catalogSvc.RefreshItemProjection(ctx, itemID); err != nil {
			return err
		}
	}
	return nil
}
