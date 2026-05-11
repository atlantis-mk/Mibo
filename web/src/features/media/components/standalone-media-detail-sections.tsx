import { useEffect, useMemo, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  ChevronLeft,
  ChevronRight,
  Disc3,
  FileX2Icon,
  Film,
  Play,
} from "lucide-react"
import { FreeMode } from "swiper/modules"
import { Swiper, SwiperSlide } from "swiper/react"
import type { Swiper as SwiperType } from "swiper/types"

import { Button } from "#/components/ui/button"
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "#/components/ui/pagination"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog"
import {
  MediaLandscapeCard,
  MediaPosterCard,
} from "#/components/media-poster-card"
import type { FilenameExclusionPreview } from "#/lib/mibo-api"
import type {
  CatalogEpisodeRail,
  CatalogSeasonRail,
  MediaDetailPresentation,
} from "#/lib/media-presentation"
import { createAuthedMiboApi } from "#/lib/mibo-query"
import { useAuthStore } from "#/stores/auth-store"

export { DetailHeroSection } from "./standalone-media-detail-hero"
export { PeopleSection, SpecsSection } from "./standalone-media-detail-specs"

import {
  formatAvailabilityStatus,
  formatRuntime,
} from "./standalone-media-detail-utils"

const EPISODES_PER_PAGE = 60

export function SeriesEpisodesSection({
  item,
  seasons,
  episodePage,
  isLoading,
  errorMessage,
}: {
  item: MediaDetailPresentation
  seasons: CatalogSeasonRail[]
  episodePage: number
  isLoading: boolean
  errorMessage: string | null
}) {
  const navigate = useNavigate({ from: "/media/$id" })
  const numberedSeasons = useMemo(
    () => seasons.filter((season) => !isSpecialSeason(season)),
    [seasons]
  )
  const specialsSeasons = useMemo(
    () => seasons.filter((season) => isSpecialSeason(season)),
    [seasons]
  )
  const [selectedSeasonNumber, setSelectedSeasonNumber] = useState<
    number | undefined
  >(numberedSeasons[0]?.season_number)

  useEffect(() => {
    if (numberedSeasons.length === 0) {
      setSelectedSeasonNumber(undefined)
      return
    }
    if (
      !numberedSeasons.some(
        (season) => season.season_number === selectedSeasonNumber
      )
    ) {
      setSelectedSeasonNumber(numberedSeasons[0].season_number)
    }
  }, [numberedSeasons, selectedSeasonNumber])

  const selectedSeason =
    numberedSeasons.find(
      (season) => season.season_number === selectedSeasonNumber
    ) ?? numberedSeasons[0]

  if (item.type !== "series" && item.type !== "episode") {
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
          {item.series_title_display || item.title}{" "}
          的剧集按季展示，每页最多 60 集。
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
          {numberedSeasons.length > 1 ? (
            <div className="flex flex-wrap gap-2">
              {numberedSeasons.map((season) => (
                <Button
                  key={season.season_number}
                  type="button"
                  size="sm"
                  variant={
                    selectedSeason?.season_number === season.season_number
                      ? "default"
                      : "outline"
                  }
                  className="rounded-full"
                  onClick={() => {
                    setSelectedSeasonNumber(season.season_number)
                    void navigate({
                      search: (previous) => ({
                        ...previous,
                        episodePage: 1,
                      }),
                    })
                  }}
                >
                  {season.name?.trim() || `第 ${season.season_number} 季`}
                </Button>
              ))}
            </div>
          ) : null}
          {selectedSeason ? (
            <SeasonEpisodesRail season={selectedSeason} page={episodePage} />
          ) : null}
          {specialsSeasons.map((season) => (
            <SeasonEpisodesRail
              key={`special-${season.season_number}-${season.name}`}
              season={season}
              title="特别篇"
              page={episodePage}
            />
          ))}
        </div>
      ) : null}
    </section>
  )
}

