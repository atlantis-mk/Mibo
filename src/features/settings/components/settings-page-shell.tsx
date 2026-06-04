import type { ComponentType, ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function SettingsPageShell({
  icon: Icon,
  title,
  description,
  actions,
  fixedContent,
  showHeader = true,
  children,
}: {
  icon: ComponentType<{ className?: string }>
  title: string
  description: string
  actions?: ReactNode
  fixedContent?: boolean
  showHeader?: boolean
  children: ReactNode
}) {
  return (
    <div
      data-layout={fixedContent ? 'fixed' : undefined}
      className={cn(
        'flex min-h-0 flex-1 flex-col overflow-y-auto',
        fixedContent && 'h-full overflow-hidden'
      )}
    >
      <div
        className={cn(
          'flex min-h-full flex-1 flex-col gap-4 px-4 py-6 sm:px-6 lg:px-8 xl:px-10',
          fixedContent && 'h-full'
        )}
      >
        {showHeader ? (
          <section className='rounded-[1.5rem] border border-border/60 bg-card/80 px-5 py-4 shadow-sm backdrop-blur-sm'>
            <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
              <div className='flex items-start gap-3'>
                <div className='flex size-10 shrink-0 items-center justify-center rounded-xl border border-border/60 bg-background/70'>
                  <Icon className='size-4 text-muted-foreground' />
                </div>
                <div className='min-w-0'>
                  <h2 className='text-xl font-semibold tracking-tight'>
                    {title}
                  </h2>
                  <p className='mt-1 text-sm leading-6 text-muted-foreground'>
                    {description}
                  </p>
                </div>
              </div>

              <div className={cn(actions ? 'sm:shrink-0' : 'hidden')}>
                {actions}
              </div>
            </div>
          </section>
        ) : null}

        <div
          className={cn(
            'flex min-h-0 flex-1 flex-col',
            fixedContent && 'overflow-hidden'
          )}
        >
          {children}
        </div>
      </div>
    </div>
  )
}
