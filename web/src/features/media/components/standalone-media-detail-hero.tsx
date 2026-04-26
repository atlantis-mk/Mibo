import type { ReactNode } from 'react'
import {
  Check,
  Ellipsis,
  Heart,
  Image as ImageIcon,
  LoaderCircle,
  Play,
  RefreshCw,
  Sparkles,
  Star,
  Trash2,
} from 'lucide-react'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '#/components/ui/alert-dialog'
import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Button } from '#/components/ui/button'
import type { CatalogAssetDetail, ProgressState } from '#/lib/mibo-api'
import type { CatalogDetailPresentation } from '#/lib/media-presentation'
import { cn } from '#/lib/utils'

import {
  describeMatchStatus,
  formatAssetLabel,
  formatDateTime,
  formatMediaType,
  formatRuntime,
  formatSeconds,
  getDisplayDatabaseLinks,
  getDisplayMatchStatus,
  getPrimaryCatalogAsset,
} from './standalone-media-detail-utils'

export function DetailHeroSection({
  item,
  progress,
  itemProgressPercent,
  overviewExpanded,
  onOverviewExpandedChange,
  onOpenPlaybackEntry,
  onOpenAssetPlaybackEntry,
  assetChoices = [],
  onManageMetadata,
  onRematchItem,
  onReprobePrimaryFile,
  isReprobePending,
  onMarkWatched,
}: {
  item: CatalogDetailPresentation
  progress: ProgressState | null
  itemProgressPercent: number
  overviewExpanded: boolean
  onOverviewExpandedChange: (value: boolean) => void
  onOpenPlaybackEntry: (options?: { fromStart?: boolean }) => void
  onOpenAssetPlaybackEntry?: (assetId: number) => void
  assetChoices?: CatalogAssetDetail[]
  onManageMetadata: () => void
  onRematchItem: () => void
  onReprobePrimaryFile?: () => void
  isReprobePending: boolean
  onMarkWatched: () => void
}) {
  const primaryAsset = getPrimaryCatalogAsset(item)
  const hasResumableProgress = Boolean(
    progress && !progress.watched && progress.position_seconds > 0,
  )
  const primaryPlayLabel = hasResumableProgress ? '继续播放' : '播放'
  const metadataScore =
    typeof item.metadata_confidence === 'number'
      ? (item.metadata_confidence * 10).toFixed(1)
      : null
  const yearLabel =
    item.year ?? (item.release_date ? item.release_date.slice(0, 4) : null)
  const titleLine = item.original_title || item.title
  const assetSummary = formatAssetLabel(primaryAsset)
  const databaseLinks = getDisplayDatabaseLinks(item)
  const matchStatus = getDisplayMatchStatus(item)

  return (
    <div className="min-w-0 max-w-[980px] pt-1">
      <div className="space-y-5">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="min-w-0 break-words text-4xl font-semibold tracking-tight text-foreground lg:text-[52px]">
              {item.title}
            </h1>
            <Button
              variant="ghost"
              size="icon-sm"
              className="size-8 rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            >
              <ImageIcon className="size-4" />
            </Button>
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-[15px] text-muted-foreground lg:text-base">
            {metadataScore ? (
              <span className="inline-flex items-center gap-1.5">
                <Star className="size-4 fill-primary text-primary" />
                {metadataScore}
              </span>
            ) : null}
            {yearLabel ? <span>{yearLabel}</span> : null}
            {item.runtime_seconds ? (
              <span>{formatRuntime(item.runtime_seconds)}</span>
            ) : null}
            {databaseLinks ? (
              <span className="rounded border border-border/50 bg-background/70 px-1.5 py-0.5 text-xs text-muted-foreground">
                {item.metadata_provider?.toUpperCase()}
              </span>
            ) : null}
            <span>{formatMediaType(item.type)}</span>
            {progress?.last_played_at ? (
              <span>结束于 {formatDateTime(progress.last_played_at)}</span>
            ) : null}
          </div>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-[15px] text-muted-foreground lg:text-base">
            <span>资源 {assetSummary}</span>
            {primaryAsset ? (
              <span>文件 {primaryAsset.file_ids.length} 个</span>
            ) : null}
            {primaryAsset?.probe_status ? (
              <span>探测 {primaryAsset.probe_status}</span>
            ) : null}
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <Button
            size="lg"
            className="h-12 rounded-full px-8 text-base"
            onClick={() => onOpenPlaybackEntry()}
          >
            <Play className="size-4 fill-current" />
            {primaryPlayLabel}
          </Button>
          <PillIconButton icon={<Check className="size-4" />} />
          <PillIconButton icon={<Heart className="size-4" />} />
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                size="icon"
                variant="outline"
                className="size-11 rounded-full border-border/50 bg-background/75 text-foreground hover:bg-accent hover:text-accent-foreground"
              >
                <Trash2 className="size-4" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>标记看完</AlertDialogTitle>
                <AlertDialogDescription>
                  标记后该条目会从继续观看中移除，并在下次默认从头播放。确认继续？
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction onClick={onMarkWatched}>
                  确认标记
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
          <PillIconButton icon={<Ellipsis className="size-4" />} />
          <PillButton
            icon={<Sparkles className="size-4" />}
            label="治理元数据"
            onClick={onManageMetadata}
          />
          {hasResumableProgress ? (
            <PillButton
              icon={<Play className="size-4" />}
              label="从头播放"
              onClick={() => onOpenPlaybackEntry({ fromStart: true })}
            />
          ) : null}
          <PillButton
            icon={<Sparkles className="size-4" />}
            label="重新匹配"
            onClick={onRematchItem}
          />
          <PillButton
            icon={
              isReprobePending ? (
                <LoaderCircle className="size-4 animate-spin" />
              ) : (
                <RefreshCw className="size-4" />
              )
            }
            label={isReprobePending ? '探测排队中' : '重新探测'}
            onClick={isReprobePending ? undefined : onReprobePrimaryFile}
          />
        </div>

        {assetChoices.length > 1 && onOpenAssetPlaybackEntry ? (
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm text-muted-foreground">播放版本</span>
            {assetChoices.map((asset) => (
              <PillButton
                key={asset.id}
                icon={<Play className="size-4" />}
                label={describeAssetChoice(asset)}
                onClick={() => onOpenAssetPlaybackEntry(asset.id)}
              />
            ))}
          </div>
        ) : null}

        {describeMatchStatus(matchStatus) && matchStatus !== 'matched' ? (
          <Alert className="border-border/40 bg-card/75 text-foreground backdrop-blur-sm">
            <AlertTitle>元数据状态</AlertTitle>
            <AlertDescription className="text-muted-foreground">
              {describeMatchStatus(matchStatus)}
            </AlertDescription>
          </Alert>
        ) : null}

        <div className="space-y-3">
          <div className="break-words text-[26px] font-semibold text-foreground">
            {titleLine}
          </div>
          <div
            className={cn(
              'max-w-5xl text-[17px] leading-9 text-muted-foreground',
              !overviewExpanded && 'line-clamp-4',
            )}
          >
            {item.overview ||
              '当前条目的元数据仍然较少。你可以手动识别、重新匹配，或者等待后续扫描完善内容。'}
          </div>
          {item.overview && item.overview.length > 120 ? (
            <button
              type="button"
              className="text-base text-muted-foreground transition hover:text-foreground"
              onClick={() => onOverviewExpandedChange(!overviewExpanded)}
            >
              {overviewExpanded ? '收起' : '更多'}
            </button>
          ) : null}
        </div>

        {progress ? (
          <div className="max-w-[620px] rounded-[26px] border border-border/40 bg-card/75 px-5 py-4 backdrop-blur-md">
            <div className="flex items-center justify-between gap-4 text-sm text-muted-foreground">
              <span className="font-medium text-foreground">你的进度</span>
              <span>
                {formatSeconds(progress.position_seconds)} /{' '}
                {formatSeconds(progress.duration_seconds)}
              </span>
            </div>
            <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-primary"
                style={{ width: `${itemProgressPercent}%` }}
              />
            </div>
            <div className="mt-2 text-xs text-muted-foreground">
              {progress.watched ? '已看完' : '继续观看中'} ·{' '}
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
  onClick,
}: {
  icon: ReactNode
  label: string
  onClick?: () => void
}) {
  return (
    <Button
      size="lg"
      variant="outline"
      className="h-11 rounded-full border-border/50 bg-background/75 px-5 text-foreground hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-90"
      onClick={onClick}
      disabled={!onClick}
    >
      {icon}
      {label}
    </Button>
  )
}

function PillIconButton({ icon }: { icon: ReactNode }) {
  return (
    <Button
      size="icon"
      variant="outline"
      type="button"
      className="pointer-events-none size-11 rounded-full border-border/50 bg-background/75 text-muted-foreground opacity-80"
      tabIndex={-1}
      aria-hidden="true"
    >
      {icon}
    </Button>
  )
}

function describeAssetChoice(asset: CatalogAssetDetail) {
  return (
    [asset.display_name, asset.edition, asset.quality_label]
      .filter(Boolean)
      .join(' · ') || `版本 ${asset.id}`
  )
}
