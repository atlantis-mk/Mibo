import { createFileRoute } from '@tanstack/react-router'

import MetadataGovernancePage from '#/features/metadata-governance'

export const Route = createFileRoute('/_app/metadata/')({
  component: MetadataGovernanceWorkspaceRoute,
})

function MetadataGovernanceWorkspaceRoute() {
  return <MetadataGovernancePage />
}
