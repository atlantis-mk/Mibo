import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  CastIcon,
  HeartIcon,
  LoaderCircleIcon,
  SearchIcon,
  Settings2Icon,
  UserCircleIcon,
} from 'lucide-react'

import { AppTopBar } from '#/components/app-top-bar'
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
import { Input } from '#/components/ui/input'
import { SidebarTrigger } from '#/components/ui/sidebar'
import { createDefaultDiscoveryFilters } from '#/features/discovery/controls'
import {
  formatMediaCardYearRange,
  formatMediaCardTitle,
  getMediaCardPosterUrl,
  getMediaCardType,
} from '#/lib/media-presentation'
import type { CatalogListItem } from '#/lib/mibo-api'
import { createAuthedMiboApi } from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

const DEFAULT_RECOMMENDATION = '死侍与金刚狼'

export default function SearchPage({
  initialQuery,
}: {
  initialQuery?: string
}) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const clearSession = useAuthStore((state) => state.clearSession)
  const navigate = useNavigate()
  const [filters, setFilters] = useState(
    createDefaultDiscoveryFilters({ q: initialQuery ?? '', sort: 'title' }),
  )
  const urlQuery = initialQuery ?? ''
  const queryFilters = { ...filters, q: urlQuery }

  useEffect(() => {
    setFilters((current) => {
      if (current.q === urlQuery) return current
      return { ...current, q: urlQuery }
    })
  }, [urlQuery])

  useEffect(() => {
    const nextQuery = filters.q.trim()
    if (nextQuery === urlQuery.trim()) return

    const timeoutId = window.setTimeout(() => {
      void navigate({
        to: '/search',
        search: { q: nextQuery || undefined },
        replace: true,
      })
    }, 350)

    return () => window.clearTimeout(timeoutId)
  }, [filters.q, navigate, urlQuery])

  const searchQuery = useQuery({
    queryKey: ['search', token, queryFilters],
    enabled: hasHydrated && !!token,
    queryFn: async () => {
      if (!token) throw new Error('当前未登录，无法搜索媒体库。')
      const api = createAuthedMiboApi(token)
      const history = await api.listSearchHistory()
      const query = queryFilters.q.trim()

      if (!query) {
        return {
          items: [],
          history,
        }
      }

      const results = await api.discoverMedia({
        q: query,
        type: queryFilters.type,
        genre: queryFilters.genre.trim() || undefined,
        region: queryFilters.region.trim() || undefined,
        year: parseOptionalInt(queryFilters.year),
        min_rating: parseOptionalFloat(queryFilters.minRating),
        watched_state: queryFilters.watchedState,
        sort: queryFilters.sort,
        sort_direction: queryFilters.sortDirection,
        limit: 50,
      })

      return {
        items: results.items,
        history,
      }
    },
  })

  if (!hasHydrated) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background text-foreground">
        <LoaderCircleIcon className="size-4 animate-spin" />
      </div>
    )
  }

  if (!token) {
    return (
      <div className="flex min-h-svh items-center justify-center px-6 text-center">
        <div>
          <h1 className="text-3xl font-semibold">登录后使用全局搜索</h1>
          <Link
            to="/login"
            search={{ redirect: '/search' }}
            className="mt-4 inline-flex rounded-full bg-primary px-5 py-2 text-primary-foreground"
          >
            前往登录
          </Link>
        </div>
      </div>
    )
  }

  const data = searchQuery.data ?? { items: [], history: [] }
  const trimmedInputQuery = filters.q.trim()
  const trimmedQuery = urlQuery.trim()
  const recommendedQuery =
    data.history.find((entry) => entry.query.trim())?.query ??
    DEFAULT_RECOMMENDATION

  const submitSearch = () => {
    void navigate({
      to: '/search',
      search: { q: trimmedInputQuery || undefined },
      replace: true,
    })
  }

  const selectRecommendation = async (query: string) => {
    if (!token) return
    const nextQuery = query.trim()
    if (!nextQuery) return

    const api = createAuthedMiboApi(token)
    const results = await api.discoverMedia({
      q: nextQuery,
      sort: 'title',
      sort_direction: 'asc',
      limit: 1,
    })
    const recommendedItem = results.items[0]

    if (!recommendedItem) return

    void navigate({
      to: '/media/$id',
      params: { id: String(recommendedItem.id) },
      search: {
        view:
          getMediaCardType(recommendedItem) === 'show' ? 'series' : undefined,
      },
    })
  }

  const handleLogout = () => {
    clearSession()
    void navigate({
      to: '/login',
      search: { redirect: '/search' },
      replace: true,
    })
  }

  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      <AppTopBar
        leftSlot={
          <>
            <SidebarTrigger className="rounded-full border border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground" />
            <Button
              asChild
              size="sm"
              variant="default"
              className="hidden h-8 rounded-full px-4 sm:inline-flex"
            >
              <Link to="/">首页</Link>
            </Button>
            <Button
              asChild
              size="sm"
              variant="ghost"
              className="hidden h-8 rounded-full px-4 text-muted-foreground sm:inline-flex"
            >
              <Link to="/favorites">收藏</Link>
            </Button>
            <div className="min-w-0">
              <div className="truncate text-sm font-medium">搜索</div>
              <div className="truncate text-xs text-muted-foreground">
                推荐 · {data.items.length} 条结果
              </div>
            </div>
          </>
        }
        rightSlot={
          <SearchTopBarActions
            username={user?.username ?? '当前用户'}
            resultCount={data.items.length}
            onLogout={handleLogout}
          />
        }
      />

      <main className="relative z-10 min-h-svh px-4 pb-12 pt-24 sm:px-8 sm:pt-32">
        <form
          className="mx-auto w-full max-w-[1550px]"
          onSubmit={(event) => {
            event.preventDefault()
            submitSearch()
          }}
        >
          <label htmlFor="global-search" className="sr-only">
            搜索
          </label>
          <Input
            id="global-search"
            value={filters.q}
            onChange={(event) =>
              setFilters((current) => ({
                ...current,
                q: event.target.value,
              }))
            }
            placeholder="搜索"
            className="h-9 rounded-lg border-transparent bg-input px-3.5 text-sm text-foreground shadow-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-0 dark:bg-input sm:h-10 sm:text-base md:text-base"
          />
        </form>

        {!trimmedQuery ? (
          <section className="mt-6 text-center sm:mt-7">
            <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">
              推荐
            </h1>
            <button
              type="button"
              className="mt-6 text-base font-semibold text-primary hover:text-primary/80 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:text-xl"
              onClick={() => void selectRecommendation(recommendedQuery)}
            >
              {recommendedQuery}
            </button>
          </section>
        ) : null}

        {trimmedQuery ? (
          <div className="mx-auto mt-7 w-full max-w-[calc(100vw-2rem)] sm:mt-8 sm:max-w-none">
            <div className="flex items-center justify-center gap-4 text-sm font-semibold sm:text-lg">
              <button
                type="button"
                className="rounded-full bg-muted px-3.5 py-1.5 text-foreground sm:px-4 sm:py-2"
              >
                热门结果
              </button>
              <button
                type="button"
                className="px-1.5 py-1.5 text-muted-foreground hover:text-foreground"
              >
                影片
              </button>
            </div>
            <div className="mt-5 border-t border-border/60" />

            {searchQuery.error ? (
              <div className="mx-auto mt-8 max-w-xl rounded-[1.5rem] border border-destructive/30 bg-destructive/10 px-6 py-8 text-sm text-destructive">
                {searchQuery.error.message}
              </div>
            ) : searchQuery.isFetching && data.items.length === 0 ? (
              <div className="mx-auto mt-8 flex min-h-32 max-w-xl items-center justify-center rounded-[1.5rem] border border-border/40 bg-card/70">
                <LoaderCircleIcon className="size-5 animate-spin text-muted-foreground" />
              </div>
            ) : data.items.length > 0 ? (
              <div className="mt-7 flex flex-wrap justify-center gap-x-7 gap-y-8">
                {data.items.map((item) => (
                  <SearchResultCard key={item.id} item={item} />
                ))}
              </div>
            ) : (
              <div className="mx-auto mt-8 max-w-xl rounded-[1.5rem] border border-border/40 bg-card/70 px-6 py-8 text-center text-sm text-muted-foreground">
                没有找到匹配“{trimmedQuery}”的内容。
              </div>
            )}
          </div>
        ) : null}
      </main>
    </div>
  )
}

