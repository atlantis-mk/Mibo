import type { ComponentType } from "react"
import { Link } from "@tanstack/react-router"
import {
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
  HomeContentSection,
  HomeMediaSectionSummary,
} from "#/lib/mibo-api"
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
              添加媒体源并完成扫描后，最近加入的影片或剧集会自动切换为首页轮播。
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
                            resourceId: undefined,
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

export function ContentSectionRail({
  contentSections,
}: {
  contentSections: HomeContentSection[]
}) {
  if (contentSections.length === 0) {
    return (
      <section className="relative flex min-h-[calc(100svh-18rem)] items-center justify-center border-t border-border/40 bg-background px-4 py-16 sm:px-6 lg:px-8">
        <div className="rounded-[2rem] border border-border/40 bg-card/70 px-6 py-8 text-center text-sm text-muted-foreground backdrop-blur-sm">
          还没有可展示的最新内容，稍后会按电影、剧集等内容形态自动补充到这里。
        </div>
      </section>
    )
  }

  return (
    <section className="relative border-t border-border/40 bg-background px-4 pt-10 pb-16 sm:px-6 lg:px-8">
      <div>
        <div className="space-y-8">
          {contentSections.map((section) => (
            <section key={section.key}>
              <div className="mb-4 flex items-center justify-between gap-3">
                <h2 className="text-xl font-semibold tracking-tight text-foreground">
                  最新{section.title}
                </h2>
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
                    <SwiperSlide
                      key={`${section.key}-${item.library_id}-${item.metadata_item_id ?? item.id}`}
                      className="!w-auto"
                    >
                      <MediaPosterCard item={item} />
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

export function ContentShapeEntrance({
  movieCount,
  showCount,
  sections,
}: {
  movieCount: number
  showCount: number
  sections: HomeMediaSectionSummary[]
}) {
  const moviePosters = getEntrancePosters(sections, "movies")
  const showPosters = getEntrancePosters(sections, "series")

  return (
    <section className="pointer-events-none absolute inset-x-0 bottom-8 z-20 px-4 sm:px-6 lg:px-8">
      <div className="pointer-events-auto flex h-64 gap-3 overflow-x-auto px-1 pt-3 pb-1">
        <ContentShapeEntranceCard
          title="电影"
          description="进入全部电影，按最近加入继续发现。"
          count={movieCount}
          type="movie"
          posters={moviePosters}
        />
        <ContentShapeEntranceCard
          title="剧集"
          description="进入全部剧集，快速回到连续内容。"
          count={showCount}
          type="show"
          posters={showPosters}
        />
      </div>
    </section>
  )
}

function getEntrancePosters(
  sections: HomeMediaSectionSummary[],
  key: "movies" | "series"
) {
  return (
    sections
      .find((section) => section.key === key)
      ?.items.map(getMediaCardPosterUrl)
      .filter(Boolean)
      .slice(0, 4) ?? []
  )
}

function ContentShapeEntranceCard({
  title,
  description,
  count,
  type,
  posters,
}: {
  title: string
  description: string
  count: number
  type: "movie" | "show"
  posters: string[]
}) {
  const shouldUseCollage = posters.length >= 3
  const primaryPoster = posters[0]

  return (
    <Link
      to="/library"
      search={{ type }}
      className="group flex h-full w-[240px] shrink-0 flex-col overflow-hidden rounded-[1rem] border border-border/40 bg-card/70 shadow-2xl shadow-black/25 backdrop-blur-xl transition-transform hover:-translate-y-1 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary sm:w-[280px] lg:w-[320px]"
    >
      <div className="relative overflow-hidden bg-muted">
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
      <div className="flex items-center justify-between gap-3 px-4 py-2">
        <div className="min-w-0">
          <div className="truncate text-sm font-semibold tracking-tight">
            {title}
          </div>
          <div className="mt-1 truncate text-xs text-muted-foreground">
            {description}
          </div>
        </div>
        <div className="shrink-0 text-right">
          <div className="text-lg font-semibold">{count}</div>
          <div className="text-[10px] text-muted-foreground">条内容</div>
        </div>
      </div>
    </Link>
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
				  key={`${entry.item.metadata_item_id}-${entry.resource_id ?? "default"}`}
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
