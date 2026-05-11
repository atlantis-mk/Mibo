import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { useEffect, useState, type ReactNode } from "react"
import {
  FileX2Icon,
  HeartIcon,
  InfoIcon,
  MoreHorizontalIcon,
  ShieldCheckIcon,
} from "lucide-react"

import { Button } from "#/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu"
import type {
  CatalogListItem,
  FilenameExclusionPreview,
  ProgressState,
} from "#/lib/mibo-api"
import { buildApiUrl } from "#/lib/mibo-api"
import {
  formatMediaCardTitle,
  formatMediaCardYearRange,
  getMediaCardOrganizingLabel,
  getMediaCardBadgeCount,
  getMediaCardPosterUrl,
  getMediaCardType,
} from "#/lib/media-presentation"
import {
  createAuthedMiboApi,
  favoritesQueryOptions,
  homeDataQueryOptions,
  miboQueryKeys,
} from "#/lib/mibo-query"
import { cn } from "#/lib/utils"
import { useAuthStore } from "#/stores/auth-store"

type MediaPosterCardProps = {
  item: CatalogListItem
  playbackItem?: CatalogListItem
  progress?: ProgressState | null
  progressMeta?: string
  progressDescription?: string
  favorite?: boolean
  libraryName?: string
  layout?: "rail" | "grid"
  imageAspect?: "poster" | "landscape"
  className?: string
}

type MediaLandscapeCardProps = {
  itemId?: number
  imageUrl?: string
  fallbackImageUrl?: string
  title: string
  subtitle?: string
  meta?: string
  status?: string
  description?: string
  current?: boolean
  actionSlot?: ReactNode
  className?: string
}

