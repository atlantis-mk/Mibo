import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  ExternalLinkIcon,
  LoaderCircleIcon,
  RotateCcwIcon,
  ShieldAlertIcon,
} from "lucide-react"

import { Button } from "#/components/ui/button"
import { Badge } from "#/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card"
import {
  createAuthedMiboApi,
  healthIssuesQueryOptions,
  ingestDiagnosticsQueryOptions,
  miboQueryKeys,
} from "#/lib/mibo-query"
import {
  affectedLibraryNames,
  healthReasonMessage,
  healthReasonTitle,
  healthSeverityClassName,
  healthSeverityLabel,
} from "#/lib/health-presentation"
import type {
  HealthAction,
  HealthIssue,
  IngestDiagnosticStage,
  MediaSourceValidationResult,
} from "#/lib/mibo-api"
import { useAuthStore } from "#/stores/auth-store"

import { getHealthCenterState } from "./health-center-state"
import { IngestDiagnosticsPanel } from "./ingest-diagnostics-panel"

export default function HealthCenter() {
  const token = useAuthStore((state) => state.token)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const queryToken = token ?? "guest"
  const queryClient = useQueryClient()
  const [validationResults, setValidationResults] = useState<
    Record<number, MediaSourceValidationResult>
  >({})
  const issuesQuery = useQuery({
    ...healthIssuesQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const ingestQuery = useQuery({
    ...ingestDiagnosticsQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const issues = issuesQuery.data ?? []

  const validateMutation = useMutation({
    mutationFn: (mediaSourceId: number) => {
      if (!token) throw new Error("当前未登录，无法验证媒体源。")
      return createAuthedMiboApi(token).validateMediaSource(mediaSourceId)
    },
    onSuccess: (result) => {
      setValidationResults((current) => ({
        ...current,
        [result.media_source_id]: result,
      }))
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.healthIssues(queryToken),
      })
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.healthSummary(queryToken),
      })
    },
  })
  const rescanMutation = useMutation({
    mutationFn: (issueId: string) => {
      if (!token) throw new Error("当前未登录，无法重新扫描。")
      return createAuthedMiboApi(token).rescanHealthIssueLibraries(issueId)
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.healthIssues(queryToken),
      })
      await queryClient.invalidateQueries({
        queryKey: ["settings", "libraries"],
      })
      await queryClient.invalidateQueries({ queryKey: ["home"] })
    },
  })
  const ignoreMutation = useMutation({
    mutationFn: (issueId: string) => {
      if (!token) throw new Error("当前未登录，无法忽略问题。")
      return createAuthedMiboApi(token).ignoreHealthIssue(issueId)
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.healthIssues(queryToken),
      })
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.healthSummary(queryToken),
      })
      await queryClient.invalidateQueries({ queryKey: ["home"] })
    },
  })
  const retryIngestMutation = useMutation({
    mutationFn: async (stages: IngestDiagnosticStage[]) => {
      if (stages.length === 0) return []
      const api = createAuthedMiboApi(queryToken)
      return Promise.all(stages.map((stage) => api.retryIngestStage(stage.id)))
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.ingestDiagnostics(queryToken),
      })
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.consoleSummary(queryToken),
      })
      await queryClient.invalidateQueries({ queryKey: ["home"] })
    },
  })
  const resolveReviewMutation = useMutation({
    mutationFn: async (stages: IngestDiagnosticStage[]) => {
      if (stages.length === 0) return []
      const api = createAuthedMiboApi(queryToken)
      return Promise.all(
        stages.map((stage) => api.resolveIngestReviewStage(stage.id))
      )
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.ingestDiagnostics(queryToken),
      })
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.consoleSummary(queryToken),
      })
      await queryClient.invalidateQueries({ queryKey: ["home"] })
    },
  })
  const healthState = getHealthCenterState(issues, {
    validatePending: validateMutation.isPending,
    rescanPending: rescanMutation.isPending,
    ignorePending: ignoreMutation.isPending,
  })

  return (
    <div className="space-y-6">
      <section className="grid gap-4 md:grid-cols-3">
        <HealthStat
          label="阻断问题"
          value={healthState.blockingIssues.length}
          tone="danger"
        />
        <HealthStat
          label="其他问题"
          value={healthState.otherIssues.length}
          tone="warning"
        />
        <HealthStat
          label="活跃问题"
          value={healthState.activeIssueCount}
          tone="neutral"
        />
      </section>

      <IngestDiagnosticsPanel
        stages={ingestQuery.data?.stages ?? []}
        isLoading={ingestQuery.isPending}
        error={ingestQuery.error}
        isRetrying={retryIngestMutation.isPending}
        isResolvingReview={resolveReviewMutation.isPending}
        onRetry={(stages) => retryIngestMutation.mutate(stages)}
        onResolveReview={(stages) => resolveReviewMutation.mutate(stages)}
        onRefetch={() => void ingestQuery.refetch()}
      />

      {issuesQuery.isLoading ? (
        <Card className="rounded-[1.5rem] border-border/60 bg-card/80">
          <CardContent className="flex items-center gap-3 p-6 text-sm text-muted-foreground">
            <LoaderCircleIcon className="size-4 animate-spin" />
            正在加载健康诊断
          </CardContent>
        </Card>
      ) : null}

      {issuesQuery.error ? (
        <Card className="rounded-[1.5rem] border-destructive/30 bg-destructive/5">
          <CardContent className="flex items-center justify-between gap-4 p-6">
            <div>
              <div className="font-medium text-destructive">
                健康诊断加载失败
              </div>
              <div className="mt-1 text-sm text-muted-foreground">
                {issuesQuery.error.message}
              </div>
            </div>
            <Button
              variant="outline"
              onClick={() => void issuesQuery.refetch()}
            >
              重试
            </Button>
          </CardContent>
        </Card>
      ) : null}

      {!issuesQuery.isLoading && !issuesQuery.error && healthState.isEmpty ? (
        <Card className="rounded-[1.5rem] border-emerald-500/30 bg-emerald-500/5">
          <CardContent className="flex items-center gap-3 p-6">
            <CheckCircle2Icon className="size-5 text-emerald-600" />
            <div>
              <div className="font-medium">当前没有需要处理的问题</div>
              <div className="text-sm text-muted-foreground">
                媒体源、媒体库和后台任务暂无活跃健康告警。
              </div>
            </div>
          </CardContent>
        </Card>
      ) : null}

      {healthState.hasBlockingIssues ? (
        <IssueGroup
          title="需要立即处理"
          description="这些问题会阻断扫描、首页展示或关键媒体库能力。"
          issues={healthState.blockingIssues}
          validatePending={validateMutation.isPending}
          validationResults={validationResults}
          rescanPending={rescanMutation.isPending}
          ignorePending={ignoreMutation.isPending}
          onValidate={(mediaSourceId) => validateMutation.mutate(mediaSourceId)}
          onRescan={(issueId) => rescanMutation.mutate(issueId)}
          onIgnore={(issueId) => ignoreMutation.mutate(issueId)}
        />
      ) : null}

      {healthState.hasOtherIssues ? (
        <IssueGroup
          title="其他活跃问题"
          description="这些问题可能影响元数据、探测或后台维护，但不一定阻断首页展示。"
          issues={healthState.otherIssues}
          validatePending={validateMutation.isPending}
          validationResults={validationResults}
          rescanPending={rescanMutation.isPending}
          ignorePending={ignoreMutation.isPending}
          onValidate={(mediaSourceId) => validateMutation.mutate(mediaSourceId)}
          onRescan={(issueId) => rescanMutation.mutate(issueId)}
          onIgnore={(issueId) => ignoreMutation.mutate(issueId)}
        />
      ) : null}
    </div>
  )
}

