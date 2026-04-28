import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import type { ComponentType } from 'react'
import {
  ArrowDownIcon,
  ArrowUpIcon,
  CastIcon,
  CheckCircle2Icon,
  Clock3Icon,
  Grid2X2Icon,
  HelpCircleIcon,
  LaptopIcon,
  ListFilterIcon,
  MonitorSmartphoneIcon,
  MoreVerticalIcon,
  RefreshCwIcon,
  UserIcon,
} from 'lucide-react'

import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '#/components/ui/dropdown-menu'
import { Skeleton } from '#/components/ui/skeleton'
import type { ConsoleDeviceSummary } from '#/lib/mibo-api'
import { consoleSummaryQueryOptions, miboQueryKeys } from '#/lib/mibo-query'
import { cn } from '#/lib/utils'
import { useAuthStore } from '#/stores/auth-store'

type SortDirection = 'desc' | 'asc'
type ViewMode = 'cards' | 'compact'

export function DeviceManagementPanel() {
  const token = useAuthStore((state) => state.token)
  const queryClient = useQueryClient()
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')
  const [viewMode, setViewMode] = useState<ViewMode>('cards')
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>()
  const summaryQuery = useQuery({
    ...consoleSummaryQueryOptions(token ?? ''),
    enabled: Boolean(token),
  })

  const devices = [...(summaryQuery.data?.devices ?? [])].sort(
    (left, right) => {
      const leftTime = new Date(left.last_seen_at).getTime()
      const rightTime = new Date(right.last_seen_at).getTime()
      const diff = safeTime(rightTime) - safeTime(leftTime)
      return sortDirection === 'desc' ? diff : -diff
    },
  )
  const selectedDevice =
    devices.find((device) => device.id === selectedDeviceId) ?? devices[0]

  const refreshDevices = () => {
    if (!token) return
    void queryClient.invalidateQueries({
      queryKey: miboQueryKeys.consoleSummary(token),
    })
  }

  return (
    <div className="space-y-4">
      <section className="rounded-[1.5rem] border border-border/60 bg-card/70 p-4 shadow-sm backdrop-blur-sm">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
          <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
            <div className="inline-flex items-center gap-2 rounded-full border border-border/60 bg-background/70 px-3 py-1.5 text-foreground">
              <MonitorSmartphoneIcon className="size-4 text-emerald-500" />共{' '}
              {devices.length} 个设备
            </div>
            <button
              type="button"
              onClick={() =>
                setSortDirection((current) =>
                  current === 'desc' ? 'asc' : 'desc',
                )
              }
              className="inline-flex items-center gap-1.5 rounded-full border border-border/60 bg-background/50 px-3 py-1.5 transition-colors hover:bg-muted hover:text-foreground"
            >
              {sortDirection === 'desc' ? (
                <ArrowDownIcon className="size-3.5 text-emerald-500" />
              ) : (
                <ArrowUpIcon className="size-3.5 text-emerald-500" />
              )}
              上次活动日期
            </button>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              onClick={refreshDevices}
              disabled={!token}
            >
              <RefreshCwIcon className="size-4" />
              刷新
            </Button>
            <Button variant="outline" disabled title="播放到设备接入后启用">
              <CastIcon className="size-4" />
              播放到设备
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon" aria-label="更多设备操作">
                  <MoreVerticalIcon className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-52">
                <DropdownMenuLabel>更多操作</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setViewMode('cards')}>
                  <Grid2X2Icon className="size-4" />
                  卡片视图
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setViewMode('compact')}>
                  <ListFilterIcon className="size-4" />
                  紧凑视图
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem disabled>筛选在线设备</DropdownMenuItem>
                <DropdownMenuItem disabled>移除设备授权</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </section>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
        <section className="min-h-[420px] rounded-[1.5rem] border border-border/60 bg-gradient-to-br from-card/90 via-card/70 to-emerald-500/5 p-5 shadow-sm backdrop-blur-sm">
          <div className="mb-5 flex items-start justify-between gap-3">
            <div>
              <h3 className="text-base font-medium">客户端设备</h3>
              <p className="text-sm text-muted-foreground">
                点击设备卡片查看最近活动、使用用户和管理状态。
              </p>
            </div>
            <Button
              variant="ghost"
              size="icon-sm"
              className="rounded-full text-muted-foreground"
              title="设备数据来自控制台最近活动摘要"
            >
              <HelpCircleIcon className="size-4" />
              <span className="sr-only">帮助</span>
            </Button>
          </div>

          {summaryQuery.isLoading ? (
            <DeviceSkeleton />
          ) : devices.length === 0 ? (
            <EmptyDeviceState />
          ) : (
            <div
              className={cn(
                'grid gap-4',
                viewMode === 'cards'
                  ? 'sm:grid-cols-2 2xl:grid-cols-3'
                  : 'grid-cols-1',
              )}
            >
              {devices.map((device) => (
                <DeviceCard
                  key={device.id}
                  device={device}
                  selected={device.id === selectedDevice?.id}
                  compact={viewMode === 'compact'}
                  onSelect={() => setSelectedDeviceId(device.id)}
                />
              ))}
            </div>
          )}
        </section>

        <DeviceDetailCard device={selectedDevice} />
      </div>
    </div>
  )
}

