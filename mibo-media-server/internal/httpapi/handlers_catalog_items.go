package httpapi

import (
	"errors"
	"net/http"

	"github.com/atlan/mibo-media-server/internal/catalog"
)

func (r *Router) handleGetMetadataItem(w http.ResponseWriter, req *http.Request) {
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
	libraryID, _ := parseOptionalUintQuery(req, "library_id")
	detail, err := r.catalog.GetMetadataItemDetail(req.Context(), itemID, libraryID)
	_ = userID
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	normalizeMetadataItemDetailArtworkURLs(req, &detail)
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

func (r *Router) handleListMetadataItemResources(w http.ResponseWriter, req *http.Request) {
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
	resources, err := r.catalog.ListMetadataItemResources(req.Context(), itemID, libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, resources)
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
	state, err := r.progress.GetMetadataItemState(req.Context(), user.ID, itemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}
