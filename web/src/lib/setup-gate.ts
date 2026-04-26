import { redirect } from '@tanstack/react-router'

import { createMiboApi, getApiBaseUrl } from '#/lib/mibo-api'

export function normalizeInternalRedirect(
  value: string | undefined,
  fallback = '/',
) {
  if (!value || !value.startsWith('/') || value === '/setup') {
    return fallback
  }

  return value
}

export async function requireCanEnterApp() {
  const status = await createMiboApi({
    baseUrl: getApiBaseUrl(),
  }).getSetupStatus()

  if (status.can_enter_app) {
    return status
  }

  throw redirect({
    to: '/setup',
    search: { redirect: undefined },
  })
}

export async function requireSetupAccess(redirectTo?: string) {
  const status = await createMiboApi({
    baseUrl: getApiBaseUrl(),
  }).getSetupStatus()

  if (!status.can_enter_app) {
    return status
  }

  throw redirect({
    to: normalizeInternalRedirect(redirectTo, '/'),
  })
}
