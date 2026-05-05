package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "mibo-media-server",
	})
}

func (r *Router) handleReady(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
	defer cancel()

	sqlDB, err := r.db.DB()
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		writeError(req.Context(), w, http.StatusServiceUnavailable, fmt.Errorf("database not ready: %w", err))
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"status":   "ready",
		"database": r.cfg.Database.Driver,
	})
}

func (r *Router) handleSystemInfo(w http.ResponseWriter, req *http.Request) {
	tmdbSettings := map[string]any{"configured": r.cfg.Metadata.TMDB.APIKey != "", "source": sourceLabel(r.cfg.Metadata.TMDB.APIKey)}
	providerInstances := []map[string]any{}
	if r.settings != nil {
		resolved, err := r.settings.ListMetadataProviderInstances(req.Context())
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		for _, instance := range resolved {
			providerInstances = append(providerInstances, map[string]any{
				"id":                  instance.ID,
				"name":                instance.Name,
				"provider_type":       instance.ProviderType,
				"configured":          instance.Configured,
				"enabled":             instance.Enabled,
				"availability_status": instance.AvailabilityStatus,
			})
			if instance.ProviderType == database.MetadataProviderTypeTMDB && instance.Configured {
				tmdbSettings = map[string]any{
					"configured": true,
					"source":     "database",
				}
			}
		}
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"service":                     "mibo-media-server",
		"database":                    r.cfg.Database.Driver,
		"available_storage_providers": r.storage.Names(),
		"modules": map[string]any{
			"auth":    "active",
			"worker":  map[string]any{"enabled": r.cfg.Worker.Enabled},
			"library": "active",
			"jobs":    "active",
			"metadata": map[string]any{
				"tmdb_configured": tmdbSettings["configured"],
				"providers":       providerInstances,
			},
			"ffmpeg":   map[string]any{"enabled": r.cfg.FFmpeg.Enabled, "path": r.cfg.FFmpeg.Path},
			"ffprobe":  map[string]any{"enabled": r.cfg.FFprobe.Enabled, "path": r.cfg.FFprobe.Path},
			"playback": r.playback.Status(),
			"progress": r.progress.Status(),
			"search":   r.search.Status(),
		},
	})
}

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

func (r *Router) handleGetNetworkSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	result, err := r.settings.GetNetworkSettings(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleUpdateNetworkSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	var input settings.UpdateNetworkSettingsInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	result, err := r.settings.UpdateNetworkSettings(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleGetScanSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	result, err := r.settings.GetScanSettings(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleUpdateScanSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	var input settings.UpdateScanSettingsInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if input.RefreshIntervalHours <= 0 || input.RefreshIntervalHours > 720 {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("refresh_interval_hours must be between 1 and 720"))
		return
	}
	result, err := r.settings.UpdateScanSettings(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func sourceLabel(value string) string {
	if strings.TrimSpace(value) != "" {
		return "env"
	}
	return "none"
}

func (r *Router) handleSetupStatus(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var userCount int64
	if err := r.db.WithContext(ctx).Model(&database.User{}).Count(&userCount).Error; err != nil {
		writeError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	var mediaSourceCount int64
	if err := r.db.WithContext(ctx).Model(&database.MediaSource{}).Count(&mediaSourceCount).Error; err != nil {
		writeError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	var libraryCount int64
	if err := r.db.WithContext(ctx).Model(&database.Library{}).Count(&libraryCount).Error; err != nil {
		writeError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(ctx, w, http.StatusOK, map[string]any{
		"initialized":        userCount > 0 && mediaSourceCount > 0 && libraryCount > 0,
		"can_enter_app":      userCount > 0,
		"has_users":          userCount > 0,
		"has_media_sources":  mediaSourceCount > 0,
		"has_libraries":      libraryCount > 0,
		"user_count":         userCount,
		"media_source_count": mediaSourceCount,
		"library_count":      libraryCount,
	})
}
