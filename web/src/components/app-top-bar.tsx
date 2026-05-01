import { useEffect, useRef, useState } from 'react'
import type { ReactNode, RefObject } from 'react'

import { cn } from '#/lib/utils'

type AppTopBarProps = {
  leftSlot?: ReactNode
  rightSlot?: ReactNode
  className?: string
  contentClassName?: string
  scrollContainerRef?: RefObject<HTMLElement | null>
}

const SCROLL_THRESHOLD = 8
const TOP_VISIBLE_OFFSET = 24

export function AppTopBar({
  leftSlot,
  rightSlot,
  className,
  contentClassName,
  scrollContainerRef,
}: AppTopBarProps) {
  const [isVisible, setIsVisible] = useState(true)
  const lastScrollOffsetRef = useRef(0)

  useEffect(() => {
    const scrollContainer = scrollContainerRef?.current
    const scrollTarget = scrollContainer ?? window
    const getScrollOffset = () =>
      scrollContainer ? scrollContainer.scrollTop : window.scrollY

    lastScrollOffsetRef.current = getScrollOffset()

    const handleScroll = () => {
      const currentScrollOffset = getScrollOffset()
      const scrollDelta = currentScrollOffset - lastScrollOffsetRef.current

      if (currentScrollOffset <= TOP_VISIBLE_OFFSET) {
        setIsVisible(true)
      } else if (scrollDelta > SCROLL_THRESHOLD) {
        setIsVisible(false)
      } else if (scrollDelta < -SCROLL_THRESHOLD) {
        setIsVisible(true)
      }

      lastScrollOffsetRef.current = currentScrollOffset
    }

    scrollTarget.addEventListener('scroll', handleScroll, { passive: true })

    return () => {
      scrollTarget.removeEventListener('scroll', handleScroll)
    }
  }, [scrollContainerRef])

  return (
    <div className={cn('pointer-events-none sticky top-0 z-30 h-0', className)}>
      <div
        className={cn(
          'pointer-events-auto relative px-4 pt-4 transition-transform duration-300 sm:px-6 sm:pt-6',
          isVisible ? 'translate-y-0' : '-translate-y-full',
        )}
      >
        <div
          className={cn(
            'mx-auto flex max-w-[calc(100%-1rem)] items-center justify-between gap-3 rounded-full border border-white/20 bg-background/55 px-3 py-2 text-foreground shadow-[0_18px_50px_rgb(0_0_0/0.28),inset_0_1px_0_rgb(255_255_255/0.22)] ring-1 ring-border/40 backdrop-blur-2xl backdrop-saturate-200 before:pointer-events-none before:absolute before:inset-px before:rounded-full before:bg-gradient-to-b before:from-white/18 before:to-transparent before:content-[""] dark:border-white/10 dark:bg-background/42 dark:shadow-[0_18px_60px_rgb(0_0_0/0.42),inset_0_1px_0_rgb(255_255_255/0.1)] sm:px-4',
            contentClassName,
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
