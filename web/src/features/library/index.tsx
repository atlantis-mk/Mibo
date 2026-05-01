import { useMemo, useRef, useState } from "react"
import { useInfiniteQuery, useQuery } from "@tanstack/react-query"
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
  SearchIcon,
  Settings2Icon,
  UserCircleIcon,
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
import { SidebarTrigger } from "#/components/ui/sidebar"
import {
  DiscoveryControls,
  createDefaultDiscoveryFilters,
  type DiscoveryFilters,
} from "#/features/discovery/controls"
import {
  createAuthedMiboApi,
  favoritesQueryOptions,
  miboQueryKeys,
} from "#/lib/mibo-query"
import type { CatalogListItem } from "#/lib/mibo-api"
import { cn } from "#/lib/utils"
import { useAuthStore } from "#/stores/auth-store"

const PAGE_SIZE = 60

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

export default function LibraryDetail({ libraryId }: { libraryId: number }) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const clearSession = useAuthStore((state) => state.clearSession)
  const navigate = useNavigate()
  const queryToken = token ?? "guest"
  const hasValidLibraryId = Number.isFinite(libraryId) && libraryId > 0
  const [filters, setFiltersState] = useState(
    createDefaultDiscoveryFilters({ sort: "title", sortDirection: "asc" })
  )
  const [filterDraft, setFilterDraft] = useState(filters)
  const [activeTab, setActiveTab] = useState<LibraryTab>("programs")
  const [filterDialogOpen, setFilterDialogOpen] = useState(false)
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)

  const setFilters = (next: DiscoveryFilters) => {
    setFiltersState(next)
    setFilterDraft(next)
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
  }

  const libraryQuery = useQuery({
    queryKey: miboQueryKeys.libraryDetail(queryToken, libraryId),
    enabled: hasHydrated && !!token && hasValidLibraryId,
    queryFn: async () => {
      if (!token) throw new Error("当前未登录，无法加载媒体库。")
      return createAuthedMiboApi(token).getLibrary(libraryId)
    },
  })
  const browseQuery = useInfiniteQuery({
    queryKey: miboQueryKeys.libraryBrowse(
      queryToken,
      libraryId,
      activeTab,
      filters,
      -1
    ),
    enabled:
      hasHydrated &&
      !!token &&
      hasValidLibraryId &&
      (activeTab === "programs" || activeTab === "episodes"),
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
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
        sort: filters.sort,
        sort_direction: filters.sortDirection,
        limit: PAGE_SIZE,
        offset: pageParam * PAGE_SIZE,
      })
    },
    getNextPageParam: (lastPage, pages) =>
      lastPage.has_more ? pages.length : undefined,
  })
  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const libraryFavorites = useMemo(
    () =>
      (favoritesQuery.data ?? []).filter(
        (entry) => entry.item.library_id === libraryId
      ),
    [favoritesQuery.data, libraryId]
  )
  const isBrowseTab = activeTab === "programs" || activeTab === "episodes"
  const browseItems = useMemo(
    () => browseQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [browseQuery.data]
  )
  const items =
    activeTab === "favorites"
      ? libraryFavorites.map((entry) => entry.item)
      : browseItems
  const total =
    activeTab === "favorites"
      ? items.length
      : (browseQuery.data?.pages[0]?.total ?? 0)
  const hasMore = isBrowseTab && browseQuery.hasNextPage
  const { fetchNextPage, isFetchingNextPage } = browseQuery
  const movieCount = items.filter((item) => item.type === "movie").length
  const showCount = items.filter((item) => item.type !== "movie").length
  const retryBrowse = () => {
    if (items.length > 0 && hasMore) {
      void fetchNextPage()
      return
    }
    void browseQuery.refetch()
  }

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
            <SidebarTrigger className="rounded-full border border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground" />
            <Button
              asChild
              size="icon-sm"
              variant="outline"
              className="hidden rounded-full border-border/50 bg-background/80 sm:inline-flex"
            >
              <Link to="/">
                <HomeIcon className="size-4" />
                <span className="sr-only">首页</span>
              </Link>
            </Button>
            <div className="min-w-0">
              <div className="truncate text-sm font-medium">
                {libraryQuery.data.name}
              </div>
              <div className="truncate text-xs text-muted-foreground">
                共 {total} 项 · {activeTabLabel(activeTab)}
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
        className="h-svh bg-[radial-gradient(circle_at_top_left,hsl(var(--primary)/0.14),transparent_32rem),radial-gradient(circle_at_top_right,hsl(var(--accent)/0.12),transparent_28rem)]"
        viewportClassName="[&>div]:block! [&>div]:w-full! [&>div]:min-w-0!"
        viewportRef={scrollContainerRef}
      >
        <section className="min-w-0 px-3 pt-22 pb-16 sm:px-6 lg:pr-20 lg:pl-8">
          <div className="mx-auto max-w-[1800px] min-w-0">
            <div className="mb-6 overflow-hidden rounded-[2rem] border border-border/50 bg-card/70 shadow-2xl shadow-black/5 backdrop-blur-xl sm:rounded-[2.5rem]">
              <div className="relative px-4 py-5 sm:px-6 sm:py-7 lg:px-8">
                <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(135deg,hsl(var(--primary)/0.12),transparent_42%,hsl(var(--accent)/0.10))]" />
                <div className="relative flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
                  <div className="min-w-0 space-y-4">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge
                        className="rounded-full border-border/60 bg-background/70 px-3 py-1 text-[11px] tracking-[0.24em] text-muted-foreground uppercase"
                        variant="outline"
                      >
                        Library
                      </Badge>
                      <Badge
                        className="rounded-full border-primary/20 bg-primary/10 px-3 py-1 text-primary"
                        variant="outline"
                      >
                        {activeTabLabel(activeTab)}
                      </Badge>
                    </div>
                    <div className="min-w-0">
                      <h1 className="truncate text-4xl font-semibold tracking-tight sm:text-6xl">
                        {libraryQuery.data.name}
                      </h1>
                      <p className="mt-3 max-w-2xl text-sm leading-7 text-muted-foreground sm:text-base">
                        浏览当前媒体库的节目、单集和收藏内容，可按类型、进度和标题排序筛选。
                      </p>
                    </div>
                  </div>
                  <div className="grid grid-cols-3 gap-2 sm:min-w-[26rem] sm:gap-3">
                    <LibraryStatCard label="总项目" value={total} />
                    <LibraryStatCard label="已加载" value={items.length} />
                    <LibraryStatCard
                      label="收藏"
                      value={libraryFavorites.length}
                    />
                  </div>
                </div>
              </div>

              <div className="relative border-t border-border/50 bg-background/55 px-3 py-3 backdrop-blur-xl sm:px-4">
                <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
                  <div className="overflow-x-auto pb-1 xl:max-w-[calc(100%-28rem)]">
                    <div className="flex min-w-max gap-1 rounded-full border border-border/50 bg-muted/35 p-1">
                      {LIBRARY_TABS.map((tab) => (
                        <Button
                          key={tab.value}
                          type="button"
                          size="sm"
                          variant="ghost"
                          className={cn(
                            "h-9 rounded-full px-4 text-muted-foreground transition-all hover:text-foreground",
                            activeTab === tab.value &&
                              "bg-background text-foreground shadow-sm hover:bg-background"
                          )}
                          onClick={() => selectTab(tab.value)}
                        >
                          {tab.label}
                        </Button>
                      ))}
                    </div>
                  </div>

                  <div className="flex flex-wrap items-center gap-2 xl:justify-end">
                    <Dialog
                      open={filterDialogOpen}
                      onOpenChange={setFilterDialogOpenState}
                    >
                      <DialogTrigger asChild>
                        <Button
                          type="button"
                          variant={filterDialogOpen ? "default" : "outline"}
                          className="rounded-full shadow-sm"
                        >
                          <FilterIcon className="size-4" />
                          筛选
                        </Button>
                      </DialogTrigger>
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
                    <Button
                      type="button"
                      variant={filters.sort === "title" ? "default" : "outline"}
                      className="rounded-full shadow-sm"
                      onClick={() =>
                        setFilters({
                          ...filters,
                          sort: "title",
                          sortDirection:
                            filters.sort === "title" &&
                            filters.sortDirection === "asc"
                              ? "desc"
                              : "asc",
                        })
                      }
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
                          className="rounded-full shadow-sm"
                        >
                          <MoreHorizontalIcon className="size-4" />
                          更多
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

            {isBrowseTab && total > 0 ? (
              <div className="mt-8 flex min-h-16 items-center justify-center rounded-[1.5rem] border border-border/40 bg-card/60 p-4 text-sm text-muted-foreground">
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
                ) : browseQuery.isFetchingNextPage ? (
                  <span className="inline-flex items-center gap-2">
                    <LoaderCircleIcon className="size-4 animate-spin" />
                    正在加载更多
                  </span>
                ) : hasMore ? (
                  <Button
                    type="button"
                    variant="outline"
                    className="rounded-full"
                    onClick={() => void fetchNextPage()}
                  >
                    加载更多，已显示 {items.length} / {total}
                  </Button>
                ) : (
                  <span>已显示全部 {total} 项</span>
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
        <Button
          asChild
          size="icon-sm"
          variant="outline"
          className="rounded-full border-border/50 bg-background/80"
        >
          <Link to="/search" search={{ q: undefined }}>
            <SearchIcon className="size-4" />
            <span className="sr-only">搜索</span>
          </Link>
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              size="icon-sm"
              variant="outline"
              className="rounded-full border-border/50 bg-background/80"
            >
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
        <Badge className="border-border/50 bg-background/80" variant="outline">
          电影 {movieCount}
        </Badge>
        <Badge className="border-border/50 bg-background/80" variant="outline">
          剧集 {showCount}
        </Badge>
        <Button
          asChild
          size="icon-sm"
          variant="outline"
          className="rounded-full border-border/50 bg-background/80"
        >
          <Link to="/search" search={{ q: undefined }}>
            <SearchIcon className="size-4" />
            <span className="sr-only">搜索</span>
          </Link>
        </Button>
        <Dialog>
          <DialogTrigger asChild>
            <Button
              size="icon-sm"
              variant="outline"
              className="rounded-full border-border/50 bg-background/80"
            >
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
            <Button
              size="icon-sm"
              variant="outline"
              className="rounded-full border-border/50 bg-background/80"
            >
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
        <Button
          asChild
          size="icon-sm"
          variant="outline"
          className="rounded-full border-border/50 bg-background/80"
        >
          <Link to="/settings">
            <Settings2Icon className="size-4" />
            <span className="sr-only">进入设置</span>
          </Link>
        </Button>
      </div>
    </>
  )
}

function LibraryStatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-2xl border border-border/50 bg-background/65 px-3 py-3 shadow-sm backdrop-blur sm:px-4">
      <div className="text-[11px] font-medium tracking-[0.18em] text-muted-foreground uppercase">
        {label}
      </div>
      <div className="mt-1 truncate text-2xl font-semibold tracking-tight sm:text-3xl">
        {value.toLocaleString()}
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
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8">
      {items.map((item) => (
        <MediaPosterCard
          key={item.id}
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
