import { createFileRoute } from '@tanstack/react-router'
import { SettingsDlnaPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/dlna')({
  component: SettingsDlnaPage,
})
