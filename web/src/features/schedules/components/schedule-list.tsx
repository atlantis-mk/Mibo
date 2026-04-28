import { LoaderCircleIcon, PlayIcon } from 'lucide-react'

import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { type Schedule } from '#/lib/mibo-api'
import { cn } from '#/lib/utils'

type Props = {
  isRunning?: boolean
  onCreate: () => void
  onEdit: (schedule: Schedule) => void
  onRunNow: (schedule: Schedule) => void
  onShowHistory: (schedule: Schedule) => void
  onToggle: (schedule: Schedule) => void
  runningScheduleId?: number
  schedules: Schedule[]
}

type ScheduleGroup = {
  key: string
  title: string
  schedules: Schedule[]
}

export function ScheduleList({
  isRunning,
  onCreate,
  onEdit,
  onRunNow,
  onShowHistory,
  onToggle,
  runningScheduleId,
  schedules,
}: Props) {
  const groups = groupSchedules(schedules)

  if (schedules.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-border/70 bg-card/50 px-6 py-10 text-center">
        <h2 className="text-lg font-medium text-foreground">暂无计划任务</h2>
        <p className="mx-auto mt-2 max-w-xl text-sm leading-6 text-muted-foreground">
          创建扫描、清理或链接检查任务后，这里会按维护类别显示上次运行时间、耗时和手动执行入口。
        </p>
        <Button className="mt-5" onClick={onCreate}>
          新建计划任务
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      {groups.map((group) => (
        <section key={group.key} className="space-y-2">
          <h2 className="px-1 text-xl font-medium text-muted-foreground">
            {group.title}
          </h2>
          <div className="overflow-hidden rounded-xl border border-border/60 bg-card/40">
            {group.schedules.map((schedule, index) => {
              const running = isRunning && runningScheduleId === schedule.id

              return (
                <article
                  key={schedule.id}
                  className={cn(
                    'grid gap-4 px-4 py-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:px-5',
                    index > 0 && 'border-t border-border/60',
                  )}
                >
                  <div className="min-w-0 space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="text-base font-medium text-foreground">
                        {schedule.name}
                      </h3>
                      <Badge
                        variant={schedule.enabled ? 'secondary' : 'outline'}
                        className="rounded-full text-[11px]"
                      >
                        {schedule.enabled ? '已启用' : '已停用'}
                      </Badge>
                    </div>

                    <div className="flex flex-wrap gap-x-5 gap-y-1 text-sm text-muted-foreground">
                      <span>上次运行：{formatLastRun(schedule)}</span>
                      <span>耗时：{formatLastDuration(schedule)}</span>
                      <span>
                        下次运行：{formatDateTime(schedule.next_run_at)}
                      </span>
                    </div>

                    <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
                      {formatDescription(schedule)}
                    </p>

                    <div className="flex flex-wrap gap-2 text-xs">
                      <button
                        type="button"
                        className="text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
                        onClick={() => onShowHistory(schedule)}
                      >
                        查看历史
                      </button>
                      <span className="text-border">/</span>
                      <button
                        type="button"
                        className="text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
                        onClick={() => onEdit(schedule)}
                      >
                        编辑
                      </button>
                      <span className="text-border">/</span>
                      <button
                        type="button"
                        className="text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
                        onClick={() => onToggle(schedule)}
                      >
                        {schedule.enabled ? '停用' : '启用'}
                      </button>
                      <span className="text-muted-foreground">
                        {formatLatestResult(schedule)}
                      </span>
                    </div>
                  </div>

                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="size-11 rounded-full border-primary/35 text-primary hover:bg-primary hover:text-primary-foreground sm:justify-self-end"
                    disabled={running}
                    onClick={() => onRunNow(schedule)}
                  >
                    {running ? (
                      <LoaderCircleIcon className="size-4 animate-spin" />
                    ) : (
                      <PlayIcon className="ml-0.5 size-4 fill-current" />
                    )}
                    <span className="sr-only">立即运行 {schedule.name}</span>
                  </Button>
                </article>
              )
            })}
          </div>
        </section>
      ))}
    </div>
  )
}

function groupSchedules(schedules: Schedule[]): ScheduleGroup[] {
  const groupMap = new Map<string, ScheduleGroup>()

  for (const schedule of schedules) {
    const group = getScheduleGroup(schedule)
    const existing = groupMap.get(group.key)

    if (existing) {
      existing.schedules.push(schedule)
    } else {
      groupMap.set(group.key, { ...group, schedules: [schedule] })
    }
  }

  return Array.from(groupMap.values())
}

