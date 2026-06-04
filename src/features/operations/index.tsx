import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangleIcon,
  ArrowUpRightIcon,
  CheckCircle2Icon,
  Clock3Icon,
  DatabaseIcon,
  ListTodoIcon,
  LoaderCircleIcon,
  RefreshCwIcon,
  ShieldAlertIcon,
  WaypointsIcon,
  WrenchIcon,
} from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import type {
  OperationsPipelineStage,
  OperationsRecommendedAction,
  OperationsStatus,
  OperationsTask,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  miboQueryKeys,
  operationsOverviewQueryOptions,
  operationsPipelineQueryOptions,
  operationsTasksQueryOptions,
} from '@/lib/mibo-query'
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
import { MetadataReviewDialog } from '@/features/operations/metadata-review-dialog'

export default function OperationsCenter({
  interactive = false,
}: {
  interactive?: boolean
}) {
  const token = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const role = useAuthStore((state) => state.auth.user?.role)
  const queryToken = token ?? 'guest'
  const queryClient = useQueryClient()
  const isAdmin = role === 'admin'
  const [optimisticActionLocks, setOptimisticActionLocks] = useState<string[]>(
    []
  )

  const overviewQuery = useQuery({
    ...operationsOverviewQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const tasksQuery = useQuery({
    ...operationsTasksQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const pipelineQuery = useQuery({
    ...operationsPipelineQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })

  const actionMutation = useMutation({
    mutationFn: async (actionId: string) => {
      if (!token) throw new Error('当前未登录，无法执行运营操作。')
      return createAuthedMiboApi(token).executeOperationsAction(actionId)
    },
    onMutate: async (actionId) => {
      setOptimisticActionLocks((current) =>
        current.includes(actionId) ? current : [...current, actionId]
      )
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.operationsOverview(queryToken),
      })
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.operationsTasks(queryToken),
      })
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.operationsPipeline(queryToken),
      })
      await queryClient.invalidateQueries({ queryKey: ['home'] })
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.consoleSummary(queryToken),
      })
    },
  })

  const activeTasks = useMemo(
    () =>
      (tasksQuery.data ?? []).filter(
        (task) => (task.lifecycle_status ?? 'active') !== 'resolved'
      ),
    [tasksQuery.data]
  )
  const resolvedTasks = useMemo(
    () =>
      (tasksQuery.data ?? []).filter(
        (task) => task.lifecycle_status === 'resolved'
      ),
    [tasksQuery.data]
  )
  const lockedActionIds = useMemo(() => {
    const activeActionIds = new Set<string>()
    for (const task of activeTasks) {
      for (const action of task.recommended_actions) {
        if (action.id) {
          activeActionIds.add(action.id)
        }
      }
    }
    return optimisticActionLocks.filter((actionId) =>
      activeActionIds.has(actionId)
    )
  }, [activeTasks, optimisticActionLocks])
  const priorityTasks = useMemo(() => activeTasks.slice(0, 6), [activeTasks])
  const pipelineStages = useMemo(
    () => pipelineQuery.data?.stages ?? [],
    [pipelineQuery.data?.stages]
  )
  const blockedStageCount = useMemo(
    () =>
      pipelineStages.filter(
        (stage) => stage.status === 'blocked' || stage.status === 'degraded'
      ).length,
    [pipelineStages]
  )
  const totalPendingPipelineItems = useMemo(
    () =>
      pipelineStages.reduce(
        (sum, stage) => sum + stage.pending + stage.running,
        0
      ),
    [pipelineStages]
  )
  const primaryFocusTask = priorityTasks[0] ?? null
  const [metadataReviewTask, setMetadataReviewTask] =
    useState<OperationsTask | null>(null)

  const isLoading =
    overviewQuery.isLoading || tasksQuery.isLoading || pipelineQuery.isLoading
  const error =
    overviewQuery.error || tasksQuery.error || pipelineQuery.error || null

  return (
    <div className='space-y-8'>
      <section className='grid gap-4 xl:grid-cols-[minmax(0,1.5fr)_minmax(320px,0.9fr)]'>
        <Card className='overflow-hidden border-border/60 bg-card pt-0 shadow-sm'>
          <CardContent className='relative p-0'>
            <div className='absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(245,158,11,0.18),transparent_40%),radial-gradient(circle_at_top_right,rgba(14,165,233,0.14),transparent_34%)]' />
            <div className='relative space-y-6 px-6 py-6 sm:px-7'>
              <div className='flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between'>
                <div className='space-y-3'>
                  <Badge
                    variant='outline'
                    className='w-fit rounded-full border-amber-500/30 bg-amber-500/10 px-3 py-1 text-amber-700'
                  >
                    运营总览
                  </Badge>
                  <div>
                    <h1 className='flex items-center gap-2 text-2xl font-semibold tracking-tight'>
                      <ShieldAlertIcon className='size-5 text-amber-500' />
                      媒体库运营中心
                    </h1>
                    <p className='mt-2 max-w-2xl text-sm leading-6 text-muted-foreground'>
                      把连接异常、扫描阻塞、整理积压和人工确认放进同一条工作视图，先处理最影响首页可见性和入库节奏的问题。
                    </p>
                  </div>
                </div>
                <div className='flex items-center gap-2 self-start'>
                  <StatusBadge status={overviewQuery.data?.status ?? 'healthy'} />
                  <Badge variant='outline' className='rounded-full px-3 py-1'>
                    {activeTasks.length} 项待处理
                  </Badge>
                </div>
              </div>

              <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
                <MetricCard
                  label='受影响媒体库'
                  value={overviewQuery.data?.affected_libraries ?? 0}
                  tone='warning'
                />
                <MetricCard
                  label='受影响文件'
                  value={overviewQuery.data?.affected_files ?? 0}
                  tone='neutral'
                />
                <MetricCard
                  label='待人工确认'
                  value={overviewQuery.data?.pending_reviews ?? 0}
                  tone='warning'
                />
                <MetricCard
                  label='失败任务'
                  value={overviewQuery.data?.failed_jobs ?? 0}
                  tone='danger'
                />
              </div>

              <div className='grid gap-3 lg:grid-cols-3'>
                <InsightCard
                  icon={ListTodoIcon}
                  label='当前重心'
                  value={primaryFocusTask?.title ?? '暂无高优先任务'}
                  description={
                    primaryFocusTask?.summary ??
                    '媒体库同步、整理和人工确认都处于相对稳定状态。'
                  }
                />
                <InsightCard
                  icon={Clock3Icon}
                  label='人工确认压力'
                  value={`${overviewQuery.data?.pending_reviews ?? 0} 项`}
                  description='越早清理 review 积压，后续扫描与整理结果越稳定。'
                />
                <InsightCard
                  icon={WaypointsIcon}
                  label='流水线积压'
                  value={`${totalPendingPipelineItems} 项`}
                  description={`当前有 ${blockedStageCount} 个阶段处于受影响或阻断状态。`}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <div className='grid gap-4'>
          <Card className='border-border/60 bg-card/90 shadow-sm'>
            <CardHeader>
              <CardTitle className='flex items-center gap-2 text-lg'>
                <DatabaseIcon className='size-4 text-sky-500' />
                状态分区
              </CardTitle>
              <CardDescription>
                从分区角度判断问题集中在连接、扫描还是人工治理。
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-3'>
              {(overviewQuery.data?.sections ?? []).map((section) => (
                <div
                  key={section.key}
                  className='flex items-start justify-between gap-3 rounded-xl border border-border/60 bg-background/75 px-3 py-3'
                >
                  <div className='min-w-0'>
                    <div className='font-medium'>{section.label}</div>
                    <div className='mt-1 text-sm leading-5 text-muted-foreground'>
                      {section.description}
                    </div>
                  </div>
                  <div className='shrink-0 text-right'>
                    <StatusBadge status={section.status} compact />
                    <div className='mt-2 text-xl font-semibold'>
                      {section.count}
                    </div>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card className='border-border/60 bg-card/90 shadow-sm'>
            <CardHeader>
              <CardTitle className='text-lg'>处理节奏</CardTitle>
              <CardDescription>
                建议先清理阻断和错误任务，再处理人工确认，最后回补低优先积压。
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-3'>
              <div className='rounded-xl border border-border/60 bg-background/75 px-4 py-3 text-sm text-muted-foreground'>
                当前概览页更适合判断优先级和定位问题，批量动作建议在治理工作台执行。
              </div>
              {!interactive ? (
                <Button asChild className='w-full'>
                  <Link to='/settings/operations/manage'>
                    <WrenchIcon className='size-4' />
                    打开治理工作台
                  </Link>
                </Button>
              ) : (
                <div className='rounded-xl border border-border/60 bg-background/75 px-4 py-3 text-sm text-muted-foreground'>
                  当前已进入可执行模式，可以直接逐条处理任务或打开元数据确认弹窗。
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </section>

      {isLoading ? (
        <Card className='border-border/60 bg-card/85'>
          <CardContent className='flex items-center gap-3 p-6 text-sm text-muted-foreground'>
            <LoaderCircleIcon className='size-4 animate-spin' />
            正在加载媒体库运营状态
          </CardContent>
        </Card>
      ) : null}

      {error ? (
        <Card className='border-destructive/30 bg-destructive/5'>
          <CardContent className='flex items-center justify-between gap-4 p-6'>
            <div>
              <div className='font-medium text-destructive'>
                运营数据加载失败
              </div>
              <div className='mt-1 text-sm text-muted-foreground'>
                {error.message}
              </div>
            </div>
            <Button
              variant='outline'
              onClick={() => {
                void overviewQuery.refetch()
                void tasksQuery.refetch()
                void pipelineQuery.refetch()
              }}
            >
              重试
            </Button>
          </CardContent>
        </Card>
      ) : null}

      {!isLoading && !error ? (
        <>
          <section className='grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]'>
            <Card className='border-border/60 bg-card/90 shadow-sm'>
              <CardHeader className='gap-3'>
                <div className='flex items-center justify-between gap-4'>
                  <div>
                    <CardTitle className='text-xl tracking-tight'>
                      优先处理
                    </CardTitle>
                    <CardDescription className='mt-1'>
                      先处理会影响刷新、首页可见性和人工治理节奏的问题。
                    </CardDescription>
                  </div>
                  <Badge variant='outline' className='rounded-full px-3 py-1'>
                    {activeTasks.length} 项
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className='space-y-4'>
                {priorityTasks.length === 0 ? (
                  <div className='rounded-xl border border-emerald-500/30 bg-emerald-500/5 p-6'>
                    <div className='flex items-center gap-3'>
                      <CheckCircle2Icon className='size-5 text-emerald-600' />
                      <div>
                        <div className='font-medium'>当前没有待处理事项</div>
                        <div className='text-sm text-muted-foreground'>
                          媒体库同步、整理和人工确认都处于相对稳定状态。
                        </div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className='grid gap-4'>
                    {priorityTasks.map((task) => (
                      <TaskCard
                        key={task.id}
                        task={task}
                        isAdmin={isAdmin}
                        pendingActionId={
                          actionMutation.isPending
                            ? actionMutation.variables
                            : undefined
                        }
                        lockedActionIds={lockedActionIds}
                        interactive={interactive}
                        onExecute={(actionId) => actionMutation.mutate(actionId)}
                        onOpenMetadataReview={() => setMetadataReviewTask(task)}
                      />
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>

            <Card className='border-border/60 bg-card/90 shadow-sm'>
              <CardHeader className='gap-3'>
                <div className='flex items-center justify-between gap-4'>
                  <div>
                    <CardTitle className='text-xl tracking-tight'>
                      流水线观察
                    </CardTitle>
                    <CardDescription className='mt-1'>
                      右侧保留阶段压力视图，方便边看任务边定位链路卡点。
                    </CardDescription>
                  </div>
                  <Badge variant='outline' className='rounded-full px-3 py-1'>
                    {pipelineStages.length} 段
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className='grid gap-4'>
                {pipelineStages.map((stage) => (
                  <PipelineCard key={stage.key} stage={stage} />
                ))}
              </CardContent>
            </Card>
          </section>

          <section className='space-y-3'>
            <div className='flex items-center justify-between gap-4'>
              <div>
                <h2 className='text-xl font-semibold tracking-tight'>
                  已处理记录
                </h2>
                <p className='mt-1 text-sm text-muted-foreground'>
                  保留已经治理完成的问题记录，便于回溯处理结果与时间线。
                </p>
              </div>
              <Badge variant='outline' className='rounded-full px-3 py-1'>
                {resolvedTasks.length} 项
              </Badge>
            </div>

            {resolvedTasks.length === 0 ? (
              <Card className='border-border/60 bg-card/75 shadow-sm'>
                <CardContent className='p-6 text-sm text-muted-foreground'>
                  当前还没有已处理的治理记录。
                </CardContent>
              </Card>
            ) : (
              <div className='grid gap-4 lg:grid-cols-2'>
                {resolvedTasks.map((task) => (
                  <TaskCard
                    key={task.id}
                    task={task}
                    isAdmin={isAdmin}
                    pendingActionId={
                      actionMutation.isPending
                        ? actionMutation.variables
                        : undefined
                    }
                    lockedActionIds={lockedActionIds}
                    interactive={interactive}
                    onExecute={(actionId) => actionMutation.mutate(actionId)}
                    onOpenMetadataReview={() => setMetadataReviewTask(task)}
                  />
                ))}
              </div>
            )}
          </section>
        </>
      ) : null}

      {metadataReviewTask ? (
        <MetadataReviewDialog
          key={metadataReviewTask.id}
          token={token}
          open
          task={metadataReviewTask}
          onOpenChange={(open) => {
            if (!open) setMetadataReviewTask(null)
          }}
          onResolved={async () => {
            setMetadataReviewTask(null)
            await queryClient.invalidateQueries({
              queryKey: miboQueryKeys.operationsOverview(queryToken),
            })
            await queryClient.invalidateQueries({
              queryKey: miboQueryKeys.operationsTasks(queryToken),
            })
            await queryClient.invalidateQueries({
              queryKey: miboQueryKeys.operationsPipeline(queryToken),
            })
            await queryClient.invalidateQueries({ queryKey: ['home'] })
            await queryClient.invalidateQueries({
              queryKey: miboQueryKeys.consoleSummary(queryToken),
            })
          }}
        />
      ) : null}
    </div>
  )
}

function TaskCard({
  task,
  isAdmin,
  pendingActionId,
  lockedActionIds,
  interactive,
  onExecute,
  onOpenMetadataReview,
}: {
  task: OperationsTask
  isAdmin: boolean
  pendingActionId?: string
  lockedActionIds: string[]
  interactive: boolean
  onExecute: (actionId: string) => void
  onOpenMetadataReview: () => void
}) {
  const hasMetadataReviewDialog =
    task.kind === 'metadata_review_required' &&
    typeof task.affected.items?.[0]?.id === 'number'
  const lifecycleStatus = task.lifecycle_status ?? 'active'
  const isResolved = lifecycleStatus === 'resolved'

  return (
    <Card
      className={cn(
        'min-w-0 border-border/60 shadow-sm',
        isResolved ? 'bg-muted/30' : 'bg-card/85'
      )}
    >
      <CardHeader className='gap-3'>
        <div className='flex items-start justify-between gap-4'>
          <div className='min-w-0'>
            <div className='flex flex-wrap items-center gap-2'>
              <SeverityBadge severity={task.severity} />
              <Badge
                variant='outline'
                className={cn(
                  'rounded-full px-2.5 py-0.5',
                  isResolved && 'border-emerald-500/40 text-emerald-700'
                )}
              >
                {isResolved ? '已处理' : '待处理'}
              </Badge>
              <Badge variant='outline' className='rounded-full px-2.5 py-0.5'>
                {taskKindLabel(task.kind)}
              </Badge>
            </div>
            <CardTitle className='mt-3 text-xl'>{task.title}</CardTitle>
            <CardDescription className='mt-2 max-w-3xl leading-6'>
              {task.summary}
            </CardDescription>
          </div>
          <div className='shrink-0 text-sm text-muted-foreground'>
            {formatDate(task.last_seen_at ?? task.first_seen_at)}
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='grid gap-3 sm:grid-cols-3'>
          <ImpactCell
            label='媒体库'
            value={String(task.impact.affected_libraries)}
          />
          <ImpactCell label='文件' value={String(task.impact.affected_files)} />
          <ImpactCell label='条目' value={String(task.impact.affected_items)} />
        </div>

        {task.evidence.length > 0 ? (
          <div className='rounded-lg border border-border/60 bg-background/70 p-4'>
            <div className='text-xs font-medium tracking-wide text-muted-foreground uppercase'>
              证据摘要
            </div>
            <div className='mt-3 grid gap-2'>
              {task.evidence.map((item, index) => (
                <div
                  key={`${item.kind}-${index}`}
                  className='grid min-w-0 gap-1 text-sm sm:grid-cols-[auto_minmax(0,1fr)] sm:gap-2'
                >
                  <span className='font-medium text-foreground'>
                    {item.label}
                  </span>
                  <span className='min-w-0 break-words text-muted-foreground'>
                    {' '}
                    {item.value || item.description || '无'}
                  </span>
                </div>
              ))}
            </div>
          </div>
        ) : null}

        {interactive ? (
          <div className='flex flex-wrap gap-2'>
            {hasMetadataReviewDialog ? (
              <Button onClick={onOpenMetadataReview}>处理元数据</Button>
            ) : null}
            {task.recommended_actions.map((action) => (
              <TaskActionButton
                key={action.id ?? `${action.type}-${action.href ?? action.label}`}
                action={action}
                isAdmin={isAdmin}
                isPending={pendingActionId === action.id}
                isBusy={
                  typeof pendingActionId === 'string' ||
                  (!!action.id && lockedActionIds.includes(action.id))
                }
                onExecute={onExecute}
              />
            ))}
          </div>
        ) : (
          <div className='text-sm text-muted-foreground'>
            前往治理工作台执行单条处理、批量处理或人工确认。
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function TaskActionButton({
  action,
  isAdmin,
  isPending,
  isBusy,
  onExecute,
}: {
  action: OperationsRecommendedAction
  isAdmin: boolean
  isPending: boolean
  isBusy: boolean
  onExecute: (actionId: string) => void
}) {
  if (action.href?.startsWith('/')) {
    return (
      <Button asChild>
        <a href={action.href}>
          <ArrowUpRightIcon className='size-4' />
          {action.label}
        </a>
      </Button>
    )
  }

  if (action.type === 'open_url' && action.href) {
    return (
      <Button asChild>
        <a href={action.href} target='_blank' rel='noreferrer'>
          <ArrowUpRightIcon className='size-4' />
          {action.label}
        </a>
      </Button>
    )
  }

  if (!action.id) return null

  return (
    <Button
      variant='outline'
      disabled={!isAdmin || isBusy}
      onClick={() => onExecute(action.id!)}
      title={isAdmin ? action.description : '需要管理员权限'}
    >
      {isPending ? (
        <LoaderCircleIcon className='size-4 animate-spin' />
      ) : (
        actionLabelIcon(action.type)
      )}
      {isPending ? `${action.label}中` : action.label}
    </Button>
  )
}

function PipelineCard({ stage }: { stage: OperationsPipelineStage }) {
  const samples = stage.samples ?? []

  return (
    <Card className='border-border/60 bg-card/85 shadow-sm'>
      <CardHeader className='gap-3'>
        <div className='flex items-start justify-between gap-3'>
          <div>
            <CardTitle className='text-lg'>{stage.label}</CardTitle>
            <CardDescription className='mt-1'>
              失败 {stage.failed} · 待确认 {stage.review_required} · 过期{' '}
              {stage.stale}
            </CardDescription>
          </div>
          <StatusBadge status={stage.status} compact />
        </div>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='grid grid-cols-3 gap-3 text-sm'>
          <ImpactCell
            label='待处理'
            value={String(stage.pending + stage.running)}
          />
          <ImpactCell label='失败' value={String(stage.failed)} />
          <ImpactCell label='可重试' value={String(stage.retry_eligible)} />
        </div>
        <div className='space-y-2'>
          {samples.length === 0 ? (
            <div className='text-sm text-muted-foreground'>暂无样本</div>
          ) : (
            samples.map((sample) => (
              <div
                key={sample.id}
                className='rounded-lg border border-border/60 bg-background/70 px-3 py-3 text-sm'
              >
                <div className='flex items-center justify-between gap-3'>
                  <span className='font-medium'>
                    {sample.library_name || `媒体库 #${sample.library_id}`}
                  </span>
                  <Badge variant='outline'>{sample.status}</Badge>
                </div>
                <div className='mt-2 text-muted-foreground'>
                  {sample.message || sample.reason || sample.condition_type}
                </div>
              </div>
            ))
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function MetricCard({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: 'warning' | 'danger' | 'neutral'
}) {
  return (
    <div className='rounded-xl border border-border/60 bg-background/80 px-4 py-4 shadow-sm'>
      <div className='text-sm text-muted-foreground'>{label}</div>
      <div
        className={cn(
          'mt-2 text-3xl font-semibold',
          tone === 'danger' && 'text-destructive',
          tone === 'warning' && 'text-amber-600'
        )}
      >
        {value}
      </div>
    </div>
  )
}

function InsightCard({
  icon: Icon,
  label,
  value,
  description,
}: {
  icon: typeof ListTodoIcon
  label: string
  value: string
  description: string
}) {
  return (
    <div className='rounded-xl border border-border/60 bg-background/80 px-4 py-4 shadow-sm'>
      <div className='flex items-center gap-2 text-sm text-muted-foreground'>
        <Icon className='size-4' />
        {label}
      </div>
      <div className='mt-3 line-clamp-2 text-base font-semibold text-foreground'>
        {value}
      </div>
      <div className='mt-2 text-sm leading-5 text-muted-foreground'>
        {description}
      </div>
    </div>
  )
}

function ImpactCell({ label, value }: { label: string; value: string }) {
  return (
    <div className='rounded-lg border border-border/60 bg-background/70 px-3 py-3'>
      <div className='text-xs text-muted-foreground'>{label}</div>
      <div className='mt-1 font-medium'>{value}</div>
    </div>
  )
}

function StatusBadge({
  status,
  compact = false,
}: {
  status: OperationsStatus
  compact?: boolean
}) {
  return (
    <Badge
      variant='outline'
      className={cn(
        'rounded-full',
        status === 'healthy' &&
          'border-emerald-500/30 bg-emerald-500/10 text-emerald-700',
        status === 'attention' &&
          'border-amber-500/30 bg-amber-500/10 text-amber-700',
        status === 'degraded' &&
          'border-orange-500/30 bg-orange-500/10 text-orange-700',
        status === 'blocked' &&
          'border-destructive/30 bg-destructive/10 text-destructive',
        compact ? 'px-2.5 py-0.5 text-xs' : 'px-3 py-1 text-xs'
      )}
    >
      {statusLabel(status)}
    </Badge>
  )
}

function SeverityBadge({ severity }: { severity: OperationsTask['severity'] }) {
  return (
    <Badge
      variant='outline'
      className={cn(
        'rounded-full',
        severity === 'blocking' &&
          'border-destructive/30 bg-destructive/10 text-destructive',
        severity === 'error' &&
          'border-orange-500/30 bg-orange-500/10 text-orange-700',
        severity === 'warning' &&
          'border-amber-500/30 bg-amber-500/10 text-amber-700',
        severity === 'info' && 'border-sky-500/30 bg-sky-500/10 text-sky-700'
      )}
    >
      {severityLabel(severity)}
    </Badge>
  )
}

function actionLabelIcon(type: OperationsRecommendedAction['type']) {
  switch (type) {
    case 'validate_media_source':
      return <CheckCircle2Icon className='size-4' />
    case 'scan_library':
      return <RefreshCwIcon className='size-4' />
    case 'retry_ingest_stage':
      return <RefreshCwIcon className='size-4' />
    case 'retry_probe_file':
      return <RefreshCwIcon className='size-4' />
    case 'resolve_review_stage':
      return <WrenchIcon className='size-4' />
    default:
      return <AlertTriangleIcon className='size-4' />
  }
}

function statusLabel(status: OperationsStatus) {
  switch (status) {
    case 'healthy':
      return '稳定'
    case 'attention':
      return '关注'
    case 'degraded':
      return '受影响'
    case 'blocked':
      return '阻断'
  }
}

function severityLabel(severity: OperationsTask['severity']) {
  switch (severity) {
    case 'blocking':
      return '阻断'
    case 'error':
      return '错误'
    case 'warning':
      return '警告'
    case 'info':
      return '提示'
  }
}

function taskKindLabel(kind: OperationsTask['kind']) {
  switch (kind) {
    case 'storage_access_required':
      return '存储连接'
    case 'scan_blocked':
      return '扫描受阻'
    case 'classification_review_required':
      return '分类确认'
    case 'metadata_review_required':
      return '元数据确认'
    case 'projection_stale':
      return '目录投影'
    case 'maintenance_backlog':
      return '流水线维护'
  }
}

function formatDate(value?: string) {
  if (!value) return '最近更新未知'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}
