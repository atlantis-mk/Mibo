import { Outlet } from '@tanstack/react-router'
import { LoaderCircleIcon } from 'lucide-react'
import { getCookie } from '@/lib/cookies'
import { useProtectedSessionRedirect } from '@/lib/use-protected-session-redirect'
import { cn } from '@/lib/utils'
import { LayoutProvider } from '@/context/layout-provider'
import { SearchProvider } from '@/context/search-provider'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { AppSidebar } from '@/components/layout/app-sidebar'
import { MobileShellControls } from '@/components/layout/mobile-shell-controls'
import { PINChangeRequiredDialog } from '@/components/pin-change-required-dialog'
import { UserSettingsThemeSync } from '@/components/user-settings-theme-sync'

type AuthenticatedLayoutProps = {
  children?: React.ReactNode
}

export function AuthenticatedLayout({ children }: AuthenticatedLayoutProps) {
  const defaultOpen = getCookie('sidebar_state') !== 'false'
  const isCheckingSession = useProtectedSessionRedirect()

  if (isCheckingSession) {
    return (
      <div className='flex h-svh w-full items-center justify-center bg-background text-foreground'>
        <div className='flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl'>
          <LoaderCircleIcon className='size-4 animate-spin' />
          <span className='text-sm text-muted-foreground'>
            正在验证访问权限
          </span>
        </div>
      </div>
    )
  }

  return (
    <SearchProvider>
      <LayoutProvider>
        <SidebarProvider defaultOpen={defaultOpen}>
          <UserSettingsThemeSync />
          <PINChangeRequiredDialog />
          <AppSidebar />
          <SidebarInset
            className={cn(
              // Set content container, so we can use container queries
              '@container/content',

              // If layout is fixed, set the height
              // to 100svh to prevent overflow
              'has-data-[layout=fixed]:h-svh',

              // If layout is fixed and sidebar is inset,
              // set the height to 100svh - spacing (total margins) to prevent overflow
              'peer-data-[variant=inset]:has-data-[layout=fixed]:h-[calc(100svh-(var(--spacing)*4))]'
            )}
          >
            <MobileShellControls />
            {children ?? <Outlet />}
          </SidebarInset>
        </SidebarProvider>
      </LayoutProvider>
    </SearchProvider>
  )
}
