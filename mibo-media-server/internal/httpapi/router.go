package httpapi

import (
	"net/http"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/health"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/listener"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/progress"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/schedule"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/webui"
	"github.com/atlan/mibo-media-server/internal/workflow"
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
	catalog  *catalog.Service
	library  *library.Service
	listener *listener.Service
	ingest   *ingest.Service
	playback *playback.Service
	progress *progress.Service
	search   *search.Service
	metadata *metadata.Service
	schedule *schedule.Service
	settings *settings.Service
	health   *health.Service
	workflow *workflow.Service
}

type Dependencies struct {
	Config   config.Config
	DB       *gorm.DB
	Registry *providers.Registry

	Auth     *auth.Service
	Catalog  *catalog.Service
	Library  *library.Service
	Listener *listener.Service
	Ingest   *ingest.Service
	Playback *playback.Service
	Progress *progress.Service
	Search   *search.Service
	Metadata *metadata.Service
	Schedule *schedule.Service
	Settings *settings.Service
	Health   *health.Service
	Workflow *workflow.Service
}

func New(deps Dependencies) http.Handler {
	deps = withDefaults(deps)
	router := newRouter(deps)

	mux := http.NewServeMux()
	router.registerRoutes(mux)
	mux.Handle("/", newWebAppHandler(deps.Config.Web, webui.EmbeddedDist()))

	return corsMiddleware(deps.Config.CORS, loggingMiddleware(mux))
}

func withDefaults(deps Dependencies) Dependencies {
	if deps.Schedule == nil {
		deps.Schedule = schedule.NewService(deps.DB)
	}
	if deps.Listener == nil {
		deps.Listener = listener.NewService(deps.DB, nil, deps.Library, deps.Registry)
	}
	if deps.Ingest == nil {
		deps.Ingest = ingest.NewService(deps.DB)
	}
	if deps.Health == nil {
		deps.Health = health.NewService(deps.DB, deps.Registry, deps.Library, deps.Config.OpenList.BaseURL)
	}
	if deps.Workflow == nil {
		deps.Workflow = workflow.NewService(deps.DB)
	}
	return deps
}

func newRouter(deps Dependencies) *Router {
	return &Router{
		cfg:      deps.Config,
		db:       deps.DB,
		storage:  deps.Registry,
		auth:     deps.Auth,
		catalog:  deps.Catalog,
		library:  deps.Library,
		listener: deps.Listener,
		ingest:   deps.Ingest,
		playback: deps.Playback,
		progress: deps.Progress,
		search:   deps.Search,
		metadata: deps.Metadata,
		schedule: deps.Schedule,
		settings: deps.Settings,
		health:   deps.Health,
		workflow: deps.Workflow,
	}
}

func (r *Router) registerRoutes(mux *http.ServeMux) {
	for _, register := range r.routeRegistrations() {
		register(mux)
	}
}

func (r *Router) routeRegistrations() []func(*http.ServeMux) {
	return []func(*http.ServeMux){
		r.registerSystemRoutes,
		r.registerSetupRoutes,
		r.registerAuthRoutes,
		r.registerHomeRoutes,
		r.registerHealthRoutes,
		r.registerAdminRoutes,
		r.registerSettingsRoutes,
		r.registerStorageRoutes,
		r.registerLibraryRoutes,
		r.registerCatalogRoutes,
		r.registerSearchRoutes,
		r.registerWorkflowRoutes,
		r.registerScheduleRoutes,
	}
}
