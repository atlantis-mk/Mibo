import { createFileRoute } from '@tanstack/react-router'
import MetadataGovernancePage from '@/features/metadata-governance'
import { SettingsPageInset } from '@/features/settings/components/settings-page-inset'

export const Route = createFileRoute('/_authenticated/settings/metadata/$id')({
  component: MetadataDetailRoute,
})

function MetadataDetailRoute() {
  const { id } = Route.useParams()

  return (
    <SettingsPageInset>
      <MetadataGovernancePage itemId={Number(id)} />
    </SettingsPageInset>
  )
}
