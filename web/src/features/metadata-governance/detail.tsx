import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  CheckCircle2Icon,
  LoaderCircleIcon,
  WandSparklesIcon,
} from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '#/components/ui/dialog'
import type { MediaItemDetail, MetadataSearchCandidate } from '#/lib/mibo-api'
import { createAuthedMiboApi, miboQueryKeys } from '#/lib/mibo-query'

import { CandidatePreviewCard } from './detail-sections'
import {
  ArtworkCard,
  AsyncActionsCard,
  CandidateSearchCard,
  DraftEditorCard,
  MetadataSummaryCard,
} from './detail-panels'
import { formatMatchStatus, formatMediaType } from './formatters'

type MetadataDraft = {
  title: string
  originalTitle: string
  year: string
  overview: string
  posterUrl: string
  backdropUrl: string
}

type AsyncActionState = {
  type: 'rematch' | 'refetch'
  jobId: number
  kind: string
  status: 'queued' | 'running' | 'completed' | 'failed'
  message: string
}

const EMPTY_DRAFT: MetadataDraft = {
  title: '',
  originalTitle: '',
  year: '',
  overview: '',
  posterUrl: '',
  backdropUrl: '',
}

export function MetadataGovernanceDetail({
  token,
  mediaItemId,
}: {
  token: string
  mediaItemId: number
}) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const workspaceQueryKey = miboQueryKeys.metadataWorkspace(token)
  const itemQuery = useQuery({
    queryKey: miboQueryKeys.mediaItemDetail(token, mediaItemId),
    queryFn: () => createAuthedMiboApi(token).getMediaItem(mediaItemId),
  })

  const [draft, setDraft] = useState<MetadataDraft>(EMPTY_DRAFT)
  const [baselineDraft, setBaselineDraft] = useState<MetadataDraft>(EMPTY_DRAFT)
  const [searchTitle, setSearchTitle] = useState('')
  const [searchYear, setSearchYear] = useState('')
  const [candidatePreview, setCandidatePreview] =
    useState<MetadataSearchCandidate | null>(null)
  const [asyncActionState, setAsyncActionState] =
    useState<AsyncActionState | null>(null)
  const [saveSuccessMessage, setSaveSuccessMessage] = useState('')
  const pollTimerRef = useRef<number | null>(null)

  useEffect(() => {
    if (!itemQuery.data) return

    const nextDraft = buildDraftFromItem(itemQuery.data)
    setDraft(nextDraft)
    setBaselineDraft(nextDraft)
    setSearchTitle(itemQuery.data.title)
    setSearchYear(itemQuery.data.year ? String(itemQuery.data.year) : '')
  }, [itemQuery.data?.id])

  const isDirty = JSON.stringify(draft) !== JSON.stringify(baselineDraft)

  useEffect(() => {
    if (saveSuccessMessage && isDirty) {
      setSaveSuccessMessage('')
    }
  }, [isDirty, saveSuccessMessage])

  useEffect(() => {
    if (!isDirty) return

    function handleBeforeUnload(event: BeforeUnloadEvent) {
      event.preventDefault()
      event.returnValue = ''
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [isDirty])

  useEffect(() => {
    if (!isDirty) return

    function handleDocumentClick(event: MouseEvent) {
      const target = event.target
      if (!(target instanceof Element)) return

      const anchor = target.closest('a[href]')
      if (!(anchor instanceof HTMLAnchorElement)) return
      if (
        anchor.target === '_blank' ||
        anchor.hasAttribute('download') ||
        event.metaKey ||
        event.ctrlKey ||
        event.shiftKey ||
        event.altKey
      ) {
        return
      }

      const destination = new URL(anchor.href, window.location.href)
      const current = new URL(window.location.href)
      const isSameDocumentNavigation =
        destination.origin === current.origin &&
        destination.pathname === current.pathname &&
        destination.search === current.search &&
        destination.hash === current.hash
      if (isSameDocumentNavigation) return

      if (!window.confirm('当前有未保存修改，确认离开治理页吗？')) {
        event.preventDefault()
        event.stopPropagation()
      }
    }

    document.addEventListener('click', handleDocumentClick, true)
    return () =>
      document.removeEventListener('click', handleDocumentClick, true)
  }, [isDirty])

  const searchMutation = useMutation({
    mutationFn: () =>
      createAuthedMiboApi(token).searchMediaItemMetadata(mediaItemId, {
        title: searchTitle.trim() || undefined,
        year: parseOptionalNumber(searchYear),
      }),
  })

  const saveDraftMutation = useMutation({
    mutationFn: () =>
      createAuthedMiboApi(token).updateMediaItemMetadata(mediaItemId, {
        title: draft.title.trim(),
        original_title: draft.originalTitle.trim() || undefined,
        year: parseOptionalNumber(draft.year),
        overview: draft.overview.trim() || undefined,
        poster_url: draft.posterUrl.trim() || undefined,
        backdrop_url: draft.backdropUrl.trim() || undefined,
      }),
    onSuccess: async (item) => {
      const nextDraft = buildDraftFromItem(item)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setSaveSuccessMessage('草稿已保存，治理页和媒体详情将使用最新元数据。')
      queryClient.setQueryData(
        miboQueryKeys.mediaItemDetail(token, mediaItemId),
        item,
      )
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.homeData(token),
        }),
      ])
    },
  })

  const applyCandidateMutation = useMutation({
    mutationFn: (externalId: string) =>
      createAuthedMiboApi(token).applyMediaItemMetadataCandidate(mediaItemId, {
        external_id: externalId,
      }),
    onSuccess: async (item) => {
      const nextDraft = buildDraftFromItem(item)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setCandidatePreview(null)
      setSaveSuccessMessage('候选结果已应用，当前治理草稿已同步为最新元数据。')
      queryClient.setQueryData(
        miboQueryKeys.mediaItemDetail(token, mediaItemId),
        item,
      )
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.mediaItemDetail(token, mediaItemId),
        }),
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.homeData(token),
        }),
      ])
    },
  })

  const rematchMutation = useMutation({
    mutationFn: () => createAuthedMiboApi(token).rematchMediaItem(mediaItemId),
    onSuccess: async (job) => {
      setAsyncActionState({
        type: 'rematch',
        jobId: job.id,
        kind: job.kind,
        status: 'queued',
        message: `重新匹配任务已排队，任务 ID #${job.id}。`,
      })
    },
  })

  const refetchMutation = useMutation({
    mutationFn: () =>
      createAuthedMiboApi(token).refetchMediaItemMetadata(mediaItemId),
    onSuccess: (job) => {
      setAsyncActionState({
        type: 'refetch',
        jobId: job.id,
        kind: job.kind,
        status: 'queued',
        message: `元数据重抓任务已排队，任务 ID #${job.id}。`,
      })
    },
  })

  useEffect(() => {
    if (
      !asyncActionState ||
      asyncActionState.status === 'completed' ||
      asyncActionState.status === 'failed'
    ) {
      if (pollTimerRef.current !== null) {
        window.clearTimeout(pollTimerRef.current)
        pollTimerRef.current = null
      }
      return
    }

    const activeAction = asyncActionState
    let cancelled = false

    async function pollJob() {
      try {
        const jobs = await createAuthedMiboApi(token).listJobs({
          limit: 20,
          kind: activeAction.kind,
        })
        if (cancelled) return

        const job = jobs.find(
          (candidate) => candidate.id === activeAction.jobId,
        )
        if (!job) {
          pollTimerRef.current = window.setTimeout(pollJob, 1500)
          return
        }

        if (job.status === 'queued') {
          setAsyncActionState((current) =>
            current && current.jobId === job.id
              ? {
                  ...current,
                  status: 'queued',
                  message: `${describeAsyncAction(current.type)}任务已排队，等待后台处理。任务 ID #${job.id}。`,
                }
              : current,
          )
          pollTimerRef.current = window.setTimeout(pollJob, 1500)
          return
        }

        if (job.status === 'running') {
          setAsyncActionState((current) =>
            current && current.jobId === job.id
              ? {
                  ...current,
                  status: 'running',
                  message: `${describeAsyncAction(current.type)}正在后台处理中，完成后会自动刷新页面数据。`,
                }
              : current,
          )
          pollTimerRef.current = window.setTimeout(pollJob, 1500)
          return
        }

        if (job.status === 'completed') {
          await Promise.all([
            itemQuery.refetch(),
            queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
            queryClient.invalidateQueries({
              queryKey: miboQueryKeys.homeData(token),
            }),
          ])
          if (cancelled) return

          setAsyncActionState((current) =>
            current && current.jobId === job.id
              ? {
                  ...current,
                  status: 'completed',
                  message: `${describeAsyncAction(current.type)}已完成，当前条目的治理结果已刷新。`,
                }
              : current,
          )
          return
        }

        setAsyncActionState((current) =>
          current && current.jobId === job.id
            ? {
                ...current,
                status: 'failed',
                message:
                  job.error_message ||
                  `${describeAsyncAction(current.type)}失败，请稍后重试。`,
              }
            : current,
        )
      } catch (error) {
        if (cancelled) return

        setAsyncActionState((current) =>
          current
            ? {
                ...current,
                status: 'failed',
                message:
                  error instanceof Error
                    ? error.message
                    : '无法获取后台任务状态。',
              }
            : current,
        )
      }
    }

    void pollJob()

    return () => {
      cancelled = true
      if (pollTimerRef.current !== null) {
        window.clearTimeout(pollTimerRef.current)
        pollTimerRef.current = null
      }
    }
  }, [asyncActionState, itemQuery, queryClient, token, workspaceQueryKey])

  if (itemQuery.isLoading) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-full border border-border/50 bg-card/85 px-5 py-3">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-muted-foreground">正在加载治理页</span>
        </div>
      </div>
    )
  }

  if (itemQuery.error || !itemQuery.data) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background px-6 text-foreground">
        <div className="max-w-xl space-y-4 rounded-[1.75rem] border border-border/60 bg-card/80 p-6 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">
            治理页暂时不可用
          </h1>
          <p className="text-sm text-muted-foreground">
            {itemQuery.error?.message ?? '未找到对应媒体条目。'}
          </p>
          <Button asChild variant="outline">
            <Link to="/metadata">返回治理工作台</Link>
          </Button>
        </div>
      </div>
    )
  }

  const item = itemQuery.data
  const activeCandidates = searchMutation.data ?? []

  async function handleNavigateAway(to: '/' | '/metadata' | '/media/$id') {
    if (isDirty && !window.confirm('当前有未保存修改，确认离开治理页吗？')) {
      return
    }

    if (to === '/media/$id') {
      await navigate({ to, params: { id: String(mediaItemId) } })
      return
    }

    await navigate({ to })
  }

  return (
    <>
      <div className="min-h-svh bg-background px-4 py-6 text-foreground sm:px-6 lg:px-8 xl:px-10">
        <div className="mx-auto max-w-7xl space-y-4">
          <div className="flex flex-col gap-4 rounded-[1.75rem] border border-border/60 bg-card/80 p-5 shadow-sm backdrop-blur-sm lg:flex-row lg:items-start lg:justify-between">
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <Badge
                  variant="outline"
                  className="border-border/60 bg-background/70"
                >
                  单条目治理
                </Badge>
                <Badge variant="secondary">{formatMediaType(item.type)}</Badge>
                <Badge
                  variant="outline"
                  className="border-border/60 bg-background/70"
                >
                  {formatMatchStatus(item.match_status)}
                </Badge>
              </div>
              <div>
                <h1 className="text-3xl font-semibold tracking-tight">
                  {item.title}
                </h1>
                <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
                  通过统一草稿会话处理基础字段校正、候选比对和异步匹配动作。当前页面已接入手工保存、候选应用、重新匹配和元数据重抓。
                </p>
              </div>
            </div>

            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                className="border-border/60 bg-background/70"
                onClick={() => void handleNavigateAway('/metadata')}
              >
                返回工作台
              </Button>
              <Button
                variant="outline"
                className="border-border/60 bg-background/70"
                onClick={() => void handleNavigateAway('/media/$id')}
              >
                查看详情页
              </Button>
            </div>
          </div>

          {isDirty ? (
            <Alert>
              <WandSparklesIcon className="size-4" />
              <AlertTitle>存在未保存草稿</AlertTitle>
              <AlertDescription>
                离开当前页面前会要求确认。保存后会同步刷新治理页、媒体详情和工作台摘要。
              </AlertDescription>
            </Alert>
          ) : null}

          {saveSuccessMessage ? (
            <Alert>
              <CheckCircle2Icon className="size-4" />
              <AlertTitle>保存成功</AlertTitle>
              <AlertDescription>{saveSuccessMessage}</AlertDescription>
            </Alert>
          ) : null}

          {asyncActionState ? (
            <Alert>
              {asyncActionState.status === 'failed' ? (
                <WandSparklesIcon className="size-4" />
              ) : asyncActionState.status === 'completed' ? (
                <CheckCircle2Icon className="size-4" />
              ) : (
                <LoaderCircleIcon className="size-4 animate-spin" />
              )}
              <AlertTitle>
                {formatAsyncActionTitle(asyncActionState)}
              </AlertTitle>
              <AlertDescription>{asyncActionState.message}</AlertDescription>
            </Alert>
          ) : null}

          {searchMutation.error ||
          saveDraftMutation.error ||
          applyCandidateMutation.error ||
          rematchMutation.error ||
          refetchMutation.error ? (
            <Alert>
              <AlertTitle>操作失败</AlertTitle>
              <AlertDescription>
                {searchMutation.error?.message ||
                  saveDraftMutation.error?.message ||
                  applyCandidateMutation.error?.message ||
                  rematchMutation.error?.message ||
                  refetchMutation.error?.message}
              </AlertDescription>
            </Alert>
          ) : null}

          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
            <div className="space-y-4">
              <DraftEditorCard
                draft={draft}
                baselineDraft={baselineDraft}
                isDirty={isDirty}
                isPending={saveDraftMutation.isPending}
                onDraftChange={(updater) => setDraft(updater)}
                onReset={setDraft}
                onSave={() => void saveDraftMutation.mutateAsync()}
              />

              <CandidateSearchCard
                searchTitle={searchTitle}
                searchYear={searchYear}
                isPending={searchMutation.isPending}
                isSuccess={searchMutation.isSuccess}
                activeCandidates={activeCandidates}
                onSearchTitleChange={setSearchTitle}
                onSearchYearChange={setSearchYear}
                onSearch={() => void searchMutation.mutateAsync()}
                onPreview={setCandidatePreview}
              />
            </div>

            <div className="space-y-4">
              <MetadataSummaryCard item={item} />
              <AsyncActionsCard
                rematchPending={rematchMutation.isPending}
                refetchPending={refetchMutation.isPending}
                onRematch={() => void rematchMutation.mutateAsync()}
                onRefetch={() => void refetchMutation.mutateAsync()}
              />
              <ArtworkCard
                posterUrl={draft.posterUrl}
                backdropUrl={draft.backdropUrl}
              />
            </div>
          </div>
        </div>
      </div>

      <Dialog
        open={candidatePreview !== null}
        onOpenChange={(open) => !open && setCandidatePreview(null)}
      >
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>候选差异预览</DialogTitle>
            <DialogDescription>
              预览当前条目与候选元数据的关键差异后，再确认应用。
            </DialogDescription>
          </DialogHeader>

          {candidatePreview ? (
            <div className="grid gap-4 lg:grid-cols-2">
              <CandidatePreviewCard title="当前条目" item={item} />
              <CandidatePreviewCard
                title="候选结果"
                candidate={candidatePreview}
              />
            </div>
          ) : null}

          <DialogFooter>
            <Button variant="outline" onClick={() => setCandidatePreview(null)}>
              取消
            </Button>
            <Button
              onClick={() => {
                if (!candidatePreview) return
                void applyCandidateMutation.mutateAsync(
                  candidatePreview.external_id,
                )
              }}
              disabled={!candidatePreview || applyCandidateMutation.isPending}
            >
              {applyCandidateMutation.isPending ? (
                <LoaderCircleIcon className="size-4 animate-spin" />
              ) : null}
              确认应用候选
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

function buildDraftFromItem(item: MediaItemDetail): MetadataDraft {
  return {
    title: item.title || '',
    originalTitle: item.original_title || '',
    year: item.year ? String(item.year) : '',
    overview: item.overview || '',
    posterUrl: item.poster_url || '',
    backdropUrl: item.backdrop_url || '',
  }
}

function parseOptionalNumber(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return undefined

  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? parsed : undefined
}

function describeAsyncAction(type: AsyncActionState['type']) {
  return type === 'rematch' ? '重新匹配' : '元数据重抓'
}

function formatAsyncActionTitle(state: AsyncActionState) {
  const action = describeAsyncAction(state.type)

  switch (state.status) {
    case 'queued':
      return `${action}已排队`
    case 'running':
      return `${action}处理中`
    case 'completed':
      return `${action}已完成`
    case 'failed':
      return `${action}失败`
    default:
      return action
  }
}