function DeviceCard({
  device,
  selected,
  compact,
  onSelect,
}: {
  device: ConsoleDeviceSummary
  selected: boolean
  compact: boolean
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        'group border bg-background/80 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:border-emerald-500/50 hover:shadow-md focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-emerald-500/25',
        compact
          ? 'flex items-center gap-4 rounded-[1.1rem] p-3'
          : 'rounded-[1.35rem] p-5 text-center',
        selected
          ? 'border-emerald-500/60 ring-3 ring-emerald-500/15'
          : 'border-border/60',
      )}
    >
      <div
        className={cn(
          'mx-auto flex size-16 shrink-0 items-center justify-center rounded-2xl border border-border/60 bg-muted/70 text-muted-foreground transition-colors group-hover:text-emerald-500',
          compact && 'mx-0 size-12 rounded-xl',
          selected &&
            'border-emerald-500/40 bg-emerald-500/10 text-emerald-600',
        )}
      >
        <LaptopIcon className={compact ? 'size-5' : 'size-7'} />
      </div>

      <div className={cn('min-w-0', compact ? 'flex-1' : 'mt-4')}>
        <div className="truncate text-base font-semibold">{device.name}</div>
        <div className="mt-1 truncate text-sm text-muted-foreground">
          {device.client_type || device.state || 'Mibo Web'}
        </div>
        <div
          className={cn(
            'mt-2 flex items-center gap-1.5 text-sm text-muted-foreground',
            !compact && 'justify-center',
          )}
        >
          <UserIcon className="size-3.5" />
          <span className="truncate">
            {device.user || '未知用户'},{' '}
            {formatRelativeTime(device.last_seen_at)}
          </span>
        </div>
      </div>
    </button>
  )
}

function DeviceDetailCard({ device }: { device?: ConsoleDeviceSummary }) {
  if (!device) {
    return (
      <Card className="rounded-[1.5rem] border-border/60 bg-card/80 shadow-sm backdrop-blur-sm">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MonitorSmartphoneIcon className="size-4 text-emerald-500" />
            设备详情
          </CardTitle>
          <CardDescription>暂无可展示的客户端设备。</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 shadow-sm backdrop-blur-sm">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <MonitorSmartphoneIcon className="size-4 text-emerald-500" />
          设备详情
        </CardTitle>
        <CardDescription>
          当前为只读设备摘要，管理操作待设备授权接口接入。
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center gap-3 rounded-2xl border border-border/60 bg-background/70 p-3">
          <div className="flex size-12 items-center justify-center rounded-2xl bg-muted text-muted-foreground">
            <LaptopIcon className="size-5" />
          </div>
          <div className="min-w-0">
            <div className="truncate font-medium">{device.name}</div>
            <div className="truncate text-xs text-muted-foreground">
              {device.id}
            </div>
          </div>
        </div>

        <DetailRow
          icon={UserIcon}
          label="使用用户"
          value={device.user || '未知用户'}
        />
        <DetailRow
          icon={Clock3Icon}
          label="最近活动"
          value={formatRelativeTime(device.last_seen_at)}
        />
        <DetailRow
          icon={CheckCircle2Icon}
          label="设备状态"
          value={device.state || '最近有活动'}
        />
        <DetailRow
          icon={LaptopIcon}
          label="客户端版本"
          value={device.client_type || '客户端信息待上报'}
        />

        <div className="rounded-2xl border border-dashed border-border/70 bg-muted/30 p-3 text-sm leading-6 text-muted-foreground">
          当前设备列表复用控制台最近活动数据。后端提供设备授权、登出与详情接口后，可在此启用移除设备和完整客户端版本管理。
        </div>
      </CardContent>
    </Card>
  )
}

function DetailRow({
  icon: Icon,
  label,
  value,
}: {
  icon: ComponentType<{ className?: string }>
  label: string
  value: string
}) {
  return (
    <div className="flex items-center gap-3 rounded-2xl border border-border/50 bg-background/50 px-3 py-2.5">
      <div className="flex size-8 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
        <Icon className="size-4" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="truncate text-sm font-medium">{value}</div>
      </div>
    </div>
  )
}

function DeviceSkeleton() {
  return (
    <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-3">
      {Array.from({ length: 3 }).map((_, index) => (
        <Skeleton key={index} className="h-48 rounded-[1.35rem]" />
      ))}
    </div>
  )
}

function EmptyDeviceState() {
  return (
    <div className="flex min-h-[260px] flex-col items-center justify-center rounded-[1.35rem] border border-dashed border-border/70 bg-background/60 p-8 text-center">
      <MonitorSmartphoneIcon className="size-10 text-muted-foreground" />
      <h4 className="mt-4 text-base font-medium">暂无设备记录</h4>
      <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">
        当
        Web、移动端或播放客户端连接并产生活动后，会在这里显示设备名称、使用用户和最近活动时间。
      </p>
    </div>
  )
}

function safeTime(value: number) {
  return Number.isFinite(value) ? value : 0
}

function formatRelativeTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '未知'
  }

  const diffSeconds = Math.max(
    1,
    Math.floor((Date.now() - date.getTime()) / 1000),
  )
  if (diffSeconds < 60) {
    return `${diffSeconds}秒钟前`
  }

  const diffMinutes = Math.floor(diffSeconds / 60)
  if (diffMinutes < 60) {
    return `${diffMinutes}分钟前`
  }

  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) {
    return `${diffHours}小时前`
  }

  return `${Math.floor(diffHours / 24)}天前`
}
