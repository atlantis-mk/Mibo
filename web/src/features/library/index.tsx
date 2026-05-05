import { useMemo, useRef, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  ArrowDownAZIcon,
  ArrowUpAZIcon,
  CastIcon,
  FilterIcon,
  HeartIcon,
  HomeIcon,
  LoaderCircleIcon,
  MoreHorizontalIcon,
  PlayIcon,
  SearchIcon,
  Settings2Icon,
  UserCircleIcon,
} from "lucide-react"
import { toast } from "sonner"

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
import {
  getMediaCardBackdropUrl,
  getMediaCardPosterUrl,
} from "#/lib/media-presentation"
import {
  createAuthedMiboApi,
  favoritesQueryOptions,
  miboQueryKeys,
} from "#/lib/mibo-query"
import type { CatalogListItem } from "#/lib/mibo-api"
import { cn } from "#/lib/utils"
import { useAuthStore } from "#/stores/auth-store"

export const LIBRARY_PAGE_SIZE_OPTIONS = [24, 48, 60, 96] as const
export const DEFAULT_LIBRARY_PAGE_SIZE = 60

export function isLibraryPageSize(value: number) {
  return LIBRARY_PAGE_SIZE_OPTIONS.some((option) => option === value)
}

type LibraryTab =
  | "programs"
  | "recommended"
  | "trailers"
  | "favorites"
  | "genres"
  | "tags"
  | "platforms"
  | "episodes"
  | "folders"

const LIBRARY_TABS: Array<{ value: LibraryTab; label: string }> = [
  { value: "programs", label: "节目" },
  { value: "recommended", label: "推荐" },
  { value: "trailers", label: "预告" },
  { value: "favorites", label: "收藏" },
  { value: "genres", label: "类型" },
  { value: "tags", label: "标签" },
  { value: "platforms", label: "播出平台" },
  { value: "episodes", label: "集" },
  { value: "folders", label: "文件夹" },
]

