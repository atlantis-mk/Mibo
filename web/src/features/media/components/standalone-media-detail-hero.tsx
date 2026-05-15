import type { ReactNode } from "react"
import {
  Check,
  Ellipsis,
  Heart,
  LoaderCircle,
  Play,
  RefreshCw,
  Sparkles,
  Star,
} from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert"
import { Button } from "#/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu"
import type {
  MediaResourceDetail,
  MetadataResourceDetail,
  ProgressState,
} from "#/lib/mibo-api"
import type { MediaDetailPresentation } from "#/lib/media-presentation"
import {
  formatMediaDetailYearRange,
  formatMediaRating,
  formatProviderLabel,
  formatSeasonSummary,
} from "#/lib/media-presentation"
import { cn } from "#/lib/utils"

import {
  formatResourceLabel,
  formatDateTime,
  formatCompactFileSize,
  canPlayMediaDetailItem,
  formatMediaType,
  formatRuntime,
  formatSeconds,
  getPrimaryCatalogResource,
} from "./standalone-media-detail-utils"

export function DetailHeroSection({
  item,
  progress,
  itemProgressPercent,
  overviewExpanded,
  onOverviewExpandedChange,
  onOpenPlaybackEntry,
  resourceChoices = [],
  resourceSummaries = [],
  selectedResourceId,
  onSelectResource,
  isSelectingResource,
  onManageMetadata,
  onReprobePrimaryFile,
  isReprobePending,
  onMarkWatched,
  isFavorite,
  onFavoriteToggle,
}: {
  item: MediaDetailPresentation
  progress: ProgressState | null
  itemProgressPercent: number
  overviewExpanded: boolean
  onOverviewExpandedChange: (value: boolean) => void
  onOpenPlaybackEntry: (options?: {
    itemId?: number
    fromStart?: boolean
    resourceId?: number
  }) => void
  resourceChoices?: MetadataResourceDetail[]
  resourceSummaries?: MediaResourceDetail[]
  selectedResourceId?: number
  onSelectResource?: (resourceId: number) => void
  isSelectingResource: boolean
  onManageMetadata: () => void
  onReprobePrimaryFile?: () => void
  isReprobePending: boolean
  onMarkWatched: () => void
  isFavorite: boolean
  onFavoriteToggle: (favorite: boolean) => void
}) {
  const primaryResourceSummary = getPrimaryCatalogResource(item)
  const primaryResourceFileIds = primaryResourceSummary?.file_ids ?? []
  const selectedResource =
    resourceChoices.find((resource) => resource.id === selectedResourceId) ??
    resourceChoices[0]
  const isEpisode = item.type === "episode"
  const seriesPlaybackTarget =
    item.type === "series" ? item.series_playback_target : undefined
  const canPlay = canPlayMediaDetailItem(
    item,
    selectedResource,
    primaryResourceSummary
  )
  const hasResumableProgress =
    Boolean(progress && !progress.watched && progress.position_seconds > 0) ||
    seriesPlaybackTarget?.selection_reason === "continue"
  const primaryPlayLabel = canPlay
    ? hasResumableProgress
      ? "继续播放"
      : "播放"
    : item.availability_status === "unaired"
      ? "未播出"
      : "暂无播放"
  const ratingLabel = formatMediaRating(item.community_rating)
  const yearLabel = formatMediaDetailYearRange(item)
  const titleLine = item.original_title || item.title
  const resourceSummary = formatResourceLabel(primaryResourceSummary)
  const genreLabel = item.genres.slice(0, 3).join(" / ")
  const seasonSummary = formatSeasonSummary(item)
  const watched = Boolean(progress?.watched)
  const hasVisibleProgress = Boolean(
    progress && (progress.position_seconds > 0 || progress.watched)
  )

  return (
    <div className="max-w-[980px] min-w-0 pt-1">
      <div className="space-y-5">
        <div className="space-y-3">
          {isEpisode && item.episode_context?.series ? (
            <div className="text-base font-medium text-muted-foreground">
              {item.episode_context.series.title}
            </div>
          ) : null}
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="min-w-0 text-4xl font-semibold tracking-tight break-words text-foreground lg:text-[52px]">
              {item.title}
            </h1>
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-[15px] text-muted-foreground lg:text-base">
            {isEpisode && item.episode_label ? (
              <span>{item.episode_label}</span>
            ) : null}
            {ratingLabel ? (
              <span className="inline-flex items-center gap-1.5">
                <Star className="size-4 fill-primary text-primary" />
                {ratingLabel}
              </span>
            ) : null}
            {yearLabel ? <span>{yearLabel}</span> : null}
            {item.official_rating ? <span>{item.official_rating}</span> : null}
            {genreLabel ? <span>{genreLabel}</span> : null}
            {item.runtime_seconds ? (
              <span>{formatRuntime(item.runtime_seconds)}</span>
            ) : null}
            {item.metadata_provider ? (
              <span className="rounded border border-border/50 bg-background/70 px-1.5 py-0.5 text-xs text-muted-foreground">
                {formatProviderLabel(item.metadata_provider)}
              </span>
            ) : null}
            {seasonSummary ? <span>{seasonSummary}</span> : null}
            <span>{formatMediaType(item.type)}</span>
            {progress?.last_played_at ? (
              <span>结束于 {formatDateTime(progress.last_played_at)}</span>
            ) : null}
          </div>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-[15px] text-muted-foreground lg:text-base">
            <span>资源 {resourceSummary}</span>
            {primaryResourceSummary ? (
              <span>文件 {primaryResourceFileIds.length} 个</span>
            ) : null}
            {primaryResourceSummary?.probe_status ? (
              <span>探测 {primaryResourceSummary.probe_status}</span>
            ) : null}
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <Button
            size="lg"
            className="h-12 rounded-full px-8 text-base"
            onClick={() =>
                onOpenPlaybackEntry({
                  itemId: seriesPlaybackTarget?.episode_metadata_item_id,
                  resourceId: selectedResource?.id,
                })
            }
            disabled={!canPlay}
          >
            <Play className="size-4 fill-current" />
            {primaryPlayLabel}
          </Button>
          <Button
            size="icon"
            variant="outline"
            type="button"
            className={cn(
              "size-11 rounded-full border-border/50 bg-background/75 hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-primary",
              watched ? "text-emerald-400" : "text-muted-foreground"
            )}
            onClick={onMarkWatched}
            aria-label={watched ? "已看完" : "标记看完"}
          >
            <Check className={cn("size-4", watched ? "stroke-[3]" : "")} />
          </Button>
          <Button
            size="icon"
            variant="outline"
            type="button"
            className={cn(
              "size-11 rounded-full border-border/50 bg-background/75 hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-primary",
              isFavorite ? "text-rose-400" : "text-muted-foreground"
            )}
            onClick={() => onFavoriteToggle(!isFavorite)}
          >
            <Heart className={cn("size-4", isFavorite ? "fill-current" : "")} />
            <span className="sr-only">
              {isFavorite ? "取消收藏" : "加入收藏"}
            </span>
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                size="icon"
                variant="outline"
                type="button"
                className="size-11 rounded-full border-border/50 bg-background/75 text-muted-foreground hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-primary"
              >
                <Ellipsis className="size-4" />
                <span className="sr-only">更多操作</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-52">
              <DropdownMenuLabel>更多操作</DropdownMenuLabel>
              <DropdownMenuSeparator />
              {hasResumableProgress ? (
                <DropdownMenuItem
                  onSelect={() =>
                    onOpenPlaybackEntry({
                      itemId: seriesPlaybackTarget?.episode_metadata_item_id,
                      fromStart: true,
                      resourceId: selectedResource?.id,
                    })
                  }
                >
                  <Play className="size-4" />
                  从头播放
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuItem onSelect={onManageMetadata}>
                <Sparkles className="size-4" />
                治理元数据
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={isReprobePending || !onReprobePrimaryFile}
                onSelect={() => onReprobePrimaryFile?.()}
              >
                {isReprobePending ? (
                  <LoaderCircle className="size-4 animate-spin" />
                ) : (
                  <RefreshCw className="size-4" />
                )}
                {isReprobePending ? "探测排队中" : "重新探测"}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {resourceChoices.length > 1 ? (
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm text-muted-foreground">播放版本</span>
            {resourceChoices.map((resource, index) => (
              <PillButton
                key={resource.id}
                icon={<Play className="size-4" />}
                selected={selectedResource?.id === resource.id}
                label={describeResourceChoice(resource, index, resourceSummaries)}
                onClick={
                  onSelectResource
                    ? () => onSelectResource(resource.id)
                    : undefined
                }
                disabled={isSelectingResource}
              />
            ))}
          </div>
        ) : null}

        {isEpisode && item.episode_context?.incomplete_hierarchy ? (
          <Alert className="border-amber-400/30 bg-amber-950/20 text-foreground backdrop-blur-sm">
            <AlertTitle>剧集层级不完整</AlertTitle>
            <AlertDescription className="text-muted-foreground">
              当前集缺少完整的剧集或季信息，页面只展示已有元数据。可以在治理页修正季集编号。
            </AlertDescription>
          </Alert>
        ) : null}

        {!canPlay ? (
          <Alert className="border-border/40 bg-card/75 text-foreground backdrop-blur-sm">
            <AlertTitle>暂不可播放</AlertTitle>
            <AlertDescription className="text-muted-foreground">
              {item.availability_status === "unaired"
                ? "这一集尚未播出，仍可查看元数据和治理信息。"
                : "这一集还没有可播放的本地资源，仍可查看元数据和治理信息。"}
            </AlertDescription>
          </Alert>
        ) : null}

        <div className="space-y-3">
          <div className="text-[26px] font-semibold break-words text-foreground">
            {titleLine}
          </div>
          <div
            className={cn(
              "max-w-5xl text-[17px] leading-9 text-muted-foreground",
              !overviewExpanded && "line-clamp-4"
            )}
          >
            {item.overview ||
              "当前条目的元数据仍然较少。你可以在治理页编辑元数据，或者等待后续扫描完善内容。"}
          </div>
          {item.overview && item.overview.length > 120 ? (
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOverviewExpandedChange(!overviewExpanded)}
            >
              {overviewExpanded ? "收起" : "更多"}
            </Button>
          ) : null}
        </div>

        {hasVisibleProgress ? (
          <div className="max-w-[620px] rounded-[26px] border border-border/40 bg-card/75 px-5 py-4 backdrop-blur-md">
            <div className="flex items-center justify-between gap-4 text-sm text-muted-foreground">
              <span className="font-medium text-foreground">你的进度</span>
              <span>
                {formatSeconds(progress?.position_seconds ?? 0)} /{" "}
                {formatSeconds(progress?.duration_seconds ?? 0)}
              </span>
            </div>
            <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-primary"
                style={{ width: `${itemProgressPercent}%` }}
              />
            </div>
            <div className="mt-2 text-xs text-muted-foreground">
              {progress?.watched ? "已看完" : "继续观看中"} ·{" "}
              {itemProgressPercent}%
            </div>
          </div>
        ) : null}
      </div>
    </div>
  )
}

function PillButton({
  icon,
  label,
  selected = false,
  disabled = false,
  onClick,
}: {
  icon: ReactNode
  label: string
  selected?: boolean
  disabled?: boolean
  onClick?: () => void
}) {
  return (
    <Button
      size="lg"
      variant={selected ? "default" : "outline"}
      onClick={onClick}
      disabled={!onClick || disabled}
      className={selected ? "border-primary" : undefined}
    >
      {icon}
      {label}
    </Button>
  )
}

function describeResourceChoice(
  resource: MetadataResourceDetail,
  index: number,
  resourceSummaries: MediaResourceDetail[]
) {
  const totalSizeBytes = resourceSummaries
    .find((summary) => summary.id === resource.id)
    ?.files?.reduce((total, file) => total + (file.size_bytes || 0), 0)

  return [
    `版本 ${index + 1}`,
    totalSizeBytes !== undefined ? formatCompactFileSize(totalSizeBytes) : null,
  ]
    .filter(Boolean)
    .join(" · ")
}
