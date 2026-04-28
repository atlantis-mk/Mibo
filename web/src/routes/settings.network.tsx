import { createFileRoute } from '@tanstack/react-router'

import { SettingsNetworkPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/network')({
  component: SettingsNetworkRoute,
})

function SettingsNetworkRoute() {
  return <SettingsNetworkPage />
}
