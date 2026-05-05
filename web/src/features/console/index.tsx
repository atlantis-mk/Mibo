import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  ActivityIcon,
  AlertTriangleIcon,
  ArrowLeftIcon,
  CastIcon,
  CheckCircle2Icon,
  ClockIcon,
  DatabaseIcon,
  HardDriveIcon,
  LayoutDashboardIcon,
  MonitorSmartphoneIcon,
  PlayCircleIcon,
  RefreshCwIcon,
  ServerIcon,
  SettingsIcon,
  ShieldIcon,
  UserIcon,
  WifiIcon,
  WrenchIcon,
  XCircleIcon,
} from "lucide-react"
import { toast } from "sonner"

import { Button } from "#/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "#/components/ui/card"
import { Skeleton } from "#/components/ui/skeleton"
import { SidebarTrigger } from "#/components/ui/sidebar"
import type {
  ConsoleActivityEvent,
  ConsoleQuickAction,
  ConsoleStatus,
  ConsoleSummary,
} from "#/lib/mibo-api"
import { createAuthedMiboApi, miboQueryKeys } from "#/lib/mibo-query"
import { consoleSummaryQueryOptions } from "#/lib/mibo-query"
import { cn } from "#/lib/utils"
import { useAuthStore } from "#/stores/auth-store"

const managementEntries = [
  {
    title: "用户",
    description: "用户与权限管理规划中",
    icon: UserIcon,
    disabled: true,
  },
  {
    title: "媒体库",
    description: "管理来源、媒体库和扫描",
    icon: DatabaseIcon,
    to: "/settings/library",
  },
  {
    title: "直播电视",
    description: "配置直播源、EPG 和录制",
    icon: PlayCircleIcon,
    to: "/settings/live-tv",
  },
  {
    title: "网络",
    description: "打开设置以配置访问方式",
    icon: WifiIcon,
    to: "/settings",
  },
  {
    title: "转码",
    description: "播放与 HLS 设置",
    icon: MonitorSmartphoneIcon,
    to: "/settings/playback",
  },
  { title: "数据库", description: "目录一致性和投影维护", icon: HardDriveIcon },
  {
    title: "转换",
    description: "媒体转换工作流规划中",
    icon: RefreshCwIcon,
    disabled: true,
  },
  {
    title: "计划任务",
    description: "查看扫描计划和历史",
    icon: ClockIcon,
    to: "/settings/schedules",
  },
  {
    title: "日志",
    description: "日志查看页面尚未实现",
    icon: ActivityIcon,
    disabled: true,
  },
  {
    title: "插件",
    description: "插件系统规划中",
    icon: WrenchIcon,
    disabled: true,
  },
  {
    title: "设备",
    description: "设备会话管理规划中",
    icon: CastIcon,
    disabled: true,
  },
  {
    title: "下载",
    description: "离线下载尚未实现",
    icon: ArrowLeftIcon,
    disabled: true,
  },
  {
    title: "相机上传",
    description: "移动端上传规划中",
    icon: MonitorSmartphoneIcon,
    disabled: true,
  },
  {
    title: "DLNA",
    description: "DLNA 服务尚未实现",
    icon: WifiIcon,
    disabled: true,
  },
  { title: "高级维护", description: "谨慎运行昂贵维护操作", icon: ShieldIcon },
] as const

