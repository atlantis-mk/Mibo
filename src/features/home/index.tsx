import { useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { LoaderCircleIcon, ShieldAlertIcon } from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { homeDataQueryOptions } from '@/lib/mibo-query'
import {
  affectedLibraryNames,
  operationTaskMessage,
  operationTaskTitle,
  operationsSeverityLabel,
} from '@/lib/operations-presentation'
import { Badge } from '@/components/ui/badge'
import { Main } from '@/components/layout/main'
import {
  ContentSectionRail,
  ContinueWatchingRail,
  HeroCarousel,
  HomeHealthToastContent,
} from './home-sections'
import { getHomeDashboardState } from './home-state'

const mobileShellViewportHeightClass =
  'min-h-[calc(100svh-(max(env(safe-area-inset-top),0.75rem)+3.25rem))] md:min-h-svh'

const mobileShellViewportExactHeightClass =
  'h-[calc(100svh-(max(env(safe-area-inset-top),0.75rem)+3.25rem))] md:h-svh'

export function HomePage() {
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const user = useAuthStore((state) => state.auth.user)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryToken = accessToken || 'guest'

  const homeQuery = useQuery({
    ...homeDataQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
  })

  const data = homeQuery.data ?? {
    items: [],
    continueWatching: [],
    continueWatchingCount: 0,
    contentSections: [],
    mediaOverview: { sections: [] },
    operationsTasks: [],
  }

  const homeState = getHomeDashboardState(data)
  const homeBlockingTask = homeState.homeBlockingTask
  const bannerTask = homeBlockingTask ?? homeState.activeOperationsTasks[0]
  const heroItems = data.items.slice(0, 6)
  const canLoopHeroItems = heroItems.length > 2
  const canAutoplayHeroItems = heroItems.length > 1
  const degradedToastIssueIdRef = useRef<string | null>(null)

  useEffect(() => {
    const toastId = 'home-health-degraded'

    if (!homeState.isPartiallyDegraded || !bannerTask) {
      degradedToastIssueIdRef.current = null
      toast.dismiss(toastId)
      return
    }

    if (degradedToastIssueIdRef.current === bannerTask.id) {
      return
    }

    degradedToastIssueIdRef.current = bannerTask.id
    toast.custom(() => <HomeHealthToastContent task={bannerTask} />, {
      id: toastId,
      duration: 10000,
      position: 'top-right',
    })
  }, [bannerTask, homeState.isPartiallyDegraded])

  if (!hasHydrated || (accessToken && homeQuery.isLoading)) {
    return (
      <div
        className={`flex w-full items-center justify-center bg-background text-foreground ${mobileShellViewportExactHeightClass}`}
      >
        <div className='flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl'>
          <LoaderCircleIcon className='size-4 animate-spin' />
          <span className='text-sm text-muted-foreground'>
            正在加载首页内容
          </span>
        </div>
      </div>
    )
  }

  if (!accessToken || !user) {
    return (
      <div
        className={`flex w-full items-center justify-center bg-background text-foreground ${mobileShellViewportExactHeightClass}`}
      >
        <div className='flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl'>
          <LoaderCircleIcon className='size-4 animate-spin' />
          <span className='text-sm text-muted-foreground'>
            正在恢复登录状态
          </span>
        </div>
      </div>
    )
  }

  if (homeQuery.error) {
    return (
      <Main
        className={`flex items-center justify-center bg-background px-6 text-foreground ${mobileShellViewportHeightClass}`}
        fluid
      >
        <div className='max-w-lg rounded-[2rem] border border-border/40 bg-card/80 p-8 text-center backdrop-blur-xl'>
          <Badge
            className='border-border/60 bg-background/80'
            variant='outline'
          >
            加载失败
          </Badge>
          <h1 className='mt-4 text-3xl font-semibold tracking-tight'>
            首页内容暂时不可用
          </h1>
          <p className='mt-3 text-sm leading-7 text-muted-foreground'>
            {homeQuery.error.message}
          </p>
        </div>
      </Main>
    )
  }

  if (homeState.isHealthBlocked && homeBlockingTask) {
    return (
      <Main
        className={`flex items-center justify-center bg-background px-6 py-8 text-foreground ${mobileShellViewportHeightClass}`}
        fluid
      >
        <div className='max-w-2xl space-y-5 text-center'>
          <Badge
            className='border-destructive/40 bg-destructive/10 text-destructive'
            variant='outline'
          >
            {operationsSeverityLabel(homeBlockingTask.severity)}
          </Badge>
          <div className='mx-auto flex size-14 items-center justify-center rounded-full border border-destructive/30 bg-destructive/10'>
            <ShieldAlertIcon className='size-6 text-destructive' />
          </div>
          <h1 className='text-4xl font-semibold tracking-tight'>
            {operationTaskTitle(homeBlockingTask)}
          </h1>
          <p className='text-sm leading-7 text-muted-foreground sm:text-base'>
            {operationTaskMessage(homeBlockingTask)}
          </p>
          {affectedLibraryNames(homeBlockingTask) ? (
            <p className='text-sm text-muted-foreground'>
              受影响来源：{affectedLibraryNames(homeBlockingTask)}
            </p>
          ) : null}
        </div>
      </Main>
    )
  }

  if (homeState.hasEmptySetupState) {
    return (
      <Main
        className={`flex items-center justify-center bg-background px-6 py-8 text-foreground ${mobileShellViewportHeightClass}`}
        fluid
      >
        <div className='max-w-xl space-y-4 text-center'>
          <Badge
            className='border-border/60 bg-background/80'
            variant='outline'
          >
            首页已就绪
          </Badge>
          <h1 className='text-4xl font-semibold tracking-tight'>
            还没有可轮播的媒体内容
          </h1>
          <p className='text-sm leading-7 text-muted-foreground sm:text-base'>
            等后端扫描到最近加入的影片或剧集后，这里会自动切换成全屏轮播首页。
          </p>
        </div>
      </Main>
    )
  }

  return (
    <Main className='bg-background px-0 py-0 text-foreground' fluid>
      <div className='relative'>
        <HeroCarousel
          heroItems={heroItems}
          canAutoplayHeroItems={canAutoplayHeroItems}
          canLoopHeroItems={canLoopHeroItems}
        />
      </div>
      <ContinueWatchingRail entries={data.continueWatching} />
      <ContentSectionRail contentSections={homeState.contentSections} />
    </Main>
  )
}
