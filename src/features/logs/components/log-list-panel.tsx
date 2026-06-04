import type { ReactNode } from 'react'
import {
  DownloadIcon,
  EyeIcon,
  FileTextIcon,
  MoreHorizontalIcon,
  RefreshCwIcon,
  Trash2Icon,
} from 'lucide-react'
import type { AdminLogFile } from '@/lib/mibo-api'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { formatBytes, formatDate } from '../format'

export function LogListPanel({
  tabs,
  logs,
  isLoading,
  isRefreshing,
  onRefresh,
  onPreview,
  onDownload,
  onDelete,
}: {
  tabs: ReactNode
  logs: AdminLogFile[]
  isLoading: boolean
  isRefreshing: boolean
  onRefresh: () => void
  onPreview: (log: AdminLogFile) => void
  onDownload: (log: AdminLogFile) => void
  onDelete: (log: AdminLogFile) => void
}) {
  return (
    <section>
      <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
        <div>
          <p className='text-sm font-medium'>服务器日志文件</p>
          <p className='mt-1 text-xs text-muted-foreground'>
            共 {logs.length} 个日志
          </p>
        </div>
        <div className='flex flex-wrap items-center gap-2 sm:justify-end'>
          {tabs}
          <Button variant='outline' size='sm' onClick={onRefresh}>
            <RefreshCwIcon
              className={cn('size-4', isRefreshing && 'animate-spin')}
            />
            刷新
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant='ghost' size='icon-sm' className='rounded-full'>
                <MoreHorizontalIcon className='size-4' />
                <span className='sr-only'>日志列表操作</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end' className='w-48'>
              <DropdownMenuLabel>列表操作</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={onRefresh}>重新加载</DropdownMenuItem>
              <DropdownMenuItem disabled>清理过期日志</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className='mt-5 overflow-hidden rounded-xl border border-border/60 bg-muted/25'>
        <div className='hidden grid-cols-[minmax(280px,1fr)_120px_160px_96px_44px] border-b border-border/60 px-3 py-2 text-xs font-medium text-muted-foreground md:grid'>
          <div>文件名</div>
          <div>类型</div>
          <div>修改日期</div>
          <div>大小</div>
          <div />
        </div>
        {isLoading ? (
          <LogEmptyState title='正在读取日志' description='请稍候。' />
        ) : logs.length === 0 ? (
          <LogEmptyState
            title='暂无日志文件'
            description='后端会读取 mibo-media-server/data/logs 下的 .txt 与 .log 文件。'
          />
        ) : (
          logs.map((log) => (
            <div
              key={log.name}
              role='button'
              tabIndex={0}
              onClick={() => onPreview(log)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  onPreview(log)
                }
              }}
              className='grid cursor-pointer gap-2 border-b border-border/45 px-3 py-3 text-sm transition-colors last:border-b-0 hover:bg-muted/55 focus-visible:bg-muted/55 focus-visible:outline-none md:min-h-14 md:grid-cols-[minmax(280px,1fr)_120px_160px_96px_44px] md:items-center md:gap-3 md:py-2'
            >
              <div className='flex min-w-0 items-center gap-3'>
                <div className='flex size-8 shrink-0 items-center justify-center rounded-md bg-background/70 text-muted-foreground'>
                  <FileTextIcon className='size-4' />
                </div>
                <div className='min-w-0'>
                  <div className='truncate leading-5 font-medium'>
                    {log.name}
                  </div>
                  <div className='mt-1 text-xs text-muted-foreground md:hidden'>
                    {log.kind}
                  </div>
                </div>
              </div>
              <div className='hidden text-muted-foreground md:block'>
                {log.kind}
              </div>
              <div className='text-muted-foreground'>
                <span className='md:hidden'>修改日期：</span>
                {formatDate(log.modified_at)}
              </div>
              <div className='text-muted-foreground'>
                <span className='md:hidden'>文件尺寸：</span>
                {formatBytes(log.size_bytes)}
              </div>
              <div className='justify-self-end md:self-center'>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant='ghost'
                      size='icon-sm'
                      className='rounded-full'
                      onClick={(event) => event.stopPropagation()}
                    >
                      <MoreHorizontalIcon className='size-4' />
                      <span className='sr-only'>{log.name} 操作</span>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align='end' className='w-44'>
                    <DropdownMenuLabel>日志操作</DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onSelect={() => onPreview(log)}>
                      <EyeIcon className='size-4' />
                      查看
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => onDownload(log)}>
                      <DownloadIcon className='size-4' />
                      下载
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      variant='destructive'
                      onSelect={() => onDelete(log)}
                    >
                      <Trash2Icon className='size-4' />
                      删除
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  )
}

function LogEmptyState({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <div className='rounded-2xl bg-muted/45 px-4 py-12 text-center'>
      <p className='text-sm font-medium'>{title}</p>
      <p className='mt-2 text-sm text-muted-foreground'>{description}</p>
    </div>
  )
}
