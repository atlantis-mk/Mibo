import { useEffect, useState } from 'react'
import { Button } from '#/components/ui/button'
import type { ExternalPlayerId } from '#/features/play/external-player'
import type {
  CatalogEpisodeRail,
  MediaDetailPresentation,
  CatalogSeasonRail,
} from '#/lib/media-presentation'
import type {
  MediaResourceDetail,
  MetadataResourceDetail,
  ProgressState,
} from '#/lib/mibo-api'
import { ChevronLeftIcon } from 'lucide-react'
import 'swiper/css'
import 'swiper/css/free-mode'
import { Autoplay as SwiperAutoplay } from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/react'
import {
  DetailHeroSection,
  PeopleSection,
  RelatedMediaSection,
  SeriesEpisodesSection,
} from './standalone-media-detail-sections'
import { getPrimaryCatalogResource } from './standalone-media-detail-utils'

type StandaloneMediaDetailProps = {
  item: MediaDetailPresentation
  itemProgressPercent: number
  progress: ProgressState | null
  seriesSeasons: CatalogSeasonRail[]
  episodePage: number
  isSeriesEpisodesLoading: boolean
  seriesEpisodesErrorMessage: string | null
  onOpenPlaybackEntry: (options?: {
    itemId?: number
    fromStart?: boolean
    resourceId?: number
  }) => void
  onOpenExternalPlaybackEntry: (options?: {
    itemId?: number
    fromStart?: boolean
    resourceId?: number
    playerId?: ExternalPlayerId
  }) => void
  resourceChoices?: MetadataResourceDetail[]
  resourceSummaries?: MediaResourceDetail[]
  selectedResourceId?: number
  onSelectResource?: (resourceId: number) => void
  isSelectingResource: boolean
  selectedEpisodeMetadataItemId?: number
  onSelectEpisode: (episodeId: number) => void
  onReprobePrimaryFile: () => void
  isReprobePending: boolean
  onManageMetadata: () => void
  onMarkWatched: () => void
  isFavorite: boolean
  onFavoriteToggle: (favorite: boolean) => void
  onBack: () => void
}

type SelectedSeriesEpisode = CatalogEpisodeRail & {
  season_name: string
}

