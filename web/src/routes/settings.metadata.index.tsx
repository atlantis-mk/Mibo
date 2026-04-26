import { createFileRoute } from '@tanstack/react-router'

import MetadataGovernancePage from '#/features/metadata-governance'

export const Route = createFileRoute('/settings/metadata/')({
  component: SettingsMetadataWorkspaceRoute,
})

function SettingsMetadataWorkspaceRoute() {
  return <MetadataGovernancePage />
}
