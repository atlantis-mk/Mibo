import { queryOptions } from '@tanstack/react-query'

import { createMiboApi, getApiBaseUrl } from '#/lib/mibo-api'

export const miboQueryKeys = {
  authUser: (token: string) => ['auth', 'me', token] as const,
  homeData: (token: string) => ['home', 'hero', token] as const,
  catalogItemDetail: (token: string, itemId: number) =>
    ['catalog', 'detail', token, itemId] as const,
  catalogItemProgress: (token: string, itemId: number) =>
    ['catalog', 'progress', token, itemId] as const,
  catalogSeriesSeasons: (token: string, itemId: number) =>
    ['catalog', 'series-seasons', token, itemId] as const,
  catalogPlayback: (token: string, itemId: number, assetId?: number) =>
    ['catalog', 'playback', token, itemId, assetId ?? 'default'] as const,
  catalogGovernanceWorkspace: (token: string, itemId: number) =>
    ['catalog', 'governance', token, itemId] as const,
  mediaItemDetail: (token: string, mediaItemId: number) =>
    ['media', 'detail', token, mediaItemId] as const,
  mediaItemProgress: (token: string, mediaItemId: number) =>
    ['media', 'progress', token, mediaItemId] as const,
  tvSeriesEpisodes: (
    token: string,
    mediaItemId: number,
    tmdbId: number,
    libraryId: number,
  ) =>
    [
      'media',
      'tv-series-episodes',
      token,
      mediaItemId,
      tmdbId,
      libraryId,
    ] as const,
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
        continueWatchingCount: continueWatching.length,
        libraryCount: libraries.length,
        latestByLibrary,
      }
    },
  })
}

export function mediaItemDetailQueryOptions(
  token: string,
  mediaItemId: number,
) {
  return queryOptions({
    queryKey: miboQueryKeys.mediaItemDetail(token, mediaItemId),
    queryFn: () => createAuthedMiboApi(token).getMediaItem(mediaItemId),
  })
}

export function catalogItemDetailQueryOptions(token: string, itemId: number) {
  return queryOptions({
    queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
    queryFn: () => createAuthedMiboApi(token).getCatalogItem(itemId),
  })
}

export function mediaItemProgressQueryOptions(
  token: string,
  mediaItemId: number,
) {
  return queryOptions({
    queryKey: miboQueryKeys.mediaItemProgress(token, mediaItemId),
    queryFn: async () => {
      try {
        return await createAuthedMiboApi(token).getMediaItemProgress(
          mediaItemId,
        )
      } catch {
        return null
      }
    },
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

export function tvSeriesEpisodesQueryOptions(
  token: string,
  mediaItemId: number,
  tmdbId: number,
  libraryId: number,
) {
  return queryOptions({
    queryKey: miboQueryKeys.tvSeriesEpisodes(
      token,
      mediaItemId,
      tmdbId,
      libraryId,
    ),
    queryFn: async () => {
      const api = createAuthedMiboApi(token)
      if (tmdbId > 0 && libraryId > 0) {
        try {
          const seasons = await api.listTVSeasons(tmdbId)
          const seasonDetails = await Promise.all(
            seasons.map(async (season) => ({
              ...season,
              episodes: await api.listTVSeasonEpisodes(
                tmdbId,
                season.season_number,
                {
                  libraryId,
                },
              ),
            })),
          )
          const matchedSeasons = seasonDetails.filter(
            (season) => season.episodes.length > 0,
          )
          if (matchedSeasons.length > 0) {
            return matchedSeasons
          }
        } catch {
          // Fall back to local scan data when TMDB data is unavailable.
        }
      }

      return api.listLocalSeriesEpisodes(mediaItemId)
    },
    enabled: mediaItemId > 0,
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
