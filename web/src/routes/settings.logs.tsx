import { createFileRoute } from '@tanstack/react-router'

import LogsPage from '#/features/logs'

export const Route = createFileRoute('/settings/logs')({
  component: SettingsLogsRoute,
})

function SettingsLogsRoute() {
  return <LogsPage />
}