function SearchResultCard({ item }: { item: CatalogListItem }) {
  const mediaType = getMediaCardType(item)
  const posterUrl = getMediaCardPosterUrl(item)

  return (
    <Link
      to="/media/$id"
      params={{ id: String(item.id) }}
      search={{ view: mediaType === 'show' ? 'series' : undefined }}
      className="group block w-[150px] text-center focus:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:w-[220px]"
    >
      <div className="aspect-[2/3] overflow-hidden rounded-xl bg-muted transition-transform duration-200 group-hover:-translate-y-1">
        {posterUrl ? (
          <img
            src={posterUrl}
            alt={`${formatMediaCardTitle(item)} poster`}
            className="h-full w-full object-cover"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center px-6 text-sm text-muted-foreground">
            暂无海报
          </div>
        )}
      </div>
      <div className="mt-3 text-base font-medium tracking-tight text-foreground sm:text-lg">
        {formatMediaCardTitle(item)}
      </div>
      <div className="mt-1.5 text-sm text-muted-foreground sm:text-base">
        {mediaType === 'movie' ? '影片' : '剧集'}
      </div>
      <div className="mt-1.5 text-sm text-muted-foreground sm:text-base">
        {formatMediaCardYearRange(item)}
      </div>
    </Link>
  )
}

function SearchTopBarActions({
  username,
  resultCount,
  onLogout,
}: {
  username: string
  resultCount: number
  onLogout: () => void
}) {
  return (
    <div className="hidden items-center gap-2 sm:flex">
      <Badge className="border-border/50 bg-background/80" variant="outline">
        {username}
      </Badge>
      <Badge className="border-border/50 bg-background/80" variant="outline">
        结果 {resultCount}
      </Badge>
      <Button
        asChild
        size="icon-sm"
        variant="outline"
        className="rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
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
            className="rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
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
            className="rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
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
          <DropdownMenuItem onSelect={() => onLogout()}>
            退出登录
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <Button
        asChild
        size="icon-sm"
        variant="outline"
        className="rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
      >
        <Link to="/settings">
          <Settings2Icon className="size-4" />
          <span className="sr-only">进入设置</span>
        </Link>
      </Button>
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
