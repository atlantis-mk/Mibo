import { createFileRoute } from '@tanstack/react-router'

import { SettingsLiveTvPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/live-tv')({
  component: SettingsLiveTvRoute,
})

function SettingsLiveTvRoute() {
  return <SettingsLiveTvPage />
}
