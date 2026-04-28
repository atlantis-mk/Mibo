import type { ComponentType } from 'react'
import { Link } from '@tanstack/react-router'
import {
  ArrowUpRightIcon,
  ClapperboardIcon,
  InfoIcon,
  LibraryBigIcon,
  PlayIcon,
  TvIcon,
} from 'lucide-react'
import { Autoplay } from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/react'
import type { Swiper as SwiperType } from 'swiper/types'

import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { MediaPosterCard, MediaRail } from '#/components/media-poster-card'
import type {
  CatalogListItem,
  CatalogUserItemEntry,
  Library,
} from '#/lib/mibo-api'
import {
  formatMediaCardTitle,
  getMediaCardBackdropUrl,
  getMediaCardPosterUrl,
  getMediaCardType,
  getPrimarySeriesTitle,
} from '#/lib/media-presentation'
import { cn } from '#/lib/utils'

const DEFAULT_OVERVIEW =
  '最近加入的内容会在这里轮播展示，方便在首页快速发现刚扫描入库的媒体。'

export function HeroCarousel({
  heroItems,
  canLoopHeroItems,
  selectedIndex,
  userName,
  continueWatchingCount,
  movieCount,
  showCount,
  onSwiper,
  onSlideChange,
  onDotClick,
}: {
  heroItems: any[]
  canLoopHeroItems: boolean
  selectedIndex: number
  userName: string
  continueWatchingCount: number
  movieCount: number
  showCount: number
  onSwiper: (instance: SwiperType) => void
  onSlideChange: (instance: SwiperType) => void
  onDotClick: (index: number) => void
}) {
  return (
    <Swiper
      modules={canLoopHeroItems ? [Autoplay] : undefined}
      loop={canLoopHeroItems}
      slidesPerView={1}
      autoplay={
        canLoopHeroItems
          ? {
              delay: 5000,
              disableOnInteraction: false,
              pauseOnMouseEnter: true,
            }
          : false
      }
      onSwiper={onSwiper}
      onSlideChange={onSlideChange}
      className="w-full"
    >
      {heroItems.map((item) => {
        const backgroundImage =
          getMediaCardBackdropUrl(item) || getMediaCardPosterUrl(item)
        const displayTitle = formatMediaCardTitle(item)
        const seriesTitle = getPrimarySeriesTitle(item)
        return (
          <SwiperSlide key={item.id}>
            <section
              className="relative min-h-svh overflow-hidden"
              style={
                backgroundImage
                  ? undefined
                  : {
                      background:
                        'linear-gradient(135deg, rgba(5,10,18,1), rgba(30,41,59,0.88), rgba(15,118,110,0.66))',
                    }
              }
            >
              {backgroundImage ? (
                <>
                  <img
                    src={backgroundImage}
                    alt={item.title}
                    className="absolute inset-0 h-full w-full object-cover"
                  />
                  <div className="absolute inset-0 bg-linear-to-r from-background via-background/15 to-background/95" />
                  <div className="absolute inset-0 bg-linear-to-t from-background/95 via-background/20 to-background/10" />
                </>
              ) : null}
              <div className="relative flex min-h-svh items-end px-6 py-8 sm:px-8 lg:px-12 lg:py-10">
                <div className="grid w-full gap-6 xl:grid-cols-[minmax(0,1.25fr)_340px] xl:items-end">
                  <div className="max-w-4xl min-w-0">
                    <Badge
                      className="border-border/50 bg-background/75 backdrop-blur-sm"
                      variant="outline"
                    >
                      最近加入
                    </Badge>
                    <div className="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground sm:text-sm">
                      <span>当前用户 {userName}</span>
                      <span>•</span>
                      <span>{formatMediaType(getMediaCardType(item))}</span>
                      {item.year ? (
                        <>
                          <span>•</span>
                          <span>{item.year}</span>
                        </>
                      ) : null}
                      {getMediaCardType(item) !== 'show' && seriesTitle ? (
                        <>
                          <span>•</span>
                          <span>{seriesTitle}</span>
                        </>
                      ) : null}
                    </div>
                    <h1 className="mt-5 max-w-3xl text-4xl font-semibold tracking-tight text-balance sm:text-5xl lg:text-6xl">
                      {displayTitle}
                    </h1>
                    <p className="mt-4 max-w-2xl text-sm leading-7 text-muted-foreground sm:text-base">
                      {item.overview || DEFAULT_OVERVIEW}
                    </p>
                    <div className="mt-6 flex flex-wrap gap-3">
                      <Button asChild size="lg">
                        <Link
                          to="/play/$id"
                          params={{ id: String(item.id) }}
                          search={{ fromStart: false, assetId: undefined }}
                        >
                          <PlayIcon className="size-4" />
                          播放
                        </Link>
                      </Button>
                      <Button
                        asChild
                        size="lg"
                        variant="outline"
                        className="border-border/50 bg-background/75 hover:bg-accent hover:text-accent-foreground"
                      >
                        <Link
                          to="/media/$id"
                          params={{ id: String(item.id) }}
                          search={{
                            view:
                              getMediaCardType(item) === 'show'
                                ? 'series'
                                : undefined,
                          }}
                        >
                          <InfoIcon className="size-4" />
                          详情
                        </Link>
                      </Button>
                    </div>
                  </div>

                  <div className="grid gap-3 sm:grid-cols-3 xl:grid-cols-1">
                    <StatCard
                      icon={PlayIcon}
                      label="轮播条目"
                      value={heroItems.length}
                      description="首页正在展示最近加入的内容。"
                    />
                    <StatCard
                      icon={LibraryBigIcon}
                      label="继续观看"
                      value={continueWatchingCount}
                      description="优先回到上次中断的位置。"
                    />
                    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-2">
                      <StatCard
                        icon={ClapperboardIcon}
                        label="电影"
                        value={movieCount}
                        description="最近加入的电影数量。"
                        compact
                      />
                      <StatCard
                        icon={TvIcon}
                        label="剧集"
                        value={showCount}
                        description="最近加入的剧集数量。"
                        compact
                      />
                    </div>
                  </div>
                </div>
              </div>
            </section>
          </SwiperSlide>
        )
      })}

      {heroItems.length > 1 ? (
        <div className="pointer-events-none absolute inset-x-0 bottom-6 z-20 flex justify-center px-6">
          <div className="pointer-events-auto inline-flex items-center gap-2 rounded-full border border-border/50 bg-background/70 px-3 py-2 backdrop-blur-xl">
            {heroItems.map((item, index) => (
              <button
                key={item.id}
                type="button"
                aria-label={`切换到第 ${index + 1} 张幻灯片`}
                onClick={() => onDotClick(index)}
                className={cn(
                  'h-2.5 rounded-full bg-muted transition-all',
                  selectedIndex === index
                    ? 'w-8 bg-primary'
                    : 'w-2.5 hover:bg-muted-foreground/60',
                )}
              />
            ))}
          </div>
        </div>
      ) : null}
    </Swiper>
  )
}

