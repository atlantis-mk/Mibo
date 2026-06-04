import { createFileRoute } from '@tanstack/react-router'
import type { DiscoveryFilters } from '@/features/discovery/controls'
import SearchPage from '@/features/search'

export const Route = createFileRoute('/_authenticated/search')({
  component: SearchRoute,
  validateSearch: (search: Record<string, unknown>) => ({
    q: typeof search.q === 'string' ? search.q : undefined,
    type: parseSearchType(search.type),
    ...(normalizeSearchPage(search.page) !== undefined
      ? { page: normalizeSearchPage(search.page) }
      : {}),
  }),
})

function SearchRoute() {
  const search = Route.useSearch()

  return (
    <SearchPage
      key={`${search.q ?? ''}:${search.type ?? 'all'}`}
      initialQuery={search.q}
      initialType={search.type}
      page={search.page ?? 1}
    />
  )
}

function parseSearchType(value: unknown): DiscoveryFilters['type'] | undefined {
  return value === 'movie' || value === 'show' || value === 'all'
    ? value
    : undefined
}

function normalizeSearchPage(value: unknown) {
  const parsed =
    typeof value === 'number'
      ? value
      : typeof value === 'string'
        ? Number.parseInt(value, 10)
        : Number.NaN

  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}
