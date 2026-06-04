import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function SettingsPageInset({
  fixedContent,
  children,
}: {
  fixedContent?: boolean
  children: ReactNode
}) {
  return (
    <div
      className={cn(
        'min-h-0 flex-1',
        fixedContent ? 'overflow-hidden' : 'overflow-y-auto'
      )}
    >
      <div
        className={cn(
          'flex min-h-full flex-col px-4 py-6 sm:px-6 lg:px-8 xl:px-10',
          fixedContent && 'h-full overflow-hidden'
        )}
      >
        {children}
      </div>
    </div>
  )
}
