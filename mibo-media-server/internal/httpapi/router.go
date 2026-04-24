package httpapi

import (
	"net/http"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/listener"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
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
	listener *listener.Service
	jobs     *jobs.Service
	playback *playback.Service
	hls      *hlsService
	progress *progress.Service
	search   *search.Service
	metadata *metadata.Service
	schedule *schedule.Service
	settings *settings.Service
}

type homeDiscoveryResponse struct {
	ContinueWatching []progress.Entry                 `json:"continue_watching"`
	RecentlyPlayed   []progress.Entry                 `json:"recently_played"`
	LatestByLibrary  []library.LatestByLibrarySection `json:"latest_by_library"`
}

func New(cfg config.Config, db *gorm.DB, registry *providers.Registry, authSvc *auth.Service, librarySvc *library.Service, jobsSvc *jobs.Service, playbackSvc *playback.Service, progressSvc *progress.Service, searchSvc *search.Service, metadataSvc *metadata.Service, settingsSvc *settings.Service, args ...any) http.Handler {
	scheduleSvc := schedule.NewService(db, schedule.WithJobs(jobsSvc))
	listenerSvc := listener.NewService(db, jobsSvc, librarySvc)
	for _, arg := range args {
		if provided, ok := arg.(*schedule.Service); ok && provided != nil {
			scheduleSvc = provided
		}
		if provided, ok := arg.(*listener.Service); ok && provided != nil {
			listenerSvc = provided
		}
	}
	router := &Router{
		cfg:      cfg,
		db:       db,
		storage:  registry,
		auth:     authSvc,
		library:  librarySvc,
		listener: listenerSvc,
		jobs:     jobsSvc,
		playback: playbackSvc,
		hls:      newHLSService(cfg, db, registry),
		progress: progressSvc,
		search:   searchSvc,
		metadata: metadataSvc,
		schedule: scheduleSvc,
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
	mux.HandleFunc("GET /api/v1/home/latest-by-library", router.handleLatestByLibrary)
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
	mux.HandleFunc("POST /api/v1/storage-events", router.handleStorageEvent)
	mux.HandleFunc("GET /api/v1/libraries/{id}/items", router.handleListLibraryItems)
	mux.HandleFunc("GET /api/v1/discovery", router.handleDiscoverMedia)
	mux.HandleFunc("GET /api/v1/search/history", router.handleListSearchHistory)
	mux.HandleFunc("GET /api/v1/media-items/{id}", router.handleGetMediaItem)
	mux.HandleFunc("GET /api/v1/tv/{tmdb_id}/seasons", router.handleListTVSeasons)
	mux.HandleFunc("GET /api/v1/tv/{tmdb_id}/seasons/{n}/episodes", router.handleListTVSeasonEpisodes)
	mux.HandleFunc("GET /api/v1/media-items/{id}/progress", router.handleGetMediaItemProgress)
	mux.HandleFunc("PUT /api/v1/media-items/{id}/metadata", router.handleUpdateMediaItemMetadata)
	mux.HandleFunc("POST /api/v1/media-items/{id}/metadata/apply", router.handleApplyMediaItemMetadata)
	mux.HandleFunc("POST /api/v1/media-items/{id}/metadata/refetch", router.handleQueueMediaItemMetadataRefetch)
	mux.HandleFunc("POST /api/v1/media-items/{id}/metadata/search", router.handleSearchMediaItemMetadata)
	mux.HandleFunc("POST /api/v1/media-items/{id}/match", router.handleQueueMediaItemMatch)
	mux.HandleFunc("GET /api/v1/media-items/{id}/playback", router.handleGetPlaybackSource)
	mux.HandleFunc("GET /api/v1/media-files/{id}/link", router.handleGetMediaFileLink)
	mux.HandleFunc("GET /api/v1/media-files/{id}/hls/index.m3u8", router.handleGetHLSPlaylist)
	mux.HandleFunc("GET /api/v1/media-files/{id}/hls/{name}", router.handleGetHLSArtifact)
	mux.HandleFunc("GET /api/v1/media-files/{id}/stream", router.handleStreamMediaFile)
	mux.HandleFunc("GET /api/v1/jobs", router.handleListJobs)
	mux.HandleFunc("POST /api/v1/jobs/{id}/retry", router.handleRetryJob)
	mux.HandleFunc("GET /api/v1/schedules", router.handleListSchedules)
	mux.HandleFunc("POST /api/v1/schedules", router.handleCreateSchedule)
	mux.HandleFunc("GET /api/v1/schedules/{id}", router.handleGetSchedule)
	mux.HandleFunc("PATCH /api/v1/schedules/{id}", router.handleUpdateSchedule)
	mux.HandleFunc("POST /api/v1/schedules/{id}/toggle", router.handleToggleSchedule)
	mux.HandleFunc("POST /api/v1/schedules/{id}/run", router.handleRunScheduleNow)
	mux.HandleFunc("GET /api/v1/schedules/{id}/history", router.handleListScheduleHistory)

	return corsMiddleware(cfg.CORS, loggingMiddleware(mux))
}
