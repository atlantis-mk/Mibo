import * as React from 'react'
import { Menu, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSidebar } from '@/components/ui/sidebar'

type SidebarEdgeTriggerProps = React.ComponentProps<'button'> & {
  side?: 'left' | 'right'
}

export function SidebarEdgeTrigger({
  className,
  onClick,
  side = 'left',
  ...props
}: SidebarEdgeTriggerProps) {
  const { isMobile, openMobile, state, toggleSidebar } = useSidebar()

  const handleToggle = React.useCallback(() => {
    toggleSidebar()
  }, [toggleSidebar])

  const handleKeyDown = React.useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (event.key !== 'Enter' && event.key !== ' ') return
      event.preventDefault()
      toggleSidebar()
    },
    [toggleSidebar]
  )

  return (
    <>
      {isMobile ? (
        <button
          type='button'
          data-sidebar='trigger'
          data-slot='sidebar-trigger'
          className={cn(
            'relative inline-flex size-10 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-transparent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none',
            className
          )}
          onClick={(event) => {
            onClick?.(event)
            handleToggle()
          }}
          {...props}
        >
          {openMobile ? <X className='size-5' /> : <Menu className='size-5' />}
          <span className='sr-only'>切换侧边栏</span>
        </button>
      ) : (
        <>
          <div
            role='button'
            tabIndex={0}
            aria-label='切换侧边栏'
            data-sidebar='trigger'
            data-slot='sidebar-trigger'
            data-side={side}
            data-state={state}
            className={cn('n-layout-toggle-bar', className)}
            onKeyDown={handleKeyDown}
            onClick={(event) => {
              onClick?.(event as unknown as React.MouseEvent<HTMLButtonElement>)
              handleToggle()
            }}
          >
            <div className='n-layout-toggle-bar__top' />
            <div className='n-layout-toggle-bar__bottom' />
          </div>
          <span className='sr-only'>切换侧边栏</span>
        </>
      )}
    </>
  )
}
