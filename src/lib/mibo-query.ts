import { queryOptions } from '@tanstack/react-query'
import {
  createMiboApi,
  getApiBaseUrl,
  type CatalogUserItemEntry,
} from '@/lib/mibo-api'

export const miboQueryKeys = {
  authUser: (token: string) => ['auth', 'me', token] as const,
  loginUsers: () => ['auth', 'login-users'] as const,
  userSettings: (token: string) => ['me', 'settings', token] as const,
  loginSessions: (token: string) => ['auth', 'sessions', token] as const,
  homeData: (token: string) => ['home', 'hero', token] as const,
  operationsOverview: (token: string) =>
    ['operations', 'overview', token] as const,
  operationsTasks: (token: string) => ['operations', 'tasks', token] as const,
  operationsIssues: (token: string) => ['operations', 'issues', token] as const,
  operationsIssueDetail: (token: string, issueId: number) =>
    ['operations', 'issues', 'detail', token, issueId] as const,
  operationsIssueEvents: (token: string, issueId: number) =>
    ['operations', 'issues', 'events', token, issueId] as const,
  operationsPipeline: (token: string) =>
    ['operations', 'pipeline', token] as const,
  favorites: (token: string) => ['me', 'favorites', token] as const,
  consoleSummary: (token: string) => ['admin', 'console', token] as const,
  ingestDiagnostics: (token: string) =>
    ['admin', 'ingest', 'diagnostics', token] as const,
  adminLogs: (token: string) => ['admin', 'logs', token] as const,
  adminLogSettings: (token: string) =>
    ['admin', 'logs', 'settings', token] as const,
  adminUsers: (token: string) => ['admin', 'users', token] as const,
  libraryDetail: (token: string, libraryId: number) =>
    ['library', 'detail', token, libraryId] as const,
  libraryBrowse: (
    token: string,
    scopeKey: string,
    tab: string,
    filters: unknown,
    page: number,
    pageSize: number
  ) =>
    [
      'library',
      'browse',
      token,
      scopeKey,
      tab,
      filters,
      page,
      pageSize,
    ] as const,
  libraryHierarchy: (
    token: string,
    libraryId: number | 'root',
    path: string,
    filters: unknown,
    page: number,
    pageSize: number
  ) =>
    [
      'library',
      'hierarchy',
      token,
      libraryId,
      path,
      filters,
      page,
      pageSize,
    ] as const,
  catalogItemDetail: (token: string, itemId: number) =>
    ['catalog', 'detail', token, itemId] as const,
  metadataItemResources: (token: string, itemId: number) =>
    ['metadata', 'resources', token, itemId] as const,
  catalogPersonDetail: (token: string, personId: number) =>
    ['catalog', 'person-detail', token, personId] as const,
  catalogItemProgress: (token: string, itemId: number) =>
    ['catalog', 'progress', token, itemId] as const,
  catalogPlayback: (
    token: string,
    itemId: number,
    options?: {
      resourceId?: number
      variant?: string
      startSeconds?: number
      audioStreamIndex?: number
    }
  ) =>
    [
      'catalog',
      'playback',
      token,
      itemId,
      options?.resourceId ?? 'default-resource',
      options?.variant ?? 'original',
      options?.startSeconds ?? 0,
      options?.audioStreamIndex ?? 'default-audio',
    ] as const,
  inventoryFilePlayback: (
    token: string,
    fileId: number,
    options?: {
      variant?: string
      startSeconds?: number
      audioStreamIndex?: number
    }
  ) =>
    [
      'inventory-file',
      'playback',
      token,
      fileId,
      options?.variant ?? 'original',
      options?.startSeconds ?? 0,
      options?.audioStreamIndex ?? 'default-audio',
    ] as const,
  catalogGovernanceWorkspace: (token: string, itemId: number) =>
    ['catalog', 'governance', token, itemId] as const,
  metadataWorkspace: (token: string) =>
    ['metadata', 'workspace', token] as const,
  metadataProviderInstances: (token: string) =>
    ['settings', 'metadata-providers', token] as const,
  generalConfig: (token: string) =>
    ['settings', 'general-config', token] as const,
  pluginProviderInstances: (token: string) =>
    ['settings', 'plugin-providers', token] as const,
  pluginProviderDetail: (token: string, providerId: number) =>
    ['settings', 'plugin-providers', token, providerId, 'detail'] as const,
  internalPlugins: (token: string) =>
    ['settings', 'plugin-internal', token] as const,
  openSubtitlesSettings: (token: string) =>
    ['settings', 'plugin-internal', 'opensubtitles', token] as const,
  subtitleProviderInstances: (token: string) =>
    ['settings', 'subtitles', 'providers', token] as const,
  localPluginInstallations: (token: string) =>
    ['settings', 'plugin-local-installations', token] as const,
  pluginCatalogOverview: (token: string) =>
    ['settings', 'plugin-catalog', token] as const,
  metadataProfiles: (token: string) =>
    ['settings', 'metadata-profiles', token] as const,
  networkSettings: (token: string) => ['settings', 'network', token] as const,
  liveTVSources: (token: string) =>
    ['settings', 'live-tv-sources', token] as const,
  liveTVChannelGroups: (token: string, filters: unknown) =>
    ['settings', 'live-tv-channel-groups', token, filters] as const,
  liveTVPrograms: (token: string, filters: unknown) =>
    ['live-tv', 'programs', token, filters] as const,
  liveTVChannels: (token: string, filters: unknown) =>
    ['settings', 'live-tv-channels', token, filters] as const,
  liveTVPlayback: (token: string, channelId: number) =>
    ['live-tv', 'playback', token, channelId] as const,
  mediaSources: (token: string) =>
    ['settings', 'media-sources', token] as const,
  libraryAccessTags: (token: string) =>
    ['settings', 'library-access-tags', token] as const,
  libraries: (token: string) => ['settings', 'libraries', token] as const,
  libraryMetadataStrategy: (token: string, libraryId: number) =>
    ['settings', 'library-metadata-strategy', token, libraryId] as const,
  scanExclusions: (token: string, filters: unknown) =>
    ['settings', 'scan-exclusions', token, filters] as const,
  scanExclusionRules: (token: string) =>
    ['settings', 'scan-exclusion-rules', token] as const,
  schedules: (token: string) => ['schedules', 'workspace', token] as const,
  scheduleDetail: (token: string, scheduleId: number) =>
    ['schedules', 'detail', token, scheduleId] as const,
  scheduleHistory: (token: string, scheduleId: number) =>
    ['schedules', 'history', token, scheduleId] as const,
  workflows: (token: string, filters: unknown) =>
    ['admin', 'workflows', token, filters] as const,
  workflowDiagnostics: (token: string) =>
    ['admin', 'workflows', 'diagnostics', token] as const,
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

export function loginUsersQueryOptions() {
  return queryOptions({
    queryKey: miboQueryKeys.loginUsers(),
    queryFn: () => createMiboApi({ baseUrl: getApiBaseUrl() }).listLoginUsers(),
  })
}

export function userSettingsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.userSettings(token),
    queryFn: () => createAuthedMiboApi(token).getUserSettings(),
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
    queryFn: async () => {
      const api = createAuthedMiboApi(token)
      const [
        items,
        continueWatching,
        contentSections,
        mediaOverview,
        operationsTaskList,
      ] = await Promise.all([
        api.recentlyAdded(6),
        api.continueWatching(),
        api.homeSections(),
        api.homeMediaOverview(),
        api.listOperationsTasks().catch(() => []),
      ])

      const safeItems = items ?? []
      const safeContinueWatching = getLatestContinueWatchingEntries(
        continueWatching ?? []
      )
      const safeContentSections = contentSections ?? []
      const safeMediaOverview = mediaOverview ?? { sections: [] }
      const safeOperationsTasks = Array.isArray(operationsTaskList)
        ? operationsTaskList
        : (operationsTaskList?.items ?? [])

      return {
        items: safeItems,
        continueWatching: safeContinueWatching,
        continueWatchingCount: safeContinueWatching.length,
        contentSections: safeContentSections,
        mediaOverview: safeMediaOverview,
        operationsTasks: safeOperationsTasks,
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

export function operationsOverviewQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.operationsOverview(token),
    queryFn: () => createAuthedMiboApi(token).getOperationsOverview(),
  })
}

