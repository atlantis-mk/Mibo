import { useMemo, useRef, useState, type FormEvent } from "react"
import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import {
  ArrowDownAZIcon,
  ArrowUpAZIcon,
  FilterIcon,
  HomeIcon,
  LoaderCircleIcon,
  SearchIcon,
  Settings2Icon,
} from "lucide-react"

import { AppTopBar } from "#/components/app-top-bar"
import { MediaPosterCard } from "#/components/media-poster-card"
import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog"
import { Input } from "#/components/ui/input"
import { ScrollArea } from "#/components/ui/scroll-area"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "#/components/ui/select"
import { SidebarTrigger } from "#/components/ui/sidebar"
import {
  DiscoveryControls,
  createDefaultDiscoveryFilters,
  type DiscoveryFilters,
} from "#/features/discovery/controls"
import { createAuthedMiboApi, homeDataQueryOptions, miboQueryKeys } from "#/lib/mibo-query"
import type { CatalogListItem, HomeMediaOverview } from "#/lib/mibo-api"
import { cn } from "#/lib/utils"
import { useAuthStore } from "#/stores/auth-store"

export const LIBRARY_PAGE_SIZE_OPTIONS = [24, 48, 60, 96] as const
export const DEFAULT_LIBRARY_PAGE_SIZE = 60

export function isLibraryPageSize(value: number) {
  return LIBRARY_PAGE_SIZE_OPTIONS.some((option) => option === value)
}

type LibraryDetailProps = {
  page: number
  pageSize: number
  filters: DiscoveryFilters
  onPaginationChange: (next: { page?: number; pageSize?: number }) => void
  onFiltersChange: (next: DiscoveryFilters) => void
}

