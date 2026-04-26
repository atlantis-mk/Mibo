import { createFileRoute } from '@tanstack/react-router'

import { SettingsNotificationsPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/notifications')({
  component: SettingsNotificationsRoute,
})

function SettingsNotificationsRoute() {
  return <SettingsNotificationsPage />
}
