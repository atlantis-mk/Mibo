import { createFileRoute } from '@tanstack/react-router'
import { SettingsRolesPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/roles')({
  component: SettingsRolesPage,
})
