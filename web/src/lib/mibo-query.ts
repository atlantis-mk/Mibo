import { queryOptions } from '@tanstack/react-query'

import { createMiboApi, getApiBaseUrl } from '#/lib/mibo-api'

export const miboQueryKeys = {
  authUser: (token: string) => ['auth', 'me', token] as const,
  homeData: (token: string) => ['home', 'hero', token] as const,
  mediaItemDetail: (token: string, mediaItemId: number) =>
    ['media', 'detail', token, mediaItemId] as const,
  mediaItemProgress: (token: string, mediaItemId: number) =>
    ['media', 'progress', token, mediaItemId] as const,
  metadataWorkspace: (token: string) =>
    ['metadata', 'workspace', token] as const,
  mediaSources: (token: string) =>
    ['settings', 'media-sources', token] as const,
  libraries: (token: string) => ['settings', 'libraries', token] as const,
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

export function mediaSourcesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.mediaSources(token),
    queryFn: () => createAuthedMiboApi(token).listMediaSources(),
  })
}

export function librariesQueryOptions(token: string) {
  return queryOptions({
    queryKey: miboQueryKeys.libraries(token),
    queryFn: () => createAuthedMiboApi(token).listLibraries(),
  })
}
