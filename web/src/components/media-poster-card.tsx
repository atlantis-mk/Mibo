import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { HeartIcon, PlayIcon, RotateCcwIcon } from 'lucide-react'

import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import type { CatalogListItem, ProgressState } from '#/lib/mibo-api'
import {
  formatMediaCardTitle,
  formatMediaCardYearRange,
  getMediaCardBadgeCount,
  getMediaCardPosterUrl,
  getMediaCardType,
  isMediaCardPlayable,
} from '#/lib/media-presentation'
import { cn } from '#/lib/utils'

type MediaPosterCardProps = {
  item: CatalogListItem
  playbackItem?: CatalogListItem
  progress?: ProgressState | null
  libraryName?: string
  isFavorite?: boolean
  onFavoriteToggle?: (item: CatalogListItem, favorite: boolean) => void
  className?: string
}

export function MediaPosterCard({
  item,
  playbackItem,
  progress,
  libraryName,
  isFavorite,
  onFavoriteToggle,
  className,
}: MediaPosterCardProps) {
  const title = formatMediaCardTitle(item)
  const posterUrl = getMediaCardPosterUrl(item)
  const badgeCount = getMediaCardBadgeCount(item)
  const playable = isMediaCardPlayable(item)
  const hasProgress = Boolean(progress && progress.position_seconds > 0)
  const mediaType = getMediaCardType(item)
  const playTarget = playbackItem ?? item

  return (
    <article
      className={cn(
        'group relative w-[172px] shrink-0 sm:w-[204px]',
        className,
      )}
    >
      <div className="relative overflow-hidden rounded-[1.35rem] border border-border/40 bg-card/75 shadow-lg transition-transform duration-200 group-hover:-translate-y-1">
        <Link
          to="/media/$id"
          params={{ id: String(item.id) }}
          search={{ view: mediaType === 'show' ? 'series' : undefined }}
          className="block focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        >
          <div
            className="relative aspect-[2/3] bg-cover bg-center bg-muted"
            style={{
              backgroundImage: posterUrl
                ? `url(${posterUrl})`
                : 'linear-gradient(180deg, rgba(80,92,255,0.35), rgba(15,118,110,0.35))',
            }}
          >
            {badgeCount ? (
              <span className="absolute right-2 top-2 inline-flex min-w-7 items-center justify-center rounded-full bg-emerald-500 px-2 py-1 text-xs font-bold text-white shadow-lg">
                {badgeCount}
              </span>
            ) : null}
          </div>
        </Link>
        <div className="space-y-3 px-3 pb-3 pt-3">
          <Link
            to="/media/$id"
            params={{ id: String(item.id) }}
            search={{ view: mediaType === 'show' ? 'series' : undefined }}
            className="block focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <div className="line-clamp-1 text-sm font-semibold tracking-tight text-foreground sm:text-base">
              {title}
            </div>
            <div className="mt-1 text-xs text-muted-foreground">
              {formatMediaCardYearRange(item)}
            </div>
          </Link>
          <div className="flex items-center gap-2">
            <Badge
              className="rounded-full border-border/50 bg-background/80 px-2 py-0.5 text-[10px]"
              variant="outline"
            >
              {mediaType === 'movie' ? '电影' : '剧集'}
            </Badge>
            {libraryName ? (
              <span className="min-w-0 truncate text-[10px] text-muted-foreground">
                {libraryName}
              </span>
            ) : null}
          </div>
          <div className="flex items-center gap-2">
            <Button
              asChild
              size="icon-sm"
              variant="outline"
              disabled={!playable}
              className="size-8 rounded-full border-border/50 bg-background/80"
            >
              <Link
                to="/play/$id"
                params={{ id: String(playTarget.id) }}
                search={{ fromStart: !hasProgress, assetId: undefined }}
              >
                {hasProgress ? (
                  <RotateCcwIcon className="size-3.5" />
                ) : (
                  <PlayIcon className="size-3.5" />
                )}
                <span className="sr-only">
                  {hasProgress ? '继续播放' : '播放'}
                </span>
              </Link>
            </Button>
            {onFavoriteToggle ? (
              <Button
                type="button"
                size="icon-sm"
                variant="outline"
                className={cn(
                  'size-8 rounded-full border-border/50 bg-background/80',
                  isFavorite ? 'text-rose-400' : 'text-muted-foreground',
                )}
                onClick={() => onFavoriteToggle(item, !isFavorite)}
              >
                <HeartIcon
                  className={cn('size-3.5', isFavorite ? 'fill-current' : '')}
                />
                <span className="sr-only">
                  {isFavorite ? '取消收藏' : '加入收藏'}
                </span>
              </Button>
            ) : null}
          </div>
        </div>
      </div>
    </article>
  )
}

export function MediaRail({
  title,
  href,
  children,
}: {
  title: string
  href?: { libraryId: number }
  children: ReactNode
}) {
  return (
    <section>
      <div className="mb-4 flex items-center justify-between gap-3">
        {href ? (
          <Link
            to="/library/$id"
            params={{ id: String(href.libraryId) }}
            className="text-xl font-semibold tracking-tight text-foreground underline-offset-4 hover:underline"
          >
            {title}
          </Link>
        ) : (
          <h2 className="text-xl font-semibold tracking-tight text-foreground">
            {title}
          </h2>
        )}
      </div>
      <div className="overflow-x-auto pb-3">
        <div className="flex min-w-max gap-4">{children}</div>
      </div>
    </section>
  )
}
