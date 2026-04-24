import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type {User} from '#/lib/mibo-api';

type AuthState = {
  token: string | null
  user: User | null
  hasHydrated: boolean
  setSession: (session: { token: string; user: User }) => void
  clearSession: () => void
  setHydrated: (hydrated: boolean) => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      hasHydrated: false,
      setSession: ({ token, user }) => set({ token, user }),
      clearSession: () => set({ token: null, user: null }),
      setHydrated: (hasHydrated) => set({ hasHydrated }),
    }),
    {
      name: 'mibo-auth',
      partialize: (state) => ({
        token: state.token,
        user: state.user,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHydrated(true)
      },
    },
  ),
)
