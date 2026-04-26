import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { LoaderCircleIcon } from 'lucide-react'

import { AppTopBar } from '#/components/app-top-bar'
import { Badge } from '#/components/ui/badge'
import { SidebarTrigger } from '#/components/ui/sidebar'
import {
  DiscoveryControls,
  createDefaultDiscoveryFilters,
} from '#/features/discovery/controls'
import {
  formatMediaCardTitle,
  getMediaCardType,
} from '#/lib/media-presentation'
import { createAuthedMiboApi } from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

export default function SearchPage({
  initialQuery,
}: {
  initialQuery?: string
}) {
  const token = useAuthStore((state) => state.token)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const [filters, setFilters] = useState(
    createDefaultDiscoveryFilters({ q: initialQuery ?? '', sort: 'title' }),
  )

  const searchQuery = useQuery({
    queryKey: ['search', token, filters],
    enabled: hasHydrated && !!token,
    queryFn: async () => {
      if (!token) throw new Error('当前未登录，无法搜索媒体库。')
      const api = createAuthedMiboApi(token)
      const [results, history] = await Promise.all([
        api.discoverMedia({
          q: filters.q.trim() || undefined,
          type: filters.type,
          genre: filters.genre.trim() || undefined,
          region: filters.region.trim() || undefined,
          year: parseOptionalInt(filters.year),
          min_rating: parseOptionalFloat(filters.minRating),
          watched_state: filters.watchedState,
          sort: filters.sort,
          limit: 50,
        }),
        api.listSearchHistory(),
      ])
      return {
        items: results.items,
        history,
      }
    },
  })

  if (!hasHydrated || (token && searchQuery.isLoading)) {
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

  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      <AppTopBar
        leftSlot={
          <>
            <SidebarTrigger className="rounded-full border border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground" />
            <div className="min-w-0">
              <div className="truncate text-sm font-medium">全局搜索</div>
              <div className="truncate text-xs text-muted-foreground">
                标题 / 原始标题 / 演员 / 导演
              </div>
            </div>
          </>
        }
        rightSlot={
          <Badge
            className="border-border/50 bg-background/80"
            variant="outline"
          >
            {data.items.length} 条结果
          </Badge>
        }
      />

      <section className="px-4 pb-16 pt-24 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-[1600px] space-y-6">
          <DiscoveryControls filters={filters} onChange={setFilters} />

          {data.history.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {data.history.map((entry) => (
                <button
                  key={entry.id}
                  type="button"
                  onClick={() =>
                    setFilters((current) => ({
                      ...current,
                      q: entry.query,
                      type: normalizeType(entry.type_filter),
                      genre: entry.genre,
                      region: entry.region,
                      year: entry.year ? String(entry.year) : '',
                      minRating: entry.min_rating
                        ? String(entry.min_rating)
                        : '',
                      watchedState: normalizeWatchedState(entry.watched_state),
                      sort: entry.sort,
                    }))
                  }
                  className="rounded-full border border-border/50 bg-card/70 px-3 py-1 text-xs text-muted-foreground hover:text-foreground"
                >
                  最近搜索: {entry.query}
                </button>
              ))}
            </div>
          ) : null}

          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {data.items.map((result) => {
              const item = 'item' in result ? result.item : result
              const watchedState =
                'watched_state' in result ? result.watched_state : ''
              const highlight =
                'highlight' in result && typeof result.highlight === 'string'
                  ? result.highlight
                  : ''
              return (
                <Link
                  key={item.id}
                  to="/media/$id"
                  params={{ id: String(item.id) }}
                  search={{
                    view:
                      getMediaCardType(item) === 'show' ? 'series' : undefined,
                  }}
                  className="rounded-[1.5rem] border border-border/40 bg-card/70 p-4 backdrop-blur-sm"
                >
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Badge
                      variant="outline"
                      className="border-border/50 bg-background/80"
                    >
                      {getMediaCardType(item) === 'movie' ? '电影' : '剧集'}
                    </Badge>
                    <span>{item.year ?? '未知年份'}</span>
                    <span>{formatWatchedState(watchedState)}</span>
                  </div>
                  <div className="mt-3 text-xl font-semibold tracking-tight">
                    {formatMediaCardTitle(item)}
                  </div>
                  {highlight ? (
                    <p className="mt-2 text-sm text-muted-foreground">
                      命中片段: {highlight}
                    </p>
                  ) : null}
                </Link>
              )
            })}
          </div>
        </div>
      </section>
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

function normalizeType(value: string) {
  return value === 'movie' || value === 'show' ? value : 'all'
}

function normalizeWatchedState(value: string) {
  return value === 'unwatched' || value === 'in_progress' || value === 'watched'
    ? value
    : 'all'
}

function formatWatchedState(value: string) {
  switch (value) {
    case 'watched':
      return '已看'
    case 'in_progress':
      return '观看中'
    default:
      return '未看'
  }
}