export default function LibraryDetail({
  libraryId,
  page,
  pageSize,
  filters,
  onPaginationChange,
  onFiltersChange,
}: {
  libraryId: number
  page: number
  pageSize: number
  filters: DiscoveryFilters
  onPaginationChange: (next: { page?: number; pageSize?: number }) => void
  onFiltersChange: (next: DiscoveryFilters) => void
}) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const clearSession = useAuthStore((state) => state.clearSession)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const queryToken = token ?? "guest"
  const hasValidLibraryId = Number.isFinite(libraryId) && libraryId > 0
  const [filterDraft, setFilterDraft] = useState(filters)
  const [activeTab, setActiveTab] = useState<LibraryTab>("programs")
  const [filterDialogOpen, setFilterDialogOpen] = useState(false)
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)

  const setFilters = (next: DiscoveryFilters) => {
    setFilterDraft(next)
    onFiltersChange(next)
  }

  const setFilterDialogOpenState = (open: boolean) => {
    if (open) setFilterDraft(filters)
    setFilterDialogOpen(open)
  }

  const applyFilterDraft = () => {
    setFilters(filterDraft)
    setFilterDialogOpen(false)
  }

  const resetFilterDraft = () => {
    setFilterDraft(
      createDefaultDiscoveryFilters({ sort: "title", sortDirection: "asc" })
    )
  }

  const selectTab = (tab: LibraryTab) => {
    setActiveTab(tab)
    onPaginationChange({ page: 1 })
  }

  const libraryQuery = useQuery({
    queryKey: miboQueryKeys.libraryDetail(queryToken, libraryId),
    enabled: hasHydrated && !!token && hasValidLibraryId,
    queryFn: async () => {
      if (!token) throw new Error("当前未登录，无法加载媒体库。")
      return createAuthedMiboApi(token).getLibrary(libraryId)
    },
  })
  const browseQuery = useQuery({
    queryKey: miboQueryKeys.libraryBrowse(
      queryToken,
      libraryId,
      activeTab,
      filters,
      page,
      pageSize
    ),
    enabled:
      hasHydrated &&
      !!token &&
      hasValidLibraryId &&
      (activeTab === "programs" || activeTab === "episodes"),
    queryFn: async () => {
      if (!token) throw new Error("当前未登录，无法加载媒体库内容。")
      return createAuthedMiboApi(token).discoverMedia({
        scope: "library",
        library_id: libraryId,
        q: filters.q.trim() || undefined,
        type: activeTab === "episodes" ? "episode" : filters.type,
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
  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const scanChangesMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error("当前未登录，无法扫描媒体库。")
      return createAuthedMiboApi(token).scanLibrary(libraryId, "changed")
    },
    onSuccess: () => {
      toast.success("变化扫描任务已提交。")
      void queryClient.invalidateQueries({
        queryKey: miboQueryKeys.libraryDetail(queryToken, libraryId),
      })
    },
    onError: (error: Error) => {
      toast.error(error.message || "提交变化扫描任务失败。")
    },
  })
  const libraryFavorites = useMemo(
    () =>
      (favoritesQuery.data ?? []).filter(
        (entry) => entry.item.library_id === libraryId
      ),
    [favoritesQuery.data, libraryId]
  )
  const isBrowseTab = activeTab === "programs" || activeTab === "episodes"
  const browseItems = browseQuery.data?.items ?? []
  const items =
    activeTab === "favorites"
      ? libraryFavorites.map((entry) => entry.item)
      : browseItems
  const total =
    activeTab === "favorites" ? items.length : (browseQuery.data?.total ?? 0)
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const pageStart = total === 0 ? 0 : (page - 1) * pageSize + 1
  const pageEnd = Math.min(total, page * pageSize)
  const movieCount = items.filter((item) => item.type === "movie").length
  const showCount = items.filter((item) => item.type !== "movie").length
  const retryBrowse = () => {
    void browseQuery.refetch()
  }
  const changePageSize = (nextPageSize: string) => {
    onPaginationChange({ page: 1, pageSize: Number(nextPageSize) })
  }
  const openFilterDialog = () => {
    setFilterDialogOpenState(true)
  }
  const toggleTitleSort = () => {
    setFilters({
      ...filters,
      sort: "title",
      sortDirection:
        filters.sort === "title" && filters.sortDirection === "asc"
          ? "desc"
          : "asc",
    })
  }
  const heroBackdrop =
    items
      .map(
        (item) => getMediaCardBackdropUrl(item) || getMediaCardPosterUrl(item)
      )
      .find(Boolean) ?? null

  const handleLogout = async () => {
    if (token) {
      try {
        await createAuthedMiboApi(token).logout()
      } catch {
        // Local session cleanup is still valid if the server session already expired.
      }
    }
    clearSession()
    await navigate({
      to: "/login",
      search: { redirect: `/library/${libraryId}` },
      replace: true,
    })
  }

  if (!hasHydrated || (token && libraryQuery.isLoading)) {
    return <LibraryDetailLoading />
  }

  if (!token || !user) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background px-6 text-foreground">
        <div className="space-y-4 text-center">
          <Badge
            className="border-border/60 bg-background/80"
            variant="outline"
          >
            Mibo Library
          </Badge>
          <h1 className="text-4xl font-semibold tracking-tight">
            登录后查看媒体库
          </h1>
          <p className="text-sm leading-7 text-muted-foreground sm:text-base">
            当前媒体库页面依赖已登录会话访问后端接口。
          </p>
          <Button asChild className="rounded-full px-6">
            <Link to="/login" search={{ redirect: `/library/${libraryId}` }}>
              前往登录
            </Link>
          </Button>
        </div>
      </div>
    )
  }

  if (!hasValidLibraryId) {
    return <LibraryDetailError message="无效的媒体库 ID。" />
  }

  if (libraryQuery.error) {
    return <LibraryDetailError message={libraryQuery.error.message} />
  }

  if (!libraryQuery.data) {
    return <LibraryDetailError message="未找到对应的媒体库。" />
  }

  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      <AppTopBar
        scrollContainerRef={scrollContainerRef}
        contentClassName="max-w-none"
        leftSlot={
          <>
            <SidebarTrigger />
            <Button
              asChild
              size="icon"
              variant="ghost"
              className="hidden sm:inline-flex"
            >
              <Link to="/">
                <HomeIcon className="size-4" />
                <span className="sr-only">首页</span>
              </Link>
            </Button>
            <div className="flex min-w-0 items-baseline gap-2">
              <div className="shrink-0 text-lg font-semibold">
                {libraryQuery.data.name}
              </div>
              <div className="truncate text-xs text-muted-foreground">
                媒体库 · 共 {total} 项 · {activeTabLabel(activeTab)}
              </div>
            </div>
          </>
        }
        rightSlot={
          <TopBarActions
            username={user.username}
            movieCount={movieCount}
            showCount={showCount}
            onLogout={handleLogout}
          />
        }
      />

      <ScrollArea
        className="h-svh"
        viewportClassName="[&>div]:block! [&>div]:w-full! [&>div]:min-w-0!"
        viewportRef={scrollContainerRef}
      >
        <section className="min-w-0 px-3 pt-22 pb-16 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-[1800px] min-w-0">
            <Dialog
              open={filterDialogOpen}
              onOpenChange={setFilterDialogOpenState}
            >
              <DialogContent className="max-h-[min(760px,calc(100svh-2rem))] max-w-none overflow-y-auto sm:max-w-none">
                <DialogHeader>
                  <DialogTitle>筛选媒体</DialogTitle>
                  <DialogDescription>
                    调整搜索、类型、进度和排序条件后应用到当前媒体库。
                  </DialogDescription>
                </DialogHeader>
                <form
                  className="space-y-4"
                  onSubmit={(event) => {
                    event.preventDefault()
                    applyFilterDraft()
                  }}
                >
                  <DiscoveryControls
                    filters={filterDraft}
                    showSearch
                    onChange={setFilterDraft}
                  />
                  <DialogFooter>
                    <Button
                      type="button"
                      variant="ghost"
                      onClick={resetFilterDraft}
                    >
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

            <div className="mb-6 rounded-[1.5rem] border border-border/45 bg-card/62 p-3 backdrop-blur-xl lg:hidden">
              <div className="flex flex-col gap-3">
                <div className="overflow-x-auto pb-1">
                  <div className="flex min-w-max gap-1 rounded-full border border-border/50 bg-background/45 p-1">
                    {LIBRARY_TABS.map((tab) => (
                      <Button
                        key={tab.value}
                        type="button"
                        size="sm"
                        variant="ghost"
                        className={cn(
                          "h-9 rounded-full px-4 text-muted-foreground transition-all hover:text-foreground",
                          activeTab === tab.value &&
                            "bg-background/90 text-foreground shadow-sm hover:bg-background/90"
                        )}
                        onClick={() => selectTab(tab.value)}
                      >
                        {tab.label}
                      </Button>
                    ))}
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-2">
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
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        type="button"
                        variant="outline"
                        className="rounded-full"
                      >
                        <MoreHorizontalIcon className="size-4" />
                        更多
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-52">
                      <DropdownMenuLabel>媒体库操作</DropdownMenuLabel>
                      <DropdownMenuItem
                        disabled={scanChangesMutation.isPending}
                        onSelect={() => scanChangesMutation.mutate()}
                      >
                        {scanChangesMutation.isPending
                          ? "正在提交变化扫描"
                          : "扫描变化"}
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        onSelect={() => void browseQuery.refetch()}
                      >
                        重新加载内容
                      </DropdownMenuItem>
                      <DropdownMenuItem asChild>
                        <Link to="/search" search={{ q: undefined }}>
                          打开搜索
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem asChild>
                        <Link to="/settings">媒体源与设置</Link>
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </div>

            <div className="mb-6 grid gap-5 lg:grid-cols-[280px_minmax(0,1fr)] xl:grid-cols-[320px_minmax(0,1fr)]">
              <aside className="hidden min-w-0 lg:block">
                <div className="space-y-4 lg:sticky lg:top-24">
                  <div className="rounded-[1.5rem] border border-border/45 bg-card/65 p-3 backdrop-blur-xl">
                    <div className="mb-3 px-1 text-xs font-medium tracking-[0.18em] text-muted-foreground uppercase">
                      分类
                    </div>
                    <div className="space-y-1">
                      {LIBRARY_TABS.map((tab) => (
                        <Button
                          key={tab.value}
                          type="button"
                          variant="ghost"
                          className={cn(
                            "h-10 w-full justify-start rounded-xl px-3 text-sm text-muted-foreground transition hover:text-foreground",
                            activeTab === tab.value &&
                              "bg-background/90 text-foreground shadow-sm"
                          )}
                          onClick={() => selectTab(tab.value)}
                        >
                          {tab.label}
                        </Button>
                      ))}
                    </div>
                  </div>

                  <div className="rounded-[1.5rem] border border-border/45 bg-card/65 p-4 backdrop-blur-xl">
                    <div className="text-xs font-medium tracking-[0.18em] text-muted-foreground uppercase">
                      筛选与排序
                    </div>
                    <p className="mt-2 text-xs leading-6 text-muted-foreground">
                      当前分区：{activeTabLabel(activeTab)} · 共 {total} 项
                    </p>
                    <div className="mt-4 space-y-2">
                      <Button
                        type="button"
                        variant={filterDialogOpen ? "default" : "outline"}
                        className="w-full rounded-xl"
                        onClick={openFilterDialog}
                      >
                        <FilterIcon className="size-4" />
                        打开筛选器
                      </Button>
                      <Button
                        type="button"
                        variant={
                          filters.sort === "title" ? "default" : "outline"
                        }
                        className="w-full rounded-xl"
                        onClick={toggleTitleSort}
                      >
                        {filters.sortDirection === "asc" ? (
                          <ArrowDownAZIcon className="size-4" />
                        ) : (
                          <ArrowUpAZIcon className="size-4" />
                        )}
                        标题{filters.sortDirection === "asc" ? "升序" : "降序"}
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        className="w-full rounded-xl"
                        onClick={resetFilterDraft}
                      >
                        重置默认筛选
                      </Button>
                    </div>
                  </div>
                </div>
              </aside>

              <div className="min-w-0 space-y-4">
                <div className="rounded-[1.35rem] border border-border/45 bg-card/60 px-4 py-3 backdrop-blur-xl">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <Badge
                        variant="outline"
                        className="rounded-full bg-background/60"
                      >
                        {activeTabLabel(activeTab)}
                      </Badge>
                      <span className="text-sm text-muted-foreground">
                        共 {total} 项
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        className="rounded-full"
                        disabled={scanChangesMutation.isPending}
                        onClick={() => scanChangesMutation.mutate()}
                      >
                        {scanChangesMutation.isPending ? (
                          <LoaderCircleIcon className="size-4 animate-spin" />
                        ) : null}
                        扫描变化
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            className="rounded-full"
                          >
                            <MoreHorizontalIcon className="size-4" />
                            操作
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-52">
                          <DropdownMenuLabel>媒体库操作</DropdownMenuLabel>
                          <DropdownMenuItem
                            onSelect={() => void browseQuery.refetch()}
                          >
                            重新加载内容
                          </DropdownMenuItem>
                          <DropdownMenuItem asChild>
                            <Link to="/search" search={{ q: undefined }}>
                              打开搜索
                            </Link>
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem asChild>
                            <Link to="/settings">媒体源与设置</Link>
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>
                </div>

                <LibraryTabContent
                  activeTab={activeTab}
                  isLoading={
                    activeTab === "favorites"
                      ? favoritesQuery.isLoading
                      : isBrowseTab
                        ? browseQuery.isLoading && items.length === 0
                        : false
                  }
                  error={
                    activeTab === "favorites"
                      ? favoritesQuery.error?.message
                      : isBrowseTab
                        ? items.length === 0
                          ? browseQuery.error?.message
                          : undefined
                        : undefined
                  }
                  onRetry={
                    activeTab === "favorites"
                      ? () => void favoritesQuery.refetch()
                      : retryBrowse
                  }
                  items={items}
                />
              </div>
            </div>

            {isBrowseTab && total > 0 ? (
              <div className="mt-6 flex min-h-16 flex-col gap-4 rounded-[1.5rem] border border-border/45 bg-card/65 p-4 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
                {browseQuery.error && items.length > 0 ? (
                  <div className="flex flex-col items-center gap-3 text-center">
                    <span className="text-destructive">
                      {browseQuery.error.message}
                    </span>
                    <Button
                      type="button"
                      variant="outline"
                      className="rounded-full"
                      onClick={retryBrowse}
                    >
                      重新加载
                    </Button>
                  </div>
                ) : (
                  <>
                    <div className="flex flex-wrap items-center justify-center gap-3 sm:justify-start">
                      <span>
                        第 {page} / {pageCount} 页，显示 {pageStart}-{pageEnd} /{" "}
                        {total} 项
                      </span>
                      {browseQuery.isFetching ? (
                        <span className="inline-flex items-center gap-1.5">
                          <LoaderCircleIcon className="size-4 animate-spin" />
                          正在加载
                        </span>
                      ) : null}
                    </div>
                    <div className="flex flex-wrap items-center justify-center gap-2 sm:justify-end">
                      <div className="flex items-center gap-2">
                        <span className="text-xs">每页</span>
                        <Select
                          value={String(pageSize)}
                          onValueChange={changePageSize}
                        >
                          <SelectTrigger
                            size="sm"
                            className="rounded-full bg-background/70"
                          >
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
                      </div>
                      <Button
                        type="button"
                        variant="outline"
                        className="rounded-full"
                        disabled={page <= 1 || browseQuery.isFetching}
                        onClick={() =>
                          onPaginationChange({ page: Math.max(1, page - 1) })
                        }
                      >
                        上一页
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        className="rounded-full"
                        disabled={page >= pageCount || browseQuery.isFetching}
                        onClick={() =>
                          onPaginationChange({
                            page: Math.min(pageCount, page + 1),
                          })
                        }
                      >
                        下一页
                      </Button>
                    </div>
                  </>
                )}
              </div>
            ) : null}
          </div>
        </section>
      </ScrollArea>
    </div>
  )
}

