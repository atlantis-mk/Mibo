import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  CastIcon,
  HeartIcon,
  LoaderCircleIcon,
  SearchIcon,
  Settings2Icon,
  UserCircleIcon,
} from 'lucide-react'
import type { Swiper as SwiperType } from 'swiper/types'

import 'swiper/css'

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
import { AppTopBar } from '#/components/app-top-bar'
import { SidebarTrigger } from '#/components/ui/sidebar'
import type { CatalogListItem } from '#/lib/mibo-api'
import {
  createAuthedMiboApi,
  favoritesQueryOptions,
  homeDataQueryOptions,
  miboQueryKeys,
} from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

import {
  ContinueWatchingRail,
  HeroCarousel,
  LatestLibraryRail,
  MyMediaSection,
} from './home-sections'

export default function Home() {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const clearSession = useAuthStore((state) => state.clearSession)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const queryToken = token ?? 'guest'

  const [swiper, setSwiper] = useState<SwiperType | null>(null)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const homeQuery = useQuery({
    ...homeDataQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
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
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.favorites(queryToken),
      })
    },
  })

  const data = homeQuery.data ?? {
    items: [],
    continueWatching: [],
    continueWatchingCount: 0,
    libraries: [],
    libraryCount: 0,
    latestByLibrary: [],
  }
  const favoriteIds = useMemo(
    () => new Set((favoritesQuery.data ?? []).map((entry) => entry.item.id)),
    [favoritesQuery.data],
  )
  const heroItems = data.items.slice(0, 6)
  const canLoopHeroItems = heroItems.length > 2
  const latestLibrarySections = useMemo(
    () => data.latestByLibrary.filter((section) => section.items.length > 0),
    [data.latestByLibrary],
  )
  const movieCount = useMemo(
    () => data.items.filter((item) => item.type === 'movie').length,
    [data.items],
  )
  const showCount = useMemo(
    () =>
      data.items.filter(
        (item) => item.type === 'show' || item.type === 'series',
      ).length,
    [data.items],
  )

  useEffect(() => {
    if (!hasHydrated || (token && user)) {
      return
    }

    void navigate({
      to: '/login',
      search: { redirect: '/' },
      replace: true,
    })
  }, [hasHydrated, navigate, token, user])

  const scrollHeroTo = (index: number) => {
    if (!swiper) return
    if (canLoopHeroItems) {
      swiper.slideToLoop(index)
      return
    }
    swiper.slideTo(index)
  }

  const handleFavoriteToggle = (item: CatalogListItem, favorite: boolean) => {
    favoriteMutation.mutate({ item, favorite })
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
    await navigate({ to: '/login', search: { redirect: '/' }, replace: true })
  }

  if (!hasHydrated || (token && homeQuery.isLoading)) {
    return (
      <div className="flex h-svh w-full items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-muted-foreground">
            正在加载首页内容
          </span>
        </div>
      </div>
    )
  }

  if (!token || !user) {
    return (
      <div className="flex h-svh w-full items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-muted-foreground">
            正在跳转到登录页
          </span>
        </div>
      </div>
    )
  }

  if (homeQuery.error) {
    return (
      <div className="flex h-svh w-full items-center justify-center bg-background px-6 text-foreground">
        <div className="max-w-lg rounded-[2rem] border border-border/40 bg-card/80 p-8 text-center backdrop-blur-xl">
          <Badge
            className="border-border/60 bg-background/80"
            variant="outline"
          >
            加载失败
          </Badge>
          <h1 className="mt-4 text-3xl font-semibold tracking-tight">
            首页内容暂时不可用
          </h1>
          <p className="mt-3 text-sm leading-7 text-muted-foreground">
            {homeQuery.error.message}
          </p>
          <Button className="mt-6" onClick={() => void homeQuery.refetch()}>
            重新加载
          </Button>
        </div>
      </div>
    )
  }

  const topBar = (
    <AppTopBar
      leftSlot={
        <>
          <SidebarTrigger className="rounded-full border border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground" />
          <Link
            to="/"
            className="hidden text-base font-semibold tracking-tight sm:block"
          >
            Mibo
          </Link>
          <div className="hidden rounded-full border border-border/50 bg-background/80 p-1 sm:flex">
            <Button asChild size="sm" className="h-8 rounded-full px-4">
              <Link to="/">首页</Link>
            </Button>
            <Button
              asChild
              size="sm"
              variant="ghost"
              className="h-8 rounded-full px-4 text-muted-foreground"
            >
              <Link to="/favorites">收藏</Link>
            </Button>
          </div>
          <div className="min-w-0">
            <div className="truncate text-sm font-medium">Mibo Home</div>
            <div className="truncate text-xs text-muted-foreground">
              最近加入轮播 · {data.items.length} 条内容
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
            {user.username}
          </Badge>
          <Badge
            className="border-border/50 bg-background/80"
            variant="outline"
          >
            媒体库 {data.libraryCount}
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
                  设备发现和投屏控制还没有接入当前播放器。后续可以在播放能力中继续实现
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
              <DropdownMenuLabel>{user.username}</DropdownMenuLabel>
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
              <DropdownMenuItem onSelect={() => void handleLogout()}>
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
      }
    />
  )

  if (
    data.items.length === 0 &&
    data.libraries.length === 0 &&
    latestLibrarySections.length === 0 &&
    data.continueWatching.length === 0
  ) {
    return (
      <div className="relative min-w-0 flex-1 bg-background text-foreground">
        {topBar}

        <div className="flex min-h-svh items-center justify-center px-6 pb-8 pt-24">
          <div className="max-w-xl space-y-4 text-center">
            <Badge
              className="border-border/60 bg-background/80"
              variant="outline"
            >
              首页已就绪
            </Badge>
            <h1 className="text-4xl font-semibold tracking-tight">
              还没有可轮播的媒体内容
            </h1>
            <p className="text-sm leading-7 text-muted-foreground sm:text-base">
              等后端扫描到最近加入的影片或剧集后，这里会自动切换成全屏轮播首页。
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      {topBar}

      <HeroCarousel
        heroItems={heroItems}
        canLoopHeroItems={canLoopHeroItems}
        selectedIndex={selectedIndex}
        userName={user.username}
        continueWatchingCount={data.continueWatchingCount}
        movieCount={movieCount}
        showCount={showCount}
        onSwiper={(instance) => {
          setSwiper(instance)
          setSelectedIndex(instance.realIndex)
        }}
        onSlideChange={(instance) => setSelectedIndex(instance.realIndex)}
        onDotClick={scrollHeroTo}
      />

      <MyMediaSection
        libraries={data.libraries}
        latestLibrarySections={latestLibrarySections}
      />
      <ContinueWatchingRail
        entries={data.continueWatching}
        favoriteIds={favoriteIds}
        onFavoriteToggle={handleFavoriteToggle}
      />
      <LatestLibraryRail
        latestLibrarySections={latestLibrarySections}
        favoriteIds={favoriteIds}
        onFavoriteToggle={handleFavoriteToggle}
      />
    </div>
  )
}
