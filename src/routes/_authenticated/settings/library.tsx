import { createFileRoute } from '@tanstack/react-router'
import { SettingsLibraryPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/library')({
  component: SettingsLibraryPage,
})
