import { Outlet, createFileRoute } from '@tanstack/react-router'

import { AppSidebar } from '#/components/app-sidebar'
import { SidebarProvider } from '#/components/ui/sidebar'
import { requireCanEnterApp } from '#/lib/setup-gate'

export const Route = createFileRoute('/_app')({
  beforeLoad: async () => {
    await requireCanEnterApp()
  },
  component: AppLayout,
})

function AppLayout() {
  return (
    <SidebarProvider defaultOpen={false}>
      <AppSidebar variant="floating" className="z-40" />
      <div className="relative flex min-w-0 flex-1">
        <Outlet />
      </div>
    </SidebarProvider>
  )
}
