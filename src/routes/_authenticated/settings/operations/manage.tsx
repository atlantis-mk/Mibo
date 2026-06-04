import { createFileRoute } from '@tanstack/react-router'
import { SettingsOperationsManagePage } from '@/features/settings/pages'

export const Route = createFileRoute(
  '/_authenticated/settings/operations/manage'
)({
  component: SettingsOperationsManagePage,
})
