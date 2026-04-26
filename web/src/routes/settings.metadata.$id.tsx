import { createFileRoute } from '@tanstack/react-router'

import MetadataGovernancePage from '#/features/metadata-governance'

export const Route = createFileRoute('/settings/metadata/$id')({
  component: SettingsMetadataDetailRoute,
})

function SettingsMetadataDetailRoute() {
  const { id } = Route.useParams()

  return <MetadataGovernancePage itemId={Number(id)} />
}
