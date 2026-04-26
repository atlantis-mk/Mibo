import { useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, Home, Search, Settings, Tv, User } from 'lucide-react'

import 'swiper/css'
import 'swiper/css/free-mode'

import { AppTopBar } from '#/components/app-top-bar'
import { Button } from '#/components/ui/button'
import { SidebarTrigger } from '#/components/ui/sidebar'
import type { CatalogAssetDetail, ProgressState } from '#/lib/mibo-api'
import type {
  CatalogDetailPresentation,
  CatalogSeasonRail,
} from '#/lib/media-presentation'

import {
  DetailHeroSection,
  SeriesEpisodesSection,
  SpecsSection,
} from './standalone-media-detail-sections'
import {
  formatAssetLabel,
  formatAvailabilityStatus,
  formatMediaType,
  formatProbeStatus,
  getDisplayDatabaseLinks,
  getDisplaySourcePath,
  getPrimaryCatalogAsset,
} from './standalone-media-detail-utils'

type StandaloneMediaDetailProps = {
  item: CatalogDetailPresentation
  itemProgressPercent: number
  progress: ProgressState | null
  seriesSeasons: CatalogSeasonRail[]
  isSeriesEpisodesLoading: boolean
  seriesEpisodesErrorMessage: string | null
  onGoBack: () => void
  onOpenPlaybackEntry: (options?: { fromStart?: boolean }) => void
  onOpenAssetPlaybackEntry?: (assetId: number) => void
  assetChoices?: CatalogAssetDetail[]
  onRematchItem: () => void
  onReprobePrimaryFile: () => void
  isReprobePending: boolean
  onManageMetadata: () => void
  onMarkWatched: () => void
}

export function StandaloneMediaDetail({
  item,
  itemProgressPercent,
  progress,
  seriesSeasons,
  isSeriesEpisodesLoading,
  seriesEpisodesErrorMessage,
  onGoBack,
  onOpenPlaybackEntry,
  onOpenAssetPlaybackEntry,
  assetChoices = [],
  onRematchItem,
  onReprobePrimaryFile,
  isReprobePending,
  onManageMetadata,
  onMarkWatched,
}: StandaloneMediaDetailProps) {
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const [showHeaderLogo, setShowHeaderLogo] = useState(Boolean(item.logo_url))
  const [overviewExpanded, setOverviewExpanded] = useState(false)

  useEffect(() => {
    setShowHeaderLogo(Boolean(item.logo_url))
  }, [item.logo_url])

  useEffect(() => {
    setOverviewExpanded(false)
  }, [item.id])

  const primaryAsset = getPrimaryCatalogAsset(item)
  const databaseLinks = getDisplayDatabaseLinks(item)

  const detailGroups = useMemo(
    () => [
      {
        title: '类型',
        value: formatMediaType(item.type),
      },
      {
        title: '可用性',
        value: formatAvailabilityStatus(item.availability_status),
      },
      {
        title: '数据库链接',
        value: databaseLinks || '暂未关联',
      },
      {
        title: '媒体信息',
        value: [
          getDisplaySourcePath(item),
          formatAssetLabel(primaryAsset),
          primaryAsset
            ? `文件 ${primaryAsset.file_ids.length} 个 · ${formatProbeStatus(primaryAsset.probe_status)}`
            : null,
        ]
          .filter(Boolean)
          .join('\n'),
      },
    ],
    [databaseLinks, item, primaryAsset],
  )

  return (
    <div className="relative min-h-svh w-full max-w-full overflow-hidden bg-background text-foreground">
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
        className="relative z-10 h-svh overflow-x-hidden overflow-y-auto"
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
              onOpenAssetPlaybackEntry={onOpenAssetPlaybackEntry}
              assetChoices={assetChoices}
              onManageMetadata={onManageMetadata}
              onRematchItem={onRematchItem}
              onReprobePrimaryFile={
                primaryAsset && primaryAsset.file_ids.length > 0
                  ? onReprobePrimaryFile
                  : undefined
              }
              isReprobePending={isReprobePending}
              onMarkWatched={onMarkWatched}
            />
          </section>

          <SeriesEpisodesSection
            item={item}
            seasons={seriesSeasons}
            isLoading={isSeriesEpisodesLoading}
            errorMessage={seriesEpisodesErrorMessage}
          />

          <SpecsSection detailGroups={detailGroups} item={item} />
        </div>
      </div>
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
