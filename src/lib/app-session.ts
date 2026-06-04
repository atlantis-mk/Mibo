import { useAuthStore } from '@/stores/auth-store'
import {
  ApiError,
  createMiboApi,
  getApiBaseUrl,
  type AppSession,
} from '@/lib/mibo-api'

let cachedToken: string | null = null
let cachedSession: AppSession | null = null
let cachedAt = 0
let sessionPromise: Promise<AppSession> | null = null

const AUTHENTICATED_APP_SESSION_TTL_MS = 10_000

export async function getAppSession() {
  await waitForAuthHydration()

  const { accessToken, clearSession, setSession } = useAuthStore.getState().auth
  const token = accessToken ?? null
  const now = Date.now()
  const cacheTtl = token ? AUTHENTICATED_APP_SESSION_TTL_MS : 0

  if (cachedSession && cachedToken === token && now - cachedAt < cacheTtl) {
    return cachedSession
  }

  if (sessionPromise && cachedToken === token) {
    return sessionPromise
  }

  cachedToken = token
  sessionPromise = createMiboApi({
    baseUrl: getApiBaseUrl(),
    token,
  })
    .getAppSession()
    .then((session) => {
      cachedSession = session
      cachedAt = Date.now()

      const currentToken = useAuthStore.getState().auth.accessToken ?? null

      if (
        token &&
        currentToken === token &&
        session.authenticated &&
        session.user
      ) {
        setSession({ token, user: session.user })
      }

      if (token && currentToken === token && !session.authenticated) {
        clearSession()
      }

      sessionPromise = null
      return session
    })
    .catch((error) => {
      cachedSession = null
      cachedAt = 0
      sessionPromise = null
      throw error
    })

  return sessionPromise
}

export function isAppSessionUnavailable(error: unknown) {
  return (
    error instanceof ApiError && (error.status === 0 || error.status >= 500)
  )
}

function waitForAuthHydration() {
  if (useAuthStore.getState().auth.hasHydrated) {
    return Promise.resolve()
  }

  return new Promise<void>((resolve) => {
    const unsubscribe = useAuthStore.subscribe((state) => {
      if (!state.auth.hasHydrated) {
        return
      }

      unsubscribe()
      resolve()
    })
  })
}
