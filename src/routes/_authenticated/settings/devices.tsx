import { createFileRoute } from '@tanstack/react-router'
import { SettingsDevicesPage } from '@/features/settings/pages'

export const Route = createFileRoute('/_authenticated/settings/devices')({
  component: SettingsDevicesPage,
})
