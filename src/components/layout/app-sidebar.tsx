import { useLocation } from '@tanstack/react-router'
import { SlidersHorizontal } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import { useLayout } from '@/context/layout-provider'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/components/ui/sidebar'
import { ConfigDrawer } from '@/components/config-drawer'
import {
  getSettingsNavGroups,
  sidebarData,
} from './data/sidebar-data'
import { NavGroup } from './nav-group'
import { NavUser } from './nav-user'
import { SidebarEdgeTrigger } from './sidebar-edge-trigger'
import { TeamSwitcher } from './team-switcher'

export function AppSidebar() {
  const { collapsible, variant } = useLayout()
  const pathname = useLocation({ select: (location) => location.pathname })
  const authUser = useAuthStore((state) => state.auth.user)
  const navGroups = pathname.startsWith('/settings')
    ? getSettingsNavGroups(authUser)
    : sidebarData.navGroups
  const sidebarUser = {
    name: authUser?.username ?? sidebarData.user.name,
    email: authUser?.role ? formatRole(authUser.role) : sidebarData.user.email,
    avatar: sidebarData.user.avatar,
  }

  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      <SidebarHeader>
        <TeamSwitcher teams={sidebarData.teams} />
      </SidebarHeader>
      <SidebarContent>
        {navGroups.map((props) => (
          <NavGroup key={props.title} {...props} />
        ))}
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu className='hidden md:flex'>
          <SidebarMenuItem>
            <ConfigDrawer
              trigger={
                <SidebarMenuButton
                  size='lg'
                  tooltip='外观与布局'
                  className='h-auto min-h-12 rounded-xl border border-sidebar-border/70 bg-sidebar-accent/35 px-3 py-2.5 shadow-sm group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:min-h-8 group-data-[collapsible=icon]:rounded-md group-data-[collapsible=icon]:border-transparent group-data-[collapsible=icon]:bg-transparent group-data-[collapsible=icon]:px-0 group-data-[collapsible=icon]:py-0 group-data-[collapsible=icon]:shadow-none'
                >
                  <div className='flex size-8 items-center justify-center rounded-lg border border-sidebar-border/70 bg-background/80 text-sidebar-foreground group-data-[collapsible=icon]:size-full group-data-[collapsible=icon]:rounded-md group-data-[collapsible=icon]:border-0 group-data-[collapsible=icon]:bg-transparent'>
                    <SlidersHorizontal className='size-4' />
                  </div>
                  <div className='grid flex-1 text-start text-sm leading-tight group-data-[collapsible=icon]:hidden'>
                    <span className='truncate font-semibold'>外观与布局</span>
                    <span className='truncate text-xs text-sidebar-foreground/70'>
                      主题、侧边栏与界面偏好
                    </span>
                  </div>
                </SidebarMenuButton>
              }
            />
          </SidebarMenuItem>
        </SidebarMenu>
        <NavUser user={sidebarUser} />
      </SidebarFooter>
      <SidebarEdgeTrigger className='hidden md:block' />
      <SidebarRail />
    </Sidebar>
  )
}

function formatRole(role: string) {
  if (role === 'admin') return '管理员'
  if (role === 'user') return '普通用户'
  return role
}
