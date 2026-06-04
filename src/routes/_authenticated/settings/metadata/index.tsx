import { createFileRoute } from '@tanstack/react-router'
import MetadataGovernancePage from '@/features/metadata-governance'
import { SettingsPageInset } from '@/features/settings/components/settings-page-inset'

export const Route = createFileRoute('/_authenticated/settings/metadata/')({
  component: SettingsMetadataPage,
})

function SettingsMetadataPage() {
  return (
    <SettingsPageInset fixedContent>
      <MetadataGovernancePage />
    </SettingsPageInset>
  )
}
