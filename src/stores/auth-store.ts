import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { type User } from '@/lib/mibo-api'

interface AuthState {
  auth: {
    user: User | null
    accessToken: string
    hasHydrated: boolean
    setSession: (session: { token: string; user: User }) => void
    setUser: (user: User | null) => void
    setAccessToken: (accessToken: string) => void
    resetAccessToken: () => void
    clearSession: () => void
    setHydrated: (hydrated: boolean) => void
    reset: () => void
  }
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      auth: {
        user: null,
        accessToken: '',
        hasHydrated: false,
        setSession: ({ token, user }) =>
          set((state) => ({
            auth: { ...state.auth, accessToken: token, user },
          })),
        setUser: (user) => set((state) => ({ auth: { ...state.auth, user } })),
        setAccessToken: (accessToken) =>
          set((state) => ({ auth: { ...state.auth, accessToken } })),
        resetAccessToken: () =>
          set((state) => ({ auth: { ...state.auth, accessToken: '' } })),
        clearSession: () =>
          set((state) => ({
            auth: { ...state.auth, accessToken: '', user: null },
          })),
        setHydrated: (hasHydrated) =>
          set((state) => ({ auth: { ...state.auth, hasHydrated } })),
        reset: () =>
          set((state) => ({
            auth: { ...state.auth, user: null, accessToken: '' },
          })),
      },
    }),
    {
      name: 'mibo-auth',
      storage: createJSONStorage(() => localStorage),
      merge: (persistedState, currentState) => {
        const persisted = persistedState as Partial<AuthState> | undefined

        return {
          ...currentState,
          auth: {
            ...currentState.auth,
            ...persisted?.auth,
          },
        }
      },
      partialize: (state) => ({
        auth: {
          accessToken: state.auth.accessToken,
          user: state.auth.user,
        },
      }),
      onRehydrateStorage: () => (state) => {
        state?.auth.setHydrated(true)
      },
    }
  )
)
