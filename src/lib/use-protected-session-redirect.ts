import { useEffect, useRef, useState } from 'react'
import { useAuthStore } from '@/stores/auth-store'
import { createMiboApi, getApiBaseUrl } from '@/lib/mibo-api'

export function useProtectedSessionRedirect() {
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const [isChecking, setIsChecking] = useState(false)
  const lastCheckedTokenRef = useRef<string | null | undefined>(undefined)

  useEffect(() => {
    if (!hasHydrated) {
      return
    }

    if (lastCheckedTokenRef.current === accessToken) {
      setIsChecking(false)
      return
    }

    let cancelled = false
    setIsChecking(true)

    void createMiboApi({
      baseUrl: getApiBaseUrl(),
      token: accessToken,
    })
      .me()
      .then((authUser) => {
        if (cancelled) {
          return
        }

        if (accessToken) {
          useAuthStore
            .getState()
            .auth.setSession({ token: accessToken, user: authUser })
        }

        lastCheckedTokenRef.current = accessToken
        setIsChecking(false)
      })
      .catch(() => {
        if (cancelled) {
          return
        }

        lastCheckedTokenRef.current = accessToken
        setIsChecking(false)
      })

    return () => {
      cancelled = true
    }
  }, [accessToken, hasHydrated])

  return !hasHydrated || isChecking
}
