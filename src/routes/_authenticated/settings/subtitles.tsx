import { createFileRoute } from '@tanstack/react-router'
import { SettingsSubtitlesPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/subtitles')({
  component: SettingsSubtitlesPage,
})
