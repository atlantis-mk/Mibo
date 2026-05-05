import { queryOptions } from "@tanstack/react-query"

import {
  createMiboApi,
  getApiBaseUrl,
  type CatalogUserItemEntry,
} from "#/lib/mibo-api"

export const miboQueryKeys = {
  authUser: (token: string) => ["auth", "me", token] as const,
  loginSessions: (token: string) => ["auth", "sessions", token] as const,
  homeData: (token: string) => ["home", "hero", token] as const,
  healthSummary: (token: string) => ["health", "summary", token] as const,
  healthIssues: (token: string) => ["health", "issues", token] as const,
  favorites: (token: string) => ["me", "favorites", token] as const,
  consoleSummary: (token: string) => ["admin", "console", token] as const,
  ingestDiagnostics: (token: string) =>
    ["admin", "ingest", "diagnostics", token] as const,
  adminLogs: (token: string) => ["admin", "logs", token] as const,
  adminUsers: (token: string) => ["admin", "users", token] as const,
  libraryDetail: (token: string, libraryId: number) =>
    ["library", "detail", token, libraryId] as const,
  libraryBrowse: (
    token: string,
    libraryId: number,
    tab: string,
    filters: unknown,
    page: number,
    pageSize: number
  ) =>
    [
      "library",
      "browse",
      token,
      libraryId,
      tab,
      filters,
      page,
      pageSize,
    ] as const,
  catalogItemDetail: (token: string, itemId: number) =>
    ["catalog", "detail", token, itemId] as const,
  catalogPersonDetail: (token: string, personId: number) =>
    ["catalog", "person-detail", token, personId] as const,
  catalogItemProgress: (token: string, itemId: number) =>
    ["catalog", "progress", token, itemId] as const,
  catalogSeriesSeasons: (token: string, itemId: number) =>
    ["catalog", "series-seasons", token, itemId] as const,
  catalogPlayback: (token: string, itemId: number, assetId?: number) =>
    ["catalog", "playback", token, itemId, assetId ?? "default"] as const,
  inventoryFilePlayback: (token: string, fileId: number) =>
    ["inventory-file", "playback", token, fileId] as const,
  catalogGovernanceWorkspace: (token: string, itemId: number) =>
    ["catalog", "governance", token, itemId] as const,
  metadataWorkspace: (token: string) =>
    ["metadata", "workspace", token] as const,
  metadataProviderInstances: (token: string) =>
    ["settings", "metadata-providers", token] as const,
  metadataProfiles: (token: string) =>
    ["settings", "metadata-profiles", token] as const,
  networkSettings: (token: string) => ["settings", "network", token] as const,
  mediaSources: (token: string) =>
    ["settings", "media-sources", token] as const,
  libraries: (token: string) => ["settings", "libraries", token] as const,
  libraryMetadataStrategy: (token: string, libraryId: number) =>
    ["settings", "library-metadata-strategy", token, libraryId] as const,
  scanExclusions: (token: string, filters: unknown) =>
    ["settings", "scan-exclusions", token, filters] as const,
  scanExclusionRules: (token: string) =>
    ["settings", "scan-exclusion-rules", token] as const,
  schedules: (token: string) => ["schedules", "workspace", token] as const,
  scheduleDetail: (token: string, scheduleId: number) =>
    ["schedules", "detail", token, scheduleId] as const,
  scheduleHistory: (token: string, scheduleId: number) =>
    ["schedules", "history", token, scheduleId] as const,
  workflows: (token: string, filters: unknown) =>
    ["admin", "workflows", token, filters] as const,
  workflowDiagnostics: (token: string) =>
    ["admin", "workflows", "diagnostics", token] as const,
  cleanupSettings: (token: string) => ["settings", "cleanup", token] as const,
}

export function createAuthedMiboApi(token: string) {
  return createMiboApi({
    baseUrl: getApiBaseUrl(),
    token,
  })
}

export function authUserQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.authUser(token),
    queryFn: () => createAuthedMiboApi(token).me(),
  })
}

export function loginSessionsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.loginSessions(token),
    queryFn: () => createAuthedMiboApi(token).listLoginSessions(),
  })
}

