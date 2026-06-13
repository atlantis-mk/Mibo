import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  CastIcon,
  CheckCircle2Icon,
  ClockIcon,
  DatabaseIcon,
  HardDriveIcon,
  MonitorSmartphoneIcon,
  PlayCircleIcon,
  RefreshCwIcon,
  ServerIcon,
  SettingsIcon,
  XCircleIcon,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import {
  type ConsoleActivityEvent,
  type ConsoleApplyUpdateResult,
  type ConsoleModuleStatus,
  type ConsolePrepareUpdateResult,
  type ConsoleQuickAction,
  type ConsoleRestartActionResult,
  type ConsoleStatus,
  type ConsoleSummary,
  type ConsoleUpdateStatus,
} from '@/lib/mibo-api'
import {
  consoleSummaryQueryOptions,
  createAuthedMiboApi,
  miboQueryKeys,
} from '@/lib/mibo-query'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { Skeleton } from '@/components/ui/skeleton'
import { ConfirmDialog } from '@/components/confirm-dialog'

const consoleRouteTargets = [
  '/settings/console',
  '/settings/library',
  '/settings/profile',
  '/settings/account',
  '/settings/appearance',
  '/settings/notifications',
  '/settings/display',
  '/settings/metadata',
] as const

type ConsoleRouteTarget = (typeof consoleRouteTargets)[number]

function isConsoleRouteTarget(route: string): route is ConsoleRouteTarget {
  return (consoleRouteTargets as readonly string[]).includes(route)
}

