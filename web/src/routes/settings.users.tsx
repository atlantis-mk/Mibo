import { createFileRoute } from '@tanstack/react-router'

import { SettingsUsersPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/users')({
  component: SettingsUsersRoute,
})

function SettingsUsersRoute() {
  return <SettingsUsersPage />
}
