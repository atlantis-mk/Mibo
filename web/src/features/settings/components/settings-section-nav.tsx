import { Link, useRouterState } from "@tanstack/react-router"

import { ScrollArea } from "#/components/ui/scroll-area"
import { cn } from "#/lib/utils"
import type { SettingsSection } from "../sections"

type SettingsSectionGroup = {
  title: string
  sections: SettingsSection[]
}

export function SettingsSectionNav({
  sections,
}: {
  sections: SettingsSection[]
}) {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const groups = sections.reduce<SettingsSectionGroup[]>((result, section) => {
    const currentGroup = result.at(-1)
    if (currentGroup?.title === section.group) {
      currentGroup.sections.push(section)
      return result
    }

    result.push({ title: section.group, sections: [section] })
    return result
  }, [])

  return (
    <nav
      aria-label="设置分类"
      className="flex h-full flex-col rounded-[1.5rem] border border-border/60 bg-card/80 p-2 shadow-sm backdrop-blur-sm"
    >
      <div className="shrink-0 px-3 pt-2 pb-2">
        <div className="text-sm font-medium text-foreground">设置中心</div>
        <div className="mt-1 text-xs leading-5 text-muted-foreground">
          每个分类都是独立页面，右侧只展示当前分类内容。
        </div>
      </div>

      <ScrollArea className="min-h-0 flex-1 overflow-hidden">
        <div className="space-y-4">
          {groups.map((group) => (
            <div key={group.title}>
              <div className="px-3 pb-1 text-[11px] font-medium tracking-wide text-muted-foreground/75 uppercase">
                {group.title}
              </div>
              <div className="grid gap-1 sm:grid-cols-2 lg:grid-cols-1">
                {group.sections.map((section) => {
                  const Icon = section.icon
                  const active =
                    pathname === section.to ||
                    pathname.startsWith(`${section.matchPrefix}/`)

                  return (
                    <Link
                      key={section.key}
                      to={section.to}
                      aria-current={active ? "page" : undefined}
                      className={cn(
                        "group flex w-full items-start gap-3 rounded-[1.1rem] border px-3 py-3 text-left transition-colors",
                        active
                          ? "border-primary/35 bg-primary text-primary-foreground shadow-sm"
                          : "border-transparent text-muted-foreground hover:border-border/60 hover:bg-muted/50 hover:text-foreground"
                      )}
                    >
                      <div
                        className={cn(
                          "mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-xl border transition-colors",
                          active
                            ? "border-primary/20 bg-primary-foreground/15 text-primary-foreground"
                            : "border-border/50 bg-background/60 text-muted-foreground group-hover:text-foreground"
                        )}
                      >
                        <Icon className="size-4" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between gap-2">
                          <div className="truncate text-sm font-medium">
                            {section.title}
                          </div>
                          <div
                            className={cn(
                              "shrink-0 rounded-full border px-2 py-0.5 text-[11px]",
                              active
                                ? "border-primary-foreground/25 bg-primary-foreground/15 text-primary-foreground/85"
                                : "border-border/60 bg-background/70 text-muted-foreground"
                            )}
                          >
                            {section.status}
                          </div>
                        </div>
                        <div
                          className={cn(
                            "mt-1 line-clamp-2 text-xs leading-5",
                            active
                              ? "text-primary-foreground/80"
                              : "text-muted-foreground"
                          )}
                        >
                          {section.description}
                        </div>
                      </div>
                    </Link>
                  )
                })}
              </div>
            </div>
          ))}
        </div>
      </ScrollArea>
    </nav>
  )
}
