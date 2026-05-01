import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { useState, type ReactNode } from "react"
import {
  FileX2Icon,
  HeartIcon,
  InfoIcon,
  MoreHorizontalIcon,
  RefreshCwIcon,
} from "lucide-react"

import { Badge } from "#/components/ui/badge"
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
import {
  formatMediaCardTitle,
  formatMediaCardYearRange,
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
  favorite?: boolean
  libraryName?: string
  layout?: "rail" | "grid"
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
  className?: string
}

export function MediaPosterCard({
  item,
  playbackItem,
  progress,
  favorite,
  libraryName,
  layout = "rail",
  className,
}: MediaPosterCardProps) {
  const token = useAuthStore((state) => state.token)
  const queryClient = useQueryClient()
  const [ignoreDialogOpen, setIgnoreDialogOpen] = useState(false)
  const [ignorePreview, setIgnorePreview] =
    useState<FilenameExclusionPreview | null>(null)
  const queryToken = token ?? "guest"
  const title = formatMediaCardTitle(item)
  const posterUrl = getMediaCardPosterUrl(item)
  const badgeCount = getMediaCardBadgeCount(item)
  const hasProgress = Boolean(progress && progress.position_seconds > 0)
  const mediaType = getMediaCardType(item)
  const playTarget = playbackItem ?? item
  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: Boolean(token) && favorite === undefined,
    staleTime: 60_000,
  })
  const isFavorite =
    favorite ??
    Boolean(favoritesQuery.data?.some((entry) => entry.item.id === item.id))
  const favoriteMutation = useMutation({
    mutationFn: async (favorite: boolean) => {
      if (!token) throw new Error("当前未登录，无法更新收藏。")
      const api = createAuthedMiboApi(token)
      return favorite ? api.addFavorite(item.id) : api.removeFavorite(item.id)
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
  const identifyMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error("当前未登录，无法重新识别。")
      return createAuthedMiboApi(token).matchCatalogItem(playTarget.id)
    },
    onSuccess: async () => {
      await invalidateMediaCardQueries(queryClient, queryToken)
    },
  })
  const ignoreMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error("当前未登录，无法标记忽略。")
      return createAuthedMiboApi(token).markCatalogItemScanExclusion(
        playTarget.id,
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
      return createAuthedMiboApi(token).previewCatalogItemScanExclusion(
        playTarget.id
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
      return createAuthedMiboApi(token).createCatalogItemFilenameExclusionRule(
        playTarget.id,
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
    identifyMutation.isPending ||
    ignoreMutation.isPending ||
    previewIgnoreMutation.isPending ||
    filenameGroupMutation.isPending
  const canIgnore = playTarget.type !== "series" && playTarget.type !== "show"

  return (
    <article
      className={cn(
        "group relative transition-transform duration-200 hover:-translate-y-1 [content-visibility:auto]",
        layout === "grid"
          ? "w-full min-w-0 [contain-intrinsic-size:220px_533px]"
          : "w-[172px] shrink-0 [contain-intrinsic-size:204px_533px] sm:w-[204px]",
        className
      )}
    >
      <div className="relative overflow-hidden rounded-[1.35rem] border border-border/40 bg-card/75 shadow-lg">
        <Link
          to="/play/$id"
          params={{ id: String(playTarget.id) }}
          search={{ fromStart: !hasProgress, assetId: undefined }}
          preload={false}
          aria-label={`${hasProgress ? "继续播放" : "播放"} ${title}`}
          className="absolute inset-0 z-10 rounded-[1.35rem] focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        />
        <div className="relative aspect-[2/3] overflow-hidden bg-muted">
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
        </div>
        <div className="space-y-3 px-3 pt-3 pb-3">
          <div>
            <div className="line-clamp-1 text-sm font-semibold tracking-tight text-foreground sm:text-base">
              {title}
            </div>
            <div className="mt-1 text-xs text-muted-foreground">
              {formatMediaCardYearRange(item)}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge
              className="rounded-full border-border/50 bg-background/80 px-2 py-0.5 text-[10px]"
              variant="outline"
            >
              {mediaType === "movie" ? "电影" : "剧集"}
            </Badge>
            {libraryName ? (
              <span className="min-w-0 truncate text-[10px] text-muted-foreground">
                {libraryName}
              </span>
            ) : null}
          </div>
          <div className="relative z-20 flex items-center gap-2">
            <Button
              asChild
              size="icon-sm"
              variant="outline"
              className="size-8 rounded-full border-border/50 bg-background/80"
            >
              <Link
                to="/media/$id"
                params={{ id: String(item.id) }}
                search={{ view: mediaType === "show" ? "series" : undefined }}
                preload={false}
              >
                <InfoIcon className="size-3.5" />
                <span className="sr-only">详情</span>
              </Link>
            </Button>
            <Button
              type="button"
              size="icon-sm"
              variant="outline"
              className={cn(
                "size-8 rounded-full border-border/50 bg-background/80",
                isFavorite ? "text-rose-400" : "text-muted-foreground"
              )}
              disabled={!token || favoriteMutation.isPending}
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
                  className="size-8 rounded-full border-border/50 bg-background/80 text-muted-foreground"
                  disabled={!token}
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
                <DropdownMenuItem
                  disabled={actionsPending}
                  onSelect={() => identifyMutation.mutate()}
                >
                  <RefreshCwIcon className="size-4" />
                  重新识别
                </DropdownMenuItem>
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
  href?: { libraryId: number }
  children: ReactNode
}) {
  return (
    <section>
      <div className="mb-4 flex items-center justify-between gap-3">
        {href ? (
          <Link
            to="/library/$id"
            params={{ id: String(href.libraryId) }}
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
        <div className="line-clamp-1 text-lg text-foreground">
          {subtitle ? `${subtitle} - ${title}` : title}
        </div>
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
      </div>
    </div>
  )

  if (!itemId) {
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
