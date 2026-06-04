import { createFileRoute } from '@tanstack/react-router'
import { SettingsPluginsPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/plugins')({
  component: SettingsPluginsPage,
})
