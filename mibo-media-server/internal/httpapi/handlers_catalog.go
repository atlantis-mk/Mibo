package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
)

type catalogGovernanceFieldUpdateInput struct {
	FieldKey   string `json:"field_key"`
	Value      any    `json:"value"`
	Lock       bool   `json:"lock"`
	LockReason string `json:"lock_reason"`
	Force      bool   `json:"force"`
}

type catalogGovernanceImageSelectionInput struct {
	ImageType string `json:"image_type"`
	URL       string `json:"url"`
}

type catalogGovernanceAssetLinkInput struct {
	TargetItemID uint `json:"target_item_id"`
}

func (r *Router) handleGetCatalogItem(w http.ResponseWriter, req *http.Request) {
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	detail, err := r.catalog.GetItemDetail(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeCatalogItemDetailArtworkURLs(req, &detail)
	writeJSON(req.Context(), w, http.StatusOK, detail)
}

func (r *Router) handleListCatalogSeriesSeasons(w http.ResponseWriter, req *http.Request) {
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	seriesID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	seasons, err := r.catalog.ListSeriesSeasons(req.Context(), seriesID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeCatalogSeasonDetailsArtworkURLs(req, seasons)
	writeJSON(req.Context(), w, http.StatusOK, seasons)
}

func (r *Router) handleListCatalogItemChildren(w http.ResponseWriter, req *http.Request) {
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	items, err := r.catalog.ListChildItems(req.Context(), itemID, req.URL.Query().Get("type"), req.URL.Query().Get("availability"))
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeCatalogListItemsArtworkURLs(req, items)
	writeJSON(req.Context(), w, http.StatusOK, items)
}

func (r *Router) handleListCatalogSeriesEpisodes(w http.ResponseWriter, req *http.Request) {
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	seriesID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	seasonNumber, hasSeason, err := parseOptionalIntQuery(req, "season")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var seasonFilter *int
	if hasSeason {
		seasonFilter = &seasonNumber
	}
	episodes, err := r.catalog.ListSeriesEpisodes(req.Context(), seriesID, seasonFilter, req.URL.Query().Get("availability"))
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeCatalogEpisodeDetailsArtworkURLs(req, episodes)
	writeJSON(req.Context(), w, http.StatusOK, episodes)
}

func (r *Router) handleListCatalogSeriesMissing(w http.ResponseWriter, req *http.Request) {
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	seriesID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	episodes, err := r.catalog.ListSeriesMissingEpisodes(req.Context(), seriesID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeCatalogEpisodeDetailsArtworkURLs(req, episodes)
	writeJSON(req.Context(), w, http.StatusOK, episodes)
}

func (r *Router) handleGetCatalogSeriesNextUp(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	seriesID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	episode, err := r.catalog.GetSeriesNextUp(req.Context(), user.ID, seriesID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if episode != nil {
		normalizeCatalogEpisodeDetailsArtworkURLs(req, []catalog.CatalogEpisodeDetail{*episode})
	}
	writeJSON(req.Context(), w, http.StatusOK, episode)
}

func (r *Router) handleGetCatalogItemProgress(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	state, err := r.progress.GetCatalogState(req.Context(), user.ID, itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}

func (r *Router) handleGetCatalogPlaybackSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	assetID, err := parseOptionalUintQuery(req, "asset_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	clientProfile, err := parseClientProfileQuery(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	source, err := r.playback.GetPlaybackSource(req.Context(), playback.PlaybackRequest{ItemID: itemID, AssetID: assetID, ClientProfile: clientProfile, AllowHLSFallback: r.hls.Enabled()})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	source.URL = buildPlaybackURL(req, source.URL)
	writeJSON(req.Context(), w, http.StatusOK, source)
}

func (r *Router) handleGetCatalogAssetLink(w http.ResponseWriter, req *http.Request) {
	assetID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	link, err := r.playback.GetAssetLink(req.Context(), assetID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	link.URL = buildPlaybackURL(req, link.URL)
	writeJSON(req.Context(), w, http.StatusOK, link)
}

func (r *Router) handleGetCatalogGovernanceWorkspace(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleUpdateCatalogGovernanceField(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceFieldUpdateInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.FieldKey) == "" {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("field_key is required"))
		return
	}
	if _, _, err := r.catalog.ApplyField(req.Context(), catalog.ApplyFieldInput{
		ItemID:         itemID,
		FieldKey:       input.FieldKey,
		Value:          input.Value,
		Lock:           input.Lock,
		LockReason:     input.LockReason,
		EditedByUserID: &user.ID,
		Force:          input.Force,
	}); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleSelectCatalogGovernanceImage(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceImageSelectionInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := r.catalog.SelectImage(req.Context(), itemID, input.ImageType, input.URL); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleLinkCatalogGovernanceAsset(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	workspaceItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	assetID, err := parseUintPathValue(req, "asset_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceAssetLinkInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	allowed, err := r.catalog.IsGovernanceTargetAllowed(req.Context(), workspaceItemID, input.TargetItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if !allowed {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("target_item_id 必须是当前治理条目或其后代"))
		return
	}
	if _, err := inventory.NewService(r.db).LinkAssetToItem(req.Context(), inventory.LinkAssetItemInput{
		AssetID: assetID,
		ItemID:  input.TargetItemID,
		Role:    inventory.AssetItemRolePrimary,
		Source:  "governance",
	}); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), workspaceItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleUnlinkCatalogGovernanceAsset(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	workspaceItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	assetID, err := parseUintPathValue(req, "asset_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	targetItemID, err := parseUintPathValue(req, "target_item_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	allowed, err := r.catalog.IsGovernanceTargetAllowed(req.Context(), workspaceItemID, targetItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if !allowed {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("target_item_id 必须是当前治理条目或其后代"))
		return
	}
	if err := inventory.NewService(r.db).UnlinkAssetFromItem(req.Context(), assetID, targetItemID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), workspaceItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleSearchCatalogItemMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input metadata.ManualSearchInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := validateManualSearchInput(input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	results, err := r.metadata.SearchCatalogCandidates(req.Context(), itemID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, results)
}

func (r *Router) handleApplyCatalogItemMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil || r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog metadata service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input metadata.ApplyCandidateInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.ExternalID) == "" {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("external_id is required"))
		return
	}
	if err := r.metadata.ApplyCatalogCandidate(req.Context(), itemID, input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleMatchCatalogItem(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil || r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog metadata service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := r.metadata.MatchCatalogItem(req.Context(), itemID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleRefetchCatalogItemMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil || r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog metadata service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := r.metadata.RefetchCatalogItem(req.Context(), itemID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleQueueInventoryFileProbe(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	job, err := r.library.QueueInventoryFileProbe(req.Context(), fileID, true)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) shouldServeCatalogLibraryItems(ctx context.Context) bool {
	if r.catalog == nil || r.settings == nil {
		return false
	}
	state, err := r.settings.GetCatalogMigrationState(ctx)
	if err != nil {
		return false
	}
	return state.CatalogReadEnabled
}

func normalizeCatalogListItemsArtworkURLs(req *http.Request, items []catalog.CatalogListItem) {
	for idx := range items {
		normalizeCatalogListItemArtworkURLs(req, &items[idx])
	}
}

func normalizeCatalogListItemArtworkURLs(req *http.Request, item *catalog.CatalogListItem) {
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

func normalizeCatalogItemDetailArtworkURLs(req *http.Request, item *catalog.CatalogItemDetail) {
	if item == nil {
		return
	}
	for idx := range item.SelectedImages {
		item.SelectedImages[idx].URL = buildAssetURL(req, item.SelectedImages[idx].URL)
	}
	normalizeCatalogSeasonDetailsArtworkURLs(req, item.Seasons)
	normalizeCatalogEpisodeDetailsArtworkURLs(req, item.Episodes)
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
	normalizeCatalogListItemsArtworkURLs(req, workspace.RecommendedChildren)
}

func (r *Router) markLegacyMediaEndpointDeprecated(req *http.Request, w http.ResponseWriter, successor string) {
	if !r.shouldServeCatalogLibraryItems(req.Context()) {
		return
	}
	w.Header().Set("Deprecation", "true")
	if strings.TrimSpace(successor) != "" {
		w.Header().Add("Link", "<"+requestBaseURL(req)+successor+">; rel=\"successor-version\"")
	}
	w.Header().Set("Sunset", "Wed, 31 Dec 2026 23:59:59 GMT")
	w.Header().Set("X-Mibo-Legacy-Endpoint", "media-items")
	if strings.TrimSpace(req.PathValue("id")) != "" {
		w.Header().Set("X-Mibo-Catalog-Item-ID", req.PathValue("id"))
	}
	if rawLibraryID := strings.TrimSpace(req.PathValue("id")); rawLibraryID != "" {
		if _, err := strconv.ParseUint(rawLibraryID, 10, 64); err == nil && strings.Contains(req.URL.Path, "/libraries/") {
			w.Header().Set("X-Mibo-Catalog-Read-Enabled", "true")
		}
	}
}

func (r *Router) rejectLegacyMediaEndpoint(req *http.Request, w http.ResponseWriter, successor string) bool {
	if !r.shouldServeCatalogLibraryItems(req.Context()) {
		return false
	}
	r.markLegacyMediaEndpointDeprecated(req, w, successor)
	writeError(req.Context(), w, http.StatusGone, fmt.Errorf("legacy media endpoint retired; use %s", successor))
	return true
}
