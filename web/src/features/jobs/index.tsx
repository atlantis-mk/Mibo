import { useQuery } from "@tanstack/react-query"
import { RefreshCwIcon } from "lucide-react"
import { useState } from "react"

import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card"
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
import type { WorkflowRunStatusView } from "#/lib/mibo-api"
import { workflowsQueryOptions } from "#/lib/mibo-query"
import { useAuthStore } from "#/stores/auth-store"

import { SettingsPageShell } from "#/features/settings/components/settings-page-shell"
import { SETTINGS_SECTIONS } from "#/features/settings/sections"

type WorkflowStatusFilter =
  | "all"
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "cancelled"
  | "superseded"

const pageSize = 25

export default function JobsPage() {
  const token = useAuthStore((state) => state.token)
  const [status, setStatus] = useState<WorkflowStatusFilter>("all")
  const [page, setPage] = useState(1)
  const queryToken = token ?? "guest"
  const filters = {
    limit: pageSize + 1,
    offset: (page - 1) * pageSize,
    status: status === "all" ? undefined : status,
  }
  const section = SETTINGS_SECTIONS.find(({ key }) => key === "jobs")
  const workflowsQuery = useQuery({
    ...workflowsQueryOptions(queryToken, filters),
    enabled: !!token,
  })
  const loadedWorkflows = workflowsQuery.data ?? []
  const workflows = loadedWorkflows.slice(0, pageSize)
  const hasNextPage = loadedWorkflows.length > pageSize
  const hasPreviousPage = page > 1

  if (!section) return null

  return (
    <SettingsPageShell
      icon={section.icon}
      title={section.title}
      description={section.description}
      actions={
        <Button
          variant="outline"
          onClick={() => workflowsQuery.refetch()}
          disabled={workflowsQuery.isFetching}
        >
          <RefreshCwIcon
            className={workflowsQuery.isFetching ? "size-4 animate-spin" : "size-4"}
          />
          刷新
        </Button>
      }
    >
      <Card className="bg-card/80 shadow-sm">
        <CardHeader className="gap-3 sm:flex sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle>Workflow 任务</CardTitle>
            <CardDescription>
              查看资源感知 Workflow DAG 的扫描、物化、探测和元数据任务状态。
            </CardDescription>
          </div>
          <NativeSelect
            value={status}
            onChange={(event) => {
              setPage(1)
              setStatus(event.target.value as WorkflowStatusFilter)
            }}
          >
            <NativeSelectOption value="all">全部状态</NativeSelectOption>
            <NativeSelectOption value="queued">待执行</NativeSelectOption>
            <NativeSelectOption value="running">运行中</NativeSelectOption>
            <NativeSelectOption value="cancelled">已取消</NativeSelectOption>
            <NativeSelectOption value="completed">已完成</NativeSelectOption>
            <NativeSelectOption value="failed">失败</NativeSelectOption>
            <NativeSelectOption value="superseded">已替代</NativeSelectOption>
          </NativeSelect>
        </CardHeader>
        <CardContent>
          {workflowsQuery.isLoading ? (
            <div className="rounded-2xl border border-border/60 bg-muted/40 px-4 py-6 text-sm text-muted-foreground">
              正在加载 workflow 任务...
            </div>
          ) : workflowsQuery.error ? (
            <div className="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-6 text-sm text-destructive">
              {workflowsQuery.error.message}
            </div>
          ) : workflows.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-border/70 px-4 py-8 text-center text-sm text-muted-foreground">
              当前没有符合条件的 workflow 任务。
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>任务</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>阶段</TableHead>
                  <TableHead>时间</TableHead>
                  <TableHead>等待/结果</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {workflows.map((workflow) => (
                  <TableRow key={workflow.run.id}>
                    <TableCell className="min-w-56 whitespace-normal">
                      <div className="font-medium text-foreground">
                        {formatKind(workflow.run.reason)}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        Workflow #{workflow.run.id}
                        {workflow.run.run_key ? ` · ${workflow.run.run_key}` : ""}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusVariant(workflow.run.status)}>
                        {formatStatus(workflow.run.status)}
                      </Badge>
                    </TableCell>
                    <TableCell className="max-w-sm text-sm whitespace-normal text-muted-foreground">
                      {summarizeWorkflowTasks(workflow)}
                    </TableCell>
                    <TableCell className="min-w-44 text-sm whitespace-normal text-muted-foreground">
                      <div>创建：{formatDateTime(workflow.run.created_at)}</div>
                      <div>更新：{formatDateTime(workflow.run.updated_at)}</div>
                    </TableCell>
                    <TableCell className="max-w-sm text-sm whitespace-normal text-muted-foreground">
                      {workflow.run.error_message || summarizeWorkflowWaits(workflow)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          {!workflowsQuery.isLoading &&
          !workflowsQuery.error &&
          loadedWorkflows.length > 0 ? (
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
                        if (hasPreviousPage) setPage((current) => current - 1)
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
                        if (hasNextPage) setPage((current) => current + 1)
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
    queued: "待执行",
    running: "运行中",
    cancelled: "已取消",
    completed: "已完成",
    failed: "失败",
    superseded: "已替代",
    blocked: "阻塞",
    retrying: "重试中",
  }
  return labels[status] ?? status
}

function formatKind(kind: string) {
  const labels: Record<string, string> = {
    library_created: "创建后扫描",
    manual_scan: "手动扫描",
    scheduled_scan: "计划扫描",
    storage_refresh: "存储刷新",
    targeted_refresh: "定向刷新",
    scan: "扫描",
    materialize: "目录物化",
    projection: "投影刷新",
    probe: "媒体探测",
    metadata_match: "元数据匹配",
  }
  return labels[kind] ?? kind
}

function summarizeWorkflowTasks(workflow: WorkflowRunStatusView) {
  const counts = workflow.task_counts ?? []
  if (counts.length === 0) return "暂无阶段任务"
  return counts
    .map(
      (count) =>
        `${formatKind(count.stage)} ${formatStatus(count.status)}×${count.count}`
    )
    .join(" · ")
}

function summarizeWorkflowWaits(workflow: WorkflowRunStatusView) {
  const waits = workflow.resource_waits ?? []
  if (waits.length > 0) {
    return `等待资源：${waits
      .map((wait) => `${wait.resource_key}×${wait.count}`)
      .join(" · ")}`
  }
  return summarizePayload(workflow.run.payload_json)
}

function summarizePayload(value: string) {
  if (!value) return "暂无结果摘要"
  try {
    const payload = JSON.parse(value) as Record<string, unknown>
    const summary = [payload.reason, payload.root_path, payload.library_id]
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
