package httpapi

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

var proxiedStreamHeaders = []string{
	"Accept-Ranges",
	"Cache-Control",
	"Content-Disposition",
	"Content-Length",
	"Content-Range",
	"Content-Type",
	"ETag",
	"Last-Modified",
}

type Router struct {
	cfg      config.Config
	db       *gorm.DB
	storage  *providers.Registry
	auth     *auth.Service
	library  *library.Service
	jobs     *jobs.Service
	playback *playback.Service
	hls      *hlsService
	progress *progress.Service
	search   *search.Service
	metadata *metadata.Service
	settings *settings.Service
}

type homeDiscoveryResponse struct {
	ContinueWatching []progress.Entry                 `json:"continue_watching"`
	RecentlyPlayed   []progress.Entry                 `json:"recently_played"`
	LatestByLibrary  []library.LatestByLibrarySection `json:"latest_by_library"`
}

func New(cfg config.Config, db *gorm.DB, registry *providers.Registry, authSvc *auth.Service, librarySvc *library.Service, jobsSvc *jobs.Service, playbackSvc *playback.Service, progressSvc *progress.Service, searchSvc *search.Service, metadataSvc *metadata.Service, settingsSvc *settings.Service) http.Handler {
	router := &Router{
		cfg:      cfg,
		db:       db,
		storage:  registry,
		auth:     authSvc,
		library:  librarySvc,
		jobs:     jobsSvc,
		playback: playbackSvc,
		hls:      newHLSService(cfg, db, registry),
		progress: progressSvc,
		search:   searchSvc,
		metadata: metadataSvc,
		settings: settingsSvc,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", router.handleHealth)
	mux.HandleFunc("GET /readyz", router.handleReady)
	mux.HandleFunc("GET /api/v1/setup/status", router.handleSetupStatus)
	mux.HandleFunc("POST /api/v1/auth/register", router.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", router.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", router.handleLogout)
	mux.HandleFunc("GET /api/v1/me", router.handleMe)
	mux.HandleFunc("POST /api/v1/me/progress", router.handleUpdateProgress)
	mux.HandleFunc("GET /api/v1/me/continue-watching", router.handleContinueWatching)
	mux.HandleFunc("GET /api/v1/me/recently-played", router.handleRecentlyPlayed)
	mux.HandleFunc("GET /api/v1/home/discovery", router.handleHomeDiscovery)
	mux.HandleFunc("GET /api/v1/home/recently-added", router.handleRecentlyAdded)
	mux.HandleFunc("GET /api/v1/system/info", router.handleSystemInfo)
	mux.HandleFunc("GET /api/v1/settings/metadata", router.handleGetMetadataSettings)
	mux.HandleFunc("PUT /api/v1/settings/metadata", router.handleUpdateMetadataSettings)
	mux.HandleFunc("GET /api/v1/settings/scan", router.handleGetScanSettings)
	mux.HandleFunc("PUT /api/v1/settings/scan", router.handleUpdateScanSettings)
	mux.HandleFunc("GET /api/v1/storage/providers/{provider}/browse", router.handleBrowseStorageProvider)
	mux.HandleFunc("POST /api/v1/storage/openlist/test", router.handleTestTemporaryOpenList)
	mux.HandleFunc("POST /api/v1/storage/openlist/browse", router.handleBrowseTemporaryOpenList)
	mux.HandleFunc("GET /api/v1/media-sources", router.handleListMediaSources)
	mux.HandleFunc("POST /api/v1/media-sources", router.handleCreateMediaSource)
	mux.HandleFunc("PATCH /api/v1/media-sources/{id}", router.handleUpdateMediaSource)
	mux.HandleFunc("DELETE /api/v1/media-sources/{id}", router.handleDeleteMediaSource)
	mux.HandleFunc("GET /api/v1/media-sources/{id}/browse", router.handleBrowseMediaSource)
	mux.HandleFunc("GET /api/v1/libraries", router.handleListLibraries)
	mux.HandleFunc("POST /api/v1/libraries", router.handleCreateLibrary)
	mux.HandleFunc("GET /api/v1/libraries/{id}", router.handleGetLibrary)
	mux.HandleFunc("DELETE /api/v1/libraries/{id}", router.handleDeleteLibrary)
	mux.HandleFunc("POST /api/v1/libraries/{id}/scan", router.handleQueueLibraryScan)
	mux.HandleFunc("GET /api/v1/libraries/{id}/items", router.handleListLibraryItems)
	mux.HandleFunc("GET /api/v1/media-items/{id}", router.handleGetMediaItem)
	mux.HandleFunc("GET /api/v1/tv/{tmdb_id}/seasons", router.handleListTVSeasons)
	mux.HandleFunc("GET /api/v1/tv/{tmdb_id}/seasons/{n}/episodes", router.handleListTVSeasonEpisodes)
	mux.HandleFunc("GET /api/v1/media-items/{id}/progress", router.handleGetMediaItemProgress)
	mux.HandleFunc("POST /api/v1/media-items/{id}/metadata/apply", router.handleApplyMediaItemMetadata)
	mux.HandleFunc("POST /api/v1/media-items/{id}/metadata/search", router.handleSearchMediaItemMetadata)
	mux.HandleFunc("POST /api/v1/media-items/{id}/match", router.handleQueueMediaItemMatch)
	mux.HandleFunc("GET /api/v1/media-items/{id}/playback", router.handleGetPlaybackSource)
	mux.HandleFunc("GET /api/v1/media-files/{id}/link", router.handleGetMediaFileLink)
	mux.HandleFunc("GET /api/v1/media-files/{id}/hls/index.m3u8", router.handleGetHLSPlaylist)
	mux.HandleFunc("GET /api/v1/media-files/{id}/hls/{name}", router.handleGetHLSArtifact)
	mux.HandleFunc("GET /api/v1/media-files/{id}/stream", router.handleStreamMediaFile)
	mux.HandleFunc("GET /api/v1/jobs", router.handleListJobs)
	mux.HandleFunc("POST /api/v1/jobs/{id}/retry", router.handleRetryJob)

	return corsMiddleware(cfg.CORS, loggingMiddleware(mux))
}

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

	provider, err := r.storage.Get(configuredStorageProvider(r.cfg))
	if err != nil {
		writeError(req.Context(), w, http.StatusServiceUnavailable, err)
		return
	}
	if _, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: storageRootPath(r.cfg)}); err != nil {
		writeError(req.Context(), w, http.StatusServiceUnavailable, fmt.Errorf("storage provider not ready: %w", err))
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"status":   "ready",
		"database": r.cfg.Database.Driver,
		"storage":  provider.Name(),
	})
}

