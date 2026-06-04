import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/live-tv/$channelId')({
  beforeLoad: ({ params, search }) => {
    const channelId = Number(params.channelId)

    throw redirect({
      to: '/play/$id',
      params: { id: String(channelId) },
      search: {
        fromStart: undefined,
        inventoryFileId: undefined,
        resourceId: undefined,
        liveChannelId: channelId,
        liveSourceId: search.sourceId,
      },
      replace: true,
    })
  },
  validateSearch: (search: Record<string, unknown>) => ({
    sourceId:
      typeof search.sourceId === 'number'
        ? search.sourceId
        : typeof search.sourceId === 'string'
          ? Number.parseInt(search.sourceId, 10) || undefined
          : undefined,
  }),
})
