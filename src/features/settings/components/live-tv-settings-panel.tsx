import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CalendarDaysIcon,
  FolderSearchIcon,
  LoaderCircleIcon,
  PencilIcon,
  PlayIcon,
  RadioIcon,
  RefreshCwIcon,
  SaveIcon,
  SlidersHorizontalIcon,
  Trash2Icon,
  TvIcon,
  XIcon,
} from 'lucide-react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import type {
  CreateLiveTVSourceInput,
  LiveTVChannel,
  LiveTVSource,
  UpdateLiveTVSourceInput,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  liveTVChannelsQueryOptions,
  liveTVPlaybackQueryOptions,
  liveTVSourcesQueryOptions,
  miboQueryKeys,
} from '@/lib/mibo-query'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { LiveTVVideo } from '@/features/live-tv/live-tv-video'
import {
  getPlaybackLaunchPreferences,
  openConfiguredExternalPlayer,
} from '@/features/play/external-player'

const LIVE_TV_SETTINGS_STORAGE_KEY = 'mibo-web-live-tv-settings'

type SourceFormState = {
  name: string
  url: string
  formatHint: 'auto' | 'm3u' | 'txt'
  userAgent: string
  referrer: string
  tunerCount: string
  importGroups: string
  importGuideData: boolean
  channelImageSource: 'm3u' | 'guide'
  allowGuideMappingByNumber: boolean
  channelTags: string
}

type LiveTvAdvancedSettings = {
  bufferLimit: string
  guideDays: string
  defaultRecordingFolder: string
  movieRecordingFolder: string
  seriesRecordingFolder: string
  startPaddingMinutes: string
  stopPaddingMinutes: string
  postProcessorApp: string
  postProcessorArguments: string
}

const defaultSourceForm: SourceFormState = {
  name: '',
  url: '',
  formatHint: 'auto',
  userAgent: '',
  referrer: '',
  tunerCount: '0',
  importGroups: '',
  importGuideData: false,
  channelImageSource: 'm3u',
  allowGuideMappingByNumber: false,
  channelTags: '',
}

const defaultAdvancedSettings: LiveTvAdvancedSettings = {
  bufferLimit: 'unlimited',
  guideDays: 'auto',
  defaultRecordingFolder: '',
  movieRecordingFolder: '',
  seriesRecordingFolder: '',
  startPaddingMinutes: '0',
  stopPaddingMinutes: '0',
  postProcessorApp: '',
  postProcessorArguments: '',
}

function loadLiveTvAdvancedSettings(): LiveTvAdvancedSettings {
  if (typeof window === 'undefined') {
    return defaultAdvancedSettings
  }
  const savedSettings = window.localStorage.getItem(
    LIVE_TV_SETTINGS_STORAGE_KEY
  )
  if (!savedSettings) {
    return defaultAdvancedSettings
  }
  try {
    return {
      ...defaultAdvancedSettings,
      ...(JSON.parse(savedSettings) as Partial<LiveTvAdvancedSettings>),
    }
  } catch {
    window.localStorage.removeItem(LIVE_TV_SETTINGS_STORAGE_KEY)
    return defaultAdvancedSettings
  }
}

