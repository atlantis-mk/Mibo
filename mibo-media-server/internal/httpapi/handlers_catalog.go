package httpapi

import (
	"net/http"

	"github.com/atlan/mibo-media-server/internal/catalog"
)

func normalizeCatalogItemListArtworkURLs(req *http.Request, items []catalog.CatalogListItem) {
	for idx := range items {
		normalizeCatalogItemSummaryArtworkURLs(req, &items[idx])
	}
}

func normalizeCatalogItemSummaryArtworkURLs(req *http.Request, item *catalog.CatalogListItem) {
	if item == nil {
		return
	}
	for idx := range item.SelectedImages {
		item.SelectedImages[idx].URL = buildAssetURL(req, item.SelectedImages[idx].URL)
	}
}

func normalizeCatalogSeasonDetailsArtworkURLs(req *http.Request, seasons []catalog.CatalogSeasonDetail) {
	for idx := range seasons {
		for imageIdx := range seasons[idx].SelectedImages {
			seasons[idx].SelectedImages[imageIdx].URL = buildAssetURL(req, seasons[idx].SelectedImages[imageIdx].URL)
		}
		normalizeCatalogEpisodeDetailsArtworkURLs(req, seasons[idx].Episodes)
	}
}

func normalizeCatalogEpisodeDetailsArtworkURLs(req *http.Request, episodes []catalog.CatalogEpisodeDetail) {
	for idx := range episodes {
		for imageIdx := range episodes[idx].SelectedImages {
			episodes[idx].SelectedImages[imageIdx].URL = buildAssetURL(req, episodes[idx].SelectedImages[imageIdx].URL)
		}
	}
}

func normalizeMetadataItemDetailArtworkURLs(req *http.Request, item *catalog.CatalogItemDetail) {
	if item == nil {
		return
	}
	for idx := range item.SelectedImages {
		item.SelectedImages[idx].URL = buildAssetURL(req, item.SelectedImages[idx].URL)
	}
	if item.EpisodeContext != nil {
		if item.EpisodeContext.Series != nil {
			for idx := range item.EpisodeContext.Series.SelectedImages {
				item.EpisodeContext.Series.SelectedImages[idx].URL = buildAssetURL(req, item.EpisodeContext.Series.SelectedImages[idx].URL)
			}
		}
		if item.EpisodeContext.Season != nil {
			for idx := range item.EpisodeContext.Season.SelectedImages {
				item.EpisodeContext.Season.SelectedImages[idx].URL = buildAssetURL(req, item.EpisodeContext.Season.SelectedImages[idx].URL)
			}
		}
	}
	for idx := range item.SameSeasonEpisodes {
		for imageIdx := range item.SameSeasonEpisodes[idx].SelectedImages {
			item.SameSeasonEpisodes[idx].SelectedImages[imageIdx].URL = buildAssetURL(req, item.SameSeasonEpisodes[idx].SelectedImages[imageIdx].URL)
		}
	}
	normalizeCatalogSeasonDetailsArtworkURLs(req, item.Seasons)
	normalizeCatalogEpisodeDetailsArtworkURLs(req, item.Episodes)
}

func normalizeCatalogPersonDetailArtworkURLs(req *http.Request, person *catalog.CatalogPersonPageDetail) {
	if person == nil {
		return
	}
	normalizeCatalogItemListArtworkURLs(req, person.RelatedItems)
}

func normalizeCatalogGovernanceWorkspaceArtworkURLs(req *http.Request, workspace *catalog.CatalogGovernanceWorkspace) {
	if workspace == nil {
		return
	}
	for idx := range workspace.SelectedImages {
		workspace.SelectedImages[idx].URL = buildAssetURL(req, workspace.SelectedImages[idx].URL)
	}
	for idx := range workspace.ImageCandidates {
		workspace.ImageCandidates[idx].URL = buildAssetURL(req, workspace.ImageCandidates[idx].URL)
	}
	normalizeCatalogItemListArtworkURLs(req, workspace.RecommendedChildren)
}