export function MediaPosterCard({
  item,
  playbackItem,
  progress,
  progressMeta,
  progressDescription,
  favorite,
  layout = "rail",
  imageAspect = "poster",
  className,
}: MediaPosterCardProps) {
  const token = useAuthStore((state) => state.token)
  const queryClient = useQueryClient()
  const [ignoreDialogOpen, setIgnoreDialogOpen] = useState(false)
  const [ignorePreview, setIgnorePreview] =
    useState<FilenameExclusionPreview | null>(null)
  const queryToken = token ?? "guest"
  const title = formatMediaCardTitle(item)
  const progressFrameUrl = useAuthedObjectUrl(progress?.progress_frame_url)
  const posterUrl = progressFrameUrl || getMediaCardPosterUrl(item)
  const badgeCount = getMediaCardBadgeCount(item)
  const hasProgress = Boolean(progress && progress.position_seconds > 0)
  const progressPercent = progress ? getProgressPercent(progress) : 0
  const mediaType = getMediaCardType(item)
  const yearRange = formatMediaCardYearRange(item)
  const resourceSummary = formatResourceSummary(item)
  const metadataLine = [yearRange, resourceSummary, progressMeta]
    .filter(Boolean)
    .join(" · ")
  const playTarget = playbackItem ?? item
  const isInventoryOnly = item.source_kind === "inventory_file"
  const playInventoryFileID = isInventoryOnly
    ? item.inventory_file_id
    : undefined
  const ignoreInventoryFileID = playTarget.inventory_file_id ?? item.inventory_file_id
  const organizingState = item.organizing_summary?.state
  const isOrganizing = Boolean(item.organizing || isInventoryOnly)
  const canOpenDetails = !isInventoryOnly
  const canApplyIgnore =
    typeof ignoreInventoryFileID === "number" &&
    ignoreInventoryFileID > 0 &&
    (!item.organizing || organizingState === "review_required")
  const organizingLabel = getMediaCardOrganizingLabel(item)
  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: Boolean(token) && favorite === undefined,
    staleTime: 60_000,
  })
  const isFavorite = isOrganizing
    ? Boolean(
        favorite ??
          favoritesQuery.data?.some((entry) => sameFavoriteItem(entry.item, item))
      )
    : (favorite ??
      Boolean(
        favoritesQuery.data?.some((entry) => sameFavoriteItem(entry.item, item))
      ))
  const favoriteMutation = useMutation({
    mutationFn: async (favorite: boolean) => {
      if (!token) throw new Error("当前未登录，无法更新收藏。")
      const api = createAuthedMiboApi(token)
      if (isInventoryOnly) throw new Error("生成条目后可收藏。")
		const metadataItemId = item.metadata_item_id
		return favorite
        ? api.addFavorite(metadataItemId)
        : api.removeFavorite(metadataItemId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.favorites(queryToken),
        }),
        queryClient.invalidateQueries({
          queryKey: homeDataQueryOptions(queryToken).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: ["library", "browse"] }),
      ])
    },
  })
  const ignoreMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error("当前未登录，无法标记忽略。")
      if (!canApplyIgnore) throw new Error("当前条目暂不支持标记忽略。")
      if (typeof ignoreInventoryFileID !== "number") {
        throw new Error("当前条目缺少文件锚点，无法标记忽略。")
      }
      return createAuthedMiboApi(token).markInventoryFileScanExclusion(
        ignoreInventoryFileID,
        "advertisement"
      )
    },
    onSuccess: async () => {
      await Promise.all([
        invalidateMediaCardQueries(queryClient, queryToken),
        queryClient.invalidateQueries({
          queryKey: ["settings", "scan-exclusions"],
        }),
      ])
    },
  })
  const previewIgnoreMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error("当前未登录，无法预览忽略影响。")
      if (!canApplyIgnore) throw new Error("当前条目暂不支持标记忽略。")
      if (typeof ignoreInventoryFileID !== "number") {
        throw new Error("当前条目缺少文件锚点，无法预览忽略影响。")
      }
      return createAuthedMiboApi(token).previewInventoryFileScanExclusion(
        ignoreInventoryFileID
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
      if (!canApplyIgnore) throw new Error("当前条目暂不支持同名忽略。")
      if (typeof ignoreInventoryFileID !== "number") {
        throw new Error("当前条目缺少文件锚点，无法标记同名忽略。")
      }
      return createAuthedMiboApi(token).createInventoryFileFilenameExclusionRule(
        ignoreInventoryFileID,
        "advertisement"
      )
    },
    onSuccess: async () => {
      setIgnoreDialogOpen(false)
      await Promise.all([
        invalidateMediaCardQueries(queryClient, queryToken),
        queryClient.invalidateQueries({
          queryKey: ["settings", "scan-exclusions"],
        }),
        queryClient.invalidateQueries({ queryKey: ["home"] }),
        queryClient.invalidateQueries({ queryKey: ["library"] }),
        queryClient.invalidateQueries({ queryKey: ["catalog"] }),
      ])
    },
  })
  const actionsPending =
    favoriteMutation.isPending ||
    ignoreMutation.isPending ||
    previewIgnoreMutation.isPending ||
    filenameGroupMutation.isPending
  const canIgnore =
    canApplyIgnore && playTarget.type !== "series" && playTarget.type !== "show"

  return (
    <article
      className={cn(
        "group relative transition-transform duration-200 [content-visibility:auto] hover:-translate-y-1",
        layout === "grid"
          ? "w-full min-w-0 [contain-intrinsic-size:220px_533px]"
          : "w-[172px] shrink-0 [contain-intrinsic-size:204px_533px] sm:w-[204px]",
        className
      )}
    >
      <div className="relative overflow-hidden rounded-[1.35rem] border border-border/40 bg-card/75 shadow-lg">
        <Link
          to="/play/$id"
          params={{
            id: String(playInventoryFileID ?? playTarget.id),
          }}
          search={{
            fromStart: !hasProgress,
            inventoryFileId: playInventoryFileID,
          }}
          preload="intent"
          aria-label={`${hasProgress ? "继续播放" : "播放"} ${title}`}
          className="absolute inset-0 z-10 rounded-[1.35rem] focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        />
        <div
          className={cn(
            "relative overflow-hidden bg-muted",
            imageAspect === "landscape" ? "aspect-video" : "aspect-[2/3]"
          )}
        >
          {posterUrl ? (
            <img
              src={posterUrl}
              alt=""
              loading="lazy"
              decoding="async"
              fetchPriority="low"
              sizes="(min-width: 1536px) 12vw, (min-width: 1280px) 18vw, (min-width: 1024px) 23vw, (min-width: 640px) 31vw, 47vw"
              className="h-full w-full object-cover"
            />
          ) : (
            <div className="h-full w-full bg-linear-to-b from-indigo-500/35 to-teal-700/35" />
          )}
          {badgeCount ? (
            <span className="absolute top-2 right-2 inline-flex min-w-7 items-center justify-center rounded-full bg-emerald-500 px-2 py-1 text-xs font-bold text-white shadow-lg">
              {badgeCount}
            </span>
          ) : null}
          {isOrganizing ? (
            <span
              className={cn(
                "absolute top-2 left-2 rounded-full border border-white/20 px-2 py-1 text-[0.65rem] font-medium text-white shadow-lg backdrop-blur",
                organizingState === "failed"
                  ? "bg-red-600/75"
                  : organizingState === "review_required"
                    ? "bg-amber-600/75"
                    : "bg-black/55"
              )}
            >
              {organizingLabel}
            </span>
          ) : null}
          {hasProgress ? (
            <div className="absolute right-0 bottom-0 left-0 h-1.5 bg-white/25">
              <div
                className="h-full bg-white shadow-[0_0_12px_rgba(255,255,255,0.6)]"
                style={{ width: `${progressPercent}%` }}
              />
            </div>
          ) : null}
        </div>
        <div className="space-y-3 px-3 pt-3 pb-3">
          <div>
            <div className="line-clamp-1 text-sm font-semibold tracking-tight text-foreground sm:text-base">
              {title}
            </div>
            <div className="mt-1 text-xs text-muted-foreground">
              {metadataLine}
            </div>
            {progressDescription ? (
              <div className="mt-1 line-clamp-1 text-xs text-muted-foreground">
                {progressDescription}
              </div>
            ) : null}
          </div>
          <div className="relative z-20 flex items-center gap-2">
            {canOpenDetails ? (
              <Button asChild size="icon-sm" variant="outline">
                <Link
                  to="/media/$id"
                  params={{ id: String(item.id) }}
                  search={{
                    view: mediaType === "show" ? "series" : undefined,
                    episodePage: undefined,
                  }}
                  preload="intent"
                >
                  <InfoIcon className="size-3.5" />
                  <span className="sr-only">详情</span>
                </Link>
              </Button>
            ) : (
              <Button size="icon-sm" variant="outline" disabled>
                <InfoIcon className="size-3.5" />
                <span className="sr-only">详情</span>
              </Button>
            )}
            <Button
              type="button"
              size="icon-sm"
              variant="outline"
              disabled={isInventoryOnly || !token || favoriteMutation.isPending}
              onClick={() => favoriteMutation.mutate(!isFavorite)}
            >
              <HeartIcon
                className={cn("size-3.5", isFavorite ? "fill-current" : "")}
              />
              <span className="sr-only">
                {isFavorite ? "取消收藏" : "加入收藏"}
              </span>
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  size="icon-sm"
                  variant="outline"
                  disabled={isInventoryOnly || !token}
                >
                  <MoreHorizontalIcon className="size-3.5" />
                  <span className="sr-only">更多操作</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel className="truncate">
                  {title}
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                {isInventoryOnly ? (
                  <DropdownMenuItem disabled>生成条目后可操作</DropdownMenuItem>
                ) : null}
                {isInventoryOnly ? null : (
                  <DropdownMenuItem asChild disabled={actionsPending}>
                    <Link
                      to="/settings/metadata/$id"
                      params={{ id: String(item.id) }}
                    >
                      <ShieldCheckIcon className="size-4" />
                      治理元数据
                    </Link>
                  </DropdownMenuItem>
                )}
                {canIgnore ? (
                  <DropdownMenuItem
                    variant="destructive"
                    disabled={actionsPending}
                    onSelect={() => previewIgnoreMutation.mutate()}
                  >
                    <FileX2Icon className="size-4" />
                    标记忽略
                  </DropdownMenuItem>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
      <Dialog open={ignoreDialogOpen} onOpenChange={setIgnoreDialogOpen}>
        <DialogContent className="grid max-h-[85vh] w-[calc(100vw-2rem)] max-w-2xl grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden p-0">
          <DialogHeader>
            <div className="space-y-2 px-6 pt-6">
              <DialogTitle>选择忽略范围</DialogTitle>
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
              disabled={
                ignoreMutation.isPending || filenameGroupMutation.isPending
              }
              onClick={() => ignoreMutation.mutate()}
            >
              仅忽略当前文件
            </Button>
            <Button
              variant="destructive"
              className="w-full sm:w-auto"
              disabled={
                ignoreMutation.isPending || filenameGroupMutation.isPending
              }
              onClick={() => filenameGroupMutation.mutate()}
            >
              忽略所有同名文件
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </article>
  )
}

function useAuthedObjectUrl(url?: string) {
  const token = useAuthStore((state) => state.token)
  const [objectUrl, setObjectUrl] = useState("")

  useEffect(() => {
    const requestUrl = url
    if (!requestUrl || !token) {
      setObjectUrl("")
      return
    }

    const resolvedUrl = buildApiUrl(requestUrl)
    let cancelled = false
    let nextObjectUrl = ""

    async function loadImage() {
      try {
        const response = await fetch(resolvedUrl, {
          headers: { Authorization: `Bearer ${token}` },
        })
        if (!response.ok) {
          return
        }
        const blob = await response.blob()
        if (cancelled) {
          return
        }
        nextObjectUrl = URL.createObjectURL(blob)
        setObjectUrl(nextObjectUrl)
      } catch {
        if (!cancelled) {
          setObjectUrl("")
        }
      }
    }

    void loadImage()

    return () => {
      cancelled = true
      if (nextObjectUrl) {
        URL.revokeObjectURL(nextObjectUrl)
      }
    }
  }, [token, url])

  return objectUrl
}

function getProgressPercent(progress: ProgressState) {
  if (typeof progress.played_percentage === "number") {
    return clampProgressPercent(progress.played_percentage)
  }

  if (progress.duration_seconds && progress.duration_seconds > 0) {
    return clampProgressPercent(
      (progress.position_seconds / progress.duration_seconds) * 100
    )
  }

  return 0
}

function clampProgressPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
}

