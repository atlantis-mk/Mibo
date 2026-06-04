import { LoaderCircleIcon, MoreVerticalIcon, Trash2Icon } from 'lucide-react'
import {
  formatProbeStatus,
  formatSourceContentClass,
} from '@/lib/library-presentation'
import type { Library, MediaSource, OperationsTask } from '@/lib/mibo-api'
import {
  operationTaskMessage,
  operationTaskTitle,
  operationsSeverityClassName,
} from '@/lib/operations-presentation'
import { cn } from '@/lib/utils'
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
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { EmptyCard } from './settings-aside-card'

export function LibrariesTab({
  libraries,
  mediaSources,
  operationsTasks,
  isLoading,
  isScanning,
  onEdit,
  onScan,
  onDelete,
}: {
  libraries: Library[]
  mediaSources: MediaSource[]
  operationsTasks: OperationsTask[]
  isLoading: boolean
  isScanning: boolean
  onEdit: (library: Library) => void
  onScan: (libraryId: number, mode: 'full' | 'changed') => void
  onDelete: (library: Library) => void
}) {
  return (
    <div className='grid gap-4 xl:grid-cols-2 2xl:grid-cols-3'>
      {libraries.map((library) => {
        const sourceName =
          mediaSources.find((source) => source.id === library.media_source_id)
            ?.name ?? `媒体源 #${library.media_source_id}`
        const enabledPaths = library.paths?.length
          ? library.paths.filter((path) => path.enabled)
          : []
        const rootPaths = enabledPaths.length
          ? enabledPaths.map((path) => path.root_path)
          : library.root_path
            ? [library.root_path]
            : ['未配置']
        const pathCount = enabledPaths.length || (library.root_path ? 1 : 0)
        const probeStatus = formatProbeStatus(
          library.probe_summary?.status || library.probe_status
        )
        const sampleText = library.probe_summary
          ? `${library.probe_summary.sampled_files} 文件 / ${library.probe_summary.sampled_dirs} 目录${library.probe_summary.budget_limited ? ' · 已达预算' : ''}`
          : '暂无样本'
        const contentText = library.collections?.length
          ? library.collections
              .map((collection) => `${collection.label} ${collection.count}`)
              .join(' / ')
          : '暂无分布'
        const metadataProfile =
          library.policies?.metadata?.metadata_profile_name || '迁移默认配置'
        const accessText = library.access_tags?.length
          ? library.access_tags.map((tag) => tag.name).join(' / ')
          : library.visibility_mode === 'allow_list_only'
            ? '未配置标签'
            : '默认开放'
        const visibilityText =
          library.visibility_mode === 'allow_list_only'
            ? '仅 allow 角色'
            : '默认开放'
        const issue = operationsTasks.find(
          (entry) =>
            (entry.lifecycle_status ?? 'active') !== 'resolved' &&
            entry.affected.libraries?.some((ref) => ref.id === library.id)
        )

        return (
          <Card
            key={library.id}
            className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'
          >
            <CardHeader className='gap-3 px-4 py-3'>
              <div className='flex items-start justify-between gap-3'>
                <div className='min-w-0'>
                  <CardTitle className='truncate text-base'>
                    {library.name}
                  </CardTitle>
                  <CardDescription className='mt-1 truncate'>
                    {sourceName} · {pathCount || 0} 个路径
                  </CardDescription>
                </div>
                <div className='flex shrink-0 items-center gap-2'>
                  <Badge
                    className='border-border/60 bg-muted text-foreground'
                    variant='outline'
                  >
                    {formatSourceContentClass(
                      library.probe_summary?.dominant_class
                    )}
                  </Badge>
                  <Button
                    variant='outline'
                    size='sm'
                    className='border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground'
                    onClick={() => onEdit(library)}
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
                        aria-label={`${library.name} 更多操作`}
                      >
                        <MoreVerticalIcon className='size-4' />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align='end' className='w-36'>
                      <DropdownMenuItem
                        onSelect={() => onScan(library.id, 'changed')}
                      >
                        {isScanning ? (
                          <LoaderCircleIcon className='size-4 animate-spin' />
                        ) : null}
                        扫描变化
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        onSelect={() => onScan(library.id, 'full')}
                      >
                        全量扫描
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        variant='destructive'
                        onSelect={() => onDelete(library)}
                      >
                        <Trash2Icon className='size-4' />
                        删除
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </CardHeader>
            <CardContent className='flex-1 space-y-3 px-4 pb-4 text-sm text-muted-foreground'>
              <div className='space-y-2'>
                <div className='flex flex-wrap items-center gap-2'>
                  <StatusBadge label={library.status} />
                  <StatusBadge label={probeStatus} />
                  <StatusBadge
                    label={library.scanner_enabled ? '扫描开启' : '扫描关闭'}
                    muted={!library.scanner_enabled}
                  />
                  <StatusBadge label={visibilityText} />
                </div>
                <div className='rounded-lg border border-border/60 bg-muted/15 px-3 py-2'>
                  <div className='truncate font-mono text-xs text-foreground'>
                    {rootPaths[0]}
                  </div>
                  {rootPaths.length > 1 ? (
                    <div className='mt-1 text-xs text-muted-foreground'>
                      另有 {rootPaths.length - 1} 个路径
                    </div>
                  ) : null}
                </div>
              </div>
              <div className='grid gap-2 sm:grid-cols-2'>
                <SummaryItem label='样本' value={sampleText} />
                <SummaryItem label='内容' value={contentText} />
                <SummaryItem label='元数据' value={metadataProfile} />
                <SummaryItem label='访问' value={accessText} />
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
              {!library.access_tags?.length ? (
                <div className='rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-900'>
                  {library.visibility_mode === 'allow_list_only'
                    ? '仅 allow 角色可见，但还没有访问标签。'
                    : '未设置访问标签，默认对所有已登录用户可见。'}
                </div>
              ) : null}
            </CardContent>
          </Card>
        )
      })}

      {!libraries.length && !isLoading ? (
        <EmptyCard text='还没有内容来源，点击上方按钮添加一个目录。' />
      ) : null}
    </div>
  )
}

function StatusBadge({ label, muted }: { label: string; muted?: boolean }) {
  return (
    <span
      className={cn(
        'inline-flex h-6 items-center rounded-md border border-border/60 px-2 text-xs text-foreground',
        muted ? 'bg-muted/20 text-muted-foreground' : 'bg-muted/35'
      )}
    >
      {label}
    </span>
  )
}

function SummaryItem({ label, value }: { label: string; value: string }) {
  return (
    <div className='min-w-0 rounded-lg border border-border/50 bg-muted/10 px-3 py-2'>
      <div className='text-[11px] text-muted-foreground'>{label}</div>
      <div className='mt-1 truncate text-sm text-foreground'>{value}</div>
    </div>
  )
}
