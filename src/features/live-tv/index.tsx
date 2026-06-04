import { useMemo, useState } from 'react'
import {
  keepPreviousData,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  CalendarDaysIcon,
  ChevronRightIcon,
  LoaderCircleIcon,
  PlayIcon,
  RadioIcon,
  SearchIcon,
  TvIcon,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import {
  liveTVChannelGroupsQueryOptions,
  liveTVChannelsQueryOptions,
  liveTVPlaybackQueryOptions,
  liveTVProgramsQueryOptions,
  liveTVSourcesQueryOptions,
} from '@/lib/mibo-query'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import {
  getPlaybackLaunchPreferences,
  openConfiguredExternalPlayer,
} from '@/features/play/external-player'
import { LiveTVVideo } from './live-tv-video'

type LiveTVTab = 'programs' | 'guide' | 'channels'

const liveTVTabs: { value: LiveTVTab; label: string }[] = [
  { value: 'programs', label: '节目' },
  { value: 'guide', label: '指南' },
  { value: 'channels', label: '频道' },
]

const liveTVProgramPageSize = 60

export function LiveTVIndexPage() {
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryClient = useQueryClient()
  const queryToken = accessToken ?? 'guest'
  const [sourceFilter, setSourceFilter] = useState('all')
  const [groupFilter, setGroupFilter] = useState('all')
  const [activeTab, setActiveTab] = useState<LiveTVTab>('programs')
  const [programPageCount, setProgramPageCount] = useState(1)
  const [channelQuery, setChannelQuery] = useState('')
  const trimmedChannelQuery = channelQuery.trim()

  const sourcesQuery = useQuery({
    ...liveTVSourcesQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
  })
  const channelGroupsQuery = useQuery({
    ...liveTVChannelGroupsQueryOptions(queryToken, {
      source_id: sourceFilter !== 'all' ? Number(sourceFilter) : undefined,
      q: trimmedChannelQuery || undefined,
      enabled: true,
    }),
    enabled: hasHydrated && !!accessToken && activeTab === 'channels',
  })
  const channelsQuery = useQuery({
    ...liveTVChannelsQueryOptions(queryToken, {
      source_id: sourceFilter !== 'all' ? Number(sourceFilter) : undefined,
      group: groupFilter !== 'all' ? groupFilter : undefined,
      q: trimmedChannelQuery || undefined,
      enabled: true,
    }),
    enabled: hasHydrated && !!accessToken && activeTab === 'channels',
  })
  const programsQuery = useQuery({
    ...liveTVProgramsQueryOptions(queryToken, {
      current: true,
      limit: programPageCount * liveTVProgramPageSize,
    }),
    enabled: hasHydrated && !!accessToken && activeTab === 'programs',
    placeholderData: keepPreviousData,
  })

  if (!hasHydrated || !accessToken) {
    return (
      <div className='flex min-h-svh w-full items-center justify-center bg-background text-foreground'>
        <div className='flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl'>
          <LoaderCircleIcon className='size-4 animate-spin' />
          <span className='text-sm text-muted-foreground'>
            正在恢复登录状态
          </span>
        </div>
      </div>
    )
  }

  const sources = sourcesQuery.data ?? []
  const channelGroups = channelGroupsQuery.data ?? []
  const channels = channelsQuery.data ?? []
  const programs = programsQuery.data?.items ?? []
  const handleLiveTVPlaybackClick = async (
    event: { preventDefault(): void },
    input: { channelId: number; fallbackTitle: string }
  ) => {
    const launchPreferences = getPlaybackLaunchPreferences()
    if (launchPreferences.mode !== 'external') {
      return
    }

    event.preventDefault()

    if (!accessToken) {
      toast.error('当前未登录，无法获取外部播放器播放链接。')
      return
    }

    try {
      const playbackSource = await queryClient.fetchQuery(
        liveTVPlaybackQueryOptions(queryToken, input.channelId)
      )
      const launchResult = openConfiguredExternalPlayer({
        playbackUrl: playbackSource.url,
        title: playbackSource.title || input.fallbackTitle,
      })

      if (!launchResult.ok) {
        toast.error(launchResult.message)
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : '无法获取外部播放器播放链接'
      )
    }
  }

  return (
    <>
      <Header className='h-auto border-b border-border/50 bg-background/80 backdrop-blur-xl'>
        <div className='flex min-w-0 flex-1 flex-col gap-3 py-1'>
          <div className='min-w-0'>
            <p className='text-xs tracking-[0.28em] text-muted-foreground uppercase'>
              Live TV
            </p>
            <h1 className='truncate text-lg font-semibold text-foreground'>
              电视直播
            </h1>
          </div>
          <div className='-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0'>
            <div className='flex min-w-max gap-6'>
              {liveTVTabs.map((tab) => {
                const active = activeTab === tab.value
                return (
                  <button
                    key={tab.value}
                    type='button'
                    aria-pressed={active}
                    onClick={() => setActiveTab(tab.value)}
                    className={
                      active
                        ? 'border-b-2 border-primary px-1 pb-2 text-sm font-medium text-foreground'
                        : 'border-b-2 border-transparent px-1 pb-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground'
                    }
                  >
                    {tab.label}
                  </button>
                )
              })}
            </div>
          </div>
        </div>
      </Header>

      <Main className='space-y-6 bg-background px-4 py-5 text-foreground sm:px-6'>
        {activeTab === 'programs' ? (
          <div className='space-y-5'>
            {programsQuery.error ? (
              <Alert variant='destructive'>
                <AlertTitle>加载节目失败</AlertTitle>
                <AlertDescription>
                  {programsQuery.error.message}
                </AlertDescription>
              </Alert>
            ) : null}

            {programsQuery.isLoading && programs.length === 0 ? (
              <div className='flex min-h-72 items-center justify-center gap-3 rounded-[1.25rem] border border-border/60 bg-muted/20 text-sm text-muted-foreground'>
                <LoaderCircleIcon className='size-4 animate-spin' />
                正在加载节目
              </div>
            ) : programs.length === 0 ? (
              <Empty className='min-h-72 rounded-[1.25rem] border border-dashed border-border/70 bg-muted/20'>
                <EmptyHeader>
                  <EmptyMedia variant='icon'>
                    <CalendarDaysIcon className='size-4' />
                  </EmptyMedia>
                  <EmptyTitle>当前没有正在播的节目。</EmptyTitle>
                  <EmptyDescription>
                    可以在直播源设置中启用并刷新指南数据，或稍后再查看。
                  </EmptyDescription>
                </EmptyHeader>
                <Button asChild variant='outline'>
                  <Link to='/settings/live-tv'>前往直播源设置</Link>
                </Button>
              </Empty>
            ) : (
              <div className='grid gap-4 lg:grid-cols-2 2xl:grid-cols-3'>
                {programs.map((program) => {
                  const sourceName =
                    sources.find((source) => source.id === program.source_id)
                      ?.name ?? '直播源'

                  return (
                    <Card
                      key={program.id}
                      className='rounded-[1.35rem] border-border/60 bg-background/70 transition-colors hover:border-primary/40'
                    >
                      <CardContent className='space-y-4 px-5 py-5'>
                        <div className='space-y-3'>
                          <div className='flex flex-wrap items-center gap-2 text-xs text-muted-foreground'>
                            <span className='rounded-full border border-border/60 bg-muted/30 px-2.5 py-1'>
                              {sourceName}
                            </span>
                            {program.channel_name ? (
                              <span className='rounded-full border border-border/60 bg-muted/30 px-2.5 py-1'>
                                {program.channel_name}
                              </span>
                            ) : null}
                            {program.group_name ? (
                              <span className='rounded-full border border-border/60 bg-muted/30 px-2.5 py-1'>
                                {program.group_name}
                              </span>
                            ) : null}
                          </div>
                          <div>
                            <h3 className='line-clamp-1 text-lg font-semibold'>
                              {program.title}
                            </h3>
                            <p className='mt-1 line-clamp-2 text-sm text-muted-foreground'>
                              {program.subtitle ||
                                program.description ||
                                '未提供节目简介'}
                            </p>
                          </div>
                          <div className='text-sm text-muted-foreground'>
                            {formatProgramSchedule(
                              program.start_at,
                              program.end_at
                            )}
                          </div>
                        </div>

                        <Button asChild className='w-full'>
                          <Link
                            to='/play/$id'
                            params={{ id: String(program.channel_id) }}
                            search={{
                              fromStart: undefined,
                              inventoryFileId: undefined,
                              resourceId: undefined,
                              liveChannelId: program.channel_id,
                              liveSourceId: program.source_id,
                            }}
                            onClick={(event) => {
                              void handleLiveTVPlaybackClick(event, {
                                channelId: program.channel_id,
                                fallbackTitle: program.title,
                              })
                            }}
                          >
                            <PlayIcon className='size-4' />
                            播放频道
                            <ChevronRightIcon className='ml-auto size-4' />
                          </Link>
                        </Button>
                      </CardContent>
                    </Card>
                  )
                })}
              </div>
            )}
            {programsQuery.data?.has_more ? (
              <div className='flex justify-center pt-1'>
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => setProgramPageCount((current) => current + 1)}
                  disabled={programsQuery.isFetching}
                >
                  {programsQuery.isFetching ? (
                    <LoaderCircleIcon className='size-4 animate-spin' />
                  ) : null}
                  加载更多正在播节目
                </Button>
              </div>
            ) : null}
          </div>
        ) : activeTab === 'guide' ? (
          <Empty className='min-h-96 rounded-[1.75rem] border border-dashed border-border/70 bg-card/70'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <CalendarDaysIcon className='size-4' />
              </EmptyMedia>
              <EmptyTitle>指南稍后接入。</EmptyTitle>
              <EmptyDescription>
                这里先保留电视指南位置，后续可以接时间轴视图。
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='space-y-5'>
            <div className='grid gap-4 md:grid-cols-[240px_minmax(0,1fr)]'>
              <Field>
                <FieldLabel>直播源</FieldLabel>
                <Select
                  value={sourceFilter}
                  onValueChange={(value) => {
                    setSourceFilter(value)
                    setGroupFilter('all')
                  }}
                >
                  <SelectTrigger className='w-full border-border/60 bg-background text-foreground'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='all'>全部直播源</SelectItem>
                    {sources.map((source) => (
                      <SelectItem key={source.id} value={String(source.id)}>
                        {source.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
              <Field>
                <FieldLabel htmlFor='live-tv-page-search'>搜索频道</FieldLabel>
                <div className='relative'>
                  <SearchIcon className='pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground' />
                  <Input
                    id='live-tv-page-search'
                    value={channelQuery}
                    onChange={(event) => setChannelQuery(event.target.value)}
                    placeholder='频道名、分组、TVG 名称'
                    className='border-border/60 bg-background pl-10 text-foreground'
                  />
                </div>
              </Field>
            </div>

            <div className='-mx-5 overflow-x-auto px-5 pb-1'>
              <div className='flex min-w-max gap-2'>
                <Button
                  type='button'
                  variant={groupFilter === 'all' ? 'default' : 'outline'}
                  className='h-12 rounded-lg px-5'
                  onClick={() => setGroupFilter('all')}
                >
                  全部
                </Button>
                {channelGroups.map((group) => (
                  <Button
                    key={group.name}
                    type='button'
                    variant={groupFilter === group.name ? 'default' : 'outline'}
                    className='h-12 rounded-lg px-5'
                    onClick={() => setGroupFilter(group.name)}
                  >
                    <span className='truncate'>{group.name}</span>
                    <span className='ml-2 text-xs opacity-70'>
                      {group.channel_count}
                    </span>
                  </Button>
                ))}
              </div>
            </div>

            {channelsQuery.error ? (
              <Alert variant='destructive'>
                <AlertTitle>加载频道失败</AlertTitle>
                <AlertDescription>
                  {channelsQuery.error.message}
                </AlertDescription>
              </Alert>
            ) : null}

            {channelsQuery.isLoading ? (
              <div className='flex min-h-72 items-center justify-center gap-3 rounded-[1.25rem] border border-border/60 bg-muted/20 text-sm text-muted-foreground'>
                <LoaderCircleIcon className='size-4 animate-spin' />
                正在加载频道
              </div>
            ) : channels.length === 0 ? (
              <Empty className='min-h-72 rounded-[1.25rem] border border-dashed border-border/70 bg-muted/20'>
                <EmptyHeader>
                  <EmptyMedia variant='icon'>
                    <TvIcon className='size-4' />
                  </EmptyMedia>
                  <EmptyTitle>还没有可展示的频道。</EmptyTitle>
                  <EmptyDescription>
                    可以先去直播源设置页刷新频道，或者调整当前筛选条件。
                  </EmptyDescription>
                </EmptyHeader>
                <Button asChild variant='outline'>
                  <Link to='/settings/live-tv'>前往直播源设置</Link>
                </Button>
              </Empty>
            ) : (
              <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3'>
                {channels.map((channel) => {
                  const sourceName =
                    sources.find((source) => source.id === channel.source_id)
                      ?.name ?? '直播源'

                  return (
                    <Card
                      key={channel.id}
                      className='rounded-[1.35rem] border-border/60 bg-background/70 transition-colors hover:border-primary/40'
                    >
                      <CardContent className='space-y-4 px-5 py-5'>
                        <div className='space-y-3'>
                          <div className='flex flex-wrap items-center gap-2 text-xs text-muted-foreground'>
                            <span className='rounded-full border border-border/60 bg-muted/30 px-2.5 py-1'>
                              {sourceName}
                            </span>
                            {channel.group_name ? (
                              <span className='rounded-full border border-border/60 bg-muted/30 px-2.5 py-1'>
                                {channel.group_name}
                              </span>
                            ) : null}
                          </div>
                          <div>
                            <h3 className='line-clamp-1 text-lg font-semibold'>
                              {channel.name}
                            </h3>
                            <p className='mt-1 line-clamp-2 text-sm text-muted-foreground'>
                              {channel.current_program
                                ? `正在播出：${channel.current_program.title}`
                                : channel.tvg_name ||
                                  channel.tvg_id ||
                                  '未提供额外频道元数据'}
                            </p>
                          </div>
                          {channel.tags?.length ? (
                            <div className='flex flex-wrap gap-1.5'>
                              {channel.tags.map((tag) => (
                                <span
                                  key={tag}
                                  className='rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground'
                                >
                                  {tag}
                                </span>
                              ))}
                            </div>
                          ) : null}
                        </div>

                        <Button asChild className='w-full'>
                          <Link
                            to='/play/$id'
                            params={{ id: String(channel.id) }}
                            search={{
                              fromStart: undefined,
                              inventoryFileId: undefined,
                              resourceId: undefined,
                              liveChannelId: channel.id,
                              liveSourceId: channel.source_id,
                            }}
                            onClick={(event) => {
                              void handleLiveTVPlaybackClick(event, {
                                channelId: channel.id,
                                fallbackTitle: channel.name,
                              })
                            }}
                          >
                            <PlayIcon className='size-4' />
                            进入播放
                            <ChevronRightIcon className='ml-auto size-4' />
                          </Link>
                        </Button>
                      </CardContent>
                    </Card>
                  )
                })}
              </div>
            )}
          </div>
        )}
      </Main>
    </>
  )
}

function formatProgramSchedule(startAt: string, endAt: string) {
  const start = new Date(startAt)
  const end = new Date(endAt)
  if (Number.isNaN(start.valueOf()) || Number.isNaN(end.valueOf())) {
    return '播出时间未知'
  }
  const date = start.toLocaleDateString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  })
  const startTime = start.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  })
  const endTime = end.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  })
  return `${date} ${startTime} - ${endTime}`
}

