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
import type {
  CatalogGovernanceWorkspace,
  MetadataSearchCandidate,
} from '#/lib/mibo-api'
import {
  catalogGovernanceWorkspaceQueryOptions,
  createAuthedMiboApi,
  miboQueryKeys,
} from '#/lib/mibo-query'

import { CandidatePreviewCard } from './detail-sections'
import {
  ArtworkCard,
  AssetLinksCard,
  AsyncActionsCard,
  CandidateSearchCard,
  ClassificationReviewCard,
  DraftEditorCard,
  FieldLocksCard,
  ImageCandidatesCard,
  MetadataSummaryCard,
  RelatedChildrenCard,
  SourceEvidenceCard,
} from './detail-panels'
import { formatMatchStatus, formatMediaType } from './formatters'

type MetadataDraft = {
  title: string
  originalTitle: string
  year: string
  overview: string
}

type AsyncActionState = {
  type: 'rematch' | 'refetch' | 'reprobe'
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
}

export function MetadataGovernanceDetail({
  token,
  itemId,
}: {
  token: string
  itemId: number
}) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const workspaceQueryKey = miboQueryKeys.catalogGovernanceWorkspace(
    token,
    itemId,
  )
  const listWorkspaceQueryKey = miboQueryKeys.metadataWorkspace(token)
  const workspaceQuery = useQuery({
    ...catalogGovernanceWorkspaceQueryOptions(token, itemId),
  })

  const [draft, setDraft] = useState<MetadataDraft>(EMPTY_DRAFT)
  const [baselineDraft, setBaselineDraft] = useState<MetadataDraft>(EMPTY_DRAFT)
  const [searchTitle, setSearchTitle] = useState('')
  const [searchYear, setSearchYear] = useState('')
  const [searchIMDbId, setSearchIMDbId] = useState('')
  const [searchTMDBId, setSearchTMDBId] = useState('')
  const [searchTVDBId, setSearchTVDBId] = useState('')
  const [candidatePreview, setCandidatePreview] =
    useState<MetadataSearchCandidate | null>(null)
  const [asyncActionState, setAsyncActionState] =
    useState<AsyncActionState | null>(null)
  const [saveSuccessMessage, setSaveSuccessMessage] = useState('')
  const pollTimerRef = useRef<number | null>(null)

  useEffect(() => {
    if (!workspaceQuery.data) return

    const nextDraft = buildDraftFromWorkspace(workspaceQuery.data)
    setDraft(nextDraft)
    setBaselineDraft(nextDraft)
    setSearchTitle(workspaceQuery.data.title)
    setSearchYear(
      fieldStateNumber(workspaceQuery.data, 'year')
        ? String(fieldStateNumber(workspaceQuery.data, 'year'))
        : '',
    )
    setSearchIMDbId('')
    setSearchTMDBId('')
    setSearchTVDBId('')
  }, [workspaceQuery.data?.item_id])

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
      createAuthedMiboApi(token).searchCatalogItemMetadata(itemId, {
        title: searchTitle.trim() || undefined,
        year: parseOptionalNumber(searchYear),
        imdb_id: searchIMDbId.trim() || undefined,
        tmdb_id: searchTMDBId.trim() || undefined,
        tvdb_id: searchTVDBId.trim() || undefined,
      }),
  })

  const saveDraftMutation = useMutation({
    mutationFn: async () => {
      const api = createAuthedMiboApi(token)
      const updates = [
        { field_key: 'title', value: draft.title.trim() },
        {
          field_key: 'original_title',
          value: draft.originalTitle.trim(),
        },
        {
          field_key: 'year',
          value: parseOptionalNumber(draft.year),
        },
        {
          field_key: 'overview',
          value: draft.overview.trim(),
        },
      ]

      for (const update of updates) {
        if (
          update.field_key !== 'title' &&
          (update.value === undefined || update.value === '')
        ) {
          continue
        }
        await api.updateCatalogGovernanceField(itemId, {
          field_key: update.field_key,
          value: update.value ?? '',
        })
      }

      return api.getCatalogGovernanceWorkspace(itemId)
    },
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setSaveSuccessMessage('草稿已保存，治理页和媒体详情将使用最新元数据。')
      queryClient.setQueryData(workspaceQueryKey, workspace)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.homeData(token),
        }),
      ])
    },
  })

  const applyCandidateMutation = useMutation({
    mutationFn: (externalId: string) =>
      createAuthedMiboApi(token).applyCatalogItemMetadataCandidate(itemId, {
        external_id: externalId,
      }),
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setCandidatePreview(null)
      setSaveSuccessMessage('候选结果已应用，当前治理草稿已同步为最新元数据。')
      queryClient.setQueryData(workspaceQueryKey, workspace)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.homeData(token),
        }),
      ])
    },
  })

  const rematchMutation = useMutation({
    mutationFn: () => createAuthedMiboApi(token).matchCatalogItem(itemId),
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setAsyncActionState({
        type: 'rematch',
        jobId: 0,
        kind: 'catalog_match',
        status: 'completed',
        message: '重新匹配已完成，治理结果已刷新。',
      })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
      ])
    },
  })

  const refetchMutation = useMutation({
    mutationFn: () =>
      createAuthedMiboApi(token).refetchCatalogItemMetadata(itemId),
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setAsyncActionState({
        type: 'refetch',
        jobId: 0,
        kind: 'catalog_refetch',
        status: 'completed',
        message: '元数据重抓已完成，来源证据和字段值已刷新。',
      })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
      ])
    },
  })

  const reprobeMutation = useMutation({
    mutationFn: (inventoryFileId: number) => {
      if (!inventoryFileId) {
        throw new Error('当前条目没有可重新探测的库存文件。')
      }
      return createAuthedMiboApi(token).reprobeInventoryFile(inventoryFileId)
    },
    onSuccess: (job) => {
      setAsyncActionState({
        type: 'reprobe',
        jobId: job.id,
        kind: job.kind,
        status: 'queued',
        message: `重新探测任务已排队，任务 ID #${job.id}。`,
      })
    },
  })

  const lockMutation = useMutation({
    mutationFn: ({
      fieldKey,
      nextLocked,
    }: {
      fieldKey: string
      nextLocked: boolean
    }) =>
      createAuthedMiboApi(token).updateCatalogGovernanceField(itemId, {
        field_key: fieldKey,
        value: fieldStateValue(workspaceQuery.data, fieldKey) ?? '',
        lock: nextLocked,
        lock_reason: nextLocked ? 'governance ui' : '',
        force: true,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: workspaceQueryKey })
    },
  })

  const imageMutation = useMutation({
    mutationFn: ({ imageType, url }: { imageType: string; url: string }) =>
      createAuthedMiboApi(token).selectCatalogGovernanceImage(itemId, {
        image_type: imageType,
        url,
      }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
      ])
    },
  })

  const assetLinkMutation = useMutation({
    mutationFn: async ({
      assetId,
      targetItemId,
      mode,
    }: {
      assetId: number
      targetItemId: number
      mode: 'link' | 'unlink'
    }) => {
      const api = createAuthedMiboApi(token)
      return mode === 'link'
        ? api.linkCatalogGovernanceAsset(itemId, assetId, {
            target_item_id: targetItemId,
          })
        : api.unlinkCatalogGovernanceAsset(itemId, assetId, targetItemId)
    },
    onSuccess: async (workspace, variables) => {
      setSaveSuccessMessage(
        variables.mode === 'link'
          ? '资产链接已更新，治理工作区已刷新。'
          : '资产链接已解除，治理工作区已刷新。',
      )
      queryClient.setQueryData(workspaceQueryKey, workspace)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
        queryClient.invalidateQueries({ queryKey: listWorkspaceQueryKey }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.catalogItemDetail(token, itemId),
        }),
      ])
    },
  })

  useEffect(() => {
    if (
      !asyncActionState ||
      asyncActionState.type !== 'reprobe' ||
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

        if (job.status === 'queued' || job.status === 'running') {
          setAsyncActionState((current) =>
            current && current.jobId === job.id
              ? {
                  ...current,
                  status: job.status as 'queued' | 'running',
                  message:
                    job.status === 'queued'
                      ? `重新探测任务已排队，等待后台处理。任务 ID #${job.id}。`
                      : '重新探测正在后台处理中，完成后会自动刷新页面数据。',
                }
              : current,
          )
          pollTimerRef.current = window.setTimeout(pollJob, 1500)
          return
        }

        if (job.status === 'completed') {
          await queryClient.invalidateQueries({ queryKey: workspaceQueryKey })
          if (cancelled) return

          setAsyncActionState((current) =>
            current && current.jobId === job.id
              ? {
                  ...current,
                  status: 'completed',
                  message: '重新探测已完成，资产状态已刷新。',
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
                message: job.error_message || '重新探测失败，请稍后重试。',
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
  }, [asyncActionState, queryClient, token, workspaceQueryKey])

  if (workspaceQuery.isLoading) {
    return (
      <div className="flex items-center gap-3 rounded-[1.5rem] border border-border/60 bg-card/80 px-5 py-4 text-foreground shadow-sm">
        <LoaderCircleIcon className="size-4 animate-spin" />
        <span className="text-sm text-muted-foreground">正在加载治理页</span>
      </div>
    )
  }

  if (workspaceQuery.error || !workspaceQuery.data) {
    return (
      <div className="rounded-[1.75rem] border border-border/60 bg-card/80 px-6 py-8 text-foreground shadow-sm">
        <div className="max-w-xl space-y-4">
          <h1 className="text-2xl font-semibold tracking-tight">
            治理页暂时不可用
          </h1>
          <p className="text-sm text-muted-foreground">
            {workspaceQuery.error?.message ?? '未找到对应治理工作区。'}
          </p>
          <Button asChild variant="outline">
            <Link to="/settings/metadata">返回治理工作台</Link>
          </Button>
        </div>
      </div>
    )
  }

  const workspace = workspaceQuery.data
  const workspaceAssets = workspace.assets ?? []
  const workspaceFieldStates = workspace.field_states ?? []
  const workspaceSourceEvidence = workspace.source_evidence ?? []
  const workspaceClassification = workspace.classification_decisions ?? []
  const workspaceSelectedImages = workspace.selected_images ?? []
  const workspaceImageCandidates = workspace.image_candidates ?? []
  const workspaceRecommendedChildren = workspace.recommended_children ?? []
  const item = buildPreviewItem(workspace)
  const activeCandidates = uniqueMetadataCandidates(searchMutation.data ?? [])
  const firstInventoryFileId = workspaceAssets.find(
    (asset) => (asset.file_ids ?? []).length > 0,
  )?.file_ids[0]

  async function handleNavigateAway(
    to: '/' | '/settings/metadata' | '/media/$id',
  ) {
    if (isDirty && !window.confirm('当前有未保存修改，确认离开治理页吗？')) {
      return
    }

    if (to === '/media/$id') {
      await navigate({
        to,
        params: { id: String(itemId) },
        search: { view: undefined },
      })
      return
    }

    await navigate({ to })
  }

  return (
    <>
      <div className="space-y-4 text-foreground">
        <div className="flex flex-col gap-4 rounded-[1.75rem] border border-border/60 bg-card/80 p-5 shadow-sm backdrop-blur-sm lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge
                variant="outline"
                className="border-border/60 bg-background/70"
              >
                单条目治理
              </Badge>
              <Badge variant="secondary">
                {formatMediaType(workspace.type)}
              </Badge>
              <Badge
                variant="outline"
                className="border-border/60 bg-background/70"
              >
                {formatMatchStatus(workspace.governance_status)}
              </Badge>
            </div>
            <div>
              <h1 className="text-3xl font-semibold tracking-tight">
                {workspace.title}
              </h1>
              <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
                当前页面已切到 catalog governance
                workspace，支持字段锁、来源证据、图片选择、资产链接与候选应用。
              </p>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => void handleNavigateAway('/settings/metadata')}
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
            <AlertTitle>{formatAsyncActionTitle(asyncActionState)}</AlertTitle>
            <AlertDescription>{asyncActionState.message}</AlertDescription>
          </Alert>
        ) : null}

        {searchMutation.error ||
        saveDraftMutation.error ||
        applyCandidateMutation.error ||
        rematchMutation.error ||
        refetchMutation.error ||
        reprobeMutation.error ||
        lockMutation.error ||
        imageMutation.error ||
        assetLinkMutation.error ? (
          <Alert>
            <AlertTitle>操作失败</AlertTitle>
            <AlertDescription>
              {searchMutation.error?.message ||
                saveDraftMutation.error?.message ||
                applyCandidateMutation.error?.message ||
                rematchMutation.error?.message ||
                refetchMutation.error?.message ||
                reprobeMutation.error?.message ||
                lockMutation.error?.message ||
                imageMutation.error?.message ||
                assetLinkMutation.error?.message}
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
              searchIMDbId={searchIMDbId}
              searchTMDBId={searchTMDBId}
              searchTVDBId={searchTVDBId}
              isPending={searchMutation.isPending}
              isSuccess={searchMutation.isSuccess}
              activeCandidates={activeCandidates}
              onSearchTitleChange={setSearchTitle}
              onSearchYearChange={setSearchYear}
              onSearchIMDbIdChange={setSearchIMDbId}
              onSearchTMDBIdChange={setSearchTMDBId}
              onSearchTVDBIdChange={setSearchTVDBId}
              onSearch={() => void searchMutation.mutateAsync()}
              onPreview={setCandidatePreview}
            />

            <FieldLocksCard
              fieldStates={workspaceFieldStates}
              isPending={lockMutation.isPending}
              onToggleLock={(fieldKey, nextLocked) => {
                void lockMutation.mutateAsync({ fieldKey, nextLocked })
              }}
            />

            <SourceEvidenceCard sourceEvidence={workspaceSourceEvidence} />

            <ClassificationReviewCard decisions={workspaceClassification} />

            <ImageCandidatesCard
              selectedImages={workspaceSelectedImages}
              imageCandidates={workspaceImageCandidates}
              isPending={imageMutation.isPending}
              onSelect={(imageType, url) => {
                void imageMutation.mutateAsync({ imageType, url })
              }}
            />
          </div>

          <div className="space-y-4">
            <MetadataSummaryCard item={item} />
            <AsyncActionsCard
              rematchPending={rematchMutation.isPending}
              refetchPending={refetchMutation.isPending}
              reprobePending={reprobeMutation.isPending}
              reprobeDisabled={!firstInventoryFileId}
              onRematch={() => void rematchMutation.mutateAsync()}
              onRefetch={() => void refetchMutation.mutateAsync()}
              onReprobe={() => {
                if (!firstInventoryFileId) return
                void reprobeMutation.mutateAsync(firstInventoryFileId)
              }}
            />
            <ArtworkCard
              posterUrl={item.poster_url}
              backdropUrl={item.backdrop_url}
            />
            <AssetLinksCard
              workspaceItem={{
                id: workspace.item_id,
                title: workspace.title,
                type: workspace.type,
                availability_status: workspace.availability_status,
                governance_status: workspace.governance_status,
              }}
              relatedChildren={workspaceRecommendedChildren}
              assets={workspaceAssets}
              reprobePendingFileId={
                typeof reprobeMutation.variables === 'number'
                  ? reprobeMutation.variables
                  : undefined
              }
              linkMutation={assetLinkMutation.variables}
              onReprobe={(fileId) => {
                void reprobeMutation.mutateAsync(fileId)
              }}
              onLink={(assetId, targetItemId) => {
                void assetLinkMutation.mutateAsync({
                  assetId,
                  targetItemId,
                  mode: 'link',
                })
              }}
              onUnlink={(assetId, targetItemId) => {
                void assetLinkMutation.mutateAsync({
                  assetId,
                  targetItemId,
                  mode: 'unlink',
                })
              }}
            />
            <RelatedChildrenCard
              workspace={workspace}
              assets={workspaceAssets}
            />
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

function buildDraftFromWorkspace(
  workspace: CatalogGovernanceWorkspace,
): MetadataDraft {
  const year = fieldStateNumber(workspace, 'year')

  return {
    title: workspace.title || '',
    originalTitle: fieldStateString(workspace, 'original_title'),
    year: year ? String(year) : '',
    overview: fieldStateString(workspace, 'overview'),
  }
}

function buildPreviewItem(workspace: CatalogGovernanceWorkspace) {
  return {
    id: workspace.item_id,
    library_id: workspace.library_id,
    type: workspace.type,
    title: workspace.title,
    original_title: fieldStateString(workspace, 'original_title'),
    overview: fieldStateString(workspace, 'overview'),
    poster_url: selectedImageUrl(workspace, 'poster'),
    backdrop_url: selectedImageUrl(workspace, 'backdrop'),
    year: fieldStateNumber(workspace, 'year'),
    governance_status: workspace.governance_status,
    availability_status: workspace.availability_status,
    metadata_provider: workspace.external_identities?.[0]?.provider ?? '',
    external_id: workspace.external_identities?.[0]?.external_id ?? '',
  }
}

function selectedImageUrl(
  workspace: CatalogGovernanceWorkspace,
  imageType: string,
) {
  return (
    (workspace.selected_images || []).find(
      (image) => image.image_type === imageType,
    )?.url || ''
  )
}

function fieldStateString(
  workspace: CatalogGovernanceWorkspace,
  fieldKey: string,
) {
  const value = fieldStateValue(workspace, fieldKey)
  return typeof value === 'string' ? value : ''
}

function fieldStateNumber(
  workspace: CatalogGovernanceWorkspace,
  fieldKey: string,
) {
  const value = fieldStateValue(workspace, fieldKey)
  return typeof value === 'number' ? value : undefined
}

function fieldStateValue(
  workspace: CatalogGovernanceWorkspace | undefined,
  fieldKey: string,
) {
  return (workspace?.field_states ?? []).find(
    (field) => field.field_key === fieldKey,
  )?.value
}

function parseOptionalNumber(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return undefined

  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? parsed : undefined
}

function uniqueMetadataCandidates(candidates: MetadataSearchCandidate[]) {
  const seen = new Set<string>()
  const result: MetadataSearchCandidate[] = []

  for (const candidate of candidates) {
    const key = metadataCandidateIdentity(candidate)
    if (seen.has(key)) continue

    seen.add(key)
    result.push(candidate)
  }

  return result
}

function metadataCandidateIdentity(candidate: MetadataSearchCandidate) {
  return `${candidate.provider.trim().toLowerCase()}-${candidate.external_id.trim()}`
}

function describeAsyncAction(type: AsyncActionState['type']) {
  switch (type) {
    case 'rematch':
      return '重新匹配'
    case 'refetch':
      return '元数据重抓'
    case 'reprobe':
      return '重新探测'
    default:
      return '后台动作'
  }
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
