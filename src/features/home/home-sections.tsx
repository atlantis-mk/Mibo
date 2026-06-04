import { Link } from '@tanstack/react-router'
import { InfoIcon, PlayIcon, ShieldAlertIcon } from 'lucide-react'
import { toast } from 'sonner'
import 'swiper/css'
import {
  Autoplay as SwiperAutoplay,
  FreeMode as SwiperFreeMode,
} from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/react'
import {
  formatMediaCardTitle,
  getMediaCardBackdropUrl,
  getMediaCardType,
  getMediaCardPosterUrl,
} from '@/lib/media-presentation'
import type {
  CatalogListItem,
  CatalogUserItemEntry,
  HomeContentSection,
  OperationsTask,
} from '@/lib/mibo-api'
import {
  affectedLibraryNames,
  operationTaskMessage,
  operationTaskTitle,
} from '@/lib/operations-presentation'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { MediaPosterCard } from '@/components/media-poster-card'

export function HeroCarousel({
  heroItems,
  canAutoplayHeroItems,
  canLoopHeroItems,
}: {
  heroItems: CatalogListItem[]
  canAutoplayHeroItems: boolean
  canLoopHeroItems: boolean
}) {
  if (heroItems.length === 0) {
    return (
      <section
        className='relative min-h-[55svh] overflow-hidden lg:min-h-[75svh]'
        style={{
          background:
            'linear-gradient(135deg, rgba(5,10,18,1), rgba(30,41,59,0.88), rgba(15,118,110,0.66))',
        }}
      >
        <div className='absolute inset-0 bg-linear-to-r from-background via-background/15 to-background/95' />
        <div className='absolute inset-0 bg-linear-to-t from-background/95 via-background/20 to-background/10' />
        <div
          className={cn(
            'relative flex min-h-[55svh] items-end px-6 pt-10 sm:px-8 lg:min-h-[75svh] lg:px-12',
            'pb-8 lg:pb-10'
          )}
        >
          <div className='max-w-4xl min-w-0'>
            <Badge
              className='border-border/50 bg-background/75 backdrop-blur-sm'
              variant='outline'
            >
              首页已就绪
            </Badge>
            <h1 className='mt-5 max-w-3xl text-4xl font-semibold tracking-tight text-balance sm:text-5xl lg:text-6xl'>
              等待扫描后的最近加入内容
            </h1>
            <p className='mt-4 max-w-2xl text-sm leading-7 text-muted-foreground sm:text-base'>
              添加媒体源并完成扫描后，最近加入的影片或剧集会自动切换为首页轮播。
            </p>
          </div>
        </div>
      </section>
    )
  }

  return (
    <Swiper
      modules={canAutoplayHeroItems ? [SwiperAutoplay] : undefined}
      loop={canLoopHeroItems}
      slidesPerView={1}
      autoplay={
        canAutoplayHeroItems
          ? {
              delay: 5000,
              disableOnInteraction: false,
              pauseOnMouseEnter: true,
            }
          : false
      }
      className='w-full'
    >
      {heroItems.map((item) => {
        return (
          <SwiperSlide key={item.id}>
            <HeroSlide item={item} />
          </SwiperSlide>
        )
      })}
    </Swiper>
  )
}

