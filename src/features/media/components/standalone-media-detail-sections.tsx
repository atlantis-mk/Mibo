import { useMemo, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { MediaPosterCard } from '#/components/media-poster-card'
import { Button } from '#/components/ui/button'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '#/components/ui/pagination'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '#/components/ui/tooltip'
import type {
  CatalogEpisodeRail,
  CatalogSeasonRail,
  MediaDetailPresentation,
} from '#/lib/media-presentation'
import { cn } from '#/lib/utils'
import { ChevronLeft, ChevronRight, Film } from 'lucide-react'
import { FreeMode } from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/react'
import type { Swiper as SwiperType } from 'swiper/types'
import { formatRuntime } from './standalone-media-detail-utils'

export { DetailHeroSection } from './standalone-media-detail-hero'
export { PeopleSection, SpecsSection } from './standalone-media-detail-specs'

const EPISODES_PER_PAGE = 60

export function SeriesEpisodesSection({
  item,
  seasons,
  episodePage,
  isLoading,
  errorMessage,
  selectedEpisodeMetadataItemId,
  onSelectEpisode,
}: {
  item: MediaDetailPresentation
  seasons: CatalogSeasonRail[]
  episodePage: number
  isLoading: boolean
  errorMessage: string | null
  selectedEpisodeMetadataItemId?: number
  onSelectEpisode: (episodeId: number) => void
}) {
  const navigate = useNavigate({ from: '/media/$id' })
  const visibleSeasons = useMemo(
    () => seasons.filter((season) => season.episodes.length > 0),
    [seasons]
  )
  const numberedSeasons = useMemo(
    () => visibleSeasons.filter((season) => !isSpecialSeason(season)),
    [visibleSeasons]
  )
  const specialsSeasons = useMemo(
    () => visibleSeasons.filter((season) => isSpecialSeason(season)),
    [visibleSeasons]
  )
  const [selectedSeasonNumber, setSelectedSeasonNumber] = useState<
    number | undefined
  >()
  const selectedSeason =
    numberedSeasons.find(
      (season) => season.season_number === selectedSeasonNumber
    ) ?? numberedSeasons[0]
  const selectSeason = (seasonNumber: number) => {
    setSelectedSeasonNumber(seasonNumber)
    void navigate({
      search: (previous) => ({
        ...previous,
        episodePage: 1,
      }),
    })
  }

  if (item.type !== 'series' && item.type !== 'episode') {
    return null
  }

  return (
    <section className='mt-12 space-y-6'>
      {isLoading ? (
        <div className='rounded-[24px] border border-border/40 bg-card/70 px-5 py-4 text-sm text-muted-foreground backdrop-blur-md'>
          正在加载剧集信息
        </div>
      ) : null}

      {!isLoading && errorMessage ? (
        <div className='rounded-[24px] border border-border/40 bg-card/70 px-5 py-4 text-sm text-muted-foreground backdrop-blur-md'>
          剧集信息加载失败：{errorMessage}
        </div>
      ) : null}

      {!isLoading && !errorMessage && visibleSeasons.length === 0 ? (
        <div className='rounded-[24px] border border-border/40 bg-card/70 px-5 py-4 text-sm text-muted-foreground backdrop-blur-md'>
          当前剧集暂时没有可展示的分季信息。
        </div>
      ) : null}

      {!isLoading && !errorMessage && visibleSeasons.length > 0 ? (
        <div className='space-y-8'>
          {numberedSeasons.length > 0 ? (
            <div className='space-y-3'>
              <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-3'>
                {numberedSeasons.map((season) => (
                  <SeasonSelectorCard
                    key={`card-${season.season_number}`}
                    season={season}
                    selected={
                      selectedSeason?.season_number === season.season_number
                    }
                    onSelect={() => selectSeason(season.season_number)}
                  />
                ))}
              </div>
            </div>
          ) : null}
          {selectedSeason ? (
            <SeasonEpisodesRail
              season={selectedSeason}
              page={episodePage}
              selectedEpisodeMetadataItemId={selectedEpisodeMetadataItemId}
              onSelectEpisode={onSelectEpisode}
            />
          ) : null}
          {specialsSeasons.map((season) => (
            <SeasonEpisodesRail
              key={`special-${season.season_number}-${season.name}`}
              season={season}
              page={episodePage}
              selectedEpisodeMetadataItemId={selectedEpisodeMetadataItemId}
              onSelectEpisode={onSelectEpisode}
            />
          ))}
        </div>
      ) : null}
    </section>
  )
}

function SeasonEpisodesRail({
  season,
  page,
  selectedEpisodeMetadataItemId,
  onSelectEpisode,
}: {
  season: CatalogSeasonRail
  page: number
  selectedEpisodeMetadataItemId?: number
  onSelectEpisode: (episodeId: number) => void
}) {
  const totalPages = Math.max(
    1,
    Math.ceil(season.episodes.length / EPISODES_PER_PAGE)
  )
  const currentPage = Math.min(Math.max(1, page), totalPages)
  const pageStart = (currentPage - 1) * EPISODES_PER_PAGE
  const visibleEpisodes = season.episodes.slice(
    pageStart,
    pageStart + EPISODES_PER_PAGE
  )

  return (
    <div className='space-y-4'>
      <div className='grid gap-5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4'>
        {visibleEpisodes.map((episode) => (
          <EpisodeCard
            key={`${season.season_number}-${episode.episode_number}-${episode.metadata_item_id}`}
            seasonNumber={season.season_number}
            episode={episode}
            selected={
              selectedEpisodeMetadataItemId === episode.metadata_item_id
            }
            onSelect={() => onSelectEpisode(episode.metadata_item_id)}
          />
        ))}
      </div>
      <EpisodePagination currentPage={currentPage} totalPages={totalPages} />
    </div>
  )
}

function EpisodePagination({
  currentPage,
  totalPages,
}: {
  currentPage: number
  totalPages: number
}) {
  if (totalPages <= 1) return null

  return (
    <Pagination className='justify-end'>
      <PaginationContent>
        <PaginationItem>
          <PaginationPrevious
            text='上一页'
            href={buildEpisodePageHref(currentPage - 1)}
            aria-disabled={currentPage === 1}
            className={
              currentPage === 1 ? 'pointer-events-none opacity-50' : ''
            }
          />
        </PaginationItem>
        {buildEpisodePageItems(currentPage, totalPages).map((page) => (
          <PaginationItem key={page}>
            <PaginationLink
              href={buildEpisodePageHref(page)}
              isActive={page === currentPage}
            >
              {page}
            </PaginationLink>
          </PaginationItem>
        ))}
        <PaginationItem>
          <PaginationNext
            text='下一页'
            href={buildEpisodePageHref(currentPage + 1)}
            aria-disabled={currentPage === totalPages}
            className={
              currentPage === totalPages ? 'pointer-events-none opacity-50' : ''
            }
          />
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  )
}

function buildEpisodePageItems(currentPage: number, totalPages: number) {
  const start = Math.max(1, Math.min(currentPage - 2, totalPages - 4))
  const end = Math.min(totalPages, start + 4)
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
}

function buildEpisodePageHref(page: number) {
  const search = new URLSearchParams(window.location.search)
  if (page <= 1) {
    search.delete('episodePage')
  } else {
    search.set('episodePage', String(page))
  }
  const query = search.toString()
  return `${window.location.pathname}${query ? `?${query}` : ''}`
}

function SeasonSelectorCard({
  season,
  selected,
  onSelect,
}: {
  season: CatalogSeasonRail
  selected: boolean
  onSelect: () => void
}) {
  const seasonName = season.name?.trim() || `第 ${season.season_number} 季`
  const episodeCountLabel = `共 ${season.episodes.length} 集`
  const seasonMeta = [season.runtime_seconds ? formatRuntime(season.runtime_seconds) : '']
    .filter(Boolean)
    .join(' · ')

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type='button'
          className={cn(
            'grid min-h-36 grid-cols-[88px_minmax(0,1fr)] overflow-hidden rounded-[26px] border border-border/50 bg-card/80 text-left shadow-xs transition hover:border-primary/40 hover:bg-card focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none sm:min-h-40 sm:grid-cols-[104px_minmax(0,1fr)]',
            selected && 'border-primary bg-primary/8 ring-1 ring-primary/35'
          )}
          onClick={onSelect}
          aria-pressed={selected}
        >
          <div className='relative h-full overflow-hidden bg-muted/40'>
            {season.poster_url ? (
              <img
                src={season.poster_url}
                alt={seasonName}
                className='absolute inset-0 h-full w-full object-cover'
              />
            ) : (
              <div className='absolute inset-0 bg-linear-to-br from-muted via-muted/80 to-muted/40' />
            )}
            <div className='absolute inset-0 bg-linear-to-t from-background/35 via-transparent to-transparent' />
          </div>
          <div className='flex min-w-0 flex-col justify-between gap-4 px-4 py-3.5 sm:px-5 sm:py-4'>
            <div className='space-y-1.5'>
              <div className='text-[11px] font-medium tracking-[0.18em] text-primary/90 uppercase'>
                {season.season_number > 0
                  ? `Season ${String(season.season_number).padStart(2, '0')}`
                  : 'Specials'}
              </div>
              <div className='flex flex-wrap items-baseline gap-x-2 gap-y-1'>
                <div className='line-clamp-2 text-[17px] leading-6 font-semibold text-foreground'>
                  {seasonName}
                </div>
                <div className='text-sm text-muted-foreground'>
                  {episodeCountLabel}
                </div>
              </div>
              {seasonMeta ? (
                <div className='text-sm text-muted-foreground'>{seasonMeta}</div>
              ) : null}
            </div>
            {season.overview ? (
              <div className='line-clamp-3 text-sm leading-6 text-muted-foreground'>
                {season.overview}
              </div>
            ) : null}
          </div>
        </button>
      </TooltipTrigger>
      <TooltipContent
        side='top'
        sideOffset={8}
        className='max-w-sm rounded-2xl border border-border/60 bg-popover px-4 py-3 text-sm text-popover-foreground shadow-xl'
      >
        <div className='space-y-2'>
          <div className='text-[11px] font-medium tracking-[0.18em] text-primary uppercase'>
            {season.season_number > 0
              ? `Season ${String(season.season_number).padStart(2, '0')}`
              : 'Specials'}
          </div>
          <div className='flex flex-wrap items-baseline gap-x-2 gap-y-1'>
            <div className='font-semibold text-foreground'>{seasonName}</div>
            <div className='text-muted-foreground'>{episodeCountLabel}</div>
          </div>
          {seasonMeta ? (
            <div className='text-muted-foreground'>{seasonMeta}</div>
          ) : null}
          {season.overview ? (
            <div className='leading-6 text-muted-foreground'>
              {season.overview}
            </div>
          ) : null}
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

function EpisodeCard({
  seasonNumber,
  episode,
  selected,
  onSelect,
}: {
  seasonNumber: number
  episode: CatalogEpisodeRail
  selected: boolean
  onSelect: () => void
}) {
  const code = `S${String(Math.max(0, seasonNumber)).padStart(2, '0')}E${String(Math.max(0, episode.episode_number)).padStart(2, '0')}`
  const episodeName = episode.name?.trim()
  const title = episodeName ? `${code}-${episodeName}` : code

  return (
    <button
      type='button'
      onClick={onSelect}
      aria-pressed={selected}
      className={cn(
        'block w-full rounded-2xl border border-border/50 bg-card/75 px-4 py-3 text-left text-sm font-medium text-foreground transition hover:border-primary/40 hover:bg-card hover:text-primary focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none',
        selected &&
          'border-primary bg-primary/8 text-primary ring-1 ring-primary/35'
      )}
    >
      <span className='line-clamp-2'>{title}</span>
    </button>
  )
}

export function RelatedMediaSection({
  item,
}: {
  item: MediaDetailPresentation
}) {
  const [swiper, setSwiper] = useState<SwiperType | null>(null)
  const [canScrollPrev, setCanScrollPrev] = useState(false)
  const [canScrollNext, setCanScrollNext] = useState(false)
  const relatedItems = item.related_items

  const updateNavigation = (instance: SwiperType) => {
    setCanScrollPrev(!instance.isBeginning)
    setCanScrollNext(!instance.isEnd)
  }

  if (relatedItems.length === 0) {
    return null
  }

  return (
    <section className='mt-12 space-y-6'>
      <div className='flex flex-wrap items-end justify-between gap-3'>
        <div className='space-y-2'>
          <div className='flex items-center gap-2 text-[19px] font-semibold text-foreground'>
            <Film className='size-4 text-muted-foreground' />
            相似推荐
          </div>
          <p className='text-sm text-muted-foreground'>
            基于同媒体库和标签生成的相关内容。
          </p>
        </div>
        <div className='hidden items-center gap-2 sm:flex'>
          <RailArrowButton
            direction='prev'
            disabled={!canScrollPrev}
            onClick={() => swiper?.slidePrev()}
          />
          <RailArrowButton
            direction='next'
            disabled={!canScrollNext}
            onClick={() => swiper?.slideNext()}
          />
        </div>
      </div>
      <div className='relative left-1/2 w-screen -translate-x-1/2'>
        <Swiper
          modules={[FreeMode]}
          freeMode
          slidesPerView='auto'
          spaceBetween={20}
          slidesOffsetBefore={40}
          onSwiper={(instance) => {
            setSwiper(instance)
            updateNavigation(instance)
          }}
          onSlideChange={updateNavigation}
          onResize={updateNavigation}
          className='!overflow-x-clip !overflow-y-visible pt-1 pb-3'
        >
          {relatedItems.map((relatedItem) => (
            <SwiperSlide key={relatedItem.id} className='!h-auto !w-auto'>
              <MediaPosterCard item={relatedItem} />
            </SwiperSlide>
          ))}
        </Swiper>
      </div>
    </section>
  )
}

function isSpecialSeason(season: CatalogSeasonRail) {
  const name = season.name.trim().toLowerCase()
  return (
    season.season_number === 0 ||
    name.includes('special') ||
    name.includes('特别') ||
    name.includes('番外')
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
      type='button'
      size='icon-sm'
      variant='outline'
      className='rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground'
      onClick={onClick}
      disabled={disabled}
    >
      {direction === 'prev' ? (
        <ChevronLeft className='size-4' />
      ) : (
        <ChevronRight className='size-4' />
      )}
      <span className='sr-only'>
        {direction === 'prev' ? '上一组剧集' : '下一组剧集'}
      </span>
    </Button>
  )
}
