import { queryOptions } from '@tanstack/react-query'

import { createMiboApi, getApiBaseUrl } from '#/lib/mibo-api'

export const miboQueryKeys = {
  authUser: (token: string) => ['auth', 'me', token] as const,
  homeData: (token: string) => ['home', 'hero', token] as const,
  favorites: (token: string) => ['me', 'favorites', token] as const,
  consoleSummary: (token: string) => ['admin', 'console', token] as const,
  adminLogs: (token: string) => ['admin', 'logs', token] as const,
  libraryDetail: (token: string, libraryId: number) =>
    ['library', 'detail', token, libraryId] as const,
  libraryBrowse: (
    token: string,
    libraryId: number,
    tab: string,
    filters: unknown,
    page: number,
  ) => ['library', 'browse', token, libraryId, tab, filters, page] as const,
  catalogItemDetail: (token: string, itemId: number) =>
    ['catalog', 'detail', token, itemId] as const,
  catalogPersonDetail: (token: string, personId: number) =>
    ['catalog', 'person-detail', token, personId] as const,
  catalogItemProgress: (token: string, itemId: number) =>
    ['catalog', 'progress', token, itemId] as const,
  catalogSeriesSeasons: (token: string, itemId: number) =>
    ['catalog', 'series-seasons', token, itemId] as const,
  catalogPlayback: (token: string, itemId: number, assetId?: number) =>
    ['catalog', 'playback', token, itemId, assetId ?? 'default'] as const,
  catalogGovernanceWorkspace: (token: string, itemId: number) =>
    ['catalog', 'governance', token, itemId] as const,
  metadataWorkspace: (token: string) =>
    ['metadata', 'workspace', token] as const,
  metadataSettings: (token: string) => ['settings', 'metadata', token] as const,
  mediaSources: (token: string) =>
    ['settings', 'media-sources', token] as const,
  libraries: (token: string) => ['settings', 'libraries', token] as const,
  schedules: (token: string) => ['schedules', 'workspace', token] as const,
  scheduleDetail: (token: string, scheduleId: number) =>
    ['schedules', 'detail', token, scheduleId] as const,
  scheduleHistory: (token: string, scheduleId: number) =>
    ['schedules', 'history', token, scheduleId] as const,
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

export function homeDataQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.homeData(token),
    queryFn: async () => {
      const api = createAuthedMiboApi(token)
      const [items, continueWatching, libraries, latestByLibrary] =
        await Promise.all([
          api.recentlyAdded(6),
          api.continueWatching(),
          api.listLibraries(),
          api.latestByLibrary(),
        ])

      return {
        items,
        continueWatching,
        continueWatchingCount: continueWatching.length,
        libraries,
        libraryCount: libraries.length,
        latestByLibrary,
      }
    },
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

export function adminLogsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.adminLogs(token),
    queryFn: () => createAuthedMiboApi(token).listAdminLogs(),
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
  personId: number,
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
        return await createAuthedMiboApi(token).getCatalogItemProgress(itemId)
      } catch {
        return null
      }
    },
  })
}

export function catalogSeriesSeasonsQueryOptions(
  token: string,
  itemId: number,
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
  assetId?: number,
) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogPlayback(token, itemId, assetId),
    queryFn: () =>
      createAuthedMiboApi(token).getCatalogPlayback(itemId, {
        clientProfile: 'web',
        assetId,
      }),
    enabled: itemId > 0,
  })
}

export function catalogGovernanceWorkspaceQueryOptions(
  token: string,
  itemId: number,
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

export function metadataSettingsQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.metadataSettings(token),
    queryFn: () => createAuthedMiboApi(token).getMetadataSettings(),
  })
}

export function librariesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.libraries(token),
    queryFn: () => createAuthedMiboApi(token).listLibraries(),
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