func (r *Router) handleSystemInfo(w http.ResponseWriter, req *http.Request) {
	provider, err := r.storage.Get(configuredStorageProvider(r.cfg))
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	caps, err := provider.Capabilities(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	tmdbSettings, tvdbSettings := map[string]any{"configured": r.cfg.Metadata.TMDB.APIKey != "", "source": sourceLabel(r.cfg.Metadata.TMDB.APIKey)}, map[string]any{"configured": r.cfg.Metadata.TVDB.APIKey != "", "source": sourceLabel(r.cfg.Metadata.TVDB.APIKey)}
	if r.settings != nil {
		resolved, err := r.settings.GetMetadataSettings(req.Context())
		if err != nil {
			writeError(req.Context(), w, http.StatusInternalServerError, err)
			return
		}
		tmdbSettings = map[string]any{
			"configured": resolved.TMDB.Configured,
			"source":     resolved.TMDB.Source,
		}
		tvdbSettings = map[string]any{
			"configured": resolved.TVDB.Configured,
			"source":     resolved.TVDB.Source,
		}
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"service":                     "mibo-media-server",
		"database":                    r.cfg.Database.Driver,
		"available_storage_providers": r.storage.Names(),
		"storage_provider": map[string]any{
			"name":         provider.Name(),
			"root_path":    storageRootPath(r.cfg),
			"capabilities": caps,
		},
		"modules": map[string]any{
			"auth":    "active",
			"worker":  map[string]any{"enabled": r.cfg.Worker.Enabled},
			"library": "active",
			"jobs":    "active",
			"metadata": map[string]any{
				"tmdb_configured": tmdbSettings["configured"],
				"providers": map[string]any{
					"tmdb": tmdbSettings,
					"tvdb": tvdbSettings,
				},
			},
			"ffmpeg":   map[string]any{"enabled": r.cfg.FFmpeg.Enabled, "path": r.cfg.FFmpeg.Path},
			"ffprobe":  map[string]any{"enabled": r.cfg.FFprobe.Enabled, "path": r.cfg.FFprobe.Path},
			"hls":      map[string]any{"enabled": r.hls.Enabled(), "root_path": r.cfg.HLS.RootPath, "segment_duration": r.cfg.HLS.SegmentDuration},
			"playback": r.playback.Status(),
			"progress": r.progress.Status(),
			"search":   r.search.Status(),
		},
	})
}

