import { createFileRoute } from '@tanstack/react-router'

import SchedulesPage from '#/features/schedules'

export const Route = createFileRoute('/settings/schedules')({
  component: SettingsSchedulesRoute,
})

function SettingsSchedulesRoute() {
  return <SchedulesPage />
}