export default function ConsolePage({
  embedded = false,
  scrollable = true,
}: {
  embedded?: boolean
  scrollable?: boolean
}) {
  const token = useAuthStore((state) => state.auth.accessToken)
  const user = useAuthStore((state) => state.auth.user)
  const queryToken = token ?? 'guest'
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [pendingAction, setPendingAction] = useState<ConsoleQuickAction | null>(
    null
  )
  const [isRestarting, setIsRestarting] = useState(false)
  const summaryQuery = useQuery({
    ...consoleSummaryQueryOptions(queryToken),
    enabled: !!token,
    refetchInterval: isRestarting ? 3000 : false,
    retry: isRestarting ? 10 : undefined,
  })
  const actionMutation = useMutation({
    mutationFn: (action: ConsoleQuickAction) =>
      createAuthedMiboApi(queryToken).runConsoleAction(action.id),
    onSuccess: async (result, action) => {
      if (action.id === 'restart') {
        const restartResult = result as ConsoleRestartActionResult
        setIsRestarting(true)
        setPendingAction(null)
        toast.success(restartResult.message)
        return
      }
      if (action.id === 'prepare-update') {
        const updateResult = result as ConsolePrepareUpdateResult
        setPendingAction(null)
        toast.success(updateResult.message)
        await queryClient.invalidateQueries({
          queryKey: miboQueryKeys.consoleSummary(queryToken),
        })
        return
      }
      if (action.id === 'apply-update') {
        const updateResult = result as ConsoleApplyUpdateResult
        setIsRestarting(updateResult.restart_required)
        setPendingAction(null)
        toast.success(updateResult.message)
        return
      }
      toast.success(`${action.label} 已完成`)
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.consoleSummary(queryToken),
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const summary = summaryQuery.data
  const restartAction =
    summary?.quick_actions?.find((action) => action.id === 'restart') ?? null
  const prepareUpdateAction =
    summary?.quick_actions?.find((action) => action.id === 'prepare-update') ??
    null
  const applyUpdateAction =
    summary?.quick_actions?.find((action) => action.id === 'apply-update') ??
    null

  useEffect(() => {
    if (!isRestarting || !summaryQuery.isSuccess) {
      return
    }
    setIsRestarting(false)
    toast.success('服务器已重新连通')
  }, [isRestarting, summaryQuery.isSuccess])

  const runAction = (action: ConsoleQuickAction) => {
    if (action.disabled) return
    if (action.kind === 'route' && action.route) {
      if (!isConsoleRouteTarget(action.route)) {
        toast.error('该快捷入口尚未迁移到当前前端')
        return
      }
      void navigate({ to: action.route })
      return
    }
    if (action.kind !== 'mutation') return
    if (action.confirm) {
      setPendingAction(action)
      return
    }
    actionMutation.mutate(action)
  }

  return (
    <div
      className={cn(
        'relative flex-1 text-foreground',
        scrollable && 'overflow-y-auto',
        embedded ? 'bg-transparent' : 'min-h-screen bg-background'
      )}
    >
      <div className='pointer-events-none absolute inset-0 overflow-hidden'>
        <div className='absolute inset-x-0 top-0 h-72 bg-[radial-gradient(circle_at_top,rgba(212,162,76,0.2),transparent_58%)]' />
        <div className='absolute top-20 right-0 h-80 w-80 rounded-full bg-primary/6 blur-3xl' />
      </div>

      <div
        className={cn(
          'relative flex w-full flex-col gap-6',
          embedded ? 'p-0' : 'px-4 py-5 sm:px-6 lg:px-8'
        )}
      >
        {!embedded ? (
          <header className='rounded-[1.75rem] border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur-sm sm:p-5'>
            <div className='flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between'>
              <div className='flex items-start gap-3'>
                <SidebarTrigger className='mt-1' />
                <Button variant='ghost' size='icon' asChild>
                  <Link to='/'>
                    <ArrowLeftIcon className='size-4' />
                  </Link>
                </Button>
                <div>
                  <p className='text-xs tracking-[0.24em] text-primary/80 uppercase'>
                    Mibo Admin
                  </p>
                  <h1 className='mt-2 text-3xl font-semibold tracking-tight'>
                    系统控制台
                  </h1>
                </div>
              </div>

              <div className='flex flex-wrap items-center gap-2'>
                <Button variant='outline' disabled title='播放到设备尚未实现'>
                  <CastIcon className='size-4' />
                  投放
                </Button>
                <Button variant='outline' asChild>
                  <Link to='/settings/console'>
                    <SettingsIcon className='size-4' />
                    设置
                  </Link>
                </Button>
                <div className='rounded-full border border-border/60 bg-background/70 px-3 py-2 text-sm text-muted-foreground'>
                  {user?.username ?? '未登录'}
                </div>
              </div>
            </div>
          </header>
        ) : null}

        {summaryQuery.isPending ? <ConsoleSkeleton /> : null}
        {summaryQuery.isError ? (
          <Card className='border-destructive/30 bg-destructive/10'>
            <CardContent className='flex flex-col gap-3 p-6 sm:flex-row sm:items-center sm:justify-between'>
              <div>
                <h2 className='font-semibold text-destructive'>
                  {isRestarting ? '服务器正在重启' : '控制台数据加载失败'}
                </h2>
                <p className='text-sm text-destructive'>
                  {isRestarting
                    ? '正在等待服务重新启动，页面会在恢复连接后自动刷新。'
                    : summaryQuery.error.message}
                </p>
              </div>
              {!isRestarting ? (
                <Button onClick={() => void summaryQuery.refetch()}>
                  重试
                </Button>
              ) : null}
            </CardContent>
          </Card>
        ) : null}

        {summary ? (
          <div className='flex flex-col gap-4'>
            {isRestarting ? (
              <Alert className='border-amber-500/30 bg-amber-500/10 text-amber-950 dark:text-amber-100'>
                <AlertTriangleIcon className='size-4' />
                <AlertTitle>服务器正在重启</AlertTitle>
                <AlertDescription>
                  当前实例已收到重启请求，连接会短暂中断。控制台会自动轮询，待服务恢复后重新显示最新状态。
                </AlertDescription>
              </Alert>
            ) : null}
            <ConsoleHero
              summary={summary}
              username={user?.username}
              restartAction={restartAction}
              prepareUpdateAction={prepareUpdateAction}
              applyUpdateAction={applyUpdateAction}
              onRunAction={runAction}
              isActionRunning={actionMutation.isPending || isRestarting}
            />
            <MetricGrid summary={summary} />
            <section className='grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]'>
              <SystemSnapshot summary={summary} />
              <SideColumn activity={summary.activity ?? []} />
            </section>
            <QuickActions
              actions={(summary.quick_actions ?? []).filter(
                (action) =>
                  action.id !== 'restart' &&
                  action.id !== 'prepare-update' &&
                  action.id !== 'apply-update'
              )}
              isRunning={actionMutation.isPending}
              onRun={runAction}
            />
          </div>
        ) : null}
      </div>
      <ConfirmDialog
        open={!!pendingAction}
        onOpenChange={(open) => {
          if (!open) {
            setPendingAction(null)
          }
        }}
        title={pendingAction?.label ?? '确认执行'}
        desc={
          pendingAction?.id === 'restart'
            ? '服务器会优雅关闭当前实例并重新启动。执行后前端连接会短暂中断，控制台会自动尝试恢复。'
            : pendingAction?.id === 'apply-update'
              ? '将备份当前二进制、用已暂存的新版本替换它，然后退出当前进程等待重新启动。'
              : pendingAction?.id === 'prepare-update'
                ? '将从 GitHub Release 下载匹配当前平台的新版本，校验后暂存到更新目录。'
                : `确认执行“${pendingAction?.label ?? ''}”？`
        }
        confirmText={
          pendingAction?.id === 'restart'
            ? '立即重启'
            : pendingAction?.id === 'apply-update'
              ? '应用并退出'
              : '继续'
        }
        cancelBtnText='取消'
        destructive={pendingAction?.risk === 'danger'}
        isLoading={actionMutation.isPending}
        handleConfirm={() => {
          if (!pendingAction) return
          actionMutation.mutate(pendingAction)
        }}
      />
    </div>
  )
}

function ConsoleHero({
  summary,
  username,
  restartAction,
  prepareUpdateAction,
  applyUpdateAction,
  onRunAction,
  isActionRunning,
}: {
  summary: ConsoleSummary
  username?: string
  restartAction: ConsoleQuickAction | null
  prepareUpdateAction: ConsoleQuickAction | null
  applyUpdateAction: ConsoleQuickAction | null
  onRunAction: (action: ConsoleQuickAction) => void
  isActionRunning: boolean
}) {
  const accessAddresses = (summary.access.addresses ?? []).slice(0, 3)
  const health = summarizeHealth(summary)
  const quickFacts = [
    {
      label: '运行时长',
      value: formatDuration(summary.server.uptime_seconds),
    },
    {
      label: '数据库',
      value: summary.server.database_driver,
    },
    {
      label: '存储',
      value: summary.server.storage_provider || '未知',
    },
    {
      label: '操作人',
      value: username ?? '未登录',
    },
  ]

  return (
    <section className='grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]'>
      <Card className='overflow-hidden rounded-[1.6rem] border-primary/15 bg-[linear-gradient(135deg,rgba(212,162,76,0.18),rgba(17,17,17,0.96)_40%,rgba(17,17,17,0.92))] py-0 text-white shadow-[0_20px_80px_rgba(0,0,0,0.24)]'>
        <CardContent className='px-5 py-5 sm:px-6 sm:py-6'>
          <div className='flex flex-col gap-6'>
            <div className='flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between'>
              <div className='max-w-2xl'>
                <p className='text-xs tracking-[0.26em] text-primary/80 uppercase'>
                  Operations Center
                </p>
                <h2 className='mt-2 text-2xl font-semibold tracking-tight sm:text-3xl'>
                  {summary.server.service}
                </h2>
              </div>

              <div className='flex items-center gap-2 self-start'>
                <div className='flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-xs text-white/80'>
                  <ServerIcon className='size-3.5 text-primary' />
                  {summary.server.name}
                  <span className='text-white/40'>·</span>
                  {summary.server.version || '未知版本'}
                </div>
                {restartAction ? (
                  <Button
                    type='button'
                    size='icon'
                    variant='ghost'
                    disabled={restartAction.disabled || isActionRunning}
                    onClick={() => onRunAction(restartAction)}
                    className='size-9 rounded-full border border-amber-400/30 bg-amber-500/12 text-amber-200 transition-colors hover:bg-amber-500/18 hover:text-amber-100 disabled:opacity-50'
                    title={restartAction.label}
                  >
                    <RefreshCwIcon
                      className={cn(
                        'size-4',
                        isActionRunning && 'animate-spin'
                      )}
                    />
                    <span className='sr-only'>{restartAction.label}</span>
                  </Button>
                ) : null}
              </div>
            </div>

            <div className='grid gap-3 sm:grid-cols-3'>
              <HeroStat
                label='系统健康'
                value={health.headline}
                detail={`${health.errorCount} 错误 · ${health.warningCount} 警告`}
              />
              <HeroStat
                label='活跃任务'
                value={String(summary.media.active_jobs)}
                detail={`${summary.media.failed_jobs} 个失败任务`}
              />
              <HeroStat
                label='可用计划'
                value={`${summary.media.enabled_schedules}/${summary.media.schedules}`}
                detail='已启用 / 全部计划'
              />
            </div>

            <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-4'>
              {quickFacts.map((fact) => (
                <div
                  key={fact.label}
                  className='rounded-xl border border-white/10 bg-white/5 px-3 py-3'
                >
                  <p className='text-[11px] tracking-[0.16em] text-white/55 uppercase'>
                    {fact.label}
                  </p>
                  <p className='mt-1.5 text-sm font-medium text-white/92'>
                    {fact.value}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className='rounded-[1.35rem] border-border/60 bg-card/90 shadow-sm backdrop-blur-sm'>
        <CardHeader>
          <CardTitle className='flex items-center justify-between gap-3 text-base'>
            <span>系统状态</span>
            <StatusPill status={summary.server.status} />
          </CardTitle>
        </CardHeader>
        <CardContent className='grid gap-3'>
          <div className='rounded-xl border border-border/60 bg-background/60 px-4 py-3'>
            <div className='flex items-center justify-between gap-3'>
              <div>
                <p className='text-xs tracking-[0.14em] text-muted-foreground uppercase'>
                  接入信息
                </p>
                <p className='mt-1 text-sm font-medium'>
                  API :{summary.server.port || 'n/a'}
                </p>
              </div>
              <div className='flex flex-wrap gap-2'>
                <StatusPill status={summary.server.status} />
                <StatusChip
                  label='地址'
                  value={summary.access.addresses?.length ?? 0}
                  tone='neutral'
                />
              </div>
            </div>
          </div>

          <UpdateCard
            summary={summary}
            prepareUpdateAction={prepareUpdateAction}
            applyUpdateAction={applyUpdateAction}
            onRunAction={onRunAction}
            isActionRunning={isActionRunning}
          />

          <div className='grid gap-2'>
            <h3 className='text-sm font-medium'>接入地址</h3>
            {accessAddresses.length === 0 ? (
              <div className='rounded-xl border border-dashed border-border/60 bg-background/20 px-4 py-3 text-xs text-muted-foreground'>
                暂无可展示的接入地址。
              </div>
            ) : (
              accessAddresses.map((address) => (
                <div
                  key={`${address.kind}-${address.url ?? address.status}`}
                  className='rounded-xl border border-border/60 bg-background/50 px-4 py-3'
                >
                  <div className='flex items-start justify-between gap-3'>
                    <div className='min-w-0'>
                      <p className='text-xs font-medium'>{address.label}</p>
                      <p className='mt-1 font-mono text-[11px] break-all text-muted-foreground'>
                        {address.url ?? address.message ?? '未配置'}
                      </p>
                    </div>
                    <StatusPill status={address.status} />
                  </div>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>
    </section>
  )
}

function UpdateCard({
  summary,
  prepareUpdateAction,
  applyUpdateAction,
  onRunAction,
  isActionRunning,
}: {
  summary: ConsoleSummary
  prepareUpdateAction: ConsoleQuickAction | null
  applyUpdateAction: ConsoleQuickAction | null
  onRunAction: (action: ConsoleQuickAction) => void
  isActionRunning: boolean
}) {
  const update = summary.server.update
  const actionUrl = update?.release_url
  const title = updateTitle(summary.server.update_status)
  const detail = updateDetail(summary)
  const canPrepareUpdate =
    summary.server.update_status === 'update_available' &&
    !update?.staged &&
    !!prepareUpdateAction &&
    !prepareUpdateAction.disabled
  const canApplyUpdate =
    summary.server.update_status === 'update_available' &&
    !!update?.staged &&
    !!applyUpdateAction &&
    !applyUpdateAction.disabled

  return (
    <div className='rounded-xl border border-border/60 bg-background/60 px-4 py-3'>
      <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
        <div className='min-w-0'>
          <div className='flex flex-wrap items-center gap-2'>
            <p className='text-xs tracking-[0.14em] text-muted-foreground uppercase'>
              版本更新
            </p>
            <UpdateStatusPill status={summary.server.update_status} />
          </div>
          <p className='mt-1 text-sm font-medium'>{title}</p>
          <p className='mt-1 text-xs break-words text-muted-foreground'>
            {detail}
          </p>
        </div>
        {canApplyUpdate ? (
          <Button
            type='button'
            size='xs'
            variant='outline'
            className='self-start'
            disabled={isActionRunning}
            onClick={() => onRunAction(applyUpdateAction)}
          >
            应用更新
          </Button>
        ) : canPrepareUpdate ? (
          <Button
            type='button'
            size='xs'
            variant='outline'
            className='self-start'
            disabled={isActionRunning}
            onClick={() => onRunAction(prepareUpdateAction)}
          >
            准备更新
          </Button>
        ) : actionUrl ? (
          <Button asChild size='xs' variant='outline' className='self-start'>
            <a href={actionUrl} target='_blank' rel='noreferrer'>
              查看 Release
            </a>
          </Button>
        ) : null}
      </div>
    </div>
  )
}

function SystemSnapshot({ summary }: { summary: ConsoleSummary }) {
  const modules = summary.health.modules ?? []
  const errorCount = modules.filter(
    (module) => module.status === 'error'
  ).length
  const warningCount = modules.filter(
    (module) => module.status === 'warning'
  ).length
  const warnings = summary.warnings ?? []
  const sections: Array<{
    label: string
    value: string
    status: ConsoleStatus | 'available'
    icon: typeof DatabaseIcon
  }> = [
    {
      label: '数据库',
      value: summary.server.database_driver,
      ...summary.health.database,
      icon: DatabaseIcon,
    },
    {
      label: '存储',
      value: summary.server.storage_provider || '未知',
      ...summary.health.storage,
      icon: HardDriveIcon,
    },
    {
      label: '运行时长',
      value: formatDuration(summary.server.uptime_seconds),
      status: summary.server.status,
      icon: ClockIcon,
    },
    {
      label: '计划任务',
      value: `${summary.media.enabled_schedules}/${summary.media.schedules}`,
      status: summary.media.enabled_schedules > 0 ? 'available' : 'unknown',
      icon: RefreshCwIcon,
    },
  ]
  return (
    <Card className='rounded-[1.35rem] border-border/60 bg-card/90 shadow-sm backdrop-blur-sm'>
      <CardHeader>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between'>
          <div>
            <CardTitle className='text-base'>系统快照</CardTitle>
          </div>
          <div className='flex flex-wrap gap-2 text-xs'>
            <StatusChip label='模块错误' value={errorCount} tone='error' />
            <StatusChip label='模块警告' value={warningCount} tone='warning' />
            <StatusChip
              label='模块总数'
              value={modules.length}
              tone='neutral'
            />
          </div>
        </div>
      </CardHeader>
      <CardContent className='grid gap-4'>
        {warnings.length > 0 ? (
          <div className='rounded-xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-100'>
            <p className='font-medium text-amber-200'>部分数据不可用</p>
            <p className='mt-1 text-amber-50/85'>
              {warnings
                .map((warning) => `${warning.section}: ${warning.message}`)
                .join('；')}
            </p>
          </div>
        ) : null}

        <div className='grid gap-3 sm:grid-cols-2'>
          {sections.map((section) => {
            const Icon = section.icon
            return (
              <div
                key={section.label}
                className='rounded-xl border border-border/60 bg-background/50 px-4 py-3'
              >
                <div className='flex items-center justify-between gap-3'>
                  <div className='flex items-center gap-2'>
                    <Icon className='size-4 text-primary' />
                    <span className='text-xs font-medium'>{section.label}</span>
                  </div>
                  <StatusPill status={section.status} />
                </div>
                <p className='mt-1.5 truncate text-[11px] text-muted-foreground'>
                  {section.value || '未返回额外说明'}
                </p>
              </div>
            )
          })}
        </div>

        <div className='grid gap-2'>
          <h3 className='text-sm font-medium'>模块状态</h3>
          {modules.length === 0 ? (
            <div className='rounded-xl border border-dashed border-border/60 bg-background/20 px-4 py-3 text-xs text-muted-foreground'>
              暂无模块级别健康信息。
            </div>
          ) : (
            <div className='grid gap-3 sm:grid-cols-2'>
              {modules.slice(0, 4).map((module) => (
                <ModuleRow key={module.name} module={module} />
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function MetricGrid({ summary }: { summary: ConsoleSummary }) {
  const metrics = [
    ['媒体库', summary.media.libraries, DatabaseIcon],
    ['媒体源', summary.media.media_sources, HardDriveIcon],
    ['电影', summary.media.movies, PlayCircleIcon],
    ['剧集', summary.media.series, MonitorSmartphoneIcon],
    ['任务', summary.media.active_jobs, RefreshCwIcon],
    ['失败', summary.media.failed_jobs, XCircleIcon],
    ['待复核', summary.media.ingest?.review_required ?? 0, AlertTriangleIcon],
  ] as const

  return (
    <section className='grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-7'>
      {metrics.map(([label, value, Icon]) => (
        <Card
          key={label}
          className='rounded-[1.15rem] border-border/60 bg-card/85 py-4 shadow-sm backdrop-blur-sm'
        >
          <CardContent className='px-4'>
            <div className='flex items-start justify-between gap-3'>
              <div>
                <p className='text-[10px] tracking-[0.16em] text-muted-foreground uppercase'>
                  {label}
                </p>
                <p className='mt-2 text-2xl font-semibold tracking-tight'>
                  {value}
                </p>
              </div>
              <div className='rounded-lg border border-primary/15 bg-primary/10 p-2 text-primary'>
                <Icon className='size-4' />
              </div>
            </div>
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
  const groupedActions = [
    {
      key: 'safe',
      title: '安全动作',
      items: actions.filter((action) => action.risk === 'safe'),
    },
    {
      key: 'expensive',
      title: '昂贵动作',
      items: actions.filter((action) => action.risk === 'expensive'),
    },
    {
      key: 'danger',
      title: '高风险动作',
      items: actions.filter((action) => action.risk === 'danger'),
    },
  ].filter((group) => group.items.length > 0)

  return (
    <Card className='rounded-[1.35rem] border-border/60 bg-card/90 shadow-sm backdrop-blur-sm'>
      <CardHeader>
        <CardTitle className='text-base'>执行面板</CardTitle>
      </CardHeader>
      <CardContent className='grid gap-4'>
        {actions.length === 0 ? (
          <div className='rounded-xl border border-dashed border-border/60 bg-background/20 px-4 py-3 text-sm text-muted-foreground'>
            暂无快捷操作。
          </div>
        ) : (
          groupedActions.map((group) => (
            <div key={group.key} className='grid gap-2.5'>
              <div className='flex items-end justify-between gap-3'>
                <div>
                  <p className='font-mono text-[10px] tracking-[0.16em] text-muted-foreground uppercase'>
                    {group.title}
                  </p>
                </div>
                <span className='rounded-full border border-border/60 px-2.5 py-1 text-xs text-muted-foreground'>
                  {group.items.length} 项
                </span>
              </div>

              <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
                {group.items.map((action) => (
                  <button
                    key={action.id}
                    type='button'
                    disabled={action.disabled || isRunning}
                    onClick={() => onRun(action)}
                    className={cn(
                      'group rounded-xl border border-border/60 bg-background/45 p-4 text-left transition-all duration-200 hover:border-primary/40 hover:bg-accent/55',
                      action.risk === 'danger' &&
                        !action.disabled &&
                        'border-destructive/30 bg-destructive/5 hover:border-destructive/50 hover:bg-destructive/10',
                      action.disabled &&
                        'cursor-not-allowed bg-muted/40 text-muted-foreground opacity-65 hover:border-border/60 hover:bg-muted/40'
                    )}
                  >
                    <div className='flex items-start justify-between gap-3'>
                      <div className='min-w-0'>
                        <div className='flex items-center gap-2'>
                          <RiskBadge risk={action.risk} />
                          <p className='text-sm font-medium'>{action.label}</p>
                        </div>
                      </div>
                      <ArrowRightIcon className='mt-0.5 size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-hover:translate-x-0.5' />
                    </div>
                  </button>
                ))}
              </div>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  )
}

function SideColumn({ activity }: { activity: ConsoleActivityEvent[] }) {
  return <ActivityTimeline events={activity} />
}

function ActivityTimeline({ events }: { events: ConsoleActivityEvent[] }) {
  return (
    <Card className='rounded-[1.35rem] border-border/60 bg-card/90 shadow-sm backdrop-blur-sm'>
      <CardHeader>
        <CardTitle className='text-base'>最近活动</CardTitle>
      </CardHeader>
      <CardContent className='grid gap-3'>
        {events.length === 0 ? (
          <div className='rounded-xl border border-dashed border-border/60 bg-background/20 px-4 py-3 text-sm text-muted-foreground'>
            暂无活动。播放、扫描和系统事件会显示在这里。
          </div>
        ) : (
          events.slice(0, 4).map((event) => (
            <div
              key={event.id}
              className='rounded-xl border border-border/50 bg-background/45 px-4 py-3'
            >
              <div className='flex items-start gap-3'>
                <SeverityIcon severity={event.severity} />
                <div className='min-w-0 flex-1'>
                  <div className='flex items-start justify-between gap-3'>
                    <p className='line-clamp-1 text-sm font-medium'>
                      {event.message}
                    </p>
                    <time className='shrink-0 text-xs text-muted-foreground'>
                      {formatDate(event.timestamp)}
                    </time>
                  </div>
                  <p className='mt-1 truncate text-xs text-muted-foreground'>
                    {[event.user, event.device, event.media_title]
                      .filter(Boolean)
                      .join(' · ') || event.type}
                  </p>
                </div>
              </div>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  )
}

function ConsoleSkeleton() {
  return (
    <div className='grid gap-4'>
      <Skeleton className='h-72 rounded-[1.9rem]' />
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-7'>
        {Array.from({ length: 7 }).map((_, index) => (
          <Skeleton key={index} className='h-28 rounded-[1.3rem]' />
        ))}
      </div>
      <div className='grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]'>
        <Skeleton className='h-[32rem] rounded-[1.7rem]' />
        <div className='grid gap-4'>
          <Skeleton className='h-72 rounded-[1.6rem]' />
          <Skeleton className='h-52 rounded-[1.6rem]' />
        </div>
      </div>
      <Skeleton className='h-72 rounded-[1.7rem]' />
    </div>
  )
}

function HeroStat({
  label,
  value,
  detail,
}: {
  label: string
  value: string
  detail: string
}) {
  return (
    <div className='rounded-[1rem] border border-white/10 bg-black/20 p-4 backdrop-blur-sm'>
      <p className='font-mono text-[10px] tracking-[0.16em] text-white/55 uppercase'>
        {label}
      </p>
      <p className='mt-2 text-xl font-semibold tracking-tight text-white'>
        {value}
      </p>
      <p className='mt-1 text-[11px] text-white/65'>{detail}</p>
    </div>
  )
}

function ModuleRow({ module }: { module: ConsoleModuleStatus }) {
  return (
    <div className='flex items-start justify-between gap-3 rounded-xl border border-border/50 bg-background/45 px-4 py-3'>
      <div className='min-w-0'>
        <p className='text-sm font-medium'>{module.name}</p>
        <p className='mt-1 text-xs text-muted-foreground'>
          {module.message || '未返回额外说明'}
        </p>
      </div>
      <StatusPill status={module.status} />
    </div>
  )
}

function StatusChip({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: 'error' | 'warning' | 'neutral'
}) {
  return (
    <span
      className={cn(
        'rounded-full px-2.5 py-1',
        tone === 'error' && 'bg-destructive/12 text-destructive',
        tone === 'warning' && 'bg-amber-500/15 text-amber-200',
        tone === 'neutral' && 'bg-background/45 text-muted-foreground'
      )}
    >
      {label} {value}
    </span>
  )
}

function RiskBadge({ risk }: { risk: ConsoleQuickAction['risk'] }) {
  const copy =
    risk === 'danger' ? '高风险' : risk === 'expensive' ? '昂贵' : '安全'

  return (
    <span
      className={cn(
        'rounded-full px-2 py-1 text-[10px] tracking-[0.16em] uppercase',
        risk === 'danger' && 'bg-destructive/15 text-destructive',
        risk === 'expensive' && 'bg-primary/15 text-primary',
        risk === 'safe' && 'bg-emerald-500/15 text-emerald-300'
      )}
    >
      {copy}
    </span>
  )
}

function StatusPill({ status }: { status: ConsoleStatus | 'available' }) {
  return (
    <span
      className={cn(
        'rounded-full px-2.5 py-1 text-[10px] font-medium whitespace-nowrap',
        statusClass(status)
      )}
    >
      {statusLabel(status)}
    </span>
  )
}

function UpdateStatusPill({ status }: { status: ConsoleUpdateStatus }) {
  return (
    <span
      className={cn(
        'rounded-full px-2.5 py-1 text-[10px] font-medium whitespace-nowrap',
        updateStatusClass(status)
      )}
    >
      {updateStatusLabel(status)}
    </span>
  )
}

function SeverityIcon({
  severity,
}: {
  severity: ConsoleActivityEvent['severity']
}) {
  if (severity === 'error') {
    return <XCircleIcon className='mt-0.5 size-4 text-destructive' />
  }
  if (severity === 'warning') {
    return <AlertTriangleIcon className='mt-0.5 size-4 text-amber-300' />
  }
  return <CheckCircle2Icon className='mt-0.5 size-4 text-primary' />
}

function summarizeHealth(summary: ConsoleSummary) {
  const statuses = [
    summary.server.status,
    summary.health.database.status,
    summary.health.storage.status,
    ...(summary.health.modules ?? []).map((module) => module.status),
  ]
  const errorCount = statuses.filter((status) => status === 'error').length
  const warningCount = statuses.filter((status) => status === 'warning').length

  if (errorCount > 0) {
    return {
      headline: '需要处理',
      errorCount,
      warningCount,
    }
  }
  if (warningCount > 0) {
    return {
      headline: '存在警告',
      errorCount,
      warningCount,
    }
  }
  return {
    headline: '运行稳定',
    errorCount,
    warningCount,
  }
}

function statusClass(status: string) {
  if (status === 'ok' || status === 'available') {
    return 'bg-primary/10 text-primary'
  }
  if (status === 'warning' || status === 'unknown') {
    return 'bg-amber-500/15 text-amber-200'
  }
  if (status === 'error') {
    return 'bg-destructive/10 text-destructive'
  }
  return 'bg-muted text-muted-foreground'
}

function updateStatusClass(status: ConsoleUpdateStatus) {
  if (status === 'up_to_date') {
    return 'bg-primary/10 text-primary'
  }
  if (status === 'update_available') {
    return 'bg-amber-500/15 text-amber-200'
  }
  if (status === 'check_failed') {
    return 'bg-destructive/10 text-destructive'
  }
  return 'bg-muted text-muted-foreground'
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    ok: '正常',
    available: '可用',
    warning: '警告',
    error: '错误',
    unknown: '未知',
    unavailable: '不可用',
    not_configured: '未配置',
  }

  return labels[status] ?? status
}

function updateStatusLabel(status: ConsoleUpdateStatus) {
  const labels: Record<ConsoleUpdateStatus, string> = {
    disabled: '未启用',
    unknown: '未知',
    up_to_date: '已是最新',
    update_available: '有更新',
    check_failed: '检查失败',
  }

  return labels[status] ?? status
}

function updateTitle(status: ConsoleUpdateStatus) {
  const titles: Record<ConsoleUpdateStatus, string> = {
    disabled: '更新检查未启用',
    unknown: '暂时无法判断更新状态',
    up_to_date: '当前版本已是最新',
    update_available: '发现可用新版本',
    check_failed: '更新检查失败',
  }

  return titles[status] ?? '更新状态未知'
}

function updateDetail(summary: ConsoleSummary) {
  const update = summary.server.update
  const current =
    update?.current_version || summary.server.version || '未知版本'
  const latest = update?.latest_version || '未知版本'

  if (summary.server.update_status === 'update_available') {
    if (update?.staged) {
      return `新版本 ${latest} 已暂存到 ${update.staged.staged_directory}。应用更新会备份当前二进制并退出当前进程，重新启动后加载新版本。`
    }
    return `当前 ${current}，最新 ${latest}。${update?.asset_name ? `匹配资产：${update.asset_name}` : '未找到当前平台的发布资产，可查看 Release 页面。'}`
  }
  if (summary.server.update_status === 'up_to_date') {
    return `当前 ${current}，最新 ${latest}。`
  }
  if (summary.server.update_status === 'check_failed') {
    return update?.message || '无法连接 GitHub Release 服务。'
  }
  if (summary.server.update_status === 'disabled') {
    return '可通过服务端环境变量重新启用更新检查。'
  }
  return update?.message || `当前 ${current}，最新版本暂不可用。`
}

function formatDuration(seconds: number) {
  if (!Number.isFinite(seconds) || seconds <= 0) return '未知'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) return `${days} 天 ${hours} 小时`
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`
  return `${minutes} 分钟`
}

function formatDate(value: string) {
  const date = new Date(value)

  if (Number.isNaN(date.getTime())) return '未知时间'
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
