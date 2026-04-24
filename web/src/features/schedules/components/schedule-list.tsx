import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { Card, CardContent } from '#/components/ui/card'
import { type Schedule } from '#/lib/mibo-api'

type Props = {
  onCreate: () => void
  onEdit: (schedule: Schedule) => void
  onRunNow: (schedule: Schedule) => void
  onSelect: (schedule: Schedule) => void
  onShowHistory: (schedule: Schedule) => void
  onToggle: (schedule: Schedule) => void
  schedules: Schedule[]
  selectedScheduleId?: number
}

export function ScheduleList({ onCreate, onEdit, onRunNow, onSelect, onShowHistory, onToggle, schedules, selectedScheduleId }: Props) {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-medium text-foreground">计划任务列表</div>
          <div className="text-sm text-muted-foreground">主列表直接呈现启停状态、next run 与 latest result。</div>
        </div>
        <Button onClick={onCreate}>新建计划任务</Button>
      </div>

      {schedules.map((schedule) => (
        <Card key={schedule.id} className={`rounded-[1.25rem] border py-0 ${selectedScheduleId === schedule.id ? 'border-primary/50 bg-primary/5' : 'border-border/60 bg-background/60'}`}>
          <CardContent className="space-y-3 px-4 py-4">
            <button type="button" className="w-full text-left" onClick={() => onSelect(schedule)}>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="space-y-2">
                  <div className="text-sm font-medium text-foreground">{schedule.name}</div>
                  <div className="flex flex-wrap gap-2">
                    <Badge variant="outline" className="border-border/60 bg-card/70 text-[11px]">{formatKind(schedule.kind)}</Badge>
                    <Badge variant="secondary" className="text-[11px]">{formatScope(schedule.scope_kind, schedule.library_id)}</Badge>
                    <Badge variant={schedule.enabled ? 'default' : 'outline'} className="text-[11px]">{schedule.enabled ? '已启用' : '已停用'}</Badge>
                  </div>
                </div>
                <div className="space-y-1 text-right text-xs text-muted-foreground">
                  <div>下次运行：{formatDateTime(schedule.next_run_at)}</div>
                  <div>最近结果：{formatLatestResult(schedule)}</div>
                </div>
              </div>
              <div className="mt-3 text-sm text-muted-foreground">{formatFrequency(schedule.frequency)}</div>
            </button>

            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" onClick={() => onEdit(schedule)}>编辑</Button>
              <Button size="sm" variant="outline" onClick={() => onToggle(schedule)}>{schedule.enabled ? '停用' : '启用'}</Button>
              <Button size="sm" variant="outline" onClick={() => onRunNow(schedule)}>立即运行</Button>
              <Button size="sm" variant="ghost" onClick={() => onShowHistory(schedule)}>查看历史</Button>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export function formatKind(kind: string) {
  switch (kind) {
    case 'scan': return '媒体扫描'
    case 'metadata_refetch': return '元数据重抓'
    case 'trailer_sync': return '预告片同步'
    case 'library_cleanup': return '库清理'
    case 'invalid_link_check': return '失效链接检查'
    case 'artwork_refresh': return '封面刷新'
    default: return kind
  }
}

export function formatScope(scope: Schedule['scope_kind'], libraryId?: number) {
  return scope === 'library' ? (libraryId ? `媒体库 #${libraryId}` : '单媒体库') : '全局范围'
}

export function formatFrequency(frequency: Schedule['frequency']) {
  if (frequency.kind === 'daily') return `每天 ${frequency.time_of_day}`
  if (frequency.kind === 'weekly') return `每周 ${formatWeekday(frequency.weekday)} ${frequency.time_of_day}`
  if (frequency.kind === 'monthly') return `每月 ${frequency.day_of_month} 日 ${frequency.time_of_day}`
  return frequency.time_of_day
}

export function formatDateTime(value?: string) {
  if (!value) return '未安排'
  return new Date(value).toLocaleString('zh-CN', { hour12: false, month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

export function formatLatestResult(schedule: Schedule) {
  if (!schedule.latest_run_status) return '暂无历史'
  const statusMap: Record<string, string> = { queued: '排队中', running: '运行中', completed: '已完成', failed: '失败' }
  const label = statusMap[schedule.latest_run_status] ?? schedule.latest_run_status
  return schedule.latest_run_message ? `${label} · ${schedule.latest_run_message}` : label
}

function formatWeekday(weekday?: number) {
  const labels = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return typeof weekday === 'number' && weekday >= 0 && weekday < labels.length ? labels[weekday] : '未设置'
}
