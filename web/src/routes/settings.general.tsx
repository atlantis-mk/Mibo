import { createFileRoute } from '@tanstack/react-router'

import { SettingsGeneralPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/general')({
  component: SettingsGeneralRoute,
})

function SettingsGeneralRoute() {
  return <SettingsGeneralPage />
}