func (r *Router) handleGetMetadataSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	result, err := r.settings.GetMetadataSettings(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleUpdateMetadataSettings(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.settings == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("settings service unavailable"))
		return
	}
	var input settings.UpdateMetadataSettingsInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	result, err := r.settings.UpdateMetadataSettings(req.Context(), input)
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

func (r *Router) handleRegister(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	user, err := r.auth.Register(req.Context(), input.Username, input.Password)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusCreated, user)
}

func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.auth.Login(req.Context(), input.Username, input.Password)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleLogout(w http.ResponseWriter, req *http.Request) {
	token, err := bearerToken(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if err := r.auth.Logout(req.Context(), token); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (r *Router) handleMe(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, user)
}

func (r *Router) handleUpdateProgress(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input progress.UpdateInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	state, err := r.progress.Update(req.Context(), user.ID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}

func (r *Router) handleContinueWatching(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	entries, err := r.progress.ContinueWatching(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func (r *Router) handleRecentlyPlayed(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	entries, err := r.progress.RecentlyPlayed(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func (r *Router) handleRecentlyAdded(w http.ResponseWriter, req *http.Request) {
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	items, err := r.library.ListRecentlyAdded(req.Context(), limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, items)
}

func (r *Router) handleHomeDiscovery(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	continueWatching, err := r.progress.ContinueWatching(req.Context(), user.ID, 20)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	recentlyPlayed, err := r.progress.RecentlyPlayed(req.Context(), user.ID, 20)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	latestByLibrary, err := r.library.ListLatestByLibrary(req.Context(), 12)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, homeDiscoveryResponse{
		ContinueWatching: continueWatching,
		RecentlyPlayed:   recentlyPlayed,
		LatestByLibrary:  latestByLibrary,
	})
}

func (r *Router) handleCreateMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input library.CreateMediaSourceInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	source, err := r.library.CreateMediaSource(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	view, err := r.library.MediaSourceView(source)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusCreated, view)
}

func (r *Router) handleBrowseStorageProvider(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	providerName := strings.TrimSpace(req.PathValue("provider"))
	result, err := r.library.BrowseProviderPath(req.Context(), providerName, req.URL.Query().Get("path"))
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleBrowseTemporaryOpenList(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input struct {
		Path   string                         `json:"path"`
		Config providers.OpenListSourceConfig `json:"config"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.library.BrowseTemporaryOpenListPath(req.Context(), input.Config, input.Path)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleTestTemporaryOpenList(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input struct {
		Config providers.OpenListSourceConfig `json:"config"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.library.TestTemporaryOpenListConnection(req.Context(), input.Config)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleUpdateMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	sourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var input library.UpdateMediaSourceInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	source, err := r.library.UpdateMediaSource(req.Context(), sourceID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	view, err := r.library.MediaSourceView(source)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, view)
}

func (r *Router) handleListMediaSources(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	sources, err := r.library.ListMediaSourceViews(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, sources)
}

func (r *Router) handleDeleteMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	sourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	if err := r.library.DeleteMediaSource(req.Context(), sourceID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"id":     sourceID,
		"status": "deleted",
		"type":   "media_source",
	})
}

func (r *Router) handleBrowseMediaSource(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	sourceID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.library.BrowseMediaSourcePath(req.Context(), sourceID, req.URL.Query().Get("path"))
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleCreateLibrary(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input library.CreateLibraryInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	libraryRecord, job, err := r.library.CreateLibrary(req.Context(), input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusCreated, map[string]any{
		"library": libraryRecord,
		"job":     job,
	})
}

func (r *Router) handleListLibraries(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	libraries, err := r.library.ListLibraries(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, libraries)
}

func (r *Router) handleGetLibrary(w http.ResponseWriter, req *http.Request) {
	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	record, err := r.library.GetLibrary(req.Context(), libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, record)
}

func (r *Router) handleDeleteLibrary(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	if err := r.library.DeleteLibrary(req.Context(), libraryID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, map[string]any{
		"id":     libraryID,
		"status": "deleted",
		"type":   "library",
	})
}

func (r *Router) handleQueueLibraryScan(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.library.QueueLibraryScan(req.Context(), libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) handleListLibraryItems(w http.ResponseWriter, req *http.Request) {
	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	browseInput := library.BrowseMediaItemsInput{
		LibraryID:  libraryID,
		Scope:      library.BrowseScopeLibrary,
		TypeFilter: normalizeBrowseTypeFilter(req.URL.Query().Get("type")),
		Year:       library.ParseBrowseYear(req.URL.Query().Get("year")),
		Sort:       normalizeBrowseSort(req.URL.Query().Get("sort")),
		Limit:      limit,
	}

	items, err := r.library.BrowseMediaItems(req.Context(), browseInput)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, items)
}

func (r *Router) handleGetMediaItem(w http.ResponseWriter, req *http.Request) {
	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	item, err := r.library.GetMediaItem(req.Context(), mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, item)
}

func (r *Router) handleListTVSeasons(w http.ResponseWriter, req *http.Request) {
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	tmdbID, err := parseIntPathValue(req, "tmdb_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	seasons, err := r.metadata.ListTVSeasons(req.Context(), tmdbID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, seasons)
}

func (r *Router) handleListTVSeasonEpisodes(w http.ResponseWriter, req *http.Request) {
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	tmdbID, err := parseIntPathValue(req, "tmdb_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	seasonNumber, err := parseIntPathValue(req, "n")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var libraryID *uint
	if rawLibraryID := strings.TrimSpace(req.URL.Query().Get("library_id")); rawLibraryID != "" {
		parsedLibraryID, err := strconv.ParseUint(rawLibraryID, 10, 64)
		if err != nil || parsedLibraryID == 0 {
			writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("library_id must be a positive integer"))
			return
		}
		value := uint(parsedLibraryID)
		libraryID = &value
	}

	episodes, err := r.metadata.ListSeasonEpisodes(req.Context(), tmdbID, seasonNumber, libraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, episodes)
}

func (r *Router) handleGetMediaItemProgress(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	state, err := r.progress.GetState(req.Context(), user.ID, mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}

func (r *Router) handleQueueMediaItemMatch(w http.ResponseWriter, req *http.Request) {
	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.library.QueueMediaItemMatch(req.Context(), mediaItemID, true)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func (r *Router) handleSearchMediaItemMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var input metadata.ManualSearchInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	results, err := r.metadata.SearchCandidates(req.Context(), mediaItemID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, results)
}

func (r *Router) handleApplyMediaItemMetadata(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.metadata == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("metadata service unavailable"))
		return
	}

	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var input metadata.ApplyCandidateInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	if err := r.metadata.ApplyCandidate(req.Context(), mediaItemID, input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	item, err := r.library.GetMediaItem(req.Context(), mediaItemID)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, item)
}

func (r *Router) handleGetPlaybackSource(w http.ResponseWriter, req *http.Request) {
	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	preferredFileID, err := parseOptionalUintQuery(req, "file_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	source, err := r.playback.GetPlaybackSource(req.Context(), mediaItemID, preferredFileID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if r.hls.Enabled() {
		source.URL = r.hls.PlaylistURL(source.MediaFileID)
		source.Container = "m3u8"
		source.Direct = false
	}
	source.URL = buildPlaybackURL(req, source.URL)

	writeJSON(req.Context(), w, http.StatusOK, source)
}

func (r *Router) handleGetMediaFileLink(w http.ResponseWriter, req *http.Request) {
	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	link, err := r.playback.GetFileLink(req.Context(), mediaFileID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	link.URL = buildPlaybackURL(req, link.URL)

	writeJSON(req.Context(), w, http.StatusOK, link)
}

func (r *Router) handleGetHLSPlaylist(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	playlistPath, err := r.hls.EnsurePlaylist(req.Context(), mediaFileID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	playlistBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write(rewriteHLSPlaylist(req, mediaFileID, playlistBytes))
}

func (r *Router) handleGetHLSArtifact(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	name := req.PathValue("name")
	if _, err := r.hls.EnsurePlaylist(req.Context(), mediaFileID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	artifactPath, err := r.hls.ArtifactPath(mediaFileID, name)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if strings.HasSuffix(strings.ToLower(name), ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
	}
	http.ServeFile(w, req, artifactPath)
}

func (r *Router) handleStreamMediaFile(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var file database.MediaFile
	if err := r.db.WithContext(req.Context()).Where("id = ? AND deleted_at IS NULL", mediaFileID).First(&file).Error; err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	provider, err := r.providerForLibrary(req.Context(), file.LibraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	object, err := provider.Get(req.Context(), storage.GetRequest{Path: file.StoragePath})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if object.IsDir {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("selected media file is a directory"))
		return
	}

	if provider.Name() == "local" {
		localPath := strings.TrimSpace(object.RawURL)
		if localPath == "" {
			writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("local file path unavailable for %s", file.StoragePath))
			return
		}
		if _, err := os.Stat(localPath); err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}

		if object.Modified != nil {
			w.Header().Set("Last-Modified", object.Modified.UTC().Format(http.TimeFormat))
		}
		http.ServeFile(w, req, localPath)
		return
	}

	streamURL := strings.TrimSpace(object.RawURL)
	if streamURL == "" {
		link, err := provider.Link(req.Context(), storage.LinkRequest{Path: file.StoragePath})
		if err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		streamURL = strings.TrimSpace(link.URL)
	}
	if streamURL == "" {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("stream url unavailable for %s", file.StoragePath))
		return
	}

	upstreamReq, err := http.NewRequestWithContext(req.Context(), http.MethodGet, streamURL, nil)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	copyRequestHeader(req.Header, upstreamReq.Header, "Range")
	copyRequestHeader(req.Header, upstreamReq.Header, "If-Range")
	copyRequestHeader(req.Header, upstreamReq.Header, "If-Modified-Since")

	upstreamResp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadGateway, err)
		return
	}
	defer upstreamResp.Body.Close()

	copySelectedHeaders(upstreamResp.Header, w.Header(), proxiedStreamHeaders)
	w.WriteHeader(upstreamResp.StatusCode)
	if _, err := io.Copy(w, upstreamResp.Body); err != nil {
		log.Printf("http stream proxy path=%s error=%v", file.StoragePath, err)
	}
}

func (r *Router) handleListJobs(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	status := req.URL.Query().Get("status")
	kind := req.URL.Query().Get("kind")
	jobList, err := r.jobs.List(req.Context(), limit, status, kind)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, jobList)
}

func (r *Router) handleRetryJob(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	jobID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.jobs.Retry(req.Context(), jobID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, job)
}

func decodeJSON(req *http.Request, out any) error {
	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}

	if decoder.More() {
		return errors.New("request body must contain a single JSON document")
	}

	return nil
}

func writeError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	writeResponse(ctx, w, status, nil, &responseError{
		Code:    codeForStatus(status),
		Message: err.Error(),
	})
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	writeResponse(ctx, w, status, data, nil)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestID := newRequestID()
		ctx := context.WithValue(req.Context(), requestIDContextKey, requestID)
		req = req.WithContext(ctx)

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		recorder.Header().Set("X-Request-ID", requestID)
		startedAt := time.Now()

		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("http request_id=%s method=%s path=%s panic=%v", requestID, req.Method, req.URL.Path, recovered)
				writeError(ctx, recorder, http.StatusInternalServerError, errors.New("internal server error"))
			}
			log.Printf(
				"http request_id=%s method=%s path=%s status=%d duration=%s",
				requestID,
				req.Method,
				req.URL.Path,
				recorder.status,
				time.Since(startedAt).Round(time.Millisecond),
			)
		}()

		next.ServeHTTP(recorder, req)
	})
}