function IssueGroup({
  title,
  description,
  issues,
  validatePending,
  validationResults,
  rescanPending,
  ignorePending,
  onValidate,
  onRescan,
  onIgnore,
}: {
  title: string
  description: string
  issues: HealthIssue[]
  validatePending: boolean
  validationResults: Record<number, MediaSourceValidationResult>
  rescanPending: boolean
  ignorePending: boolean
  onValidate: (mediaSourceId: number) => void
  onRescan: (issueId: string) => void
  onIgnore: (issueId: string) => void
}) {
  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      </div>
      <div className="grid gap-4">
        {issues.map((issue) => (
          <IssueCard
            key={issue.id}
            issue={issue}
            validatePending={validatePending}
            validationResults={validationResults}
            rescanPending={rescanPending}
            ignorePending={ignorePending}
            onValidate={onValidate}
            onRescan={onRescan}
            onIgnore={onIgnore}
          />
        ))}
      </div>
    </section>
  )
}

function IssueCard({
  issue,
  validatePending,
  validationResults,
  rescanPending,
  ignorePending,
  onValidate,
  onRescan,
  onIgnore,
}: {
  issue: HealthIssue
  validatePending: boolean
  validationResults: Record<number, MediaSourceValidationResult>
  rescanPending: boolean
  ignorePending: boolean
  onValidate: (mediaSourceId: number) => void
  onRescan: (issueId: string) => void
  onIgnore: (issueId: string) => void
}) {
  const affectedNames = affectedLibraryNames(issue)
  return (
    <Card className="overflow-hidden rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="space-y-3 px-5 py-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="space-y-2">
            <Badge
              variant="outline"
              className={healthSeverityClassName(issue.severity)}
            >
              {healthSeverityLabel(issue.severity)}
            </Badge>
            <CardTitle className="flex items-center gap-2 text-xl">
              {issue.severity === "blocking" ? (
                <ShieldAlertIcon className="size-5 text-destructive" />
              ) : (
                <AlertTriangleIcon className="size-5 text-amber-500" />
              )}
              {healthReasonTitle(issue)}
            </CardTitle>
            <CardDescription className="max-w-3xl leading-6">
              {healthReasonMessage(issue)}
            </CardDescription>
          </div>
          <div className="text-sm text-muted-foreground sm:text-right">
            {issue.last_seen_at ? (
              <div>最近发生：{formatDate(issue.last_seen_at)}</div>
            ) : null}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4 px-5 pb-5">
        <div className="grid gap-3 text-sm md:grid-cols-3">
          <ImpactPill label="影响媒体库" value={affectedNames || "未关联"} />
          <ImpactPill
            label="目录项目"
            value={`${issue.impact.affected_metadata_items} 个`}
          />
          <ImpactPill
            label="库存文件"
            value={`${issue.impact.affected_files} 个`}
          />
        </div>
        <IssueErrorSummary issue={issue} />
        <div className="flex flex-wrap gap-2">
          {(issue.actions ?? [])
            .filter((action) => action.type !== "ignore_issue")
            .map((action) => (
              <IssueActionButton
                key={`${action.type}-${action.media_source_id ?? issue.id}`}
                issue={issue}
                action={action}
                validatePending={validatePending}
                rescanPending={rescanPending}
                onValidate={onValidate}
                onRescan={onRescan}
              />
            ))}
          <Button
            variant="ghost"
            disabled={ignorePending}
            onClick={() => onIgnore(issue.id)}
          >
            {ignorePending ? (
              <LoaderCircleIcon className="size-4 animate-spin" />
            ) : null}
            忽略
          </Button>
        </div>
        <ValidationResultPanel
          issue={issue}
          validationResults={validationResults}
        />
        <details className="rounded-[1rem] border border-border/60 bg-background/60 p-4 text-sm">
          <summary className="cursor-pointer font-medium">技术详情</summary>
          <div className="mt-3 space-y-2 text-muted-foreground">
            <div>任务类型：{issue.technical_detail.job_kind || "未知"}</div>
            <div>任务状态：{issue.technical_detail.job_status || "未知"}</div>
            {issue.technical_detail.payload_json ? (
              <pre className="overflow-x-auto rounded-lg bg-muted p-3 text-xs">
                {issue.technical_detail.payload_json}
              </pre>
            ) : null}
            {issue.technical_detail.error_message ? (
              <pre className="overflow-x-auto rounded-lg bg-muted p-3 text-xs whitespace-pre-wrap">
                {issue.technical_detail.error_message}
              </pre>
            ) : null}
          </div>
        </details>
      </CardContent>
    </Card>
  )
}

