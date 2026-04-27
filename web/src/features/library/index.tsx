import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  ArrowDownAZIcon,
  ArrowUpAZIcon,
  CastIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  FilterIcon,
  HeartIcon,
  HomeIcon,
  LoaderCircleIcon,
  MoreHorizontalIcon,
  SearchIcon,
  Settings2Icon,
  UserCircleIcon,
} from 'lucide-react'

import { AppTopBar } from '#/components/app-top-bar'
import { MediaPosterCard } from '#/components/media-poster-card'
import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '#/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '#/components/ui/dropdown-menu'
import { SidebarTrigger } from '#/components/ui/sidebar'
import {
  DiscoveryControls,
  createDefaultDiscoveryFilters,
  type DiscoveryFilters,
} from '#/features/discovery/controls'
import type { CatalogListItem, CatalogUserItemEntry } from '#/lib/mibo-api'
import { formatMediaCardTitle } from '#/lib/media-presentation'
import {
  createAuthedMiboApi,
  favoritesQueryOptions,
  miboQueryKeys,
} from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

const PAGE_SIZE = 60

type LibraryTab =
  | 'programs'
  | 'recommended'
  | 'trailers'
  | 'favorites'
  | 'genres'
  | 'tags'
  | 'platforms'
  | 'episodes'
  | 'folders'

const LIBRARY_TABS: Array<{ value: LibraryTab; label: string }> = [
  { value: 'programs', label: '节目' },
  { value: 'recommended', label: '推荐' },
  { value: 'trailers', label: '预告' },
  { value: 'favorites', label: '收藏' },
  { value: 'genres', label: '类型' },
  { value: 'tags', label: '标签' },
  { value: 'platforms', label: '播出平台' },
  { value: 'episodes', label: '集' },
  { value: 'folders', label: '文件夹' },
]

