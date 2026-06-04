import { createFileRoute } from '@tanstack/react-router'
import PlayPage from '@/features/play'

export const Route = createFileRoute('/play/$id')({
  component: PlayRoute,
  validateSearch: (search: Record<string, unknown>) => ({
    fromStart:
      search.fromStart === true ||
      search.fromStart === 'true' ||
      search.fromStart === '1'
        ? true
        : undefined,
    inventoryFileId:
      typeof search.inventoryFileId === 'number'
        ? search.inventoryFileId
        : typeof search.inventoryFileId === 'string'
          ? Number.parseInt(search.inventoryFileId, 10) || undefined
          : undefined,
    resourceId:
      typeof search.resourceId === 'number'
        ? search.resourceId
        : typeof search.resourceId === 'string'
          ? Number.parseInt(search.resourceId, 10) || undefined
          : undefined,
    liveChannelId:
      typeof search.liveChannelId === 'number'
        ? search.liveChannelId
        : typeof search.liveChannelId === 'string'
          ? Number.parseInt(search.liveChannelId, 10) || undefined
          : undefined,
    liveSourceId:
      typeof search.liveSourceId === 'number'
        ? search.liveSourceId
        : typeof search.liveSourceId === 'string'
          ? Number.parseInt(search.liveSourceId, 10) || undefined
          : undefined,
  }),
})

function PlayRoute() {
  const { id } = Route.useParams()
  const search = Route.useSearch()

  return (
    <PlayPage
      itemId={Number(id)}
      fromStart={search.fromStart}
      inventoryFileId={search.inventoryFileId}
      resourceId={search.resourceId}
      liveTVChannelId={search.liveChannelId}
      liveTVSourceId={search.liveSourceId}
    />
  )
}