function IssueErrorSummary({ issue }: { issue: HealthIssue }) {
  const errorMessage = issue.technical_detail.error_message?.trim()
  const jobKind = issue.technical_detail.job_kind?.trim()
  const jobStatus = issue.technical_detail.job_status?.trim()

  if (!errorMessage && !jobKind && !jobStatus) return null

  return (
    <div className="rounded-[1rem] border border-border/60 bg-background/70 p-4 text-sm">
      <div className="flex flex-wrap items-center gap-2">
        <Badge
          variant="outline"
          className="border-destructive/30 bg-destructive/10 text-destructive"
        >
          错误原因
        </Badge>
        {jobKind ? (
          <span className="text-xs text-muted-foreground">任务：{jobKind}</span>
        ) : null}
        {jobStatus ? (
          <span className="text-xs text-muted-foreground">
            状态：{jobStatus}
          </span>
        ) : null}
      </div>
      {errorMessage ? (
        <div className="mt-3 rounded-lg bg-muted/70 p-3 font-mono text-xs leading-5 break-words whitespace-pre-wrap text-foreground">
          {summarizeErrorMessage(errorMessage)}
        </div>
      ) : null}
    </div>
  )
}

function summarizeErrorMessage(message: string) {
  const trimmed = message.trim()
  if (
    trimmed.includes("captcha_invalid") ||
    trimmed.includes("captcha_token expired")
  ) {
    return `OpenList/PikPak 验证过期：${trimmed}`
  }
  return trimmed
}