export function operationsTasksQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.operationsTasks(token),
    queryFn: async () =>
      (await createAuthedMiboApi(token).listOperationsTasks()).items,
  })
}

export function operationsTaskListQueryOptions(
  token: string,
  filters: {
    page: number
    page_size: number
    lifecycle_status?: 'active' | 'resolved' | 'all'
    kind?: string
    action_type?: string
    library_id?: number
    q?: string
  }
) {
  return queryOptions({
    queryKey: ['operations', 'tasks', token, filters] as const,
    queryFn: () => createAuthedMiboApi(token).listOperationsTasks(filters),
  })
}

export function operationsIssuesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.operationsIssues(token),
    queryFn: async () =>
      (await createAuthedMiboApi(token).listOperationsIssues()).items,
  })
}

export function operationsIssueListQueryOptions(
  token: string,
  filters: {
    page: number
    page_size: number
    status?:
      | 'active'
      | 'in_progress'
      | 'resolved'
      | 'reopened'
      | 'ignored'
      | 'all'
    kind?:
      | 'metadata'
      | 'classification'
      | 'probe'
      | 'workflow'
      | 'storage'
      | 'projection'
      | 'all'
    action_type?:
      | 'retry'
      | 'apply_candidate'
      | 'mark_governed'
      | 'accept_classification'
      | 'correct_classification'
      | 'relink_resource'
      | 'unlink_resource'
      | 'exclude'
      | 'ignore'
      | 'all'
    library_id?: number
    q?: string
  }
) {
  return queryOptions({
    queryKey: ['operations', 'issues', token, filters] as const,
    queryFn: () => createAuthedMiboApi(token).listOperationsIssues(filters),
  })
}

