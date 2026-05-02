package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/library"
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
	TargetItemID uint     `json:"target_item_id"`
	SourceItemID *uint    `json:"source_item_id"`
	Mode         string   `json:"mode"`
	SegmentIndex int      `json:"segment_index"`
	StartSeconds *float64 `json:"start_seconds"`
	EndSeconds   *float64 `json:"end_seconds"`
}

type catalogGovernanceEpisodeNumberingInput struct {
	SeasonNumber     int  `json:"season_number"`
	EpisodeNumber    int  `json:"episode_number"`
	EpisodeNumberEnd *int `json:"episode_number_end"`
}

type catalogGovernanceClassificationRuleInput struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	PathPattern     string `json:"path_pattern"`
	RuleType        string `json:"rule_type"`
	Role            string `json:"role"`
	CandidateType   string `json:"candidate_type"`
	SeriesTitle     string `json:"series_title"`
	SeasonNumber    *int   `json:"season_number"`
	NumberingSource string `json:"numbering_source"`
}

type scanExclusionMarkInput struct {
	Reason string `json:"reason"`
}

type filenameExclusionRestoreInput struct {
	InventoryFileID uint `json:"inventory_file_id"`
}

type scanExclusionEnabledInput struct {
	Enabled bool `json:"enabled"`
}

