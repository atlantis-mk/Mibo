import {
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from "@tanstack/react-router"

import { AppSidebar } from "#/components/app-sidebar"
import { LoginForm } from "#/components/login-form"
import { SidebarProvider } from "#/components/ui/sidebar"
import FavoritesPage from "#/features/favorites"
import Home from "#/features/home"
import JobsPage from "#/features/jobs"
import LibraryDetail, {
  DEFAULT_LIBRARY_PAGE_SIZE,
  isLibraryPageSize,
} from "#/features/library"
import {
  createDefaultDiscoveryFilters,
  type DiscoveryFilters,
} from "#/features/discovery/controls"
import MediaDetail from "#/features/media"
import { parseMediaDetailView } from "#/lib/media-presentation"
import LogsPage from "#/features/logs"
import MetadataGovernancePage from "#/features/metadata-governance"
import PersonDetailPage from "#/features/person"
import PlayExperience from "#/features/play"
import SchedulesPage from "#/features/schedules"
import SearchPage from "#/features/search"
import SetupPage from "#/features/setup"
import SettingsLayout from "#/features/settings"
import {
  SettingsCleanupPage,
  SettingsConsolePage,
  SettingsDatabasePage,
  SettingsDevicesPage,
  SettingsDlnaPage,
  SettingsGeneralPage,
  SettingsHealthPage,
  SettingsLibraryPage,
  SettingsLiveTvPage,
  SettingsMetadataSourcesPage,
  SettingsNetworkPage,
  SettingsNotificationsPage,
  SettingsPlaybackPage,
  SettingsScanExclusionsPage,
  SettingsSecurityPage,
  SettingsUsersPage,
} from "#/features/settings/pages"
import {
  normalizeInternalRedirect,
  requireCanEnterApp,
  requireSetupAccess,
} from "#/lib/setup-gate"
import { useAuthStore } from "#/stores/auth-store"

const rootRoute = createRootRoute({
  component: RootLayout,
})

const appLayoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "app-layout",
  beforeLoad: async ({ location }) => {
    await requireCanEnterApp()
    await requireAuthenticated(location.href)
  },
  component: AppLayout,
})

const indexRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/",
  component: Home,
})

const libraryRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/library/$id",
  validateSearch: (search: Record<string, unknown>) => ({
    page: normalizeLibraryPageSearch(search.page),
    pageSize: normalizeLibraryPageSizeSearch(search.pageSize),
    ...libraryFiltersToSearch(parseLibraryFiltersSearch(search)),
  }),
  component: LibraryRoute,
})

const mediaRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/media/$id",
  validateSearch: (search: Record<string, unknown>) => ({
    view: search.view === "series" ? "series" : undefined,
    episodePage: normalizeEpisodePageSearch(search.episodePage),
  }),
  component: MediaRoute,
})

const personRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/person/$id",
  component: PersonRoute,
})

const favoritesRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/favorites",
  component: FavoritesPage,
})

const searchRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/search",
  validateSearch: (search: Record<string, unknown>) => ({
    q: typeof search.q === "string" ? search.q : undefined,
  }),
  component: SearchRoute,
})

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/settings",
  beforeLoad: async ({ location }) => {
    await requireCanEnterApp()
    await requireAuthenticated(location.href)
  },
  component: SettingsLayout,
})

const playRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/play/$id",
  beforeLoad: async ({ location }) => {
    await requireCanEnterApp()
    await requireAuthenticated(location.href)
  },
  validateSearch: (search: Record<string, unknown>) => ({
    fromStart:
      search.fromStart === true ||
      search.fromStart === "true" ||
      search.fromStart === "1",
    assetId:
      typeof search.assetId === "number"
        ? search.assetId
        : typeof search.assetId === "string"
          ? Number.parseInt(search.assetId, 10) || undefined
          : undefined,
    inventoryFileId:
      typeof search.inventoryFileId === "number"
        ? search.inventoryFileId
        : typeof search.inventoryFileId === "string"
          ? Number.parseInt(search.inventoryFileId, 10) || undefined
          : undefined,
  }),
  component: PlayRoute,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  validateSearch: (search: Record<string, unknown>) => ({
    redirect:
      typeof search.redirect === "string" &&
      search.redirect.startsWith("/") &&
      search.redirect !== "/login"
        ? search.redirect
        : undefined,
  }),
  beforeLoad: async () => {
    await requireCanEnterApp()
  },
  component: LoginRoute,
})

const setupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/setup",
  validateSearch: (search: Record<string, unknown>) => ({
    redirect:
      typeof search.redirect === "string" && search.redirect.startsWith("/")
        ? search.redirect
        : undefined,
  }),
  beforeLoad: async ({ search }) => {
    await requireSetupAccess(search.redirect)
  },
  component: SetupRoute,
})

const settingsIndexRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/",
  beforeLoad: () => {
    throw redirect({ to: "/settings/console" })
  },
})

const settingsGeneralRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/general",
  component: SettingsGeneralPage,
})

const settingsHealthRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/health",
  component: SettingsHealthPage,
})

const settingsLibraryRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/library",
  component: SettingsLibraryPage,
})

const settingsPlaybackRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/playback",
  component: SettingsPlaybackPage,
})

const settingsMetadataSourcesRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/metadata-sources",
  component: SettingsMetadataSourcesPage,
})

const settingsSchedulesRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/schedules",
  component: SchedulesPage,
})

const settingsJobsRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/jobs",
  component: JobsPage,
})

const settingsCleanupRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/cleanup",
  component: SettingsCleanupPage,
})

const settingsUsersRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/users",
  component: SettingsUsersPage,
})

const settingsDevicesRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/devices",
  component: SettingsDevicesPage,
})

const settingsNetworkRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/network",
  component: SettingsNetworkPage,
})

const settingsDlnaRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/dlna",
  component: SettingsDlnaPage,
})

const settingsLiveTvRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/live-tv",
  component: SettingsLiveTvPage,
})

const settingsNotificationsRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/notifications",
  component: SettingsNotificationsPage,
})

const settingsSecurityRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/security",
  component: SettingsSecurityPage,
})

const settingsDatabaseRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/database",
  component: SettingsDatabasePage,
})

const settingsLogsRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/logs",
  component: LogsPage,
})

const settingsConsoleRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/console",
  component: SettingsConsolePage,
})

const settingsScanExclusionsRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/scan-exclusions",
  component: SettingsScanExclusionsPage,
})

const settingsMetadataIndexRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/metadata/",
  component: MetadataGovernancePage,
})

const settingsMetadataDetailRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/metadata/$id",
  component: SettingsMetadataDetailRoute,
})

const routeTree = rootRoute.addChildren([
  appLayoutRoute.addChildren([
    indexRoute,
    libraryRoute,
    mediaRoute,
    personRoute,
    favoritesRoute,
    searchRoute,
  ]),
  settingsRoute.addChildren([
    settingsIndexRoute,
    settingsGeneralRoute,
    settingsHealthRoute,
    settingsLibraryRoute,
    settingsPlaybackRoute,
    settingsMetadataSourcesRoute,
    settingsMetadataIndexRoute,
    settingsMetadataDetailRoute,
    settingsSchedulesRoute,
    settingsJobsRoute,
    settingsCleanupRoute,
    settingsUsersRoute,
    settingsDevicesRoute,
    settingsNetworkRoute,
    settingsDlnaRoute,
    settingsLiveTvRoute,
    settingsNotificationsRoute,
    settingsSecurityRoute,
    settingsDatabaseRoute,
    settingsLogsRoute,
    settingsConsoleRoute,
    settingsScanExclusionsRoute,
  ]),
  playRoute,
  loginRoute,
  setupRoute,
])

export const router = createRouter({
  routeTree,
  scrollRestoration: true,
  defaultPreload: "intent",
  defaultPreloadStaleTime: 0,
})

function RootLayout() {
  return <Outlet />
}

async function requireAuthenticated(redirectTo: string) {
  await waitForAuthHydration()

  if (useAuthStore.getState().token) {
    return
  }

  throw redirect({
    to: "/login",
    search: { redirect: normalizeInternalRedirect(redirectTo, "/") },
  })
}

function waitForAuthHydration() {
  if (useAuthStore.getState().hasHydrated) {
    return Promise.resolve()
  }

  return new Promise<void>((resolve) => {
    const unsubscribe = useAuthStore.subscribe((state) => {
      if (!state.hasHydrated) {
        return
      }

      unsubscribe()
      resolve()
    })
  })
}

function AppLayout() {
  return (
    <SidebarProvider defaultOpen={false}>
      <AppSidebar variant="floating" className="z-40" />
      <div className="relative flex min-w-0 flex-1">
        <Outlet />
      </div>
    </SidebarProvider>
  )
}

function LibraryRoute() {
  const { id } = libraryRoute.useParams()
  const search = libraryRoute.useSearch()
  const navigate = libraryRoute.useNavigate()
  const page = search.page ?? 1
  const pageSize = search.pageSize ?? DEFAULT_LIBRARY_PAGE_SIZE
  const filters = parseLibraryFiltersSearch(search)

  return (
    <LibraryDetail
      libraryId={Number(id)}
      page={page}
      pageSize={pageSize}
      filters={filters}
      onPaginationChange={(next) => {
        void navigate({
          search: (previous) => {
            const nextPage = next.page ?? previous.page ?? 1
            const nextPageSize =
              next.pageSize ?? previous.pageSize ?? DEFAULT_LIBRARY_PAGE_SIZE

            return {
              ...previous,
              page: nextPage === 1 ? undefined : nextPage,
              pageSize:
                nextPageSize === DEFAULT_LIBRARY_PAGE_SIZE
                  ? undefined
                  : nextPageSize,
            }
          },
          replace: true,
        })
      }}
      onFiltersChange={(filters) => {
        void navigate({
          search: (previous) => {
            return {
              ...previous,
              ...libraryFiltersToSearch(filters),
              page: undefined,
            }
          },
          replace: true,
        })
      }}
    />
  )
}

