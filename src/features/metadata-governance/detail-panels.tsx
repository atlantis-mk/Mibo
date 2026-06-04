import type { ReactNode } from 'react'
import {
  CheckIcon,
  LoaderCircleIcon,
  RefreshCwIcon,
  SearchIcon,
} from 'lucide-react'
import type {
  CatalogClassificationDecision,
  CatalogFieldState,
  CatalogGovernanceWorkspace,
  CatalogListItem,
  CatalogMetadataSearchCandidate,
  CatalogSelectedImage,
  CatalogSourceEvidence,
  MediaResourceDetail,
} from '@/lib/mibo-api'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import { formatResourceVariantLabel } from '@/features/media/components/standalone-media-detail-utils'
import { ArtworkPreview, SummaryRow } from './detail-sections'
import {
  formatClassificationStatus,
  formatClassificationType,
  formatMediaType,
} from './formatters'

type PanelChrome = 'card' | 'dialog'

type MetadataDraft = {
  title: string
  originalTitle: string
  year: string
  overview: string
}

function GovernancePanel({
  chrome = 'card',
  title,
  description,
  bodyClassName,
  children,
}: {
  chrome?: PanelChrome
  title?: string
  description?: string
  bodyClassName?: string
  children: ReactNode
}) {
  if (chrome === 'dialog') {
    return (
      <section className='space-y-4'>
        {description ? (
          <p className='text-sm leading-6 text-muted-foreground'>
            {description}
          </p>
        ) : null}
        <div className={bodyClassName}>{children}</div>
      </section>
    )
  }

  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      {title ? (
        <CardHeader className='px-4 py-3'>
          <CardTitle className='text-base'>{title}</CardTitle>
          {description ? (
            <CardDescription>{description}</CardDescription>
          ) : null}
        </CardHeader>
      ) : null}
      <CardContent className={bodyClassName}>{children}</CardContent>
    </Card>
  )
}

export function DraftEditorCard({
  chrome = 'card',
  draft,
  baselineDraft,
  isDirty,
  isPending,
  onDraftChange,
  onReset,
  onSave,
}: {
  chrome?: PanelChrome
  draft: MetadataDraft
  baselineDraft: MetadataDraft
  isDirty: boolean
  isPending: boolean
  onDraftChange: (updater: (current: MetadataDraft) => MetadataDraft) => void
  onReset: (baseline: MetadataDraft) => void
  onSave: () => void
}) {
  return (
    <GovernancePanel
      chrome={chrome}
      title='基础元数据草稿'
      description='按 Phase 7 的统一草稿模式组织基础文本和图片字段，并通过一次保存提交当前编辑会话。'
      bodyClassName={chrome === 'dialog' ? undefined : 'px-5 py-5'}
    >
      <FieldGroup>
        <div className='grid gap-4 md:grid-cols-2'>
          <Field>
            <FieldLabel htmlFor='metadata-title'>标题</FieldLabel>
            <Input
              id='metadata-title'
              value={draft.title}
              onChange={(event) =>
                onDraftChange((current) => ({
                  ...current,
                  title: event.target.value,
                }))
              }
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='metadata-original-title'>原始标题</FieldLabel>
            <Input
              id='metadata-original-title'
              value={draft.originalTitle}
              onChange={(event) =>
                onDraftChange((current) => ({
                  ...current,
                  originalTitle: event.target.value,
                }))
              }
            />
          </Field>
        </div>
        <Field>
          <FieldLabel htmlFor='metadata-year'>年份</FieldLabel>
          <Input
            id='metadata-year'
            inputMode='numeric'
            value={draft.year}
            onChange={(event) =>
              onDraftChange((current) => ({
                ...current,
                year: event.target.value,
              }))
            }
          />
        </Field>
        <Field>
          <FieldLabel htmlFor='metadata-overview'>简介</FieldLabel>
          <Textarea
            id='metadata-overview'
            value={draft.overview}
            onChange={(event) =>
              onDraftChange((current) => ({
                ...current,
                overview: event.target.value,
              }))
            }
            className='min-h-32'
          />
        </Field>
        {chrome === 'card' ? (
          <div className='flex flex-wrap gap-2'>
            <Button
              variant='outline'
              className='border-border/60 bg-background/70'
              onClick={() => onReset(baselineDraft)}
              disabled={!isDirty}
            >
              放弃草稿
            </Button>
            <Button onClick={onSave} disabled={!isDirty || isPending}>
              {isPending ? (
                <LoaderCircleIcon className='size-4 animate-spin' />
              ) : null}
              保存草稿
            </Button>
          </div>
        ) : null}
      </FieldGroup>
    </GovernancePanel>
  )
}