export function operationsIssueDetailQueryOptions(
  token: string,
  issueId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.operationsIssueDetail(token, issueId),
    queryFn: () => createAuthedMiboApi(token).getOperationsIssue(issueId),
  })
}

export function operationsIssueEventsQueryOptions(
  token: string,
  issueId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.operationsIssueEvents(token, issueId),
    queryFn: () =>
      createAuthedMiboApi(token).listOperationsIssueEvents(issueId),
  })
}

export function operationsPipelineQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.operationsPipeline(token),
    queryFn: () => createAuthedMiboApi(token).getOperationsPipeline(),
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

export function adminLogSettingsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.adminLogSettings(token),
    queryFn: () => createAuthedMiboApi(token).getAdminLogSettings(),
  })
}

export function adminUsersQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.adminUsers(token),
    queryFn: () => createAuthedMiboApi(token).listAdminUsers(),
  })
}

export function metadataItemDetailQueryOptions(token: string, itemId: number) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
    queryFn: () => createAuthedMiboApi(token).getMetadataItem(itemId),
  })
}

export function metadataItemResourcesQueryOptions(
  token: string,
  itemId: number,
  libraryId?: number
) {
  return queryOptions({
    queryKey: [
      ...miboQueryKeys.metadataItemResources(token, itemId),
      libraryId ?? 'all',
    ],
    queryFn: () =>
      createAuthedMiboApi(token).listMetadataItemResources(itemId, {
        libraryId,
      }),
    enabled: itemId > 0,
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

export function metadataItemProgressQueryOptions(
  token: string,
  itemId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogItemProgress(token, itemId),
    queryFn: async () => {
      try {
        const progress =
          await createAuthedMiboApi(token).getMetadataItemProgress(itemId)

        return progress.position_seconds > 0 ||
          progress.watched ||
          typeof progress.preferred_resource_id === 'number'
          ? progress
          : null
      } catch {
        return null
      }
    },
  })
}

export function catalogPlaybackQueryOptions(
  token: string,
  itemId: number,
  options?: {
    resourceId?: number
    variant?: string
    startSeconds?: number
    audioStreamIndex?: number
  }
) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogPlayback(token, itemId, options),
    queryFn: () =>
      createAuthedMiboApi(token).getCatalogPlayback(itemId, {
        clientProfile: 'web',
        resourceId: options?.resourceId,
        variant: options?.variant,
        startSeconds: options?.startSeconds,
        audioStreamIndex: options?.audioStreamIndex,
      }),
    enabled: itemId > 0,
  })
}

export function inventoryFilePlaybackQueryOptions(
  token: string,
  fileId: number,
  options?: {
    variant?: string
    startSeconds?: number
    audioStreamIndex?: number
  }
) {
  return queryOptions({
    queryKey: miboQueryKeys.inventoryFilePlayback(token, fileId, options),
    queryFn: () =>
      createAuthedMiboApi(token).getInventoryFilePlayback(fileId, {
        clientProfile: 'web',
        variant: options?.variant,
        startSeconds: options?.startSeconds,
        audioStreamIndex: options?.audioStreamIndex,
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

export function generalConfigQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.generalConfig(token),
    queryFn: () => createAuthedMiboApi(token).getGeneralConfig(),
  })
}

export function liveTVSourcesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.liveTVSources(token),
    queryFn: () => createAuthedMiboApi(token).listLiveTVSources(),
  })
}

