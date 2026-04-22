declare module '@/lib/mibo-api' {
  export type BrowseTypeFilter = 'all' | 'movie' | 'show'

  export type BrowseSort = 'recent' | 'title' | 'year' | 'watch_status'

  export type BrowseFilters = {
    type: BrowseTypeFilter
    year: number | null
    sort: BrowseSort
  }

  export const DEFAULT_BROWSE_FILTERS: BrowseFilters

  export function createMiboApi(options: {
    baseUrl: string
    token?: string | null
  }): {
    getMediaItem(mediaItemId: number): Promise<{
      library_id: number
    }>
  }
}

declare module '@/lib/client-config' {
  export const TOKEN_STORAGE_KEY: string
  export function getStoredApiBaseUrl(): string
}

declare module '@/features/app/pages/home-page' {
  import type { ComponentType } from 'react'
  import type { BrowseFilters } from '@/lib/mibo-api'

  export const HomePage: ComponentType<{
    browseFilters: BrowseFilters
  }>
}

declare module '@/features/app/pages/movies-page' {
  import type { ComponentType } from 'react'
  import type { BrowseFilters } from '@/lib/mibo-api'

  export const MoviesPage: ComponentType<{
    browseFilters: BrowseFilters
  }>
}

declare module '@/features/app/pages/shows-page' {
  import type { ComponentType } from 'react'
  import type { BrowseFilters } from '@/lib/mibo-api'

  export const ShowsPage: ComponentType<{
    browseFilters: BrowseFilters
  }>
}

declare module '@/features/app/pages/library-page' {
  import type { ComponentType } from 'react'
  import type { BrowseFilters } from '@/lib/mibo-api'

  export const LibraryPage: ComponentType<{
    browseFilters: BrowseFilters
    libraryId: number
  }>
}

declare module '@/features/app/pages/settings-page' {
  import type { ComponentType } from 'react'

  export const SettingsPage: ComponentType
}

declare module '@/features/app/pages/media-item-page' {
  import type { ComponentType } from 'react'
  import type { BrowseFilters } from '@/lib/mibo-api'

  export const MediaItemPage: ComponentType<{
    browseFilters: BrowseFilters
    libraryId: number
    mediaItemId: number
    originLibraryId: number | null
    originSection: 'home' | 'movies' | 'shows'
  }>
}
