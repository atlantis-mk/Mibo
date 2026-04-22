import type { SetupStatus } from '~/lib/mibo-api'

export const TOKEN_STORAGE_KEY = 'mibo-web-session-token'
export const API_BASE_STORAGE_KEY = 'mibo-web-api-base-url'
export const SETUP_STATUS_EVENT = 'mibo:setup-status-changed'
export const defaultApiBaseUrl =
  import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:8080'

export function getStoredApiBaseUrl() {
  if (typeof window === 'undefined') {
    return defaultApiBaseUrl
  }

  return window.localStorage.getItem(API_BASE_STORAGE_KEY) ?? defaultApiBaseUrl
}

export function canEnterApp(status: SetupStatus) {
  return status.can_enter_app
}

export function isSetupFullyInitialized(status: SetupStatus) {
  return status.initialized
}

export function needsSetupGuide(
  status: Pick<SetupStatus, 'can_enter_app' | 'has_media_sources' | 'has_libraries'>
) {
  return status.can_enter_app && (!status.has_media_sources || !status.has_libraries)
}
