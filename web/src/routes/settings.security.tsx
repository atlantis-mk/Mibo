import { createFileRoute } from '@tanstack/react-router'

import { SettingsSecurityPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/security')({
  component: SettingsSecurityRoute,
})

function SettingsSecurityRoute() {
  return <SettingsSecurityPage />
}