function TopBarActions({
  username,
  movieCount,
  showCount,
  onLogout,
}: {
  username: string
  movieCount: number
  showCount: number
  onLogout: () => void
}) {
  return (
    <>
      <div className="flex items-center gap-2 sm:hidden">
        <Button asChild size="icon" variant="ghost">
          <Link to="/search" search={{ q: undefined }}>
            <SearchIcon className="size-4" />
            <span className="sr-only">搜索</span>
          </Link>
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="icon" variant="ghost">
              <MoreHorizontalIcon className="size-4" />
              <span className="sr-only">更多操作</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuLabel>{username}</DropdownMenuLabel>
            <DropdownMenuItem asChild>
              <Link to="/favorites">收藏</Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link to="/settings">设置</Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => void onLogout()}>
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="hidden items-center gap-2 sm:flex">
        <Badge variant="outline">电影 {movieCount}</Badge>
        <Badge variant="outline">剧集 {showCount}</Badge>
        <Button asChild size="icon" variant="ghost">
          <Link to="/search" search={{ q: undefined }}>
            <SearchIcon className="size-4" />
            <span className="sr-only">搜索</span>
          </Link>
        </Button>
        <Dialog>
          <DialogTrigger asChild>
            <Button size="icon" variant="ghost">
              <CastIcon className="size-4" />
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
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="icon" variant="ghost">
              <UserCircleIcon className="size-4" />
              <span className="sr-only">用户菜单</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuLabel>{username}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link to="/favorites">
                <HeartIcon className="size-4" />
                收藏
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link to="/settings">
                <Settings2Icon className="size-4" />
                设置
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => void onLogout()}>
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button asChild size="icon" variant="ghost">
          <Link to="/settings">
            <Settings2Icon className="size-4" />
            <span className="sr-only">进入设置</span>
          </Link>
        </Button>
      </div>
    </>
  )
}