function parsePositiveIntSearch(value: unknown) {
  const parsed =
    typeof value === "number"
      ? value
      : typeof value === "string"
        ? Number.parseInt(value, 10)
        : undefined

  return parsed && Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

function normalizeLibraryPageSearch(value: unknown) {
  const parsed = parsePositiveIntSearch(value)

  return parsed && parsed !== 1 ? parsed : undefined
}

function normalizeLibraryPageSizeSearch(value: unknown) {
  const parsed = parsePositiveIntSearch(value)

  return parsed &&
    isLibraryPageSize(parsed) &&
    parsed !== DEFAULT_LIBRARY_PAGE_SIZE
    ? parsed
    : undefined
}

function parseLibraryFiltersSearch(
  search: Record<string, unknown>
): DiscoveryFilters {
  return createDefaultDiscoveryFilters({
    q: parseStringSearch(search.q),
    type: parseLibraryTypeSearch(search.type),
    genre: parseStringSearch(search.genre),
    region: parseStringSearch(search.region),
    year: parseStringSearch(search.year),
    minRating: parseStringSearch(search.minRating),
    watchedState: parseWatchedStateSearch(search.watchedState),
    organizingState: parseOrganizingStateSearch(search.organizingState),
    sort: parseLibrarySortSearch(search.sort) ?? "title",
    sortDirection: parseSortDirectionSearch(search.sortDirection) ?? "asc",
  })
}

function libraryFiltersToSearch(filters: DiscoveryFilters) {
  return {
    q: filters.q.trim() || undefined,
    type: filters.type === "all" ? undefined : filters.type,
    genre: filters.genre.trim() || undefined,
    region: filters.region.trim() || undefined,
    year: filters.year.trim() || undefined,
    minRating: filters.minRating.trim() || undefined,
    watchedState:
      filters.watchedState === "all" ? undefined : filters.watchedState,
    organizingState:
      filters.organizingState === "all" ? undefined : filters.organizingState,
    sort: filters.sort === "title" ? undefined : filters.sort,
    sortDirection:
      filters.sortDirection === "asc" ? undefined : filters.sortDirection,
  }
}

function parseStringSearch(value: unknown) {
  return typeof value === "string" ? value : undefined
}

function parseLibraryTypeSearch(
  value: unknown
): DiscoveryFilters["type"] | undefined {
  return value === "movie" || value === "show" || value === "all"
    ? value
    : undefined
}

function parseWatchedStateSearch(
  value: unknown
): DiscoveryFilters["watchedState"] | undefined {
  return value === "unwatched" ||
    value === "in_progress" ||
    value === "watched" ||
    value === "all"
    ? value
    : undefined
}

function parseOrganizingStateSearch(
  value: unknown
): DiscoveryFilters["organizingState"] | undefined {
  return value === "organized" || value === "unorganized" || value === "all"
    ? value
    : undefined
}

function parseLibrarySortSearch(
  value: unknown
): DiscoveryFilters["sort"] | undefined {
  return value === "recent" ||
    value === "title" ||
    value === "year" ||
    value === "watch_status"
    ? value
    : undefined
}

function parseSortDirectionSearch(
  value: unknown
): DiscoveryFilters["sortDirection"] | undefined {
  return value === "asc" || value === "desc" ? value : undefined
}

function MediaRoute() {
  const { id } = mediaRoute.useParams()
  const { view, episodePage } = mediaRoute.useSearch()

  return (
    <MediaDetail
      itemId={Number(id)}
      detailView={parseMediaDetailView(view)}
      episodePage={episodePage}
    />
  )
}

function normalizeEpisodePageSearch(value: unknown): number {
  const page = typeof value === "number" ? value : Number(value)
  return Number.isInteger(page) && page > 0 ? page : 1
}

function PersonRoute() {
  const { id } = personRoute.useParams()

  return <PersonDetailPage personId={Number(id)} />
}

function SearchRoute() {
  const search = searchRoute.useSearch()

  return <SearchPage initialQuery={search.q} />
}

function PlayRoute() {
  const { id } = playRoute.useParams()
  const { fromStart, assetId, inventoryFileId } = playRoute.useSearch()

  return (
    <PlayExperience
      itemId={Number(id)}
      assetId={assetId}
      inventoryFileId={inventoryFileId}
      fromStart={fromStart}
    />
  )
}

function LoginRoute() {
  const { redirect } = loginRoute.useSearch()

  return (
    <div className="flex min-h-svh flex-col items-center justify-center bg-muted p-6 md:p-10">
      <div className="w-full max-w-sm md:max-w-4xl">
        <LoginForm redirectTo={redirect ?? "/"} />
      </div>
    </div>
  )
}

function SetupRoute() {
  const { redirect } = setupRoute.useSearch()

  return <SetupPage redirectTo={normalizeInternalRedirect(redirect, "/")} />
}

function SettingsMetadataDetailRoute() {
  const { id } = settingsMetadataDetailRoute.useParams()

  return <MetadataGovernancePage itemId={Number(id)} />
}

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}
