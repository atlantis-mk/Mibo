import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/auth-store'
import { userSettingsQueryOptions } from '@/lib/mibo-query'
import { useTheme } from '@/context/theme-provider'

export function UserSettingsThemeSync() {
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryToken = accessToken || 'guest'
  const { theme, setTheme } = useTheme()

  const settingsQuery = useQuery({
    ...userSettingsQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
  })

  useEffect(() => {
    if (hasHydrated && !accessToken && theme !== 'system') {
      setTheme('system')
      return
    }

    const nextTheme = settingsQuery.data?.appearance.theme
    if (!nextTheme || nextTheme === theme) {
      return
    }

    setTheme(nextTheme)
  }, [
    accessToken,
    hasHydrated,
    setTheme,
    settingsQuery.data?.appearance.theme,
    theme,
  ])

  return null
}
