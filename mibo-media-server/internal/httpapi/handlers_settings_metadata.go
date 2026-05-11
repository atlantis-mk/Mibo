package httpapi

import (
	"errors"
	"net/http"

	"github.com/atlan/mibo-media-server/internal/settings"
)

func (r *Router) handleListMetadataProviderInstances(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	result, err := r.settings.ListMetadataProviderInstances(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleCreateMetadataProviderInstance(w http.ResponseWriter, req *http.Request) {
	r.upsertMetadataProviderInstance(w, req, 0, http.StatusCreated)
}

func (r *Router) handleUpdateMetadataProviderInstance(w http.ResponseWriter, req *http.Request) {
	providerID, err := parseUintPathValue(req, "provider_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.upsertMetadataProviderInstance(w, req, providerID, http.StatusOK)
}

func (r *Router) upsertMetadataProviderInstance(w http.ResponseWriter, req *http.Request, providerID uint, status int) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	var input settings.UpdateMetadataProviderInstanceInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	result, err := r.settings.UpsertMetadataProviderInstance(req.Context(), providerID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, status, result)
}

func (r *Router) handleListMetadataProfiles(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	result, err := r.settings.ListMetadataProfiles(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleCreateMetadataProfile(w http.ResponseWriter, req *http.Request) {
	r.upsertMetadataProfile(w, req, 0, http.StatusCreated)
}

func (r *Router) handleUpdateMetadataProfile(w http.ResponseWriter, req *http.Request) {
	profileID, err := parseUintPathValue(req, "profile_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	r.upsertMetadataProfile(w, req, profileID, http.StatusOK)
}

func (r *Router) upsertMetadataProfile(w http.ResponseWriter, req *http.Request, profileID uint, status int) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	var input settings.UpdateMetadataProfileInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	result, err := r.settings.UpsertMetadataProfile(req.Context(), profileID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, status, result)
}