func corsMiddleware(cfg config.CORSConfig, next http.Handler) http.Handler {
	allowedOrigins := cfg.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		switch {
		case len(allowedOrigins) == 1 && allowedOrigins[0] == "*":
			w.Header().Set("Access-Control-Allow-Origin", "*")
		case origin != "" && containsOrigin(allowedOrigins, origin):
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, req)
	})
}

type contextKey string

const requestIDContextKey contextKey = "request_id"

type envelope struct {
	RequestID string         `json:"request_id"`
	Data      any            `json:"data,omitempty"`
	Error     *responseError `json:"error,omitempty"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func writeResponse(ctx context.Context, w http.ResponseWriter, status int, data any, err *responseError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{
		RequestID: requestIDFromContext(ctx),
		Data:      data,
		Error:     err,
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		return "internal_error"
	}
}

func newRequestID() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(raw)
}

func parseUintPathValue(req *http.Request, key string) (uint, error) {
	value := req.PathValue(key)
	if value == "" {
		return 0, fmt.Errorf("missing path parameter %q", key)
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid path parameter %q", key)
	}

	return uint(parsed), nil
}

func parseIntPathValue(req *http.Request, key string) (int, error) {
	value := req.PathValue(key)
	if value == "" {
		return 0, fmt.Errorf("missing path parameter %q", key)
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid path parameter %q", key)
	}

	return parsed, nil
}

func parseOptionalUintQuery(req *http.Request, key string) (uint, error) {
	value := req.URL.Query().Get(key)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid query parameter %q", key)
	}
	return uint(parsed), nil
}

func containsOrigin(origins []string, target string) bool {
	for _, origin := range origins {
		if origin == target {
			return true
		}
	}
	return false
}

func bearerToken(req *http.Request) (string, error) {
	authorization := strings.TrimSpace(req.Header.Get("Authorization"))
	if authorization == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return "", fmt.Errorf("authorization header must use Bearer token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
	if token == "" {
		return "", fmt.Errorf("missing bearer token")
	}
	return token, nil
}

func playbackToken(req *http.Request) (string, error) {
	token := strings.TrimSpace(req.URL.Query().Get("access_token"))
	if token != "" {
		return token, nil
	}
	return bearerToken(req)
}

func copyRequestHeader(from http.Header, to http.Header, key string) {
	value := strings.TrimSpace(from.Get(key))
	if value != "" {
		to.Set(key, value)
	}
}

func copySelectedHeaders(from http.Header, to http.Header, keys []string) {
	for _, key := range keys {
		for _, value := range from.Values(key) {
			to.Add(key, value)
		}
	}
}

func (r *Router) requireUser(req *http.Request) (database.User, error) {
	token, err := bearerToken(req)
	if err != nil {
		return database.User{}, err
	}
	return r.auth.Authenticate(req.Context(), token)
}

func (r *Router) requirePlaybackUser(req *http.Request) (database.User, error) {
	token, err := playbackToken(req)
	if err != nil {
		return database.User{}, err
	}
	return r.auth.Authenticate(req.Context(), token)
}

func buildPlaybackURL(req *http.Request, rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return trimmed
	}
	if !strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	trimmed = requestBaseURL(req) + trimmed
	token := strings.TrimSpace(req.URL.Query().Get("access_token"))
	if token == "" {
		token, _ = bearerToken(req)
	}
	if token == "" {
		return trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	query := parsed.Query()
	query.Set("access_token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func normalizeBrowseTypeFilter(value string) library.BrowseTypeFilter {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(library.BrowseTypeFilterMovie):
		return library.BrowseTypeFilterMovie
	case string(library.BrowseTypeFilterShow):
		return library.BrowseTypeFilterShow
	default:
		return library.BrowseTypeFilterAll
	}
}

func normalizeBrowseSort(value string) library.BrowseSort {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(library.BrowseSortTitle):
		return library.BrowseSortTitle
	case string(library.BrowseSortYear):
		return library.BrowseSortYear
	case string(library.BrowseSortWatchStatus):
		return library.BrowseSortWatchStatus
	default:
		return library.BrowseSortRecent
	}
}

func rewriteHLSPlaylist(req *http.Request, mediaFileID uint, playlist []byte) []byte {
	scanner := bufio.NewScanner(strings.NewReader(string(playlist)))
	lines := make([]string, 0, 128)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "://") {
			lines = append(lines, line)
			continue
		}
		artifactPath := fmt.Sprintf("/api/v1/media-files/%d/hls/%s", mediaFileID, path.Base(trimmed))
		lines = append(lines, buildPlaybackURL(req, artifactPath))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func requestBaseURL(req *http.Request) string {
	scheme := "http"
	if forwardedProto := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	} else if req.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + req.Host
}

func storageRootPath(cfg config.Config) string {
	if configuredStorageProvider(cfg) == "local" {
		return cfg.Local.RootPath
	}
	return cfg.OpenList.RootPath
}

func configuredStorageProvider(cfg config.Config) string {
	providerName := strings.ToLower(strings.TrimSpace(cfg.Storage.Provider))
	if providerName == "" {
		return "openlist"
	}
	return providerName
}

func (r *Router) providerForLibrary(ctx context.Context, libraryID uint) (storage.Provider, error) {
	var libraryRecord database.Library
	if err := r.db.WithContext(ctx).First(&libraryRecord, libraryID).Error; err != nil {
		return nil, err
	}
	var source database.MediaSource
	if err := r.db.WithContext(ctx).First(&source, libraryRecord.MediaSourceID).Error; err != nil {
		return nil, err
	}
	return r.storage.BuildForSource(source)
}
