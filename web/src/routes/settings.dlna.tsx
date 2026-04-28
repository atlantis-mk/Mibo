import { createFileRoute } from '@tanstack/react-router'

import { SettingsDlnaPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/dlna')({
  component: SettingsDlnaRoute,
})

function SettingsDlnaRoute() {
  return <SettingsDlnaPage />
}