export function StandaloneMediaDetail({
  item,
  itemProgressPercent,
  progress,
  seriesSeasons,
  episodePage,
  isSeriesEpisodesLoading,
  seriesEpisodesErrorMessage,
  onOpenPlaybackEntry,
  onOpenExternalPlaybackEntry,
  resourceChoices = [],
  resourceSummaries = [],
  selectedResourceId,
  onSelectResource,
  isSelectingResource,
  selectedEpisodeMetadataItemId,
  onSelectEpisode,
  onReprobePrimaryFile,
  isReprobePending,
  onManageMetadata,
  onMarkWatched,
  isFavorite,
  onFavoriteToggle,
  onBack,
}: StandaloneMediaDetailProps) {
  const [overviewExpanded, setOverviewExpanded] = useState(false)

  useEffect(() => {
    setOverviewExpanded(false)
  }, [item.id])

  const selectedSeriesEpisode: SelectedSeriesEpisode | undefined =
    item.type === 'series'
      ? seriesSeasons
          .flatMap((season) =>
            season.episodes.map((episode) => ({
              ...episode,
              season_name: season.name,
            }))
          )
          .find(
            (episode) =>
              episode.metadata_item_id ===
              (selectedEpisodeMetadataItemId ??
                item.series_playback_target?.episode_metadata_item_id)
          )
      : undefined

  const primaryResource = getPrimaryCatalogResource(item)
  const primaryResourceFileIds = primaryResource?.file_ids ?? []
  const backdropSlides =
    item.backdrop_images.length > 0
      ? item.backdrop_images
      : item.backdrop_url
        ? [item.backdrop_url]
        : []
  return (
    <div className='relative min-h-svh w-full max-w-full overflow-hidden bg-background text-foreground'>
      <div className='h-svh overflow-x-hidden overflow-y-auto'>
        <div className='relative'>
          <div className='pointer-events-none absolute inset-x-0 top-0 z-30 mx-auto hidden w-full max-w-[1960px] px-6 pt-6 sm:px-8 md:block lg:px-10 lg:pt-8'>
            <Button
              type='button'
              variant='outline'
              size='icon'
              aria-label='返回上一页'
              className='pointer-events-auto rounded-full border-white/15 bg-background/70 text-foreground shadow-lg backdrop-blur-md hover:bg-background/85'
              onClick={onBack}
            >
              <ChevronLeftIcon className='size-5' />
            </Button>
          </div>

          {backdropSlides.length > 0 ? (
            <Swiper
              modules={backdropSlides.length > 1 ? [SwiperAutoplay] : undefined}
              loop={backdropSlides.length > 1}
              slidesPerView={1}
              autoplay={
                backdropSlides.length > 1
                  ? {
                      delay: 5000,
                      disableOnInteraction: false,
                      pauseOnMouseEnter: true,
                    }
                  : false
              }
              className='w-full'
            >
              {backdropSlides.map((imageUrl) => (
                <SwiperSlide key={imageUrl}>
                  <section className='relative h-[80svh] min-h-[80svh] overflow-hidden'>
                    <img
                      src={imageUrl}
                      alt={item.title}
                      className='absolute inset-0 h-full w-full object-cover'
                    />
                    <div className='absolute inset-y-0 left-0 w-[55%] bg-linear-to-r from-background via-background/55 to-transparent' />
                    <div className='absolute inset-0 bg-linear-to-t from-background/95 via-background/20 to-background/10' />
                  </section>
                </SwiperSlide>
              ))}
            </Swiper>
          ) : (
            <div
              className='relative h-[80svh] min-h-[80svh]'
              style={{
                background:
                  'linear-gradient(135deg, rgba(5,10,18,1), rgba(30,41,59,0.88), rgba(15,118,110,0.66))',
              }}
            >
              <div className='absolute inset-y-0 left-0 w-[55%] bg-linear-to-r from-background via-background/55 to-transparent' />
              <div className='absolute inset-0 bg-linear-to-t from-background/95 via-background/20 to-background/10' />
            </div>
          )}

          <div className='pointer-events-none absolute inset-x-0 top-0 z-10 flex h-[80svh] min-h-[80svh] items-end'>
            <div className='mx-auto w-full max-w-[1960px] px-6 pb-8 sm:px-8 lg:px-10 lg:pb-10'>
              <div className='pointer-events-auto'>
                <DetailHeroSection
                  item={item}
                  progress={progress}
                  itemProgressPercent={itemProgressPercent}
                  selectedSeriesEpisode={selectedSeriesEpisode}
                  overviewExpanded={overviewExpanded}
                  onOverviewExpandedChange={setOverviewExpanded}
                  onOpenPlaybackEntry={onOpenPlaybackEntry}
                  onOpenExternalPlaybackEntry={onOpenExternalPlaybackEntry}
                  seriesSeasons={seriesSeasons}
                  resourceChoices={resourceChoices}
                  resourceSummaries={resourceSummaries}
                  selectedResourceId={selectedResourceId}
                  onSelectResource={onSelectResource}
                  isSelectingResource={isSelectingResource}
                  onManageMetadata={onManageMetadata}
                  onReprobePrimaryFile={
                    primaryResource && primaryResourceFileIds.length > 0
                      ? onReprobePrimaryFile
                      : undefined
                  }
                  isReprobePending={isReprobePending}
                  onMarkWatched={onMarkWatched}
                  isFavorite={isFavorite}
                  onFavoriteToggle={onFavoriteToggle}
                />
              </div>
            </div>
          </div>

          <div className='mx-auto flex min-h-full w-full max-w-[1960px] flex-col px-6 sm:px-8 lg:px-10'>
            <SeriesEpisodesSection
              item={item}
              seasons={seriesSeasons}
              episodePage={episodePage}
              isLoading={isSeriesEpisodesLoading}
              errorMessage={seriesEpisodesErrorMessage}
              selectedEpisodeMetadataItemId={
                selectedSeriesEpisode?.metadata_item_id
              }
              onSelectEpisode={onSelectEpisode}
            />

            <RelatedMediaSection item={item} />

            <PeopleSection item={item} />
          </div>
        </div>
      </div>
    </div>
  )
}