function HeroSlide({ item }: { item: CatalogListItem }) {
  const backgroundImage =
    getMediaCardBackdropUrl(item) || getMediaCardPosterUrl(item)
  const displayTitle = formatMediaCardTitle(item)
  const overview = item.overview?.trim() || '暂无媒体描述。'

  return (
    <section
      className='relative min-h-[55svh] overflow-hidden lg:min-h-[75svh]'
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
            className='absolute inset-0 h-full w-full object-cover'
          />
          <div className='absolute inset-0 bg-linear-to-r from-background via-background/15 to-background/95' />
          <div className='absolute inset-0 bg-linear-to-t from-background/95 via-background/20 to-background/10' />
        </>
      ) : null}

      <div
        className={cn(
          'relative flex min-h-[55svh] items-end px-6 pt-10 sm:px-8 lg:min-h-[75svh] lg:px-12',
          'pb-8 lg:pb-10'
        )}
      >
        <div className='max-w-4xl min-w-0'>
          <h1 className='max-w-3xl text-4xl font-semibold tracking-tight text-balance sm:text-5xl lg:text-6xl'>
            {displayTitle}
          </h1>
          <p className='mt-4 line-clamp-3 max-w-2xl text-sm leading-7 text-muted-foreground sm:text-base'>
            {overview}
          </p>
          <div className='mt-6 flex flex-wrap gap-3'>
            <Link
              to='/play/$id'
              params={{ id: String(item.id) }}
              search={{
                fromStart: false,
                inventoryFileId: undefined,
                resourceId: undefined,
                liveChannelId: undefined,
                liveSourceId: undefined,
              }}
            >
              <Badge className='cursor-pointer rounded-full px-4 py-2 text-sm'>
                <PlayIcon className='size-4' />
                播放
              </Badge>
            </Link>
            <Link
              to='/media/$id'
              params={{ id: String(item.id) }}
              search={{
                view: getMediaCardType(item) === 'show' ? 'series' : undefined,
                episodePage: undefined,
              }}
            >
              <Badge
                className='cursor-pointer rounded-full border-border/50 bg-background/75 px-4 py-2 text-sm hover:bg-accent hover:text-accent-foreground'
                variant='outline'
              >
                <InfoIcon className='size-4' />
                详情
              </Badge>
            </Link>
          </div>
        </div>
      </div>
    </section>
  )
}

export function HomeHealthToastContent({ task }: { task: OperationsTask }) {
  return (
    <div className='flex items-start gap-3 rounded-[1.5rem] border bg-background px-4 py-4'>
      <div className='mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-full'>
        <ShieldAlertIcon className='size-4' />
      </div>
      <div className='min-w-0'>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge variant='default'>首页可用，存在降级</Badge>
          <span className='text-sm font-medium'>
            {operationTaskTitle(task)}
          </span>
        </div>
        <p className='mt-2 text-sm text-muted-foreground'>
          {operationTaskMessage(task)}
        </p>
        {affectedLibraryNames(task) ? (
          <p className='mt-1 text-xs text-muted-foreground'>
            受影响来源：{affectedLibraryNames(task)}
          </p>
        ) : null}
        <Button asChild className='mt-3' size='sm' variant='outline'>
          <Link
            to='/settings/operations'
            onClick={() => {
              toast.dismiss('home-health-degraded')
            }}
          >
            前往运营中心
          </Link>
        </Button>
      </div>
    </div>
  )
}

export function ContentSectionRail({
  contentSections,
}: {
  contentSections: HomeContentSection[]
}) {
  if (contentSections.length === 0) {
    return (
      <section className='relative flex min-h-[calc(100svh-18rem)] items-center justify-center border-t border-border/40 bg-background px-4 py-16 sm:px-6 lg:px-8'>
        <div className='rounded-[2rem] border border-border/40 bg-card/70 px-6 py-8 text-center text-sm text-muted-foreground backdrop-blur-sm'>
          还没有可展示的最新内容，稍后会按电影、剧集等内容形态自动补充到这里。
        </div>
      </section>
    )
  }

  return (
    <section className='relative bg-background px-4 pt-10 pb-16 sm:px-6 lg:px-8'>
      <div className='space-y-8'>
        {contentSections.map((section) => (
          <section key={section.key}>
            <div className='mb-4 flex items-center justify-between gap-3'>
              {getLibrarySectionType(section.key) ? (
                <Link
                  to='/library'
                  search={(previous) => ({
                    q: previous.q,
                    genre: previous.genre,
                    region: previous.region,
                    year: previous.year,
                    minRating: previous.minRating,
                    watchedState: previous.watchedState,
                    organizingState: previous.organizingState,
                    sort: previous.sort,
                    sortDirection: previous.sortDirection,
                    page: 1,
                    pageSize: previous.pageSize,
                    type: getLibrarySectionType(section.key),
                  })}
                  className='text-xl font-semibold tracking-tight text-foreground transition-colors hover:text-primary'
                >
                  最新{section.title}
                </Link>
              ) : (
                <h2 className='text-xl font-semibold tracking-tight text-foreground'>
                  最新{section.title}
                </h2>
              )}
            </div>
            <div className='-mx-4 sm:-mx-6 lg:-mx-8'>
              <Swiper
                modules={[SwiperFreeMode]}
                slidesPerView='auto'
                spaceBetween={16}
                slidesOffsetBefore={16}
                breakpoints={{
                  640: { slidesOffsetBefore: 24 },
                  1024: { slidesOffsetBefore: 32 },
                }}
                freeMode
                className='!overflow-x-clip !overflow-y-visible pt-1 pb-3'
              >
                {section.items.map((item) => (
                  <SwiperSlide
                    key={`${section.key}-${item.id}`}
                    className='!w-auto'
                  >
                    <MediaPosterCard item={item} />
                  </SwiperSlide>
                ))}
              </Swiper>
            </div>
          </section>
        ))}
      </div>
    </section>
  )
}

