import { redirect } from '@tanstack/react-router'
import { getAppSession, isAppSessionUnavailable } from '@/lib/app-session'

type AuthSessionState = 'authenticated' | 'unauthorized' | 'unavailable'

export function normalizeInternalRedirect(
  value: string | undefined,
  fallback = '/'
) {
  if (!isSafeInternalRedirect(value)) {
    return fallback
  }

  return value
}

export async function requireAuthenticated(redirectTo: string) {
  const authState = await ensureAuthenticatedSession()

  if (authState === 'authenticated') {
    return
  }

  if (authState === 'unavailable') {
    throw redirect({ to: '/503' })
  }

  throw redirect({
    to: '/sign-in',
    search: { redirect: normalizeInternalRedirect(redirectTo, '/') },
  })
}

export async function redirectAuthenticated(redirectTo?: string) {
  if ((await ensureAuthenticatedSession()) !== 'authenticated') {
    return
  }

  throw redirect({
    to: normalizeInternalRedirect(redirectTo, '/'),
  })
}

async function ensureAuthenticatedSession(): Promise<AuthSessionState> {
  try {
    const session = await getAppSession()
    return session.authenticated ? 'authenticated' : 'unauthorized'
  } catch (error) {
    if (isAppSessionUnavailable(error)) {
      return 'unavailable'
    }

    return 'unauthorized'
  }
}

function isSafeInternalRedirect(value: string | undefined): value is string {
  if (typeof value !== 'string' || !/^\/(?![\\/])/.test(value)) {
    return false
  }

  const normalizedUrl = new URL(value, 'http://localhost')
  return normalizedUrl.pathname !== '/sign-in'
}
