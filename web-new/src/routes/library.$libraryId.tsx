import { createFileRoute, stripSearchParams } from '@tanstack/react-router'

import { LegacyLibraryRoute } from '~/features/app/legacy-app-shell'
import { validateBrowseSearch } from '~/lib/app-route-search'
import { DEFAULT_BROWSE_FILTERS } from '@/lib/mibo-api'

export const Route = createFileRoute('/library/$libraryId')({
  validateSearch: validateBrowseSearch,
  search: {
    middlewares: [stripSearchParams(DEFAULT_BROWSE_FILTERS)],
  },
  component: LibraryRoute,
})

function LibraryRoute() {
  const { libraryId } = Route.useParams()
  const search = Route.useSearch()

  return <LegacyLibraryRoute browseFilters={search} libraryId={Number(libraryId)} />
}
