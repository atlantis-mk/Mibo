import { createFileRoute } from '@tanstack/react-router'
import { SettingsNetworkPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/network')({
  component: SettingsNetworkPage,
})
