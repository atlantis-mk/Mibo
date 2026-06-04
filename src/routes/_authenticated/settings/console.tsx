import { createFileRoute } from '@tanstack/react-router'
import { SettingsConsolePage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/console')({
  component: SettingsConsolePage,
})
