import type { ComponentType } from "react"
import { Link } from "@tanstack/react-router"
import {
  ArrowUpRightIcon,
  ClapperboardIcon,
  InfoIcon,
  LibraryBigIcon,
  PlayIcon,
  TvIcon,
} from "lucide-react"
import { Autoplay, FreeMode } from "swiper/modules"
import { Swiper, SwiperSlide } from "swiper/react"

import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import { MediaPosterCard } from "#/components/media-poster-card"
import type {
  CatalogListItem,
  CatalogUserItemEntry,
  Library,
} from "#/lib/mibo-api"
import { formatSourceContentClass } from "#/lib/library-presentation"
import {
  formatMediaCardTitle,
  getMediaCardBackdropUrl,
  getMediaCardPosterUrl,
  getMediaCardType,
  getPrimarySeriesTitle,
} from "#/lib/media-presentation"
import { cn } from "#/lib/utils"

const DEFAULT_OVERVIEW =
  "最近加入的内容会在这里轮播展示，方便在首页快速发现刚扫描入库的媒体。"

export function HeroCarousel({
  heroItems,
  canLoopHeroItems,
  userName,
  continueWatchingCount,
  movieCount,
  showCount,
  hasBottomOverlay = false,
}: {
  heroItems: any[]
  canLoopHeroItems: boolean
  userName: string
  continueWatchingCount: number
  movieCount: number
  showCount: number
  hasBottomOverlay?: boolean
}) {
  if (heroItems.length === 0) {
    return (
      <section
        className="relative min-h-svh overflow-hidden"
        style={{
          background:
            "linear-gradient(135deg, rgba(5,10,18,1), rgba(30,41,59,0.88), rgba(15,118,110,0.66))",
        }}
      >
        <div className="absolute inset-0 bg-linear-to-r from-background via-background/15 to-background/95" />
        <div className="absolute inset-0 bg-linear-to-t from-background/95 via-background/20 to-background/10" />
        <div
          className={cn(
            "relative flex min-h-svh items-end px-6 pt-24 sm:px-8 lg:px-12",
            hasBottomOverlay ? "pb-81" : "pb-8 lg:pb-10"
          )}
        >
          <div className="max-w-4xl min-w-0">
            <Badge
              className="border-border/50 bg-background/75 backdrop-blur-sm"
              variant="outline"
            >
              首页已就绪
            </Badge>
            <h1 className="mt-5 max-w-3xl text-4xl font-semibold tracking-tight text-balance sm:text-5xl lg:text-6xl">
              等待扫描后的最近加入内容
            </h1>
            <p className="mt-4 max-w-2xl text-sm leading-7 text-muted-foreground sm:text-base">
              媒体库入口已显示在下方。扫描完成后，最近加入的影片或剧集会自动切换为首页轮播。
            </p>
          </div>
        </div>
      </section>
    )
  }

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
                        "linear-gradient(135deg, rgba(5,10,18,1), rgba(30,41,59,0.88), rgba(15,118,110,0.66))",
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
              <div
                className={cn(
                  "relative flex min-h-svh items-end px-6 pt-24 sm:px-8 lg:px-12",
                  hasBottomOverlay ? "pb-81" : "pb-8 lg:pb-10"
                )}
              >
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
                      {getMediaCardType(item) !== "show" && seriesTitle ? (
                        <>
                          <span>•</span>
                          <span>{seriesTitle}</span>
                        </>
                      ) : null}
                    </div>
                    <h1 className="mt-5 max-w-3xl text-4xl font-semibold tracking-tight text-balance sm:text-5xl lg:text-6xl">
                      {displayTitle}
                    </h1>
                    <p className="mt-4 line-clamp-3 max-w-2xl text-sm leading-7 text-muted-foreground sm:text-base">
                      {item.overview || DEFAULT_OVERVIEW}
                    </p>
                    <div className="mt-6 flex flex-wrap gap-3">
                      <Button asChild size="lg">
                        <Link
                          to="/play/$id"
                          params={{ id: String(item.id) }}
                          search={{
                            fromStart: false,
                            assetId: undefined,
                            inventoryFileId: undefined,
                          }}
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
                              getMediaCardType(item) === "show"
                                ? "series"
                                : undefined,
                            episodePage: undefined,
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
    </Swiper>
  )
}

