import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { LoaderCircleIcon } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import {
  formatMediaCardTitle,
  formatMediaCardYearRange,
  getMediaCardPosterUrl,
  getMediaCardType,
} from '@/lib/media-presentation'
import type { CatalogListItem } from '@/lib/mibo-api'
import { createAuthedMiboApi } from '@/lib/mibo-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import {
  createDefaultDiscoveryFilters,
  type DiscoveryFilters,
} from '@/features/discovery/controls'

const SEARCH_PAGE_SIZE = 24

export default function SearchPage({
  initialQuery,
  initialType = 'all',
  page,
}: {
  initialQuery?: string
  initialType?: DiscoveryFilters['type']
  page: number
}) {
  const token = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const navigate = useNavigate()
  const [filters, setFilters] = useState(
    createDefaultDiscoveryFilters({
      q: initialQuery ?? '',
      type: initialType,
      sort: initialType === 'all' ? 'title' : 'recent',
    })
  )
  const urlQuery = initialQuery ?? ''
  const queryFilters = { ...filters, q: urlQuery, type: initialType }
  const searchRedirect = buildSearchRedirect(urlQuery, initialType)

  useEffect(() => {
    const nextQuery = filters.q.trim()
    if (nextQuery === urlQuery.trim()) return

    const timeoutId = window.setTimeout(() => {
      void navigate({
        to: '/search',
        search: { q: nextQuery || undefined, type: filters.type, page: 1 },
        replace: true,
      })
    }, 350)

    return () => window.clearTimeout(timeoutId)
  }, [filters.q, filters.type, navigate, urlQuery])

  const searchQuery = useQuery({
    queryKey: ['search', token, queryFilters, page],
    enabled: hasHydrated && !!token,
    queryFn: async () => {
      if (!token) throw new Error('当前未登录，无法搜索媒体库。')
      const api = createAuthedMiboApi(token)
      const query = queryFilters.q.trim()

      if (!query && queryFilters.type === 'all') {
        const recommendedItems = await api.discoverMedia({
          type: 'movie',
          sort: 'recent',
          sort_direction: 'desc',
          limit: 6,
        })

        return {
          items: [],
          recommendedItems: recommendedItems.items,
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
        limit: SEARCH_PAGE_SIZE,
        offset: (page - 1) * SEARCH_PAGE_SIZE,
      })

      return {
        items: results.items,
        recommendedItems: [],
        total: results.total,
        hasMore: results.has_more,
      }
    },
  })

  if (!hasHydrated) {
    return (
      <div className='flex min-h-svh items-center justify-center bg-background text-foreground'>
        <LoaderCircleIcon className='size-4 animate-spin' />
      </div>
    )
  }

  if (!token) {
    return (
      <div className='flex min-h-svh items-center justify-center px-6 text-center'>
        <div>
          <h1 className='text-3xl font-semibold'>登录后使用全局搜索</h1>
          <Link
            to='/sign-in'
            search={{ redirect: searchRedirect }}
            className='mt-4 inline-flex rounded-full bg-primary px-5 py-2 text-primary-foreground'
          >
            前往登录
          </Link>
        </div>
      </div>
    )
  }

  const data = searchQuery.data ?? {
    items: [],
    recommendedItems: [],
    total: 0,
    hasMore: false,
  }
  const total = data.total ?? 0
  const hasNextPage = data.hasMore ?? false
  const trimmedInputQuery = filters.q.trim()
  const trimmedQuery = urlQuery.trim()
  const totalPages = Math.max(1, Math.ceil(total / SEARCH_PAGE_SIZE))
  const hasPreviousPage = page > 1
  const pageStart = total === 0 ? 0 : (page - 1) * SEARCH_PAGE_SIZE + 1
  const pageEnd = Math.min(total, page * SEARCH_PAGE_SIZE)

  const submitSearch = () => {
    void navigate({
      to: '/search',
      search: {
        q: trimmedInputQuery || undefined,
        type: filters.type,
        page: 1,
      },
      replace: true,
    })
  }

  return (
    <div className='relative min-w-0 flex-1 bg-background text-foreground'>
      <main className='relative z-10 min-h-svh px-4 py-10 sm:px-8 sm:py-12'>
        <form
          className='mx-auto w-full max-w-[1550px]'
          onSubmit={(event) => {
            event.preventDefault()
            submitSearch()
          }}
        >
          <label htmlFor='global-search' className='sr-only'>
            搜索
          </label>
          <Input
            id='global-search'
            value={filters.q}
            onChange={(event) =>
              setFilters((current) => ({
                ...current,
                q: event.target.value,
              }))
            }
            placeholder='搜索'
            className='h-9 rounded-lg border-transparent bg-input px-3.5 text-sm text-foreground shadow-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-0 sm:h-10 sm:text-base md:text-base dark:bg-input'
          />
        </form>

        {!trimmedQuery && initialType === 'all' ? (
          <section className='mx-auto mt-6 w-full max-w-[1550px] sm:mt-7'>
            <div className='text-center'>
              <h1 className='text-2xl font-semibold tracking-tight sm:text-3xl'>
                推荐
              </h1>
              <p className='mt-2 text-sm text-muted-foreground'>
                最近加入的电影
              </p>
            </div>

            {searchQuery.isFetching && data.recommendedItems.length === 0 ? (
              <div className='mx-auto mt-8 flex min-h-32 max-w-xl items-center justify-center rounded-[1.5rem] border border-border/40 bg-card/70'>
                <LoaderCircleIcon className='size-5 animate-spin text-muted-foreground' />
              </div>
            ) : data.recommendedItems.length > 0 ? (
              <div className='mt-7 flex flex-wrap justify-center gap-x-7 gap-y-8'>
                {data.recommendedItems.map((item) => (
                  <SearchResultCard key={item.id} item={item} />
                ))}
              </div>
            ) : (
              <div className='mx-auto mt-8 max-w-xl rounded-[1.5rem] border border-border/40 bg-card/70 px-6 py-8 text-center text-sm text-muted-foreground'>
                还没有最近加入的电影可以推荐。
              </div>
            )}
          </section>
        ) : null}

        {trimmedQuery || initialType !== 'all' ? (
          <div className='mx-auto mt-7 w-full max-w-[calc(100vw-2rem)] sm:mt-8 sm:max-w-none'>
            <div className='flex items-center justify-center gap-4 text-sm font-semibold sm:text-lg'>
              <Button
                asChild
                type='button'
                variant={initialType === 'all' ? 'secondary' : 'ghost'}
              >
                <Link
                  to='/search'
                  search={{
                    q: trimmedInputQuery || undefined,
                    type: 'all',
                    page: 1,
                  }}
                >
                  热门结果
                </Link>
              </Button>
              <Button
                asChild
                type='button'
                variant={initialType === 'movie' ? 'secondary' : 'ghost'}
              >
                <Link
                  to='/search'
                  search={{
                    q: trimmedInputQuery || undefined,
                    type: 'movie',
                    page: 1,
                  }}
                >
                  电影
                </Link>
              </Button>
              <Button
                asChild
                type='button'
                variant={initialType === 'show' ? 'secondary' : 'ghost'}
              >
                <Link
                  to='/search'
                  search={{
                    q: trimmedInputQuery || undefined,
                    type: 'show',
                    page: 1,
                  }}
                >
                  剧集
                </Link>
              </Button>
            </div>
            <div className='mt-5 border-t border-border/60' />

            {searchQuery.error ? (
              <div className='mx-auto mt-8 max-w-xl rounded-[1.5rem] border border-destructive/30 bg-destructive/10 px-6 py-8 text-sm text-destructive'>
                {searchQuery.error.message}
              </div>
            ) : searchQuery.isFetching && data.items.length === 0 ? (
              <div className='mx-auto mt-8 flex min-h-32 max-w-xl items-center justify-center rounded-[1.5rem] border border-border/40 bg-card/70'>
                <LoaderCircleIcon className='size-5 animate-spin text-muted-foreground' />
              </div>
            ) : data.items.length > 0 ? (
              <>
                <div className='mt-7 flex flex-wrap justify-center gap-x-7 gap-y-8'>
                  {data.items.map((item) => (
                    <SearchResultCard key={item.id} item={item} />
                  ))}
                </div>
                <SearchResultsPagination
                  page={page}
                  totalPages={totalPages}
                  pageStart={pageStart}
                  pageEnd={pageEnd}
                  total={total}
                  hasPreviousPage={hasPreviousPage}
                  hasNextPage={hasNextPage}
                  onPageChange={(nextPage) => {
                    void navigate({
                      to: '/search',
                      search: {
                        q: trimmedQuery || undefined,
                        type: initialType,
                        page: nextPage,
                      },
                      replace: true,
                    })
                  }}
                />
              </>
            ) : (
              <div className='mx-auto mt-8 max-w-xl rounded-[1.5rem] border border-border/40 bg-card/70 px-6 py-8 text-center text-sm text-muted-foreground'>
                {trimmedQuery
                  ? `没有找到匹配“${trimmedQuery}”的内容。`
                  : '还没有这个类型的内容。'}
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
      to='/media/$id'
      params={{ id: String(item.id) }}
      search={{
        view: mediaType === 'show' ? 'series' : undefined,
        episodePage: undefined,
      }}
      className='group block w-[150px] text-center focus:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:w-[220px]'
    >
      <div className='aspect-[2/3] overflow-hidden rounded-xl bg-muted transition-transform duration-200 group-hover:-translate-y-1'>
        {posterUrl ? (
          <img
            src={posterUrl}
            alt={`${formatMediaCardTitle(item)} poster`}
            className='h-full w-full object-cover'
          />
        ) : (
          <div className='flex h-full w-full items-center justify-center px-6 text-sm text-muted-foreground'>
            暂无海报
          </div>
        )}
      </div>
      <div className='mt-3 text-base font-medium tracking-tight text-foreground sm:text-lg'>
        {formatMediaCardTitle(item)}
      </div>
      <div className='mt-1.5 text-sm text-muted-foreground sm:text-base'>
        {mediaType === 'movie' ? '影片' : '剧集'}
      </div>
      <div className='mt-1.5 text-sm text-muted-foreground sm:text-base'>
        {formatMediaCardYearRange(item)}
      </div>
    </Link>
  )
}

function SearchResultsPagination({
  page,
  totalPages,
  pageStart,
  pageEnd,
  total,
  hasPreviousPage,
  hasNextPage,
  onPageChange,
}: {
  page: number
  totalPages: number
  pageStart: number
  pageEnd: number
  total: number
  hasPreviousPage: boolean
  hasNextPage: boolean
  onPageChange: (nextPage: number) => void
}) {
  const pageNumbers = buildPageNumbers(page, totalPages)

  return (
    <div className='mt-8 flex flex-col gap-3 border-t border-border/60 pt-4 sm:flex-row sm:items-center sm:justify-between'>
      <div className='text-sm text-muted-foreground'>
        {total > 0
          ? `第 ${page} / ${totalPages} 页 · 显示 ${pageStart}-${pageEnd} / ${total}`
          : `第 ${page} / ${totalPages} 页`}
      </div>
      <Pagination className='mx-0 w-auto justify-start sm:justify-end'>
        <PaginationContent>
          <PaginationItem>
            <PaginationPrevious
              text='上一页'
              href='#'
              aria-disabled={!hasPreviousPage}
              className={
                hasPreviousPage ? undefined : 'pointer-events-none opacity-50'
              }
              onClick={(event) => {
                event.preventDefault()
                if (hasPreviousPage) {
                  onPageChange(page - 1)
                }
              }}
            />
          </PaginationItem>
          {pageNumbers.map((value, index) =>
            value === 'ellipsis' ? (
              <PaginationItem key={`ellipsis-${index}`}>
                <span className='flex size-9 items-center justify-center text-muted-foreground'>
                  ...
                </span>
              </PaginationItem>
            ) : (
              <PaginationItem key={value}>
                <PaginationLink
                  href='#'
                  isActive={value === page}
                  onClick={(event) => {
                    event.preventDefault()
                    if (value !== page) {
                      onPageChange(value)
                    }
                  }}
                >
                  {value}
                </PaginationLink>
              </PaginationItem>
            )
          )}
          <PaginationItem>
            <PaginationNext
              text='下一页'
              href='#'
              aria-disabled={!hasNextPage}
              className={
                hasNextPage ? undefined : 'pointer-events-none opacity-50'
              }
              onClick={(event) => {
                event.preventDefault()
                if (hasNextPage) {
                  onPageChange(page + 1)
                }
              }}
            />
          </PaginationItem>
        </PaginationContent>
      </Pagination>
    </div>
  )
}

function buildPageNumbers(currentPage: number, totalPages: number) {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => index + 1)
  }

  if (currentPage <= 3) {
    return [1, 2, 3, 4, 'ellipsis', totalPages] as const
  }

  if (currentPage >= totalPages - 2) {
    return [
      1,
      'ellipsis',
      totalPages - 3,
      totalPages - 2,
      totalPages - 1,
      totalPages,
    ] as const
  }

  return [
    1,
    'ellipsis',
    currentPage - 1,
    currentPage,
    currentPage + 1,
    'ellipsis',
    totalPages,
  ] as const
}

function parseOptionalInt(value: string) {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) ? parsed : undefined
}

function parseOptionalFloat(value: string) {
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function buildSearchRedirect(
  query: string | undefined,
  type: DiscoveryFilters['type']
) {
  const params = new URLSearchParams()

  if (query?.trim()) {
    params.set('q', query.trim())
  }

  if (type !== 'all') {
    params.set('type', type)
  }

  const nextSearch = params.toString()
  return nextSearch ? `/search?${nextSearch}` : '/search'
}
