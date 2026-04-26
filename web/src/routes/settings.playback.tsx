import { createFileRoute } from '@tanstack/react-router'

import { SettingsPlaybackPage } from '#/features/settings/pages'

export const Route = createFileRoute('/settings/playback')({
  component: SettingsPlaybackRoute,
})

function SettingsPlaybackRoute() {
  return <SettingsPlaybackPage />
}