function formatResourceSummary(item: CatalogListItem) {
  if (typeof item.resource_count !== "number" || item.resource_count <= 0) {
    return ""
  }
  const parts = [`${item.resource_count} 个资源`]
  if (typeof item.available_count === "number" && item.available_count > 0) {
    parts.push(`${item.available_count} 可播`)
  }
  if (typeof item.missing_count === "number" && item.missing_count > 0) {
    parts.push(`${item.missing_count} 缺失`)
  }
  return parts.join(" / ")
}

function sameFavoriteItem(candidate: CatalogListItem, item: CatalogListItem) {
	return candidate.metadata_item_id === item.metadata_item_id
}

async function invalidateMediaCardQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  queryToken: string
) {
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: homeDataQueryOptions(queryToken).queryKey,
    }),
    queryClient.invalidateQueries({ queryKey: ["library", "browse"] }),
    queryClient.invalidateQueries({ queryKey: ["catalog", "detail"] }),
    queryClient.invalidateQueries({ queryKey: ["catalog", "series-seasons"] }),
  ])
}

export function MediaRail({
  title,
  href,
  children,
}: {
  title: string
  href?: { type?: "movie" | "show" }
  children: ReactNode
}) {
  return (
    <section>
      <div className="mb-4 flex items-center justify-between gap-3">
        {href ? (
          <Link
            to="/library"
            search={{ type: href.type }}
            className="text-xl font-semibold tracking-tight text-foreground underline-offset-4 hover:underline"
          >
            {title}
          </Link>
        ) : (
          <h2 className="text-xl font-semibold tracking-tight text-foreground">
            {title}
          </h2>
        )}
      </div>
      <div className="overflow-x-auto pb-3">
        <div className="flex min-w-max gap-4">{children}</div>
      </div>
    </section>
  )
}

