import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCwIcon } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import type { WorkflowRunStatusView, WorkflowTask } from '@/lib/mibo-api'
import { workflowsQueryOptions } from '@/lib/mibo-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SettingsPageShell } from '@/features/settings/components/settings-page-shell'

type WorkflowStatusFilter =
  | 'all'
  | 'queued'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'superseded'

const pageSize = 25

export default function JobsPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  const [status, setStatus] = useState<WorkflowStatusFilter>('all')
  const [page, setPage] = useState(1)
  const queryToken = token ?? 'guest'
  const filters = {
    limit: pageSize + 1,
    offset: (page - 1) * pageSize,
    status: status === 'all' ? undefined : status,
  }
  const workflowsQuery = useQuery({
    ...workflowsQueryOptions(queryToken, filters),
    enabled: !!token,
  })
  const loadedWorkflows = workflowsQuery.data ?? []
  const workflows = loadedWorkflows.slice(0, pageSize)
  const hasNextPage = loadedWorkflows.length > pageSize
  const hasPreviousPage = page > 1

  return (
    <SettingsPageShell
      icon={RefreshCwIcon}
      title='后台任务'
      description='查看资源感知 Workflow DAG 的扫描、物化、探测和元数据任务状态。'
      fixedContent
      showHeader={false}
    >
      <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
        <header className='flex flex-col gap-3 pb-4 sm:flex-row sm:items-start sm:justify-between'>
          <div>
            <h2 className='text-lg font-semibold tracking-tight'>
              Workflow 任务
            </h2>
            <p className='mt-1 text-sm text-muted-foreground'>
              任务列表会填满当前页面高度，分页保持固定，滚动发生在表格内部。
            </p>
          </div>
          <div className='flex flex-wrap items-center gap-2 sm:shrink-0 sm:justify-end'>
            <NativeSelect
              value={status}
              onChange={(event) => {
                setPage(1)
                setStatus(event.target.value as WorkflowStatusFilter)
              }}
            >
              <NativeSelectOption value='all'>全部状态</NativeSelectOption>
              <NativeSelectOption value='queued'>待执行</NativeSelectOption>
              <NativeSelectOption value='running'>运行中</NativeSelectOption>
              <NativeSelectOption value='cancelled'>已取消</NativeSelectOption>
              <NativeSelectOption value='completed'>已完成</NativeSelectOption>
              <NativeSelectOption value='failed'>失败</NativeSelectOption>
              <NativeSelectOption value='superseded'>已替代</NativeSelectOption>
            </NativeSelect>
            <Button
              variant='outline'
              onClick={() => workflowsQuery.refetch()}
              disabled={workflowsQuery.isFetching}
            >
              <RefreshCwIcon
                className={
                  workflowsQuery.isFetching ? 'size-4 animate-spin' : 'size-4'
                }
              />
              刷新
            </Button>
          </div>
        </header>
        <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
          {workflowsQuery.isLoading ? (
            <div className='rounded-2xl border border-border/60 bg-muted/40 px-4 py-6 text-sm text-muted-foreground'>
              正在加载 workflow 任务...
            </div>
          ) : workflowsQuery.error ? (
            <div className='rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-6 text-sm text-destructive'>
              {workflowsQuery.error.message}
            </div>
          ) : workflows.length === 0 ? (
            <div className='rounded-2xl border border-dashed border-border/70 px-4 py-8 text-center text-sm text-muted-foreground'>
              当前没有符合条件的 workflow 任务。
            </div>
          ) : (
            <div className='min-h-0 flex-1 overflow-auto'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>运行</TableHead>
                    <TableHead>触发原因</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>任务概览</TableHead>
                    <TableHead>最近任务</TableHead>
                    <TableHead>时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {workflows.map((workflow) => (
                    <TableRow key={workflow.run.id}>
                      <TableCell className='min-w-56 whitespace-normal'>
                        <div className='font-medium text-foreground'>
                          Workflow #{workflow.run.id}
                        </div>
                        <div className='text-sm text-muted-foreground'>
                          {workflow.run.run_key || '未提供 run_key'}
                        </div>
                        <div className='text-xs text-muted-foreground'>
                          库 #{workflow.run.library_id} ·{' '}
                          {workflow.run.scope_key}
                        </div>
                      </TableCell>
                      <TableCell className='min-w-52 whitespace-normal'>
                        <div className='font-medium text-foreground'>
                          {formatKind(workflow.run.reason)}
                        </div>
                        <div className='text-xs text-muted-foreground'>
                          {workflow.run.reason}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className='space-y-2'>
                          <Badge variant={statusVariant(workflow.run.status)}>
                            {formatStatus(workflow.run.status)}
                          </Badge>
                          {workflow.run.error_message ? (
                            <div className='max-w-xs text-xs whitespace-normal text-destructive'>
                              {workflow.run.error_message}
                            </div>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell className='max-w-2xl min-w-80 text-sm whitespace-normal text-muted-foreground'>
                        <div>{summarizeWorkflowTasks(workflow)}</div>
                        <div>{summarizeWorkflowWaits(workflow)}</div>
                      </TableCell>
                      <TableCell className='max-w-2xl min-w-80 text-sm whitespace-normal text-muted-foreground'>
                        {renderRecentTask(workflow.recent_tasks?.[0] ?? null)}
                      </TableCell>
                      <TableCell className='min-w-56 text-sm whitespace-normal text-muted-foreground'>
                        <div>
                          创建：{formatDateTime(workflow.run.created_at)}
                        </div>
                        <div>
                          开始：{formatDateTime(workflow.run.started_at)}
                        </div>
                        <div>
                          结束：{formatDateTime(workflow.run.finished_at)}
                        </div>
                        <div>
                          更新：{formatDateTime(workflow.run.updated_at)}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
          {!workflowsQuery.isLoading &&
          !workflowsQuery.error &&
          loadedWorkflows.length > 0 ? (
            <div className='mt-4 flex flex-col gap-3 border-t border-border/60 pt-4 sm:flex-row sm:items-center sm:justify-between'>
              <div className='text-sm text-muted-foreground'>
                第 {page} 页 · 每页 {pageSize} 条
              </div>
              <Pagination className='mx-0 w-auto justify-start sm:justify-end'>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      text='上一页'
                      href='#'
                      aria-disabled={!hasPreviousPage}
                      className={
                        hasPreviousPage
                          ? undefined
                          : 'pointer-events-none opacity-50'
                      }
                      onClick={(event) => {
                        event.preventDefault()
                        if (hasPreviousPage) setPage((current) => current - 1)
                      }}
                    />
                  </PaginationItem>
                  <PaginationItem>
                    <PaginationNext
                      text='下一页'
                      href='#'
                      aria-disabled={!hasNextPage}
                      className={
                        hasNextPage
                          ? undefined
                          : 'pointer-events-none opacity-50'
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
        </div>
      </div>
    </SettingsPageShell>
  )
}

function statusVariant(status: string) {
  if (status === 'completed') return 'default'
  if (status === 'failed') return 'destructive'
  if (status === 'running') return 'secondary'
  return 'outline'
}

function formatStatus(status: string) {
  const labels: Record<string, string> = {
    queued: '待执行',
    running: '运行中',
    cancelled: '已取消',
    completed: '已完成',
    failed: '失败',
    superseded: '已替代',
    blocked: '阻塞',
    retrying: '重试中',
  }
  return labels[status] ?? status
}

function formatKind(kind: string) {
  const labels: Record<string, string> = {
    library_created: '创建后扫描',
    manual_scan: '手动扫描',
    scheduled_scan: '计划扫描',
    storage_refresh: '存储刷新',
    targeted_refresh: '定向刷新',
    ingest_dispatch: '增量分发',
    scan: '扫描',
    materialize: '目录物化',
    projection: '投影刷新',
    refresh_projection: '投影刷新',
    probe: '媒体探测',
    metadata_match: '元数据匹配',
  }
  return labels[kind] ?? kind
}

function summarizeWorkflowTasks(workflow: WorkflowRunStatusView) {
  const counts = workflow.task_counts ?? []
  if (counts.length === 0) return '暂无阶段任务'
  return counts
    .map(
      (count) =>
        `${formatKind(count.stage)} ${formatStatus(count.status)}×${count.count}`
    )
    .join(' · ')
}

function summarizeWorkflowWaits(workflow: WorkflowRunStatusView) {
  const waits = workflow.resource_waits ?? []
  if (waits.length > 0) {
    return `等待资源：${waits
      .map((wait) => `${wait.resource_key}×${wait.count}`)
      .join(' · ')}`
  }
  return `载荷：${summarizePayload(workflow.run.payload_json)}`
}

function renderRecentTask(task: WorkflowTask | null) {
  if (!task) return '暂无最近任务'

  return (
    <div className='space-y-1'>
      <div className='font-medium text-foreground'>
        {formatKind(task.task_type || task.stage)}
      </div>
      <div>
        阶段：{formatKind(task.stage)} · {formatStatus(task.status)}
      </div>
      <div>
        尝试：{task.attempts}/{task.max_attempts}
      </div>
      <div>{summarizePayload(task.payload_json)}</div>
    </div>
  )
}

function summarizePayload(value: string) {
  if (!value) return '暂无结果摘要'
  try {
    const payload = JSON.parse(value) as Record<string, unknown>
    const summary = [
      typeof payload.reason === 'string'
        ? formatKind(payload.reason)
        : undefined,
      typeof payload.root_path === 'string' && payload.root_path
        ? payload.root_path
        : undefined,
      payload.library_id !== undefined && payload.library_id !== null
        ? `库 #${String(payload.library_id)}`
        : undefined,
    ]
      .filter((item) => item !== undefined && item !== null && item !== '')
      .join(' · ')
    return summary || '暂无结果摘要'
  } catch {
    return value.length > 120 ? `${value.slice(0, 120)}...` : value
  }
}

function formatDateTime(value?: string) {
  if (!value) return '未记录'
  return new Date(value).toLocaleString('zh-CN', {
    hour12: false,
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
