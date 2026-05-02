import { useEffect, useMemo, useRef, useState } from "react"
import type { ReactNode } from "react"
import { Link, useNavigate } from "@tanstack/react-router"
import { ArrowLeft, Home, Search, Settings, Tv, User } from "lucide-react"

import "swiper/css"
import "swiper/css/free-mode"

import { AppTopBar } from "#/components/app-top-bar"
import { Button } from "#/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "#/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu"
import { SidebarTrigger } from "#/components/ui/sidebar"
import type { CatalogAssetDetail, ProgressState } from "#/lib/mibo-api"
import type {
  CatalogDetailPresentation,
  CatalogSeasonRail,
} from "#/lib/media-presentation"
import {
  formatMediaDetailYearRange,
  formatMediaRating,
  formatProviderLabel,
  formatSeasonSummary,
} from "#/lib/media-presentation"
import { useAuthStore } from "#/stores/auth-store"

import {
  DetailHeroSection,
  PeopleSection,
  RelatedMediaSection,
  SeriesEpisodesSection,
  SpecsSection,
} from "./standalone-media-detail-sections"
import {
  formatAssetLabel,
  formatAvailabilityStatus,
  formatMediaType,
  formatProbeStatus,
  getDisplayDatabaseLinks,
  getDisplaySourcePath,
  getPrimaryCatalogAsset,
} from "./standalone-media-detail-utils"
import { cn } from "#/lib/utils"

