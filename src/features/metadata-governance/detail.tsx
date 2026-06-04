import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  CheckCircle2Icon,
  LoaderCircleIcon,
  RefreshCwIcon,
  SearchIcon,
  WandSparklesIcon,
} from 'lucide-react'
import type {
  CatalogGovernanceWorkspace,
  CatalogMetadataSearchCandidate,
} from '@/lib/mibo-api'
import {
  catalogGovernanceWorkspaceQueryOptions,
  createAuthedMiboApi,
  miboQueryKeys,
} from '@/lib/mibo-query'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  ArtworkCard,
  ResourceLinksCard,
  AsyncActionsCard,
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
  status: 'queued' | 'running' | 'completed' | 'failed'
  message: string
}

type OperationDialog =
  | 'metadata'
  | 'actions'
  | 'locks'
  | 'images'
  | 'resources'
  | null

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
    itemId
  )
  const listWorkspaceQueryKey = miboQueryKeys.metadataWorkspace(token)
  const workspaceQuery = useQuery({
    ...catalogGovernanceWorkspaceQueryOptions(token, itemId),
  })
  const workspaceData = workspaceQuery.data

  const [draft, setDraft] = useState<MetadataDraft>(EMPTY_DRAFT)
  const [baselineDraft, setBaselineDraft] = useState<MetadataDraft>(EMPTY_DRAFT)
  const [operationDialog, setOperationDialog] = useState<OperationDialog>(null)
  const [asyncActionState, setAsyncActionState] =
    useState<AsyncActionState | null>(null)
  const [saveSuccessMessage, setSaveSuccessMessage] = useState('')
  const [candidateSearchTitle, setCandidateSearchTitle] = useState('')
  const [candidateSearchYear, setCandidateSearchYear] = useState('')
  const [searchResults, setSearchResults] = useState<
    CatalogMetadataSearchCandidate[]
  >([])
  const [selectedReprobeFileId, setSelectedReprobeFileId] = useState<
    number | undefined
  >()
  const isDirty = JSON.stringify(draft) !== JSON.stringify(baselineDraft)

  useEffect(() => {
    if (!workspaceData || isDirty) return

    const nextDraft = buildDraftFromWorkspace(workspaceData)
    setDraft(nextDraft)
    setBaselineDraft(nextDraft)
    setCandidateSearchTitle(workspaceSearchTitle(workspaceData))
    const year = fieldStateNumber(workspaceData, 'year')
    setCandidateSearchYear(year ? String(year) : '')
    const firstFileId = workspaceData.resources?.find(
      (resource) => (resource.file_ids ?? []).length > 0
    )?.file_ids[0]
    setSelectedReprobeFileId(firstFileId)
  }, [isDirty, workspaceData])

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
      setCandidateSearchTitle(workspaceSearchTitle(workspace))
      const year = fieldStateNumber(workspace, 'year')
      setCandidateSearchYear(year ? String(year) : '')
      setSearchResults([])
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

  const reprobeMutation = useMutation({
    mutationFn: (inventoryFileId: number) => {
      if (!inventoryFileId) {
        throw new Error('当前条目没有可重新探测的库存文件。')
      }
      return createAuthedMiboApi(token).reprobeInventoryFile(inventoryFileId)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: workspaceQueryKey })
      setAsyncActionState({
        type: 'reprobe',
        status: 'completed',
        message: '重新探测已提交，资源状态会在后台刷新。',
      })
    },
  })

  const searchCandidatesMutation = useMutation({
    mutationFn: () =>
      createAuthedMiboApi(token).searchCatalogGovernanceCandidates(
        itemId,
        {
          title: candidateSearchTitle.trim() || undefined,
          year: parseOptionalNumber(candidateSearchYear),
        },
        { libraryId: workspaceQuery.data?.library_id }
      ),
    onSuccess: (response) => {
      const candidates = response.candidates ?? []
      setSearchResults(candidates)
    },
  })

  const applyCandidateMutation = useMutation({
    mutationFn: (externalID: string) =>
      createAuthedMiboApi(token).applyCatalogGovernanceCandidate(
        itemId,
        { external_id: externalID },
        { libraryId: workspaceQuery.data?.library_id }
      ),
    onSuccess: async (workspace) => {
      const nextDraft = buildDraftFromWorkspace(workspace)
      setDraft(nextDraft)
      setBaselineDraft(nextDraft)
      setCandidateSearchTitle(workspaceSearchTitle(workspace))
      const year = fieldStateNumber(workspace, 'year')
      setCandidateSearchYear(year ? String(year) : '')
      setSearchResults([])
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

  if (workspaceQuery.isLoading) {
    return (
      <div className='flex items-center gap-3 px-1 py-2 text-foreground'>
        <LoaderCircleIcon className='size-4 animate-spin' />
        <span className='text-sm text-muted-foreground'>正在加载治理页</span>
      </div>
    )
  }

  if (workspaceQuery.error || !workspaceQuery.data) {
    return (
      <div className='px-1 py-2 text-foreground'>
        <div className='max-w-xl space-y-4'>
          <h1 className='text-2xl font-semibold tracking-tight'>
            治理页暂时不可用
          </h1>
          <p className='text-sm text-muted-foreground'>
            {workspaceQuery.error?.message ?? '未找到对应治理工作区。'}
          </p>
          <Button asChild variant='outline'>
            <Link to='/settings/metadata'>返回治理工作台</Link>
          </Button>
        </div>
      </div>
    )
  }

  const workspace = workspaceQuery.data
  const workspaceResources = workspace.resources ?? []
  const workspaceFieldStates = workspace.field_states ?? []
  const workspaceSourceEvidence = workspace.source_evidence ?? []
  const workspaceClassification = workspace.classification_decisions ?? []
  const workspaceSelectedImages = workspace.selected_images ?? []
  const workspaceImageCandidates = workspace.image_candidates ?? []
  const workspaceRecommendedChildren = workspace.recommended_children ?? []
  const item = buildPreviewItem(workspace)
  const firstInventoryFileId = workspaceResources.find(
    (resource) => (resource.file_ids ?? []).length > 0
  )?.file_ids[0]
  async function handleNavigateAway(
    to: '/' | '/settings/metadata' | '/media/$id'
  ) {
    if (isDirty && !window.confirm('当前有未保存修改，确认离开治理页吗？')) {
      return
    }

    if (to === '/media/$id') {
      await navigate({
        to,
        params: { id: String(itemId) },
        search: { view: undefined, episodePage: undefined },
      })
      return
    }

    await navigate({ to })
  }

  return (
    <>
      <div className='space-y-4 text-foreground'>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between'>
          <div className='min-w-0 space-y-2'>
            <div className='flex flex-wrap items-center gap-2'>
              <Badge
                variant='outline'
                className='border-border/60 bg-background/70'
              >
                单条目治理
              </Badge>
              <Badge variant='secondary'>
                {formatMediaType(workspace.type)}
              </Badge>
              <Badge
                variant='outline'
                className='border-border/60 bg-background/70'
              >
                {formatMatchStatus(workspace.governance_status)}
              </Badge>
            </div>
            <h1 className='truncate text-2xl font-semibold tracking-tight'>
              {workspace.title}
            </h1>
          </div>

          <div className='flex flex-wrap gap-2 lg:justify-end'>
            <Button
              variant='outline'
              className='border-border/60 bg-background/70'
              onClick={() => void handleNavigateAway('/settings/metadata')}
            >
              返回工作台
            </Button>
            <Button
              variant='outline'
              className='border-border/60 bg-background/70'
              onClick={() => void handleNavigateAway('/media/$id')}
            >
              查看详情页
            </Button>
          </div>
        </div>

        <div className='flex flex-wrap gap-2'>
          <Button onClick={() => setOperationDialog('metadata')}>
            编辑元数据
          </Button>
          <Button
            variant='outline'
            className='border-border/60 bg-background/70'
            onClick={() => setOperationDialog('actions')}
          >
            后台动作
          </Button>
          <Button
            variant='outline'
            className='border-border/60 bg-background/70'
            onClick={() => setOperationDialog('locks')}
          >
            字段锁
          </Button>
          <Button
            variant='outline'
            className='border-border/60 bg-background/70'
            onClick={() => setOperationDialog('images')}
          >
            图片选择
          </Button>
          <Button
            variant='outline'
            className='border-border/60 bg-background/70'
            onClick={() => setOperationDialog('resources')}
          >
            资源链接
          </Button>
        </div>

        {isDirty ? (
          <Alert>
            <WandSparklesIcon className='size-4' />
            <AlertTitle>存在未保存草稿</AlertTitle>
            <AlertDescription>
              离开当前页面前会要求确认。保存后会同步刷新治理页、媒体详情和工作台摘要。
            </AlertDescription>
          </Alert>
        ) : null}

        {saveSuccessMessage ? (
          <Alert>
            <CheckCircle2Icon className='size-4' />
            <AlertTitle>保存成功</AlertTitle>
            <AlertDescription>{saveSuccessMessage}</AlertDescription>
          </Alert>
        ) : null}

        {asyncActionState ? (
          <Alert>
            {asyncActionState.status === 'failed' ? (
              <WandSparklesIcon className='size-4' />
            ) : asyncActionState.status === 'completed' ? (
              <CheckCircle2Icon className='size-4' />
            ) : (
              <LoaderCircleIcon className='size-4 animate-spin' />
            )}
            <AlertTitle>{formatAsyncActionTitle(asyncActionState)}</AlertTitle>
            <AlertDescription>{asyncActionState.message}</AlertDescription>
          </Alert>
        ) : null}

        {saveDraftMutation.error ||
        searchCandidatesMutation.error ||
        applyCandidateMutation.error ||
        reprobeMutation.error ||
        lockMutation.error ||
        imageMutation.error ? (
          <Alert>
            <AlertTitle>操作失败</AlertTitle>
            <AlertDescription>
              {saveDraftMutation.error?.message ||
                searchCandidatesMutation.error?.message ||
                applyCandidateMutation.error?.message ||
                reprobeMutation.error?.message ||
                lockMutation.error?.message ||
                imageMutation.error?.message}
            </AlertDescription>
          </Alert>
        ) : null}

        <div className='space-y-4'>
          <MetadataSummaryCard item={item} />

          <div className='grid gap-4 lg:grid-cols-2'>
            <ArtworkCard
              posterUrl={item.poster_url}
              backdropUrl={item.backdrop_url}
            />
            <SourceEvidenceCard sourceEvidence={workspaceSourceEvidence} />
          </div>

          <div className='grid gap-4 lg:grid-cols-2'>
            <ClassificationReviewCard decisions={workspaceClassification} />
            <RelatedChildrenCard
              workspace={workspace}
              resources={workspaceResources}
            />
          </div>
        </div>
      </div>

      <Dialog
        open={operationDialog !== null}
        onOpenChange={(open) => !open && setOperationDialog(null)}
      >
        <DialogContent className='grid max-h-[90vh] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0 sm:max-w-5xl'>
          <DialogHeader className='px-6 pt-6 pb-4'>
            <DialogTitle>{operationDialogTitle(operationDialog)}</DialogTitle>
            <DialogDescription>
              操作内容集中在弹窗内处理，关闭后详情页继续保持信息展示视图。
            </DialogDescription>
          </DialogHeader>

          <div className='min-h-0 overflow-y-auto px-6 pb-6'>
            {operationDialog === 'metadata' ? (
              <DraftEditorCard
                chrome='dialog'
                draft={draft}
                baselineDraft={baselineDraft}
                isDirty={isDirty}
                isPending={saveDraftMutation.isPending}
                onDraftChange={(updater) => setDraft(updater)}
                onReset={setDraft}
                onSave={() => void saveDraftMutation.mutateAsync()}
              />
            ) : null}

            {operationDialog === 'actions' ? (
              <AsyncActionsCard
                chrome='dialog'
                searchTitle={candidateSearchTitle}
                searchYear={candidateSearchYear}
                onSearchTitleChange={(value) => {
                  setCandidateSearchTitle(value)
                  setSearchResults([])
                }}
                onSearchYearChange={(value) => {
                  setCandidateSearchYear(value)
                  setSearchResults([])
                }}
                onSearch={() => {
                  void searchCandidatesMutation.mutateAsync()
                }}
                searchPending={searchCandidatesMutation.isPending}
                searchResults={searchResults}
                applyPending={applyCandidateMutation.isPending}
                applyPendingExternalID={applyCandidateMutation.variables}
                onApplyCandidate={(externalID) => {
                  void applyCandidateMutation.mutateAsync(externalID)
                }}
                reprobePending={reprobeMutation.isPending}
                reprobeDisabled={!firstInventoryFileId}
                onReprobe={() => {
                  if (!firstInventoryFileId) return
                  void reprobeMutation.mutateAsync(firstInventoryFileId)
                }}
              />
            ) : null}

            {operationDialog === 'locks' ? (
              <FieldLocksCard
                chrome='dialog'
                fieldStates={workspaceFieldStates}
                isPending={lockMutation.isPending}
                onToggleLock={(fieldKey, nextLocked) => {
                  void lockMutation.mutateAsync({ fieldKey, nextLocked })
                }}
              />
            ) : null}

            {operationDialog === 'images' ? (
              <ImageCandidatesCard
                chrome='dialog'
                selectedImages={workspaceSelectedImages}
                imageCandidates={workspaceImageCandidates}
                isPending={imageMutation.isPending}
                onSelect={(imageType, url) => {
                  void imageMutation.mutateAsync({ imageType, url })
                }}
              />
            ) : null}

            {operationDialog === 'resources' ? (
              <ResourceLinksCard
                chrome='dialog'
                selectedFileId={selectedReprobeFileId}
                workspaceItem={{
                  id: workspace.metadata_item_id,
                  title: workspace.title,
                  type: workspace.type,
                  availability_status: workspace.availability_status,
                  governance_status: workspace.governance_status,
                }}
                relatedChildren={workspaceRecommendedChildren}
                resources={workspaceResources}
                reprobePendingFileId={
                  typeof reprobeMutation.variables === 'number'
                    ? reprobeMutation.variables
                    : undefined
                }
                onSelectFile={setSelectedReprobeFileId}
                onReprobe={(fileId) => {
                  void reprobeMutation.mutateAsync(fileId)
                }}
              />
            ) : null}
          </div>

          {operationDialog !== 'images' && operationDialog !== 'locks' ? (
            <DialogFooter className='border-t bg-background/95 px-6 py-4 backdrop-blur-sm'>
              {operationDialog === 'metadata' ? (
                <>
                  <Button
                    variant='outline'
                    className='border-border/60 bg-background/70'
                    onClick={() => setDraft(baselineDraft)}
                    disabled={!isDirty}
                  >
                    放弃草稿
                  </Button>
                  <Button
                    onClick={() => void saveDraftMutation.mutateAsync()}
                    disabled={!isDirty || saveDraftMutation.isPending}
                  >
                    {saveDraftMutation.isPending ? (
                      <LoaderCircleIcon className='size-4 animate-spin' />
                    ) : null}
                    保存草稿
                  </Button>
                </>
              ) : null}

              {operationDialog === 'actions' ? (
                <>
                  <Button
                    variant='outline'
                    className='border-border/60 bg-background/70'
                    onClick={() => {
                      if (!firstInventoryFileId) return
                      void reprobeMutation.mutateAsync(firstInventoryFileId)
                    }}
                    disabled={
                      reprobeMutation.isPending || !firstInventoryFileId
                    }
                  >
                    {reprobeMutation.isPending ? (
                      <LoaderCircleIcon className='size-4 animate-spin' />
                    ) : (
                      <RefreshCwIcon className='size-4' />
                    )}
                    重新探测主文件
                  </Button>
                  <Button
                    variant='outline'
                    className='border-border/60 bg-background/70'
                    onClick={() => void searchCandidatesMutation.mutateAsync()}
                    disabled={searchCandidatesMutation.isPending}
                  >
                    {searchCandidatesMutation.isPending ? (
                      <LoaderCircleIcon className='size-4 animate-spin' />
                    ) : (
                      <SearchIcon className='size-4' />
                    )}
                    搜索候选
                  </Button>
                </>
              ) : null}

              {operationDialog === 'resources' ? (
                <Button
                  onClick={() => {
                    if (!selectedReprobeFileId) return
                    void reprobeMutation.mutateAsync(selectedReprobeFileId)
                  }}
                  disabled={!selectedReprobeFileId || reprobeMutation.isPending}
                >
                  {reprobeMutation.isPending ? (
                    <LoaderCircleIcon className='size-4 animate-spin' />
                  ) : (
                    <RefreshCwIcon className='size-4' />
                  )}
                  重新探测选中文件
                </Button>
              ) : null}
            </DialogFooter>
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  )
}

function operationDialogTitle(dialog: OperationDialog) {
  switch (dialog) {
    case 'metadata':
      return '编辑元数据'
    case 'actions':
      return '后台动作'
    case 'locks':
      return '字段锁'
    case 'images':
      return '图片选择'
    case 'resources':
      return '资源链接'
    default:
      return '操作'
  }
}

function buildDraftFromWorkspace(
  workspace: CatalogGovernanceWorkspace
): MetadataDraft {
  const year = fieldStateNumber(workspace, 'year')

  return {
    title: workspace.title || '',
    originalTitle:
      fieldStateString(workspace, 'original_title') ||
      workspace.original_title ||
      '',
    year: year ? String(year) : workspace.year ? String(workspace.year) : '',
    overview:
      fieldStateString(workspace, 'overview') || workspace.overview || '',
  }
}

function buildPreviewItem(workspace: CatalogGovernanceWorkspace) {
  return {
    id: workspace.metadata_item_id,
    library_id: workspace.library_id,
    type: workspace.type,
    title: workspace.title,
    original_title:
      fieldStateString(workspace, 'original_title') ||
      workspace.original_title ||
      '',
    overview:
      fieldStateString(workspace, 'overview') || workspace.overview || '',
    local_title: workspace.local_title || '',
    poster_url: selectedImageUrl(workspace, 'poster'),
    backdrop_url: selectedImageUrl(workspace, 'backdrop'),
    year: fieldStateNumber(workspace, 'year') ?? workspace.year,
    governance_status: workspace.governance_status,
    availability_status: workspace.availability_status,
    metadata_provider: workspace.external_identities?.[0]?.provider ?? '',
    external_id: workspace.external_identities?.[0]?.external_id ?? '',
  }
}

function workspaceSearchTitle(workspace: CatalogGovernanceWorkspace) {
  return workspace.local_title || workspace.title || ''
}

function selectedImageUrl(
  workspace: CatalogGovernanceWorkspace,
  imageType: string
) {
  return (
    (workspace.selected_images || []).find(
      (image) => image.image_type === imageType
    )?.url || ''
  )
}

function fieldStateString(
  workspace: CatalogGovernanceWorkspace,
  fieldKey: string
) {
  const value = fieldStateValue(workspace, fieldKey)
  return typeof value === 'string' ? value : ''
}

function fieldStateNumber(
  workspace: CatalogGovernanceWorkspace,
  fieldKey: string
) {
  const value = fieldStateValue(workspace, fieldKey)
  return typeof value === 'number' ? value : undefined
}

function fieldStateValue(
  workspace: CatalogGovernanceWorkspace | undefined,
  fieldKey: string
) {
  return (workspace?.field_states ?? []).find(
    (field) => field.field_key === fieldKey
  )?.value
}

function parseOptionalNumber(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return undefined

  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? parsed : undefined
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
