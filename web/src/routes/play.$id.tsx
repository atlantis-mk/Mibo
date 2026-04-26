import { createFileRoute } from '@tanstack/react-router'

import PlayExperience from '#/features/play'
import { requireCanEnterApp } from '#/lib/setup-gate'

export const Route = createFileRoute('/play/$id')({
  beforeLoad: async () => {
    await requireCanEnterApp()
  },
  validateSearch: (search: Record<string, unknown>) => ({
    fromStart:
      search.fromStart === true ||
      search.fromStart === 'true' ||
      search.fromStart === '1',
    assetId:
      typeof search.assetId === 'number'
        ? search.assetId
        : typeof search.assetId === 'string'
          ? Number.parseInt(search.assetId, 10) || undefined
          : undefined,
  }),
  component: PlayPage,
})

function PlayPage() {
  const { id } = Route.useParams()
  const { fromStart, assetId } = Route.useSearch()

  return (
    <PlayExperience
      itemId={Number(id)}
      assetId={assetId}
      fromStart={fromStart}
    />
  )
}
