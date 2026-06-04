import { createFileRoute } from '@tanstack/react-router'
import { normalizeInternalRedirect } from '@/lib/auth-guard'
import { SignIn } from '@/features/auth/sign-in'

export const Route = createFileRoute('/(auth)/sign-in')({
  component: SignIn,
  validateSearch: (search: Record<string, unknown>) => {
    const redirect =
      typeof search.redirect === 'string'
        ? normalizeInternalRedirect(search.redirect, '')
        : ''

    return {
      redirect: redirect || undefined,
    }
  },
})
