"use client"

import { useEffect, useState, type ReactNode } from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'

import { canEnterApp, getStoredApiBaseUrl, SETUP_STATUS_EVENT } from '~/lib/client-config'
import { createMiboApi } from '~/lib/mibo-api'

export function SetupGate(props: { children: ReactNode }) {
  const navigate = useNavigate()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [isCheckingSetup, setIsCheckingSetup] = useState(pathname !== '/setup')
  const [isInitialized, setIsInitialized] = useState<boolean | null>(null)

  useEffect(() => {
    let cancelled = false

    const checkSetupStatus = async () => {
      setIsCheckingSetup(pathname !== '/setup')

      try {
        const setupStatus = await createMiboApi({
          baseUrl: getStoredApiBaseUrl(),
        }).getSetupStatus()

        if (cancelled) {
          return
        }

        const nextCanEnterApp = canEnterApp(setupStatus)
        setIsInitialized(nextCanEnterApp)

        if (!nextCanEnterApp && pathname !== '/setup') {
          await navigate({ to: '/setup', replace: true })
        }
      } catch {
        if (!cancelled) {
          setIsInitialized(true)
        }
      } finally {
        if (!cancelled) {
          setIsCheckingSetup(false)
        }
      }
    }

    void checkSetupStatus()

    const handleSetupStatusChanged = () => {
      void checkSetupStatus()
    }

    window.addEventListener(SETUP_STATUS_EVENT, handleSetupStatusChanged)

    return () => {
      cancelled = true
      window.removeEventListener(SETUP_STATUS_EVENT, handleSetupStatusChanged)
    }
  }, [navigate, pathname])

  if (pathname !== '/setup' && isCheckingSetup && isInitialized === null) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-2xl border border-border/70 bg-card px-5 py-4 shadow-sm">
          <Loader2 className="size-4 animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">正在检查初始化状态...</span>
        </div>
      </div>
    )
  }

  return <>{props.children}</>
}