export function MetadataSummaryCard({
  item,
}: {
  item: {
    type: string
    title: string
    original_title?: string
    local_title?: string
    year?: number
    availability_status: string
    governance_status: string
    metadata_provider?: string
    external_id?: string
  }
}) {
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='px-4 py-3'>
        <CardTitle className='text-base'>当前元数据摘要</CardTitle>
      </CardHeader>
      <CardContent className='grid gap-3 px-4 pb-4 text-sm sm:grid-cols-2 lg:grid-cols-4'>
        <SummaryRow label='正式标题' value={item.title} />
        <SummaryRow label='原始标题' value={item.original_title || '未填写'} />
        <SummaryRow label='本地识别标题' value={item.local_title || '未填写'} />
        <SummaryRow
          label='年份'
          value={item.year ? String(item.year) : '未填写'}
        />
        <SummaryRow label='类型' value={formatMediaType(item.type)} />
        <SummaryRow label='可用性' value={item.availability_status || '未知'} />
        <SummaryRow
          label='治理状态'
          value={item.governance_status || 'pending'}
        />
        <SummaryRow
          label='元数据来源'
          value={item.metadata_provider?.toUpperCase() || '未匹配'}
        />
        <SummaryRow label='外部 ID' value={item.external_id || '未关联'} />
      </CardContent>
    </Card>
  )
}

