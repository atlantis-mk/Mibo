import { createFileRoute } from '@tanstack/react-router'

import MetadataGovernancePage from '#/features/metadata-governance'

export const Route = createFileRoute('/_app/metadata/$id')({
  component: MetadataGovernanceDetailRoute,
})

function MetadataGovernanceDetailRoute() {
  const { id } = Route.useParams()

  return <MetadataGovernancePage mediaItemId={Number(id)} />
}
