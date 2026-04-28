import {
  DownloadIcon,
  EyeIcon,
  FileTextIcon,
  MoreHorizontalIcon,
  RefreshCwIcon,
  Trash2Icon,
} from 'lucide-react'

import { Button } from '#/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '#/components/ui/dropdown-menu'
import type { AdminLogFile } from '#/lib/mibo-api'
import { cn } from '#/lib/utils'

import { formatBytes, formatDate } from '../format'

export function LogListPanel({
  logs,
  isLoading,
  isRefreshing,
  onRefresh,
  onPreview,
  onDownload,
  onDelete,
}: {
  logs: AdminLogFile[]
  isLoading: boolean
  isRefreshing: boolean
  onRefresh: () => void
  onPreview: (log: AdminLogFile) => void
  onDownload: (log: AdminLogFile) => void
  onDelete: (log: AdminLogFile) => void
}) {
  return (
    <section className="rounded-[1.75rem] border border-border/60 bg-card/70 p-4 shadow-sm backdrop-blur-sm sm:p-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-sm font-medium">服务器日志文件</p>
          <p className="mt-1 text-xs text-muted-foreground">
            共 {logs.length} 个日志
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh}>
            <RefreshCwIcon
              className={cn('size-4', isRefreshing && 'animate-spin')}
            />
            刷新
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon-sm" className="rounded-full">
                <MoreHorizontalIcon className="size-4" />
                <span className="sr-only">日志列表操作</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <DropdownMenuLabel>列表操作</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={onRefresh}>重新加载</DropdownMenuItem>
              <DropdownMenuItem disabled>清理过期日志</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className="mt-5 hidden grid-cols-[minmax(0,1fr)_170px_120px_44px] px-4 text-xs font-medium text-muted-foreground md:grid">
        <div>文件名</div>
        <div>修改日期</div>
        <div>文件尺寸</div>
        <div />
      </div>

      <div className="mt-2 space-y-2">
        {isLoading ? (
          <LogEmptyState title="正在读取日志" description="请稍候。" />
        ) : logs.length === 0 ? (
          <LogEmptyState
            title="暂无日志文件"
            description="后端会读取 mibo-media-server/data/logs 下的 .txt 与 .log 文件。"
          />
        ) : (
          logs.map((log) => (
            <div
              key={log.name}
              className="grid gap-3 rounded-2xl bg-muted/55 px-4 py-4 text-sm transition-colors hover:bg-muted/80 md:grid-cols-[minmax(0,1fr)_170px_120px_44px] md:items-center"
            >
              <div className="flex min-w-0 items-center gap-3">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-background/80 text-muted-foreground">
                  <FileTextIcon className="size-4" />
                </div>
                <div className="min-w-0">
                  <div className="truncate font-medium">{log.name}</div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {log.kind}
                  </div>
                </div>
              </div>
              <div className="text-muted-foreground md:text-foreground">
                <span className="md:hidden">修改日期：</span>
                {formatDate(log.modified_at)}
              </div>
              <div className="text-muted-foreground md:text-foreground">
                <span className="md:hidden">文件尺寸：</span>
                {formatBytes(log.size_bytes)}
              </div>
              <div className="justify-self-end">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="rounded-full"
                    >
                      <MoreHorizontalIcon className="size-4" />
                      <span className="sr-only">{log.name} 操作</span>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-44">
                    <DropdownMenuLabel>日志操作</DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onSelect={() => onPreview(log)}>
                      <EyeIcon className="size-4" />
                      查看
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => onDownload(log)}>
                      <DownloadIcon className="size-4" />
                      下载
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      variant="destructive"
                      onSelect={() => onDelete(log)}
                    >
                      <Trash2Icon className="size-4" />
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
    <div className="rounded-2xl bg-muted/45 px-4 py-12 text-center">
      <p className="text-sm font-medium">{title}</p>
      <p className="mt-2 text-sm text-muted-foreground">{description}</p>
    </div>
  )
}
