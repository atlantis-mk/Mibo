import {
  CheckIcon,
  LoaderCircleIcon,
  RefreshCwIcon,
  SearchIcon,
} from 'lucide-react'

import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import { Field, FieldGroup, FieldLabel } from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import { Separator } from '#/components/ui/separator'
import { Textarea } from '#/components/ui/textarea'
import type {
  CatalogAssetDetail,
  CatalogFieldState,
  CatalogGovernanceWorkspace,
  CatalogListItem,
  CatalogSelectedImage,
  CatalogSourceEvidence,
  MetadataSearchCandidate,
} from '#/lib/mibo-api'

import { ArtworkPreview, CandidateCard, SummaryRow } from './detail-sections'
import { formatMediaType } from './formatters'

type MetadataDraft = {
  title: string
  originalTitle: string
  year: string
  overview: string
}

export function DraftEditorCard({
  draft,
  baselineDraft,
  isDirty,
  isPending,
  onDraftChange,
  onReset,
  onSave,
}: {
  draft: MetadataDraft
  baselineDraft: MetadataDraft
  isDirty: boolean
  isPending: boolean
  onDraftChange: (updater: (current: MetadataDraft) => MetadataDraft) => void
  onReset: (baseline: MetadataDraft) => void
  onSave: () => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>基础元数据草稿</CardTitle>
        <CardDescription>
          按 Phase 7
          的统一草稿模式组织基础文本和图片字段，并通过一次保存提交当前编辑会话。
        </CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="px-5 py-5">
        <FieldGroup>
          <div className="grid gap-4 md:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="metadata-title">标题</FieldLabel>
              <Input
                id="metadata-title"
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
              <FieldLabel htmlFor="metadata-original-title">
                原始标题
              </FieldLabel>
              <Input
                id="metadata-original-title"
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
            <FieldLabel htmlFor="metadata-year">年份</FieldLabel>
            <Input
              id="metadata-year"
              inputMode="numeric"
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
            <FieldLabel htmlFor="metadata-overview">简介</FieldLabel>
            <Textarea
              id="metadata-overview"
              value={draft.overview}
              onChange={(event) =>
                onDraftChange((current) => ({
                  ...current,
                  overview: event.target.value,
                }))
              }
              className="min-h-32"
            />
          </Field>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              className="border-border/60 bg-background/70"
              onClick={() => onReset(baselineDraft)}
              disabled={!isDirty}
            >
              放弃草稿
            </Button>
            <Button onClick={onSave} disabled={!isDirty || isPending}>
              {isPending ? (
                <LoaderCircleIcon className="size-4 animate-spin" />
              ) : null}
              保存草稿
            </Button>
          </div>
        </FieldGroup>
      </CardContent>
    </Card>
  )
}

