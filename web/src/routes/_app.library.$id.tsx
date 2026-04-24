import { createFileRoute } from '@tanstack/react-router'

import LibraryDetail from '#/features/library'

export const Route = createFileRoute('/_app/library/$id')({
  component: LibraryDetailPage,
})

function LibraryDetailPage() {
  const { id } = Route.useParams()

  return <LibraryDetail libraryId={Number(id)} />
}
