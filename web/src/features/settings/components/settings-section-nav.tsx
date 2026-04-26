import { Link, useRouterState } from '@tanstack/react-router'

import { cn } from '#/lib/utils'
import type { SettingsSection } from '../sections'

export function SettingsSectionNav({
  sections,
}: {
  sections: SettingsSection[]
}) {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  return (
    <nav
      aria-label="设置分类"
      className="rounded-[1.5rem] border border-border/60 bg-card/80 p-2 shadow-sm backdrop-blur-sm"
    >
      <div className="px-3 pb-2 pt-2">
        <div className="text-sm font-medium text-foreground">设置中心</div>
        <div className="mt-1 text-xs leading-5 text-muted-foreground">
          每个分类都是独立页面，右侧只展示当前分类内容。
        </div>
      </div>

      <div className="grid gap-1 sm:grid-cols-2 lg:grid-cols-1">
        {sections.map((section) => {
          const Icon = section.icon
          const active =
            pathname === section.to ||
            pathname.startsWith(`${section.matchPrefix}/`)

          return (
            <Link
              key={section.key}
              to={section.to}
              aria-current={active ? 'page' : undefined}
              className={cn(
                'group flex w-full items-start gap-3 rounded-[1.1rem] border px-3 py-3 text-left transition-colors',
                active
                  ? 'border-border bg-muted text-foreground shadow-sm'
                  : 'border-transparent text-muted-foreground hover:border-border/60 hover:bg-muted/50 hover:text-foreground',
              )}
            >
              <div
                className={cn(
                  'mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-xl border transition-colors',
                  active
                    ? 'border-border bg-background text-foreground'
                    : 'border-border/50 bg-background/60 text-muted-foreground group-hover:text-foreground',
                )}
              >
                <Icon className="size-4" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <div className="truncate text-sm font-medium">
                    {section.title}
                  </div>
                  <div className="shrink-0 rounded-full border border-border/60 bg-background/70 px-2 py-0.5 text-[11px] text-muted-foreground">
                    {section.status}
                  </div>
                </div>
                <div className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">
                  {section.description}
                </div>
              </div>
            </Link>
          )
        })}
      </div>
    </nav>
  )
}