export function homeDataQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.homeData(token),
    refetchInterval: (query) => {
      const libraries = query.state.data?.libraries ?? []
      const hasActiveIngest = libraries.some((library) =>
        ["pending", "syncing"].includes(library.status)
      )
      return hasActiveIngest ? 5000 : false
    },
    queryFn: async () => {
      const api = createAuthedMiboApi(token)
      const [
        items,
        continueWatching,
        libraries,
        latestByLibrary,
        healthIssues,
      ] = await Promise.all([
        api.recentlyAdded(6),
        api.continueWatching(),
        api.listLibraries(),
        api.latestByLibrary(),
        api.listHealthIssues().catch(() => []),
      ])

      const safeItems = items ?? []
      const safeContinueWatching = getLatestContinueWatchingEntries(
        continueWatching ?? []
      )
      const safeLibraries = libraries ?? []
      const safeLatestByLibrary = latestByLibrary ?? []
      const safeHealthIssues = healthIssues ?? []

      return {
        items: safeItems,
        continueWatching: safeContinueWatching,
        continueWatchingCount: safeContinueWatching.length,
        libraries: safeLibraries,
        libraryCount: safeLibraries.length,
        latestByLibrary: safeLatestByLibrary,
        healthIssues: safeHealthIssues,
      }
    },
  })
}

export function getLatestContinueWatchingEntries(
  entries: CatalogUserItemEntry[]
) {
  const latestByDisplayItem = new Map<string, CatalogUserItemEntry>()

  for (const entry of entries) {
    const key = getContinueWatchingDisplayKey(entry)
    const existing = latestByDisplayItem.get(key)

    if (!existing || isNewerContinueWatchingEntry(entry, existing)) {
      latestByDisplayItem.set(key, entry)
    }
  }

  return Array.from(latestByDisplayItem.values()).sort(
    (left, right) => getPlayedAtTime(right) - getPlayedAtTime(left)
  )
}

function getContinueWatchingDisplayKey(entry: CatalogUserItemEntry) {
  const displayItem = entry.display_item ?? entry.item
  return `${displayItem.library_id}:${displayItem.type}:${displayItem.id}`
}

function isNewerContinueWatchingEntry(
  candidate: CatalogUserItemEntry,
  current: CatalogUserItemEntry
) {
  return getPlayedAtTime(candidate) > getPlayedAtTime(current)
}

function getPlayedAtTime(entry: CatalogUserItemEntry) {
  return entry.last_played_at ? Date.parse(entry.last_played_at) || 0 : 0
}

export function healthSummaryQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.healthSummary(token),
    queryFn: () => createAuthedMiboApi(token).getHealthSummary(),
  })
}

export function healthIssuesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.healthIssues(token),
    queryFn: () => createAuthedMiboApi(token).listHealthIssues(),
  })
}

export function favoritesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.favorites(token),
    queryFn: () => createAuthedMiboApi(token).listFavorites(),
  })
}

export function consoleSummaryQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.consoleSummary(token),
    queryFn: () => createAuthedMiboApi(token).getConsoleSummary(),
  })
}

export function ingestDiagnosticsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.ingestDiagnostics(token),
    queryFn: () => createAuthedMiboApi(token).getIngestDiagnostics(),
  })
}

export function adminLogsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.adminLogs(token),
    queryFn: () => createAuthedMiboApi(token).listAdminLogs(),
  })
}

export function adminUsersQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.adminUsers(token),
    queryFn: () => createAuthedMiboApi(token).listAdminUsers(),
  })
}

export function catalogItemDetailQueryOptions(token: string, itemId: number) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
    queryFn: () => createAuthedMiboApi(token).getCatalogItem(itemId),
  })
}

export function catalogPersonDetailQueryOptions(
  token: string,
  personId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogPersonDetail(token, personId),
    queryFn: () => createAuthedMiboApi(token).getCatalogPerson(personId),
    enabled: personId > 0,
  })
}

export function catalogItemProgressQueryOptions(token: string, itemId: number) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogItemProgress(token, itemId),
    queryFn: async () => {
      try {
        const progress =
          await createAuthedMiboApi(token).getCatalogItemProgress(itemId)

        return progress.position_seconds > 0 || progress.watched
          ? progress
          : null
      } catch {
        return null
      }
    },
  })
}

