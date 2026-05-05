import type { ReactNode, RefObject } from "react"

import { cn } from "#/lib/utils"

type AppTopBarProps = {
  leftSlot?: ReactNode
  rightSlot?: ReactNode
  className?: string
  contentClassName?: string
  scrollContainerRef?: RefObject<HTMLElement | null>
}

export function AppTopBar({
  leftSlot,
  rightSlot,
  className,
  contentClassName,
}: AppTopBarProps) {
  return (
    <div className={cn("pointer-events-none sticky top-0 z-30 h-0", className)}>
      <div
        className={cn(
          "pointer-events-auto relative translate-y-0 px-4 pt-4 sm:px-6 sm:pt-6"
        )}
      >
        <div
          className={cn(
            'mx-auto flex max-w-[calc(100%-1rem)] items-center justify-between gap-3 rounded-full border border-white/20 bg-background/55 px-3 py-2 text-foreground shadow-[0_18px_50px_rgb(0_0_0/0.28),inset_0_1px_0_rgb(255_255_255/0.22)] ring-1 ring-border/40 backdrop-blur-2xl backdrop-saturate-200 before:pointer-events-none before:absolute before:inset-px before:rounded-full before:bg-gradient-to-b before:from-white/18 before:to-transparent before:content-[""] sm:px-4 dark:border-white/10 dark:bg-background/42 dark:shadow-[0_18px_60px_rgb(0_0_0/0.42),inset_0_1px_0_rgb(255_255_255/0.1)]',
            contentClassName
          )}
        >
          <div className="flex min-w-0 items-center gap-3">{leftSlot}</div>
          {rightSlot ? (
            <div className="flex items-center gap-2">{rightSlot}</div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
