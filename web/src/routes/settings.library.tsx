import { createFileRoute } from '@tanstack/react-router'

import { SettingsLibraryPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/library')({
  component: SettingsLibraryRoute,
})

function SettingsLibraryRoute() {
  return <SettingsLibraryPage />
}
