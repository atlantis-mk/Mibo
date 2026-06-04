import { createFileRoute } from '@tanstack/react-router'
import { SettingsLiveTvPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/live-tv')({
  component: SettingsLiveTvPage,
})
