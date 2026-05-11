package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
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

func sourceLabel(value string) string {
	if strings.TrimSpace(value) != "" {
		return "env"
	}
	return "none"
}