export function LatestLibraryRail({
  latestLibrarySections,
}: {
  latestLibrarySections: any[]
}) {
  if (latestLibrarySections.length === 0) {
    return (
      <section className="relative flex min-h-[calc(100svh-18rem)] items-center justify-center border-t border-border/40 bg-background px-4 py-16 sm:px-6 lg:px-8">
        <div className="rounded-[2rem] border border-border/40 bg-card/70 px-6 py-8 text-center text-sm text-muted-foreground backdrop-blur-sm">
          还没有可展示的最新内容，稍后会按媒体库自动补充到这里。
        </div>
      </section>
    )
  }

  return (
    <section className="relative border-t border-border/40 bg-background px-4 pt-10 pb-16 sm:px-6 lg:px-8">
      <div>
        <div className="space-y-8">
          {latestLibrarySections.map((section) => (
            <section key={section.library_id}>
              <div className="mb-4 flex items-center justify-between gap-3">
                <Link
                  to="/library/$id"
                  params={{ id: String(section.library_id) }}
                  className="text-xl font-semibold tracking-tight text-foreground underline-offset-4 hover:underline"
                >
                  最新{section.library_name}
                </Link>
              </div>
              <div className="-mx-4 sm:-mx-6 lg:-mx-8">
                <Swiper
                  modules={[FreeMode]}
                  slidesPerView="auto"
                  spaceBetween={16}
                  slidesOffsetBefore={16}
                  breakpoints={{
                    640: { slidesOffsetBefore: 24 },
                    1024: { slidesOffsetBefore: 32 },
                  }}
                  freeMode
                  className="!overflow-x-clip !overflow-y-visible pt-1 pb-3"
                >
                  {section.items.map((item: CatalogListItem) => (
                    <SwiperSlide key={item.id} className="!w-auto">
                      <MediaPosterCard
                        item={item}
                        libraryName={section.library_name}
                      />
                    </SwiperSlide>
                  ))}
                </Swiper>
              </div>
            </section>
          ))}
        </div>
      </div>
    </section>
  )
}

