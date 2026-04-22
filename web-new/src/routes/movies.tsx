import { createFileRoute, stripSearchParams } from '@tanstack/react-router'

import { LegacyMoviesRoute } from '~/features/app/legacy-app-shell'
import { validateBrowseSearch } from '~/lib/app-route-search'
import { DEFAULT_BROWSE_FILTERS } from '@/lib/mibo-api'

export const Route = createFileRoute('/movies')({
  validateSearch: validateBrowseSearch,
  search: {
    middlewares: [stripSearchParams(DEFAULT_BROWSE_FILTERS)],
  },
  component: MoviesRoute,
})

function MoviesRoute() {
  const search = Route.useSearch()

  return <LegacyMoviesRoute browseFilters={search} />
}
