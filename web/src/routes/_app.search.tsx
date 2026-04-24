import { createFileRoute } from '@tanstack/react-router'

import SearchPage from '#/features/search'

export const Route = createFileRoute('/_app/search')({
  validateSearch: (search: Record<string, unknown>) => ({
    q: typeof search.q === 'string' ? search.q : undefined,
  }),
  component: SearchRoute,
})

function SearchRoute() {
  const search = Route.useSearch()

  return <SearchPage initialQuery={search.q} />
}