export function AsyncActionsCard({
  chrome = 'card',
  searchTitle,
  searchYear,
  onSearchTitleChange,
  onSearchYearChange,
  onSearch,
  searchPending,
  searchResults,
  applyPending,
  applyPendingExternalID,
  onApplyCandidate,
  reprobePending,
  reprobeDisabled,
  onReprobe,
}: {
  chrome?: PanelChrome
  searchTitle: string
  searchYear: string
  onSearchTitleChange: (value: string) => void
  onSearchYearChange: (value: string) => void
  onSearch: () => void
  searchPending: boolean
  searchResults: CatalogMetadataSearchCandidate[]
  applyPending: boolean
  applyPendingExternalID?: string
  onApplyCandidate: (externalID: string) => void
  reprobePending: boolean
  reprobeDisabled: boolean
  onReprobe: () => void
}) {
  return (
    <GovernancePanel
      chrome={chrome}
      title='后台动作'
      description='支持搜索远程元数据候选并应用，同时保留资源重新探测入口。'
      bodyClassName={chrome === 'dialog' ? 'space-y-4' : 'space-y-3 px-5 py-5'}
    >
      <FieldGroup>
        <div className='grid gap-4 md:grid-cols-[minmax(0,1fr)_140px_auto]'>
          <Field>
            <FieldLabel htmlFor='governance-search-title'>候选标题</FieldLabel>
            <Input
              id='governance-search-title'
              value={searchTitle}
              onChange={(event) => onSearchTitleChange(event.target.value)}
              placeholder='输入标题或使用当前条目标题'
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='governance-search-year'>年份</FieldLabel>
            <Input
              id='governance-search-year'
              inputMode='numeric'
              value={searchYear}
              onChange={(event) => onSearchYearChange(event.target.value)}
              placeholder='可选'
            />
          </Field>
          {chrome === 'card' ? (
            <Field className='self-end'>
              <Button onClick={onSearch} disabled={searchPending}>
                {searchPending ? (
                  <LoaderCircleIcon className='size-4 animate-spin' />
                ) : (
                  <SearchIcon className='size-4' />
                )}
                搜索候选
              </Button>
            </Field>
          ) : null}
        </div>
        <FieldDescription>
          会按当前库的 metadata strategy 搜索远程候选，应用后立即刷新治理页。
        </FieldDescription>
      </FieldGroup>

      <div className='space-y-2'>
        <div className='text-sm font-medium text-foreground'>远程候选</div>
        {searchResults.length ? (
          <ScrollArea className='max-h-96 pr-3'>
            <div className='divide-y divide-border/60'>
              {searchResults.map((candidate) => (
                <div
                  key={candidate.external_id}
                  className='flex gap-3 py-3 first:pt-0 last:pb-0'
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
                      {candidate.media_type} · {candidate.year ?? '年份未知'} ·
                      匹配度{' '}
                      {Math.round(
                        (candidate.match_score ?? candidate.confidence) * 100
                      )}
                      %{candidate.auto_match_eligible ? ' · 可自动匹配' : ''}
                    </div>
                    {candidate.match_score_breakdown ? (
                      <div className='text-xs text-muted-foreground'>
                        {candidate.match_score_breakdown.reason || '评分完成'}
                        {candidate.matched_language
                          ? ` · ${candidate.matched_language}`
                          : ''}
                      </div>
                    ) : null}
                    <div className='line-clamp-2 text-xs text-muted-foreground'>
                      {candidate.reason_summary ||
                        candidate.overview ||
                        '无摘要'}
                    </div>
                  </div>
                  <Button
                    size='sm'
                    onClick={() => onApplyCandidate(candidate.external_id)}
                    disabled={applyPending}
                  >
                    {applyPendingExternalID === candidate.external_id ? (
                      <LoaderCircleIcon className='size-4 animate-spin' />
                    ) : null}
                    应用候选
                  </Button>
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

      {chrome === 'card' ? (
        <Button
          className='w-full justify-start'
          variant='outline'
          onClick={onReprobe}
          disabled={reprobePending || reprobeDisabled}
        >
          {reprobePending ? (
            <LoaderCircleIcon className='size-4 animate-spin' />
          ) : (
            <RefreshCwIcon className='size-4' />
          )}
          重新探测主文件
        </Button>
      ) : null}
    </GovernancePanel>
  )
}

export function ArtworkCard({
  posterUrl,
  backdropUrl,
}: {
  posterUrl: string
  backdropUrl: string
}) {
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='px-4 py-3'>
        <CardTitle className='text-base'>图片预览</CardTitle>
      </CardHeader>
      <CardContent className='grid gap-3 px-4 pb-4'>
        <ArtworkPreview label='当前海报' imageUrl={posterUrl} />
        <ArtworkPreview label='当前背景图' imageUrl={backdropUrl} wide />
      </CardContent>
    </Card>
  )
}

export function FieldLocksCard({
  chrome = 'card',
  fieldStates,
  isPending,
  onToggleLock,
}: {
  chrome?: PanelChrome
  fieldStates: CatalogFieldState[]
  isPending: boolean
  onToggleLock: (fieldKey: string, nextLocked: boolean) => void
}) {
  return (
    <GovernancePanel
      chrome={chrome}
      title='字段锁'
      description='锁定后的字段不会被自动重抓覆盖。'
      bodyClassName={
        chrome === 'dialog'
          ? 'divide-y divide-border/60'
          : 'space-y-3 px-5 py-5'
      }
    >
      {fieldStates.length ? (
        fieldStates.map((field) => (
          <div
            key={field.field_key}
            className={
              chrome === 'dialog'
                ? 'flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0'
                : 'flex items-center justify-between gap-3 rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3'
            }
          >
            <div>
              <div className='text-sm font-medium text-foreground'>
                {field.field_key}
              </div>
              <div className='mt-1 text-xs text-muted-foreground'>
                {field.is_locked ? field.lock_reason || '已锁定' : '当前未锁定'}
              </div>
            </div>
            <Button
              size='sm'
              variant={field.is_locked ? 'secondary' : 'outline'}
              onClick={() => onToggleLock(field.field_key, !field.is_locked)}
              disabled={isPending}
            >
              {field.is_locked ? '解锁' : '锁定'}
            </Button>
          </div>
        ))
      ) : (
        <div className='text-sm text-muted-foreground'>当前没有字段锁。</div>
      )}
    </GovernancePanel>
  )
}

export function ClassificationReviewCard({
  decisions,
}: {
  decisions: CatalogClassificationDecision[]
}) {
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='px-4 py-3'>
        <CardTitle className='text-base'>分类复核</CardTitle>
        <CardDescription>
          展示扫描器对电影、单集、版本和附属视频的候选判断。
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4'>
        {decisions.length ? (
          decisions.map((decision) => {
            const alternatives = decision.alternatives ?? []
            const evidence = decision.evidence ?? []
            const correctionActions = decision.correction_actions ?? []

            return (
              <div
                key={decision.id}
                className='rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3 text-sm'
              >
                <div className='flex items-start justify-between gap-3'>
                  <div>
                    <div className='font-medium text-foreground'>
                      {formatClassificationType(decision.candidate_type ?? '')}
                    </div>
                    <div className='mt-1 text-xs text-muted-foreground'>
                      {decision.source_path}
                    </div>
                  </div>
                  <span className='rounded-full bg-muted px-2.5 py-1 text-xs text-muted-foreground'>
                    {formatClassificationStatus(decision.status)}
                  </span>
                </div>
                <div className='mt-3 grid gap-2 text-xs text-muted-foreground'>
                  <div>
                    置信度：
                    {typeof decision.confidence === 'number'
                      ? `${Math.round(decision.confidence * 100)}%`
                      : '未记录'}
                  </div>
                  {decision.reason ? <div>原因：{decision.reason}</div> : null}
                  {alternatives.length ? (
                    <div>
                      备选：
                      {alternatives
                        .map((item) => formatClassificationType(item.type))
                        .join('、')}
                    </div>
                  ) : null}
                  {evidence.length ? (
                    <div>
                      证据：
                      {evidence
                        .slice(0, 3)
                        .map((item) => item.value || item.kind)
                        .join('、')}
                    </div>
                  ) : null}
                  {correctionActions.length ? (
                    <div>
                      可选操作：
                      {correctionActions
                        .map((action) => action.label)
                        .join('、')}
                    </div>
                  ) : null}
                </div>
              </div>
            )
          })
        ) : (
          <div className='text-sm text-muted-foreground'>
            当前没有需要展示的分类复核项。
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function SourceEvidenceCard({
  sourceEvidence,
}: {
  sourceEvidence: CatalogSourceEvidence[]
}) {
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='px-4 py-3'>
        <CardTitle className='text-base'>来源证据</CardTitle>
        <CardDescription>展示 provider、语言、抓取时间和摘要。</CardDescription>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4'>
        {sourceEvidence.length ? (
          sourceEvidence.map((source, index) => (
            <div
              key={catalogSourceEvidenceKey(source, index)}
              className='rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3'
            >
              <div className='flex flex-wrap items-center gap-2'>
                <div className='text-sm font-medium text-foreground'>
                  {source.source_name}
                </div>
                <div className='text-xs text-muted-foreground'>
                  {source.source_type}
                </div>
                {source.language ? (
                  <div className='text-xs text-muted-foreground'>
                    {source.language}
                  </div>
                ) : null}
              </div>
              <div className='mt-1 text-xs text-muted-foreground'>
                {source.external_id || '无外部 ID'} · {source.fetched_at}
              </div>
              <div className='mt-2 text-xs [overflow-wrap:anywhere] text-foreground/80'>
                {source.summary ? JSON.stringify(source.summary) : '无来源摘要'}
              </div>
            </div>
          ))
        ) : (
          <div className='text-sm text-muted-foreground'>
            当前没有来源证据。
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function catalogSourceEvidenceKey(
  source: CatalogSourceEvidence,
  index: number
) {
  return [
    source.source_type,
    source.source_name,
    source.external_id || 'no-external-id',
    source.language || 'no-language',
    source.fetched_at,
    index,
  ]
    .map((part) => String(part).trim())
    .join('-')
}

export function ImageCandidatesCard({
  chrome = 'card',
  selectedImages,
  imageCandidates,
  isPending,
  onSelect,
}: {
  chrome?: PanelChrome
  selectedImages: CatalogSelectedImage[]
  imageCandidates: CatalogSelectedImage[]
  isPending: boolean
  onSelect: (imageType: string, url: string) => void
}) {
  return (
    <GovernancePanel
      chrome={chrome}
      title='图片候选'
      description='点击候选即可切换当前选中图片。'
      bodyClassName={
        chrome === 'dialog'
          ? 'divide-y divide-border/60'
          : 'space-y-4 px-5 py-5'
      }
    >
      {(imageCandidates || []).length ? (
        imageCandidates.map((image) => {
          const isSelected = selectedImages.some(
            (selected) =>
              selected.image_type === image.image_type &&
              selected.url === image.url
          )

          return (
            <div
              key={`${image.image_type}-${image.url}`}
              className={
                chrome === 'dialog'
                  ? 'flex items-center gap-3 py-3 first:pt-0 last:pb-0'
                  : 'flex items-center gap-3 rounded-[1rem] border border-border/60 bg-background/60 p-3'
              }
            >
              <div className='h-16 w-12 overflow-hidden rounded-md bg-muted'>
                {image.url ? (
                  <img
                    src={image.url}
                    alt={image.image_type}
                    className='h-full w-full object-cover'
                  />
                ) : null}
              </div>
              <div className='min-w-0 flex-1'>
                <div className='text-sm font-medium text-foreground'>
                  {image.image_type}
                </div>
                <div className='line-clamp-2 text-xs text-muted-foreground'>
                  {image.url}
                </div>
              </div>
              <Button
                size='sm'
                variant={isSelected ? 'secondary' : 'outline'}
                onClick={() => onSelect(image.image_type, image.url)}
                disabled={isPending}
              >
                {isSelected ? <CheckIcon className='size-4' /> : null}
                {isSelected ? '已选中' : '设为当前'}
              </Button>
            </div>
          )
        })
      ) : (
        <div className='text-sm text-muted-foreground'>当前没有图片候选。</div>
      )}
    </GovernancePanel>
  )
}

export function ResourceLinksCard({
  chrome = 'card',
  selectedFileId,
  workspaceItem,
  relatedChildren,
  resources,
  reprobePendingFileId,
  onSelectFile,
  onReprobe,
}: {
  chrome?: PanelChrome
  selectedFileId?: number
  workspaceItem: {
    id: number
    title: string
    type: string
    availability_status: string
    governance_status: string
  }
  relatedChildren: CatalogListItem[]
  resources: MediaResourceDetail[]
  reprobePendingFileId?: number
  onSelectFile?: (fileId: number) => void
  onReprobe: (fileId: number) => void
}) {
  const candidateItems = [
    {
      id: workspaceItem.id,
      title: workspaceItem.title,
      type: workspaceItem.type,
      availability_status: workspaceItem.availability_status,
      governance_status: workspaceItem.governance_status,
    },
    ...relatedChildren,
  ]

  return (
    <GovernancePanel
      chrome={chrome}
      title='资源链接'
      description='展示播放资源、质量、元数据链接和重新探测入口。'
      bodyClassName={
        chrome === 'dialog'
          ? 'divide-y divide-border/60'
          : 'space-y-3 px-5 py-5'
      }
    >
      {resources.length ? (
        resources.map((resource) => (
          <ResourceLinkEditor
            key={resource.id}
            chrome={chrome}
            selectedFileId={selectedFileId}
            workspaceItem={workspaceItem}
            candidateItems={candidateItems}
            resource={resource}
            reprobePendingFileId={reprobePendingFileId}
            onSelectFile={onSelectFile}
            onReprobe={onReprobe}
          />
        ))
      ) : (
        <div className='text-sm text-muted-foreground'>当前没有资源链接。</div>
      )}
    </GovernancePanel>
  )
}

export function RelatedChildrenCard({
  workspace,
  resources,
}: {
  workspace: CatalogGovernanceWorkspace
  resources: MediaResourceDetail[]
}) {
  const relatedChildren = workspace.recommended_children ?? []
  const linkedCounts = new Map<number, number>()
  for (const resource of resources) {
    for (const link of resource.links ?? []) {
      linkedCounts.set(
        link.metadata_item_id,
        (linkedCounts.get(link.metadata_item_id) ?? 0) + 1
      )
    }
  }

  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='px-4 py-3'>
        <CardTitle className='text-base'>层级复核</CardTitle>
        <CardDescription>查看推荐子项的可用性和治理状态。</CardDescription>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4'>
        {relatedChildren.length ? (
          relatedChildren.map((child) => (
            <div
              key={child.id}
              className='rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3'
            >
              <div className='text-sm font-medium text-foreground'>
                {child.title}
              </div>
              <div className='mt-1 text-xs text-muted-foreground'>
                {formatMediaType(child.type)} · {child.availability_status} ·{' '}
                {child.governance_status}
              </div>
              <div className='mt-2 text-xs text-muted-foreground'>
                已链接资源 {linkedCounts.get(child.id) ?? 0} 个
              </div>
            </div>
          ))
        ) : (
          <div className='text-sm text-muted-foreground'>
            当前没有待复核子项。
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ResourceLinkEditor({
  chrome = 'card',
  selectedFileId,
  workspaceItem,
  candidateItems,
  resource,
  reprobePendingFileId,
  onSelectFile,
  onReprobe,
}: {
  chrome?: PanelChrome
  selectedFileId?: number
  workspaceItem: {
    id: number
    title: string
  }
  candidateItems: Array<{
    id: number
    title: string
    type: string
    availability_status: string
    governance_status: string
  }>
  resource: MediaResourceDetail
  reprobePendingFileId?: number
  onSelectFile?: (fileId: number) => void
  onReprobe: (fileId: number) => void
}) {
  const resourceLinks = resource.links ?? []
  const linkedItems = resourceLinks.map((link) => ({
    ...link,
    item: candidateItems.find(
      (candidate) => candidate.id === link.metadata_item_id
    ),
  }))

  return (
    <div
      className={
        chrome === 'dialog'
          ? 'py-4 first:pt-0 last:pb-0'
          : 'rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3'
      }
    >
      <div className='flex items-center justify-between gap-3'>
        <div>
          <div className='text-sm font-medium text-foreground'>
            {formatResourceVariantLabel(resource)}
          </div>
          <div className='mt-1 text-xs text-muted-foreground'>
            {[resource.resource_type].filter(Boolean).join(' · ')}
          </div>
        </div>
        <div className='text-xs text-muted-foreground'>
          {resource.status} · {resource.probe_status}
        </div>
      </div>
      <div className='mt-2 text-xs text-muted-foreground'>
        当前治理条目：{workspaceItem.title} · 关联条目 {resourceLinks.length} 个
        · 文件 {(resource.file_ids ?? []).length} 个
      </div>

      <div className='mt-3 space-y-2'>
        <div className='text-xs font-medium text-foreground'>现有链接</div>
        {linkedItems.length ? (
          linkedItems.map(({ item, ...link }) => (
            <div
              key={`${resource.id}-${link.metadata_item_id}-${link.role}-${link.segment_index}`}
              className={
                chrome === 'dialog'
                  ? 'border-l border-border/60 pl-3'
                  : 'rounded-lg border border-border/50 bg-card/70 px-3 py-2'
              }
            >
              <div className='min-w-0 text-xs text-muted-foreground'>
                <span className='font-medium text-foreground'>
                  {item?.title || `条目 ${link.metadata_item_id}`}
                </span>{' '}
                · {item ? formatMediaType(item.type) : '未知类型'} · {link.role}
              </div>
            </div>
          ))
        ) : (
          <div className='text-xs text-muted-foreground'>
            当前没有已登记链接。
          </div>
        )}
      </div>

      <div className='mt-3 space-y-2'>
        <div className='text-xs font-medium text-foreground'>文件路径</div>
        {(resource.files ?? []).length ? (
          (resource.files ?? []).map((file) => (
            <div
              key={`${resource.id}-file-${file.file_id}`}
              className={
                chrome === 'dialog'
                  ? selectedFileId === file.file_id
                    ? 'cursor-pointer border-l border-primary bg-muted/50 px-3 py-2'
                    : 'cursor-pointer border-l border-border/60 px-3 py-2 hover:bg-muted/35'
                  : 'rounded-lg border border-border/50 bg-card/70 px-3 py-2'
              }
              onClick={
                chrome === 'dialog'
                  ? () => onSelectFile?.(file.file_id)
                  : undefined
              }
            >
              <div className='text-xs font-medium text-foreground'>
                文件 {file.file_id}
                {file.container ? ` · ${file.container}` : ''}
              </div>
              <div className='mt-1 text-xs [overflow-wrap:anywhere] text-muted-foreground'>
                {file.storage_path || '未记录路径'}
              </div>
            </div>
          ))
        ) : (
          <div className='text-xs text-muted-foreground'>
            当前没有可展示的文件路径。
          </div>
        )}
      </div>

      <div className='mt-3 space-y-2'>
        <div className='text-xs font-medium text-foreground'>安全修正</div>
        <div className='text-xs text-muted-foreground'>
          旧资产链接修正已下线；请使用资源治理操作修正 metadata 关系。
        </div>
      </div>

      {chrome === 'card' ? (
        <div className='mt-3 flex flex-wrap gap-2'>
          {(resource.file_ids ?? []).map((fileId) => (
            <Button
              key={fileId}
              size='sm'
              variant='outline'
              onClick={() => onReprobe(fileId)}
              disabled={reprobePendingFileId === fileId}
            >
              {reprobePendingFileId === fileId ? (
                <LoaderCircleIcon className='size-4 animate-spin' />
              ) : null}
              重新探测文件 {fileId}
            </Button>
          ))}
        </div>
      ) : null}
    </div>
  )
}
