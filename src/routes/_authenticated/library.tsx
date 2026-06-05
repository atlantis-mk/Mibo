import { useEffect, useRef } from 'react'
import {
  createFileRoute,
  useLocation,
  useNavigate,
} from '@tanstack/react-router'
import {
  createDefaultDiscoveryFilters,
  type DiscoveryFilters,
} from '@/features/discovery/controls'
import LibraryDetail, {
  DEFAULT_LIBRARY_PAGE_SIZE,
  isLibraryPageSize,
} from '@/features/library'

export const Route = createFileRoute('/_authenticated/library')({
  component: LibraryRoute,
  validateSearch: (search: Record<string, unknown>) => ({
    ...libraryFiltersToSearch(parseLibraryFiltersSearch(search)),
    ...(normalizeLibraryIdSearch(search.libraryId) !== undefined
      ? { libraryId: normalizeLibraryIdSearch(search.libraryId) }
      : {}),
    ...(normalizeLibraryPageSearch(search.page) !== undefined
      ? { page: normalizeLibraryPageSearch(search.page) }
      : {}),
    ...(normalizeLibraryPageSizeSearch(search.pageSize) !== undefined
      ? { pageSize: normalizeLibraryPageSizeSearch(search.pageSize) }
      : {}),
  }),
})

function LibraryRoute() {
  const navigate = useNavigate({ from: '/library' })
  const locationHref = useLocation({ select: (location) => location.href })
  const search = Route.useSearch()
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)

  const filters = filtersFromLibrarySearch(search)
  const page = search.page ?? 1
  const pageSize = search.pageSize ?? DEFAULT_LIBRARY_PAGE_SIZE
  const scrollStorageKey = `mibo:library-scroll:${locationHref}`

  useEffect(() => {
    let frameId = 0
    let attempts = 0

    const savedScrollTop = window.sessionStorage.getItem(scrollStorageKey)
    const parsedScrollTop =
      savedScrollTop === null ? 0 : Number.parseFloat(savedScrollTop)
    const targetScrollTop = Number.isFinite(parsedScrollTop)
      ? parsedScrollTop
      : 0

    const restoreScrollPosition = () => {
      const container = scrollContainerRef.current

      if (!container) {
        frameId = window.requestAnimationFrame(restoreScrollPosition)
        return
      }

      const maxScrollTop = Math.max(
        0,
        container.scrollHeight - container.clientHeight
      )
      const nextScrollTop = Math.min(targetScrollTop, maxScrollTop)

      container.scrollTop = nextScrollTop

      if (
        Math.abs(container.scrollTop - nextScrollTop) <= 1 &&
        (targetScrollTop <= maxScrollTop || attempts >= 30)
      ) {
        return
      }

      attempts += 1
      frameId = window.requestAnimationFrame(restoreScrollPosition)
    }

    frameId = window.requestAnimationFrame(restoreScrollPosition)

    return () => {
      if (frameId) {
        window.cancelAnimationFrame(frameId)
      }
    }
  }, [scrollStorageKey])

  useEffect(() => {
    const container = scrollContainerRef.current

    if (!container) return

    const persistScrollPosition = () => {
      window.sessionStorage.setItem(
        scrollStorageKey,
        String(container.scrollTop)
      )
    }

    container.addEventListener('scroll', persistScrollPosition, {
      passive: true,
    })

    return () => {
      container.removeEventListener('scroll', persistScrollPosition)
    }
  }, [scrollStorageKey])

  return (
    <LibraryDetail
      page={page}
      pageSize={pageSize}
      filters={filters}
      scrollContainerRef={scrollContainerRef}
      onPaginationChange={(next) => {
        void navigate({
          search: (previous) => ({
            ...previous,
            ...(next.page !== undefined ? { page: next.page } : {}),
            ...(next.pageSize !== undefined ? { pageSize: next.pageSize } : {}),
          }),
        })
      }}
      onFiltersChange={(next, options) => {
        void navigate({
          search: (previous) => ({
            ...previous,
            ...(options?.resetPage ? { page: 1 } : {}),
            ...libraryFiltersToSearch(next),
          }),
        })
      }}
    />
  )
}

function parseLibraryTypeSearch(
  value: unknown
): DiscoveryFilters['type'] | undefined {
  return value === 'movie' || value === 'show' || value === 'all'
    ? value
    : undefined
}