export function MyMediaSection({
  libraries,
  latestLibrarySections,
  variant = "default",
}: {
  libraries: Library[]
  latestLibrarySections: { library_id: number; items: CatalogListItem[] }[]
  variant?: "default" | "heroOverlay"
}) {
  const postersByLibrary = new Map(
    latestLibrarySections.map((section) => [
      section.library_id,
      section.items.map(getMediaCardPosterUrl).filter(Boolean).slice(0, 4),
    ])
  )

  if (libraries.length === 0) {
    return (
      <section
        className={cn(
          variant === "heroOverlay"
            ? "pointer-events-none absolute inset-x-0 bottom-8 z-20 px-4 sm:px-6 lg:px-8"
            : "px-4 py-10 sm:px-6 lg:px-8"
        )}
      >
        <div
          className={cn(
            "mx-auto max-w-[1600px] rounded-[2rem] border border-border/40 px-6 py-8 text-sm text-muted-foreground backdrop-blur-sm",
            variant === "heroOverlay"
              ? "pointer-events-auto bg-background/70 shadow-2xl shadow-black/25"
              : "bg-card/70"
          )}
        >
          还没有媒体库。前往设置添加媒体源和媒体库后，这里会显示你的媒体入口。
        </div>
      </section>
    )
  }

  return (
    <section
      className={cn(
        variant === "heroOverlay"
          ? "pointer-events-none absolute inset-x-0 bottom-8 z-20 px-4 sm:px-6 lg:px-8"
          : "px-4 py-10 sm:px-6 lg:px-8"
      )}
    >
      <div
        className={cn(
          variant === "heroOverlay"
            ? "pointer-events-auto"
            : "mx-auto max-w-[1600px]"
        )}
      >
        {variant === "default" ? (
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
        ) : null}
        <div
          className={cn(
            variant === "heroOverlay"
              ? "flex h-64 gap-3 overflow-x-auto px-1 pt-3 pb-1"
              : "grid gap-4 sm:grid-cols-2 xl:grid-cols-4"
          )}
        >
          {libraries.map((library) => (
            <LibraryCollageCard
              key={library.id}
              library={library}
              posters={postersByLibrary.get(library.id) ?? []}
              variant={variant}
            />
          ))}
        </div>
      </div>
    </section>
  )
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
    <section className="border-t border-border/40 bg-background px-4 py-10 sm:px-6 lg:px-8">
      <div>
        <section>
          <div className="mb-4 flex items-center justify-between gap-3">
            <h2 className="text-xl font-semibold tracking-tight text-foreground">
              继续观看
            </h2>
          </div>
          <Swiper
            modules={[FreeMode]}
            slidesPerView="auto"
            spaceBetween={16}
            freeMode
            className="!overflow-x-clip !overflow-y-visible pt-1 pb-3"
          >
            {entries.map((entry) => {
              const displayItem = entry.display_item ?? entry.item
              const playbackItem = entry.play_item ?? entry.item
              const { progressMeta, progressDescription } =
                formatContinueWatchingProgress(playbackItem)

              return (
                <SwiperSlide
                  key={`${entry.item.id}-${entry.asset_id ?? "default"}`}
                  className="!w-auto"
                >
                  <MediaPosterCard
                    item={displayItem}
                    playbackItem={playbackItem}
                    progress={entry}
                    progressMeta={progressMeta}
                    progressDescription={progressDescription}
                    imageAspect="landscape"
                    className="w-[280px] sm:w-[360px]"
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

function formatContinueWatchingProgress(playbackItem: CatalogUserItemEntry["item"]) {
  if (playbackItem.type !== "episode") {
    return { progressMeta: "", progressDescription: "" }
  }

  const episodeLabel = formatEpisodeProgressLabel(playbackItem)
  const episodeTitle = playbackItem.title?.trim()
  const progressMeta = [episodeLabel, episodeTitle].filter(Boolean).join("-")

  return { progressMeta, progressDescription: "" }
}

function formatEpisodeProgressLabel(item: CatalogUserItemEntry["item"]) {
  const label = item.episode_label?.trim()
  if (label) {
    return label
  }

  const seasonNumber = item.parent_index_number
  const episodeNumber = item.index_number
  const episodeNumberEnd = item.index_number_end

  if (typeof seasonNumber !== "number" && typeof episodeNumber !== "number") {
    return ""
  }

  const season = typeof seasonNumber === "number" ? `S${seasonNumber}` : ""
  const episode =
    typeof episodeNumber === "number"
      ? `E${episodeNumber}${
          typeof episodeNumberEnd === "number" && episodeNumberEnd !== episodeNumber
            ? `-E${episodeNumberEnd}`
            : ""
        }`
      : ""

  return [season, episode].filter(Boolean).join(":")
}

function LibraryCollageCard({
  library,
  posters,
  variant = "default",
}: {
  library: Library
  posters: string[]
  variant?: "default" | "heroOverlay"
}) {
  const shouldUseCollage = posters.length >= 3
  const primaryPoster = posters[0]

  return (
    <Link
      to="/library/$id"
      params={{ id: String(library.id) }}
      className={cn(
        "group flex flex-col overflow-hidden rounded-[1.75rem] border border-border/40 bg-card/70 shadow-lg transition-transform hover:-translate-y-1 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary",
        variant === "heroOverlay"
          ? "h-full w-[240px] shrink-0 rounded-[1rem] sm:w-[280px] lg:w-[320px]"
          : ""
      )}
    >
      <div
        className={cn(
          "relative overflow-hidden bg-muted"
        )}
      >
        {shouldUseCollage ? (
          <div className="flex justify-center overflow-hidden">
            {posters.slice(0, 3).map((poster, index) => (
              <div
                key={`${poster}-${index}`}
                className="aspect-[2/3] w-1/3 shrink-0 rounded-none bg-cover bg-center"
                style={{ backgroundImage: `url(${poster})` }}
              />
            ))}
          </div>
        ) : primaryPoster ? (
          <div
            className="aspect-[2/3] w-full bg-cover bg-center"
            style={{ backgroundImage: `url(${primaryPoster})` }}
          />
        ) : (
          <div className="flex aspect-[2/3] w-full items-center justify-center bg-card/80 px-4 text-center text-sm text-muted-foreground">
            暂无封面
          </div>
        )}
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(115deg,hsl(var(--background)/0.92)_0%,hsl(var(--background)/0.58)_34%,transparent_34.5%,transparent_61%,hsl(var(--background)/0.74)_61.5%,hsl(var(--background)/0.9)_100%)] opacity-95 transition-opacity group-hover:opacity-80" />
      </div>
      <div
        className={cn(
          "flex items-center justify-between gap-3 px-4",
          variant === "heroOverlay" ? "py-2" : "py-4"
        )}
      >
        <div className="min-w-0">
          <div
            className={cn(
              "truncate font-semibold tracking-tight",
              variant === "heroOverlay" ? "text-sm" : "text-lg"
            )}
          >
            {library.name}
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {formatSourceContentClass(library.probe_summary?.dominant_class)} ·{" "}
            {library.status}
          </div>
          {library.collections?.length ? (
            <div className="mt-1 truncate text-xs text-muted-foreground">
              {library.collections
                .map((collection) => `${collection.label} ${collection.count}`)
                .join(" · ")}
            </div>
          ) : null}
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
        "rounded-[1.75rem] border border-border/40 bg-card/75 p-4 backdrop-blur-md",
        compact ? "min-w-0" : ""
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
  if (type === "movie") return "电影"
  if (type === "show" || type === "episode") return "剧集"
  return "媒体"
}