type scanExclusionRuleInput struct {
	LibraryID   *uint  `json:"library_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RuleType    string `json:"rule_type"`
	Value       string `json:"value"`
	Reason      string `json:"reason"`
	Enabled     *bool  `json:"enabled"`
}

type replaceLibraryScanExclusionRulesInput struct {
	Rules []scanExclusionRuleInput `json:"rules"`
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
	var userID *uint
	if user, err := r.optionalUser(req); err == nil && user != nil {
		id := user.ID
		userID = &id
	}
	detail, err := r.catalog.GetItemDetailForUser(req.Context(), itemID, userID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeCatalogItemDetailArtworkURLs(req, &detail)
	writeJSON(req.Context(), w, http.StatusOK, detail)
}

func (r *Router) handleGetMediaItem(w http.ResponseWriter, req *http.Request) {
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var userID *uint
	if user, err := r.optionalUser(req); err == nil && user != nil {
		id := user.ID
		userID = &id
	}
	detail, err := r.catalog.GetMediaItemDTO(req.Context(), itemID, userID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, detail)
}

func (r *Router) handleGetCatalogPerson(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	personID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	person, err := r.catalog.GetPersonDetail(req.Context(), personID)
	if err != nil {
		if catalog.IsPersonNotFound(err) {
			writeError(req.Context(), w, http.StatusNotFound, err)
			return
		}
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeCatalogPersonDetailArtworkURLs(req, &person)
	writeJSON(req.Context(), w, http.StatusOK, person)
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
	userID := user.ID
	source, err := r.playback.GetPlaybackSource(req.Context(), playback.PlaybackRequest{ItemID: itemID, AssetID: assetID, UserID: &userID, ClientProfile: clientProfile})
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

func (r *Router) handleCorrectCatalogEpisodeNumbering(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	episodeID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceEpisodeNumberingInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	updated, err := r.catalog.CorrectEpisodeNumbering(req.Context(), catalog.CorrectEpisodeNumberingInput{
		EpisodeID:        episodeID,
		SeasonNumber:     input.SeasonNumber,
		EpisodeNumber:    input.EpisodeNumber,
		EpisodeNumberEnd: input.EpisodeNumberEnd,
	})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), updated.ID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleCreateCatalogGovernanceClassificationRule(w http.ResponseWriter, req *http.Request) {
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
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceClassificationRuleInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	createdBy := user.ID
	if _, err := r.catalog.CreateClassificationRule(req.Context(), catalog.ClassificationRuleInput{
		LibraryID:       workspace.LibraryID,
		Name:            input.Name,
		Description:     input.Description,
		PathPattern:     input.PathPattern,
		RuleType:        input.RuleType,
		Role:            input.Role,
		CandidateType:   input.CandidateType,
		SeriesTitle:     input.SeriesTitle,
		SeasonNumber:    input.SeasonNumber,
		NumberingSource: input.NumberingSource,
		CreatedByUserID: &createdBy,
	}); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err = r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusCreated, workspace)
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
	if input.SourceItemID != nil {
		allowed, err := r.catalog.IsGovernanceTargetAllowed(req.Context(), workspaceItemID, *input.SourceItemID)
		if err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		if !allowed {
			writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("source_item_id 必须是当前治理条目或其后代"))
			return
		}
	}
	if _, err := inventory.NewService(r.db).LinkAssetToItem(req.Context(), inventory.LinkAssetItemInput{
		AssetID:      assetID,
		ItemID:       input.TargetItemID,
		Role:         inventory.AssetItemRolePrimary,
		SegmentIndex: input.SegmentIndex,
		StartSeconds: input.StartSeconds,
		EndSeconds:   input.EndSeconds,
		Source:       "governance",
	}); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if strings.EqualFold(strings.TrimSpace(input.Mode), "move") && input.SourceItemID != nil && *input.SourceItemID != input.TargetItemID {
		if err := inventory.NewService(r.db).UnlinkAssetFromItem(req.Context(), assetID, *input.SourceItemID); err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
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
	operation, err := r.metadata.ApplyCatalogCandidateOperation(req.Context(), itemID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	workspace.MetadataOperation = metadata.OperationResponseFromResult(operation)
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
	operation, err := r.metadata.MatchCatalogItemOperation(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	operationResponse := metadata.OperationResponseFromResult(operation)
	workspace.MetadataOperation = operationResponse
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
	operation, err := r.metadata.RefetchCatalogItemOperation(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	operationResponse := metadata.OperationResponseFromResult(operation)
	workspace.MetadataOperation = operationResponse
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

func (r *Router) handleMarkInventoryFileScanExclusion(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.markScanExclusion(w, req, library.MarkScanExclusionInput{InventoryFileID: fileID, UserID: &user.ID})
}

func (r *Router) handlePreviewInventoryFileFilenameExclusion(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.previewFilenameExclusion(w, req, library.FilenameExclusionTargetInput{InventoryFileID: fileID})
}

func (r *Router) handleMarkAssetScanExclusion(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	assetID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.markScanExclusion(w, req, library.MarkScanExclusionInput{AssetID: assetID, UserID: &user.ID})
}

func (r *Router) handlePreviewAssetFilenameExclusion(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	assetID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.previewFilenameExclusion(w, req, library.FilenameExclusionTargetInput{AssetID: assetID})
}

func (r *Router) handleMarkItemScanExclusion(w http.ResponseWriter, req *http.Request) {
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
	r.markScanExclusion(w, req, library.MarkScanExclusionInput{ItemID: itemID, UserID: &user.ID})
}

func (r *Router) handlePreviewItemFilenameExclusion(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.previewFilenameExclusion(w, req, library.FilenameExclusionTargetInput{ItemID: itemID})
}

func (r *Router) previewFilenameExclusion(w http.ResponseWriter, req *http.Request, input library.FilenameExclusionTargetInput) {
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	preview, err := r.library.PreviewFilenameExclusion(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, preview)
}

func (r *Router) handleCreateItemFilenameExclusionRule(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var body scanExclusionMarkInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	rule, err := r.library.CreateFilenameExclusionRule(req.Context(), library.CreateFilenameExclusionRuleInput{FilenameExclusionTargetInput: library.FilenameExclusionTargetInput{ItemID: itemID}, Reason: body.Reason, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rule)
}

func (r *Router) markScanExclusion(w http.ResponseWriter, req *http.Request, input library.MarkScanExclusionInput) {
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	var body scanExclusionMarkInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	input.Reason = body.Reason
	exclusion, err := r.library.MarkScanExclusion(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, exclusion)
}

func (r *Router) handleSetScanExclusionEnabled(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	exclusionID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input scanExclusionEnabledInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	exclusion, err := r.library.SetScanExclusionEnabled(req.Context(), library.SetScanExclusionEnabledInput{ExclusionID: exclusionID, Enabled: input.Enabled, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, exclusion)
}

func (r *Router) handleListScanExclusions(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	libraryID, err := parseOptionalUintQuery(req, "library_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var enabled *bool
	switch strings.ToLower(strings.TrimSpace(req.URL.Query().Get("enabled"))) {
	case "true", "1", "yes":
		value := true
		enabled = &value
	case "false", "0", "no":
		value := false
		enabled = &value
	case "", "all":
	default:
		writeError(req.Context(), w, http.StatusBadRequest, errors.New("invalid query parameter \"enabled\""))
		return
	}
	exclusions, err := r.library.ListScanExclusionsView(req.Context(), library.ListScanExclusionsInput{LibraryID: libraryID, Enabled: enabled})
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, exclusions)
}

func (r *Router) handleListFilenameExclusionRules(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	libraryID, err := parseOptionalUintQuery(req, "library_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var enabled *bool
	switch strings.ToLower(strings.TrimSpace(req.URL.Query().Get("enabled"))) {
	case "true", "1", "yes":
		value := true
		enabled = &value
	case "false", "0", "no":
		value := false
		enabled = &value
	case "", "all":
	default:
		writeError(req.Context(), w, http.StatusBadRequest, errors.New("invalid query parameter \"enabled\""))
		return
	}
	rules, err := r.library.ListFilenameExclusionRules(req.Context(), library.ListScanExclusionsInput{LibraryID: libraryID, Enabled: enabled})
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rules)
}

func (r *Router) handleSetFilenameExclusionRuleEnabled(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	ruleID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input scanExclusionEnabledInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	rule, err := r.library.SetFilenameExclusionRuleEnabled(req.Context(), library.SetFilenameExclusionRuleEnabledInput{RuleID: ruleID, Enabled: input.Enabled, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rule)
}

func (r *Router) handleRestoreFilenameExclusionMatch(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	ruleID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input filenameExclusionRestoreInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	restore, err := r.library.RestoreFilenameExclusionMatch(req.Context(), library.RestoreFilenameExclusionMatchInput{RuleID: ruleID, InventoryFileID: input.InventoryFileID, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, restore)
}

func (r *Router) handleListScanExclusionRules(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	rules, err := r.library.ListScanExclusionRules(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rules)
}

func (r *Router) handleCreateScanExclusionRule(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	var body scanExclusionRuleInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	rule, err := r.library.CreateScanExclusionRule(req.Context(), library.ScanExclusionRuleInput{LibraryID: body.LibraryID, Name: body.Name, Description: body.Description, RuleType: body.RuleType, Value: body.Value, Reason: body.Reason, Enabled: enabled, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusCreated, rule)
}

func (r *Router) handleUpdateScanExclusionRule(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	ruleID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var body scanExclusionRuleInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if body.Enabled != nil && body.LibraryID == nil && strings.TrimSpace(body.Name) == "" && strings.TrimSpace(body.RuleType) == "" && strings.TrimSpace(body.Value) == "" && strings.TrimSpace(body.Reason) == "" && strings.TrimSpace(body.Description) == "" {
		rule, err := r.library.SetScanExclusionRuleEnabled(req.Context(), library.SetScanExclusionRuleEnabledInput{RuleID: ruleID, Enabled: *body.Enabled, UserID: &user.ID})
		if err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		writeJSON(req.Context(), w, http.StatusOK, rule)
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	rule, err := r.library.UpdateScanExclusionRule(req.Context(), library.UpdateScanExclusionRuleInput{RuleID: ruleID, LibraryID: body.LibraryID, Name: body.Name, Description: body.Description, RuleType: body.RuleType, Value: body.Value, Reason: body.Reason, Enabled: enabled, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rule)
}

func (r *Router) handleDeleteScanExclusionRule(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	ruleID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := r.library.DeleteScanExclusionRule(req.Context(), ruleID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (r *Router) handleReplaceLibraryScanExclusionRules(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var body replaceLibraryScanExclusionRulesInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	inputs := make([]library.ScanExclusionRuleInput, 0, len(body.Rules))
	for _, rule := range body.Rules {
		enabled := true
		if rule.Enabled != nil {
			enabled = *rule.Enabled
		}
		inputs = append(inputs, library.ScanExclusionRuleInput{Name: rule.Name, Description: rule.Description, RuleType: rule.RuleType, Value: rule.Value, Reason: rule.Reason, Enabled: enabled})
	}
	rules, err := r.library.ReplaceLibraryScanExclusionRules(req.Context(), libraryID, inputs, &user.ID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rules)
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
	normalizeCatalogListItemsArtworkURLs(req, person.RelatedItems)
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
