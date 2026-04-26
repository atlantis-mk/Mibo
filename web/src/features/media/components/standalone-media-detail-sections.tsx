import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { ChevronLeft, ChevronRight, Disc3 } from 'lucide-react'
import { FreeMode } from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/react'
import type { Swiper as SwiperType } from 'swiper/types'

import { Button } from '#/components/ui/button'
import type {
  CatalogDetailPresentation,
  CatalogEpisodeRail,
  CatalogSeasonRail,
} from '#/lib/media-presentation'
import { cn } from '#/lib/utils'

export { DetailHeroSection } from './standalone-media-detail-hero'
export { SpecsSection } from './standalone-media-detail-specs'

import {
  formatAvailabilityStatus,
  formatRuntime,
} from './standalone-media-detail-utils'

export function SeriesEpisodesSection({
  item,
  seasons,
  isLoading,
  errorMessage,
}: {
  item: CatalogDetailPresentation
  seasons: CatalogSeasonRail[]
  isLoading: boolean
  errorMessage: string | null
}) {
  if (item.type !== 'series' && item.type !== 'episode') {
    return null
  }

  return (
    <section className="mt-12 space-y-6">
      <div className="space-y-2">
        <div className="flex items-center gap-2 text-[19px] font-semibold text-foreground">
          <Disc3 className="size-4 text-muted-foreground" />
          剧集信息
        </div>
        <p className="text-sm text-muted-foreground">
          {item.series_title_display || item.title}{' '}
          的剧集按季展示，可左右滑动浏览。
        </p>
      </div>

      {isLoading ? (
        <div className="rounded-[24px] border border-border/40 bg-card/70 px-5 py-4 text-sm text-muted-foreground backdrop-blur-md">
          正在加载剧集信息
        </div>
      ) : null}

      {!isLoading && errorMessage ? (
        <div className="rounded-[24px] border border-border/40 bg-card/70 px-5 py-4 text-sm text-muted-foreground backdrop-blur-md">
          剧集信息加载失败：{errorMessage}
        </div>
      ) : null}

      {!isLoading && !errorMessage && seasons.length === 0 ? (
        <div className="rounded-[24px] border border-border/40 bg-card/70 px-5 py-4 text-sm text-muted-foreground backdrop-blur-md">
          当前剧集暂时没有可展示的分季信息。
        </div>
      ) : null}

      {!isLoading && !errorMessage && seasons.length > 0 ? (
        <div className="space-y-8">
          {seasons.map((season) => (
            <SeasonEpisodesRail key={season.season_number} season={season} />
          ))}
        </div>
      ) : null}
    </section>
  )
}

function SeasonEpisodesRail({ season }: { season: CatalogSeasonRail }) {
  const [swiper, setSwiper] = useState<SwiperType | null>(null)
  const [canScrollPrev, setCanScrollPrev] = useState(false)
  const [canScrollNext, setCanScrollNext] = useState(false)

  const updateNavigation = (instance: SwiperType) => {
    setCanScrollPrev(!instance.isBeginning)
    setCanScrollNext(!instance.isEnd)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-1">
          <h3 className="text-[28px] font-semibold tracking-tight text-foreground">
            {season.name?.trim() || `第 ${season.season_number} 季`}
          </h3>
          <div className="text-sm text-muted-foreground">
            共 {season.episodes.length} 集
            {season.runtime_seconds
              ? ` · ${formatRuntime(season.runtime_seconds)}`
              : ''}
          </div>
        </div>
        <div className="hidden items-center gap-2 sm:flex">
          <RailArrowButton
            direction="prev"
            disabled={!canScrollPrev}
            onClick={() => swiper?.slidePrev()}
          />
          <RailArrowButton
            direction="next"
            disabled={!canScrollNext}
            onClick={() => swiper?.slideNext()}
          />
        </div>
      </div>

      <div className="relative px-0 sm:px-12">
        <Swiper
          modules={[FreeMode]}
          freeMode
          slidesPerView="auto"
          spaceBetween={20}
          onSwiper={(instance) => {
            setSwiper(instance)
            updateNavigation(instance)
          }}
          onSlideChange={updateNavigation}
          onResize={updateNavigation}
          className="w-full"
        >
          {season.episodes.map((episode) => (
            <SwiperSlide
              key={`${season.season_number}-${episode.episode_number}`}
              className="!h-auto !w-[290px] sm:!w-[360px] lg:!w-[392px]"
            >
              <EpisodeCard
                episode={episode}
                fallbackImage={season.poster_url}
              />
            </SwiperSlide>
          ))}
        </Swiper>

        <div className="mt-4 flex items-center justify-end gap-2 sm:hidden">
          <RailArrowButton
            direction="prev"
            disabled={!canScrollPrev}
            onClick={() => swiper?.slidePrev()}
          />
          <RailArrowButton
            direction="next"
            disabled={!canScrollNext}
            onClick={() => swiper?.slideNext()}
          />
        </div>
      </div>
    </div>
  )
}

function EpisodeCard({
  episode,
  fallbackImage,
}: {
  episode: CatalogEpisodeRail
  fallbackImage: string
}) {
  const title = episode.name?.trim() || `第 ${episode.episode_number} 集`
  const cardContent = (
    <div
      className={cn(
        'group overflow-hidden rounded-[16px] border border-border/40 bg-card/70 shadow-lg backdrop-blur-md transition',
        episode.item_id
          ? 'hover:border-border/70 hover:bg-card/85'
          : 'opacity-90',
      )}
    >
      <div className="relative aspect-video overflow-hidden bg-muted">
        {episode.still_url || fallbackImage ? (
          <img
            src={episode.still_url || fallbackImage}
            alt={title}
            className="h-full w-full object-cover transition duration-300 group-hover:scale-[1.03]"
          />
        ) : null}
        <div className="absolute inset-0 bg-gradient-to-t from-background/90 via-background/15 to-transparent" />
      </div>
      <div className="space-y-2 p-4">
        <div className="line-clamp-1 text-lg text-foreground">
          {episode.episode_number}. {title}
        </div>
        <div className="text-sm text-muted-foreground">
          {[
            formatEpisodeAirDate(episode.air_date),
            formatRuntime(episode.runtime_seconds),
          ]
            .filter(Boolean)
            .join('  ')}
        </div>
        <div className="text-xs text-muted-foreground">
          {formatAvailabilityStatus(episode.availability_status)}
        </div>
        <p className="line-clamp-3 text-sm leading-6 text-muted-foreground">
          {episode.overview || '暂无剧情简介'}
        </p>
      </div>
    </div>
  )

  if (!episode.item_id) {
    return cardContent
  }

  return (
    <Link
      to="/media/$id"
      params={{ id: String(episode.item_id) }}
      search={{ view: undefined }}
    >
      {cardContent}
    </Link>
  )
}

function RailArrowButton({
  direction,
  disabled,
  onClick,
}: {
  direction: 'prev' | 'next'
  disabled: boolean
  onClick: () => void
}) {
  return (
    <Button
      type="button"
      size="icon-sm"
      variant="outline"
      className="rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
      onClick={onClick}
      disabled={disabled}
    >
      {direction === 'prev' ? (
        <ChevronLeft className="size-4" />
      ) : (
        <ChevronRight className="size-4" />
      )}
      <span className="sr-only">
        {direction === 'prev' ? '上一组剧集' : '下一组剧集'}
      </span>
    </Button>
  )
}

function formatEpisodeAirDate(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
  }).format(date)
}