export default function ConsolePage({
  embedded = false,
}: {
  embedded?: boolean
}) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const queryToken = token ?? "guest"
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const summaryQuery = useQuery({
    ...consoleSummaryQueryOptions(queryToken),
    enabled: !!token,
  })
  const actionMutation = useMutation({
    mutationFn: (action: ConsoleQuickAction) =>
      createAuthedMiboApi(queryToken).runConsoleAction(action.id),
    onSuccess: async (_result, action) => {
      toast.success(`${action.label} 已完成`)
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.consoleSummary(queryToken),
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const summary = summaryQuery.data

  const runAction = (action: ConsoleQuickAction) => {
    if (action.disabled) return
    if (action.kind === "route" && action.route) {
      void navigate({ to: action.route as "/" })
      return
    }
    if (action.kind !== "mutation") return
    if (action.confirm && !window.confirm(`确认执行“${action.label}”？`)) return
    actionMutation.mutate(action)
  }

  return (
    <div
      className={cn(
        "flex-1 overflow-y-auto text-foreground",
        embedded ? "bg-transparent" : "min-h-screen bg-background"
      )}
    >
      <div
        className={cn(
          "flex w-full flex-col gap-6",
          embedded ? "p-0" : "px-4 py-5 sm:px-6 lg:px-8"
        )}
      >
        {!embedded ? (
          <header className="flex flex-col gap-4 rounded-3xl border border-border bg-card p-4 shadow-sm sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <SidebarTrigger />
              <Button variant="ghost" size="icon" asChild>
                <Link to="/">
                  <ArrowLeftIcon className="size-4" />
                </Link>
              </Button>
              <div>
                <p className="text-sm font-medium text-primary">Mibo Admin</p>
                <h1 className="text-2xl font-semibold tracking-tight">
                  控制台
                </h1>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" disabled title="播放到设备尚未实现">
                <CastIcon className="size-4" />
                投放
              </Button>
              <Button variant="outline" asChild>
                <Link to="/settings">
                  <SettingsIcon className="size-4" />
                  设置
                </Link>
              </Button>
              <div className="rounded-full border border-border px-3 py-2 text-sm text-muted-foreground">
                {user?.username ?? "未登录"}
              </div>
            </div>
          </header>
        ) : null}

        {summaryQuery.isPending ? <ConsoleSkeleton /> : null}
        {summaryQuery.isError ? (
          <Card className="border-destructive/30 bg-destructive/10">
            <CardContent className="flex flex-col gap-3 p-6 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 className="font-semibold text-destructive">
                  控制台数据加载失败
                </h2>
                <p className="text-sm text-destructive">
                  {summaryQuery.error.message}
                </p>
              </div>
              <Button onClick={() => void summaryQuery.refetch()}>重试</Button>
            </CardContent>
          </Card>
        ) : null}

        {summary ? (
          <div className="flex flex-col gap-6">
            <PartialWarnings summary={summary} />
            <ServerOverview summary={summary} />
            <MetricGrid summary={summary} />
            <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
              <QuickActions
                actions={summary.quick_actions ?? []}
                isRunning={actionMutation.isPending}
                onRun={runAction}
              />
              <DeviceSection summary={summary} />
            </section>
            <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
              <ManagementGrid summary={summary} />
              <ActivityTimeline events={summary.activity ?? []} />
            </section>
          </div>
        ) : null}
      </div>
    </div>
  )
}

function ServerOverview({ summary }: { summary: ConsoleSummary }) {
  const addresses = summary.access.addresses ?? []
  const storageLabel = summary.health.storage.message || "暂无"
  const fields = [
    ["服务", summary.server.service],
    ["版本", summary.server.version || "未知"],
    ["更新状态", summary.server.update_status || "未知"],
    ["API 端口", summary.server.port ? String(summary.server.port) : "未知"],
    ["运行时长", formatDuration(summary.server.uptime_seconds)],
    ["存储", storageLabel],
    ["数据库", summary.server.database_driver],
  ]
  return (
    <Card className="border-border bg-card shadow-sm">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <ServerIcon className="size-5 text-primary" />
          服务器概览
        </CardTitle>
        <StatusPill status={summary.server.status} />
      </CardHeader>
      <CardContent className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {fields.map(([label, value]) => (
            <div key={label} className="rounded-2xl bg-muted p-3">
              <p className="text-xs text-muted-foreground">{label}</p>
              <p className="mt-1 truncate text-sm font-medium">{value}</p>
            </div>
          ))}
        </div>
        <div className="space-y-2 rounded-2xl bg-muted p-3">
          <p className="text-xs font-medium text-muted-foreground">访问地址</p>
          {addresses.map((address) => (
            <div
              key={`${address.kind}-${address.url ?? address.status}`}
              className="flex items-center justify-between gap-3 rounded-xl bg-card px-3 py-2 text-sm"
            >
              <span className="text-muted-foreground">{address.label}</span>
              <span className="truncate font-mono text-xs">
                {address.url ?? address.message ?? "未配置"}
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function MetricGrid({ summary }: { summary: ConsoleSummary }) {
  const metrics = [
    ["媒体库", summary.media.libraries, DatabaseIcon],
    ["媒体源", summary.media.media_sources, HardDriveIcon],
    ["目录项目", summary.media.catalog_items, LayoutDashboardIcon],
    ["库存文件", summary.media.inventory_files, HardDriveIcon],
    ["电影", summary.media.movies, PlayCircleIcon],
    ["剧集", summary.media.series, MonitorSmartphoneIcon],
    ["分集", summary.media.episodes, PlayCircleIcon],
    ["人物", summary.media.people, UserIcon],
    ["活动任务", summary.media.active_jobs, RefreshCwIcon],
    ["整理失败", summary.media.ingest?.failed ?? 0, XCircleIcon],
    ["待确认", summary.media.ingest?.review_required ?? 0, AlertTriangleIcon],
    ["告警/失败", summary.media.warnings, AlertTriangleIcon],
  ] as const
  return (
    <section className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
      {metrics.map(([label, value, Icon]) => (
        <Card key={label} className="border-border bg-card shadow-sm">
          <CardContent className="flex items-center justify-between p-4">
            <div>
              <p className="text-xs text-muted-foreground">{label}</p>
              <p className="mt-1 text-2xl font-semibold">{value}</p>
            </div>
            <Icon className="size-5 text-primary" />
          </CardContent>
        </Card>
      ))}
    </section>
  )
}

function QuickActions({
  actions,
  isRunning,
  onRun,
}: {
  actions: ConsoleQuickAction[]
  isRunning: boolean
  onRun: (action: ConsoleQuickAction) => void
}) {
  return (
    <Card className="bg-card shadow-sm">
      <CardHeader>
        <CardTitle>快捷操作</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {actions.map((action) => (
          <button
            key={action.id}
            type="button"
            disabled={action.disabled || isRunning}
            onClick={() => onRun(action)}
            className={cn(
              "rounded-2xl border p-4 text-left transition hover:border-primary/30 hover:bg-accent hover:text-accent-foreground",
              action.disabled &&
                "cursor-not-allowed border-border bg-muted text-muted-foreground opacity-60 hover:border-border hover:bg-muted hover:text-muted-foreground",
              action.risk === "danger" &&
                !action.disabled &&
                "border-destructive/30"
            )}
          >
            <div className="flex items-center justify-between gap-3">
              <p className="font-medium">{action.label}</p>
              {action.confirm ? (
                <AlertTriangleIcon className="size-4 text-muted-foreground" />
              ) : (
                <CheckCircle2Icon className="size-4 text-primary" />
              )}
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              {action.disabled_reason ?? action.description}
            </p>
          </button>
        ))}
      </CardContent>
    </Card>
  )
}

function ActivityTimeline({ events }: { events: ConsoleActivityEvent[] }) {
  return (
    <Card className="bg-card shadow-sm">
      <CardHeader>
        <CardTitle>最近活动</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {events.length === 0 ? (
          <div className="rounded-2xl border border-dashed p-6 text-sm text-muted-foreground">
            暂无活动。播放、扫描和系统事件会显示在这里。
          </div>
        ) : (
          events.map((event) => (
            <div key={event.id} className="flex gap-3 rounded-2xl bg-muted p-3">
              <SeverityIcon severity={event.severity} />
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium">{event.message}</p>
                <p className="truncate text-xs text-muted-foreground">
                  {[event.user, event.device, event.media_title]
                    .filter(Boolean)
                    .join(" · ") || event.type}
                </p>
              </div>
              <time className="text-xs text-muted-foreground">
                {formatDate(event.timestamp)}
              </time>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  )
}

function ManagementGrid({ summary }: { summary: ConsoleSummary }) {
  return (
    <Card className="bg-card shadow-sm">
      <CardHeader>
        <CardTitle>管理入口</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {managementEntries.map((entry) => {
          const Icon = entry.icon
          const isDisabled = "disabled" in entry && entry.disabled
          const stats = managementEntryStats(entry.title, summary)
          const content = (
            <div
              className={cn(
                "h-full rounded-2xl border p-4",
                isDisabled
                  ? "bg-muted text-muted-foreground opacity-70"
                  : "bg-card hover:border-primary/30 hover:bg-accent hover:text-accent-foreground"
              )}
            >
              <Icon className="mb-3 size-5" />
              <p className="font-medium">{entry.title}</p>
              <p className="mt-1 text-sm text-muted-foreground">
                {entry.description}
              </p>
              {stats.length > 0 ? (
                <div className="mt-3 flex flex-wrap gap-2">
                  {stats.map((stat) => (
                    <span
                      key={stat.label}
                      className="rounded-full border border-border bg-muted px-2 py-1 text-xs text-muted-foreground"
                    >
                      {stat.label} {stat.value}
                    </span>
                  ))}
                </div>
              ) : null}
              {isDisabled ? <p className="mt-3 text-xs">即将推出</p> : null}
            </div>
          )
          return "to" in entry && entry.to ? (
            <Link key={entry.title} to={entry.to as "/settings"}>
              {content}
            </Link>
          ) : (
            <div key={entry.title}>{content}</div>
          )
        })}
      </CardContent>
    </Card>
  )
}

function managementEntryStats(title: string, summary: ConsoleSummary) {
  switch (title) {
    case "媒体库":
      return [
        { label: "媒体库", value: summary.media.libraries },
        { label: "媒体源", value: summary.media.media_sources },
      ]
    case "数据库":
      return [
        { label: "目录项目", value: summary.media.catalog_items },
        { label: "库存文件", value: summary.media.inventory_files },
      ]
    case "计划任务":
      return [
        { label: "计划", value: summary.media.schedules },
        { label: "启用", value: summary.media.enabled_schedules },
      ]
    case "高级维护":
      return [
        { label: "活动任务", value: summary.media.active_jobs },
        { label: "告警", value: summary.media.warnings },
        { label: "整理失败", value: summary.media.ingest?.failed ?? 0 },
      ]
    default:
      return []
  }
}

function DeviceSection({ summary }: { summary: ConsoleSummary }) {
  const devices = summary.devices ?? []
  return (
    <Card className="bg-card shadow-sm">
      <CardHeader>
        <CardTitle>设备</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {devices.length === 0 ? (
          <div className="rounded-2xl bg-muted p-4 text-sm text-muted-foreground">
            暂无连接设备数据。下载、相机上传和 DLNA 仍为计划功能。
          </div>
        ) : (
          devices.map((device) => (
            <div key={device.id} className="rounded-2xl bg-muted p-3">
              <p className="font-medium">{device.name}</p>
              <p className="text-sm text-muted-foreground">
                {device.user ?? "未知用户"} · {device.state ?? "未知状态"}
              </p>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  )
}

function PartialWarnings({ summary }: { summary: ConsoleSummary }) {
  const warnings = summary.warnings ?? []
  if (warnings.length === 0) return null
  return (
    <div className="rounded-2xl border border-border bg-muted p-4 text-sm text-foreground">
      <p className="font-medium">部分数据不可用</p>
      <p className="mt-1">
        {warnings
          .map((warning) => `${warning.section}: ${warning.message}`)
          .join("；")}
      </p>
    </div>
  )
}

function ConsoleSkeleton() {
  return (
    <div className="grid gap-4">
      <Skeleton className="h-44 rounded-3xl" />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        {Array.from({ length: 10 }).map((_, index) => (
          <Skeleton key={index} className="h-24 rounded-2xl" />
        ))}
      </div>
    </div>
  )
}

function StatusPill({ status }: { status: ConsoleStatus }) {
  return (
    <span
      className={cn(
        "rounded-full px-2.5 py-1 text-xs font-medium",
        statusClass(status)
      )}
    >
      {statusLabel(status)}
    </span>
  )
}

function SeverityIcon({
  severity,
}: {
  severity: ConsoleActivityEvent["severity"]
}) {
  if (severity === "error")
    return <XCircleIcon className="mt-0.5 size-4 text-destructive" />
  if (severity === "warning")
    return <AlertTriangleIcon className="mt-0.5 size-4 text-muted-foreground" />
  return <CheckCircle2Icon className="mt-0.5 size-4 text-primary" />
}

function statusClass(status: string) {
  if (status === "ok" || status === "available")
    return "bg-primary/10 text-primary"
  if (status === "warning" || status === "unknown")
    return "bg-muted text-muted-foreground"
  if (status === "error") return "bg-destructive/10 text-destructive"
  return "bg-muted text-muted-foreground"
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    ok: "正常",
    available: "可用",
    warning: "警告",
    error: "错误",
    unknown: "未知",
    unavailable: "不可用",
    not_configured: "未配置",
  }
  return labels[status] ?? status
}

function formatDuration(seconds: number) {
  if (!Number.isFinite(seconds) || seconds <= 0) return "未知"
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`
  return `${minutes} 分钟`
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "未知时间"
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  })
}
