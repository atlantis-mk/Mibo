import { useEffect, useMemo, useState } from "react"
import { Link } from "@tanstack/react-router"
import { AlertTriangleIcon } from "lucide-react"

import { Button } from "#/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card"
import { Checkbox } from "#/components/ui/checkbox"
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
import type { IngestDiagnosticStage } from "#/lib/mibo-api"
import { cn } from "#/lib/utils"

export function IngestDiagnosticsPanel({
  stages,
  isLoading,
  error,
  isRetrying,
  isResolvingReview,
  onRetry,
  onResolveReview,
  onRefetch,
}: {
  stages: IngestDiagnosticStage[]
  isLoading: boolean
  error: Error | null
  isRetrying: boolean
  isResolvingReview: boolean
  onRetry: (stages: IngestDiagnosticStage[]) => void
  onResolveReview: (stages: IngestDiagnosticStage[]) => void
  onRefetch: () => void
}) {
  const pageSize = 10
  const [page, setPage] = useState(1)
  const [selectedStageIds, setSelectedStageIds] = useState<Set<number>>(
    () => new Set()
  )
  const issueStages = useMemo(
    () =>
      stages.filter(
        (stage) =>
          stage.retry_eligible ||
          stage.stale ||
          stage.status === "failed" ||
          stage.status === "review_required"
      ),
    [stages]
  )
  const pageCount = Math.max(1, Math.ceil(issueStages.length / pageSize))
  const visibleStages = issueStages.slice(
    (page - 1) * pageSize,
    page * pageSize
  )
  const selectedStages = issueStages.filter((stage) =>
    selectedStageIds.has(stage.id)
  )
  const selectedRetryableStages = selectedStages.filter(
    (stage) => stage.retry_eligible
  )
  const selectedResolvableStages = selectedStages.filter(
    isResolvableReviewStage
  )
  const allVisibleSelected =
    visibleStages.length > 0 &&
    visibleStages.every((stage) => selectedStageIds.has(stage.id))
  const someVisibleSelected = visibleStages.some((stage) =>
    selectedStageIds.has(stage.id)
  )
  const actionPending = isRetrying || isResolvingReview

  useEffect(() => {
    setPage((current) => Math.min(current, pageCount))
  }, [pageCount])

  useEffect(() => {
    const validIds = new Set(issueStages.map((stage) => stage.id))
    setSelectedStageIds((current) => {
      const next = new Set<number>()
      for (const id of current) {
        if (validIds.has(id)) next.add(id)
      }
      return next.size === current.size ? current : next
    })
  }, [issueStages])

  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 shadow-sm">
      <CardHeader className="flex flex-row items-center justify-between gap-3">
        <div>
          <CardTitle className="flex items-center gap-2">
            <AlertTriangleIcon className="size-5 text-amber-500" />
            媒体整理诊断
          </CardTitle>
          <CardDescription className="mt-1">
            处理整理失败、过期阶段和需要人工确认的媒体。
          </CardDescription>
        </div>
        <span className="text-xs text-muted-foreground">
          {isLoading ? "加载中" : `${issueStages.length} 个关注项`}
        </span>
      </CardHeader>
      <CardContent className="space-y-3">
        {error ? (
          <div className="flex flex-col gap-3 rounded-2xl border border-destructive/30 bg-destructive/5 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <div className="font-medium text-destructive">
                整理诊断加载失败
              </div>
              <div className="mt-1 text-sm text-muted-foreground">
                {error.message}
              </div>
            </div>
            <Button variant="outline" onClick={onRefetch}>
              重试
            </Button>
          </div>
        ) : null}
        {selectedStages.length > 0 ? (
          <div className="flex flex-col gap-3 rounded-2xl border border-border bg-muted/40 p-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-muted-foreground">
              已选择 {selectedStages.length} 项
            </p>
            <div className="flex flex-wrap gap-2 sm:justify-end">
              <Button
                size="sm"
                variant="outline"
                disabled={selectedRetryableStages.length === 0 || actionPending}
                onClick={() => onRetry(selectedRetryableStages)}
              >
                批量重试 ({selectedRetryableStages.length})
              </Button>
              <Button
                size="sm"
                disabled={
                  selectedResolvableStages.length === 0 || actionPending
                }
                onClick={() => onResolveReview(selectedResolvableStages)}
              >
                批量标记 ({selectedResolvableStages.length})
              </Button>
              <Button
                size="sm"
                variant="ghost"
                disabled={actionPending}
                onClick={() => setSelectedStageIds(new Set())}
              >
                清除选择
              </Button>
            </div>
          </div>
        ) : null}
        {!error && issueStages.length === 0 ? (
          <div className="rounded-2xl border border-dashed p-6 text-sm text-muted-foreground">
            暂无失败、过期或待确认的整理阶段。
          </div>
        ) : null}
        {issueStages.length > 0 ? (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-10">
                      <Checkbox
                        checked={
                          allVisibleSelected
                            ? true
                            : someVisibleSelected
                              ? "indeterminate"
                              : false
                        }
                        aria-label="选择当前页整理诊断项"
                        onCheckedChange={(checked) => {
                          setSelectedStageIds((current) => {
                            const next = new Set(current)
                            for (const stage of visibleStages) {
                              if (checked === true) {
                                next.add(stage.id)
                              } else {
                                next.delete(stage.id)
                              }
                            }
                            return next
                          })
                        }}
                      />
                    </TableHead>
                    <TableHead>阶段</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>媒体</TableHead>
                    <TableHead>原因</TableHead>
                    <TableHead>更新时间</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visibleStages.map((stage) => (
                    <TableRow
                      key={stage.id}
                      data-state={
                        selectedStageIds.has(stage.id) ? "selected" : undefined
                      }
                    >
                      <TableCell>
                        <Checkbox
                          checked={selectedStageIds.has(stage.id)}
                          aria-label={`选择整理诊断项 #${stage.id}`}
                          onCheckedChange={(checked) => {
                            setSelectedStageIds((current) => {
                              const next = new Set(current)
                              if (checked === true) {
                                next.add(stage.id)
                              } else {
                                next.delete(stage.id)
                              }
                              return next
                            })
                          }}
                        />
                      </TableCell>
                      <TableCell>
                        <span className="rounded-full border border-border bg-card px-2 py-1 text-xs font-medium">
                          {stageLabel(stage.condition_type)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1.5">
                          <span
                            className={cn(
                              "rounded-full px-2 py-1 text-xs font-medium",
                              stage.status === "failed"
                                ? "bg-red-500/15 text-red-600"
                                : stage.status === "review_required"
                                  ? "bg-amber-500/15 text-amber-600"
                                  : "bg-muted text-muted-foreground"
                            )}
                          >
                            {stageStatusLabel(stage.status)}
                          </span>
                          {stage.stale ? (
                            <span className="rounded-full bg-orange-500/15 px-2 py-1 text-xs font-medium text-orange-600">
                              已过期
                            </span>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell className="max-w-sm min-w-64 whitespace-normal">
                        <div className="font-medium text-foreground">
                          {stage.catalog_title ||
                            stage.storage_path ||
                            stage.unit_key}
                        </div>
                        <div className="mt-1 text-xs text-muted-foreground">
                          {stage.library_name || `媒体库 #${stage.library_id}`}
                          {stage.inventory_file_id
                            ? ` · 文件 #${stage.inventory_file_id}`
                            : ""}
                        </div>
                      </TableCell>
                      <TableCell className="max-w-sm text-xs whitespace-normal text-muted-foreground">
                        {stage.message || stage.reason || "等待整理状态收敛"}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatDate(stage.updated_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex flex-wrap justify-end gap-2">
                          {isResolvableReviewStage(stage) ? (
                            <Button
                              size="sm"
                              disabled={isResolvingReview}
                              onClick={() => onResolveReview([stage])}
                            >
                              {stage.reason === "classification_needs_review"
                                ? "确认分类"
                                : "标记已治理"}
                            </Button>
                          ) : null}
                          {stage.catalog_item_id ? (
                            <Button size="sm" variant="outline" asChild>
                              <Link
                                to="/settings/metadata/$id"
                                params={{ id: String(stage.catalog_item_id) }}
                              >
                                治理元数据
                              </Link>
                            </Button>
                          ) : null}
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={!stage.retry_eligible || isRetrying}
                            onClick={() => onRetry([stage])}
                          >
                            重试阶段
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <div className="flex flex-col gap-3 border-t border-border/60 pt-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="text-sm text-muted-foreground">
                第 {page} / {pageCount} 页 · 每页 {pageSize} 条
              </div>
              <Pagination className="mx-0 w-auto justify-start sm:justify-end">
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      text="上一页"
                      href="#"
                      aria-disabled={page <= 1}
                      className={
                        page > 1 ? undefined : "pointer-events-none opacity-50"
                      }
                      onClick={(event) => {
                        event.preventDefault()
                        setPage((current) => Math.max(1, current - 1))
                      }}
                    />
                  </PaginationItem>
                  <PaginationItem>
                    <PaginationNext
                      text="下一页"
                      href="#"
                      aria-disabled={page >= pageCount}
                      className={
                        page < pageCount
                          ? undefined
                          : "pointer-events-none opacity-50"
                      }
                      onClick={(event) => {
                        event.preventDefault()
                        setPage((current) => Math.min(pageCount, current + 1))
                      }}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          </>
        ) : null}
      </CardContent>
    </Card>
  )
}

function isResolvableReviewStage(stage: IngestDiagnosticStage) {
  if (
    stage.condition_type !== "review_required" ||
    stage.status !== "review_required"
  ) {
    return false
  }
  if (stage.reason === "classification_needs_review") {
    return Boolean(stage.inventory_file_id)
  }
  if (
    stage.reason === "metadata_no_candidate" ||
    stage.reason === "metadata_needs_review"
  ) {
    return Boolean(stage.catalog_item_id)
  }
  return false
}

function stageLabel(stage: string) {
  const labels: Record<string, string> = {
    materialized: "识别媒体",
    probed: "分析视频",
    metadata_matched: "匹配元数据",
    projection_current: "更新视图",
    review_required: "人工确认",
    visible: "可见性",
  }
  return labels[stage] ?? stage
}

function stageStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "等待中",
    running: "运行中",
    failed: "失败",
    review_required: "待确认",
    skipped: "已跳过",
    true: "完成",
    false: "未完成",
  }
  return labels[status] ?? status
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString("zh-CN", { hour12: false })
}
