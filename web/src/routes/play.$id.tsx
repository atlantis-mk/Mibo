import { createFileRoute } from '@tanstack/react-router'

import PlayExperience from '#/features/play'

export const Route = createFileRoute('/play/$id')({
  validateSearch: (search: Record<string, unknown>) => ({
    fromStart:
      search.fromStart === true ||
      search.fromStart === 'true' ||
      search.fromStart === '1',
  }),
  component: PlayPage,
})

function PlayPage() {
  const { id } = Route.useParams()
  const { fromStart } = Route.useSearch()

  return <PlayExperience mediaItemId={Number(id)} fromStart={fromStart} />
}
