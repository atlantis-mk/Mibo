import { createFileRoute } from '@tanstack/react-router'
import { SettingsOperationsPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/operations/')({
  component: SettingsOperationsPage,
})