export function CandidateSearchCard({
  searchTitle,
  searchYear,
  searchIMDbId,
  searchTMDBId,
  searchTVDBId,
  isPending,
  isSuccess,
  activeCandidates,
  onSearchTitleChange,
  onSearchYearChange,
  onSearchIMDbIdChange,
  onSearchTMDBIdChange,
  onSearchTVDBIdChange,
  onSearch,
  onPreview,
}: {
  searchTitle: string
  searchYear: string
  searchIMDbId: string
  searchTMDBId: string
  searchTVDBId: string
  isPending: boolean
  isSuccess: boolean
  activeCandidates: MetadataSearchCandidate[]
  onSearchTitleChange: (value: string) => void
  onSearchYearChange: (value: string) => void
  onSearchIMDbIdChange: (value: string) => void
  onSearchTMDBIdChange: (value: string) => void
  onSearchTVDBIdChange: (value: string) => void
  onSearch: () => void
  onPreview: (candidate: MetadataSearchCandidate) => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>匹配候选治理</CardTitle>
        <CardDescription>
          搜索候选后先预览差异，再确认应用，避免直接覆盖当前元数据。
        </CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-4 px-5 py-5">
        <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_180px_auto]">
          <Field>
            <FieldLabel htmlFor="candidate-title">搜索标题</FieldLabel>
            <Input
              id="candidate-title"
              value={searchTitle}
              onChange={(event) => onSearchTitleChange(event.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="candidate-year">年份</FieldLabel>
            <Input
              id="candidate-year"
              inputMode="numeric"
              value={searchYear}
              onChange={(event) => onSearchYearChange(event.target.value)}
            />
          </Field>
          <div className="flex items-end">
            <Button className="w-full" onClick={onSearch} disabled={isPending}>
              {isPending ? (
                <LoaderCircleIcon className="size-4 animate-spin" />
              ) : (
                <SearchIcon className="size-4" />
              )}
              搜索候选
            </Button>
          </div>
        </div>
        <div className="grid gap-4 md:grid-cols-3">
          <Field>
            <FieldLabel htmlFor="candidate-imdb-id">IMDb ID</FieldLabel>
            <Input
              id="candidate-imdb-id"
              value={searchIMDbId}
              onChange={(event) => onSearchIMDbIdChange(event.target.value)}
              placeholder="tt1234567"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="candidate-tmdb-id">TMDB ID</FieldLabel>
            <Input
              id="candidate-tmdb-id"
              value={searchTMDBId}
              onChange={(event) => onSearchTMDBIdChange(event.target.value)}
              placeholder="101"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="candidate-tvdb-id">TVDB ID</FieldLabel>
            <Input
              id="candidate-tvdb-id"
              value={searchTVDBId}
              onChange={(event) => onSearchTVDBIdChange(event.target.value)}
              placeholder="12345"
            />
          </Field>
        </div>
        <p className="text-xs leading-5 text-muted-foreground">
          支持标题模糊搜索，也支持通过 IMDb / TMDB / TVDB ID 直接精确定位候选。
        </p>
        <div className="space-y-3">
          {activeCandidates.length ? (
            activeCandidates.map((candidate) => (
              <CandidateCard
                key={metadataCandidateKey(candidate)}
                candidate={candidate}
                onPreview={() => onPreview(candidate)}
              />
            ))
          ) : (
            <div className="rounded-[1.25rem] border border-dashed border-border/70 px-4 py-8 text-center text-sm text-muted-foreground">
              {isSuccess
                ? '没有找到候选，可以调整标题后重试。'
                : '输入标题后搜索候选。'}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function metadataCandidateKey(candidate: MetadataSearchCandidate) {
  return `${candidate.provider.trim().toLowerCase()}-${candidate.external_id.trim()}`
}

export function MetadataSummaryCard({
  item,
}: {
  item: {
    type: string
    title: string
    original_title?: string
    year?: number
    availability_status: string
    governance_status: string
    metadata_provider?: string
    external_id?: string
  }
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>当前元数据摘要</CardTitle>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-3 px-5 py-5 text-sm">
        <SummaryRow label="标题" value={item.title} />
        <SummaryRow label="原始标题" value={item.original_title || '未填写'} />
        <SummaryRow
          label="年份"
          value={item.year ? String(item.year) : '未填写'}
        />
        <SummaryRow label="类型" value={formatMediaType(item.type)} />
        <SummaryRow label="可用性" value={item.availability_status || '未知'} />
        <SummaryRow
          label="治理状态"
          value={item.governance_status || 'pending'}
        />
        <SummaryRow
          label="元数据来源"
          value={item.metadata_provider?.toUpperCase() || '未匹配'}
        />
        <SummaryRow label="外部 ID" value={item.external_id || '未关联'} />
      </CardContent>
    </Card>
  )
}

export function AsyncActionsCard({
  rematchPending,
  refetchPending,
  reprobePending,
  reprobeDisabled,
  onRematch,
  onRefetch,
  onReprobe,
}: {
  rematchPending: boolean
  refetchPending: boolean
  reprobePending: boolean
  reprobeDisabled: boolean
  onRematch: () => void
  onRefetch: () => void
  onReprobe: () => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>后台动作</CardTitle>
        <CardDescription>
          将重匹配和重抓拆开显示，保留明确职责边界。
        </CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-3 px-5 py-5">
        <Button
          className="w-full justify-start"
          variant="outline"
          onClick={onRematch}
          disabled={rematchPending}
        >
          {rematchPending ? (
            <LoaderCircleIcon className="size-4 animate-spin" />
          ) : (
            <RefreshCwIcon className="size-4" />
          )}
          重新匹配
        </Button>
        <Button
          className="w-full justify-start"
          variant="outline"
          onClick={onRefetch}
          disabled={refetchPending}
        >
          {refetchPending ? (
            <LoaderCircleIcon className="size-4 animate-spin" />
          ) : (
            <RefreshCwIcon className="size-4" />
          )}
          元数据重抓
        </Button>
        <Button
          className="w-full justify-start"
          variant="outline"
          onClick={onReprobe}
          disabled={reprobePending || reprobeDisabled}
        >
          {reprobePending ? (
            <LoaderCircleIcon className="size-4 animate-spin" />
          ) : (
            <RefreshCwIcon className="size-4" />
          )}
          重新探测主文件
        </Button>
      </CardContent>
    </Card>
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
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>图片预览</CardTitle>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="grid gap-3 px-5 py-5">
        <ArtworkPreview label="当前海报" imageUrl={posterUrl} />
        <ArtworkPreview label="当前背景图" imageUrl={backdropUrl} wide />
      </CardContent>
    </Card>
  )
}

export function FieldLocksCard({
  fieldStates,
  isPending,
  onToggleLock,
}: {
  fieldStates: CatalogFieldState[]
  isPending: boolean
  onToggleLock: (fieldKey: string, nextLocked: boolean) => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>字段锁</CardTitle>
        <CardDescription>锁定后的字段不会被自动重抓覆盖。</CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-3 px-5 py-5">
        {fieldStates.length ? (
          fieldStates.map((field) => (
            <div
              key={field.field_key}
              className="flex items-center justify-between gap-3 rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3"
            >
              <div>
                <div className="text-sm font-medium text-foreground">
                  {field.field_key}
                </div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {field.is_locked
                    ? field.lock_reason || '已锁定'
                    : '当前未锁定'}
                </div>
              </div>
              <Button
                size="sm"
                variant={field.is_locked ? 'secondary' : 'outline'}
                onClick={() => onToggleLock(field.field_key, !field.is_locked)}
                disabled={isPending}
              >
                {field.is_locked ? '解锁' : '锁定'}
              </Button>
            </div>
          ))
        ) : (
          <div className="text-sm text-muted-foreground">当前没有字段锁。</div>
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
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>来源证据</CardTitle>
        <CardDescription>展示 provider、语言、抓取时间和摘要。</CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-3 px-5 py-5">
        {sourceEvidence.length ? (
          sourceEvidence.map((source, index) => (
            <div
              key={catalogSourceEvidenceKey(source, index)}
              className="rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <div className="text-sm font-medium text-foreground">
                  {source.source_name}
                </div>
                <div className="text-xs text-muted-foreground">
                  {source.source_type}
                </div>
                {source.language ? (
                  <div className="text-xs text-muted-foreground">
                    {source.language}
                  </div>
                ) : null}
              </div>
              <div className="mt-1 text-xs text-muted-foreground">
                {source.external_id || '无外部 ID'} · {source.fetched_at}
              </div>
              <div className="mt-2 text-xs text-foreground/80 [overflow-wrap:anywhere]">
                {source.summary ? JSON.stringify(source.summary) : '无来源摘要'}
              </div>
            </div>
          ))
        ) : (
          <div className="text-sm text-muted-foreground">
            当前没有来源证据。
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function catalogSourceEvidenceKey(
  source: CatalogSourceEvidence,
  index: number,
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
  selectedImages,
  imageCandidates,
  isPending,
  onSelect,
}: {
  selectedImages: CatalogSelectedImage[]
  imageCandidates: CatalogSelectedImage[]
  isPending: boolean
  onSelect: (imageType: string, url: string) => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>图片候选</CardTitle>
        <CardDescription>点击候选即可切换当前选中图片。</CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-4 px-5 py-5">
        {(imageCandidates || []).length ? (
          imageCandidates.map((image) => {
            const isSelected = selectedImages.some(
              (selected) =>
                selected.image_type === image.image_type &&
                selected.url === image.url,
            )

            return (
              <div
                key={`${image.image_type}-${image.url}`}
                className="flex items-center gap-3 rounded-[1rem] border border-border/60 bg-background/60 p-3"
              >
                <div className="h-16 w-12 overflow-hidden rounded-md bg-muted">
                  {image.url ? (
                    <img
                      src={image.url}
                      alt={image.image_type}
                      className="h-full w-full object-cover"
                    />
                  ) : null}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium text-foreground">
                    {image.image_type}
                  </div>
                  <div className="line-clamp-2 text-xs text-muted-foreground">
                    {image.url}
                  </div>
                </div>
                <Button
                  size="sm"
                  variant={isSelected ? 'secondary' : 'outline'}
                  onClick={() => onSelect(image.image_type, image.url)}
                  disabled={isPending}
                >
                  {isSelected ? <CheckIcon className="size-4" /> : null}
                  {isSelected ? '已选中' : '设为当前'}
                </Button>
              </div>
            )
          })
        ) : (
          <div className="text-sm text-muted-foreground">
            当前没有图片候选。
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function AssetLinksCard({
  workspaceItem,
  relatedChildren,
  assets,
  reprobePendingFileId,
  linkMutation,
  onReprobe,
  onLink,
  onUnlink,
}: {
  workspaceItem: {
    id: number
    title: string
    type: string
    availability_status: string
    governance_status: string
  }
  relatedChildren: CatalogListItem[]
  assets: CatalogAssetDetail[]
  reprobePendingFileId?: number
  linkMutation?: {
    assetId: number
    targetItemId: number
    mode: 'link' | 'unlink'
  }
  onReprobe: (fileId: number) => void
  onLink: (assetId: number, targetItemId: number) => void
  onUnlink: (assetId: number, targetItemId: number) => void
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
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>资产链接</CardTitle>
        <CardDescription>
          展示播放版本、质量、链接条目和重新探测入口。
        </CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-3 px-5 py-5">
        {assets.length ? (
          assets.map((asset) => (
            <AssetLinkEditor
              key={asset.id}
              workspaceItem={workspaceItem}
              candidateItems={candidateItems}
              asset={asset}
              reprobePendingFileId={reprobePendingFileId}
              linkMutation={linkMutation}
              onReprobe={onReprobe}
              onLink={onLink}
              onUnlink={onUnlink}
            />
          ))
        ) : (
          <div className="text-sm text-muted-foreground">
            当前没有资产链接。
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function RelatedChildrenCard({
  workspace,
  assets,
}: {
  workspace: CatalogGovernanceWorkspace
  assets: CatalogAssetDetail[]
}) {
  const relatedChildren = workspace.recommended_children ?? []
  const linkedCounts = new Map<number, number>()
  for (const asset of assets) {
    for (const link of asset.links) {
      linkedCounts.set(link.item_id, (linkedCounts.get(link.item_id) ?? 0) + 1)
    }
  }

  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle>层级复核</CardTitle>
        <CardDescription>查看推荐子项的可用性和治理状态。</CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-3 px-5 py-5">
        {relatedChildren.length ? (
          relatedChildren.map((child) => (
            <div
              key={child.id}
              className="rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3"
            >
              <div className="text-sm font-medium text-foreground">
                {child.title}
              </div>
              <div className="mt-1 text-xs text-muted-foreground">
                {formatMediaType(child.type)} · {child.availability_status} ·{' '}
                {child.governance_status}
              </div>
              <div className="mt-2 text-xs text-muted-foreground">
                已链接资产 {linkedCounts.get(child.id) ?? 0} 个
              </div>
            </div>
          ))
        ) : (
          <div className="text-sm text-muted-foreground">
            当前没有待复核子项。
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function AssetLinkEditor({
  workspaceItem,
  candidateItems,
  asset,
  reprobePendingFileId,
  linkMutation,
  onReprobe,
  onLink,
  onUnlink,
}: {
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
  asset: CatalogAssetDetail
  reprobePendingFileId?: number
  linkMutation?: {
    assetId: number
    targetItemId: number
    mode: 'link' | 'unlink'
  }
  onReprobe: (fileId: number) => void
  onLink: (assetId: number, targetItemId: number) => void
  onUnlink: (assetId: number, targetItemId: number) => void
}) {
  const linkedItemIds = new Set(asset.links.map((link) => link.item_id))
  const linkedItems = asset.links.map((link) => ({
    ...link,
    item: candidateItems.find((candidate) => candidate.id === link.item_id),
  }))
  const availableTargets = candidateItems.filter(
    (candidate) => !linkedItemIds.has(candidate.id),
  )

  return (
    <div className="rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3">
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-medium text-foreground">
            {asset.display_name || `资产 ${asset.id}`}
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {[asset.asset_type, asset.edition, asset.quality_label]
              .filter(Boolean)
              .join(' · ')}
          </div>
        </div>
        <div className="text-xs text-muted-foreground">
          {asset.status} · {asset.probe_status}
        </div>
      </div>
      <div className="mt-2 text-xs text-muted-foreground">
        当前治理条目：{workspaceItem.title} · 关联条目 {asset.links.length} 个 ·
        文件 {asset.file_ids.length} 个
      </div>

      <div className="mt-3 space-y-2">
        <div className="text-xs font-medium text-foreground">现有链接</div>
        {linkedItems.length ? (
          linkedItems.map(({ item, ...link }) => {
            const isPending =
              linkMutation?.mode === 'unlink' &&
              linkMutation.assetId === asset.id &&
              linkMutation.targetItemId === link.item_id
            return (
              <div
                key={`${asset.id}-${link.item_id}-${link.role}-${link.segment_index}`}
                className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border/50 bg-card/70 px-3 py-2"
              >
                <div className="min-w-0 text-xs text-muted-foreground">
                  <span className="font-medium text-foreground">
                    {item?.title || `条目 ${link.item_id}`}
                  </span>{' '}
                  · {item ? formatMediaType(item.type) : '未知类型'} ·{' '}
                  {link.role}
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => onUnlink(asset.id, link.item_id)}
                  disabled={isPending}
                >
                  {isPending ? (
                    <LoaderCircleIcon className="size-4 animate-spin" />
                  ) : null}
                  取消链接
                </Button>
              </div>
            )
          })
        ) : (
          <div className="text-xs text-muted-foreground">
            当前没有已登记链接。
          </div>
        )}
      </div>

      <div className="mt-3 space-y-2">
        <div className="text-xs font-medium text-foreground">安全修正</div>
        {availableTargets.length ? (
          <div className="flex flex-wrap gap-2">
            {availableTargets.map((target) => {
              const isPending =
                linkMutation?.mode === 'link' &&
                linkMutation.assetId === asset.id &&
                linkMutation.targetItemId === target.id
              return (
                <Button
                  key={`${asset.id}-target-${target.id}`}
                  size="sm"
                  variant="outline"
                  onClick={() => onLink(asset.id, target.id)}
                  disabled={isPending}
                >
                  {isPending ? (
                    <LoaderCircleIcon className="size-4 animate-spin" />
                  ) : null}
                  链接到 {target.title}
                </Button>
              )
            })}
          </div>
        ) : (
          <div className="text-xs text-muted-foreground">
            当前治理条目及其推荐子项都已经包含此资产链接。
          </div>
        )}
      </div>

      <div className="mt-3 flex flex-wrap gap-2">
        {asset.file_ids.map((fileId) => (
          <Button
            key={fileId}
            size="sm"
            variant="outline"
            onClick={() => onReprobe(fileId)}
            disabled={reprobePendingFileId === fileId}
          >
            {reprobePendingFileId === fileId ? (
              <LoaderCircleIcon className="size-4 animate-spin" />
            ) : null}
            重新探测文件 {fileId}
          </Button>
        ))}
      </div>
    </div>
  )
}
