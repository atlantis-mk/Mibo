import { DEFAULT_BROWSE_FILTERS, type BrowseFilters, type BrowseSort, type BrowseTypeFilter } from '@/lib/mibo-api'

type BrowseSection = 'home' | 'movies' | 'shows'

export type MediaItemSearch = BrowseFilters & {
  from: BrowseSection
  libraryId: number | null
}

export const MEDIA_ITEM_SEARCH_DEFAULTS: MediaItemSearch = {
  ...DEFAULT_BROWSE_FILTERS,
  from: 'home',
  libraryId: null,
}

function normalizeBrowseType(value: unknown): BrowseTypeFilter {
  return value === 'movie' || value === 'show' ? value : 'all'
}

function normalizeBrowseSort(value: unknown): BrowseSort {
  return value === 'title' || value === 'year' || value === 'watch_status'
    ? value
    : 'recent'
}

function normalizePositiveInteger(value: unknown): number | null {
  if (typeof value === 'number' && Number.isInteger(value) && value > 0) {
    return value
  }

  if (typeof value === 'string') {
    const parsed = Number(value)

    if (Number.isInteger(parsed) && parsed > 0) {
      return parsed
    }
  }

  return null
}

export function validateBrowseSearch(search: Record<string, unknown>): BrowseFilters {
  return {
    type: normalizeBrowseType(search.type),
    year: normalizePositiveInteger(search.year),
    sort: normalizeBrowseSort(search.sort),
  }
}

export function validateMediaItemSearch(search: Record<string, unknown>): MediaItemSearch {
  const browse = validateBrowseSearch(search)

  return {
    ...browse,
    from: search.from === 'movies' || search.from === 'shows' ? search.from : 'home',
    libraryId: normalizePositiveInteger(search.libraryId),
  }
}
