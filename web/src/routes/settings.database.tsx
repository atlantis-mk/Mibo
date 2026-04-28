import { createFileRoute } from '@tanstack/react-router'

import { SettingsDatabasePage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/database')({
  component: SettingsDatabaseRoute,
})

function SettingsDatabaseRoute() {
  return <SettingsDatabasePage />
}
