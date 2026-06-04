import { createFileRoute } from '@tanstack/react-router'
import { SettingsUsersPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/users')({
  component: SettingsUsersPage,
})
