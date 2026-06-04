import { createFileRoute, redirect } from '@tanstack/react-router'
import { useAuthStore } from '@/stores/auth-store'
import {
  getVisibleSettingsSections,
  isAdminUser,
} from '@/features/settings/sections'

export const Route = createFileRoute('/_authenticated/settings/')({
  beforeLoad: () => {
    const authUser = useAuthStore.getState().auth.user
    const fallbackSection = getVisibleSettingsSections(authUser)[0]
    const destination = isAdminUser(authUser)
      ? '/settings/console'
      : (fallbackSection?.to ?? '/settings/profile')

    throw redirect({ to: destination })
  },
})