function ValidationResultPanel({
  issue,
  validationResults,
}: {
  issue: HealthIssue
  validationResults: Record<number, MediaSourceValidationResult>
}) {
  const result = (issue.affected?.media_sources ?? [])
    .map((source) => validationResults[source.id])
    .find(Boolean)

  if (!result) return null

  const isOk = result.status === "ok"
  return (
    <div
      className={
        isOk
          ? "flex items-start gap-3 rounded-[1rem] border border-emerald-500/30 bg-emerald-500/10 p-4 text-sm"
          : "flex items-start gap-3 rounded-[1rem] border border-destructive/30 bg-destructive/10 p-4 text-sm"
      }
    >
      {isOk ? (
        <CheckCircle2Icon className="mt-0.5 size-4 shrink-0 text-emerald-600" />
      ) : (
        <AlertTriangleIcon className="mt-0.5 size-4 shrink-0 text-destructive" />
      )}
      <div>
        <div
          className={
            isOk
              ? "font-medium text-emerald-700 dark:text-emerald-300"
              : "font-medium text-destructive"
          }
        >
          {isOk ? "连接验证成功" : "连接验证失败"}
        </div>
        <div className="mt-1 leading-6 text-muted-foreground">
          {result.message}
        </div>
        {isOk ? (
          <div className="mt-1 text-xs text-muted-foreground">
            该错误已不再发生，刷新后会从活跃问题中移除。是否重新扫描由你决定。
          </div>
        ) : null}
      </div>
    </div>
  )
}

function IssueActionButton({
  issue,
  action,
  validatePending,
  rescanPending,
  onValidate,
  onRescan,
}: {
  issue: HealthIssue
  action: HealthAction
  validatePending: boolean
  rescanPending: boolean
  onValidate: (mediaSourceId: number) => void
  onRescan: (issueId: string) => void
}) {
  if (action.type === "open_external_admin" && action.href) {
    return (
      <Button asChild variant="default">
        <a href={action.href} target="_blank" rel="noreferrer">
          <ExternalLinkIcon className="size-4" />
          {action.label}
        </a>
      </Button>
    )
  }
  if (action.type === "validate_media_source" && action.media_source_id) {
    return (
      <Button
        variant="outline"
        disabled={validatePending}
        onClick={() => onValidate(action.media_source_id!)}
      >
        {validatePending ? (
          <LoaderCircleIcon className="size-4 animate-spin" />
        ) : null}
        {action.label}
      </Button>
    )
  }
  if (action.type === "rescan_affected_libraries") {
    return (
      <Button
        variant="outline"
        disabled={rescanPending}
        onClick={() => onRescan(issue.id)}
      >
        {rescanPending ? (
          <LoaderCircleIcon className="size-4 animate-spin" />
        ) : (
          <RotateCcwIcon className="size-4" />
        )}
        {action.label}
      </Button>
    )
  }
  if (action.type === "view_job") return null
  return null
}

function ImpactPill({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[1rem] border border-border/60 bg-background/60 p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 font-medium text-foreground">{value}</div>
    </div>
  )
}

function HealthStat({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: "danger" | "warning" | "neutral"
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80">
      <CardContent className="p-5">
        <div className="text-sm text-muted-foreground">{label}</div>
        <div
          className={
            tone === "danger"
              ? "mt-2 text-3xl font-semibold text-destructive"
              : tone === "warning"
                ? "mt-2 text-3xl font-semibold text-amber-600"
                : "mt-2 text-3xl font-semibold"
          }
        >
          {value}
        </div>
      </CardContent>
    </Card>
  )
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString("zh-CN", { hour12: false })
}