function getScheduleGroup(schedule: Schedule) {
  switch (schedule.kind) {
    case 'scan':
      return { key: 'library', title: 'Library' }
    case 'library_cleanup':
      return { key: 'database', title: 'Database' }
    case 'invalid_link_check':
      return { key: 'application', title: 'Application' }
    default:
      return { key: 'maintenance', title: 'Maintenance' }
  }
}

export function formatKind(kind: string) {
  switch (kind) {
    case 'scan':
      return '媒体扫描'
    case 'library_cleanup':
      return '库清理'
    case 'invalid_link_check':
      return '失效链接检查'
    default:
      return kind
  }
}

export function formatScope(scope: Schedule['scope_kind'], libraryId?: number) {
  return scope === 'library'
    ? libraryId
      ? `媒体库 #${libraryId}`
      : '单媒体库'
    : '全局范围'
}

export function formatFrequency(frequency: Schedule['frequency']) {
  if (frequency.kind === 'daily') return `每天 ${frequency.time_of_day}`
  if (frequency.kind === 'weekly') {
    return `每周 ${formatWeekday(frequency.weekday)} ${frequency.time_of_day}`
  }
  if (frequency.kind === 'monthly') {
    return `每月 ${frequency.day_of_month} 日 ${frequency.time_of_day}`
  }
  return frequency.time_of_day
}

export function formatDateTime(value?: string) {
  if (!value) return '未安排'
  return new Date(value).toLocaleString('zh-CN', {
    hour12: false,
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function formatLatestResult(schedule: Schedule) {
  if (!schedule.latest_run_status) return '暂无历史'
  const statusMap: Record<string, string> = {
    queued: '排队中',
    running: '运行中',
    completed: '已完成',
    failed: '失败',
  }
  const label =
    statusMap[schedule.latest_run_status] ?? schedule.latest_run_status
  return schedule.latest_run_message
    ? `${label} · ${schedule.latest_run_message}`
    : label
}

function formatDescription(schedule: Schedule) {
  const scope = formatScope(schedule.scope_kind, schedule.library_id)
  const frequency = formatFrequency(schedule.frequency)

  switch (schedule.kind) {
    case 'scan':
      return `扫描 ${scope} 的媒体文件与目录变化，并按 ${frequency} 自动更新媒体库。`
    case 'library_cleanup':
      return `清理 ${scope} 中不再存在的条目、孤立记录和过期索引，保持数据库可用。`
    case 'invalid_link_check':
      return `检查 ${scope} 的资源链接和目录引用，发现失效项后交给治理流程处理。`
    default:
      return `${formatKind(schedule.kind)}，作用范围为 ${scope}，计划频率为 ${frequency}。`
  }
}

function formatLastRun(schedule: Schedule) {
  const value =
    schedule.latest_run_finished_at ??
    schedule.latest_run_started_at ??
    schedule.recent_runs?.[0]?.finished_at ??
    schedule.recent_runs?.[0]?.started_at

  if (!value) return '从未运行'
  return formatRelativeTime(value)
}

function formatLastDuration(schedule: Schedule) {
  const startedAt =
    schedule.latest_run_started_at ?? schedule.recent_runs?.[0]?.started_at
  const finishedAt =
    schedule.latest_run_finished_at ?? schedule.recent_runs?.[0]?.finished_at

  if (!startedAt || !finishedAt) {
    return schedule.latest_run_status === 'running' ? '运行中' : '0 seconds'
  }

  const durationSeconds = Math.max(
    0,
    Math.round((Date.parse(finishedAt) - Date.parse(startedAt)) / 1000),
  )

  if (durationSeconds < 60) return `${durationSeconds} seconds`

  const minutes = Math.floor(durationSeconds / 60)
  const seconds = durationSeconds % 60
  return seconds > 0 ? `${minutes} min ${seconds} sec` : `${minutes} min`
}

function formatRelativeTime(value: string) {
  const timestamp = Date.parse(value)
  if (Number.isNaN(timestamp)) return formatDateTime(value)

  const seconds = Math.max(0, Math.round((Date.now() - timestamp) / 1000))
  if (seconds < 60) return `${seconds} 秒钟前`

  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes} 分钟前`

  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours} 小时前`

  const days = Math.round(hours / 24)
  return `${days} 天前`
}

function formatWeekday(weekday?: number) {
  const labels = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return typeof weekday === 'number' && weekday >= 0 && weekday < labels.length
    ? labels[weekday]
    : '未设置'
}
