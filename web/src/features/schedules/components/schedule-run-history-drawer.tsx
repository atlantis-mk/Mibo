import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from '#/components/ui/drawer'
import { Badge } from '#/components/ui/badge'
import { type Schedule, type ScheduleRun } from '#/lib/mibo-api'

import { formatDateTime, formatKind } from './schedule-list'

type Props = {
  isLoading: boolean
  onOpenChange: (open: boolean) => void
  open: boolean
  runs: ScheduleRun[]
  schedule?: Schedule | null
}

export function ScheduleRunHistoryDrawer({ isLoading, onOpenChange, open, runs, schedule }: Props) {
  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="max-w-xl">
        <DrawerHeader>
          <DrawerTitle>{schedule ? `${schedule.name} · 运行历史` : '运行历史'}</DrawerTitle>
          <DrawerDescription>
            历史按 schedule 维度组织，只展示状态、时间和错误摘要，不暴露原始 payload JSON。
          </DrawerDescription>
        </DrawerHeader>

        <div className="space-y-3 overflow-y-auto px-4 pb-6">
          {schedule ? (
            <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3 text-sm text-muted-foreground">
              {formatKind(schedule.kind)} · {schedule.scope_kind === 'library' ? '单媒体库' : '全局范围'}
            </div>
          ) : null}

          {isLoading ? (
            <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-6 text-sm text-muted-foreground">正在加载最近运行…</div>
          ) : runs.length ? (
            runs.map((run) => (
              <div key={run.id} className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <Badge variant={run.status === 'completed' ? 'default' : run.status === 'failed' ? 'destructive' : 'outline'}>{run.status}</Badge>
                  <div className="text-xs text-muted-foreground">Job #{run.job_id ?? '—'}</div>
                </div>
                <div className="mt-2 space-y-1 text-sm text-muted-foreground">
                  <div>开始：{formatDateTime(run.started_at)}</div>
                  <div>结束：{formatDateTime(run.finished_at)}</div>
                  <div>摘要：{run.error_summary || '无额外摘要'}</div>
                </div>
              </div>
            ))
          ) : (
            <div className="rounded-[1.1rem] border border-dashed border-border/60 px-4 py-6 text-sm text-muted-foreground">当前没有可展示的运行历史。</div>
          )}
        </div>
      </DrawerContent>
    </Drawer>
  )
}