export function liveTVChannelsQueryOptions(
  token: string,
  filters: { source_id?: number; group?: string; q?: string; enabled?: boolean }
) {
  return queryOptions({
    queryKey: miboQueryKeys.liveTVChannels(token, filters),
    queryFn: () => createAuthedMiboApi(token).listLiveTVChannels(filters),
  })
}

export function liveTVChannelGroupsQueryOptions(
  token: string,
  filters: { source_id?: number; q?: string; enabled?: boolean }
) {
  return queryOptions({
    queryKey: miboQueryKeys.liveTVChannelGroups(token, filters),
    queryFn: () => createAuthedMiboApi(token).listLiveTVChannelGroups(filters),
  })
}

export function liveTVProgramsQueryOptions(
  token: string,
  filters: {
    source_id?: number
    q?: string
    current?: boolean
    limit?: number
    offset?: number
  }
) {
  return queryOptions({
    queryKey: miboQueryKeys.liveTVPrograms(token, filters),
    queryFn: () => createAuthedMiboApi(token).listLiveTVPrograms(filters),
  })
}

export function liveTVPlaybackQueryOptions(token: string, channelId: number) {
  return queryOptions({
    queryKey: miboQueryKeys.liveTVPlayback(token, channelId),
    queryFn: () => createAuthedMiboApi(token).getLiveTVPlayback(channelId),
    enabled: channelId > 0,
  })
}

export function metadataProviderInstancesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.metadataProviderInstances(token),
    queryFn: () => createAuthedMiboApi(token).listMetadataProviderInstances(),
  })
}

export function pluginProviderInstancesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.pluginProviderInstances(token),
    queryFn: () => createAuthedMiboApi(token).listPluginProviderInstances(),
  })
}

export function pluginProviderDetailQueryOptions(
  token: string,
  providerId: number
) {
  return queryOptions({
    queryKey: miboQueryKeys.pluginProviderDetail(token, providerId),
    queryFn: () =>
      createAuthedMiboApi(token).getPluginProviderDetail(providerId),
    enabled: providerId > 0,
  })
}

export function localPluginInstallationsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.localPluginInstallations(token),
    queryFn: () => createAuthedMiboApi(token).listLocalPluginInstallations(),
  })
}

export function internalPluginsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.internalPlugins(token),
    queryFn: () => createAuthedMiboApi(token).listInternalPlugins(),
  })
}

export function openSubtitlesSettingsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.openSubtitlesSettings(token),
    queryFn: () => createAuthedMiboApi(token).getOpenSubtitlesSettings(),
  })
}

export function subtitleProviderInstancesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.subtitleProviderInstances(token),
    queryFn: () => createAuthedMiboApi(token).listSubtitleProviderInstances(),
  })
}

export function pluginCatalogOverviewQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.pluginCatalogOverview(token),
    queryFn: () => createAuthedMiboApi(token).getPluginCatalogOverview(),
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

export function libraryItemsQueryOptions(
  token: string,
  libraryId: number,
  options?: { type?: 'all' | 'movie' | 'show'; limit?: number }
) {
  return queryOptions({
    queryKey: ['library', 'items', token, libraryId, options] as const,
    queryFn: () =>
      createAuthedMiboApi(token).listLibraryItems(libraryId, options),
    enabled: libraryId > 0,
  })
}

export function libraryInventoryFilesQueryOptions(
  token: string,
  libraryId: number,
  options?: { page?: number; limit?: number }
) {
  return queryOptions({
    queryKey: [
      'library',
      'inventory-files',
      token,
      libraryId,
      options,
    ] as const,
    queryFn: () =>
      createAuthedMiboApi(token).listLibraryInventoryFiles(libraryId, options),
    enabled: libraryId > 0,
  })
}

export function inventoryFilesQueryOptions(
  token: string,
  options?: { page?: number; limit?: number; libraryId?: number; q?: string }
) {
  return queryOptions({
    queryKey: ['inventory-files', token, options] as const,
    queryFn: () =>
      createAuthedMiboApi(token).listInventoryFiles({
        page: options?.page,
        limit: options?.limit,
        library_id: options?.libraryId,
        q: options?.q,
      }),
    enabled: Boolean(token),
  })
}

export function schedulesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.schedules(token),
    queryFn: () => createAuthedMiboApi(token).listSchedules(),
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
