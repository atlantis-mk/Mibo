import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { SearchIcon, LoaderCircleIcon } from 'lucide-react'
import type {
  CatalogGovernanceWorkspace,
  CatalogMetadataSearchCandidate,
  OperationsTask,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  miboQueryKeys,
  catalogGovernanceWorkspaceQueryOptions,
} from '@/lib/mibo-query'
import { cn } from '@/lib/utils'
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
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { ArtworkPreview } from '@/features/metadata-governance/detail-sections'

export function MetadataReviewDialog({
  token,
  open,
  task,
  onOpenChange,
  onResolved,
  onResolveReview,
}: {
  token: string | null
  open: boolean
  task: OperationsTask
  onOpenChange: (open: boolean) => void
  onResolved: () => Promise<void> | void
  onResolveReview?: () => Promise<void>
}) {
  const queryClient = useQueryClient()
  const affectedItems = task.affected.items ?? []
  const affectedFiles = task.affected.files ?? []
  const affectedItem = task.affected.items?.[0]
  const affectedLibrary = task.affected.libraries?.[0]
  const affectedFile = task.affected.files?.[0]
  const itemId = affectedItem?.id ?? 0
  const libraryId = affectedLibrary?.id
  const stageId = reviewStageIdFromTask(task)
  const [searchTitle, setSearchTitle] = useState(affectedItem?.title ?? '')
  const [searchYear, setSearchYear] = useState('')
  const [searchResults, setSearchResults] = useState<
    CatalogMetadataSearchCandidate[]
  >([])
  const [selectedCandidateExternalID, setSelectedCandidateExternalID] =
    useState('')

  const workspaceQuery = useQuery({
    ...catalogGovernanceWorkspaceQueryOptions(token ?? '', itemId),
    enabled: open && !!token && itemId > 0,
  })

  useEffect(() => {
    const candidates = workspaceQuery.data?.review_candidates ?? []
    if (!open || candidates.length === 0 || searchResults.length > 0) return
    setSearchResults(candidates)
    setSelectedCandidateExternalID(candidates[0]?.external_id ?? '')
  }, [open, searchResults.length, workspaceQuery.data?.review_candidates])

  const effectiveSearchTitle =
    searchTitle || workspaceQuery.data?.title || affectedItem?.title || ''
  const effectiveSearchYear =
    searchYear ||
    (fieldStateNumber(workspaceQuery.data, 'year')
      ? String(fieldStateNumber(workspaceQuery.data, 'year'))
      : '')

  const resolveMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('当前未登录，无法处理元数据确认。')
      if (onResolveReview) {
        await onResolveReview()
        return null
      }
      if (!itemId) throw new Error('当前任务缺少对应元数据条目。')
      const api = createAuthedMiboApi(token)
      const itemIds = uniqueAffectedItemIds(affectedItems)
      for (const targetItemId of itemIds) {
        await api.updateCatalogGovernanceField(
          targetItemId,
          {
            field_key: 'governance_status',
            value: 'manual',
            force: true,
          },
          { libraryId }
        )
      }
      if (!stageId) throw new Error('当前任务缺少可关闭的整理阶段。')
      return api.resolveIngestReviewStage(stageId)
    },
    onSuccess: async () => {
      await onResolved()
    },
  })

  const searchMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('当前未登录，无法搜索候选。')
      if (!itemId) throw new Error('当前任务缺少对应元数据条目。')
      return createAuthedMiboApi(token).searchCatalogGovernanceCandidates(
        itemId,
        {
          title: effectiveSearchTitle.trim() || undefined,
          year: parseOptionalNumber(effectiveSearchYear),
        },
        { libraryId }
      )
    },
    onSuccess: (response) => {
      setSearchResults(response.candidates ?? [])
      setSelectedCandidateExternalID(
        response.candidates?.[0]?.external_id ?? ''
      )
    },
  })

  const applyCandidateMutation = useMutation({
    mutationFn: async (externalID: string) => {
      if (!token) throw new Error('当前未登录，无法应用候选。')
      if (!itemId) throw new Error('当前任务缺少对应元数据条目。')
      const api = createAuthedMiboApi(token)
      const workspace = await api.applyCatalogGovernanceCandidate(
        itemId,
        { external_id: externalID },
        { libraryId }
      )
      if (onResolveReview) {
        await onResolveReview()
      } else if (stageId) {
        await api.resolveIngestReviewStage(stageId)
      }
      return workspace
    },
    onSuccess: async (workspace) => {
      queryClient.setQueryData(
        miboQueryKeys.catalogGovernanceWorkspace(token ?? '', itemId),
        workspace
      )
      setSearchResults([])
      setSelectedCandidateExternalID('')
      await onResolved()
    },
  })

  const selectedCandidate =
    searchResults.find(
      (candidate) => candidate.external_id === selectedCandidateExternalID
    ) ?? null

  const errorMessage =
    workspaceQuery.error?.message ||
    resolveMutation.error?.message ||
    searchMutation.error?.message ||
    applyCandidateMutation.error?.message ||
    null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[90vh] overflow-hidden sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>治理元数据确认</DialogTitle>
          <DialogDescription>
            确认当前元数据无误，或重新搜索并应用新的候选。处理完成后会自动清除此提示。
          </DialogDescription>
        </DialogHeader>

        <div className='flex max-h-[calc(90vh-7rem)] flex-col'>
          <div className='space-y-4 overflow-y-auto pr-1'>
            <Card className='flex flex-col border-border/60 bg-card/85 shadow-sm'>
              <CardHeader>
                <CardTitle className='text-base'>当前任务</CardTitle>
                <CardDescription>{task.summary}</CardDescription>
              </CardHeader>
              <CardContent className='grid gap-3 text-sm sm:grid-cols-2'>
                <SummaryCell
                  label='条目'
                  value={
                    affectedItem?.title
                      ? affectedItems.length > 1
                        ? `${affectedItem.title} 等 ${affectedItems.length} 个条目`
                        : affectedItem.title
                      : '未命名条目'
                  }
                />
                <SummaryCell
                  label='媒体库'
                  value={affectedLibrary?.name || '未知媒体库'}
                />
                <SummaryCell
                  label='文件'
                  value={affectedFile?.storage_path || '未记录'}
                  truncate
                />
                <SummaryCell
                  label='状态'
                  value={task.evidence[1]?.value || '待人工确认'}
                />
              </CardContent>
              {affectedItems.length > 1 || affectedFiles.length > 1 ? (
                <CardContent className='pt-0'>
                  <div className='rounded-lg border border-border/60 bg-background/70 px-3 py-3 text-sm'>
                    <div className='font-medium'>分组上下文</div>
                    <div className='mt-1 text-muted-foreground'>
                      当前问题覆盖 {affectedItems.length || 0} 个条目、{' '}
                      {affectedFiles.length || 0}{' '}
                      个文件。下面的治理工作区会先打开代表样本，方便你核对同组问题的共同原因。
                    </div>
                    {affectedItems.length > 1 ? (
                      <div className='mt-3 flex flex-wrap gap-2'>
                        {affectedItems.slice(0, 6).map((item) => (
                          <span
                            key={item.id}
                            className='rounded-full border border-border/60 bg-background px-2 py-1 text-xs'
                          >
                            {item.title}
                          </span>
                        ))}
                        {affectedItems.length > 6 ? (
                          <span className='rounded-full border border-border/60 bg-background px-2 py-1 text-xs text-muted-foreground'>
                            +{affectedItems.length - 6}
                          </span>
                        ) : null}
                      </div>
                    ) : null}
                  </div>
                </CardContent>
              ) : null}
            </Card>

            <Card className='border-border/60 bg-card/85 shadow-sm'>
              <CardHeader>
                <CardTitle className='text-base'>当前元数据</CardTitle>
                <CardDescription>
                  先核对现有内容；如果现在的元数据已经正确，可以直接确认并关闭提示。
                </CardDescription>
              </CardHeader>
              <CardContent className='gap-4'>
                {workspaceQuery.isLoading ? (
                  <div className='flex items-center gap-2 text-sm text-muted-foreground'>
                    <LoaderCircleIcon className='size-4 animate-spin' />
                    正在加载当前元数据
                  </div>
                ) : workspaceQuery.data ? (
                  <div className='grid gap-4 lg:grid-cols-[180px_minmax(0,1fr)]'>
                    <ArtworkPreview
                      label='当前封面'
                      imageUrl={selectedImageUrl(workspaceQuery.data, 'poster')}
                    />
                    <div className='grid gap-3 text-sm sm:grid-cols-2'>
                      <SummaryCell
                        label='标题'
                        value={workspaceQuery.data.title || '未填写'}
                      />
                      <SummaryCell
                        label='年份'
                        value={
                          fieldStateNumber(workspaceQuery.data, 'year')
                            ? String(
                                fieldStateNumber(workspaceQuery.data, 'year')
                              )
                            : '未填写'
                        }
                      />
                      <SummaryCell
                        label='治理状态'
                        value={
                          workspaceQuery.data.governance_status || 'pending'
                        }
                      />
                      <SummaryCell
                        label='外部 ID'
                        value={
                          workspaceQuery.data.external_identities?.[0]
                            ?.external_id || '未关联'
                        }
                      />
                    </div>
                  </div>
                ) : (
                  <div className='text-sm text-muted-foreground'>
                    当前无法加载条目详情，但仍可尝试直接确认或前往治理页处理。
                  </div>
                )}
              </CardContent>
            </Card>

            <Card className='flex flex-col border-border/60 bg-card/85 shadow-sm'>
              <CardHeader>
                <CardTitle className='text-base'>重新生成候选</CardTitle>
                <CardDescription>
                  按当前标题重新搜索远程候选，选择一个应用后会自动清除这条确认任务。
                </CardDescription>
              </CardHeader>
              <CardContent className='flex flex-1 flex-col gap-4'>
                <FieldGroup>
                  <div className='grid gap-4 md:grid-cols-[minmax(0,1fr)_140px_auto]'>
                    <Field>
                      <FieldLabel htmlFor='operations-review-search-title'>
                        候选标题
                      </FieldLabel>
                      <Input
                        id='operations-review-search-title'
                        value={effectiveSearchTitle}
                        onChange={(event) => {
                          setSearchTitle(event.target.value)
                          setSearchResults([])
                          setSelectedCandidateExternalID('')
                        }}
                        placeholder='输入标题或使用当前条目标题'
                      />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor='operations-review-search-year'>
                        年份
                      </FieldLabel>
                      <Input
                        id='operations-review-search-year'
                        inputMode='numeric'
                        value={effectiveSearchYear}
                        onChange={(event) => {
                          setSearchYear(event.target.value)
                          setSearchResults([])
                          setSelectedCandidateExternalID('')
                        }}
                        placeholder='可选'
                      />
                    </Field>
                    <Field className='self-end'>
                      <Button
                        onClick={() => void searchMutation.mutateAsync()}
                        disabled={searchMutation.isPending || !itemId}
                      >
                        {searchMutation.isPending ? (
                          <LoaderCircleIcon className='size-4 animate-spin' />
                        ) : (
                          <SearchIcon className='size-4' />
                        )}
                        搜索候选
                      </Button>
                    </Field>
                  </div>
                  <FieldDescription>
                    搜索会走当前媒体库的元数据策略，应用成功后自动关闭这条
                    review 提示。
                  </FieldDescription>
                </FieldGroup>

                <div>
                  {searchResults.length > 0 ? (
                    <ScrollArea className='max-h-96 pr-3'>
                      <div className='space-y-3'>
                        {searchResults.map((candidate) => (
                          <div
                            key={candidate.external_id}
                            className={cn(
                              'flex cursor-pointer items-stretch gap-3 rounded-xl border p-3 transition-colors',
                              selectedCandidateExternalID ===
                                candidate.external_id
                                ? 'border-primary bg-primary/5'
                                : 'border-border/60 bg-background/70 hover:border-primary/40'
                            )}
                            onClick={() =>
                              setSelectedCandidateExternalID(
                                candidate.external_id
                              )
                            }
                          >
                            <div className='h-20 w-14 shrink-0 overflow-hidden rounded-md bg-muted'>
                              {candidate.poster_url ? (
                                <img
                                  src={candidate.poster_url}
                                  alt={candidate.title}
                                  className='h-full w-full object-cover'
                                />
                              ) : null}
                            </div>
                            <div className='min-w-0 flex-1 space-y-1'>
                              <div className='text-sm font-medium text-foreground'>
                                {candidate.title}
                              </div>
                              <div className='text-xs text-muted-foreground'>
                                {candidate.original_title || '无原始标题'}
                              </div>
                              <div className='text-xs text-muted-foreground'>
                                {candidate.provider.toUpperCase()} ·{' '}
                                {candidate.media_type} ·{' '}
                                {candidate.year ?? '年份未知'} · 匹配度{' '}
                                {Math.round(candidate.confidence * 100)}%
                              </div>
                              <div className='line-clamp-2 text-xs text-muted-foreground'>
                                {candidate.reason_summary ||
                                  candidate.overview ||
                                  '无摘要'}
                              </div>
                            </div>
                            <div className='flex items-start'>
                              <div
                                className={cn(
                                  'mt-1 size-3 rounded-full border',
                                  selectedCandidateExternalID ===
                                    candidate.external_id
                                    ? 'border-primary bg-primary'
                                    : 'border-muted-foreground/40'
                                )}
                              />
                            </div>
                          </div>
                        ))}
                      </div>
                    </ScrollArea>
                  ) : (
                    <div className='text-sm text-muted-foreground'>
                      暂无候选，先执行一次搜索。
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            {errorMessage ? (
              <div className='rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive'>
                {errorMessage}
              </div>
            ) : null}
          </div>

          <div className='mt-4 border-t border-border/60 bg-background/95 pt-4 backdrop-blur'>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
              <div className='min-w-0 text-sm text-muted-foreground'>
                {selectedCandidate ? (
                  <div className='truncate'>
                    已选候选: {selectedCandidate.title} ·{' '}
                    {selectedCandidate.year ?? '年份未知'} · 匹配度{' '}
                    {Math.round(selectedCandidate.confidence * 100)}%
                  </div>
                ) : (
                  <div>未选择候选时，可以直接确认当前元数据。</div>
                )}
              </div>
              <div className='flex flex-wrap gap-2 sm:justify-end'>
                <Button
                  variant='outline'
                  onClick={() => {
                    if (!selectedCandidate) return
                    void applyCandidateMutation.mutateAsync(
                      selectedCandidate.external_id
                    )
                  }}
                  disabled={
                    !selectedCandidate || applyCandidateMutation.isPending
                  }
                >
                  {applyCandidateMutation.isPending ? (
                    <LoaderCircleIcon className='size-4 animate-spin' />
                  ) : null}
                  应用已选候选
                </Button>
                <Button
                  onClick={() => void resolveMutation.mutateAsync()}
                  disabled={
                    resolveMutation.isPending ||
                    applyCandidateMutation.isPending
                  }
                >
                  {resolveMutation.isPending ? (
                    <LoaderCircleIcon className='size-4 animate-spin' />
                  ) : null}
                  确认当前元数据
                </Button>
                {itemId > 0 ? (
                  <Button asChild variant='outline'>
                    <a href={`/settings/metadata/${itemId}`}>前往完整治理页</a>
                  </Button>
                ) : null}
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function reviewStageIdFromTask(task: OperationsTask) {
  const actionID = task.recommended_actions.find(
    (action) => action.type === 'resolve_review_stage' && action.id
  )?.id
  if (!actionID) return undefined
  const parts = actionID.split(':')
  const parsed = Number(parts[1])
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

function uniqueAffectedItemIds(items: OperationsTask['affected']['items']) {
  return Array.from(
    new Set(
      (items ?? [])
        .map((item) => item.id)
        .filter((itemId) => Number.isFinite(itemId) && itemId > 0)
    )
  )
}

function fieldStateValue(
  workspace: CatalogGovernanceWorkspace | undefined,
  fieldKey: string
) {
  return (workspace?.field_states ?? []).find(
    (field) => field.field_key === fieldKey
  )?.value
}

function fieldStateNumber(
  workspace: CatalogGovernanceWorkspace | undefined,
  fieldKey: string
) {
  const value = fieldStateValue(workspace, fieldKey)
  return typeof value === 'number' ? value : undefined
}

function selectedImageUrl(
  workspace: CatalogGovernanceWorkspace | undefined,
  imageType: string
) {
  return (
    (workspace?.selected_images ?? []).find(
      (image) => image.image_type === imageType
    )?.url || ''
  )
}

function parseOptionalNumber(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? parsed : undefined
}

function SummaryCell({
  label,
  value,
  truncate,
}: {
  label: string
  value: string
  truncate?: boolean
}) {
  return (
    <div className='rounded-lg border border-border/60 bg-background/70 px-3 py-3'>
      <div className='text-xs font-medium tracking-wide text-muted-foreground uppercase'>
        {label}
      </div>
      <div
        className={cn(
          'mt-2 text-sm text-foreground',
          truncate ? 'truncate' : undefined
        )}
        title={truncate ? value : undefined}
      >
        {value}
      </div>
    </div>
  )
}
