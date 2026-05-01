import { redirect } from "@tanstack/react-router"

import { createMiboApi, getApiBaseUrl } from "#/lib/mibo-api"

const CAN_ENTER_APP_CACHE_MS = 5 * 60 * 1000

let canEnterAppCacheExpiresAt = 0
let canEnterAppStatusPromise: ReturnType<
  ReturnType<typeof createMiboApi>["getSetupStatus"]
> | null = null

export function normalizeInternalRedirect(
  value: string | undefined,
  fallback = "/"
) {
  if (!value || !value.startsWith("/") || value === "/setup") {
    return fallback
  }

  return value
}

export async function requireCanEnterApp() {
  const status = await getCanEnterAppStatus()

  if (status.can_enter_app) {
    return status
  }

  throw redirect({
    to: "/setup",
    search: { redirect: undefined },
  })
}

export async function requireSetupAccess(redirectTo?: string) {
  if (Date.now() < canEnterAppCacheExpiresAt) {
    throw redirect({
      to: normalizeInternalRedirect(redirectTo, "/"),
    })
  }

  const status = await createMiboApi({
    baseUrl: getApiBaseUrl(),
  }).getSetupStatus()

  if (!status.can_enter_app) {
    return status
  }

  throw redirect({
    to: normalizeInternalRedirect(redirectTo, "/"),
  })
}

async function getCanEnterAppStatus() {
  if (Date.now() < canEnterAppCacheExpiresAt && canEnterAppStatusPromise) {
    return canEnterAppStatusPromise
  }

  canEnterAppStatusPromise = createMiboApi({
    baseUrl: getApiBaseUrl(),
  }).getSetupStatus()

  const status = await canEnterAppStatusPromise
  if (status.can_enter_app) {
    canEnterAppCacheExpiresAt = Date.now() + CAN_ENTER_APP_CACHE_MS
  } else {
    canEnterAppStatusPromise = null
    canEnterAppCacheExpiresAt = 0
  }

  return status
}