export function LatestLibraryRail({
  latestLibrarySections,
  favoriteIds,
  onFavoriteToggle,
}: {
  latestLibrarySections: any[]
  favoriteIds?: Set<number>
  onFavoriteToggle?: (item: CatalogListItem, favorite: boolean) => void
}) {
  return (
    <section className="relative border-t border-border/40 bg-background px-4 pb-16 pt-10 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-[1600px]">
        {latestLibrarySections.length > 0 ? (
          <div className="space-y-8">
            {latestLibrarySections.map((section) => (
              <MediaRail
                key={section.library_id}
                title={`最新${section.library_name}`}
                href={{ libraryId: section.library_id }}
              >
                {section.items.map((item: CatalogListItem) => (
                  <MediaPosterCard
                    key={item.id}
                    item={item}
                    libraryName={section.library_name}
                    isFavorite={favoriteIds?.has(item.id)}
                    onFavoriteToggle={onFavoriteToggle}
                  />
                ))}
              </MediaRail>
            ))}
          </div>
        ) : (
          <div className="mt-8 rounded-[2rem] border border-border/40 bg-card/70 px-6 py-8 text-sm text-muted-foreground backdrop-blur-sm">
            还没有可展示的最新内容，稍后会按媒体库自动补充到这里。
          </div>
        )}
      </div>
    </section>
  )
}

