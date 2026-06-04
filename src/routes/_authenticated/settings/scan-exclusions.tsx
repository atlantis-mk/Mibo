import { createFileRoute } from '@tanstack/react-router'
import { SettingsScanExclusionsPage } from '@/features/settings/pages'

export const Route = createFileRoute(
  '/_authenticated/settings/scan-exclusions'
)({
  component: SettingsScanExclusionsPage,
})
