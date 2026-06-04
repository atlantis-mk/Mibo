import { createFileRoute } from '@tanstack/react-router'
import { SettingsSecurityPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/security')({
  component: SettingsSecurityPage,
})
