import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { RefreshCwIcon, RotateCcwIcon, SquareIcon } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card"
import { Checkbox } from "#/components/ui/checkbox"
import { NativeSelect, NativeSelectOption } from "#/components/ui/native-select"
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "#/components/ui/pagination"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "#/components/ui/table"
import type { Job } from "#/lib/mibo-api"
import {
  createAuthedMiboApi,
  jobsQueryOptions,
  miboQueryKeys,
} from "#/lib/mibo-query"
import { useAuthStore } from "#/stores/auth-store"

import { SettingsPageShell } from "#/features/settings/components/settings-page-shell"
import { SETTINGS_SECTIONS } from "#/features/settings/sections"

type JobStatusFilter =
  | "all"
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "cancel_requested"
  | "cancelled"

const pageSize = 25

export default function JobsPage() {
  const token = useAuthStore((state) => state.token)
  const queryClient = useQueryClient()
  const [status, setStatus] = useState<JobStatusFilter>("all")
  const [page, setPage] = useState(1)
  const [selectedJobIds, setSelectedJobIds] = useState<Set<number>>(new Set())
  const queryToken = token ?? "guest"
  const filters = {
    limit: pageSize + 1,
    offset: (page - 1) * pageSize,
    status: status === "all" ? undefined : status,
  }
  const section = SETTINGS_SECTIONS.find(({ key }) => key === "jobs")
  const jobsQuery = useQuery({
    ...jobsQueryOptions(queryToken, filters),
    enabled: !!token,
  })
  const loadedJobs = jobsQuery.data ?? []
  const jobs = loadedJobs.slice(0, pageSize)
  const hasNextPage = loadedJobs.length > pageSize
  const hasPreviousPage = page > 1
  const selectedJobs = jobs.filter((job) => selectedJobIds.has(job.id))
  const selectedCancelableJobs = selectedJobs.filter(canCancel)
  const selectedRetryableJobs = selectedJobs.filter(canRetry)
  const currentPageSelectedCount = jobs.filter((job) =>
    selectedJobIds.has(job.id)
  ).length
  const allCurrentPageSelected =
    jobs.length > 0 && currentPageSelectedCount === jobs.length
  const someCurrentPageSelected = currentPageSelectedCount > 0

  const retryMutation = useMutation({
    mutationFn: (job: Job) => createAuthedMiboApi(queryToken).retryJob(job.id),
    onSuccess: async () => {
      toast.success("任务已重新加入队列")
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.jobs(queryToken, filters),
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const cancelMutation = useMutation({
    mutationFn: (job: Job) => createAuthedMiboApi(queryToken).cancelJob(job.id),
    onSuccess: async (job) => {
      toast.success(
        job.status === "cancel_requested"
          ? "已请求停止运行中任务"
          : "任务已取消"
      )
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.jobs(queryToken, filters),
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const bulkRetryMutation = useMutation({
    mutationFn: async (targets: Job[]) => {
      const api = createAuthedMiboApi(queryToken)
      await Promise.all(targets.map((job) => api.retryJob(job.id)))
      return targets.length
    },
    onSuccess: async (count) => {
      toast.success(`${count} 个任务已重新加入队列`)
      setSelectedJobIds(new Set())
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.jobs(queryToken, filters),
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const bulkCancelMutation = useMutation({
    mutationFn: async (targets: Job[]) => {
      const api = createAuthedMiboApi(queryToken)
      await Promise.all(targets.map((job) => api.cancelJob(job.id)))
      return targets.length
    },
    onSuccess: async (count) => {
      toast.success(`${count} 个任务已提交停止`)
      setSelectedJobIds(new Set())
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.jobs(queryToken, filters),
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const actionPending =
    retryMutation.isPending ||
    cancelMutation.isPending ||
    bulkRetryMutation.isPending ||
    bulkCancelMutation.isPending

  if (!section) return null

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
      actions={
        <Button
          variant="outline"
          onClick={() => jobsQuery.refetch()}
          disabled={jobsQuery.isFetching}
        >
          <RefreshCwIcon
            className={jobsQuery.isFetching ? "size-4 animate-spin" : "size-4"}
          />
          刷新
        </Button>
      }
    >
      <Card className="bg-card/80 shadow-sm">
        <CardHeader className="gap-3 sm:flex sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle>任务队列</CardTitle>
            <CardDescription>
              按页查看后台任务。排队中任务可直接取消，运行中任务会请求 worker
              安全停止。
            </CardDescription>
          </div>
          <NativeSelect
            value={status}
            onChange={(event) => {
              setPage(1)
              setSelectedJobIds(new Set())
              setStatus(event.target.value as JobStatusFilter)
            }}
          >
            <NativeSelectOption value="all">全部状态</NativeSelectOption>
            <NativeSelectOption value="queued">排队中</NativeSelectOption>
            <NativeSelectOption value="running">运行中</NativeSelectOption>
            <NativeSelectOption value="cancel_requested">
              停止中
            </NativeSelectOption>
            <NativeSelectOption value="cancelled">已取消</NativeSelectOption>
            <NativeSelectOption value="completed">已完成</NativeSelectOption>
            <NativeSelectOption value="failed">失败</NativeSelectOption>
          </NativeSelect>
        </CardHeader>
        <CardContent>
          {selectedJobs.length > 0 ? (
            <div className="mb-4 flex flex-col gap-3 rounded-2xl border border-border/70 bg-muted/35 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="text-sm text-muted-foreground">
                已选择 {selectedJobs.length} 个任务 · 可停止{" "}
                {selectedCancelableJobs.length} 个 · 可重试{" "}
                {selectedRetryableJobs.length} 个
              </div>
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={
                    selectedCancelableJobs.length === 0 ||
                    actionPending ||
                    !token
                  }
                  onClick={() =>
                    bulkCancelMutation.mutate(selectedCancelableJobs)
                  }
                >
                  <SquareIcon className="size-3.5" />
                  批量停止
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={
                    selectedRetryableJobs.length === 0 ||
                    actionPending ||
                    !token
                  }
                  onClick={() =>
                    bulkRetryMutation.mutate(selectedRetryableJobs)
                  }
                >
                  <RotateCcwIcon className="size-3.5" />
                  批量重试
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setSelectedJobIds(new Set())}
                >
                  清除选择
                </Button>
              </div>
            </div>
          ) : null}
          {jobsQuery.isLoading ? (
            <div className="rounded-2xl border border-border/60 bg-muted/40 px-4 py-6 text-sm text-muted-foreground">
              正在加载后台任务...
            </div>
          ) : jobsQuery.error ? (
            <div className="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-6 text-sm text-destructive">
              {jobsQuery.error.message}
            </div>
          ) : jobs.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-border/70 px-4 py-8 text-center text-sm text-muted-foreground">
              当前没有符合条件的后台任务。
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10">
                    <Checkbox
                      checked={
                        allCurrentPageSelected
                          ? true
                          : someCurrentPageSelected
                            ? "indeterminate"
                            : false
                      }
                      aria-label="选择当前页任务"
                      onCheckedChange={(checked) => {
                        setSelectedJobIds((current) => {
                          const next = new Set(current)
                          for (const job of jobs) {
                            if (checked === true) {
                              next.add(job.id)
                            } else {
                              next.delete(job.id)
                            }
                          }
                          return next
                        })
                      }}
                    />
                  </TableHead>
                  <TableHead>任务</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>尝试</TableHead>
                  <TableHead>时间</TableHead>
                  <TableHead>结果</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {jobs.map((job) => (
                  <TableRow key={job.id}>
                    <TableCell>
                      <Checkbox
                        checked={selectedJobIds.has(job.id)}
                        aria-label={`选择 Job #${job.id}`}
                        onCheckedChange={(checked) => {
                          setSelectedJobIds((current) => {
                            const next = new Set(current)
                            if (checked === true) {
                              next.add(job.id)
                            } else {
                              next.delete(job.id)
                            }
                            return next
                          })
                        }}
                      />
                    </TableCell>
                    <TableCell className="min-w-48 whitespace-normal">
                      <div className="font-medium text-foreground">
                        {formatKind(job.kind)}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        Job #{job.id}
                        {job.job_key ? ` · ${job.job_key}` : ""}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusVariant(job.status)}>
                        {formatStatus(job.status)}
                      </Badge>
                    </TableCell>
                    <TableCell>{job.attempts} 次</TableCell>
                    <TableCell className="min-w-44 text-sm whitespace-normal text-muted-foreground">
                      <div>创建：{formatDateTime(job.created_at)}</div>
                      <div>更新：{formatDateTime(job.updated_at)}</div>
                    </TableCell>
                    <TableCell className="max-w-sm text-sm whitespace-normal text-muted-foreground">
                      {job.error_message || summarizePayload(job.payload_json)}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={!canCancel(job) || actionPending || !token}
                          onClick={() => cancelMutation.mutate(job)}
                        >
                          <SquareIcon className="size-3.5" />
                          停止
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={!canRetry(job) || actionPending || !token}
                          onClick={() => retryMutation.mutate(job)}
                        >
                          <RotateCcwIcon className="size-3.5" />
                          重试
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          {!jobsQuery.isLoading && !jobsQuery.error && loadedJobs.length > 0 ? (
            <div className="mt-4 flex flex-col gap-3 border-t border-border/60 pt-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="text-sm text-muted-foreground">
                第 {page} 页 · 每页 {pageSize} 条
              </div>
              <Pagination className="mx-0 w-auto justify-start sm:justify-end">
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      text="上一页"
                      href="#"
                      aria-disabled={!hasPreviousPage}
                      className={
                        hasPreviousPage
                          ? undefined
                          : "pointer-events-none opacity-50"
                      }
                      onClick={(event) => {
                        event.preventDefault()
                        if (hasPreviousPage) {
                          setSelectedJobIds(new Set())
                          setPage((current) => current - 1)
                        }
                      }}
                    />
                  </PaginationItem>
                  <PaginationItem>
                    <PaginationNext
                      text="下一页"
                      href="#"
                      aria-disabled={!hasNextPage}
                      className={
                        hasNextPage
                          ? undefined
                          : "pointer-events-none opacity-50"
                      }
                      onClick={(event) => {
                        event.preventDefault()
                        if (hasNextPage) {
                          setSelectedJobIds(new Set())
                          setPage((current) => current + 1)
                        }
                      }}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </SettingsPageShell>
  )
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
    cancel_requested: "停止中",
    cancelled: "已取消",
    completed: "已完成",
    failed: "失败",
  }
  return labels[status] ?? status
}

function canCancel(job: Job) {
  return job.status === "queued" || job.status === "running"
}

function canRetry(job: Job) {
  return job.status !== "running" && job.status !== "cancel_requested"
}

function formatKind(kind: string) {
  const labels: Record<string, string> = {
    sync_library: "同步媒体库",
    targeted_refresh: "定向刷新",
    listener_reconcile: "监听器校准",
    apply_storage_event_refresh: "存储事件刷新",
    catalog_refresh_item_projection: "刷新条目投影",
    catalog_refresh_library_projection: "刷新媒体库投影",
    catalog_match_batch: "批量匹配目录",
    inventory_probe_batch: "批量探测库存",
    match_catalog_item: "匹配目录条目",
    probe_inventory_file: "探测库存文件",
  }
  if (kind.startsWith("schedule_")) return `计划任务：${kind.slice(9)}`
  return labels[kind] ?? kind
}

function summarizePayload(value: string) {
  if (!value) return "暂无结果摘要"
  try {
    const payload = JSON.parse(value) as Record<string, unknown>
    const summary = [payload.kind, payload.scope_kind, payload.library_id]
      .filter((item) => item !== undefined && item !== null && item !== "")
      .join(" · ")
    return summary || "暂无结果摘要"
  } catch {
    return value.length > 120 ? `${value.slice(0, 120)}...` : value
  }
}

function formatDateTime(value?: string) {
  if (!value) return "未记录"
  return new Date(value).toLocaleString("zh-CN", {
    hour12: false,
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  })
}
