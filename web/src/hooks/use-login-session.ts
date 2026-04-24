import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { ApiError, createMiboApi, getApiBaseUrl } from '#/lib/mibo-api'
import { authUserQueryOptions, miboQueryKeys } from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

export function useLoginSession() {
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('admin123')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const setSession = useAuthStore((state) => state.setSession)
  const clearSession = useAuthStore((state) => state.clearSession)

  const apiBaseUrl = getApiBaseUrl()
  const queryClient = useQueryClient()
  const queryToken = token ?? 'guest'

  const sessionQuery = useQuery({
    ...authUserQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
    retry: false,
  })

  useEffect(() => {
    if (!hasHydrated || !token) {
      return
    }

    if (sessionQuery.data) {
      setSession({ token, user: sessionQuery.data })
      return
    }

    if (sessionQuery.error) {
      clearSession()
      void queryClient.removeQueries({
        queryKey: miboQueryKeys.authUser(token),
      })
    }
  }, [
    clearSession,
    hasHydrated,
    queryClient,
    sessionQuery.data,
    sessionQuery.error,
    setSession,
    token,
  ])

  const loginMutation = useMutation({
    mutationFn: () =>
      createMiboApi({ baseUrl: apiBaseUrl }).login(username, password),
    onMutate: () => {
      setErrorMessage(null)
      setIsSubmitting(true)
    },
    onSuccess: (session) => {
      setSession({ token: session.token, user: session.user })
      void queryClient.setQueryData(
        miboQueryKeys.authUser(session.token),
        session.user,
      )
    },
    onError: (error) => {
      setErrorMessage(
        error instanceof ApiError ? error.message : '登录失败，请稍后重试。',
      )
    },
    onSettled: () => {
      setIsSubmitting(false)
    },
  })

  const logoutMutation = useMutation({
    mutationFn: async () => {
      if (token) {
        await createMiboApi({ baseUrl: apiBaseUrl, token }).logout()
      }
    },
    onMutate: () => {
      setErrorMessage(null)
      setIsSubmitting(true)
    },
    onSettled: async () => {
      clearSession()
      await queryClient.invalidateQueries({
        queryKey: ['home'],
      })
      await queryClient.removeQueries({
        queryKey: ['auth'],
      })
      setIsSubmitting(false)
    },
  })

  function login() {
    loginMutation.mutate()
  }

  function logout() {
    logoutMutation.mutate()
  }

  return {
    username,
    setUsername,
    password,
    setPassword,
    errorMessage,
    isSubmitting,
    hasHydrated,
    user: user ?? sessionQuery.data ?? null,
    login,
    logout,
  }
}
