import { createFileRoute } from '@tanstack/react-router'

import MediaDetail from '#/features/media'
import { parseMediaDetailView } from '#/lib/media-presentation'

export const Route = createFileRoute('/_app/media/$id')({
  validateSearch: (search: Record<string, unknown>) => ({
    view: search.view === 'series' ? 'series' : undefined,
  }),
  component: MediaDetailPage,
})

function MediaDetailPage() {
  const { id } = Route.useParams()
  const { view } = Route.useSearch()

  return (
    <MediaDetail itemId={Number(id)} detailView={parseMediaDetailView(view)} />
  )
}