export default function LibraryDetail({
  page,
  pageSize,
  filters,
  onPaginationChange,
  onFiltersChange,
}: LibraryDetailProps) {
  const token = useAuthStore((state) => state.token)
  const queryToken = token ?? "guest"
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const [filterDraft, setFilterDraft] = useState(filters)
  const [filterDialogOpen, setFilterDialogOpen] = useState(false)
  const [pageJumpDraft, setPageJumpDraft] = useState(String(page))

  const overviewQuery = useQuery({
    queryKey: [...homeDataQueryOptions(queryToken).queryKey, "library-overview"],
    enabled: !!token,
    queryFn: async () => {
      if (!token) throw new Error("当前未登录，无法加载内容库。")
      return createAuthedMiboApi(token).homeMediaOverview()
    },
  })

  const browseQuery = useQuery({
    queryKey: miboQueryKeys.libraryBrowse(
      queryToken,
      "global",
      "programs",
      filters,
      page,
      pageSize
    ),
    enabled: !!token,
    queryFn: async () => {
      if (!token) throw new Error("当前未登录，无法加载内容库。")
      return createAuthedMiboApi(token).discoverMedia({
        q: filters.q.trim() || undefined,
        type: filters.type === "all" ? undefined : filters.type,
        genre: filters.genre.trim() || undefined,
        region: filters.region.trim() || undefined,
        year: parseOptionalInt(filters.year),
        min_rating: parseOptionalFloat(filters.minRating),
        watched_state: filters.watchedState,
        organizing_state: filters.organizingState,
        sort: filters.sort,
        sort_direction: filters.sortDirection,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      })
    },
  })

  const items = browseQuery.data?.items ?? []
  const total = browseQuery.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const pageStart = total === 0 ? 0 : (page - 1) * pageSize + 1
  const pageEnd = Math.min(total, page * pageSize)
  const stats = useMemo(() => buildLibraryStats(overviewQuery.data, items), [overviewQuery.data, items])

  const openFilterDialog = () => {
    setFilterDraft(filters)
    setFilterDialogOpen(true)
  }

  const applyFilterDraft = () => {
    onFiltersChange(filterDraft)
    onPaginationChange({ page: 1 })
    setFilterDialogOpen(false)
  }

  const resetFilterDraft = () => {
    setFilterDraft(
      createDefaultDiscoveryFilters({
        type: filters.type,
        sort: "title",
        sortDirection: "asc",
      })
    )
  }

  const toggleTitleSort = () => {
    onFiltersChange({
      ...filters,
      sort: "title",
      sortDirection:
        filters.sort === "title" && filters.sortDirection === "asc"
          ? "desc"
          : "asc",
    })
    onPaginationChange({ page: 1 })
  }

  const submitPageJump = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const nextPage = Number.parseInt(pageJumpDraft, 10)
    if (!Number.isFinite(nextPage)) {
      setPageJumpDraft(String(page))
      return
    }
    const clampedPage = Math.min(pageCount, Math.max(1, nextPage))
    setPageJumpDraft(String(clampedPage))
    if (clampedPage !== page) onPaginationChange({ page: clampedPage })
  }

  const pageTitle =
    filters.type === "movie" ? "电影库" : filters.type === "show" ? "剧集库" : "内容库"

  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      <AppTopBar
        scrollContainerRef={scrollContainerRef}
        contentClassName="max-w-none"
        leftSlot={
          <>
            <SidebarTrigger />
            <Button asChild size="icon" variant="ghost" className="hidden sm:inline-flex">
              <Link to="/">
                <HomeIcon className="size-4" />
                <span className="sr-only">首页</span>
              </Link>
            </Button>
            <div className="flex min-w-0 items-baseline gap-2">
              <div className="shrink-0 text-lg font-semibold">{pageTitle}</div>
              <div className="truncate text-xs text-muted-foreground">
                基于内容聚合浏览 · 共 {total} 项
              </div>
            </div>
          </>
        }
        rightSlot={
          <div className="flex items-center gap-2">
            <Badge variant="outline">电影 {stats.movieCount}</Badge>
            <Badge variant="outline">剧集 {stats.showCount}</Badge>
            <Button asChild size="icon" variant="ghost">
              <Link to="/search" search={{ q: undefined }}>
                <SearchIcon className="size-4" />
                <span className="sr-only">搜索</span>
              </Link>
            </Button>
            <Button asChild size="icon" variant="ghost">
              <Link to="/settings">
                <Settings2Icon className="size-4" />
                <span className="sr-only">进入设置</span>
              </Link>
            </Button>
          </div>
        }
      />

      <ScrollArea
        className="h-svh"
        viewportClassName="[&>div]:block! [&>div]:w-full! [&>div]:min-w-0!"
        viewportRef={scrollContainerRef}
      >
        <section className="min-w-0 px-3 pt-22 pb-16 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-[1800px] min-w-0 space-y-6">
            <Dialog open={filterDialogOpen} onOpenChange={setFilterDialogOpen}>
              <DialogContent className="max-h-[min(760px,calc(100svh-2rem))] max-w-none overflow-y-auto sm:max-w-none">
                <DialogHeader>
                  <DialogTitle>筛选内容库</DialogTitle>
                  <DialogDescription>
                    当前页面直接使用 discovery 接口的筛选条件。
                  </DialogDescription>
                </DialogHeader>
                <form
                  className="space-y-4"
                  onSubmit={(event) => {
                    event.preventDefault()
                    applyFilterDraft()
                  }}
                >
                  <DiscoveryControls filters={filterDraft} showSearch onChange={setFilterDraft} />
                  <DialogFooter>
                    <Button type="button" variant="ghost" onClick={resetFilterDraft}>
                      重置
                    </Button>
                    <DialogClose asChild>
                      <Button type="button" variant="outline">
                        取消
                      </Button>
                    </DialogClose>
                    <Button type="submit">应用筛选</Button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>

            <div className="flex flex-wrap items-center gap-2 rounded-[1.5rem] border border-border/45 bg-card/62 p-3 backdrop-blur-xl">
              <Button
                type="button"
                variant={filterDialogOpen ? "default" : "outline"}
                className="rounded-full"
                onClick={openFilterDialog}
              >
                <FilterIcon className="size-4" />
                筛选
              </Button>
              <Button
                type="button"
                variant={filters.sort === "title" ? "default" : "outline"}
                className="rounded-full"
                onClick={toggleTitleSort}
              >
                {filters.sortDirection === "asc" ? (
                  <ArrowDownAZIcon className="size-4" />
                ) : (
                  <ArrowUpAZIcon className="size-4" />
                )}
                标题{filters.sortDirection === "asc" ? "升序" : "降序"}
              </Button>
              <Badge variant="outline" className="rounded-full bg-background/60">
                {formatActiveType(filters.type)}
              </Badge>
              <Badge variant="outline" className="rounded-full bg-background/60">
                {total} 项
              </Badge>
            </div>

            <LibraryResults
              isLoading={browseQuery.isLoading}
              isFetching={browseQuery.isFetching}
              error={browseQuery.error?.message}
              items={items}
              onRetry={() => void browseQuery.refetch()}
            />

            {total > 0 ? (
              <div className="flex min-h-16 flex-col gap-4 rounded-[1.5rem] border border-border/45 bg-card/65 p-4 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
                <div className="flex flex-wrap items-center gap-3">
                  <span>
                    第 {page} / {pageCount} 页，显示 {pageStart}-{pageEnd} / {total} 项
                  </span>
                  {browseQuery.isFetching ? (
                    <span className="inline-flex items-center gap-1.5">
                      <LoaderCircleIcon className="size-4 animate-spin" />
                      正在加载
                    </span>
                  ) : null}
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <Select
                    value={String(pageSize)}
                    onValueChange={(nextPageSize) =>
                      onPaginationChange({ page: 1, pageSize: Number(nextPageSize) })
                    }
                  >
                    <SelectTrigger size="sm" className="rounded-full bg-background/70">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {LIBRARY_PAGE_SIZE_OPTIONS.map((option) => (
                        <SelectItem key={option} value={String(option)}>
                          {option} 项
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <form className="flex items-center gap-2" onSubmit={submitPageJump}>
                    <Input
                      type="number"
                      min={1}
                      max={pageCount}
                      inputMode="numeric"
                      value={pageJumpDraft}
                      onChange={(event) => setPageJumpDraft(event.currentTarget.value)}
                      className="w-20 rounded-full bg-background/70 text-center"
                    />
                    <Button type="submit" variant="outline" className="rounded-full">
                      跳转
                    </Button>
                  </form>
                  <Button
                    type="button"
                    variant="outline"
                    className="rounded-full"
                    disabled={page <= 1 || browseQuery.isFetching}
                    onClick={() => onPaginationChange({ page: page - 1 })}
                  >
                    上一页
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    className="rounded-full"
                    disabled={page >= pageCount || browseQuery.isFetching}
                    onClick={() => onPaginationChange({ page: page + 1 })}
                  >
                    下一页
                  </Button>
                </div>
              </div>
            ) : null}
          </div>
        </section>
      </ScrollArea>
    </div>
  )
}

function LibraryResults({
  isLoading,
  isFetching,
  error,
  items,
  onRetry,
}: {
  isLoading: boolean
  isFetching: boolean
  error?: string
  items: CatalogListItem[]
  onRetry: () => void
}) {
  if (isLoading) {
    return (
      <div className="flex min-h-80 items-center justify-center rounded-[2rem] border border-border/50 bg-card/70 shadow-sm backdrop-blur-xl">
        <div className="flex items-center gap-3 rounded-full border border-border/50 bg-background/70 px-5 py-3 text-sm text-muted-foreground">
          <LoaderCircleIcon className="size-4 animate-spin" />
          正在加载内容库
        </div>
      </div>
    )
  }

  if (error && items.length === 0) {
    return (
      <div className="space-y-4 rounded-[2rem] border border-destructive/30 bg-destructive/10 px-6 py-10 text-center text-sm text-destructive shadow-sm backdrop-blur-xl">
        <p>{error}</p>
        <Button type="button" variant="outline" className="rounded-full" onClick={onRetry}>
          重新加载
        </Button>
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="rounded-[2rem] border border-border/50 bg-card/70 px-6 py-14 text-center shadow-sm backdrop-blur-xl">
        <h2 className="text-2xl font-semibold tracking-tight">暂无可显示内容</h2>
        <p className="mx-auto mt-3 max-w-xl text-sm leading-7 text-muted-foreground">
          当前筛选条件下没有内容，可以调整类型、进度、排序或关键词后重试。
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {error ? (
        <div className="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      ) : null}
      <div
        className={cn(
          "grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6",
          isFetching && "opacity-80"
        )}
      >
        {items.map((item) => (
          <MediaPosterCard
            key={item.source_kind === "inventory_file" ? `file-${item.inventory_file_id}` : `item-${item.metadata_item_id || item.id}`}
            item={item}
            layout="grid"
          />
        ))}
      </div>
    </div>
  )
}

function buildLibraryStats(overview: HomeMediaOverview | undefined, items: CatalogListItem[]) {
  return {
    movieCount:
      overview?.sections.find((section) => section.key === "movies")?.count ??
      items.filter((item) => item.type === "movie").length,
    showCount:
      overview?.sections.find((section) => section.key === "series")?.count ??
      items.filter((item) => item.type === "show" || item.type === "series").length,
  }
}

function formatActiveType(type: DiscoveryFilters["type"]) {
  if (type === "movie") return "电影"
  if (type === "show") return "剧集"
  return "全部类型"
}

function parseOptionalInt(value: string) {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) ? parsed : undefined
}

function parseOptionalFloat(value: string) {
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : undefined
}