export function catalogSeriesSeasonsQueryOptions(
  token: string,
  itemId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogSeriesSeasons(token, itemId),
    queryFn: () => createAuthedMiboApi(token).listCatalogSeriesSeasons(itemId),
    enabled: itemId > 0,
  })
}

export function catalogPlaybackQueryOptions(
  token: string,
  itemId: number,
  assetId?: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogPlayback(token, itemId, assetId),
    queryFn: () =>
      createAuthedMiboApi(token).getCatalogPlayback(itemId, {
        clientProfile: "web",
        assetId,
      }),
    enabled: itemId > 0,
  })
}

export function inventoryFilePlaybackQueryOptions(
  token: string,
  fileId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.inventoryFilePlayback(token, fileId),
    queryFn: () =>
      createAuthedMiboApi(token).getInventoryFilePlayback(fileId, {
        clientProfile: "web",
      }),
    enabled: fileId > 0,
  })
}

export function catalogGovernanceWorkspaceQueryOptions(
  token: string,
  itemId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogGovernanceWorkspace(token, itemId),
    queryFn: () =>
      createAuthedMiboApi(token).getCatalogGovernanceWorkspace(itemId),
    enabled: itemId > 0,
  })
}

export function mediaSourcesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.mediaSources(token),
    queryFn: () => createAuthedMiboApi(token).listMediaSources(),
  })
}

export function networkSettingsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.networkSettings(token),
    queryFn: () => createAuthedMiboApi(token).getNetworkSettings(),
  })
}

export function metadataProviderInstancesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.metadataProviderInstances(token),
    queryFn: () => createAuthedMiboApi(token).listMetadataProviderInstances(),
  })
}

export function metadataProfilesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.metadataProfiles(token),
    queryFn: () => createAuthedMiboApi(token).listMetadataProfiles(),
  })
}

export function librariesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.libraries(token),
    queryFn: () => createAuthedMiboApi(token).listLibraries(),
  })
}

export function libraryMetadataStrategyQueryOptions(
  token: string,
  libraryId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.libraryMetadataStrategy(token, libraryId),
    queryFn: () =>
      createAuthedMiboApi(token).getLibraryMetadataStrategy(libraryId),
    enabled: libraryId > 0,
  })
}

export function scanExclusionsQueryOptions(
  token: string,
  filters: { libraryId?: number; enabled?: boolean }
) {
  return queryOptions({
    queryKey: miboQueryKeys.scanExclusions(token, filters),
    queryFn: () => createAuthedMiboApi(token).listScanExclusions(filters),
  })
}

export function scanExclusionRulesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.scanExclusionRules(token),
    queryFn: () => createAuthedMiboApi(token).listScanExclusionRules(),
  })
}

export function schedulesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.schedules(token),
    queryFn: () => createAuthedMiboApi(token).listSchedules(),
  })
}

export function scheduleDetailQueryOptions(token: string, scheduleId: number) {
  return queryOptions({
    queryKey: miboQueryKeys.scheduleDetail(token, scheduleId),
    queryFn: () => createAuthedMiboApi(token).getSchedule(scheduleId),
    enabled: scheduleId > 0,
  })
}

export function scheduleHistoryQueryOptions(token: string, scheduleId: number) {
  return queryOptions({
    queryKey: miboQueryKeys.scheduleHistory(token, scheduleId),
    queryFn: () => createAuthedMiboApi(token).listScheduleHistory(scheduleId),
    enabled: scheduleId > 0,
  })
}

export function workflowsQueryOptions(
  token: string,
  filters: { limit?: number; offset?: number; status?: string }
) {
  return queryOptions({
    queryKey: miboQueryKeys.workflows(token, filters),
    queryFn: () => createAuthedMiboApi(token).listWorkflows(filters),
  })
}

export function workflowDiagnosticsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.workflowDiagnostics(token),
    queryFn: () => createAuthedMiboApi(token).getWorkflowDiagnostics(),
  })
}

export function cleanupSettingsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.cleanupSettings(token),
    queryFn: () => createAuthedMiboApi(token).getCleanupSettings(),
  })
}