export function MediaLandscapeCard({
  itemId,
  imageUrl,
  fallbackImageUrl,
  title,
  subtitle,
  meta,
  status,
  description,
  current,
  actionSlot,
  className,
}: MediaLandscapeCardProps) {
  const visualUrl = imageUrl || fallbackImageUrl
  const cardContent = (
    <div
      className={cn(
        "group overflow-hidden rounded-[16px] border border-border/40 bg-card/70 shadow-lg backdrop-blur-md transition",
        current && "border-primary/70 bg-primary/10",
        itemId ? "hover:border-border/70 hover:bg-card/85" : "opacity-90",
        className
      )}
    >
      <div className="relative aspect-video overflow-hidden bg-muted">
        {visualUrl ? (
          <img
            src={visualUrl}
            alt={title}
            className="h-full w-full object-cover transition duration-300 group-hover:scale-[1.03]"
          />
        ) : null}
        <div className="absolute inset-0 bg-gradient-to-t from-background/90 via-background/15 to-transparent" />
      </div>
      <div className="space-y-2 p-4">
        {actionSlot && itemId ? (
          <Link
            to="/media/$id"
            params={{ id: String(itemId) }}
            search={{ view: undefined }}
            className="line-clamp-1 text-lg text-foreground underline-offset-4 hover:underline"
          >
            {subtitle ? `${subtitle} - ${title}` : title}
          </Link>
        ) : (
          <div className="line-clamp-1 text-lg text-foreground">
            {subtitle ? `${subtitle} - ${title}` : title}
          </div>
        )}
        {meta ? (
          <div className="text-sm text-muted-foreground">{meta}</div>
        ) : null}
        {status ? (
          <div className="text-xs text-muted-foreground">{status}</div>
        ) : null}
        {description ? (
          <p className="line-clamp-3 text-sm leading-6 text-muted-foreground">
            {description}
          </p>
        ) : null}
        {actionSlot ? <div className="pt-1">{actionSlot}</div> : null}
      </div>
    </div>
  )

  if (!itemId) {
    return cardContent
  }

  if (actionSlot) {
    return cardContent
  }

  return (
    <Link
      to="/media/$id"
      params={{ id: String(itemId) }}
      search={{ view: undefined }}
    >
      {cardContent}
    </Link>
  )
}
