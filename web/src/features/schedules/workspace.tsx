import { lazy, Suspense, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarClockIcon, CircleHelpIcon } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Button } from '#/components/ui/button'
import { type Schedule, type ScheduleMutationInput } from '#/lib/mibo-api'
import {
  createAuthedMiboApi,
  librariesQueryOptions,
  miboQueryKeys,
  scheduleHistoryQueryOptions,
  schedulesQueryOptions,
} from '#/lib/mibo-query'

import { ScheduleList } from './components/schedule-list'

const ScheduleFormDialog = lazy(() =>
  import('./components/schedule-form-dialog').then((module) => ({
    default: module.ScheduleFormDialog,
  })),
)
const ScheduleRunHistoryDrawer = lazy(() =>
  import('./components/schedule-run-history-drawer').then((module) => ({
    default: module.ScheduleRunHistoryDrawer,
  })),
)

export function SchedulesWorkspace({ token }: { token: string }) {
  const queryClient = useQueryClient()
  const api = createAuthedMiboApi(token)
  const schedulesQuery = useQuery(schedulesQueryOptions(token))
  const librariesQuery = useQuery(librariesQueryOptions(token))
  const [formOpen, setFormOpen] = useState(false)
  const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null)
  const [historySchedule, setHistorySchedule] = useState<Schedule | null>(null)

  const schedules = schedulesQuery.data ?? []

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
    onSuccess: async () => {
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
    <div className="space-y-6 text-foreground">
      <header className="flex flex-col gap-4 border-b border-border/70 pb-5 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-2">
          <div className="flex items-center gap-2">
            <h1 className="text-3xl font-semibold tracking-tight">计划任务</h1>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className="size-8 rounded-full text-muted-foreground"
              onClick={() =>
                toast.info(
                  '计划任务会在后台自动执行，也可以从列表右侧手动运行。',
                )
              }
            >
              <CircleHelpIcon className="size-4" />
              <span className="sr-only">计划任务帮助</span>
            </Button>
          </div>
          <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
            查看服务器后台维护任务的最近运行状态、耗时和用途说明，并可直接手动触发执行。
          </p>
        </div>
        <Button onClick={() => setFormOpen(true)}>新建计划任务</Button>
      </header>

      {schedulesQuery.isLoading ? (
        <div className="rounded-xl border border-border/60 bg-card/60 px-4 py-6 text-sm text-muted-foreground">
          正在加载计划任务…
        </div>
      ) : schedulesQuery.error ? (
        <Alert>
          <CalendarClockIcon className="size-4" />
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>{schedulesQuery.error.message}</AlertDescription>
        </Alert>
      ) : (
        <ScheduleList
          schedules={schedules}
          onCreate={() => {
            setEditingSchedule(null)
            setFormOpen(true)
          }}
          onEdit={(schedule) => {
            setEditingSchedule(schedule)
            setFormOpen(true)
          }}
          onRunNow={(schedule) => runNowMutation.mutate(schedule)}
          onShowHistory={(schedule) => setHistorySchedule(schedule)}
          onToggle={(schedule) => toggleMutation.mutate(schedule)}
          runningScheduleId={runNowMutation.variables?.id}
          isRunning={runNowMutation.isPending}
        />
      )}

      <Suspense fallback={null}>
        {formOpen ? (
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
        ) : null}

        {historySchedule ? (
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
        ) : null}
      </Suspense>
    </div>
  )
}
