import { MoreVerticalIcon, Trash2Icon } from 'lucide-react'
import type {
  MediaSource,
  OperationsTask,
  PluginProviderInstance,
} from '@/lib/mibo-api'
import {
  operationTaskMessage,
  operationTaskTitle,
  operationsSeverityClassName,
} from '@/lib/operations-presentation'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { EmptyCard } from './settings-aside-card'

export function MediaSourcesTab({
  mediaSources,
  operationsTasks,
  isLoading,
  onEdit,
  onDelete,
}: {
  mediaSources: MediaSource[]
  pluginProviderInstances: PluginProviderInstance[]
  operationsTasks: OperationsTask[]
  isLoading: boolean
  onEdit: (source: MediaSource) => void
  onDelete: (source: MediaSource) => void
}) {
  return (
    <div className='grid gap-4 xl:grid-cols-2 2xl:grid-cols-3'>
      {mediaSources.map((source) => {
        const issue = operationsTasks.find((entry) =>
          entry.affected.media_sources?.some((ref) => ref.id === source.id)
        )
        const providerLabel = source.provider_label || source.provider
        const updatedAt = formatDateTime(source.updated_at)
        const connectionLabel =
          source.provider === 'openlist'
            ? 'OpenList 地址'
            : source.plugin_provider
              ? '插件端点'
              : '本地目录'
        const connectionValue =
          source.provider === 'openlist'
            ? (source.config?.openlist?.base_url ?? '未配置')
            : source.plugin_provider
              ? source.plugin_provider.endpoint
              : source.root_path

        return (
          <Card
            key={source.id}
            className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'
          >
            <CardHeader className='gap-3 px-4 py-3'>
              <div className='flex items-start justify-between gap-3'>
                <div className='min-w-0'>
                  <CardTitle className='truncate text-base'>
                    {source.name}
                  </CardTitle>
                  <CardDescription className='mt-1 truncate'>
                    #{source.id} · 最近更新 {updatedAt}
                  </CardDescription>
                </div>
                <div className='flex shrink-0 items-center gap-2'>
                  <Badge
                    className='border-border/60 bg-muted text-foreground'
                    variant='outline'
                  >
                    {providerLabel}
                  </Badge>
                  <Button
                    variant='outline'
                    size='sm'
                    className='border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground'
                    onClick={() => onEdit(source)}
                  >
                    编辑
                  </Button>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        type='button'
                        variant='outline'
                        size='icon'
                        className='size-8 border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground'
                        aria-label={`${source.name} 更多操作`}
                      >
                        <MoreVerticalIcon className='size-4' />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align='end' className='w-32'>
                      <DropdownMenuItem
                        variant='destructive'
                        onSelect={() => onDelete(source)}
                      >
                        <Trash2Icon className='size-4' />
                        删除
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </CardHeader>
            <CardContent className='space-y-3 px-4 pb-4 text-sm text-muted-foreground'>
              <div className='grid gap-2'>
                <SourceSummaryLine label='根路径' value={source.root_path} />
                <SourceSummaryLine
                  label={connectionLabel}
                  value={connectionValue}
                />
              </div>
              <div className='flex flex-wrap gap-2'>
                <SourceMeta label='类型' value={providerLabel} />
                <SourceMeta label='标识' value={`#${source.id}`} />
                <SourceMeta label='更新' value={updatedAt} />
              </div>
              {issue ? (
                <div className='rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm'>
                  <Badge
                    className={operationsSeverityClassName(issue.severity)}
                    variant='outline'
                  >
                    {operationTaskTitle(issue)}
                  </Badge>
                  <div className='mt-2 line-clamp-2 leading-6 text-foreground'>
                    {operationTaskMessage(issue)}
                  </div>
                </div>
              ) : null}
            </CardContent>
          </Card>
        )
      })}

      {!mediaSources.length && !isLoading ? (
        <EmptyCard text='还没有媒体源，点击上方按钮创建一个本地目录或 OpenList 来源。' />
      ) : null}
    </div>
  )
}

function SourceSummaryLine({ label, value }: { label: string; value: string }) {
  return (
    <div className='rounded-lg border border-border/60 bg-muted/15 px-3 py-2'>
      <div className='text-[11px] text-muted-foreground'>{label}</div>
      <div className='mt-1 truncate font-mono text-xs text-foreground'>
        {value}
      </div>
    </div>
  )
}

function SourceMeta({ label, value }: { label: string; value: string }) {
  return (
    <span className='inline-flex h-6 min-w-0 items-center rounded-md border border-border/60 bg-muted/20 px-2 text-xs text-muted-foreground'>
      <span className='text-muted-foreground/80'>{label}</span>
      <span className='mx-1 text-border'>/</span>
      <span className='truncate text-foreground'>{value}</span>
    </span>
  )
}

function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}