function getLibrarySectionType(sectionKey: string) {
  if (sectionKey === 'movies') {
    return 'movie' as const
  }

  if (sectionKey === 'series') {
    return 'show' as const
  }

  return undefined
}

export function ContinueWatchingRail({
  entries,
}: {
  entries: CatalogUserItemEntry[]
}) {
  if (entries.length === 0) {
    return null
  }

  return (
    <section className='border-t border-border/40 bg-background px-4 py-10 sm:px-6 lg:px-8'>
      <div>
        <section>
          <div className='mb-4 flex items-center justify-between gap-3'>
            <h2 className='text-xl font-semibold tracking-tight text-foreground'>
              继续观看
            </h2>
          </div>
          <Swiper
            modules={[SwiperFreeMode]}
            slidesPerView='auto'
            spaceBetween={16}
            freeMode
            className='!overflow-x-clip !overflow-y-visible pt-1 pb-3'
          >
            {entries.map((entry) => {
              const displayItem = entry.display_item ?? entry.item
              const playbackItem = entry.play_item ?? entry.item
              const { progressMeta, progressDescription } =
                formatContinueWatchingProgress(playbackItem)

              return (
                <SwiperSlide
                  key={`${entry.item.metadata_item_id}-${entry.resource_id ?? 'default'}`}
                  className='!w-auto'
                >
                  <MediaPosterCard
                    item={displayItem}
                    playbackItem={playbackItem}
                    progress={entry}
                    progressMeta={progressMeta}
                    progressDescription={progressDescription}
                    imageAspect='landscape'
                    className='w-[280px] sm:w-[360px]'
                  />
                </SwiperSlide>
              )
            })}
          </Swiper>
        </section>
      </div>
    </section>
  )
}

function formatContinueWatchingProgress(
  playbackItem: CatalogUserItemEntry['item']
) {
  if (playbackItem.type !== 'episode') {
    return { progressMeta: '', progressDescription: '' }
  }

  const episodeLabel = formatEpisodeProgressLabel(playbackItem)
  const episodeTitle = playbackItem.title?.trim()
  const progressMeta = [episodeLabel, episodeTitle].filter(Boolean).join(' - ')

  return { progressMeta, progressDescription: '' }
}

function formatEpisodeProgressLabel(item: CatalogUserItemEntry['item']) {
  const label = item.episode_label?.trim()
  if (label) {
    return label
  }

  const seasonNumber = item.parent_index_number
  const episodeNumber = item.index_number
  const episodeNumberEnd = item.index_number_end

  if (typeof seasonNumber !== 'number' && typeof episodeNumber !== 'number') {
    return ''
  }

  const season = typeof seasonNumber === 'number' ? `S${seasonNumber}` : ''
  const episode =
    typeof episodeNumber === 'number'
      ? `E${episodeNumber}${
          typeof episodeNumberEnd === 'number' &&
          episodeNumberEnd !== episodeNumber
            ? `-E${episodeNumberEnd}`
            : ''
        }`
      : ''

  return [season, episode].filter(Boolean).join(':')
}
