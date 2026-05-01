import { useEffect, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  CastIcon,
  HeartIcon,
  LoaderCircleIcon,
  MonitorSmartphoneIcon,
  SearchIcon,
  Settings2Icon,
  ShieldAlertIcon,
  UserCircleIcon,
} from "lucide-react"
import "swiper/css"

import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "#/components/ui/tooltip"
import { AppTopBar } from "#/components/app-top-bar"
import { SidebarTrigger } from "#/components/ui/sidebar"
import { createAuthedMiboApi, homeDataQueryOptions } from "#/lib/mibo-query"
import {
  affectedLibraryNames,
  findBlockingHomeIssue,
  healthReasonMessage,
  healthReasonTitle,
} from "#/lib/health-presentation"
import { useAuthStore } from "#/stores/auth-store"

import {
  ContinueWatchingRail,
  HeroCarousel,
  LatestLibraryRail,
  MyMediaSection,
} from "./home-sections"

export default function Home() {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const clearSession = useAuthStore((state) => state.clearSession)
  const navigate = useNavigate()
  const queryToken = token ?? "guest"

  const homeQuery = useQuery({
    ...homeDataQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })

  const data = homeQuery.data ?? {
    items: [],
    continueWatching: [],
    continueWatchingCount: 0,
    libraries: [],
    libraryCount: 0,
    latestByLibrary: [],
    healthIssues: [],
  }
  const homeBlockingIssue = findBlockingHomeIssue(data.healthIssues)
  const heroItems = data.items.slice(0, 6)
  const canLoopHeroItems = heroItems.length > 2
  const latestLibrarySections = useMemo(
    () => data.latestByLibrary.filter((section) => section.items.length > 0),
    [data.latestByLibrary]
  )
  const hasDisplayableHomeContent =
    data.items.length > 0 ||
    latestLibrarySections.length > 0 ||
    data.continueWatching.length > 0
  const movieCount = useMemo(
    () => data.items.filter((item) => item.type === "movie").length,
    [data.items]
  )
  const showCount = useMemo(
    () =>
      data.items.filter(
        (item) => item.type === "show" || item.type === "series"
      ).length,
    [data.items]
  )
  useEffect(() => {
    if (!hasHydrated || (token && user)) {
      return
    }

    void navigate({
      to: "/login",
      search: { redirect: "/" },
      replace: true,
    })
  }, [hasHydrated, navigate, token, user])

  const handleLogout = async () => {
    if (token) {
      try {
        await createAuthedMiboApi(token).logout()
      } catch {
        // Local session cleanup is still valid if the server session already expired.
      }
    }
    clearSession()
    await navigate({ to: "/login", search: { redirect: "/" }, replace: true })
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
          <div className="flex min-w-0 items-baseline gap-2">
            <div className="shrink-0 text-lg font-semibold">Mibo Home</div>
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
          {data.healthIssues.length > 0 ? (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Link
                    to="/settings/health"
                    className="relative inline-flex size-9 items-center justify-center rounded-full border border-destructive/30 bg-destructive/10 text-destructive shadow-sm transition-colors hover:bg-destructive/15 focus-visible:ring-2 focus-visible:ring-destructive/40 focus-visible:outline-none"
                  >
                    <ShieldAlertIcon className="size-4" />
                    <span className="absolute -top-0.5 -right-0.5 flex min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] leading-4 font-semibold text-destructive-foreground">
                      {data.healthIssues.length}
                    </span>
                    <span className="sr-only">查看健康中心</span>
                  </Link>
                </TooltipTrigger>
                <TooltipContent
                  side="bottom"
                  align="end"
                  sideOffset={8}
                  className="flex max-w-sm flex-col items-start gap-2 rounded-xl px-4 py-3 text-left"
                >
                  <div className="font-medium">
                    {data.healthIssues.length} 个健康问题需要关注
                  </div>
                  <div className="text-xs opacity-85">
                    {homeBlockingIssue
                      ? healthReasonTitle(homeBlockingIssue)
                      : healthReasonTitle(data.healthIssues[0])}
                  </div>
                  {homeBlockingIssue && affectedLibraryNames(homeBlockingIssue) ? (
                    <div className="text-xs opacity-75">
                      影响：{affectedLibraryNames(homeBlockingIssue)}
                    </div>
                  ) : null}
                  <div className="text-xs opacity-70">点击进入设置里的健康中心</div>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ) : null}
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
              <DropdownMenuItem asChild>
                <Link to="/settings/devices">
                  <MonitorSmartphoneIcon className="size-4" />
                  登录设备
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

        <div className="flex min-h-svh items-center justify-center px-6 pt-24 pb-8">
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

  if (!hasDisplayableHomeContent && homeBlockingIssue) {
    return (
      <div className="relative min-w-0 flex-1 bg-background text-foreground">
        {topBar}

        <div className="flex min-h-svh items-center justify-center px-6 pt-24 pb-8">
          <div className="max-w-2xl space-y-5 text-center">
            <Badge
              className="border-destructive/40 bg-destructive/10 text-destructive"
              variant="outline"
            >
              媒体库需要处理
            </Badge>
            <div className="mx-auto flex size-14 items-center justify-center rounded-full border border-destructive/30 bg-destructive/10">
              <ShieldAlertIcon className="size-6 text-destructive" />
            </div>
            <h1 className="text-4xl font-semibold tracking-tight">
              {healthReasonTitle(homeBlockingIssue)}
            </h1>
            <p className="text-sm leading-7 text-muted-foreground sm:text-base">
              {healthReasonMessage(homeBlockingIssue)}
            </p>
            {affectedLibraryNames(homeBlockingIssue) ? (
              <p className="text-sm text-muted-foreground">
                受影响媒体库：{affectedLibraryNames(homeBlockingIssue)}
              </p>
            ) : null}
            <div className="flex flex-col justify-center gap-3 sm:flex-row">
              <Button asChild>
                <Link to="/settings/health">处理问题</Link>
              </Button>
              <Button asChild variant="outline">
                <Link to="/settings/jobs">查看失败任务</Link>
              </Button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      {topBar}
      <div className="relative">
        <HeroCarousel
          heroItems={heroItems}
          canLoopHeroItems={canLoopHeroItems}
          userName={user.username}
          continueWatchingCount={data.continueWatchingCount}
          movieCount={movieCount}
          showCount={showCount}
          hasBottomOverlay={data.libraries.length > 0}
        />

        <MyMediaSection
          libraries={data.libraries}
          latestLibrarySections={latestLibrarySections}
          variant="heroOverlay"
        />
      </div>
      <ContinueWatchingRail entries={data.continueWatching} />
      <LatestLibraryRail latestLibrarySections={latestLibrarySections} />
    </div>
  )
}
