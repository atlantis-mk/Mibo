import { useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, Home, Search, Settings, Tv, User } from 'lucide-react'
import type { Swiper as SwiperType } from 'swiper/types'

import 'swiper/css'
import 'swiper/css/free-mode'

import { AppTopBar } from '#/components/app-top-bar'
import { Button } from '#/components/ui/button'
import { SidebarTrigger } from '#/components/ui/sidebar'
import type { MediaItemDetail, ProgressState } from '#/lib/mibo-api'

import {
  CastSection,
  DetailHeroSection,
  SpecsSection,
} from './standalone-media-detail-sections'
import { StandaloneMediaDetailTrailerDialog } from './standalone-media-detail-trailer-dialog'
import {
  formatDate,
  formatFileSize,
  formatMediaType,
  formatProbeStatus,
} from './standalone-media-detail-utils'

type StandaloneMediaDetailProps = {
  item: MediaItemDetail
  itemProgressPercent: number
  progress: ProgressState | null
  onGoBack: () => void
  onOpenPlaybackEntry: (options?: { fromStart?: boolean }) => void
  onRematchItem: () => void
  onManageMetadata: () => void
  onMarkWatched: () => void
}

export function StandaloneMediaDetail({
  item,
  itemProgressPercent,
  progress,
  onGoBack,
  onOpenPlaybackEntry,
  onRematchItem,
  onManageMetadata,
  onMarkWatched,
}: StandaloneMediaDetailProps) {
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const [showHeaderLogo, setShowHeaderLogo] = useState(Boolean(item.logo_url))
  const [overviewExpanded, setOverviewExpanded] = useState(false)
  const [castSwiper, setCastSwiper] = useState<SwiperType | null>(null)
  const [canScrollCastPrev, setCanScrollCastPrev] = useState(false)
  const [canScrollCastNext, setCanScrollCastNext] = useState(false)
  const [isTrailerOpen, setIsTrailerOpen] = useState(false)

  useEffect(() => {
    setShowHeaderLogo(Boolean(item.logo_url))
  }, [item.logo_url])

  useEffect(() => {
    setOverviewExpanded(false)
  }, [item.id])

  useEffect(() => {
    setIsTrailerOpen(false)
  }, [item.id])

  const updateCastNavigation = (swiper: SwiperType) => {
    setCanScrollCastPrev(!swiper.isBeginning)
    setCanScrollCastNext(!swiper.isEnd)
  }

  const primaryFile = item.files[0]
  const databaseLinks = [
    item.metadata_provider?.toUpperCase() || null,
    item.external_id || null,
  ]
    .filter(Boolean)
    .join('，')

  const detailGroups = useMemo(
    () => [
      {
        title: '类型',
        value: item.genres?.length
          ? item.genres.join('、')
          : formatMediaType(item.type),
      },
      {
        title: '导演',
        value:
          (item.directors ?? []).map((person) => person.name).join('、') ||
          '暂未识别',
      },
      {
        title: '数据库链接',
        value: databaseLinks || '暂未关联',
      },
      {
        title: '媒体信息',
        value: [
          item.source_path,
          primaryFile ? primaryFile.container.toUpperCase() : null,
          primaryFile?.size_bytes
            ? `${formatFileSize(primaryFile.size_bytes)}  添加于 ${formatDate(item.created_at)}`
            : formatProbeStatus(primaryFile?.probe_status ?? 'pending'),
        ]
          .filter(Boolean)
          .join('\n'),
      },
    ],
    [
      databaseLinks,
      item.created_at,
      item.directors,
      item.genres,
      item.source_path,
      item.type,
      primaryFile,
    ],
  )

  return (
    <div className="relative min-h-svh overflow-hidden bg-background text-foreground">
      <div
        className="absolute inset-0 bg-cover bg-center"
        style={{
          backgroundImage: item.backdrop_url
            ? `url(${item.backdrop_url})`
            : 'linear-gradient(135deg, rgba(54,54,54,0.96), rgba(29,29,29,0.98))',
        }}
      />
      {item.backdrop_url ? (
        <>
          <div className="absolute inset-0 bg-gradient-to-b from-background/80 via-background/70 to-background/95" />
          <div className="absolute inset-0 bg-gradient-to-r from-background via-background/82 to-background/92" />
        </>
      ) : null}

      <div
        ref={scrollContainerRef}
        className="relative z-10 h-svh overflow-y-auto"
      >
        <AppTopBar
          scrollContainerRef={scrollContainerRef}
          leftSlot={
            <>
              <TopBarIconButton
                icon={<ArrowLeft className="size-5" />}
                onClick={onGoBack}
              />
              <Button
                asChild
                variant="ghost"
                size="icon-sm"
                className="size-9 rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              >
                <Link to="/">
                  <Home className="size-4.5" />
                  <span className="sr-only">返回首页</span>
                </Link>
              </Button>
              <SidebarTrigger className="rounded-full border border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground" />
              <div className="min-w-0 pl-1">
                {showHeaderLogo && item.logo_url ? (
                  <img
                    src={item.logo_url}
                    alt={`${item.title} logo`}
                    className="h-6 max-w-65 object-contain object-left"
                    onError={() => setShowHeaderLogo(false)}
                  />
                ) : (
                  <div className="truncate text-[17px] font-semibold tracking-tight text-foreground">
                    {item.title}
                  </div>
                )}
              </div>
            </>
          }
          rightSlot={
            <div className="hidden items-center gap-2 text-muted-foreground md:flex">
              <TopBarIcon icon={<Tv className="size-4.5" />} />
              <TopBarIcon icon={<Search className="size-4.5" />} />
              <TopBarIcon icon={<User className="size-4.5" />} />
              <Button
                asChild
                variant="ghost"
                size="icon-sm"
                className="size-9 rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              >
                <Link to="/settings">
                  <Settings className="size-4.5" />
                  <span className="sr-only">进入设置</span>
                </Link>
              </Button>
            </div>
          }
        />

        <div className="mx-auto flex min-h-full w-full max-w-[1960px] flex-col px-6 pb-14 pt-28 sm:px-8 lg:px-10">
          <section className="grid gap-8 lg:grid-cols-[336px_minmax(0,1fr)] xl:gap-9">
            <div className="mx-auto w-full max-w-[336px] lg:mx-0">
              <div className="overflow-hidden rounded-[10px] border border-border/40 bg-card/80 shadow-xl">
                {item.poster_url ? (
                  <img
                    src={item.poster_url}
                    alt={`${item.title} poster`}
                    className="aspect-[2/3] w-full object-cover"
                  />
                ) : (
                  <div className="aspect-[2/3] bg-muted" />
                )}
              </div>
            </div>

            <DetailHeroSection
              item={item}
              progress={progress}
              itemProgressPercent={itemProgressPercent}
              overviewExpanded={overviewExpanded}
              onOverviewExpandedChange={setOverviewExpanded}
              onOpenPlaybackEntry={onOpenPlaybackEntry}
              onManageMetadata={onManageMetadata}
              onRematchItem={onRematchItem}
              onMarkWatched={onMarkWatched}
            />
          </section>

          <CastSection
            item={item}
            canScrollCastPrev={canScrollCastPrev}
            canScrollCastNext={canScrollCastNext}
            onSwiper={(instance) => {
              setCastSwiper(instance)
              updateCastNavigation(instance)
            }}
            onSlideChange={updateCastNavigation}
            onPrev={() => castSwiper?.slidePrev()}
            onNext={() => castSwiper?.slideNext()}
          />

          <SpecsSection
            detailGroups={detailGroups}
            item={item}
            onOpenTrailer={() => setIsTrailerOpen(true)}
          />
        </div>
      </div>

      <StandaloneMediaDetailTrailerDialog
        open={isTrailerOpen}
        trailer={item.trailer}
        title={item.title}
        onOpenChange={setIsTrailerOpen}
      />
    </div>
  )
}

function TopBarIcon({ icon }: { icon: ReactNode }) {
  return (
    <div className="flex size-9 items-center justify-center rounded-full text-muted-foreground transition hover:bg-accent hover:text-accent-foreground">
      {icon}
    </div>
  )
}

function TopBarIconButton({
  icon,
  onClick,
}: {
  icon: ReactNode
  onClick: () => void
}) {
  return (
    <Button
      variant="ghost"
      size="icon-sm"
      className="size-9 rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground"
      onClick={onClick}
    >
      {icon}
    </Button>
  )
}