export function LiveTvSettingsPanel({ token }: { token: string | null }) {
  const queryClient = useQueryClient()
  const queryToken = token ?? 'guest'
  const [sourceForm, setSourceForm] =
    useState<SourceFormState>(defaultSourceForm)
  const [sourceDialogOpen, setSourceDialogOpen] = useState(false)
  const [editingSource, setEditingSource] = useState<LiveTVSource | null>(null)
  const [channelDialogSource, setChannelDialogSource] =
    useState<LiveTVSource | null>(null)
  const [channelSourceFilter, setChannelSourceFilter] = useState('all')
  const [channelQuery, setChannelQuery] = useState('')
  const [playbackChannel, setPlaybackChannel] = useState<LiveTVChannel | null>(
    null
  )
  const [activeTab, setActiveTab] = useState<
    'settings' | 'channels' | 'advanced'
  >('settings')
  const [advancedDraft, setAdvancedDraft] = useState<LiveTvAdvancedSettings>(
    loadLiveTvAdvancedSettings
  )

  const sourcesQuery = useQuery({
    ...liveTVSourcesQueryOptions(queryToken),
    enabled: !!token,
  })
  const channelsQuery = useQuery({
    ...liveTVChannelsQueryOptions(queryToken, {
      source_id:
        channelSourceFilter !== 'all' ? Number(channelSourceFilter) : undefined,
      q: channelQuery || undefined,
    }),
    enabled: !!token,
  })
  const playbackQuery = useQuery({
    ...liveTVPlaybackQueryOptions(queryToken, playbackChannel?.id ?? 0),
    enabled: !!token && !!playbackChannel,
  })

  const handlePlaybackChannelClick = async (channel: LiveTVChannel) => {
    const launchPreferences = getPlaybackLaunchPreferences()
    if (launchPreferences.mode !== 'external') {
      setPlaybackChannel(channel)
      return
    }

    if (!token) {
      toast.error('当前未登录，无法获取外部播放器播放链接。')
      return
    }

    try {
      const playbackSource = await queryClient.fetchQuery(
        liveTVPlaybackQueryOptions(queryToken, channel.id)
      )
      const launchResult = openConfiguredExternalPlayer({
        playbackUrl: playbackSource.url,
        title: playbackSource.title || channel.name,
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

  const sourceChannelsQuery = useQuery({
    ...liveTVChannelsQueryOptions(queryToken, {
      source_id: channelDialogSource?.id,
    }),
    enabled: !!token && !!channelDialogSource,
  })

  const createMutation = useMutation({
    mutationFn: async (draft: CreateLiveTVSourceInput) => {
      if (!token) throw new Error('当前未登录，无法保存直播源。')
      return createAuthedMiboApi(token).createLiveTVSource(draft)
    },
    onSuccess: () => {
      if (!token) return
      void queryClient.invalidateQueries({
        queryKey: miboQueryKeys.liveTVSources(token),
      })
      setSourceForm(defaultSourceForm)
      setSourceDialogOpen(false)
      toast.success('直播源已创建')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const updateMutation = useMutation({
    mutationFn: async ({
      sourceId,
      draft,
    }: {
      sourceId: number
      draft: UpdateLiveTVSourceInput
    }) => {
      if (!token) throw new Error('当前未登录，无法更新直播源。')
      return createAuthedMiboApi(token).updateLiveTVSource(sourceId, draft)
    },
    onSuccess: () => {
      if (!token) return
      void queryClient.invalidateQueries({
        queryKey: miboQueryKeys.liveTVSources(token),
      })
      setSourceForm(defaultSourceForm)
      setEditingSource(null)
      setSourceDialogOpen(false)
      toast.success('直播源已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const deleteMutation = useMutation({
    mutationFn: async (sourceId: number) => {
      if (!token) throw new Error('当前未登录，无法删除直播源。')
      return createAuthedMiboApi(token).deleteLiveTVSource(sourceId)
    },
    onSuccess: async (_, sourceId) => {
      if (!token) return
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.liveTVSources(token),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.liveTVChannels(token, {
            source_id:
              channelSourceFilter !== 'all'
                ? Number(channelSourceFilter)
                : undefined,
            q: channelQuery || undefined,
          }),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.liveTVChannels(token, {
            source_id: channelDialogSource?.id,
          }),
        }),
      ])
      if (editingSource?.id === sourceId) {
        setEditingSource(null)
        setSourceForm(defaultSourceForm)
      }
      if (channelDialogSource?.id === sourceId) {
        setChannelDialogSource(null)
      }
      toast.success('直播源已删除')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const refreshMutation = useMutation({
    mutationFn: async (sourceId: number) => {
      if (!token) throw new Error('当前未登录，无法刷新直播源。')
      return createAuthedMiboApi(token).refreshLiveTVSource(sourceId)
    },
    onSuccess: async (source) => {
      if (!token) return
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.liveTVSources(token),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.liveTVChannels(token, {
            source_id:
              channelSourceFilter !== 'all'
                ? Number(channelSourceFilter)
                : undefined,
            q: channelQuery || undefined,
          }),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.liveTVChannels(token, {
            source_id: source.id,
          }),
        }),
      ])
      toast.success(`已刷新 ${source.name}`)
    },
    onError: (error: Error) => toast.error(error.message),
  })

  function handleSubmitSource(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const tunerCount = Number.parseInt(sourceForm.tunerCount, 10)
    const draft = {
      name: sourceForm.name.trim(),
      url: sourceForm.url.trim(),
      format_hint: sourceForm.formatHint,
      user_agent: sourceForm.userAgent.trim(),
      referrer: sourceForm.referrer.trim(),
      tuner_count:
        Number.isFinite(tunerCount) && tunerCount > 0 ? tunerCount : 0,
      import_groups: sourceForm.importGroups.trim(),
      import_guide_data: sourceForm.importGuideData,
      channel_image_source: sourceForm.channelImageSource,
      allow_guide_mapping_by_number: sourceForm.allowGuideMappingByNumber,
      channel_tags: sourceForm.channelTags.trim(),
    } satisfies CreateLiveTVSourceInput

    if (editingSource) {
      updateMutation.mutate({
        sourceId: editingSource.id,
        draft,
      })
      return
    }

    createMutation.mutate(draft)
  }

  function handleEditSource(source: LiveTVSource) {
    setEditingSource(source)
    setSourceForm({
      name: source.name,
      url: source.url,
      formatHint: source.format_hint,
      userAgent: source.user_agent ?? '',
      referrer: source.referrer ?? '',
      tunerCount: String(source.tuner_count ?? 0),
      importGroups: source.import_groups ?? '',
      importGuideData: source.import_guide_data ?? false,
      channelImageSource: source.channel_image_source ?? 'm3u',
      allowGuideMappingByNumber: source.allow_guide_mapping_by_number ?? false,
      channelTags: source.channel_tags ?? '',
    })
    setSourceDialogOpen(true)
  }

  function handleCancelEdit() {
    setEditingSource(null)
    setSourceForm(defaultSourceForm)
    setSourceDialogOpen(false)
  }

  function handleCreateSource() {
    setEditingSource(null)
    setSourceForm(defaultSourceForm)
    setSourceDialogOpen(true)
  }

  function handleOpenSourceChannels(source: LiveTVSource) {
    setChannelDialogSource(source)
  }

  function handleSaveAdvanced(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    window.localStorage.setItem(
      LIVE_TV_SETTINGS_STORAGE_KEY,
      JSON.stringify(advancedDraft)
    )
    toast.success('电视直播高级设置已保存到当前浏览器')
  }

  function updateAdvanced<Value extends keyof LiveTvAdvancedSettings>(
    key: Value,
    value: LiveTvAdvancedSettings[Value]
  ) {
    setAdvancedDraft((current) => ({ ...current, [key]: value }))
  }

  if (!token) {
    return (
      <Alert>
        <AlertTitle>登录后可管理直播源</AlertTitle>
        <AlertDescription>
          当前页面需要管理员会话来导入直播源、刷新频道并发起播放。
        </AlertDescription>
      </Alert>
    )
  }

  const sources = sourcesQuery.data ?? []
  const channels = channelsQuery.data ?? []
  const sourceChannels = sourceChannelsQuery.data ?? []
  const channelDialogRefreshPending =
    refreshMutation.isPending &&
    refreshMutation.variables === channelDialogSource?.id

  return (
    <>
      <Tabs
        value={activeTab}
        onValueChange={(value) =>
          setActiveTab(value as 'settings' | 'channels' | 'advanced')
        }
        className='space-y-4 pb-20'
      >
        <div className='flex justify-center'>
          <TabsList className='grid w-full max-w-md grid-cols-3'>
            <TabsTrigger value='settings'>设置</TabsTrigger>
            <TabsTrigger value='channels'>频道</TabsTrigger>
            <TabsTrigger value='advanced'>高级</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value='settings' className='mt-0 space-y-4'>
          <section className='space-y-4'>
            <div className='space-y-1'>
              <h3 className='text-base font-medium text-foreground'>
                直播数据来源
              </h3>
              <p className='text-sm leading-6 text-muted-foreground'>
                导入远程 IPTV 播放列表 URL，当前支持 `.m3u` 与 `.txt`。
              </p>
            </div>
            <div className='space-y-5'>
              {sourcesQuery.error ? (
                <Alert variant='destructive'>
                  <AlertTitle>加载直播源失败</AlertTitle>
                  <AlertDescription>
                    {sourcesQuery.error.message}
                  </AlertDescription>
                </Alert>
              ) : null}

              {sourcesQuery.isLoading ? (
                <div className='flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground'>
                  <LoaderCircleIcon className='size-4 animate-spin' />
                  正在加载直播源
                </div>
              ) : sources.length === 0 ? (
                <Empty className='min-h-64 border border-dashed border-border/70 bg-muted/20'>
                  <EmptyHeader>
                    <EmptyMedia variant='icon'>
                      <RadioIcon className='size-4' />
                    </EmptyMedia>
                    <EmptyTitle>还没有直播源。</EmptyTitle>
                    <EmptyDescription>
                      添加 M3U 或 TXT 播放列表 URL
                      后，就能从后端刷新出频道列表。
                    </EmptyDescription>
                  </EmptyHeader>
                </Empty>
              ) : (
                <div className='space-y-4'>
                  {sources.map((source) => (
                    <div
                      key={source.id}
                      className='rounded-[1.25rem] border border-border/60 bg-background/70 p-4'
                    >
                      <div className='flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between'>
                        <div className='min-w-0 space-y-2'>
                          <div className='flex flex-wrap items-center gap-2'>
                            <h3 className='font-medium'>{source.name}</h3>
                            <SourceStatusChip source={source} />
                          </div>
                          <p className='font-mono text-xs break-all text-muted-foreground'>
                            {source.url}
                          </p>
                          <div className='flex flex-wrap gap-4 text-sm text-muted-foreground'>
                            <span>
                              格式：{source.format_hint.toUpperCase()}
                            </span>
                            <span>频道：{source.channel_count}</span>
                            <span>
                              上次刷新：
                              {source.last_refresh_at
                                ? new Date(
                                    source.last_refresh_at
                                  ).toLocaleString()
                                : '从未'}
                            </span>
                          </div>
                          {source.refresh.error ? (
                            <p className='text-sm text-destructive'>
                              {source.refresh.error}
                            </p>
                          ) : null}
                        </div>
                        <div className='flex flex-wrap gap-2'>
                          <Button
                            type='button'
                            variant='outline'
                            onClick={() => handleOpenSourceChannels(source)}
                          >
                            <TvIcon className='size-4' />
                            查看频道
                          </Button>
                          <Button
                            type='button'
                            variant='outline'
                            onClick={() => handleEditSource(source)}
                          >
                            <PencilIcon className='size-4' />
                            编辑
                          </Button>
                          <Button
                            type='button'
                            variant='outline'
                            onClick={() => refreshMutation.mutate(source.id)}
                            disabled={refreshMutation.isPending}
                          >
                            <RefreshCwIcon
                              className={`size-4 ${refreshMutation.isPending ? 'animate-spin' : ''}`}
                            />
                            刷新
                          </Button>
                          <Button
                            type='button'
                            variant='outline'
                            onClick={() => deleteMutation.mutate(source.id)}
                            disabled={deleteMutation.isPending}
                          >
                            <Trash2Icon className='size-4' />
                            删除
                          </Button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </section>
        </TabsContent>

        <TabsContent value='channels' className='mt-0 space-y-4'>
          <section className='space-y-4'>
            <div className='space-y-1'>
              <h3 className='text-base font-medium text-foreground'>频道</h3>
              <p className='text-sm leading-6 text-muted-foreground'>
                浏览已导入的频道并直接发起播放。
              </p>
            </div>
            <div className='space-y-4'>
              <div className='grid gap-4 md:grid-cols-[220px_minmax(0,1fr)]'>
                <Field>
                  <FieldLabel>按直播源筛选</FieldLabel>
                  <Select
                    value={channelSourceFilter}
                    onValueChange={setChannelSourceFilter}
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
                  <FieldLabel htmlFor='live-tv-channel-search'>
                    搜索频道
                  </FieldLabel>
                  <Input
                    id='live-tv-channel-search'
                    value={channelQuery}
                    onChange={(event) => setChannelQuery(event.target.value)}
                    placeholder='频道名、分组、TVG 名称'
                    className='border-border/60 bg-background text-foreground'
                  />
                </Field>
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
                <div className='flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground'>
                  <LoaderCircleIcon className='size-4 animate-spin' />
                  正在加载频道
                </div>
              ) : channels.length === 0 ? (
                <Empty className='min-h-72 border border-dashed border-border/70 bg-muted/20'>
                  <EmptyHeader>
                    <EmptyMedia variant='icon'>
                      <TvIcon className='size-4' />
                    </EmptyMedia>
                    <EmptyTitle>未找到频道。</EmptyTitle>
                    <EmptyDescription>
                      请先刷新直播源，或者调整当前筛选条件。
                    </EmptyDescription>
                  </EmptyHeader>
                  <EmptyContent>
                    <Button
                      type='button'
                      variant='outline'
                      onClick={() => {
                        const firstSource = sources[0]
                        if (!firstSource) {
                          toast.info('请先创建直播源')
                          return
                        }
                        refreshMutation.mutate(firstSource.id)
                      }}
                    >
                      <RefreshCwIcon className='size-4' />
                      刷新首个直播源
                    </Button>
                  </EmptyContent>
                </Empty>
              ) : (
                <div className='space-y-3'>
                  {channels.map((channel) => (
                    <div
                      key={channel.id}
                      className='flex flex-col gap-4 rounded-[1.25rem] border border-border/60 bg-background/70 p-4 sm:flex-row sm:items-center sm:justify-between'
                    >
                      <div className='min-w-0 space-y-1'>
                        <div className='flex flex-wrap items-center gap-2'>
                          <h3 className='font-medium'>{channel.name}</h3>
                          {channel.group_name ? (
                            <span className='rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground'>
                              {channel.group_name}
                            </span>
                          ) : null}
                        </div>
                        <p className='text-sm text-muted-foreground'>
                          {channel.current_program
                            ? `正在播出：${channel.current_program.title}`
                            : channel.tvg_name ||
                              channel.tvg_id ||
                              '未提供 TVG 元数据'}
                        </p>
                        {channel.tags?.length ? (
                          <div className='flex flex-wrap gap-1.5'>
                            {channel.tags.map((tag) => (
                              <span
                                key={tag}
                                className='rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-700'
                              >
                                {tag}
                              </span>
                            ))}
                          </div>
                        ) : null}
                      </div>
                      <Button
                        type='button'
                        className='bg-emerald-600 text-white hover:bg-emerald-700'
                        onClick={() => setPlaybackChannel(channel)}
                      >
                        <PlayIcon className='size-4' />
                        播放
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </section>
        </TabsContent>

        <TabsContent value='advanced' className='mt-0 space-y-4'>
          <section className='space-y-4'>
            <div className='space-y-1'>
              <h3 className='text-base font-medium text-foreground'>高级</h3>
              <p className='text-sm leading-6 text-muted-foreground'>
                配置直播缓存、指南下载范围、默认录制目录和录制后处理。
              </p>
            </div>
            <div className='space-y-5'>
              <div className='flex items-start gap-3 rounded-[1.15rem] border border-border/60 bg-muted/30 px-4 py-3 text-sm leading-6 text-muted-foreground'>
                <CalendarDaysIcon className='mt-0.5 size-4 shrink-0' />
                <span>
                  这些高级项目前仍作为本机草稿保存。当前变更只接入直播源管理、频道导入和播放链路，不包含
                  DVR / EPG / 后处理执行。
                </span>
              </div>

              <form
                id='live-tv-advanced-form'
                onSubmit={handleSaveAdvanced}
                className='space-y-6'
              >
                <FieldGroup>
                  <div className='grid gap-4 md:grid-cols-2'>
                    <Field>
                      <FieldLabel>直播流缓冲区尺寸限制</FieldLabel>
                      <Select
                        value={advancedDraft.bufferLimit}
                        onValueChange={(value) =>
                          updateAdvanced('bufferLimit', value)
                        }
                      >
                        <SelectTrigger className='w-full border-border/60 bg-background text-foreground'>
                          <SelectValue placeholder='选择缓冲限制' />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value='unlimited'>无限制</SelectItem>
                          <SelectItem value='1'>1 小时</SelectItem>
                          <SelectItem value='2'>2 小时</SelectItem>
                          <SelectItem value='4'>4 小时</SelectItem>
                        </SelectContent>
                      </Select>
                      <FieldDescription>
                        控制直播回看与缓存范围。缓冲越长，占用磁盘越多。
                      </FieldDescription>
                    </Field>

                    <Field>
                      <FieldLabel>指南数据下载天数</FieldLabel>
                      <Select
                        value={advancedDraft.guideDays}
                        onValueChange={(value) =>
                          updateAdvanced('guideDays', value)
                        }
                      >
                        <SelectTrigger className='w-full border-border/60 bg-background text-foreground'>
                          <SelectValue placeholder='选择下载天数' />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value='auto'>自动</SelectItem>
                          <SelectItem value='1'>1 天</SelectItem>
                          <SelectItem value='3'>3 天</SelectItem>
                          <SelectItem value='7'>7 天</SelectItem>
                          <SelectItem value='14'>14 天</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                  </div>

                  <RecordingFolderField
                    id='live-tv-default-recording-folder'
                    label='默认录制文件夹'
                    description='保存录制内容的默认媒体库位置，建议使用已创建的混合内容媒体库。'
                    value={advancedDraft.defaultRecordingFolder}
                    onChange={(value) =>
                      updateAdvanced('defaultRecordingFolder', value)
                    }
                  />

                  <div className='grid gap-4 md:grid-cols-2'>
                    <RecordingFolderField
                      id='live-tv-movie-recording-folder'
                      label='影片录制文件夹'
                      description='可选。电影类录制内容会优先保存到这里。'
                      value={advancedDraft.movieRecordingFolder}
                      onChange={(value) =>
                        updateAdvanced('movieRecordingFolder', value)
                      }
                    />
                    <RecordingFolderField
                      id='live-tv-series-recording-folder'
                      label='剧集录制文件夹'
                      description='可选。电视剧和节目类录制内容会优先保存到这里。'
                      value={advancedDraft.seriesRecordingFolder}
                      onChange={(value) =>
                        updateAdvanced('seriesRecordingFolder', value)
                      }
                    />
                  </div>

                  <div className='rounded-[1.25rem] border border-border/60 bg-muted/20 p-4'>
                    <div className='mb-4 flex items-center gap-2'>
                      <SlidersHorizontalIcon className='size-4 text-muted-foreground' />
                      <h3 className='font-medium'>默认录制设置</h3>
                    </div>
                    <div className='grid gap-4 md:grid-cols-2'>
                      <Field>
                        <FieldLabel htmlFor='live-tv-start-padding'>
                          随时开始
                        </FieldLabel>
                        <Input
                          id='live-tv-start-padding'
                          type='number'
                          min='0'
                          value={advancedDraft.startPaddingMinutes}
                          onChange={(event) =>
                            updateAdvanced(
                              'startPaddingMinutes',
                              event.target.value
                            )
                          }
                          className='border-border/60 bg-background text-foreground'
                        />
                      </Field>
                      <Field>
                        <FieldLabel htmlFor='live-tv-stop-padding'>
                          随时停止
                        </FieldLabel>
                        <Input
                          id='live-tv-stop-padding'
                          type='number'
                          min='0'
                          value={advancedDraft.stopPaddingMinutes}
                          onChange={(event) =>
                            updateAdvanced(
                              'stopPaddingMinutes',
                              event.target.value
                            )
                          }
                          className='border-border/60 bg-background text-foreground'
                        />
                      </Field>
                    </div>
                  </div>

                  <div className='rounded-[1.25rem] border border-border/60 bg-muted/20 p-4'>
                    <div className='mb-4 flex items-center gap-2'>
                      <SaveIcon className='size-4 text-muted-foreground' />
                      <h3 className='font-medium'>录制后期处理</h3>
                    </div>
                    <FieldGroup>
                      <Field>
                        <FieldLabel htmlFor='live-tv-post-processor-app'>
                          后期处理应用程序
                        </FieldLabel>
                        <Input
                          id='live-tv-post-processor-app'
                          value={advancedDraft.postProcessorApp}
                          onChange={(event) =>
                            updateAdvanced(
                              'postProcessorApp',
                              event.target.value
                            )
                          }
                          placeholder='/usr/local/bin/process-recording'
                          className='border-border/60 bg-background text-foreground'
                        />
                      </Field>
                      <Field>
                        <FieldLabel htmlFor='live-tv-post-processor-arguments'>
                          后期处理器命令行参数
                        </FieldLabel>
                        <Textarea
                          id='live-tv-post-processor-arguments'
                          value={advancedDraft.postProcessorArguments}
                          onChange={(event) =>
                            updateAdvanced(
                              'postProcessorArguments',
                              event.target.value
                            )
                          }
                          placeholder='--path {path} --channel {channelname} --number {channelnumber}'
                          className='min-h-28 border-border/60 bg-background font-mono text-sm text-foreground'
                        />
                      </Field>
                    </FieldGroup>
                  </div>
                </FieldGroup>
              </form>
            </div>
          </section>
        </TabsContent>
      </Tabs>

      {createPortal(
        <div className='fixed right-6 bottom-6 z-[100] flex justify-end'>
          {activeTab === 'advanced' ? (
            <Button type='submit' form='live-tv-advanced-form'>
              保存浏览器草稿
            </Button>
          ) : (
            <Button type='button' onClick={handleCreateSource}>
              <SaveIcon className='size-4' />
              添加直播源
            </Button>
          )}
        </div>,
        document.body
      )}

      <Dialog
        open={sourceDialogOpen}
        onOpenChange={(open) => {
          setSourceDialogOpen(open)
          if (!open) {
            setEditingSource(null)
            setSourceForm(defaultSourceForm)
          }
        }}
      >
        <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>
              {editingSource ? '编辑直播源' : '添加直播源'}
            </DialogTitle>
            <DialogDescription>
              配置 M3U 播放列表地址、请求头和频道导入规则。
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmitSource} className='space-y-4'>
            <FieldGroup>
              <div className='space-y-1'>
                <h3 className='text-sm font-medium'>M3U</h3>
                <p className='text-xs text-muted-foreground'>
                  当前支持远程 HTTP/HTTPS 播放列表，保存后可刷新抓取频道。
                </p>
              </div>
              <div className='grid gap-4 md:grid-cols-[minmax(0,1fr)_12rem]'>
                <Field>
                  <FieldLabel htmlFor='live-tv-source-name'>源名称</FieldLabel>
                  <Input
                    id='live-tv-source-name'
                    value={sourceForm.name}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        name: event.target.value,
                      }))
                    }
                    placeholder='例如：央视频道合集'
                  />
                </Field>
                <Field>
                  <FieldLabel>格式提示</FieldLabel>
                  <Select
                    value={sourceForm.formatHint}
                    onValueChange={(value: 'auto' | 'm3u' | 'txt') =>
                      setSourceForm((current) => ({
                        ...current,
                        formatHint: value,
                      }))
                    }
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value='auto'>自动识别</SelectItem>
                      <SelectItem value='m3u'>M3U</SelectItem>
                      <SelectItem value='txt'>TXT</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>
              </div>

              <Field>
                <FieldLabel htmlFor='live-tv-source-url'>文件或网址</FieldLabel>
                <div className='flex gap-2'>
                  <Input
                    id='live-tv-source-url'
                    value={sourceForm.url}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        url: event.target.value,
                      }))
                    }
                    placeholder='https://example.com/playlist.m3u'
                    className='font-mono text-sm'
                    required
                  />
                  <Button
                    type='button'
                    variant='outline'
                    size='icon'
                    title='选择路径'
                    aria-label='选择路径'
                    disabled
                  >
                    <FolderSearchIcon className='size-4' />
                  </Button>
                </div>
                <FieldDescription>
                  本地文件选择入口已预留；当前后端只接收 HTTP/HTTPS
                  播放列表地址。
                </FieldDescription>
              </Field>

              <div className='grid gap-4 md:grid-cols-2'>
                <Field>
                  <FieldLabel htmlFor='live-tv-source-user-agent'>
                    用户代理 HTTP 标头
                  </FieldLabel>
                  <Input
                    id='live-tv-source-user-agent'
                    value={sourceForm.userAgent}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        userAgent: event.target.value,
                      }))
                    }
                    placeholder='Mozilla/5.0'
                  />
                  <FieldDescription>
                    必要时提供自定义 User-Agent。
                  </FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor='live-tv-source-referrer'>
                    引用者 HTTP 标头
                  </FieldLabel>
                  <Input
                    id='live-tv-source-referrer'
                    value={sourceForm.referrer}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        referrer: event.target.value,
                      }))
                    }
                    placeholder='https://example.com'
                  />
                  <FieldDescription>
                    必要时提供自定义 Referer。
                  </FieldDescription>
                </Field>
              </div>

              <div className='grid gap-4 md:grid-cols-[12rem_minmax(0,1fr)]'>
                <Field>
                  <FieldLabel htmlFor='live-tv-source-tuner-count'>
                    并发流限制
                  </FieldLabel>
                  <Input
                    id='live-tv-source-tuner-count'
                    type='number'
                    min={0}
                    step={1}
                    inputMode='decimal'
                    value={sourceForm.tunerCount}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        tunerCount: event.target.value,
                      }))
                    }
                    required
                  />
                  <FieldDescription>输入 0 表示无限制。</FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor='live-tv-source-import-groups'>
                    仅导入包含这些组的频道
                  </FieldLabel>
                  <Input
                    id='live-tv-source-import-groups'
                    value={sourceForm.importGroups}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        importGroups: event.target.value,
                      }))
                    }
                    placeholder='央视;卫视;体育'
                  />
                  <FieldDescription>
                    可选。使用 ; 分隔多个组，刷新时只导入这些分组。
                  </FieldDescription>
                </Field>
              </div>

              <div className='grid gap-4 md:grid-cols-2'>
                <ToggleField
                  title='如果可用，直接从 M3U 导入节目指南'
                  description='当播放列表提供 url-tvg 或 x-tvg-url 时保留该偏好。'
                  checked={sourceForm.importGuideData}
                  onCheckedChange={(checked) =>
                    setSourceForm((current) => ({
                      ...current,
                      importGuideData: checked,
                    }))
                  }
                />
                <ToggleField
                  title='允许使用频道编号映射指南数据'
                  description='名称匹配失败时可作为后备，频道编号不准时可能误匹配。'
                  checked={sourceForm.allowGuideMappingByNumber}
                  onCheckedChange={(checked) =>
                    setSourceForm((current) => ({
                      ...current,
                      allowGuideMappingByNumber: checked,
                    }))
                  }
                />
              </div>

              <div className='grid gap-4 md:grid-cols-2'>
                <Field>
                  <FieldLabel>首选频道图像来源</FieldLabel>
                  <Select
                    value={sourceForm.channelImageSource}
                    onValueChange={(value: 'm3u' | 'guide') =>
                      setSourceForm((current) => ({
                        ...current,
                        channelImageSource: value,
                      }))
                    }
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value='m3u'>M3U</SelectItem>
                      <SelectItem value='guide'>指南数据源</SelectItem>
                    </SelectContent>
                  </Select>
                  <FieldDescription>
                    当调谐器和指南数据都有图像时使用。
                  </FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor='live-tv-source-channel-tags'>
                    为频道添加标签
                  </FieldLabel>
                  <Input
                    id='live-tv-source-channel-tags'
                    value={sourceForm.channelTags}
                    onChange={(event) =>
                      setSourceForm((current) => ({
                        ...current,
                        channelTags: event.target.value,
                      }))
                    }
                    placeholder='新闻;体育'
                  />
                  <FieldDescription>
                    可选。多个标签之间用 ; 分隔。
                  </FieldDescription>
                </Field>
              </div>
            </FieldGroup>

            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={handleCancelEdit}
              >
                <XIcon className='size-4' />
                取消
              </Button>
              <Button
                type='submit'
                disabled={createMutation.isPending || updateMutation.isPending}
              >
                {createMutation.isPending || updateMutation.isPending ? (
                  <LoaderCircleIcon className='size-4 animate-spin' />
                ) : (
                  <SaveIcon className='size-4' />
                )}
                {editingSource ? '保存修改' : '添加直播源'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog
        open={!!channelDialogSource}
        onOpenChange={(open) => {
          if (!open) {
            setChannelDialogSource(null)
          }
        }}
      >
        <DialogContent className='sm:max-w-3xl'>
          <DialogHeader>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
              <div className='space-y-1'>
                <DialogTitle>
                  {channelDialogSource
                    ? `${channelDialogSource.name} 的频道`
                    : '直播源频道'}
                </DialogTitle>
                <DialogDescription>
                  浏览当前直播源已导入的频道，并可直接发起播放。
                </DialogDescription>
              </div>
              {channelDialogSource ? (
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => refreshMutation.mutate(channelDialogSource.id)}
                  disabled={channelDialogRefreshPending}
                >
                  <RefreshCwIcon
                    className={`size-4 ${channelDialogRefreshPending ? 'animate-spin' : ''}`}
                  />
                  刷新频道
                </Button>
              ) : null}
            </div>
          </DialogHeader>
          {sourceChannelsQuery.isLoading ? (
            <div className='flex min-h-48 items-center justify-center gap-3 text-sm text-muted-foreground'>
              <LoaderCircleIcon className='size-4 animate-spin' />
              正在加载频道
            </div>
          ) : sourceChannelsQuery.error ? (
            <Alert variant='destructive'>
              <AlertTitle>加载频道失败</AlertTitle>
              <AlertDescription>
                {sourceChannelsQuery.error.message}
              </AlertDescription>
            </Alert>
          ) : (sourceChannelsQuery.data?.length ?? 0) === 0 ? (
            <Empty className='min-h-64 border border-dashed border-border/70 bg-muted/20'>
              <EmptyHeader>
                <EmptyMedia variant='icon'>
                  <TvIcon className='size-4' />
                </EmptyMedia>
                <EmptyTitle>这个直播源还没有频道。</EmptyTitle>
                <EmptyDescription>
                  先刷新直播源，再回来查看这里的频道列表。
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => {
                    if (!channelDialogSource) {
                      return
                    }
                    refreshMutation.mutate(channelDialogSource.id)
                  }}
                  disabled={refreshMutation.isPending || !channelDialogSource}
                >
                  <RefreshCwIcon
                    className={`size-4 ${refreshMutation.isPending ? 'animate-spin' : ''}`}
                  />
                  刷新直播源
                </Button>
              </EmptyContent>
            </Empty>
          ) : (
            <div className='max-h-[65vh] space-y-3 overflow-y-auto pr-1'>
              <div className='flex items-center justify-between rounded-[1rem] border border-border/60 bg-muted/20 px-4 py-3 text-sm text-muted-foreground'>
                <span>
                  当前展示来自该直播源的频道列表，不会混入其他源的数据。
                </span>
                <span>共 {sourceChannels.length} 个频道</span>
              </div>
              {sourceChannels.map((channel) => (
                <div
                  key={channel.id}
                  className='flex flex-col gap-4 rounded-[1.25rem] border border-border/60 bg-background/70 p-4 sm:flex-row sm:items-center sm:justify-between'
                >
                  <div className='min-w-0 space-y-1'>
                    <div className='flex flex-wrap items-center gap-2'>
                      <h3 className='font-medium'>{channel.name}</h3>
                      {channel.group_name ? (
                        <span className='rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground'>
                          {channel.group_name}
                        </span>
                      ) : null}
                    </div>
                    <p className='text-sm text-muted-foreground'>
                      {channel.current_program
                        ? `正在播出：${channel.current_program.title}`
                        : channel.tvg_name ||
                          channel.tvg_id ||
                          '未提供 TVG 元数据'}
                    </p>
                    {channel.tags?.length ? (
                      <div className='flex flex-wrap gap-1.5'>
                        {channel.tags.map((tag) => (
                          <span
                            key={tag}
                            className='rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-700'
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    ) : null}
                  </div>
                  <Button
                    type='button'
                    className='bg-emerald-600 text-white hover:bg-emerald-700'
                    onClick={() => {
                      void handlePlaybackChannelClick(channel)
                    }}
                  >
                    <PlayIcon className='size-4' />
                    播放
                  </Button>
                </div>
              ))}
            </div>
          )}
        </DialogContent>
      </Dialog>

      <Dialog
        open={!!playbackChannel}
        onOpenChange={(open) => {
          if (!open) {
            setPlaybackChannel(null)
          }
        }}
      >
        <DialogContent className='sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>{playbackChannel?.name || '播放直播频道'}</DialogTitle>
            <DialogDescription>
              {playbackChannel?.group_name || '通过后端代理打开直播流'}
            </DialogDescription>
          </DialogHeader>
          {playbackQuery.isLoading ? (
            <div className='flex min-h-48 items-center justify-center gap-3 text-sm text-muted-foreground'>
              <LoaderCircleIcon className='size-4 animate-spin' />
              正在准备播放流
            </div>
          ) : playbackQuery.error ? (
            <Alert variant='destructive'>
              <AlertTitle>无法播放该频道</AlertTitle>
              <AlertDescription>{playbackQuery.error.message}</AlertDescription>
            </Alert>
          ) : playbackQuery.data ? (
            <div className='space-y-3'>
              <LiveTVVideo
                key={playbackQuery.data.url}
                src={playbackQuery.data.url}
                controls
                autoPlay
                playsInline
                className='aspect-video w-full rounded-xl bg-black'
              />
              <p className='text-sm text-muted-foreground'>
                当前通过后端代理播放直播流。如果频道无法播放，通常是上游源失效、格式不兼容或浏览器不支持该流媒体类型。
              </p>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  )
}

function ToggleField({
  title,
  description,
  checked,
  onCheckedChange,
}: {
  title: string
  description: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <div className='flex items-start justify-between gap-4 rounded-md border bg-muted/20 p-3'>
      <div className='space-y-1'>
        <div className='text-sm leading-none font-medium'>{title}</div>
        <p className='text-xs leading-relaxed text-muted-foreground'>
          {description}
        </p>
      </div>
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        aria-label={title}
        className='mt-0.5 shrink-0'
      />
    </div>
  )
}

function SourceStatusChip({ source }: { source: LiveTVSource }) {
  if (source.refresh.status === 'success') {
    return (
      <span className='rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-700'>
        已刷新
      </span>
    )
  }
  if (source.refresh.status === 'failed') {
    return (
      <span className='rounded-full bg-destructive/10 px-2 py-0.5 text-xs text-destructive'>
        刷新失败
      </span>
    )
  }
  return (
    <span className='rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground'>
      未刷新
    </span>
  )
}

function RecordingFolderField({
  id,
  label,
  description,
  value,
  onChange,
}: {
  id: string
  label: string
  description: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder='选择或输入录制目录'
        className='border-border/60 bg-background text-foreground'
      />
      <FieldDescription>{description}</FieldDescription>
    </Field>
  )
}