export default function LibraryDetail({ libraryId }: { libraryId: number }) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const clearSession = useAuthStore((state) => state.clearSession)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const queryToken = token ?? 'guest'
  const hasValidLibraryId = Number.isFinite(libraryId) && libraryId > 0
  const [filters, setFiltersState] = useState(
    createDefaultDiscoveryFilters({ sort: 'title', sortDirection: 'asc' }),
  )
  const [activeTab, setActiveTab] = useState<LibraryTab>('programs')
  const [showFilters, setShowFilters] = useState(false)
  const [page, setPage] = useState(0)

  const setFilters = (next: DiscoveryFilters) => {
    setPage(0)
    setFiltersState(next)
  }

  const selectTab = (tab: LibraryTab) => {
    setPage(0)
    setActiveTab(tab)
  }

  const libraryQuery = useQuery({
    queryKey: miboQueryKeys.libraryDetail(queryToken, libraryId),
    enabled: hasHydrated && !!token && hasValidLibraryId,
    queryFn: async () => {
      if (!token) throw new Error('当前未登录，无法加载媒体库。')
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
    ),
    enabled:
      hasHydrated &&
      !!token &&
      hasValidLibraryId &&
      (activeTab === 'programs' || activeTab === 'episodes'),
    queryFn: async () => {
      if (!token) throw new Error('当前未登录，无法加载媒体库内容。')
      return createAuthedMiboApi(token).discoverMedia({
        scope: 'library',
        library_id: libraryId,
        q: filters.q.trim() || undefined,
        type: activeTab === 'episodes' ? 'episode' : filters.type,
        genre: filters.genre.trim() || undefined,
        region: filters.region.trim() || undefined,
        year: parseOptionalInt(filters.year),
        min_rating: parseOptionalFloat(filters.minRating),
        watched_state: filters.watchedState,
        sort: filters.sort,
        sort_direction: filters.sortDirection,
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      })
    },
  })
  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const favoriteMutation = useMutation({
    mutationFn: async ({
      item,
      favorite,
    }: {
      item: CatalogListItem
      favorite: boolean
    }) => {
      if (!token) throw new Error('当前未登录，无法更新收藏。')
      const api = createAuthedMiboApi(token)
      return favorite ? api.addFavorite(item.id) : api.removeFavorite(item.id)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.favorites(queryToken),
        }),
        queryClient.invalidateQueries({ queryKey: ['library', 'browse'] }),
      ])
    },
  })

  const libraryFavorites = useMemo(
    () =>
      (favoritesQuery.data ?? []).filter(
        (entry) => entry.item.library_id === libraryId,
      ),
    [favoritesQuery.data, libraryId],
  )
  const favoriteIds = useMemo(
    () => new Set((favoritesQuery.data ?? []).map((entry) => entry.item.id)),
    [favoritesQuery.data],
  )
  const progressByItemId = useMemo(
    () => new Map(libraryFavorites.map((entry) => [entry.item.id, entry])),
    [libraryFavorites],
  )

  const isBrowseTab = activeTab === 'programs' || activeTab === 'episodes'
  const items =
    activeTab === 'favorites'
      ? libraryFavorites.map((entry) => entry.item)
      : (browseQuery.data?.items ?? [])
  const total =
    activeTab === 'favorites' ? items.length : (browseQuery.data?.total ?? 0)
  const hasMore = isBrowseTab ? (browseQuery.data?.has_more ?? false) : false
  const pageCount = isBrowseTab ? Math.max(1, Math.ceil(total / PAGE_SIZE)) : 1
  const movieCount = items.filter((item) => item.type === 'movie').length
  const showCount = items.filter((item) => item.type !== 'movie').length
  const sections = useMemo(() => buildTitleSections(items), [items])
  const showQuickIndex =
    filters.sort === 'title' && isBrowseTab && sections.length > 2

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
      to: '/login',
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
        <div className="max-w-xl space-y-4 text-center">
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

      <section className="px-4 pb-16 pt-24 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-[1700px]">
          <div className="mb-6 space-y-5">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
              <div className="min-w-0">
                <Badge
                  className="border-border/60 bg-background/80"
                  variant="outline"
                >
                  Library Browse
                </Badge>
                <h1 className="mt-3 truncate text-4xl font-semibold tracking-tight sm:text-5xl">
                  {libraryQuery.data.name}
                </h1>
                <p className="mt-2 text-sm text-muted-foreground">
                  共 {total} 项，当前第 {page + 1} / {pageCount} 页
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  type="button"
                  variant={showFilters ? 'default' : 'outline'}
                  className="rounded-full"
                  onClick={() => setShowFilters((value) => !value)}
                >
                  <FilterIcon className="size-4" />
                  筛选
                </Button>
                <Button
                  type="button"
                  variant={filters.sort === 'title' ? 'default' : 'outline'}
                  className="rounded-full"
                  onClick={() =>
                    setFilters({
                      ...filters,
                      sort: 'title',
                      sortDirection:
                        filters.sort === 'title' &&
                        filters.sortDirection === 'asc'
                          ? 'desc'
                          : 'asc',
                    })
                  }
                >
                  {filters.sortDirection === 'asc' ? (
                    <ArrowDownAZIcon className="size-4" />
                  ) : (
                    <ArrowUpAZIcon className="size-4" />
                  )}
                  标题{filters.sortDirection === 'asc' ? '升序' : '降序'}
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
                      onSelect={() => void browseQuery.refetch()}
                    >
                      重新加载当前页
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

            <div className="overflow-x-auto pb-1">
              <div className="flex min-w-max gap-2 rounded-full border border-border/40 bg-card/60 p-1 backdrop-blur-sm">
                {LIBRARY_TABS.map((tab) => (
                  <Button
                    key={tab.value}
                    type="button"
                    size="sm"
                    variant={activeTab === tab.value ? 'default' : 'ghost'}
                    className="h-8 rounded-full px-4"
                    onClick={() => selectTab(tab.value)}
                  >
                    {tab.label}
                  </Button>
                ))}
              </div>
            </div>

            {showFilters ? (
              <DiscoveryControls
                filters={filters}
                showSearch
                onChange={setFilters}
              />
            ) : null}

            {showQuickIndex ? (
              <div className="flex gap-1 overflow-x-auto rounded-full border border-border/40 bg-card/60 p-2 text-xs text-muted-foreground lg:hidden">
                {sections.map((section) => (
                  <button
                    key={section.key}
                    type="button"
                    className="rounded-full px-3 py-1 hover:bg-accent hover:text-accent-foreground"
                    onClick={() => scrollToSection(section.id)}
                  >
                    {section.key}
                  </button>
                ))}
              </div>
            ) : null}
          </div>

          <LibraryTabContent
            activeTab={activeTab}
            isLoading={
              activeTab === 'favorites'
                ? favoritesQuery.isLoading
                : isBrowseTab
                  ? browseQuery.isLoading
                  : false
            }
            error={
              activeTab === 'favorites'
                ? favoritesQuery.error?.message
                : isBrowseTab
                  ? browseQuery.error?.message
                  : undefined
            }
            items={items}
            sections={sections}
            showSections={showQuickIndex}
            favoriteIds={favoriteIds}
            progressByItemId={progressByItemId}
            onFavoriteToggle={(item, favorite) =>
              favoriteMutation.mutate({ item, favorite })
            }
          />

          {isBrowseTab && total > 0 ? (
            <div className="mt-8 flex flex-col gap-3 rounded-[1.5rem] border border-border/40 bg-card/60 p-4 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
              <span>
                已显示 {browseQuery.data?.items.length ?? 0} / {total}，第{' '}
                {page + 1} 页
              </span>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  className="rounded-full"
                  disabled={page === 0 || browseQuery.isFetching}
                  onClick={() => setPage((value) => Math.max(0, value - 1))}
                >
                  <ChevronLeftIcon className="size-4" />
                  上一页
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="rounded-full"
                  disabled={!hasMore || browseQuery.isFetching}
                  onClick={() => setPage((value) => value + 1)}
                >
                  下一页
                  <ChevronRightIcon className="size-4" />
                </Button>
              </div>
            </div>
          ) : null}
        </div>
      </section>

      {showQuickIndex ? (
        <div className="fixed right-3 top-1/2 z-20 hidden -translate-y-1/2 flex-col gap-1 rounded-full border border-border/50 bg-background/80 p-1 text-xs shadow-xl backdrop-blur-xl lg:flex">
          {sections.map((section) => (
            <button
              key={section.key}
              type="button"
              className="flex size-7 items-center justify-center rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              onClick={() => scrollToSection(section.id)}
            >
              {section.key}
            </button>
          ))}
        </div>
      ) : null}
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

