import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/auth-store'
import { ApiError, createMiboApi, getApiBaseUrl } from '@/lib/mibo-api'
import {
  authUserQueryOptions,
  loginUsersQueryOptions,
  miboQueryKeys,
} from '@/lib/mibo-query'

export function useLoginSession() {
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const user = useAuthStore((state) => state.auth.user)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const setSession = useAuthStore((state) => state.auth.setSession)
  const clearSession = useAuthStore((state) => state.auth.clearSession)

  const apiBaseUrl = getApiBaseUrl()
  const queryClient = useQueryClient()
  const queryToken = accessToken || 'guest'

  const sessionQuery = useQuery({
    ...authUserQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
    retry: false,
  })
  const loginUsersQuery = useQuery({
    ...loginUsersQueryOptions(),
    enabled: hasHydrated && !accessToken,
    retry: false,
  })

  useEffect(() => {
    if (!hasHydrated || !accessToken) {
      return
    }

    if (sessionQuery.data) {
      setSession({ token: accessToken, user: sessionQuery.data })
      return
    }

    if (
      sessionQuery.error instanceof ApiError &&
      sessionQuery.error.status === 401
    ) {
      clearSession()
      void queryClient.removeQueries({
        queryKey: miboQueryKeys.authUser(accessToken),
      })
    }
  }, [
    accessToken,
    clearSession,
    hasHydrated,
    queryClient,
    sessionQuery.data,
    sessionQuery.error,
    setSession,
  ])

  const loginMutation = useMutation({
    mutationFn: ({
      username,
      password,
    }: {
      username: string
      password: string
    }) => createMiboApi({ baseUrl: apiBaseUrl }).login(username, password),
    onMutate: () => {
      setErrorMessage(null)
    },
    onSuccess: (session) => {
      setSession({ token: session.token, user: session.user })
      queryClient.setQueryData(
        miboQueryKeys.authUser(session.token),
        session.user
      )
    },
    onError: (error) => {
      setErrorMessage(
        error instanceof ApiError ? error.message : '登录失败，请重试。'
      )
    },
  })

  const pinLoginMutation = useMutation({
    mutationFn: ({ userId, pin }: { userId: number; pin: string }) =>
      createMiboApi({ baseUrl: apiBaseUrl }).loginWithPin(userId, pin),
    onMutate: () => {
      setErrorMessage(null)
    },
    onSuccess: (session) => {
      setSession({ token: session.token, user: session.user })
      queryClient.setQueryData(
        miboQueryKeys.authUser(session.token),
        session.user
      )
    },
    onError: (error) => {
      setErrorMessage(
        error instanceof ApiError ? error.message : 'PIN 登录失败，请重试。'
      )
    },
  })

  async function login(username: string, password: string) {
    await loginMutation.mutateAsync({ username, password })
  }

  async function loginWithPin(userId: number, pin: string) {
    await pinLoginMutation.mutateAsync({ userId, pin })
  }

  return {
    errorMessage,
    hasHydrated,
    isSubmitting: loginMutation.isPending || pinLoginMutation.isPending,
    login,
    loginUsers: loginUsersQuery.data ?? [],
    loginUsersError: loginUsersQuery.error,
    loginUsersLoading: loginUsersQuery.isLoading,
    loginWithPin,
    user: user ?? sessionQuery.data ?? null,
  }
}
