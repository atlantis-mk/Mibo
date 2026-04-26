import { createFileRoute } from '@tanstack/react-router'

import SetupPage from '#/features/setup'
import { normalizeInternalRedirect, requireSetupAccess } from '#/lib/setup-gate'

export const Route = createFileRoute('/setup')({
  validateSearch: (search: Record<string, unknown>) => ({
    redirect:
      typeof search.redirect === 'string' && search.redirect.startsWith('/')
        ? search.redirect
        : undefined,
  }),
  beforeLoad: async ({ search }) => {
    await requireSetupAccess(search.redirect)
  },
  component: SetupRoute,
})

function SetupRoute() {
  const { redirect } = Route.useSearch()

  return <SetupPage redirectTo={normalizeInternalRedirect(redirect, '/')} />
}
