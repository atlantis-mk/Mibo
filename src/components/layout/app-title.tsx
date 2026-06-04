import { Link } from '@tanstack/react-router'
import { APP_NAME, APP_TAGLINE } from '@/config/app-shell'
import { Menu, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { Button } from '../ui/button'

export function AppTitle() {
  const { setOpenMobile } = useSidebar()
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <div className='flex items-center gap-2'>
          <SidebarMenuButton
            size='lg'
            className='min-w-0 flex-1 py-0 hover:bg-transparent active:bg-transparent'
            asChild
          >
            <Link
              to='/'
              onClick={() => setOpenMobile(false)}
              className='grid flex-1 text-start text-sm leading-tight'
            >
              <span className='truncate font-bold'>{APP_NAME}</span>
              <span className='truncate text-xs'>{APP_TAGLINE}</span>
            </Link>
          </SidebarMenuButton>
          <ToggleSidebar />
        </div>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

function ToggleSidebar({
  className,
  onClick,
  ...props
}: React.ComponentProps<typeof Button>) {
  const { toggleSidebar } = useSidebar()

  return (
    <Button
      data-sidebar='trigger'
      data-slot='sidebar-trigger'
      variant='ghost'
      size='icon'
      className={cn('aspect-square size-8 max-md:scale-125', className)}
      onClick={(event) => {
        onClick?.(event)
        toggleSidebar()
      }}
      {...props}
    >
      <X className='md:hidden' />
      <Menu className='max-md:hidden' />
      <span className='sr-only'>切换侧边栏</span>
    </Button>
  )
}
