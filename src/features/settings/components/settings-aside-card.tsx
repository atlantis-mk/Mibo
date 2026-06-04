import type { ReactNode } from 'react'
import { Card, CardContent } from '@/components/ui/card'

export function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className='rounded-[1rem] border border-border/60 bg-muted/30 px-3.5 py-3'>
      <div className='text-xs tracking-[0.16em] text-muted-foreground uppercase'>
        {label}
      </div>
      <div className='mt-2 text-sm break-all text-foreground'>{value}</div>
    </div>
  )
}

export function DescriptionGrid({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <div className='overflow-hidden rounded-[1rem] border border-border/60 bg-muted/10'>
      <div
        className={`grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 ${className ?? ''}`}
      >
        {children}
      </div>
    </div>
  )
}

export function DescriptionItem({
  label,
  value,
  lines,
  className,
  compact,
}: {
  label: string
  value?: string
  lines?: string[]
  className?: string
  compact?: boolean
}) {
  const items = lines?.filter(Boolean) ?? (value ? [value] : [])

  return (
    <div
      className={`border-border/60 px-5 py-4 sm:[&:not(:nth-child(2n))]:border-r xl:[&:not(:nth-child(3n))]:border-r [&:not(:nth-last-child(-n+1))]:border-b sm:[&:nth-last-child(-n+2)]:border-b-0 xl:[&:nth-last-child(-n+3)]:border-b-0 ${compact ? 'px-3 py-3' : ''} ${className ?? ''}`}
    >
      <div
        className={`text-xs font-medium tracking-[0.08em] text-muted-foreground ${compact ? 'text-[11px]' : ''}`}
      >
        {label}
      </div>
      <div className={`mt-3 space-y-1.5 ${compact ? 'mt-2 space-y-1' : ''}`}>
        {items.map((item, index) => (
          <p
            key={`${label}-${index}`}
            className={`text-sm leading-7 break-all text-foreground ${compact ? 'leading-5' : ''}`}
          >
            {item}
          </p>
        ))}
      </div>
    </div>
  )
}

export function DescriptionSpacer({ className }: { className?: string }) {
  return (
    <div
      aria-hidden='true'
      className={`border-b border-border/60 sm:border-r xl:border-r ${className ?? ''}`}
    />
  )
}

export function EmptyCard({ text }: { text: string }) {
  return (
    <Card className='rounded-[1.5rem] border border-dashed border-border/60 bg-card/60 py-0'>
      <CardContent className='px-5 py-8 text-sm leading-7 text-muted-foreground'>
        {text}
      </CardContent>
    </Card>
  )
}
