import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import {
  AlertTriangleIcon,
  LaptopIcon,
  Loader2Icon,
  LogOutIcon,
  MonitorSmartphoneIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
  Trash2Icon,
} from 'lucide-react'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '#/components/ui/alert-dialog'
import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import { Skeleton } from '#/components/ui/skeleton'
import type { LoginSession } from '#/lib/mibo-api'
import {
  createAuthedMiboApi,
  loginSessionsQueryOptions,
  miboQueryKeys,
} from '#/lib/mibo-query'
import { cn } from '#/lib/utils'
import { useAuthStore } from '#/stores/auth-store'

export function DeviceManagementPanel() {
  const token = useAuthStore((state) => state.token)
  const queryClient = useQueryClient()
  const [sessionToRevoke, setSessionToRevoke] = useState<LoginSession | null>(
    null,
  )
  const [confirmRevokeOthers, setConfirmRevokeOthers] = useState(false)
  const sessionsQuery = useQuery({
    ...loginSessionsQueryOptions(token ?? ''),
    enabled: Boolean(token),
  })
  const sessions = sessionsQuery.data ?? []
  const otherSessions = sessions.filter((session) => !session.is_current)

  const invalidateSessions = async () => {
    if (!token) return
    await queryClient.invalidateQueries({
      queryKey: miboQueryKeys.loginSessions(token),
    })
  }

  const revokeSession = useMutation({
    mutationFn: async (sessionId: number) => {
      if (!token) throw new Error('缺少登录状态')
      return createAuthedMiboApi(token).revokeLoginSession(sessionId)
    },
    onSuccess: async () => {
      setSessionToRevoke(null)
      await invalidateSessions()
    },
  })
  const revokeOthers = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('缺少登录状态')
      return createAuthedMiboApi(token).revokeOtherLoginSessions()
    },
    onSuccess: async () => {
      setConfirmRevokeOthers(false)
      await invalidateSessions()
    },
  })
  const mutationError = revokeSession.error ?? revokeOthers.error
  const isMutating = revokeSession.isPending || revokeOthers.isPending

  return (
    <div className="space-y-4">
      <section className="rounded-[1.5rem] border border-border/60 bg-card/70 p-4 shadow-sm backdrop-blur-sm">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
          <div className="space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="outline" className="gap-1.5 bg-background/70">
                <MonitorSmartphoneIcon className="size-3.5 text-emerald-500" />
                {sessions.length} 个登录会话
              </Badge>
              <Badge variant="outline" className="gap-1.5 bg-background/70">
                <ShieldCheckIcon className="size-3.5 text-emerald-500" />
                当前设备受保护
              </Badge>
            </div>
            <p className="text-sm text-muted-foreground">
              查看当前账号已登录的浏览器和客户端，撤销不再使用的会话。
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              onClick={() => void invalidateSessions()}
              disabled={!token || sessionsQuery.isFetching}
            >
              <RefreshCwIcon
                className={cn(
                  'size-4',
                  sessionsQuery.isFetching && 'animate-spin',
                )}
              />
              刷新
            </Button>
            <Button
              variant="destructive"
              onClick={() => setConfirmRevokeOthers(true)}
              disabled={otherSessions.length === 0 || isMutating}
            >
              <LogOutIcon className="size-4" />
              撤销其他会话
            </Button>
          </div>
        </div>
      </section>

      {mutationError ? (
        <div className="flex items-start gap-3 rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
          <AlertTriangleIcon className="mt-0.5 size-4 shrink-0" />
          <span>{errorMessage(mutationError)}</span>
        </div>
      ) : null}

      <section className="min-h-[420px] rounded-[1.5rem] border border-border/60 bg-gradient-to-br from-card/90 via-card/70 to-emerald-500/5 p-5 shadow-sm backdrop-blur-sm">
        <div className="mb-5">
          <h3 className="text-base font-medium">登录设备</h3>
          <p className="text-sm text-muted-foreground">
            当前会话只能通过退出登录结束，不能在这里撤销。
          </p>
        </div>

        {sessionsQuery.isLoading ? (
          <DeviceSkeleton />
        ) : sessionsQuery.isError ? (
          <ErrorState onRetry={() => void invalidateSessions()} />
        ) : sessions.length === 0 ? (
          <EmptyDeviceState />
        ) : (
          <div className="grid gap-4 lg:grid-cols-2 2xl:grid-cols-3">
            {sessions.map((session) => (
              <SessionCard
                key={session.id}
                session={session}
                pending={
                  revokeSession.isPending && sessionToRevoke?.id === session.id
                }
                onRevoke={() => setSessionToRevoke(session)}
              />
            ))}
          </div>
        )}
      </section>

      <AlertDialog
        open={Boolean(sessionToRevoke)}
        onOpenChange={(open) => !open && setSessionToRevoke(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>撤销此登录会话？</AlertDialogTitle>
            <AlertDialogDescription>
              {sessionToRevoke
                ? `${sessionDisplayName(sessionToRevoke)} 将需要重新登录。当前会话不会受影响。`
                : '此会话将需要重新登录。'}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={revokeSession.isPending}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={!sessionToRevoke || revokeSession.isPending}
              onClick={() => {
                if (!sessionToRevoke) return
                revokeSession.mutate(sessionToRevoke.id)
              }}
            >
              {revokeSession.isPending ? '撤销中...' : '确认撤销'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={confirmRevokeOthers}
        onOpenChange={(open) => setConfirmRevokeOthers(open)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>撤销所有其他会话？</AlertDialogTitle>
            <AlertDialogDescription>
              将撤销 {otherSessions.length} 个其他登录会话。当前设备会保持登录。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={revokeOthers.isPending}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={otherSessions.length === 0 || revokeOthers.isPending}
              onClick={() => revokeOthers.mutate()}
            >
              {revokeOthers.isPending ? '撤销中...' : '确认撤销'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function SessionCard({
  session,
  pending,
  onRevoke,
}: {
  session: LoginSession
  pending: boolean
  onRevoke: () => void
}) {
  return (
    <Card
      className={cn(
        'rounded-[1.35rem] border-border/60 bg-background/80 shadow-sm',
        session.is_current &&
          'border-emerald-500/50 ring-3 ring-emerald-500/10',
      )}
    >
      <CardHeader className="space-y-3">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex size-12 shrink-0 items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
              <LaptopIcon className="size-5" />
            </div>
            <div className="min-w-0">
              <CardTitle className="truncate text-base">
                {sessionDisplayName(session)}
              </CardTitle>
              <CardDescription className="truncate">
                {session.client_type || session.user_agent || 'Mibo Web'}
              </CardDescription>
            </div>
          </div>
          {session.is_current ? (
            <Badge className="shrink-0 bg-emerald-500/15 text-emerald-700 hover:bg-emerald-500/15 dark:text-emerald-300">
              当前
            </Badge>
          ) : null}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <DetailRow
          label="最近活动"
          value={formatDateTime(session.last_used_at)}
        />
        <DetailRow
          label="创建时间"
          value={formatDateTime(session.created_at)}
        />
        <DetailRow
          label="过期时间"
          value={formatDateTime(session.expires_at)}
        />
        <DetailRow label="远程地址" value={session.remote_addr || '未知地址'} />
        <DetailRow label="User-Agent" value={session.user_agent || '未记录'} />

        <Button
          variant={session.is_current ? 'outline' : 'destructive'}
          className="w-full"
          disabled={session.is_current || pending}
          title={session.is_current ? '请使用退出登录结束当前会话' : undefined}
          onClick={onRevoke}
        >
          {pending ? (
            <Loader2Icon className="size-4 animate-spin" />
          ) : (
            <Trash2Icon className="size-4" />
          )}
          {session.is_current ? '当前会话不可撤销' : '撤销此会话'}
        </Button>
      </CardContent>
    </Card>
  )
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-3 rounded-2xl border border-border/50 bg-card/50 px-3 py-2.5 text-sm">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right font-medium">{value}</span>
    </div>
  )
}

function DeviceSkeleton() {
  return (
    <div className="grid gap-4 lg:grid-cols-2 2xl:grid-cols-3">
      {Array.from({ length: 3 }).map((_, index) => (
        <Skeleton key={index} className="h-80 rounded-[1.35rem]" />
      ))}
    </div>
  )
}

function EmptyDeviceState() {
  return (
    <div className="flex min-h-[260px] flex-col items-center justify-center rounded-[1.35rem] border border-dashed border-border/70 bg-background/60 p-8 text-center">
      <MonitorSmartphoneIcon className="size-10 text-muted-foreground" />
      <h4 className="mt-4 text-base font-medium">暂无登录会话</h4>
      <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">
        登录后创建的浏览器和客户端会话会显示在这里。缺失设备信息时会使用安全的备用标签。
      </p>
    </div>
  )
}

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex min-h-[260px] flex-col items-center justify-center rounded-[1.35rem] border border-dashed border-destructive/30 bg-destructive/5 p-8 text-center">
      <AlertTriangleIcon className="size-10 text-destructive" />
      <h4 className="mt-4 text-base font-medium">无法加载登录会话</h4>
      <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">
        请检查当前登录状态或稍后重试。
      </p>
      <Button className="mt-4" variant="outline" onClick={onRetry}>
        重新加载
      </Button>
    </div>
  )
}

function sessionDisplayName(session: LoginSession) {
  return session.device_name || session.client_type || '未知设备'
}

function formatDateTime(value?: string) {
  if (!value) return '未知'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未知'
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function errorMessage(error: unknown) {
  if (error instanceof Error) return error.message
  return '操作失败，请稍后重试。'
}
