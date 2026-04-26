import type { ComponentType, ReactNode } from 'react'

import { cn } from '#/lib/utils'

export function SettingsPageShell({
  icon: Icon,
  title,
  description,
  actions,
  children,
}: {
  icon: ComponentType<{ className?: string }>
  title: string
  description: string
  actions?: ReactNode
  children: ReactNode
}) {
  return (
    <div className="space-y-4">
      <section className="rounded-[1.5rem] border border-border/60 bg-card/80 px-5 py-4 shadow-sm backdrop-blur-sm">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-border/60 bg-background/70">
              <Icon className="size-4 text-muted-foreground" />
            </div>
            <div className="min-w-0">
              <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
              <p className="mt-1 text-sm leading-6 text-muted-foreground">
                {description}
              </p>
            </div>
          </div>

          <div className={cn(actions ? 'sm:shrink-0' : 'hidden')}>
            {actions}
          </div>
        </div>
      </section>

      {children}
    </div>
  )
}
