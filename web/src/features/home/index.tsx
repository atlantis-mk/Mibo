import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { LoaderCircleIcon, Settings2Icon } from 'lucide-react'
import type { Swiper as SwiperType } from 'swiper/types'

import 'swiper/css'

import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { AppTopBar } from '#/components/app-top-bar'
import { SidebarTrigger } from '#/components/ui/sidebar'
import { homeDataQueryOptions } from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

import { HeroCarousel, LatestLibraryRail } from './home-sections'

export default function Home() {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const navigate = useNavigate()
  const queryToken = token ?? 'guest'

  const [swiper, setSwiper] = useState<SwiperType | null>(null)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const homeQuery = useQuery({
    ...homeDataQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })

  const data = homeQuery.data ?? {
    items: [],
    continueWatchingCount: 0,
    libraryCount: 0,
    latestByLibrary: [],
  }
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
            <Link to="/settings">
              <Settings2Icon className="size-4" />
              <span className="sr-only">进入设置</span>
            </Link>
          </Button>
        </div>
      }
    />
  )

  if (data.items.length === 0) {
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

      <LatestLibraryRail latestLibrarySections={latestLibrarySections} />
    </div>
  )
}