type StandaloneMediaDetailProps = {
  item: CatalogDetailPresentation
  itemProgressPercent: number
  progress: ProgressState | null
  seriesSeasons: CatalogSeasonRail[]
  isSeriesEpisodesLoading: boolean
  seriesEpisodesErrorMessage: string | null
  onGoBack: () => void
  onOpenPlaybackEntry: (options?: {
    itemId?: number
    fromStart?: boolean
    assetId?: number
  }) => void
  onOpenAssetPlaybackEntry?: (assetId: number) => void
  assetChoices?: CatalogAssetDetail[]
  onRematchItem: () => void
  onReprobePrimaryFile: () => void
  isReprobePending: boolean
  onManageMetadata: () => void
  onMarkWatched: () => void
  isFavorite: boolean
  onFavoriteToggle: (favorite: boolean) => void
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
  isFavorite,
  onFavoriteToggle,
}: StandaloneMediaDetailProps) {
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.user)
  const clearSession = useAuthStore((state) => state.clearSession)
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
  const isEpisode = item.type === "episode"
  const databaseLinks = getDisplayDatabaseLinks(item)
  const genreLabel = (
    item.genres.length > 0 ? item.genres : item.tags.map((tag) => tag.name)
  ).join(" / ")
  const ratingLabel = [
    formatMediaRating(item.community_rating)
      ? `评分 ${formatMediaRating(item.community_rating)}`
      : null,
    item.official_rating ? `分级 ${item.official_rating}` : null,
  ]
    .filter(Boolean)
    .join("\n")
  const dateLabel = [
    formatMediaDetailYearRange(item)
      ? `年份 ${formatMediaDetailYearRange(item)}`
      : null,
    item.release_date ? `上映 ${formatShortDate(item.release_date)}` : null,
    item.first_air_date ? `首播 ${formatShortDate(item.first_air_date)}` : null,
    item.last_air_date
      ? `最近播出 ${formatShortDate(item.last_air_date)}`
      : null,
    item.series_status
      ? `状态 ${formatSeriesStatus(item.series_status)}`
      : null,
    formatSeasonSummary(item) || null,
  ]
    .filter(Boolean)
    .join("\n")

  const handleLogout = async () => {
    clearSession()
    await navigate({ to: "/login", search: { redirect: "/" }, replace: true })
  }

  const detailGroups = useMemo(
    () => [
      {
        title: "类型与可用性",
        value: [
          formatMediaType(item.type),
          formatAvailabilityStatus(item.availability_status),
        ]
          .filter(Boolean)
          .join("\n"),
      },
      {
        title: "类型 / 标签",
        value: genreLabel || "暂未标注",
      },
      {
        title: "评分与分级",
        value: ratingLabel || "暂无评分",
      },
      {
        title: "日期与状态",
        value: dateLabel || "暂无日期信息",
      },
      {
        title: "来源",
        value: [
          item.metadata_provider
            ? `主提供方 ${formatProviderLabel(item.metadata_provider)}`
            : null,
          databaseLinks || null,
          getDisplaySourcePath(item),
        ]
          .filter(Boolean)
          .join("\n"),
      },
      {
        title: "技术信息",
        value: [
          formatAssetLabel(primaryAsset),
          primaryAsset
            ? `文件 ${primaryAsset.file_ids.length} 个 · ${formatProbeStatus(primaryAsset.probe_status)}`
            : null,
        ]
          .filter(Boolean)
          .join("\n"),
      },
    ],
    [databaseLinks, dateLabel, genreLabel, item, primaryAsset, ratingLabel]
  )

  return (
    <div className="relative min-h-svh w-full max-w-full overflow-hidden bg-background text-foreground">
      <div
        className="absolute inset-0 bg-cover bg-center"
        style={{
          backgroundImage: item.backdrop_url
            ? `url(${item.backdrop_url})`
            : "linear-gradient(135deg, rgba(54,54,54,0.96), rgba(29,29,29,0.98))",
        }}
      />
      {item.backdrop_url ? (
        <>
          <div className="absolute inset-0 bg-gradient-to-b from-background/46 via-background/32 to-background/78" />
          <div className="absolute inset-0 bg-gradient-to-r from-background/56 via-background/40 to-background/60" />
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
                label="返回上一页"
                onClick={onGoBack}
              />
              <Button asChild variant="ghost" size="icon-sm">
                <Link to="/">
                  <Home className="size-4.5" />
                  <span className="sr-only">返回首页</span>
                </Link>
              </Button>
              <SidebarTrigger />
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
              <Dialog>
                <DialogTrigger asChild>
                  <Button variant="ghost" size="icon-sm">
                    <Tv className="size-4.5" />
                    <span className="sr-only">投屏</span>
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>投屏暂不可用</DialogTitle>
                    <DialogDescription>
                      设备发现和投屏控制还没有接入当前播放器。后续可以继续实现
                      Chromecast / AirPlay。
                    </DialogDescription>
                  </DialogHeader>
                </DialogContent>
              </Dialog>
              <Button asChild variant="ghost" size="icon-sm">
                <Link to="/search" search={{ q: undefined }}>
                  <Search className="size-4.5" />
                  <span className="sr-only">搜索</span>
                </Link>
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon-sm">
                    <User className="size-4.5" />
                    <span className="sr-only">用户菜单</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                  <DropdownMenuLabel>
                    {user?.username ?? "当前用户"}
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem asChild>
                    <Link to="/favorites">收藏</Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link to="/settings">设置</Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={() => void handleLogout()}>
                    退出登录
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <Button asChild variant="ghost" size="icon-sm">
                <Link to="/settings">
                  <Settings className="size-4.5" />
                  <span className="sr-only">进入设置</span>
                </Link>
              </Button>
            </div>
          }
        />

        <div className="mx-auto flex min-h-full w-full max-w-[1960px] flex-col px-6 pt-28 pb-14 sm:px-8 lg:px-10">
          <section
            className={cn(
              "grid gap-8 xl:gap-9",
              isEpisode
                ? "lg:grid-cols-[minmax(360px,560px)_minmax(0,1fr)]"
                : "lg:grid-cols-[336px_minmax(0,1fr)]"
            )}
          >
            <div
              className={cn(
                "mx-auto w-full lg:mx-0",
                isEpisode ? "max-w-[560px]" : "max-w-[336px]"
              )}
            >
              <div className="overflow-hidden rounded-[10px] border border-border/40 bg-card/80 shadow-xl">
                {item.poster_url ? (
                  <img
                    src={item.poster_url}
                    alt={`${item.title} poster`}
                    className={cn(
                      "w-full object-cover",
                      isEpisode ? "aspect-video" : "aspect-[2/3]"
                    )}
                  />
                ) : (
                  <div
                    className={cn(
                      isEpisode ? "aspect-video" : "aspect-[2/3]",
                      "bg-muted"
                    )}
                  />
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
              isFavorite={isFavorite}
              onFavoriteToggle={onFavoriteToggle}
            />
          </section>

          <SeriesEpisodesSection
            item={item}
            seasons={seriesSeasons}
            isLoading={isSeriesEpisodesLoading}
            errorMessage={seriesEpisodesErrorMessage}
          />

          <RelatedMediaSection item={item} />

          <PeopleSection item={item} />

          <SpecsSection detailGroups={detailGroups} item={item} />
        </div>
      </div>
    </div>
  )
}

function TopBarIconButton({
  icon,
  label,
  onClick,
}: {
  icon: ReactNode
  label: string
  onClick: () => void
}) {
  return (
    <Button variant="ghost" size="icon-sm" onClick={onClick}>
      {icon}
      <span className="sr-only">{label}</span>
    </Button>
  )
}

function formatShortDate(value?: string) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value.slice(0, 10)
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "numeric",
    day: "numeric",
  }).format(date)
}

function formatSeriesStatus(status: string) {
  switch (status.trim().toLowerCase()) {
    case "continuing":
    case "returning series":
      return "更新中"
    case "ended":
      return "已完结"
    case "canceled":
    case "cancelled":
      return "已取消"
    default:
      return status
  }
}
