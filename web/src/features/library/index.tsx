import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowUpRightIcon, LoaderCircleIcon } from 'lucide-react'

import { AppTopBar } from '#/components/app-top-bar'
import {
  DiscoveryControls,
  createDefaultDiscoveryFilters,
} from '#/features/discovery/controls'
import { Badge } from '#/components/ui/badge'
import { SidebarTrigger } from '#/components/ui/sidebar'
import { createAuthedMiboApi } from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

export default function LibraryDetail({ libraryId }: { libraryId: number }) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const hasValidLibraryId = Number.isFinite(libraryId) && libraryId > 0
  const [filters, setFilters] = useState(createDefaultDiscoveryFilters())

  const libraryQuery = useQuery({
    queryKey: ['library', 'detail', token, libraryId, filters],
    enabled: hasHydrated && !!token && hasValidLibraryId,
    queryFn: async () => {
      if (!token) {
        throw new Error('当前未登录，无法加载媒体库。')
      }

      const api = createAuthedMiboApi(token)
      const [library, items] = await Promise.all([
        api.getLibrary(libraryId),
        api.discoverMedia({
          scope: 'library',
          library_id: libraryId,
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
      ])

      return { library, items: items.items }
    },
  })

  if (!hasHydrated || (token && libraryQuery.isLoading)) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-muted-foreground">正在加载媒体库</span>
        </div>
      </div>
    )
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
          <Link
            to="/login"
            search={{ redirect: `/library/${libraryId}` }}
            className="inline-flex min-w-36 items-center justify-center rounded-full bg-primary px-6 py-3 text-sm font-medium text-primary-foreground"
          >
            前往登录
          </Link>
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

  const movieCount = libraryQuery.data.items.filter(
    (entry) => entry.item.type === 'movie',
  ).length
  const showCount = libraryQuery.data.items.filter(
    (entry) => entry.item.type !== 'movie',
  ).length

  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      <AppTopBar
        leftSlot={
          <>
            <SidebarTrigger className="rounded-full border border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground" />
            <div className="min-w-0">
              <div className="truncate text-sm font-medium">
                {libraryQuery.data.library.name}
              </div>
              <div className="truncate text-xs text-muted-foreground">
                共享 discovery contract · {libraryQuery.data.items.length}{' '}
                条内容
              </div>
            </div>
          </>
        }
        rightSlot={
          <div className="hidden items-center gap-2 sm:flex">
            <Badge
              className="border-border/50 bg-background/80"
              variant="outline"
            >
              电影 {movieCount}
            </Badge>
            <Badge
              className="border-border/50 bg-background/80"
              variant="outline"
            >
              剧集 {showCount}
            </Badge>
          </div>
        }
      />

      <section className="px-4 pb-16 pt-24 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-[1600px]">
          <div className="mb-6">
            <DiscoveryControls
              filters={filters}
              showSearch={false}
              onChange={setFilters}
            />
          </div>
          {libraryQuery.data.items.length > 0 ? (
            <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-5">
              {libraryQuery.data.items.map((entry) => {
                const item = entry.item
                return (
                  <article key={item.id} className="min-w-0">
                    <Link
                      to="/media/$id"
                      params={{ id: String(item.id) }}
                      className="group block overflow-hidden rounded-[1.75rem] border border-border/40 bg-card/70 shadow-lg transition-transform hover:-translate-y-1"
                    >
                      <div
                        className="aspect-[3/4] bg-cover bg-center bg-muted"
                        style={{
                          backgroundImage: item.poster_url
                            ? `url(${item.poster_url})`
                            : 'linear-gradient(180deg, rgba(80,92,255,0.35), rgba(15,118,110,0.35))',
                        }}
                      />
                      <div className="space-y-4 px-4 pb-4 pt-4">
                        <div>
                          <div className="line-clamp-1 text-xl font-semibold tracking-tight text-foreground">
                            {item.series_title
                              ? `${item.series_title} · ${item.title}`
                              : item.title}
                          </div>
                          <div className="mt-1 text-sm text-muted-foreground">
                            {item.year || '未知年份'}
                          </div>
                        </div>
                        <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                          <Badge
                            className="rounded-full border-border/50 bg-background/80 px-3 py-1"
                            variant="outline"
                          >
                            {formatMediaType(item.type)}
                          </Badge>
                          <Badge
                            className="rounded-full border-border/50 bg-background/80 px-3 py-1"
                            variant="outline"
                          >
                            {formatWatchedState(entry.watched_state)}
                          </Badge>
                          <span>{formatCreatedAt(item.created_at)}</span>
                          <ArrowUpRightIcon className="size-3.5 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
                        </div>
                      </div>
                    </Link>
                  </article>
                )
              })}
            </div>
          ) : (
            <div className="rounded-[2rem] border border-border/40 bg-card/70 px-6 py-8 text-sm text-muted-foreground backdrop-blur-sm">
              这个媒体库还没有内容。
            </div>
          )}
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

function formatMediaType(type: string) {
  if (type === 'movie') {
    return '电影'
  }

  if (type === 'show' || type === 'episode') {
    return '剧集'
  }

  return '媒体'
}

function formatCreatedAt(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '最近入库'
  }

  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
  })
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
