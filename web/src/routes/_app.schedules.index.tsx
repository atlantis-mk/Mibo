import { createFileRoute } from '@tanstack/react-router'

import SchedulesPage from '#/features/schedules'

export const Route = createFileRoute('/_app/schedules/')({
  component: SchedulesWorkspaceRoute,
})

function SchedulesWorkspaceRoute() {
  return <SchedulesPage />
}
