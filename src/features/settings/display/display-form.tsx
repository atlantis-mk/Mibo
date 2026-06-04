import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import {
  createAuthedMiboApi,
  miboQueryKeys,
  userSettingsQueryOptions,
} from '@/lib/mibo-query'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  EXTERNAL_PLAYER_OPTIONS,
  getCurrentPlatform,
  getPlaybackLaunchPreferences,
  getRecommendedExternalPlayer,
  getSupportedExternalPlayerOptions,
  setPlaybackLaunchPreferences,
  type ExternalPlayerId,
  type PlaybackLaunchMode,
} from '@/features/play/external-player'
import { mergeUserSettings, subtitleModeOptions } from '../user-settings'

const displayFormSchema = z.object({
  autoplayNextEpisode: z.boolean(),
  preferDirectPlay: z.boolean(),
  defaultSubtitleMode: z.enum(['auto', 'always', 'never']),
})

type DisplayFormValues = z.infer<typeof displayFormSchema>

const defaultValues: DisplayFormValues = {
  autoplayNextEpisode: true,
  preferDirectPlay: true,
  defaultSubtitleMode: 'auto',
}

export function DisplayForm() {
  const queryClient = useQueryClient()
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryToken = accessToken || 'guest'

  const settingsQuery = useQuery({
    ...userSettingsQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
  })

  const form = useForm<DisplayFormValues>({
    resolver: zodResolver(displayFormSchema),
    defaultValues,
  })

  const [launchMode, setLaunchMode] = useState<PlaybackLaunchMode>('internal')
  const [externalPlayerId, setExternalPlayerId] = useState<ExternalPlayerId>(
    getRecommendedExternalPlayer
  )

  const supportedExternalPlayers = getSupportedExternalPlayerOptions()
  const currentPlatform = getCurrentPlatform()
  const externalPlayerOptions =
    supportedExternalPlayers.length > 0
      ? supportedExternalPlayers
      : EXTERNAL_PLAYER_OPTIONS

  useEffect(() => {
    if (!settingsQuery.data || form.formState.isDirty) {
      return
    }

    form.reset({
      autoplayNextEpisode: settingsQuery.data.playback.autoplay_next_episode,
      preferDirectPlay: settingsQuery.data.playback.prefer_direct_play,
      defaultSubtitleMode: settingsQuery.data.playback.default_subtitle_mode,
    })
  }, [form, form.formState.isDirty, settingsQuery.data])

  useEffect(() => {
    const preferences = getPlaybackLaunchPreferences()
    setLaunchMode(preferences.mode)
    setExternalPlayerId(preferences.externalPlayerId)
  }, [])

  const saveMutation = useMutation({
    mutationFn: async (values: DisplayFormValues) => {
      if (!accessToken) {
        throw new Error('缺少登录会话')
      }

      return createAuthedMiboApi(accessToken).updateUserSettings(
        mergeUserSettings(settingsQuery.data, {
          playback: {
            autoplay_next_episode: values.autoplayNextEpisode,
            prefer_direct_play: values.preferDirectPlay,
            default_subtitle_mode: values.defaultSubtitleMode,
          },
        })
      )
    },
    onSuccess: (settings) => {
      queryClient.setQueryData(miboQueryKeys.userSettings(queryToken), settings)
      form.reset({
        autoplayNextEpisode: settings.playback.autoplay_next_episode,
        preferDirectPlay: settings.playback.prefer_direct_play,
        defaultSubtitleMode: settings.playback.default_subtitle_mode,
      })
      setPlaybackLaunchPreferences({
        mode: launchMode,
        externalPlayerId,
      })
      toast.success('播放默认设置已更新')
    },
  })

  function onSubmit(values: DisplayFormValues) {
    saveMutation.mutate(values)
  }

  if (!hasHydrated || (accessToken && settingsQuery.isLoading)) {
    return (
      <div className='text-sm text-muted-foreground'>
        正在加载播放默认设置...
      </div>
    )
  }

  if (!accessToken) {
    return (
      <div className='text-sm text-muted-foreground'>
        登录后即可管理播放默认设置。
      </div>
    )
  }

  if (settingsQuery.error) {
    return (
      <Alert variant='destructive'>
        <AlertTitle>无法加载播放默认设置</AlertTitle>
        <AlertDescription>{settingsQuery.error.message}</AlertDescription>
      </Alert>
    )
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-8'>
        <FormField
          control={form.control}
          name='autoplayNextEpisode'
          render={({ field }) => (
            <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
              <div className='space-y-0.5'>
                <FormLabel className='text-base'>自动播放下一集</FormLabel>
                <FormDescription>
                  当前一集播放结束后，自动继续播放下一集。
                </FormDescription>
              </div>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='preferDirectPlay'
          render={({ field }) => (
            <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
              <div className='space-y-0.5'>
                <FormLabel className='text-base'>优先直放</FormLabel>
                <FormDescription>
                  当设备支持时，优先让播放器直接使用原始媒体文件。
                </FormDescription>
              </div>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='defaultSubtitleMode'
          render={({ field }) => (
            <FormItem>
              <FormLabel>默认字幕模式</FormLabel>
              <Select onValueChange={field.onChange} value={field.value}>
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder='选择字幕模式' />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {subtitleModeOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormDescription>
                控制在你手动选择之前，Mibo 如何默认选择字幕。
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className='space-y-4 rounded-lg border p-4'>
          <div className='space-y-1'>
            <div className='text-base font-medium'>播放打开方式</div>
            <p className='text-sm text-muted-foreground'>
              决定点击“播放”后，是进入 Mibo 内置播放器，还是直接交给外部播放器。
            </p>
          </div>
          <div className='grid gap-4 md:grid-cols-2'>
            <button
              type='button'
              onClick={() => setLaunchMode('internal')}
              className={`rounded-lg border p-4 text-left transition-colors ${launchMode === 'internal' ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/40'}`}
            >
              <div className='font-medium'>Mibo 内置播放器</div>
              <div className='mt-1 text-sm text-muted-foreground'>
                支持站内进度恢复、字幕偏好和完整播放控制。
              </div>
            </button>
            <button
              type='button'
              onClick={() => setLaunchMode('external')}
              className={`rounded-lg border p-4 text-left transition-colors ${launchMode === 'external' ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/40'}`}
            >
              <div className='font-medium'>外部播放器</div>
              <div className='mt-1 text-sm text-muted-foreground'>
                直接把播放链接交给 VLC、IINA、mpv 等外部播放器。
              </div>
            </button>
          </div>
          <div className='space-y-2'>
            <FormLabel>外部播放器</FormLabel>
            <Select
              value={externalPlayerId}
              onValueChange={(value) =>
                setExternalPlayerId(value as ExternalPlayerId)
              }
              disabled={launchMode !== 'external'}
            >
              <SelectTrigger>
                <SelectValue placeholder='选择外部播放器' />
              </SelectTrigger>
              <SelectContent>
                {externalPlayerOptions.map((option) => (
                  <SelectItem key={option.id} value={option.id}>
                    {option.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <FormDescription>
              当前平台识别为 {currentPlatform}。这项偏好只保存在当前浏览器。
            </FormDescription>
            {launchMode === 'external' ? (
              <FormDescription>
                外部播放器当前会直接打开播放链接，不会自动同步站内的恢复进度和字幕控制。
              </FormDescription>
            ) : null}
          </div>
        </div>
        <Button type='submit' disabled={saveMutation.isPending}>
          {saveMutation.isPending ? '保存中...' : '保存播放设置'}
        </Button>
      </form>
    </Form>
  )
}
