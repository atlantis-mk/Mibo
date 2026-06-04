import { createFileRoute } from '@tanstack/react-router'
import { SettingsMetadataSourcesPage } from '@/features/settings/pages'

export const Route = createFileRoute(
  '/_authenticated/settings/metadata-sources'
)({
  component: SettingsMetadataSourcesPage,
})
