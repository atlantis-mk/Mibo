import { createFileRoute } from '@tanstack/react-router'

import MediaDetail from '#/features/media'

export const Route = createFileRoute('/_app/media/$id')({
  component: MediaDetailPage,
})

function MediaDetailPage() {
  const { id } = Route.useParams()

  return <MediaDetail mediaItemId={Number(id)} />
}
