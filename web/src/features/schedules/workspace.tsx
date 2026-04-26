import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarClockIcon } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Badge } from '#/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import { Separator } from '#/components/ui/separator'
import { type Schedule, type ScheduleMutationInput } from '#/lib/mibo-api'
import {
  createAuthedMiboApi,
  librariesQueryOptions,
  miboQueryKeys,
  scheduleHistoryQueryOptions,
  schedulesQueryOptions,
} from '#/lib/mibo-query'

import { ScheduleFormDialog } from './components/schedule-form-dialog'
import {
  ScheduleList,
  formatDateTime,
  formatFrequency,
  formatKind,
  formatLatestResult,
  formatScope,
} from './components/schedule-list'
import { ScheduleRunHistoryDrawer } from './components/schedule-run-history-drawer'

export function SchedulesWorkspace({ token }: { token: string }) {
  const queryClient = useQueryClient()
  const api = createAuthedMiboApi(token)
  const schedulesQuery = useQuery(schedulesQueryOptions(token))
  const librariesQuery = useQuery(librariesQueryOptions(token))
  const [selectedScheduleId, setSelectedScheduleId] = useState<number | null>(
    null,
  )
  const [formOpen, setFormOpen] = useState(false)
  const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null)
  const [historySchedule, setHistorySchedule] = useState<Schedule | null>(null)

  const schedules = schedulesQuery.data ?? []
  const selectedSchedule = useMemo(() => {
    if (selectedScheduleId == null) {
      return schedules[0] ?? null
    }
    return (
      schedules.find((schedule) => schedule.id === selectedScheduleId) ??
      schedules[0] ??
      null
    )
  }, [schedules, selectedScheduleId])

  const historyQuery = useQuery({
    ...scheduleHistoryQueryOptions(token, historySchedule?.id ?? 0),
    enabled: Boolean(historySchedule?.id),
  })

  const refreshSchedules = async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: miboQueryKeys.schedules(token),
      }),
      historySchedule?.id
        ? queryClient.invalidateQueries({
            queryKey: miboQueryKeys.scheduleHistory(token, historySchedule.id),
          })
        : Promise.resolve(),
    ])
  }

  const saveMutation = useMutation({
    mutationFn: async (input: ScheduleMutationInput) => {
      if (editingSchedule) {
        return api.updateSchedule(editingSchedule.id, input)
      }
      return api.createSchedule(input)
    },
    onSuccess: async (schedule) => {
      setSelectedScheduleId(schedule.id)
      setEditingSchedule(null)
      await refreshSchedules()
      toast.success(editingSchedule ? '计划任务已更新' : '计划任务已创建')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const toggleMutation = useMutation({
    mutationFn: (schedule: Schedule) =>
      api.toggleSchedule(schedule.id, !schedule.enabled),
    onSuccess: async () => {
      await refreshSchedules()
      toast.success('计划任务状态已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const runNowMutation = useMutation({
    mutationFn: (schedule: Schedule) => api.runScheduleNow(schedule.id),
    onSuccess: async () => {
      await refreshSchedules()
      toast.success('计划任务已加入后台队列')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <div className="space-y-4 text-foreground">
      <div className="flex flex-col gap-4 rounded-[1.75rem] border border-border/60 bg-card/80 p-5 shadow-sm backdrop-blur-sm sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-2">
          <Badge
            variant="outline"
            className="border-border/60 bg-background/70"
          >
            Phase 10
          </Badge>
          <h1 className="text-3xl font-semibold tracking-tight">
            计划任务工作台
          </h1>
          <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
            独立管理 recurring maintenance schedules，直接查看 enabled
            状态、next run、latest result，并在详情层展开最近运行历史。
          </p>
        </div>
      </div>

      <Alert>
        <CalendarClockIcon className="size-4" />
        <AlertTitle>计划任务成为独立主入口</AlertTitle>
        <AlertDescription>
          当前工作台已经并入设置中心，仍保持 schedule-first 视角，完整 CRUD /
          run-now / history 都在当前页面完成。
        </AlertDescription>
      </Alert>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
          <CardHeader className="px-5 py-5">
            <CardTitle>计划任务列表</CardTitle>
            <CardDescription>
              主列表直接呈现任务类型、目标范围、频率模板、启停状态和最新结果。
            </CardDescription>
          </CardHeader>
          <Separator className="bg-border" />
          <CardContent className="space-y-4 px-5 py-5">
            {schedulesQuery.isLoading ? (
              <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-6 text-sm text-muted-foreground">
                正在加载计划任务…
              </div>
            ) : schedulesQuery.error ? (
              <Alert>
                <AlertTitle>加载失败</AlertTitle>
                <AlertDescription>
                  {schedulesQuery.error.message}
                </AlertDescription>
              </Alert>
            ) : (
              <ScheduleList
                schedules={schedules}
                selectedScheduleId={selectedSchedule?.id}
                onCreate={() => {
                  setEditingSchedule(null)
                  setFormOpen(true)
                }}
                onEdit={(schedule) => {
                  setEditingSchedule(schedule)
                  setFormOpen(true)
                }}
                onRunNow={(schedule) => runNowMutation.mutate(schedule)}
                onSelect={(schedule) => setSelectedScheduleId(schedule.id)}
                onShowHistory={(schedule) => setHistorySchedule(schedule)}
                onToggle={(schedule) => toggleMutation.mutate(schedule)}
              />
            )}
          </CardContent>
        </Card>

        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
          <CardHeader className="px-5 py-5">
            <CardTitle>任务详情摘要</CardTitle>
            <CardDescription>
              详情层继续承接运行历史与后续执行反馈。
            </CardDescription>
          </CardHeader>
          <Separator className="bg-border" />
          <CardContent className="space-y-4 px-5 py-5 text-sm text-muted-foreground">
            {selectedSchedule ? (
              <>
                <DetailRow
                  label="任务类型"
                  value={formatKind(selectedSchedule.kind)}
                />
                <DetailRow
                  label="目标范围"
                  value={formatScope(
                    selectedSchedule.scope_kind,
                    selectedSchedule.library_id,
                  )}
                />
                <DetailRow
                  label="频率模板"
                  value={formatFrequency(selectedSchedule.frequency)}
                />
                <DetailRow
                  label="下次运行"
                  value={formatDateTime(selectedSchedule.next_run_at)}
                />
                <DetailRow
                  label="最近结果"
                  value={formatLatestResult(selectedSchedule)}
                />
                <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3">
                  <div className="text-xs uppercase tracking-[0.16em] text-muted-foreground">
                    详情层入口
                  </div>
                  <div className="mt-1 text-sm text-foreground">
                    查看最近运行历史并跟踪 queued / running / completed / failed
                    状态。
                  </div>
                </div>
              </>
            ) : (
              <div className="rounded-[1.1rem] border border-dashed border-border/60 px-4 py-6 text-center">
                选择一条计划任务后查看更多信息。
              </div>
            )}

            <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3">
              <div className="text-xs uppercase tracking-[0.16em] text-muted-foreground">
                数据来源
              </div>
              <div className="mt-1 text-sm text-foreground">
                列表、详情和历史都通过 typed API + TanStack Query 管理，不解析
                payload_json，也不回退到 /api/v1/jobs 重建语义。
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <ScheduleFormDialog
        open={formOpen}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) {
            setEditingSchedule(null)
          }
        }}
        libraries={librariesQuery.data ?? []}
        schedule={editingSchedule}
        onSubmit={async (input) => {
          await saveMutation.mutateAsync(input)
        }}
      />

      <ScheduleRunHistoryDrawer
        open={Boolean(historySchedule)}
        onOpenChange={(open) => {
          if (!open) {
            setHistorySchedule(null)
          }
        }}
        schedule={historySchedule}
        runs={historyQuery.data ?? []}
        isLoading={historyQuery.isLoading}
      />
    </div>
  )
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3">
      <div className="text-xs uppercase tracking-[0.16em] text-muted-foreground">
        {label}
      </div>
      <div className="mt-1 text-sm text-foreground">{value}</div>
    </div>
  )
}
