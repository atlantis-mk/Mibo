import { createFileRoute } from '@tanstack/react-router'
import SchedulesPage from '@/features/schedules'
import { SettingsPageInset } from '@/features/settings/components/settings-page-inset'

export const Route = createFileRoute('/_authenticated/settings/schedules')({
  component: SettingsSchedulesPage,
})

function SettingsSchedulesPage() {
  return (
    <SettingsPageInset>
      <SchedulesPage />
    </SettingsPageInset>
  )
}
