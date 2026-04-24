import { TabsTrigger } from '#/components/ui/tabs'

export function SettingsMenuTrigger({
  value,
  icon: Icon,
  title,
  description,
}: {
  value: string
  icon: React.ComponentType<{ className?: string }>
  title: string
  description: string
}) {
  return (
    <TabsTrigger
      value={value}
      className="rounded-[1rem] border border-transparent px-2.5 py-2.5 text-left data-active:border-border data-active:bg-muted"
    >
      <div className="flex items-start gap-3">
        <div className="mt-0.5 flex size-8 items-center justify-center rounded-lg border border-border/60 bg-muted/50">
          <Icon className="size-4 text-muted-foreground" />
        </div>
        <div className="min-w-0">
          <div className="text-sm font-medium text-foreground">{title}</div>
          <div className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">
            {description}
          </div>
        </div>
      </div>
    </TabsTrigger>
  )
}
