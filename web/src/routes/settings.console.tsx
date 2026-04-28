import { createFileRoute } from '@tanstack/react-router'

import { SettingsConsolePage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/console')({
  component: SettingsConsoleRoute,
})

function SettingsConsoleRoute() {
  return <SettingsConsolePage />
}
