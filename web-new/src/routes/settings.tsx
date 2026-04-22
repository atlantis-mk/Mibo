import { createFileRoute } from '@tanstack/react-router'

import { LegacySettingsRoute } from '~/features/app/legacy-app-shell'

export const Route = createFileRoute('/settings')({
  component: SettingsRoute,
})

function SettingsRoute() {
  return <LegacySettingsRoute />
}
