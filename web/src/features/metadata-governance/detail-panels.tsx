import { LoaderCircleIcon, RefreshCwIcon, SearchIcon } from 'lucide-react'

import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import { Separator } from '#/components/ui/separator'
import { Textarea } from '#/components/ui/textarea'
import type { MediaItemDetail, MetadataSearchCandidate } from '#/lib/mibo-api'

import { ArtworkPreview, CandidateCard, SummaryRow } from './detail-sections'
import { formatMediaType } from './formatters'

type MetadataDraft = {
  title: string
  originalTitle: string
  year: string
  overview: string
  posterUrl: string
  backdropUrl: string
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
          <div className="grid gap-4 md:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="metadata-poster-url">海报 URL</FieldLabel>
              <Input
                id="metadata-poster-url"
                value={draft.posterUrl}
                onChange={(event) =>
                  onDraftChange((current) => ({
                    ...current,
                    posterUrl: event.target.value,
                  }))
                }
              />
              <FieldDescription>
                本阶段主路径仍然是从候选中选图，这里展示草稿值，便于比对是否被候选覆盖。
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="metadata-backdrop-url">
                背景图 URL
              </FieldLabel>
              <Input
                id="metadata-backdrop-url"
                value={draft.backdropUrl}
                onChange={(event) =>
                  onDraftChange((current) => ({
                    ...current,
                    backdropUrl: event.target.value,
                  }))
                }
              />
            </Field>
          </div>
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
  isPending,
  isSuccess,
  activeCandidates,
  onSearchTitleChange,
  onSearchYearChange,
  onSearch,
  onPreview,
}: {
  searchTitle: string
  searchYear: string
  isPending: boolean
  isSuccess: boolean
  activeCandidates: MetadataSearchCandidate[]
  onSearchTitleChange: (value: string) => void
  onSearchYearChange: (value: string) => void
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
        <div className="space-y-3">
          {activeCandidates.length ? (
            activeCandidates.map((candidate) => (
              <CandidateCard
                key={candidate.external_id}
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

export function MetadataSummaryCard({ item }: { item: MediaItemDetail }) {
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
        <SummaryRow
          label="类型标签"
          value={item.genres.length ? item.genres.join('、') : '未识别'}
        />
        <SummaryRow
          label="演员"
          value={
            item.cast.length
              ? item.cast
                  .slice(0, 4)
                  .map((person) => person.name)
                  .join('、')
              : '未识别'
          }
        />
        {item.type === 'episode' ? (
          <>
            <SummaryRow
              label="季 / 集"
              value={`S${String(item.season_number ?? 0).padStart(2, '0')} · E${String(item.episode_number ?? 0).padStart(2, '0')}`}
            />
            <SummaryRow
              label="剧集归属"
              value={item.series_title_display || item.series_title || '未关联'}
            />
          </>
        ) : null}
      </CardContent>
    </Card>
  )
}

export function AsyncActionsCard({
  rematchPending,
  refetchPending,
  onRematch,
  onRefetch,
}: {
  rematchPending: boolean
  refetchPending: boolean
  onRematch: () => void
  onRefetch: () => void
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