function LibraryTabContent({
  activeTab,
  isLoading,
  error,
  items,
  sections,
  showSections,
  favoriteIds,
  progressByItemId,
  onFavoriteToggle,
}: {
  activeTab: LibraryTab
  isLoading: boolean
  error?: string
  items: CatalogListItem[]
  sections: TitleSection[]
  showSections: boolean
  favoriteIds: Set<number>
  progressByItemId: Map<number, CatalogUserItemEntry>
  onFavoriteToggle: (item: CatalogListItem, favorite: boolean) => void
}) {
  if (!isSupportedTab(activeTab)) {
    return <UnsupportedTabState tab={activeTab} />
  }
  if (isLoading) {
    return (
      <div className="flex min-h-80 items-center justify-center rounded-[2rem] border border-border/40 bg-card/60">
        <LoaderCircleIcon className="size-5 animate-spin text-muted-foreground" />
      </div>
    )
  }
  if (error) {
    return (
      <div className="rounded-[2rem] border border-destructive/30 bg-destructive/10 px-6 py-8 text-sm text-destructive">
        {error}
      </div>
    )
  }
  if (items.length === 0) {
    return (
      <div className="rounded-[2rem] border border-border/40 bg-card/70 px-6 py-12 text-center text-sm text-muted-foreground backdrop-blur-sm">
        {activeTab === 'favorites'
          ? '这个媒体库还没有收藏项目。'
          : activeTab === 'episodes'
            ? '当前媒体库还没有可浏览的单集。'
            : '这个媒体库还没有匹配当前条件的内容。'}
      </div>
    )
  }

  if (!showSections) {
    return (
      <PosterGrid
        items={items}
        favoriteIds={favoriteIds}
        progressByItemId={progressByItemId}
        onFavoriteToggle={onFavoriteToggle}
      />
    )
  }

  return (
    <div className="space-y-10">
      {sections.map((section) => (
        <section key={section.key} id={section.id} className="scroll-mt-24">
          <h2 className="mb-4 text-xl font-semibold tracking-tight">
            {section.key}
          </h2>
          <PosterGrid
            items={section.items}
            favoriteIds={favoriteIds}
            progressByItemId={progressByItemId}
            onFavoriteToggle={onFavoriteToggle}
          />
        </section>
      ))}
    </div>
  )
}

function PosterGrid({
  items,
  favoriteIds,
  progressByItemId,
  onFavoriteToggle,
}: {
  items: CatalogListItem[]
  favoriteIds: Set<number>
  progressByItemId: Map<number, CatalogUserItemEntry>
  onFavoriteToggle: (item: CatalogListItem, favorite: boolean) => void
}) {
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
      {items.map((item) => (
        <MediaPosterCard
          key={item.id}
          item={item}
          progress={progressByItemId.get(item.id) ?? null}
          isFavorite={favoriteIds.has(item.id)}
          onFavoriteToggle={onFavoriteToggle}
          className="w-full"
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
      <p className="mx-auto mt-3 max-w-xl text-sm leading-7 text-muted-foreground">
        当前版本先保留入口，等推荐、预告片、类型分面、标签、播出平台或文件夹数据稳定后再显示真实内容。
      </p>
    </div>
  )
}

function LibraryDetailLoading() {
  return (
    <div className="flex min-h-svh items-center justify-center bg-background text-foreground">
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
      <div className="max-w-lg rounded-[2rem] border border-border/40 bg-card/80 p-8 text-center backdrop-blur-xl">
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

type TitleSection = {
  key: string
  id: string
  items: CatalogListItem[]
}

function buildTitleSections(items: CatalogListItem[]): TitleSection[] {
  const sections = new Map<string, CatalogListItem[]>()
  for (const item of items) {
    const key = titleSectionKey(formatMediaCardTitle(item))
    sections.set(key, [...(sections.get(key) ?? []), item])
  }
  return Array.from(sections.entries()).map(([key, sectionItems], index) => ({
    key,
    id: `library-title-section-${index}-${key.charCodeAt(0)}`,
    items: sectionItems,
  }))
}

function titleSectionKey(title: string) {
  const first = title.trim().charAt(0)
  if (!first) return '#'
  if (/\p{Number}/u.test(first)) return '#'
  if (/\p{Script=Han}/u.test(first)) return first
  if (/\p{Letter}/u.test(first)) return first.toLocaleUpperCase()
  return '符'
}

function scrollToSection(id: string) {
  document.getElementById(id)?.scrollIntoView({
    block: 'start',
    behavior: 'smooth',
  })
}

function isSupportedTab(tab: LibraryTab) {
  return tab === 'programs' || tab === 'favorites' || tab === 'episodes'
}

function activeTabLabel(tab: LibraryTab) {
  return (
    LIBRARY_TABS.find((candidate) => candidate.value === tab)?.label ?? '节目'
  )
}
