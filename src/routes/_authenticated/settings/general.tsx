import { createFileRoute } from '@tanstack/react-router'
import { SettingsGeneralPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/general')({
  component: SettingsGeneralPage,
})
