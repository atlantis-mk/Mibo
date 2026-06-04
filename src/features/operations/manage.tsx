import { lazy, Suspense, useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  CheckCircle2Icon,
  ChevronRightIcon,
  LoaderCircleIcon,
  SearchIcon,
  WrenchIcon,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import type {
  OperationsActionResult,
  OperationsIssue,
  OperationsIssueAction,
  OperationsIssueActionType,
  OperationsIssueKind,
  OperationsIssueLifecycleStatus,
  OperationsTask,
  OperationsTaskKind,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  librariesQueryOptions,
  miboQueryKeys,
  operationsIssueDetailQueryOptions,
  operationsIssueListQueryOptions,
  operationsIssueEventsQueryOptions,
  operationsTaskListQueryOptions,
} from '@/lib/mibo-query'
import {
  operationsSeverityClassName,
  operationsSeverityLabel,
} from '@/lib/operations-presentation'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'

const MetadataReviewDialog = lazy(() =>
  import('@/features/operations/metadata-review-dialog').then((module) => ({
    default: module.MetadataReviewDialog,
  }))
)

const PAGE_SIZE = 20

const ISSUE_STATUS_OPTIONS: Array<{
  value: 'all' | OperationsIssueLifecycleStatus
  label: string
}> = [
  { value: 'all', label: '全部状态' },
  { value: 'active', label: '待处理' },
  { value: 'in_progress', label: '处理中' },
  { value: 'reopened', label: '重新出现' },
  { value: 'resolved', label: '已解决' },
  { value: 'ignored', label: '已忽略' },
]

const ISSUE_KIND_OPTIONS: Array<{
  value: 'all' | OperationsIssueKind
  label: string
}> = [
  { value: 'all', label: '全部问题类型' },
  { value: 'metadata', label: '元数据治理' },
  { value: 'classification', label: '分类确认' },
  { value: 'probe', label: '探测修复' },
  { value: 'workflow', label: '扫描与工作流' },
  { value: 'projection', label: '投影刷新' },
  { value: 'storage', label: '媒体源连接' },
]

const ISSUE_ACTION_OPTIONS: Array<{
  value: 'all' | OperationsIssueActionType
  label: string
}> = [
  { value: 'all', label: '全部动作' },
  { value: 'retry', label: '批量重试' },
  { value: 'apply_candidate', label: '应用候选' },
  { value: 'mark_governed', label: '标记已治理' },
  { value: 'accept_classification', label: '接受分类' },
  { value: 'correct_classification', label: '修正分类' },
  { value: 'relink_resource', label: '重关联资源' },
  { value: 'unlink_resource', label: '解除资源关联' },
  { value: 'exclude', label: '排除文件' },
  { value: 'ignore', label: '忽略问题' },
]

const LEGACY_TASK_KIND_OPTIONS: Array<{
  value: 'all' | OperationsTaskKind
  label: string
}> = [
  { value: 'all', label: '全部任务类型' },
  { value: 'maintenance_backlog', label: '整理流水线' },
  { value: 'scan_blocked', label: '扫描受阻' },
  { value: 'metadata_review_required', label: '元数据确认' },
  { value: 'classification_review_required', label: '分类确认' },
  { value: 'projection_stale', label: '投影刷新' },
  { value: 'storage_access_required', label: '媒体源连接' },
]

type ActionDialogState = {
  issue: OperationsIssue
  action: OperationsIssueAction
}

export default function OperationsManagePage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const role = useAuthStore((state) => state.auth.user?.role)
  const queryToken = token ?? 'guest'
  const isAdmin = role === 'admin'
  const queryClient = useQueryClient()

  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [issueStatus, setIssueStatus] = useState<
    'all' | OperationsIssueLifecycleStatus
  >('active')
  const [issueKind, setIssueKind] = useState<'all' | OperationsIssueKind>('all')
  const [issueActionType, setIssueActionType] = useState<
    'all' | OperationsIssueActionType
  >('all')
  const [libraryFilter, setLibraryFilter] = useState<'all' | `${number}`>('all')
  const [selectedIssueId, setSelectedIssueId] = useState<number | null>(null)
  const [activeActionKey, setActiveActionKey] = useState<string | null>(null)
  const [actionDialog, setActionDialog] = useState<ActionDialogState | null>(
    null
  )
  const [actionReason, setActionReason] = useState('')
  const [targetMetadataItemID, setTargetMetadataItemID] = useState('')
  const [sourceMetadataItemID, setSourceMetadataItemID] = useState('')
  const [resourceRole, setResourceRole] = useState('')
  const [resourceMode, setResourceMode] = useState('move')
  const [classificationTargetKind, setClassificationTargetKind] = useState('')
  const [classificationTargetKey, setClassificationTargetKey] = useState('')
  const [classificationRole, setClassificationRole] = useState('')
  const [metadataReviewTask, setMetadataReviewTask] =
    useState<OperationsTask | null>(null)
  const [metadataReviewIssue, setMetadataReviewIssue] =
    useState<OperationsIssue | null>(null)
  const [legacyTaskKind, setLegacyTaskKind] = useState<
    'all' | OperationsTaskKind
  >('all')

  const issueFilters = useMemo(
    () => ({
      page,
      page_size: PAGE_SIZE,
      status: issueStatus === 'all' ? undefined : issueStatus,
      kind: issueKind === 'all' ? undefined : issueKind,
      action_type: issueActionType === 'all' ? undefined : issueActionType,
      library_id:
        libraryFilter === 'all'
          ? undefined
          : Number.parseInt(libraryFilter, 10),
      q: searchQuery || undefined,
    }),
    [issueActionType, issueKind, issueStatus, libraryFilter, page, searchQuery]
  )

  const issuesQuery = useQuery({
    ...operationsIssueListQueryOptions(queryToken, issueFilters),
    enabled: hasHydrated && !!token,
  })

  const librariesQuery = useQuery({
    ...librariesQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })

  const tasksQuery = useQuery({
    ...operationsTaskListQueryOptions(queryToken, {
      page,
      page_size: PAGE_SIZE,
      lifecycle_status: 'active',
      kind: legacyTaskKind === 'all' ? undefined : legacyTaskKind,
      q: searchQuery || undefined,
    }),
    enabled: hasHydrated && !!token && issuesQuery.isError,
  })

  const issues = useMemo(
    () => issuesQuery.data?.items ?? [],
    [issuesQuery.data?.items]
  )
  const total = issuesQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const usingLegacyFallback = !!issuesQuery.error

  useEffect(() => {
    if (page > totalPages) {
      setPage(totalPages)
    }
  }, [page, totalPages])

  useEffect(() => {
    if (
      selectedIssueId &&
      issues.every((issue) => issue.id !== selectedIssueId)
    ) {
      setSelectedIssueId(null)
    }
  }, [issues, selectedIssueId])

  const selectedIssue = useMemo(
    () => issues.find((issue) => issue.id === selectedIssueId) ?? null,
    [issues, selectedIssueId]
  )

  const issueDetailQuery = useQuery({
    ...operationsIssueDetailQueryOptions(queryToken, selectedIssueId ?? 0),
    enabled: hasHydrated && !!token && !!selectedIssueId,
  })

  const issueEventsQuery = useQuery({
    ...operationsIssueEventsQueryOptions(queryToken, selectedIssueId ?? 0),
    enabled: hasHydrated && !!token && !!selectedIssueId,
  })

  const executeActionMutation = useMutation({
    mutationFn: async ({
      issueId,
      action,
      payload,
    }: {
      issueId: number
      action: OperationsIssueAction
      payload: Parameters<
        ReturnType<typeof createAuthedMiboApi>['executeOperationsIssueAction']
      >[1]
    }) => {
      if (!token) throw new Error('当前未登录，无法执行治理操作。')
      setActiveActionKey(action.action_key)
      return createAuthedMiboApi(token).executeOperationsIssueAction(
        issueId,
        payload
      )
    },
    onSuccess: async (result) => {
      showActionResultToast(result)
      setActionDialog(null)
      await invalidateOperationsQueries(queryClient, queryToken)
      if (selectedIssueId) {
        await queryClient.invalidateQueries({
          queryKey: miboQueryKeys.operationsIssueDetail(
            queryToken,
            selectedIssueId
          ),
        })
        await queryClient.invalidateQueries({
          queryKey: miboQueryKeys.operationsIssueEvents(
            queryToken,
            selectedIssueId
          ),
        })
      }
    },
    onError: (error) => {
      toast.error('操作失败', {
        description: error instanceof Error ? error.message : '请稍后重试。',
      })
    },
    onSettled: () => {
      setActiveActionKey(null)
    },
  })

  const issueRows = useMemo(
    () =>
      issues.map((issue) => ({
        issue,
        sampleLabel:
          issue.samples?.[0]?.label ||
          issue.targets?.[0]?.label ||
          '未记录样本',
        actions: issue.actions ?? [],
      })),
    [issues]
  )

  const detailIssue = issueDetailQuery.data ?? selectedIssue
  const detailIssueActions = useMemo(
    () => detailIssue?.actions?.filter(isDetailIssueAction) ?? [],
    [detailIssue?.actions]
  )
  const detailEvents = issueEventsQuery.data ?? detailIssue?.events ?? []
  const metadataTaskFromIssue = useMemo(
    () => (detailIssue ? issueToMetadataReviewTask(detailIssue) : null),
    [detailIssue]
  )
  const metadataReviewIssueAction = useMemo(
    () =>
      metadataReviewIssue?.actions?.find(
        (action) =>
          action.action_type === 'mark_governed' &&
          action.action_key === 'issue_mark_governed' &&
          action.eligible
      ) ?? null,
    [metadataReviewIssue]
  )

  const isIssueLoading = issuesQuery.isLoading

  return (
    <div className='flex h-full min-h-0 flex-col overflow-hidden'>
      <Card className='flex h-full min-h-0 flex-col border-border/60 bg-card/85 shadow-sm'>
        <CardHeader className='gap-4'>
          <div className='flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between'>
            <div className='flex flex-wrap items-center gap-3'>
              <Button asChild variant='ghost' size='sm'>
                <Link to='/settings/operations'>
                  <ArrowLeftIcon className='size-4' />
                  返回运营概览
                </Link>
              </Button>
              <CardTitle className='flex items-center gap-2 text-xl'>
                <WrenchIcon className='size-5 text-amber-500' />
                治理工作台
              </CardTitle>
            </div>
            <div className='flex flex-wrap items-center gap-2'>
              <Badge variant='outline' className='rounded-full px-3 py-1'>
                {usingLegacyFallback ? 'Legacy Task Fallback' : 'Issue Inbox'}
              </Badge>
              <Badge variant='outline' className='rounded-full px-3 py-1'>
                当前结果{' '}
                {usingLegacyFallback ? (tasksQuery.data?.total ?? 0) : total} 项
              </Badge>
            </div>
          </div>

          <FieldGroup>
            <div className='grid gap-3 lg:grid-cols-[minmax(0,1.3fr)_180px_180px_180px_160px]'>
              <Field>
                <FieldLabel htmlFor='operations-manage-search'>搜索</FieldLabel>
                <div className='flex gap-2'>
                  <Input
                    id='operations-manage-search'
                    value={searchInput}
                    onChange={(event) => setSearchInput(event.target.value)}
                    placeholder='按标题、范围、样本或动作搜索'
                  />
                  <Button
                    variant='outline'
                    onClick={() => {
                      setPage(1)
                      setSearchQuery(searchInput.trim())
                    }}
                  >
                    <SearchIcon className='size-4' />
                    搜索
                  </Button>
                </div>
              </Field>

              <Field>
                <FieldLabel>状态</FieldLabel>
                <Select
                  value={issueStatus}
                  onValueChange={(value) => {
                    setPage(1)
                    setIssueStatus(
                      value as 'all' | OperationsIssueLifecycleStatus
                    )
                  }}
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder='全部状态' />
                  </SelectTrigger>
                  <SelectContent>
                    {ISSUE_STATUS_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>

              <Field>
                <FieldLabel>问题类型</FieldLabel>
                <Select
                  value={issueKind}
                  onValueChange={(value) => {
                    setPage(1)
                    setIssueKind(value as 'all' | OperationsIssueKind)
                    if (usingLegacyFallback) {
                      setLegacyTaskKind(
                        mapLegacyTaskKind(value as 'all' | OperationsIssueKind)
                      )
                    }
                  }}
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder='全部问题类型' />
                  </SelectTrigger>
                  <SelectContent>
                    {ISSUE_KIND_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>

              <Field>
                <FieldLabel>动作类型</FieldLabel>
                <Select
                  value={issueActionType}
                  onValueChange={(value) => {
                    setPage(1)
                    setIssueActionType(
                      value as 'all' | OperationsIssueActionType
                    )
                  }}
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder='全部动作' />
                  </SelectTrigger>
                  <SelectContent>
                    {ISSUE_ACTION_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>

              <Field>
                <FieldLabel>媒体库</FieldLabel>
                <Select
                  value={libraryFilter}
                  onValueChange={(value) => {
                    setPage(1)
                    setLibraryFilter(value as 'all' | `${number}`)
                  }}
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder='全部媒体库' />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='all'>全部媒体库</SelectItem>
                    {(librariesQuery.data ?? []).map((library) => (
                      <SelectItem key={library.id} value={String(library.id)}>
                        {library.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            </div>
          </FieldGroup>
        </CardHeader>

        <CardContent className='flex min-h-0 flex-1 flex-col gap-4 overflow-hidden'>
          {usingLegacyFallback ? (
            <Alert variant='destructive'>
              <AlertTriangleIcon className='size-4' />
              <AlertTitle>Issue API 当前不可用</AlertTitle>
              <AlertDescription>
                {issuesQuery.error?.message ||
                  '问题接口返回失败，已回退到旧任务视图。'}
              </AlertDescription>
            </Alert>
          ) : null}

          {isIssueLoading ? (
            <div className='flex items-center gap-2 text-sm text-muted-foreground'>
              <LoaderCircleIcon className='size-4 animate-spin' />
              正在加载治理问题
            </div>
          ) : null}

          {!usingLegacyFallback ? (
            <>
              <div className='min-h-0 flex-1 overflow-hidden rounded-xl border border-border/60 bg-background/70'>
                <div className='h-full overflow-auto'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>问题</TableHead>
                        <TableHead>范围</TableHead>
                        <TableHead>影响</TableHead>
                        <TableHead>样本</TableHead>
                        <TableHead>动作</TableHead>
                        <TableHead className='w-44'>详情</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {issueRows.length === 0 ? (
                        <TableRow>
                          <TableCell
                            colSpan={6}
                            className='py-8 text-center text-sm text-muted-foreground'
                          >
                            当前筛选条件下没有待处理问题。
                          </TableCell>
                        </TableRow>
                      ) : (
                        issueRows.map(({ issue, sampleLabel, actions }) => {
                          const ignoreAction = findIssueIgnoreAction(actions)
                          const ignoreRunning =
                            executeActionMutation.isPending &&
                            activeActionKey === ignoreAction?.action_key

                          return (
                            <TableRow
                              key={issue.id}
                              className='cursor-pointer'
                              onClick={() => setSelectedIssueId(issue.id)}
                            >
                              <TableCell className='align-top'>
                                <div className='space-y-2'>
                                  <div className='flex flex-wrap items-center gap-2'>
                                    <span className='font-medium'>
                                      {issue.title}
                                    </span>
                                    <Badge
                                      variant='outline'
                                      className={cn(
                                        'rounded-full',
                                        operationsSeverityClassName(
                                          issue.severity
                                        )
                                      )}
                                    >
                                      {operationsSeverityLabel(issue.severity)}
                                    </Badge>
                                    <Badge
                                      variant='outline'
                                      className='rounded-full'
                                    >
                                      {issueLifecycleLabel(
                                        issue.lifecycle_status
                                      )}
                                    </Badge>
                                  </div>
                                  <div className='text-xs leading-5 text-muted-foreground'>
                                    {issue.summary}
                                  </div>
                                </div>
                              </TableCell>
                              <TableCell className='align-top'>
                                <div className='space-y-1 text-sm'>
                                  <div>{issueScopeLabel(issue)}</div>
                                  <div className='text-xs text-muted-foreground'>
                                    {issue.library?.name || '未绑定媒体库'}
                                  </div>
                                </div>
                              </TableCell>
                              <TableCell className='align-top'>
                                <div className='space-y-1 text-sm'>
                                  <div>{issue.target_count} 个目标</div>
                                  <div className='text-xs text-muted-foreground'>
                                    {issue.occurrence_count} 条证据,{' '}
                                    {issue.impact.affected_files} 个文件
                                  </div>
                                </div>
                              </TableCell>
                              <TableCell className='align-top'>
                                <div className='max-w-[22rem] text-sm break-all whitespace-normal text-muted-foreground'>
                                  {sampleLabel}
                                </div>
                              </TableCell>
                              <TableCell className='align-top'>
                                <div className='flex flex-wrap gap-2'>
                                  {actions.slice(0, 3).map((action) => (
                                    <Badge
                                      key={action.action_key}
                                      variant='outline'
                                      className='rounded-full'
                                    >
                                      {action.label}
                                    </Badge>
                                  ))}
                                  {actions.length > 3 ? (
                                    <Badge
                                      variant='outline'
                                      className='rounded-full'
                                    >
                                      +{actions.length - 3}
                                    </Badge>
                                  ) : null}
                                </div>
                              </TableCell>
                              <TableCell className='align-top'>
                                <div className='flex flex-wrap gap-2'>
                                  <Button
                                    variant='outline'
                                    size='sm'
                                    onClick={(event) => {
                                      event.stopPropagation()
                                      setSelectedIssueId(issue.id)
                                    }}
                                  >
                                    查看
                                    <ChevronRightIcon className='size-4' />
                                  </Button>
                                  {ignoreAction ? (
                                    <Button
                                      variant='outline'
                                      size='sm'
                                      disabled={!isAdmin || ignoreRunning}
                                      onClick={(event) => {
                                        event.stopPropagation()
                                        resetActionDraft()
                                        setActionDialog({
                                          issue,
                                          action: ignoreAction,
                                        })
                                      }}
                                    >
                                      {ignoreRunning ? (
                                        <LoaderCircleIcon className='size-4 animate-spin' />
                                      ) : (
                                        <AlertTriangleIcon className='size-4' />
                                      )}
                                      忽略
                                    </Button>
                                  ) : null}
                                </div>
                              </TableCell>
                            </TableRow>
                          )
                        })
                      )}
                    </TableBody>
                  </Table>
                </div>
              </div>

              <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
                <div className='text-sm text-muted-foreground'>
                  第 {issuesQuery.data?.page ?? page} / {totalPages} 页，共{' '}
                  {total} 项
                </div>
                <div className='flex items-center gap-2'>
                  <Button
                    variant='outline'
                    size='sm'
                    disabled={(issuesQuery.data?.page ?? page) <= 1}
                    onClick={() =>
                      setPage((current) => Math.max(1, current - 1))
                    }
                  >
                    上一页
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    disabled={(issuesQuery.data?.page ?? page) >= totalPages}
                    onClick={() =>
                      setPage((current) => Math.min(totalPages, current + 1))
                    }
                  >
                    下一页
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <LegacyTaskFallback
              queryToken={queryToken}
              isAdmin={isAdmin}
              searchQuery={searchQuery}
              page={page}
              taskKind={legacyTaskKind}
              onTaskKindChange={setLegacyTaskKind}
              onMetadataReview={(task) => {
                setMetadataReviewIssue(null)
                setMetadataReviewTask(task)
              }}
            />
          )}
        </CardContent>
      </Card>

      <Sheet
        open={!!selectedIssueId && !usingLegacyFallback}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedIssueId(null)
          }
        }}
      >
        <SheetContent side='right' className='w-full sm:max-w-2xl'>
          <SheetHeader>
            <SheetTitle>{detailIssue?.title || '问题详情'}</SheetTitle>
            <SheetDescription>
              {detailIssue?.summary || '查看证据、样本和可执行动作。'}
            </SheetDescription>
          </SheetHeader>

          <ScrollArea className='min-h-0 flex-1 px-4'>
            <div className='space-y-6 pb-6'>
              {issueDetailQuery.isLoading ? (
                <div className='flex items-center gap-2 text-sm text-muted-foreground'>
                  <LoaderCircleIcon className='size-4 animate-spin' />
                  正在加载问题详情
                </div>
              ) : null}

              {detailIssue ? (
                <>
                  <div className='grid gap-3 sm:grid-cols-2'>
                    <DetailMetric
                      label='问题类型'
                      value={issueKindLabel(detailIssue.kind)}
                    />
                    <DetailMetric
                      label='当前状态'
                      value={issueLifecycleLabel(detailIssue.lifecycle_status)}
                    />
                    <DetailMetric
                      label='影响范围'
                      value={issueScopeLabel(detailIssue)}
                    />
                    <DetailMetric
                      label='影响规模'
                      value={`${detailIssue.target_count} 个目标 / ${detailIssue.occurrence_count} 条证据`}
                    />
                  </div>

                  <Card className='border-border/60 bg-background/70 shadow-none'>
                    <CardHeader className='pb-3'>
                      <CardTitle className='text-base'>可执行动作</CardTitle>
                      <CardDescription>
                        执行期间会锁定相同动作；成功后自动刷新详情和问题列表。
                      </CardDescription>
                    </CardHeader>
                    <CardContent className='space-y-3'>
                      {detailIssueActions.length ? (
                        detailIssueActions.map((action) => {
                          const running =
                            executeActionMutation.isPending &&
                            activeActionKey === action.action_key
                          return (
                            <div
                              key={action.action_key}
                              className='flex flex-col gap-3 rounded-lg border border-border/60 px-3 py-3'
                            >
                              <div className='flex items-start justify-between gap-3'>
                                <div className='space-y-1'>
                                  <div className='font-medium'>
                                    {action.label}
                                  </div>
                                  {action.description ? (
                                    <div className='text-sm text-muted-foreground'>
                                      {action.description}
                                    </div>
                                  ) : null}
                                  <div className='text-xs text-muted-foreground'>
                                    覆盖 {action.target_count} 个目标
                                  </div>
                                </div>
                                <Button
                                  size='sm'
                                  variant='outline'
                                  disabled={
                                    !isAdmin ||
                                    running ||
                                    (action.action_type === 'apply_candidate' &&
                                      !metadataTaskFromIssue)
                                  }
                                  onClick={() => {
                                    if (
                                      action.action_type === 'apply_candidate'
                                    ) {
                                      if (!metadataTaskFromIssue) return
                                      setMetadataReviewIssue(
                                        detailIssue ?? null
                                      )
                                      setMetadataReviewTask(
                                        metadataTaskFromIssue
                                      )
                                      return
                                    }
                                    handleIssueActionClick(
                                      detailIssue,
                                      action,
                                      setActionDialog,
                                      runIssueAction
                                    )
                                  }}
                                >
                                  {running ? (
                                    <LoaderCircleIcon className='size-4 animate-spin' />
                                  ) : (
                                    <CheckCircle2Icon className='size-4' />
                                  )}
                                  {action.action_type === 'apply_candidate'
                                    ? '查看候选'
                                    : '执行'}
                                </Button>
                              </div>
                            </div>
                          )
                        })
                      ) : (
                        <div className='text-sm text-muted-foreground'>
                          当前问题没有可执行动作。
                        </div>
                      )}
                    </CardContent>
                  </Card>

                  <Card className='border-border/60 bg-background/70 shadow-none'>
                    <CardHeader className='flex flex-row items-start justify-between gap-3 pb-3'>
                      <div className='space-y-1'>
                        <CardTitle className='text-base'>受影响目标</CardTitle>
                      </div>
                      {metadataTaskFromIssue ? (
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={!isAdmin}
                          onClick={() => {
                            setMetadataReviewIssue(detailIssue ?? null)
                            setMetadataReviewTask(metadataTaskFromIssue)
                          }}
                        >
                          治理元数据
                        </Button>
                      ) : null}
                    </CardHeader>
                    <CardContent className='space-y-2'>
                      {(detailIssue.targets ?? []).length ? (
                        detailIssue.targets?.map((target) => (
                          <div
                            key={`${target.target_type}:${target.target_key}`}
                            className='rounded-lg border border-border/60 px-3 py-2 text-sm'
                          >
                            <div className='font-medium'>
                              {target.label || target.target_key}
                            </div>
                            <div className='mt-1 text-muted-foreground'>
                              {target.description || target.target_type}
                            </div>
                          </div>
                        ))
                      ) : (
                        <div className='text-sm text-muted-foreground'>
                          暂无目标详情。
                        </div>
                      )}
                    </CardContent>
                  </Card>

                  <Card className='border-border/60 bg-background/70 shadow-none'>
                    <CardHeader className='pb-3'>
                      <CardTitle className='text-base'>证据与事件</CardTitle>
                    </CardHeader>
                    <CardContent className='space-y-3'>
                      {(detailIssue.occurrences ?? [])
                        .slice(0, 6)
                        .map((occurrence) => (
                          <div
                            key={`${occurrence.source_type}:${occurrence.source_key}`}
                            className='rounded-lg border border-border/60 px-3 py-2 text-sm'
                          >
                            <div className='font-medium'>
                              {occurrence.source_key}
                            </div>
                            <div className='mt-1 text-muted-foreground'>
                              {occurrence.message ||
                                occurrence.reason ||
                                occurrence.status ||
                                '未提供说明'}
                            </div>
                          </div>
                        ))}
                      {(detailEvents ?? []).length ? (
                        <>
                          <Separator />
                          {(detailEvents ?? []).slice(0, 8).map((event) => (
                            <div
                              key={`${event.id}:${event.created_at}`}
                              className='text-sm'
                            >
                              <div className='font-medium'>
                                {eventTypeLabel(event.event_type)}
                              </div>
                              <div className='text-muted-foreground'>
                                {event.message || event.status || '已记录事件'}
                              </div>
                            </div>
                          ))}
                        </>
                      ) : null}
                    </CardContent>
                  </Card>
                </>
              ) : null}
            </div>
          </ScrollArea>
        </SheetContent>
      </Sheet>

      <Dialog
        open={!!actionDialog}
        onOpenChange={(open) => {
          if (!open) {
            setActionDialog(null)
            resetActionDraft()
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {actionDialog?.action.label || '执行动作'}
            </DialogTitle>
            <DialogDescription>
              {actionDialog?.action.confirmation_message ||
                actionDialog?.action.description ||
                '请确认本次操作的附加信息。'}
            </DialogDescription>
          </DialogHeader>

          <FieldGroup>
            {actionDialog && requiresReason(actionDialog.action) ? (
              <Field>
                <FieldLabel htmlFor='issue-action-reason'>原因</FieldLabel>
                <Textarea
                  id='issue-action-reason'
                  value={actionReason}
                  onChange={(event) => setActionReason(event.target.value)}
                  placeholder='请输入忽略或排除原因'
                />
              </Field>
            ) : null}

            {actionDialog?.action.action_type === 'relink_resource' ? (
              <>
                <Field>
                  <FieldLabel htmlFor='issue-target-metadata'>
                    目标元数据 ID
                  </FieldLabel>
                  <Input
                    id='issue-target-metadata'
                    value={targetMetadataItemID}
                    onChange={(event) =>
                      setTargetMetadataItemID(event.target.value)
                    }
                    placeholder='例如 123'
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor='issue-source-metadata'>
                    来源元数据 ID
                  </FieldLabel>
                  <Input
                    id='issue-source-metadata'
                    value={sourceMetadataItemID}
                    onChange={(event) =>
                      setSourceMetadataItemID(event.target.value)
                    }
                    placeholder='可选'
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor='issue-resource-role'>
                    资源角色
                  </FieldLabel>
                  <Input
                    id='issue-resource-role'
                    value={resourceRole}
                    onChange={(event) => setResourceRole(event.target.value)}
                    placeholder='可选，例如 primary'
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor='issue-resource-mode'>模式</FieldLabel>
                  <Input
                    id='issue-resource-mode'
                    value={resourceMode}
                    onChange={(event) => setResourceMode(event.target.value)}
                    placeholder='move / add'
                  />
                </Field>
              </>
            ) : null}

            {actionDialog?.action.action_type === 'unlink_resource' ? (
              <>
                <Field>
                  <FieldLabel htmlFor='issue-target-metadata-unlink'>
                    元数据 ID
                  </FieldLabel>
                  <Input
                    id='issue-target-metadata-unlink'
                    value={targetMetadataItemID}
                    onChange={(event) =>
                      setTargetMetadataItemID(event.target.value)
                    }
                    placeholder='可选，默认使用当前目标'
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor='issue-resource-role-unlink'>
                    资源角色
                  </FieldLabel>
                  <Input
                    id='issue-resource-role-unlink'
                    value={resourceRole}
                    onChange={(event) => setResourceRole(event.target.value)}
                    placeholder='可选，例如 primary'
                  />
                </Field>
              </>
            ) : null}

            {actionDialog?.action.action_type === 'correct_classification' ? (
              <>
                <Field>
                  <FieldLabel htmlFor='classification-target-kind'>
                    目标类型
                  </FieldLabel>
                  <Input
                    id='classification-target-kind'
                    value={classificationTargetKind}
                    onChange={(event) =>
                      setClassificationTargetKind(event.target.value)
                    }
                    placeholder='例如 work'
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor='classification-target-key'>
                    目标键
                  </FieldLabel>
                  <Input
                    id='classification-target-key'
                    value={classificationTargetKey}
                    onChange={(event) =>
                      setClassificationTargetKey(event.target.value)
                    }
                    placeholder='例如 work:series:show'
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor='classification-role'>角色</FieldLabel>
                  <Input
                    id='classification-role'
                    value={classificationRole}
                    onChange={(event) =>
                      setClassificationRole(event.target.value)
                    }
                    placeholder='可选'
                  />
                </Field>
              </>
            ) : null}
          </FieldGroup>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                setActionDialog(null)
                resetActionDraft()
              }}
            >
              取消
            </Button>
            <Button
              disabled={executeActionMutation.isPending || !actionDialog}
              onClick={() => {
                if (!actionDialog) return
                void runIssueAction(actionDialog.issue, actionDialog.action)
              }}
            >
              {executeActionMutation.isPending ? (
                <LoaderCircleIcon className='size-4 animate-spin' />
              ) : (
                <CheckCircle2Icon className='size-4' />
              )}
              确认执行
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {metadataReviewTask ? (
        <Suspense fallback={null}>
          <MetadataReviewDialog
            key={metadataReviewTask.id}
            token={token}
            open
            task={metadataReviewTask}
            onOpenChange={(open) => {
              if (!open) {
                setMetadataReviewTask(null)
                setMetadataReviewIssue(null)
              }
            }}
            onResolveReview={
              metadataReviewIssue && metadataReviewIssueAction
                ? async () => {
                    await executeActionMutation.mutateAsync({
                      issueId: metadataReviewIssue.id,
                      action: metadataReviewIssueAction,
                      payload: {
                        action_key: metadataReviewIssueAction.action_key,
                        confirmation: true,
                      },
                    })
                  }
                : undefined
            }
            onResolved={async () => {
              setMetadataReviewTask(null)
              setMetadataReviewIssue(null)
              await invalidateOperationsQueries(queryClient, queryToken)
            }}
          />
        </Suspense>
      ) : null}
    </div>
  )

  function resetActionDraft() {
    setActionReason('')
    setTargetMetadataItemID('')
    setSourceMetadataItemID('')
    setResourceRole('')
    setResourceMode('move')
    setClassificationTargetKind('')
    setClassificationTargetKey('')
    setClassificationRole('')
  }

  async function runIssueAction(
    issue: OperationsIssue,
    action: OperationsIssueAction
  ) {
    const payload = {
      action_key: action.action_key,
      reason: actionReason.trim() || undefined,
      confirmation: requiresConfirmation(action),
      target_metadata_item_id: parseOptionalNumber(targetMetadataItemID),
      source_metadata_item_id: parseOptionalNumber(sourceMetadataItemID),
      metadata_item_id: parseOptionalNumber(targetMetadataItemID),
      role: resourceRole.trim() || undefined,
      mode: resourceMode.trim() || undefined,
      classification_target_kind: classificationTargetKind.trim() || undefined,
      classification_target_key: classificationTargetKey.trim() || undefined,
      classification_role: classificationRole.trim() || undefined,
    }
    await executeActionMutation.mutateAsync({
      issueId: issue.id,
      action,
      payload,
    })
    resetActionDraft()
  }
}

function LegacyTaskFallback({
  queryToken,
  isAdmin,
  searchQuery,
  page,
  taskKind,
  onTaskKindChange,
  onMetadataReview,
}: {
  queryToken: string
  isAdmin: boolean
  searchQuery: string
  page: number
  taskKind: 'all' | OperationsTaskKind
  onTaskKindChange: (value: 'all' | OperationsTaskKind) => void
  onMetadataReview: (task: OperationsTask | null) => void
}) {
  const token = useAuthStore((state) => state.auth.accessToken)
  const queryClient = useQueryClient()
  const tasksQuery = useQuery({
    ...operationsTaskListQueryOptions(queryToken, {
      page,
      page_size: PAGE_SIZE,
      lifecycle_status: 'active',
      kind: taskKind === 'all' ? undefined : taskKind,
      q: searchQuery || undefined,
    }),
    enabled: !!token,
  })

  const singleMutation = useMutation({
    mutationFn: async (actionId: string) => {
      if (!token) throw new Error('当前未登录，无法执行治理操作。')
      return createAuthedMiboApi(token).executeOperationsAction(actionId)
    },
    onSuccess: async () => {
      await invalidateOperationsQueries(queryClient, queryToken)
    },
  })

  const items = tasksQuery.data?.items ?? []

  return (
    <div className='flex min-h-0 flex-1 flex-col gap-4'>
      <div className='flex items-center gap-3'>
        <Field className='w-56'>
          <FieldLabel>Legacy 任务类型</FieldLabel>
          <Select
            value={taskKind}
            onValueChange={(value) =>
              onTaskKindChange(value as 'all' | OperationsTaskKind)
            }
          >
            <SelectTrigger className='w-full'>
              <SelectValue placeholder='全部任务类型' />
            </SelectTrigger>
            <SelectContent>
              {LEGACY_TASK_KIND_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
      </div>

      <div className='min-h-0 flex-1 overflow-hidden rounded-xl border border-border/60 bg-background/70'>
        <div className='h-full overflow-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>任务</TableHead>
                <TableHead>对象</TableHead>
                <TableHead>动作</TableHead>
                <TableHead className='w-32'>处理</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={4}
                    className='py-8 text-center text-sm text-muted-foreground'
                  >
                    当前筛选条件下没有 legacy 任务。
                  </TableCell>
                </TableRow>
              ) : (
                items.map((task) => {
                  const primaryAction = task.recommended_actions.find(
                    (action) =>
                      action.id && action.id !== 'issue_apply_candidate'
                  )
                  const firstAffectedItem = task.affected.items?.[0]
                  const objectLabel =
                    task.affected.files?.[0]?.storage_path ||
                    firstAffectedItem?.title ||
                    task.affected.libraries?.[0]?.name ||
                    '未记录'
                  return (
                    <TableRow key={task.id}>
                      <TableCell className='align-top'>
                        <div className='space-y-1'>
                          <div className='font-medium'>{task.title}</div>
                          <div className='text-xs text-muted-foreground'>
                            {task.summary}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className='align-top'>
                        <div className='max-w-[28rem] text-sm break-all whitespace-normal text-muted-foreground'>
                          {objectLabel}
                        </div>
                      </TableCell>
                      <TableCell className='align-top'>
                        <div className='flex flex-wrap gap-2'>
                          {task.recommended_actions
                            .filter((action) => action.id)
                            .slice(0, 3)
                            .map((action) => (
                              <Badge
                                key={action.id}
                                variant='outline'
                                className='rounded-full'
                              >
                                {action.label}
                              </Badge>
                            ))}
                        </div>
                      </TableCell>
                      <TableCell className='align-top'>
                        {task.kind === 'metadata_review_required' &&
                        firstAffectedItem?.id ? (
                          <Button
                            variant='outline'
                            size='sm'
                            className='mr-2'
                            disabled={!isAdmin}
                            onClick={() => onMetadataReview(task)}
                          >
                            治理元数据
                          </Button>
                        ) : null}
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={
                            !isAdmin ||
                            !primaryAction?.id ||
                            singleMutation.isPending
                          }
                          onClick={() =>
                            primaryAction?.id &&
                            singleMutation.mutate(primaryAction.id)
                          }
                        >
                          处理
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  )
}

function handleIssueActionClick(
  issue: OperationsIssue,
  action: OperationsIssueAction,
  setActionDialog: (value: ActionDialogState | null) => void,
  runIssueAction: (
    issue: OperationsIssue,
    action: OperationsIssueAction
  ) => Promise<void>
) {
  if (requiresDialog(action)) {
    setActionDialog({ issue, action })
    return
  }
  void runIssueAction(issue, action)
}

function findIssueIgnoreAction(actions: OperationsIssueAction[]) {
  return actions.find(
    (action) => action.action_type === 'ignore' && action.eligible
  )
}

function isDetailIssueAction(action: OperationsIssueAction) {
  return action.action_type !== 'ignore'
}

function requiresDialog(action: OperationsIssueAction) {
  return (
    requiresReason(action) ||
    action.action_type === 'relink_resource' ||
    action.action_type === 'unlink_resource' ||
    action.action_type === 'correct_classification'
  )
}

function requiresReason(action: OperationsIssueAction) {
  return action.action_type === 'exclude' || action.action_type === 'ignore'
}

function requiresConfirmation(action: OperationsIssueAction) {
  return !!action.confirmation_message || requiresReason(action)
}

function issueKindLabel(kind: OperationsIssueKind) {
  return (
    ISSUE_KIND_OPTIONS.find((option) => option.value === kind)?.label ?? kind
  )
}

function issueLifecycleLabel(status: OperationsIssueLifecycleStatus) {
  return (
    ISSUE_STATUS_OPTIONS.find((option) => option.value === status)?.label ??
    status
  )
}

function issueScopeLabel(issue: OperationsIssue) {
  return `${scopeKindLabel(issue.scope_kind)} · ${issue.scope_key}`
}

function scopeKindLabel(scopeKind: OperationsIssue['scope_kind']) {
  switch (scopeKind) {
    case 'library':
      return '媒体库'
    case 'media_source':
      return '媒体源'
    case 'folder':
      return '文件夹'
    case 'inventory_file':
      return '文件'
    case 'resource':
      return '资源'
    case 'metadata_item':
      return '条目'
    case 'series':
      return '剧集'
    case 'season':
      return '季度'
    case 'episode':
      return '单集'
    default:
      return scopeKind
  }
}

function eventTypeLabel(eventType: string) {
  switch (eventType) {
    case 'created':
      return '已创建'
    case 'observed':
      return '已观测'
    case 'updated':
      return '已更新'
    case 'action_requested':
      return '已请求动作'
    case 'action_succeeded':
      return '动作成功'
    case 'action_failed':
      return '动作失败'
    case 'resolved':
      return '已解决'
    case 'ignored':
      return '已忽略'
    case 'reopened':
      return '已重新打开'
    default:
      return eventType
  }
}

function showActionResultToast(result: OperationsActionResult) {
  const title = `最近一次操作: ${actionResultStatusLabel(result.status)}`
  const description = formatActionResultToastDescription(result)

  if (result.status === 'failed') {
    toast.error(title, { description })
    return
  }
  if (result.status === 'partial') {
    toast.warning(title, { description })
    return
  }
  toast.success(title, { description })
}

function formatActionResultToastDescription(result: OperationsActionResult) {
  const messages = (result.results ?? [])
    .filter((targetResult) => targetResult.status !== 'ok')
    .map(
      (targetResult) =>
        `${targetResult.target_key}: ${targetResult.message || actionResultStatusLabel(targetResult.status)}`
    )

  if (messages.length > 0) {
    return [result.message, ...messages].filter(Boolean).join('\n')
  }
  return result.message || '操作已完成。'
}

function actionResultStatusLabel(status: string) {
  switch (status) {
    case 'ok':
      return '成功'
    case 'partial':
      return '部分成功'
    case 'failed':
      return '失败'
    default:
      return status || '已完成'
  }
}

function parseOptionalNumber(value: string) {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

function mapLegacyTaskKind(
  value: 'all' | OperationsIssueKind
): 'all' | OperationsTaskKind {
  switch (value) {
    case 'metadata':
      return 'metadata_review_required'
    case 'classification':
      return 'classification_review_required'
    case 'workflow':
      return 'scan_blocked'
    case 'projection':
      return 'projection_stale'
    case 'storage':
      return 'storage_access_required'
    case 'probe':
      return 'maintenance_backlog'
    default:
      return 'all'
  }
}

function issueToMetadataReviewTask(
  issue: OperationsIssue
): OperationsTask | null {
  const metadataItems = collectMetadataIssueItems(issue)
  const affectedItem = metadataItems[0]
  if (!affectedItem) {
    return null
  }
  const fileTargets =
    issue.targets
      ?.filter((target) => target.inventory_file_id)
      .slice(0, 6)
      .map((target) => ({
        id: target.inventory_file_id!,
        library_id: target.library_id,
        storage_path: target.label || target.target_key,
        scan_state: 'review_required',
      })) ?? []

  return {
    id: `issue:${issue.id}`,
    kind: 'metadata_review_required',
    lifecycle_status: 'active',
    severity: issue.severity,
    title: issue.title,
    summary: issue.summary,
    impact: issue.impact,
    affected: {
      media_sources: [],
      libraries: issue.library ? [issue.library] : [],
      files: fileTargets,
      items: metadataItems,
    },
    recommended_actions:
      issue.actions
        ?.filter(
          (action) =>
            action.action_type === 'mark_governed' &&
            (action.action_key.startsWith('resolve_review_stage:') ||
              action.action_key === 'issue_mark_governed')
        )
        .map((action) => ({
          id: action.action_key,
          type:
            action.action_type === 'mark_governed'
              ? 'resolve_review_stage'
              : 'open_url',
          label: action.label,
          description: action.description,
        })) ?? [],
    evidence: [
      {
        kind: 'issue_scope',
        label: '问题范围',
        value: issue.scope_key,
      },
      {
        kind: 'issue_status',
        label: '当前状态',
        value: issue.lifecycle_status,
      },
    ],
    first_seen_at: issue.first_seen_at,
    last_seen_at: issue.last_seen_at,
  }
}

function collectMetadataIssueItems(issue: OperationsIssue) {
  const items = new Map<number, { id: number; title: string; type: string }>()
  for (const target of issue.targets ?? []) {
    if (!target.metadata_item_id) continue
    items.set(target.metadata_item_id, {
      id: target.metadata_item_id,
      title: target.label || issue.title,
      type: 'item',
    })
  }
  for (const occurrence of issue.occurrences ?? []) {
    if (
      !occurrence.metadata_item_id ||
      items.has(occurrence.metadata_item_id)
    ) {
      continue
    }
    items.set(occurrence.metadata_item_id, {
      id: occurrence.metadata_item_id,
      title: issue.title,
      type: 'item',
    })
  }
  return Array.from(items.values()).slice(0, 6)
}

function DetailMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className='rounded-lg border border-border/60 bg-background/70 px-3 py-3'>
      <div className='text-xs text-muted-foreground'>{label}</div>
      <div className='mt-1 font-medium'>{value}</div>
    </div>
  )
}

async function invalidateOperationsQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  token: string
) {
  await queryClient.invalidateQueries({
    queryKey: miboQueryKeys.operationsOverview(token),
  })
  await queryClient.invalidateQueries({
    queryKey: miboQueryKeys.operationsTasks(token),
  })
  await queryClient.invalidateQueries({
    queryKey: miboQueryKeys.operationsIssues(token),
  })
  await queryClient.invalidateQueries({
    queryKey: ['operations', 'issues', token],
  })
  await queryClient.invalidateQueries({
    queryKey: ['operations', 'tasks', token],
  })
  await queryClient.invalidateQueries({
    queryKey: miboQueryKeys.operationsPipeline(token),
  })
  await queryClient.invalidateQueries({ queryKey: ['home'] })
  await queryClient.invalidateQueries({
    queryKey: miboQueryKeys.consoleSummary(token),
  })
}