function parseLibrarySortSearch(
  value: unknown
): DiscoveryFilters['sort'] | undefined {
  return value === 'recent' ||
    value === 'imdb_rating' ||
    value === 'last_episode_release_date' ||
    value === 'last_episode_added_date' ||
    value === 'added_date' ||
    value === 'release_date' ||
    value === 'parental_rating' ||
    value === 'director' ||
    value === 'year' ||
    value === 'critic_rating' ||
    value === 'played_date' ||
    value === 'runtime' ||
    value === 'title' ||
    value === 'random' ||
    value === 'audience_rating' ||
    value === 'watch_status'
    ? value
    : undefined
}

function parseLibrarySortDirectionSearch(
  value: unknown
): DiscoveryFilters['sortDirection'] | undefined {
  return value === 'asc' || value === 'desc' ? value : undefined
}

function parseWatchedStateSearch(
  value: unknown
): DiscoveryFilters['watchedState'] | undefined {
  return value === 'all' ||
    value === 'unwatched' ||
    value === 'in_progress' ||
    value === 'watched'
    ? value
    : undefined
}

function parseOrganizingStateSearch(
  value: unknown
): DiscoveryFilters['organizingState'] | undefined {
  return value === 'organized' || value === 'unorganized' ? value : undefined
}

function normalizeStringSearch(value: unknown) {
  return typeof value === 'string' ? value : undefined
}

function normalizeLibraryPageSearch(value: unknown) {
  const parsed =
    typeof value === 'number'
      ? value
      : typeof value === 'string'
        ? Number.parseInt(value, 10)
        : Number.NaN

  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

function normalizeLibraryIdSearch(value: unknown) {
  const parsed =
    typeof value === 'number'
      ? value
      : typeof value === 'string'
        ? Number.parseInt(value, 10)
        : Number.NaN

  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

function normalizeLibraryPageSizeSearch(value: unknown) {
  const parsed =
    typeof value === 'number'
      ? value
      : typeof value === 'string'
        ? Number.parseInt(value, 10)
        : Number.NaN

  return Number.isFinite(parsed) && isLibraryPageSize(parsed)
    ? parsed
    : undefined
}

function parseLibraryFiltersSearch(search: Record<string, unknown>) {
  return createDefaultDiscoveryFilters({
    q: normalizeStringSearch(search.q),
    type: parseLibraryTypeSearch(search.type),
    genre: normalizeStringSearch(search.genre),
    region: normalizeStringSearch(search.region),
    year: normalizeStringSearch(search.year),
    minRating: normalizeStringSearch(search.minRating),
    libraryId: normalizeLibraryIdSearch(search.libraryId),
    watchedState: parseWatchedStateSearch(search.watchedState),
    organizingState:
      parseOrganizingStateSearch(search.organizingState) ?? 'organized',
    sort: parseLibrarySortSearch(search.sort),
    sortDirection: parseLibrarySortDirectionSearch(search.sortDirection),
  })
}

function libraryFiltersToSearch(filters: DiscoveryFilters) {
  return {
    q: filters.q || undefined,
    type: filters.type === 'all' ? undefined : filters.type,
    genre: filters.genre || undefined,
    region: filters.region || undefined,
    year: filters.year || undefined,
    minRating: filters.minRating || undefined,
    ...(filters.libraryId !== undefined
      ? { libraryId: filters.libraryId }
      : {}),
    watchedState:
      filters.watchedState === 'all' ? undefined : filters.watchedState,
    organizingState:
      filters.organizingState === 'all' ? undefined : filters.organizingState,
    sort: filters.sort === 'recent' ? undefined : filters.sort,
    sortDirection:
      filters.sortDirection === 'desc' ? undefined : filters.sortDirection,
  }
}

function filtersFromLibrarySearch(search: {
  q?: string
  type?: DiscoveryFilters['type']
  genre?: string
  region?: string
  year?: string
  minRating?: string
  libraryId?: number
  watchedState?: DiscoveryFilters['watchedState']
  organizingState?: DiscoveryFilters['organizingState']
  sort?: DiscoveryFilters['sort']
  sortDirection?: DiscoveryFilters['sortDirection']
}) {
  return createDefaultDiscoveryFilters(search)
}
