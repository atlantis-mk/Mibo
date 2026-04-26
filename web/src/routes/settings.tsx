import { createFileRoute } from '@tanstack/react-router'

import SettingsLayout from '#/features/settings'
import { requireCanEnterApp } from '#/lib/setup-gate'

export const Route = createFileRoute('/settings')({
  beforeLoad: async () => {
    await requireCanEnterApp()
  },
  component: SettingsLayout,
})