function SeasonEpisodesRail({
  season,
  title,
  page,
}: {
  season: CatalogSeasonRail
  title?: string
  page: number
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
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-1">
          <h3 className="text-[28px] font-semibold tracking-tight text-foreground">
            {title || season.name?.trim() || `第 ${season.season_number} 季`}
          </h3>
          <div className="text-sm text-muted-foreground">
            共 {season.episodes.length} 集
            {season.runtime_seconds
              ? ` · ${formatRuntime(season.runtime_seconds)}`
              : ""}
            {totalPages > 1
              ? ` · 当前 ${pageStart + 1}-${pageStart + visibleEpisodes.length}`
              : ""}
          </div>
        </div>
      </div>

      <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        {visibleEpisodes.map((episode) => (
          <EpisodeCard
			key={`${season.season_number}-${episode.episode_number}-${episode.metadata_item_id}`}
			episode={episode}
			fallbackImage={season.poster_url}
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
    <Pagination className="justify-end">
      <PaginationContent>
        <PaginationItem>
          <PaginationPrevious
            text="上一页"
            href={buildEpisodePageHref(currentPage - 1)}
            aria-disabled={currentPage === 1}
            className={currentPage === 1 ? "pointer-events-none opacity-50" : ""}
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
            text="下一页"
            href={buildEpisodePageHref(currentPage + 1)}
            aria-disabled={currentPage === totalPages}
            className={
              currentPage === totalPages ? "pointer-events-none opacity-50" : ""
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
    search.delete("episodePage")
  } else {
    search.set("episodePage", String(page))
  }
  const query = search.toString()
  return `${window.location.pathname}${query ? `?${query}` : ""}`
}

function EpisodeCard({
  episode,
  fallbackImage,
}: {
  episode: CatalogEpisodeRail
  fallbackImage: string
}) {
  const token = useAuthStore((state) => state.token)
  const queryClient = useQueryClient()
  const [ignoreDialogOpen, setIgnoreDialogOpen] = useState(false)
  const [ignorePreview, setIgnorePreview] =
    useState<FilenameExclusionPreview | null>(null)
  const title = episode.name?.trim() || `第 ${episode.episode_number} 集`
  const episodeLabel = `S${episode.season_number}:E${episode.episode_number}`
  const statusLabel = episode.watched
    ? "已看完"
    : typeof episode.progress_percent === "number" &&
        episode.progress_percent > 0
      ? `已观看 ${Math.round(episode.progress_percent)}%`
      : formatAvailabilityStatus(episode.availability_status)
  const hasProgress = Boolean(
    !episode.watched &&
      typeof episode.progress_percent === "number" &&
      episode.progress_percent > 0
  )
  const inventoryFileId = episode.inventory_file_id
  const invalidateAfterIgnore = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["catalog"] }),
      queryClient.invalidateQueries({ queryKey: ["library"] }),
      queryClient.invalidateQueries({ queryKey: ["home"] }),
      queryClient.invalidateQueries({
        queryKey: ["settings", "scan-exclusions"],
      }),
    ])
  }
  const singleFileIgnoreMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error("当前未登录，无法标记忽略。")
      if (typeof inventoryFileId !== "number" || inventoryFileId <= 0) {
        throw new Error("当前剧集缺少文件锚点，无法标记忽略。")
      }
      return createAuthedMiboApi(token).markInventoryFileScanExclusion(
        inventoryFileId,
        "advertisement"
      )
    },
    onSuccess: async () => {
      setIgnoreDialogOpen(false)
      await invalidateAfterIgnore()
    },
  })
  const previewIgnoreMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error("当前未登录，无法预览忽略影响。")
      if (typeof inventoryFileId !== "number" || inventoryFileId <= 0) {
        throw new Error("当前剧集缺少文件锚点，无法预览忽略影响。")
      }
      return createAuthedMiboApi(token).previewInventoryFileScanExclusion(
        inventoryFileId
      )
    },
    onSuccess: (preview) => {
      setIgnorePreview(preview)
      setIgnoreDialogOpen(true)
    },
  })
  const filenameGroupMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error("当前未登录，无法标记同名忽略。")
      if (typeof inventoryFileId !== "number" || inventoryFileId <= 0) {
        throw new Error("当前剧集缺少文件锚点，无法标记同名忽略。")
      }
      return createAuthedMiboApi(token).createInventoryFileFilenameExclusionRule(
        inventoryFileId,
        "advertisement"
      )
    },
    onSuccess: async () => {
      setIgnoreDialogOpen(false)
      await invalidateAfterIgnore()
    },
  })
  const ignorePending =
    singleFileIgnoreMutation.isPending ||
    previewIgnoreMutation.isPending ||
    filenameGroupMutation.isPending

  const actionSlot =
    episode.availability_status === "available" ? (
      <div className="flex flex-wrap items-center gap-2">
        <Button asChild size="sm" className="rounded-full">
          <Link
            to="/play/$id"
				params={{ id: String(episode.metadata_item_id) }}
            search={{
              fromStart: !hasProgress,
              inventoryFileId: undefined,
            }}
            preload={false}
            aria-label={`${hasProgress ? "继续播放" : "播放"} ${title}`}
            onClick={(event) => event.stopPropagation()}
          >
            <Play className="size-4 fill-current" />
            {hasProgress ? "继续" : "播放"}
          </Link>
        </Button>
        <Button
          type="button"
          size="sm"
          variant="destructive"
          className="rounded-full"
          disabled={!token || !inventoryFileId || ignorePending}
          onClick={(event) => {
            event.preventDefault()
            event.stopPropagation()
            previewIgnoreMutation.mutate()
          }}
        >
          <FileX2Icon className="size-4" />
          忽略
        </Button>
      </div>
    ) : null

  return (
    <div className="group relative">
      <MediaLandscapeCard
		itemId={episode.metadata_item_id}
        imageUrl={episode.still_url}
        fallbackImageUrl={fallbackImage}
        title={title}
        subtitle={episodeLabel}
        meta={[
          formatEpisodeAirDate(episode.air_date),
          formatRuntime(episode.runtime_seconds),
        ]
          .filter(Boolean)
          .join("  ")}
        status={statusLabel}
        description={episode.overview || "暂无剧情简介"}
        current={episode.current}
        actionSlot={actionSlot}
      />
      <Dialog open={ignoreDialogOpen} onOpenChange={setIgnoreDialogOpen}>
        <DialogContent className="grid max-h-[85vh] w-[calc(100vw-2rem)] max-w-2xl grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden p-0">
          <DialogHeader>
            <div className="space-y-2 px-6 pt-6">
              <DialogTitle>忽略这一集</DialogTitle>
              <DialogDescription>
                先确认同名文件影响范围，再选择只忽略当前文件或忽略所有同名文件。
              </DialogDescription>
            </div>
          </DialogHeader>
          <div className="min-h-0 overflow-y-auto px-6 py-4">
            {ignorePreview ? (
              <div className="min-w-0 space-y-3">
                <div className="min-w-0 rounded-xl border border-border/60 bg-muted/40 p-3 text-sm">
                  <div className="font-medium break-all">
                    {ignorePreview.normalized_filename}
                  </div>
                  <div className="mt-1 break-all text-muted-foreground">
                    {ignorePreview.library_name ||
                      `#${ignorePreview.library_id}`}{" "}
                    / {ignorePreview.storage_provider}，共影响{" "}
                    {ignorePreview.affected_count} 个文件
                  </div>
                </div>
                <div className="max-h-64 min-w-0 space-y-2 overflow-y-auto rounded-xl border border-border/60 p-3">
                  {ignorePreview.affected_files.map((file) => (
                    <div
                      key={file.id}
                      className="text-xs break-all text-muted-foreground"
                      title={file.storage_path}
                    >
                      {file.storage_path}
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
          <div className="flex flex-col gap-2 border-t border-border/60 bg-muted/30 px-6 py-4 sm:flex-row sm:justify-end">
            <Button
              variant="outline"
              className="w-full sm:w-auto"
              disabled={ignorePending}
              onClick={() => singleFileIgnoreMutation.mutate()}
            >
              仅忽略当前文件
            </Button>
            <Button
              variant="destructive"
              className="w-full sm:w-auto"
              disabled={ignorePending}
              onClick={() => filenameGroupMutation.mutate()}
            >
              忽略所有同名文件
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
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
    <section className="mt-12 space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-[19px] font-semibold text-foreground">
            <Film className="size-4 text-muted-foreground" />
            相似推荐
          </div>
          <p className="text-sm text-muted-foreground">
            基于同媒体库和标签生成的相关内容。
          </p>
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
      <div className="relative left-1/2 w-screen -translate-x-1/2">
        <Swiper
          modules={[FreeMode]}
          freeMode
          slidesPerView="auto"
          spaceBetween={20}
          slidesOffsetBefore={40}
          onSwiper={(instance) => {
            setSwiper(instance)
            updateNavigation(instance)
          }}
          onSlideChange={updateNavigation}
          onResize={updateNavigation}
          className="!overflow-x-clip !overflow-y-visible pt-1 pb-3"
        >
          {relatedItems.map((relatedItem) => (
            <SwiperSlide key={relatedItem.id} className="!h-auto !w-auto">
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
    name.includes("special") ||
    name.includes("特别") ||
    name.includes("番外")
  )
}

function RailArrowButton({
  direction,
  disabled,
  onClick,
}: {
  direction: "prev" | "next"
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
      {direction === "prev" ? (
        <ChevronLeft className="size-4" />
      ) : (
        <ChevronRight className="size-4" />
      )}
      <span className="sr-only">
        {direction === "prev" ? "上一组剧集" : "下一组剧集"}
      </span>
    </Button>
  )
}

function formatEpisodeAirDate(value?: string) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "numeric",
    day: "numeric",
  }).format(date)
}
