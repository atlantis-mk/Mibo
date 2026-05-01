import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "#/components/ui/drawer"
import { Badge } from "#/components/ui/badge"
import { type Job, type Schedule, type ScheduleRun } from "#/lib/mibo-api"

import { formatDateTime, formatKind } from "./schedule-list"

type Props = {
  isLoading: boolean
  onOpenChange: (open: boolean) => void
  open: boolean
  runs: ScheduleRun[]
  schedule?: Schedule | null
}

export function ScheduleRunHistoryDrawer({
  isLoading,
  onOpenChange,
  open,
  runs,
  schedule,
}: Props) {
  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="max-w-xl">
        <DrawerHeader>
          <DrawerTitle>
            {schedule ? `${schedule.name} · 执行详情` : "执行详情"}
          </DrawerTitle>
          <DrawerDescription>
            查看每次计划触发的后台任务、执行状态、耗时和失败原因。
          </DrawerDescription>
        </DrawerHeader>

        <div className="space-y-3 overflow-y-auto px-4 pb-6">
          {schedule ? (
            <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3 text-sm text-muted-foreground">
              {formatKind(schedule.kind)} ·{" "}
              {schedule.scope_kind === "library" ? "单媒体库" : "全局范围"}
            </div>
          ) : null}

          {isLoading ? (
            <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-6 text-sm text-muted-foreground">
              正在加载最近运行…
            </div>
          ) : runs.length ? (
            runs.map((run) => <RunDetailCard key={run.id} run={run} />)
          ) : (
            <div className="rounded-[1.1rem] border border-dashed border-border/60 px-4 py-6 text-sm text-muted-foreground">
              当前没有可展示的运行历史。
            </div>
          )}
        </div>
      </DrawerContent>
    </Drawer>
  )
}

function RunDetailCard({ run }: { run: ScheduleRun }) {
  const job = run.job
  const payload = parsePayload(job?.payload_json)
  const startedAt = job?.started_at ?? run.started_at
  const finishedAt = job?.finished_at ?? run.finished_at
  const message = job?.error_message || run.error_summary

  return (
    <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={statusVariant(run.status)}>
            {formatStatus(run.status)}
          </Badge>
          <span className="text-sm font-medium text-foreground">
            {formatJobKind(job?.kind, payload.kind)}
          </span>
        </div>
        <div className="text-xs text-muted-foreground">
          Job #{run.job_id ?? "—"}
        </div>
      </div>

      <div className="mt-3 grid gap-2 text-sm text-muted-foreground sm:grid-cols-2">
        <Detail label="执行范围" value={formatPayloadScope(payload)} />
        <Detail
          label="尝试次数"
          value={job ? `${job.attempts} 次` : "未开始"}
        />
        <Detail label="开始时间" value={formatDateTime(startedAt)} />
        <Detail label="结束时间" value={formatDateTime(finishedAt)} />
        <Detail
          label="耗时"
          value={formatDuration(startedAt, finishedAt, run.status)}
        />
        <Detail
          label="队列状态"
          value={formatStatus(job?.status ?? run.status)}
        />
      </div>

      <div className="mt-3 rounded-lg border border-border/50 bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
        <span className="font-medium text-foreground">结果：</span>
        {message || "暂无额外摘要"}
      </div>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="text-xs text-muted-foreground/80">{label}</span>
      <div className="text-foreground">{value}</div>
    </div>
  )
}

function parsePayload(value?: string) {
  if (!value) return {}
  try {
    return JSON.parse(value) as Record<string, unknown>
  } catch {
    return {}
  }
}

function statusVariant(status: string) {
  if (status === "completed") return "default"
  if (status === "failed") return "destructive"
  return "outline"
}

function formatStatus(status: string) {
  const labels: Record<string, string> = {
    queued: "排队中",
    running: "运行中",
    completed: "成功",
    failed: "失败",
  }
  return labels[status] ?? status
}

function formatJobKind(jobKind?: Job["kind"], payloadKind?: unknown) {
  const kind = typeof payloadKind === "string" ? payloadKind : jobKind
  if (kind?.startsWith("schedule_"))
    return formatKind(kind.slice("schedule_".length))
  return kind ? formatKind(kind) : "后台任务"
}

function formatPayloadScope(payload: Record<string, unknown>) {
  if (payload.scope_kind === "library") {
    return typeof payload.library_id === "number"
      ? `媒体库 #${payload.library_id}`
      : "单媒体库"
  }
  if (payload.scope_kind === "global") return "全局范围"
  return "未记录"
}

function formatDuration(
  startedAt?: string,
  finishedAt?: string,
  status?: string
) {
  if (!startedAt || !finishedAt)
    return status === "running" ? "运行中" : "未完成"
  const seconds = Math.max(
    0,
    Math.round((Date.parse(finishedAt) - Date.parse(startedAt)) / 1000)
  )
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return remainingSeconds > 0
    ? `${minutes} 分 ${remainingSeconds} 秒`
    : `${minutes} 分`
}
