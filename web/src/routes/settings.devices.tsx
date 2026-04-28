import { createFileRoute } from '@tanstack/react-router'

import { SettingsDevicesPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/devices')({
  component: SettingsDevicesRoute,
})

function SettingsDevicesRoute() {
  return <SettingsDevicesPage />
}
