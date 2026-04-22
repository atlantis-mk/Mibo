import { createFileRoute, stripSearchParams } from '@tanstack/react-router'

import { LegacyShowsRoute } from '~/features/app/legacy-app-shell'
import { validateBrowseSearch } from '~/lib/app-route-search'
import { DEFAULT_BROWSE_FILTERS } from '@/lib/mibo-api'

export const Route = createFileRoute('/shows')({
  validateSearch: validateBrowseSearch,
  search: {
    middlewares: [stripSearchParams(DEFAULT_BROWSE_FILTERS)],
  },
  component: ShowsRoute,
})

function ShowsRoute() {
  const search = Route.useSearch()

  return <LegacyShowsRoute browseFilters={search} />
}
