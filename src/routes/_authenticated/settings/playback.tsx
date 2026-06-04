import { createFileRoute } from '@tanstack/react-router'
import { SettingsPlaybackPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/playback')({
  component: SettingsPlaybackPage,
})
