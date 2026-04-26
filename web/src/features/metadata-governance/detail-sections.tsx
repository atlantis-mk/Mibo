import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '#/components/ui/card'
import { Separator } from '#/components/ui/separator'
import type { MetadataSearchCandidate } from '#/lib/mibo-api'

export function CandidateCard({
  candidate,
  onPreview,
}: {
  candidate: MetadataSearchCandidate
  onPreview: () => void
}) {
  return (
    <div className="rounded-[1.25rem] border border-border/60 bg-background/60 p-4">
      <div className="flex gap-4">
        <div className="h-28 w-20 overflow-hidden rounded-lg bg-muted">
          {candidate.poster_url ? (
            <img
              src={candidate.poster_url}
              alt={candidate.title}
              className="h-full w-full object-cover"
            />
          ) : null}
        </div>
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <div className="text-sm font-medium text-foreground">
              {candidate.title}
            </div>
            <Badge variant="secondary">
              {candidate.provider.toUpperCase()}
            </Badge>
            <Badge variant="outline" className="border-border/60 bg-card/70">
              置信度 {(candidate.confidence * 100).toFixed(0)}%
            </Badge>
          </div>
          <div className="text-xs text-muted-foreground">
            {candidate.year ?? '年份未知'} ·{' '}
            {candidate.media_type === 'tv' ? '剧集' : '电影'}
          </div>
          {candidate.matched_query ? (
            <div className="text-xs text-muted-foreground">
              匹配 query: {candidate.matched_query}
            </div>
          ) : null}
          {candidate.reason_summary ? (
            <div className="text-xs text-muted-foreground">
              {candidate.reason_summary}
            </div>
          ) : null}
          <p className="line-clamp-3 text-sm text-muted-foreground">
            {candidate.overview || '暂无简介'}
          </p>
          <Button
            variant="outline"
            className="border-border/60 bg-card/70"
            onClick={onPreview}
          >
            预览并应用
          </Button>
        </div>
      </div>
    </div>
  )
}

export function CandidatePreviewCard({
  title,
  item,
  candidate,
}: {
  title: string
  item?: {
    title: string
    original_title?: string
    year?: number
    overview?: string
    poster_url?: string
    backdrop_url?: string
  }
  candidate?: MetadataSearchCandidate
}) {
  const posterUrl = item?.poster_url || candidate?.poster_url || ''
  const backdropUrl = item?.backdrop_url || candidate?.backdrop_url || ''
  const overview = item?.overview || candidate?.overview || '暂无简介'
  const mainTitle = item?.title || candidate?.title || '未命名'
  const originalTitle =
    item?.original_title || candidate?.original_title || '未填写'
  const year = item?.year ?? candidate?.year
  const matchedQuery = candidate?.matched_query || ''
  const reasonSummary = candidate?.reason_summary || ''

  return (
    <Card className="rounded-[1.25rem] border-border/60 bg-card/80 py-0">
      <CardHeader className="px-5 py-5">
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-4 px-5 py-5">
        <ArtworkPreview label="海报" imageUrl={posterUrl} />
        <ArtworkPreview label="背景图" imageUrl={backdropUrl} wide />
        <SummaryRow label="标题" value={mainTitle} />
        <SummaryRow label="原始标题" value={originalTitle} />
        <SummaryRow label="年份" value={year ? String(year) : '未填写'} />
        {matchedQuery ? (
          <SummaryRow label="匹配 query" value={matchedQuery} />
        ) : null}
        {reasonSummary ? (
          <SummaryRow label="候选说明" value={reasonSummary} multiline />
        ) : null}
        <SummaryRow label="简介" value={overview} multiline />
      </CardContent>
    </Card>
  )
}

export function ArtworkPreview({
  label,
  imageUrl,
  wide,
}: {
  label: string
  imageUrl: string
  wide?: boolean
}) {
  return (
    <div className="space-y-2">
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
        {label}
      </div>
      <div
        className={
          wide
            ? 'aspect-[16/7] overflow-hidden rounded-xl bg-muted'
            : 'aspect-[2/3] max-w-42 overflow-hidden rounded-xl bg-muted'
        }
      >
        {imageUrl ? (
          <img
            src={imageUrl}
            alt={label}
            className="h-full w-full object-cover"
          />
        ) : null}
      </div>
    </div>
  )
}

export function SummaryRow({
  label,
  value,
  multiline,
}: {
  label: string
  value: string
  multiline?: boolean
}) {
  return (
    <div className="space-y-1">
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
        {label}
      </div>
      <div
        className={
          multiline
            ? 'whitespace-pre-wrap text-sm text-foreground'
            : 'text-sm text-foreground'
        }
      >
        {value}
      </div>
    </div>
  )
}
