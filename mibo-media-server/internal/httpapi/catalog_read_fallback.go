package httpapi

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/library"
)

func (r *Router) listCatalogDiscoveryItems(ctx context.Context, input library.BrowseMediaItemsInput) ([]catalog.CatalogListItem, error) {
	if r.catalog == nil {
		return []catalog.CatalogListItem{}, nil
	}
	if strings.TrimSpace(input.Query) == "" {
		return r.catalog.ListItems(ctx, input.LibraryID, "", string(input.TypeFilter), input.Limit)
	}
	return r.catalog.SearchItems(ctx, input.LibraryID, input.Query, string(input.TypeFilter), input.Limit)
}