type LiveTVPlaybackPageProps = {
  channelId: number
  sourceId?: number
}

export function LiveTVPlaybackPage({
  channelId,
  sourceId,
}: LiveTVPlaybackPageProps) {
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryToken = accessToken ?? 'guest'

  const playbackQuery = useQuery({
    ...liveTVPlaybackQueryOptions(queryToken, channelId),
    enabled: hasHydrated && !!accessToken && channelId > 0,
  })
  const sourceChannelsQuery = useQuery({
    ...liveTVChannelsQueryOptions(queryToken, {
      source_id: sourceId,
      enabled: true,
    }),
    enabled: hasHydrated && !!accessToken && typeof sourceId === 'number',
  })
  const allChannelsQuery = useQuery({
    ...liveTVChannelsQueryOptions(queryToken, {
      enabled: true,
    }),
    enabled: hasHydrated && !!accessToken && typeof sourceId !== 'number',
  })
  const sourcesQuery = useQuery({
    ...liveTVSourcesQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
  })

  const channels = useMemo(
    () =>
      (typeof sourceId === 'number'
        ? sourceChannelsQuery.data
        : allChannelsQuery.data) ?? [],
    [allChannelsQuery.data, sourceChannelsQuery.data, sourceId]
  )
  const currentChannel = useMemo(
    () => channels.find((channel) => channel.id === channelId) ?? null,
    [channelId, channels]
  )
  const currentSource = useMemo(
    () =>
      sourcesQuery.data?.find(
        (source) =>
          source.id === (sourceId ?? currentChannel?.source_id ?? Number.NaN)
      ) ?? null,
    [currentChannel?.source_id, sourceId, sourcesQuery.data]
  )

  if (!hasHydrated || !accessToken) {
    return (
      <div className='flex min-h-svh w-full items-center justify-center bg-background text-foreground'>
        <div className='flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl'>
          <LoaderCircleIcon className='size-4 animate-spin' />
          <span className='text-sm text-muted-foreground'>
            正在恢复登录状态
          </span>
        </div>
      </div>
    )
  }

  return (
    <>
      <Header className='border-b border-border/50 bg-background/80 backdrop-blur-xl'>
        <div className='flex min-w-0 flex-1 items-center justify-between gap-3'>
          <div className='min-w-0'>
            <p className='text-xs tracking-[0.28em] text-muted-foreground uppercase'>
              Live TV
            </p>
            <h1 className='truncate text-lg font-semibold text-foreground'>
              {playbackQuery.data?.title || currentChannel?.name || '电视直播'}
            </h1>
          </div>
          <Button asChild variant='outline' size='sm'>
            <Link to='/settings/live-tv'>管理直播源</Link>
          </Button>
        </div>
      </Header>

      <Main className='space-y-6 bg-background px-4 py-5 text-foreground sm:px-6'>
        <div className='grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_360px]'>
          <Card className='overflow-hidden rounded-[1.75rem] border-border/60 bg-card/80 shadow-sm backdrop-blur-sm'>
            <CardHeader className='space-y-3 px-5 py-5'>
              <div className='flex flex-wrap items-center gap-2 text-sm text-muted-foreground'>
                <span className='rounded-full border border-border/60 bg-background/70 px-3 py-1'>
                  {currentSource?.name || '直播频道'}
                </span>
                {playbackQuery.data?.group_name ? (
                  <span className='rounded-full border border-border/60 bg-background/70 px-3 py-1'>
                    {playbackQuery.data.group_name}
                  </span>
                ) : null}
              </div>
              <div>
                <CardTitle className='text-2xl'>
                  {playbackQuery.data?.title ||
                    currentChannel?.name ||
                    '电视直播'}
                </CardTitle>
                <CardDescription>
                  {playbackQuery.data?.current_program
                    ? `正在播出：${playbackQuery.data.current_program.title}`
                    : '通过后端代理打开直播流，避免把上游源地址直接暴露给浏览器。'}
                </CardDescription>
              </div>
            </CardHeader>
            <CardContent className='space-y-4 px-5 pb-5'>
              {playbackQuery.isLoading ? (
                <div className='flex aspect-video items-center justify-center gap-3 rounded-[1.25rem] border border-border/60 bg-muted/20 text-sm text-muted-foreground'>
                  <LoaderCircleIcon className='size-4 animate-spin' />
                  正在准备播放流
                </div>
              ) : playbackQuery.error ? (
                <Alert variant='destructive'>
                  <AlertTitle>无法播放该频道</AlertTitle>
                  <AlertDescription>
                    {playbackQuery.error.message}
                  </AlertDescription>
                </Alert>
              ) : playbackQuery.data ? (
                <div className='space-y-4'>
                  <LiveTVVideo
                    key={playbackQuery.data.url}
                    src={playbackQuery.data.url}
                    controls
                    autoPlay
                    playsInline
                    className='aspect-video w-full rounded-[1.25rem] bg-muted'
                  />
                  <div className='rounded-[1.1rem] border border-border/60 bg-muted/20 px-4 py-3 text-sm leading-6 text-muted-foreground'>
                    {playbackQuery.data.current_program ? (
                      <div className='space-y-1'>
                        <div className='font-medium text-foreground'>
                          {playbackQuery.data.current_program.title}
                        </div>
                        {playbackQuery.data.current_program.subtitle ? (
                          <div>
                            {playbackQuery.data.current_program.subtitle}
                          </div>
                        ) : null}
                        {playbackQuery.data.current_program.description ? (
                          <div>
                            {playbackQuery.data.current_program.description}
                          </div>
                        ) : null}
                      </div>
                    ) : (
                      '如果频道无法播放，通常是上游直播源失效、当前浏览器不支持该流媒体格式，或者代理请求被上游拦截。'
                    )}
                  </div>
                </div>
              ) : null}
            </CardContent>
          </Card>

          <Card className='rounded-[1.75rem] border-border/60 bg-card/80 shadow-sm backdrop-blur-sm'>
            <CardHeader className='px-5 py-5'>
              <div className='flex items-center gap-2 text-muted-foreground'>
                <TvIcon className='size-4' />
                <span className='text-sm'>
                  {currentSource?.name || '当前直播源'}频道
                </span>
              </div>
              <CardTitle className='text-xl'>切换频道</CardTitle>
              <CardDescription>
                侧边栏和这里都会展示直播频道，方便快速切换。
              </CardDescription>
            </CardHeader>
            <CardContent className='px-5 pb-5'>
              {sourceChannelsQuery.isLoading || allChannelsQuery.isLoading ? (
                <div className='flex min-h-48 items-center justify-center gap-3 rounded-[1.25rem] border border-border/60 bg-muted/20 text-sm text-muted-foreground'>
                  <LoaderCircleIcon className='size-4 animate-spin' />
                  正在加载频道列表
                </div>
              ) : channels.length === 0 ? (
                <Empty className='min-h-56 rounded-[1.25rem] border border-dashed border-border/70 bg-muted/20'>
                  <EmptyHeader>
                    <EmptyMedia variant='icon'>
                      <RadioIcon className='size-4' />
                    </EmptyMedia>
                    <EmptyTitle>没有可切换的频道。</EmptyTitle>
                    <EmptyDescription>
                      请先在直播源设置页刷新频道，再回来播放。
                    </EmptyDescription>
                  </EmptyHeader>
                </Empty>
              ) : (
                <div className='max-h-[68vh] space-y-3 overflow-y-auto pr-1'>
                  {channels.map((channel) => {
                    const isActive = channel.id === channelId

                    return (
                      <Button
                        key={channel.id}
                        asChild
                        variant={isActive ? 'default' : 'outline'}
                        className='h-auto w-full justify-start rounded-[1.2rem] px-4 py-3 text-left'
                      >
                        <Link
                          to='/live-tv/$channelId'
                          params={{ channelId: String(channel.id) }}
                          search={{ sourceId: channel.source_id }}
                        >
                          <div className='flex min-w-0 flex-1 items-center justify-between gap-3'>
                            <div className='min-w-0'>
                              <div className='truncate font-medium'>
                                {channel.name}
                              </div>
                              <div className='truncate text-xs text-muted-foreground'>
                                {channel.group_name ||
                                  channel.tvg_name ||
                                  channel.tvg_id ||
                                  '未提供元数据'}
                              </div>
                            </div>
                            {isActive ? (
                              <PlayIcon className='size-4 shrink-0' />
                            ) : null}
                          </div>
                        </Link>
                      </Button>
                    )
                  })}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </Main>
    </>
  )
}
