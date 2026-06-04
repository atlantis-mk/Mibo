import { createFileRoute, redirect } from '@tanstack/react-router'
import { useAuthStore } from '@/stores/auth-store'
import SettingsLayout from '@/features/settings'
import { canAccessSettingsPath } from '@/features/settings/sections'

export const Route = createFileRoute('/_authenticated/settings')({
  beforeLoad: ({ location }) => {
    const authUser = useAuthStore.getState().auth.user

    if (!canAccessSettingsPath(location.pathname, authUser)) {
      throw redirect({ to: '/403' })
    }
  },
  component: SettingsLayout,
})
