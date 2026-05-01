import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  AlertTriangleIcon,
  LoaderCircleIcon,
  PlayIcon,
  SaveIcon,
} from "lucide-react"
import { useEffect, useState } from "react"

import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert"
import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import { Input } from "#/components/ui/input"
import { Label } from "#/components/ui/label"
import { Switch } from "#/components/ui/switch"
import type { CleanupSettingsInput, Job } from "#/lib/mibo-api"
import {
  cleanupSettingsQueryOptions,
  createAuthedMiboApi,
  miboQueryKeys,
} from "#/lib/mibo-query"

export function CleanupSettingsPanel({ token }: { token: string | null }) {
  if (!token) {
    return (
      <PanelCard>
        <p className="text-sm text-muted-foreground">
          登录后可查看清理策略并触发缺失媒体清理。
        </p>
      </PanelCard>
    )
  }

  return <CleanupSettingsWorkspace token={token} />
}

function CleanupSettingsWorkspace({ token }: { token: string }) {
  const queryClient = useQueryClient()
  const [confirmText, setConfirmText] = useState("")
  const [queuedJob, setQueuedJob] = useState<Job | null>(null)
  const [enabled, setEnabled] = useState(false)
  const [retentionDays, setRetentionDays] = useState("30")
  const [batchSize, setBatchSize] = useState("100")
  const settingsQuery = useQuery(cleanupSettingsQueryOptions(token))
  const saveSettingsMutation = useMutation({
    mutationFn: (input: CleanupSettingsInput) =>
      createAuthedMiboApi(token).updateCleanupSettings(input),
    onSuccess: (settings) => {
      queryClient.setQueryData(miboQueryKeys.cleanupSettings(token), settings)
    },
  })
  const runCleanupMutation = useMutation({
    mutationFn: () => createAuthedMiboApi(token).runMissingMediaCleanup(),
    onSuccess: (job) => {
      setQueuedJob(job)
      setConfirmText("")
      void queryClient.invalidateQueries({
        queryKey: miboQueryKeys.jobs(token, { kind: "missing_media_cleanup" }),
      })
    },
  })

  const settings = settingsQuery.data

  useEffect(() => {
    if (!settings) return
    setEnabled(settings.missing_cleanup_enabled)
    setRetentionDays(
      String(Math.floor(settings.missing_retention_seconds / 86400))
    )
    setBatchSize(String(settings.missing_cleanup_batch_size))
  }, [settings])

  const canConfirm = confirmText.trim() === "确认清理"
  const canRun = Boolean(settings?.can_run && canConfirm)
  const retentionSeconds =
    Math.max(0, Number.parseInt(retentionDays, 10) || 0) * 86400
  const parsedBatchSize = Number.parseInt(batchSize, 10) || 0
  const canSave = parsedBatchSize >= 1 && parsedBatchSize <= 1000

  function handleSaveSettings() {
    if (!canSave) return
    saveSettingsMutation.mutate({
      missing_cleanup_enabled: enabled,
      missing_retention_seconds: retentionSeconds,
      missing_cleanup_batch_size: parsedBatchSize,
    })
  }

  return (
    <div className="space-y-4">
      <Alert className="border-destructive/40 bg-destructive/5">
        <AlertTriangleIcon className="size-4" />
        <AlertTitle>高危操作</AlertTitle>
        <AlertDescription>
          {settings?.warning ??
            "缺失媒体清理会永久删除目录、资产、库存、播放进度、收藏和人工治理数据。"}
        </AlertDescription>
      </Alert>

      <PanelCard>
        <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div className="space-y-2">
            <Badge
              variant={
                settings?.missing_cleanup_enabled ? "default" : "outline"
              }
            >
              {settings?.missing_cleanup_enabled ? "已启用" : "未启用"}
            </Badge>
            <h3 className="text-lg font-semibold tracking-tight">
              缺失媒体硬删除清理
            </h3>
            <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
              点击运行后会创建后台任务，worker 会按 retention 策略删除已经
              missing 超过保留期的媒体图谱。扫描仍只负责标记
              missing，不会在扫描请求中直接删除。
            </p>
          </div>

          {settingsQuery.isLoading ? (
            <div className="inline-flex items-center gap-2 text-sm text-muted-foreground">
              <LoaderCircleIcon className="size-4 animate-spin" />
              正在读取策略
            </div>
          ) : null}
        </div>

        <div className="mt-5 grid gap-3 sm:grid-cols-3">
          <PolicyMetric
            label="保留期"
            value={formatRetention(settings?.missing_retention_seconds)}
          />
          <PolicyMetric
            label="批处理大小"
            value={String(settings?.missing_cleanup_batch_size ?? "-")}
          />
          <PolicyMetric label="执行方式" value="后台任务" />
        </div>
      </PanelCard>

      <PanelCard>
        <div className="space-y-5">
          <div className="space-y-1">
            <h3 className="text-base font-semibold tracking-tight">清理策略</h3>
            <p className="text-sm leading-6 text-muted-foreground">
              保存后会写入数据库，下一次计划任务和主动触发都会立即使用这些配置。
            </p>
          </div>

          <div className="flex items-center justify-between gap-4 rounded-xl border border-border/60 bg-background/50 px-4 py-3">
            <div className="space-y-1">
              <Label htmlFor="cleanup-enabled">启用缺失媒体硬删除</Label>
              <p className="text-sm text-muted-foreground">
                关闭时计划任务和主动触发都不会删除 missing 媒体。
              </p>
            </div>
            <Switch
              id="cleanup-enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
              disabled={saveSettingsMutation.isPending}
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="cleanup-retention-days">保留天数</Label>
              <Input
                id="cleanup-retention-days"
                type="number"
                min={0}
                value={retentionDays}
                onChange={(event) => setRetentionDays(event.target.value)}
                disabled={saveSettingsMutation.isPending}
              />
              <p className="text-xs text-muted-foreground">
                0 表示清理任务运行时立即删除已标记 missing 的媒体。
              </p>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="cleanup-batch-size">批处理大小</Label>
              <Input
                id="cleanup-batch-size"
                type="number"
                min={1}
                max={1000}
                value={batchSize}
                onChange={(event) => setBatchSize(event.target.value)}
                disabled={saveSettingsMutation.isPending}
              />
              <p className="text-xs text-muted-foreground">
                每批删除候选文件数，范围 1-1000。
              </p>
            </div>
          </div>

          {saveSettingsMutation.error ? (
            <p className="text-sm text-destructive">
              {saveSettingsMutation.error instanceof Error
                ? saveSettingsMutation.error.message
                : "保存清理策略失败"}
            </p>
          ) : null}

          {saveSettingsMutation.isSuccess ? (
            <p className="text-sm text-muted-foreground">清理策略已保存。</p>
          ) : null}

          <Button
            disabled={!canSave || saveSettingsMutation.isPending}
            onClick={handleSaveSettings}
          >
            {saveSettingsMutation.isPending ? (
              <LoaderCircleIcon className="size-4 animate-spin" />
            ) : (
              <SaveIcon className="size-4" />
            )}
            保存清理策略
          </Button>
        </div>
      </PanelCard>

      <PanelCard>
        <div className="space-y-4">
          <div className="space-y-1">
            <h3 className="text-base font-semibold tracking-tight">
              主动触发清理
            </h3>
            <p className="text-sm leading-6 text-muted-foreground">
              输入 <span className="font-medium text-foreground">确认清理</span>
              后可入队一次全局缺失媒体清理任务。
            </p>
          </div>

          <div className="grid max-w-xl gap-2">
            <Label htmlFor="cleanup-confirm">确认文本</Label>
            <Input
              id="cleanup-confirm"
              value={confirmText}
              onChange={(event) => setConfirmText(event.target.value)}
              placeholder="确认清理"
              disabled={runCleanupMutation.isPending}
            />
          </div>

          {!settings?.can_run ? (
            <p className="text-sm text-muted-foreground">
              当前未启用缺失媒体清理。打开上方开关并保存后即可主动触发。
            </p>
          ) : null}

          {runCleanupMutation.error ? (
            <p className="text-sm text-destructive">
              {runCleanupMutation.error instanceof Error
                ? runCleanupMutation.error.message
                : "清理任务创建失败"}
            </p>
          ) : null}

          {queuedJob ? (
            <div className="rounded-xl border border-border/60 bg-muted/30 px-4 py-3 text-sm">
              已创建后台任务 #{queuedJob.id}，状态：{queuedJob.status}。
            </div>
          ) : null}

          <Button
            variant="destructive"
            disabled={!canRun || runCleanupMutation.isPending}
            onClick={() => runCleanupMutation.mutate()}
          >
            {runCleanupMutation.isPending ? (
              <LoaderCircleIcon className="size-4 animate-spin" />
            ) : (
              <PlayIcon className="size-4" />
            )}
            运行缺失媒体清理
          </Button>
        </div>
      </PanelCard>
    </div>
  )
}

function PanelCard({ children }: { children: React.ReactNode }) {
  return (
    <section className="rounded-[1.25rem] border border-border/60 bg-card/80 p-5 shadow-sm">
      {children}
    </section>
  )
}

function PolicyMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border/60 bg-background/50 px-4 py-3">
      <p className="text-xs tracking-[0.18em] text-muted-foreground uppercase">
        {label}
      </p>
      <p className="mt-2 text-lg font-semibold">{value}</p>
    </div>
  )
}

function formatRetention(seconds?: number) {
  if (typeof seconds !== "number") return "-"
  if (seconds <= 0) return "立即"
  const days = Math.floor(seconds / 86400)
  if (days >= 1) return `${days} 天`
  const hours = Math.floor(seconds / 3600)
  if (hours >= 1) return `${hours} 小时`
  return `${seconds} 秒`
}
