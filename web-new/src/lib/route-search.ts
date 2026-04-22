import { DEFAULT_BROWSE_FILTERS, type BrowseFilters } from '~/lib/mibo-api'

export function buildBrowseRouteSearch(filters: BrowseFilters) {
  const search: Partial<BrowseFilters> = {}

  if (filters.sort !== DEFAULT_BROWSE_FILTERS.sort) {
    search.sort = filters.sort
  }

  if (filters.type !== DEFAULT_BROWSE_FILTERS.type) {
    search.type = filters.type
  }

  if (filters.year !== null) {
    search.year = filters.year
  }

  return search
}