function LibraryStatCard({
  label,
  value,
}: {
  label: string
  value: number | string
}) {
  return (
    <div className="rounded-2xl border border-border/45 bg-background/60 px-3 py-3 shadow-sm backdrop-blur sm:px-4">
      <div className="text-[11px] font-medium tracking-[0.18em] text-muted-foreground uppercase">
        {label}
      </div>
      <div className="mt-1 truncate text-2xl font-semibold tracking-tight sm:text-3xl">
        {typeof value === "number" ? value.toLocaleString() : value}
      </div>
    </div>
  )
}

function LibraryTabContent({
  activeTab,
  isLoading,
  error,
  onRetry,
  items,
}: {
  activeTab: LibraryTab
  isLoading: boolean
  error?: string
  onRetry: () => void
  items: CatalogListItem[]
}) {
  if (!isSupportedTab(activeTab)) {
    return <UnsupportedTabState tab={activeTab} />
  }
  if (isLoading) {
    return (
      <div className="flex min-h-80 items-center justify-center rounded-[2rem] border border-border/50 bg-card/70 shadow-sm backdrop-blur-xl">
        <div className="flex items-center gap-3 rounded-full border border-border/50 bg-background/70 px-5 py-3 text-sm text-muted-foreground">
          <LoaderCircleIcon className="size-4 animate-spin" />
          正在整理媒体墙
        </div>
      </div>
    )
  }
  if (error) {
    return (
      <div className="space-y-4 rounded-[2rem] border border-destructive/30 bg-destructive/10 px-6 py-10 text-center text-sm text-destructive shadow-sm backdrop-blur-xl">
        <p>{error}</p>
        <Button
          type="button"
          variant="outline"
          className="rounded-full border-destructive/30 bg-background/80 text-foreground hover:bg-background"
          onClick={onRetry}
        >
          重新加载
        </Button>
      </div>
    )
  }
  if (items.length === 0) {
    return (
      <div className="rounded-[2rem] border border-border/50 bg-card/70 px-6 py-14 text-center shadow-sm backdrop-blur-xl">
        <Badge
          className="rounded-full border-border/60 bg-background/70"
          variant="outline"
        >
          {activeTabLabel(activeTab)}
        </Badge>
        <h2 className="mt-4 text-2xl font-semibold tracking-tight">
          暂无可显示内容
        </h2>
        <p className="mx-auto mt-3 max-w-xl text-sm leading-7 text-muted-foreground">
          {activeTab === "favorites"
            ? "这个媒体库还没有收藏项目。"
            : activeTab === "episodes"
              ? "当前媒体库还没有可浏览的单集。"
              : "这个媒体库还没有匹配当前条件的内容，可以调整筛选或重新扫描媒体库。"}
        </p>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
      {items.map((item) => (
        <MediaPosterCard
          key={
            item.source_kind === "inventory_file"
              ? `file-${item.inventory_file_id}`
              : `item-${item.id}`
          }
          item={item}
          favorite={activeTab === "favorites" ? true : undefined}
          layout="grid"
        />
      ))}
    </div>
  )
}

function UnsupportedTabState({ tab }: { tab: LibraryTab }) {
  return (
    <div className="rounded-[2rem] border border-border/40 bg-card/70 px-6 py-12 text-center backdrop-blur-sm">
      <Badge className="border-border/60 bg-background/80" variant="outline">
        {activeTabLabel(tab)}
      </Badge>
      <h2 className="mt-4 text-2xl font-semibold tracking-tight">
        这个维度还没有接入
      </h2>
      <p className="mx-auto mt-3 text-sm leading-7 text-muted-foreground">
        当前版本先保留入口，等推荐、预告片、类型分面、标签、播出平台或文件夹数据稳定后再显示真实内容。
      </p>
    </div>
  )
}

function LibraryDetailLoading() {
  return (
    <div className="flex h-svh w-full items-center justify-center bg-background text-foreground">
      <div className="flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl">
        <LoaderCircleIcon className="size-4 animate-spin" />
        <span className="text-sm text-muted-foreground">正在加载媒体库</span>
      </div>
    </div>
  )
}

function parseOptionalInt(value: string) {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) ? parsed : undefined
}

function parseOptionalFloat(value: string) {
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function LibraryDetailError({ message }: { message: string }) {
  return (
    <div className="flex min-h-svh items-center justify-center bg-background px-6 text-foreground">
      <div className="rounded-[2rem] border border-border/40 bg-card/80 p-8 text-center backdrop-blur-xl">
        <Badge className="border-border/60 bg-background/80" variant="outline">
          加载失败
        </Badge>
        <h1 className="mt-4 text-3xl font-semibold tracking-tight">
          媒体库暂时不可用
        </h1>
        <p className="mt-3 text-sm leading-7 text-muted-foreground">
          {message}
        </p>
      </div>
    </div>
  )
}

function isSupportedTab(tab: LibraryTab) {
  return tab === "programs" || tab === "favorites" || tab === "episodes"
}

function activeTabLabel(tab: LibraryTab) {
  return (
    LIBRARY_TABS.find((candidate) => candidate.value === tab)?.label ?? "节目"
  )
}
