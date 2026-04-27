import { createFileRoute } from '@tanstack/react-router'

import PersonDetailPage from '#/features/person'

export const Route = createFileRoute('/_app/person/$id')({
  component: PersonRoutePage,
})

function PersonRoutePage() {
  const { id } = Route.useParams()

  return <PersonDetailPage personId={Number(id)} />
}
