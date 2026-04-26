import { createFileRoute } from '@tanstack/react-router'

import { SettingsMetadataSourcesPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/metadata-sources')({
  component: SettingsMetadataSourcesRoute,
})

function SettingsMetadataSourcesRoute() {
  return <SettingsMetadataSourcesPage />
}
