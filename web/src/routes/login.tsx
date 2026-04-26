import { createFileRoute } from '@tanstack/react-router'

import { LoginForm } from '#/components/login-form'
import { requireCanEnterApp } from '#/lib/setup-gate'

export const Route = createFileRoute('/login')({
  validateSearch: (search: Record<string, unknown>) => ({
    redirect:
      typeof search.redirect === 'string' &&
      search.redirect.startsWith('/') &&
      search.redirect !== '/login'
        ? search.redirect
        : undefined,
  }),
  beforeLoad: async () => {
    await requireCanEnterApp()
  },
  component: LoginPage,
})

function LoginPage() {
  const { redirect } = Route.useSearch()

  return (
    <div className="flex min-h-svh flex-col items-center justify-center bg-muted p-6 md:p-10">
      <div className="w-full max-w-sm md:max-w-4xl">
        <LoginForm redirectTo={redirect ?? '/'} />
      </div>
    </div>
  )
}
