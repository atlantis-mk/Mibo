import { createFileRoute } from '@tanstack/react-router'

import { SetupWizard } from '~/components/setup-wizard'

export const Route = createFileRoute('/setup')({
  component: SetupRoute,
})

function SetupRoute() {
  return <SetupWizard />
}
