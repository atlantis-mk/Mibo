package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/atlan/mibo-media-server/internal/metadata"
)

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
	libraryID, err := parseOptionalUintQuery(req, "library_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID, libraryID)
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
	if r.catalog == nil || r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog metadata service unavailable"))
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
	operation, err := r.metadata.ApplyGovernanceMetadataFieldOperation(req.Context(), metadata.ApplyGovernanceMetadataFieldInput{
		MetadataItemID: itemID,
		FieldKey:       input.FieldKey,
		Value:          input.Value,
		Lock:           input.Lock,
		LockReason:     input.LockReason,
		EditedByUserID: &user.ID,
		Force:          input.Force,
	})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	libraryID, _ := parseOptionalUintQuery(req, "library_id")
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID, libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	workspace.MetadataOperation = metadata.OperationResponseFromResult(operation)
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleSelectCatalogGovernanceImage(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil || r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog metadata service unavailable"))
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
	operation, err := r.metadata.SelectGovernanceMetadataImageOperation(req.Context(), metadata.SelectGovernanceImageInput{MetadataItemID: itemID, ImageType: input.ImageType, URL: input.URL, EditedByUserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	libraryID, _ := parseOptionalUintQuery(req, "library_id")
	workspace, err := r.catalog.GetGovernanceWorkspace(req.Context(), itemID, libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	workspace.MetadataOperation = metadata.OperationResponseFromResult(operation)
	normalizeCatalogGovernanceWorkspaceArtworkURLs(req, &workspace)
	writeJSON(req.Context(), w, http.StatusOK, workspace)
}

func (r *Router) handleLinkGovernanceResource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}
	metadataItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	resourceID, err := parseUintPathValue(req, "resource_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceResourceLinkInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if input.TargetMetadataItemID == 0 {
		input.TargetMetadataItemID = metadataItemID
	}
	operation, err := r.metadata.LinkGovernanceResourceOperation(req.Context(), metadata.LinkGovernanceResourceInput{ResourceID: resourceID, TargetMetadataItemID: input.TargetMetadataItemID, SourceMetadataItemID: input.SourceMetadataItemID, LibraryID: input.LibraryID, Mode: input.Mode, Role: input.Role, SegmentIndex: input.SegmentIndex, StartSeconds: input.StartSeconds, EndSeconds: input.EndSeconds})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, metadata.OperationResponseFromResult(operation))
}

func (r *Router) handleUnlinkGovernanceResource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}
	resourceID, err := parseUintPathValue(req, "resource_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	metadataItemID, err := parseUintPathValue(req, "target_metadata_item_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	libraryID, err := parseOptionalUintQuery(req, "library_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	segmentIndex, _, err := parseOptionalIntQuery(req, "segment_index")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	operation, err := r.metadata.UnlinkGovernanceResourceOperation(req.Context(), metadata.UnlinkGovernanceResourceInput{ResourceID: resourceID, MetadataItemID: metadataItemID, Role: req.URL.Query().Get("role"), SegmentIndex: segmentIndex, LibraryID: libraryID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, metadata.OperationResponseFromResult(operation))
}

func (r *Router) handleUpdateGovernanceResourceLink(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}
	resourceID, err := parseUintPathValue(req, "resource_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	metadataItemID, err := parseUintPathValue(req, "target_metadata_item_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceResourceLinkUpdateInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	operation, err := r.metadata.UpdateGovernanceResourceLinkOperation(req.Context(), metadata.UpdateGovernanceResourceLinkInput{ResourceID: resourceID, MetadataItemID: metadataItemID, LibraryID: input.LibraryID, Role: input.Role, SegmentIndex: input.SegmentIndex, NewRole: input.NewRole, ReviewState: input.ReviewState})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, metadata.OperationResponseFromResult(operation))
}

func (r *Router) handleMergeGovernanceMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}
	sourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceMetadataMergeInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	operation, err := r.metadata.MergeGovernanceMetadataOperation(req.Context(), metadata.MergeGovernanceMetadataInput{SourceMetadataItemID: sourceID, TargetMetadataItemID: input.TargetMetadataItemID, LibraryID: input.LibraryID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, metadata.OperationResponseFromResult(operation))
}

func (r *Router) handleSplitGovernanceMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}
	sourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceMetadataSplitInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	operation, err := r.metadata.SplitGovernanceMetadataOperation(req.Context(), metadata.SplitGovernanceMetadataInput{SourceMetadataItemID: sourceID, TargetMetadataItemID: input.TargetMetadataItemID, ResourceIDs: input.ResourceIDs, LibraryID: input.LibraryID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, metadata.OperationResponseFromResult(operation))
}

func (r *Router) handleSetGovernanceProjectionVisibility(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}
	metadataItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input catalogGovernanceProjectionVisibilityInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	operation, err := r.metadata.SetGovernanceProjectionVisibilityOperation(req.Context(), metadata.SetGovernanceProjectionVisibilityInput{MetadataItemID: metadataItemID, LibraryID: input.LibraryID, Hidden: input.Hidden})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, metadata.OperationResponseFromResult(operation))
}