export function MyMediaSection({
  libraries,
  latestLibrarySections,
}: {
  libraries: Library[]
  latestLibrarySections: { library_id: number; items: CatalogListItem[] }[]
}) {
  const postersByLibrary = new Map(
    latestLibrarySections.map((section) => [
      section.library_id,
      section.items.map(getMediaCardPosterUrl).filter(Boolean).slice(0, 4),
    ]),
  )

  if (libraries.length === 0) {
    return (
      <section className="px-4 py-10 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-[1600px] rounded-[2rem] border border-border/40 bg-card/70 px-6 py-8 text-sm text-muted-foreground backdrop-blur-sm">
          还没有媒体库。前往设置添加媒体源和媒体库后，这里会显示你的媒体入口。
        </div>
      </section>
    )
  }

  return (
    <section className="px-4 py-10 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-[1600px]">
        <div className="mb-5 flex items-end justify-between gap-4">
          <div>
            <div className="text-xs tracking-[0.24em] text-muted-foreground uppercase">
              My Media
            </div>
            <h2 className="mt-2 text-2xl font-semibold tracking-tight">
              我的媒体
            </h2>
          </div>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {libraries.map((library) => (
            <LibraryCollageCard
              key={library.id}
              library={library}
              posters={postersByLibrary.get(library.id) ?? []}
            />
          ))}
        </div>
      </div>
    </section>
  )
}

export function ContinueWatchingRail({
  entries,
  favoriteIds,
  onFavoriteToggle,
}: {
  entries: CatalogUserItemEntry[]
  favoriteIds?: Set<number>
  onFavoriteToggle?: (item: CatalogListItem, favorite: boolean) => void
}) {
  if (entries.length === 0) {
    return null
  }

  return (
    <section className="border-t border-border/40 bg-background px-4 py-10 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-[1600px]">
        <MediaRail title="继续观看">
          {entries.map((entry) => {
            const displayItem = entry.display_item ?? entry.item
            const playbackItem = entry.play_item ?? entry.item

            return (
              <MediaPosterCard
                key={`${entry.item.id}-${entry.asset_id ?? 'default'}`}
                item={displayItem}
                playbackItem={playbackItem}
                progress={entry}
                isFavorite={Boolean(
                  favoriteIds?.has(displayItem.id) || entry.favorite,
                )}
                onFavoriteToggle={onFavoriteToggle}
              />
            )
          })}
        </MediaRail>
      </div>
    </section>
  )
}

function LibraryCollageCard({
  library,
  posters,
}: {
  library: Library
  posters: string[]
}) {
  const shouldUseCollage = posters.length >= 4
  const primaryPoster = posters[0]

  return (
    <Link
      to="/library/$id"
      params={{ id: String(library.id) }}
      className="group overflow-hidden rounded-[1.75rem] border border-border/40 bg-card/70 shadow-lg transition-transform hover:-translate-y-1 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
    >
      <div className="aspect-[16/10] bg-muted">
        {shouldUseCollage ? (
          <div className="grid h-full grid-cols-4 gap-1 p-1">
            {posters.slice(0, 4).map((poster, index) => (
              <div
                key={`${poster}-${index}`}
                className="h-full rounded-xl bg-cover bg-center"
                style={{ backgroundImage: `url(${poster})` }}
              />
            ))}
          </div>
        ) : primaryPoster ? (
          <div
            className="h-full bg-cover bg-center"
            style={{ backgroundImage: `url(${primaryPoster})` }}
          />
        ) : (
          <div className="flex h-full items-center justify-center bg-card/80 px-4 text-center text-sm text-muted-foreground">
            暂无封面
          </div>
        )}
      </div>
      <div className="flex items-center justify-between gap-3 px-4 py-4">
        <div className="min-w-0">
          <div className="truncate text-lg font-semibold tracking-tight">
            {library.name}
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {library.type || '媒体库'} · {library.status}
          </div>
        </div>
        <ArrowUpRightIcon className="size-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
      </div>
    </Link>
  )
}

export function StatCard({
  icon: Icon,
  label,
  value,
  description,
  compact = false,
}: {
  icon: ComponentType<{ className?: string }>
  label: string
  value: number
  description: string
  compact?: boolean
}) {
  return (
    <div
      className={cn(
        'rounded-[1.75rem] border border-border/40 bg-card/75 p-4 backdrop-blur-md',
        compact ? 'min-w-0' : '',
      )}
    >
      <div className="flex items-center gap-2 text-xs tracking-[0.2em] text-muted-foreground uppercase">
        <Icon className="size-3.5" />
        {label}
      </div>
      <div className="mt-3 text-3xl font-semibold">{value}</div>
      <div className="mt-1 text-sm leading-6 text-muted-foreground">
        {description}
      </div>
    </div>
  )
}

export function formatMediaType(type: string) {
  if (type === 'movie') return '电影'
  if (type === 'show' || type === 'episode') return '剧集'
  return '媒体'
}
