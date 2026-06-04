import { createFileRoute } from '@tanstack/react-router'
import { parseMediaDetailView } from '@/lib/media-presentation'
import MediaDetail from '@/features/media'

export const Route = createFileRoute('/_authenticated/media/$id')({
  component: MediaDetailRoute,
  validateSearch: (search: Record<string, unknown>) => ({
    view: parseMediaDetailView(search.view),
    episodePage:
      typeof search.episodePage === 'number'
        ? search.episodePage
        : typeof search.episodePage === 'string'
          ? Number.parseInt(search.episodePage, 10) || undefined
          : undefined,
  }),
})

function MediaDetailRoute() {
  const { id } = Route.useParams()
  const search = Route.useSearch()

  return (
    <MediaDetail
      itemId={Number(id)}
      detailView={search.view ?? 'episode'}
      episodePage={search.episodePage ?? 1}
    />
  )
}
