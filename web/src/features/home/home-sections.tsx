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
}: {
  latestLibrarySections: any[]
}) {
  return (
    <section className="relative border-t border-border/40 bg-background px-4 pb-16 pt-10 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-[1600px]">
        {latestLibrarySections.length > 0 ? (
          <div className="space-y-8">
            {latestLibrarySections.map((section) => (
              <div key={section.library_id}>
                <Link
                  to="/library/$id"
                  params={{ id: String(section.library_id) }}
                  className="group inline-flex items-center gap-2 text-lg font-medium tracking-tight text-foreground transition-colors hover:text-primary sm:text-xl"
                >
                  <span className="underline-offset-4 group-hover:underline">{`最新${section.library_name}`}</span>
                  <ArrowUpRightIcon className="size-4 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
                </Link>
                <div className="mt-4 overflow-x-auto pb-2">
                  <div className="flex min-w-max gap-5">
                    {section.items.map((item: any) => (
                      <article key={item.id} className="w-[212px] shrink-0">
                        <Link
                          to="/media/$id"
                          params={{ id: String(item.id) }}
                          search={{
                            view:
                              getMediaCardType(item) === 'show'
                                ? 'series'
                                : undefined,
                          }}
                          className="block overflow-hidden rounded-[1.75rem] border border-border/40 bg-card/70 shadow-lg transition-transform hover:-translate-y-1"
                        >
                          <div
                            className="aspect-[3/4] bg-cover bg-center bg-muted"
                            style={{
                              backgroundImage: getMediaCardPosterUrl(item)
                                ? `url(${getMediaCardPosterUrl(item)})`
                                : 'linear-gradient(180deg, rgba(80,92,255,0.35), rgba(15,118,110,0.35))',
                            }}
                          />
                          <div className="px-1 pb-1 pt-4">
                            <div className="line-clamp-1 text-[2rem] font-semibold tracking-tight text-foreground">
                              {formatMediaCardTitle(item)}
                            </div>
                            <div className="mt-1 text-2xl text-muted-foreground">
                              {item.year || '未知年份'}
                            </div>
                            <div className="mt-4 flex flex-wrap gap-2">
                              <Badge
                                className="rounded-full border-border/50 bg-background/80 px-3 py-1"
                                variant="outline"
                              >
                                {formatMediaType(getMediaCardType(item))}
                              </Badge>
                              <Badge
                                className="rounded-full border-border/50 bg-background/80 px-3 py-1"
                                variant="outline"
                              >
                                {section.library_name}
                              </Badge>
                            </div>
                          </div>
                        </Link>
                      </article>
                    ))}
                  </div>
                </div>
              </div>
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
