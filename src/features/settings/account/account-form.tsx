import { useEffect } from 'react'
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
import {
  fromSelectValue,
  languageOptions,
  localeOptions,
  mergeUserSettings,
  selectValue,
} from '../user-settings'

const accountFormSchema = z.object({
  locale: z.string().max(32),
  preferredAudioLanguage: z.string().max(32),
  preferredSubtitleLanguage: z.string().max(32),
})

type AccountFormValues = z.infer<typeof accountFormSchema>

const defaultValues: AccountFormValues = {
  locale: '',
  preferredAudioLanguage: '',
  preferredSubtitleLanguage: '',
}

export function AccountForm() {
  const queryClient = useQueryClient()
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryToken = accessToken || 'guest'

  const settingsQuery = useQuery({
    ...userSettingsQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
  })

  const form = useForm<AccountFormValues>({
    resolver: zodResolver(accountFormSchema),
    defaultValues,
  })

  useEffect(() => {
    if (!settingsQuery.data || form.formState.isDirty) {
      return
    }

    form.reset({
      locale: settingsQuery.data.appearance.locale,
      preferredAudioLanguage:
        settingsQuery.data.playback.preferred_audio_language,
      preferredSubtitleLanguage:
        settingsQuery.data.playback.preferred_subtitle_language,
    })
  }, [form, form.formState.isDirty, settingsQuery.data])

  const saveMutation = useMutation({
    mutationFn: async (values: AccountFormValues) => {
      if (!accessToken) {
        throw new Error('缺少登录会话')
      }

      return createAuthedMiboApi(accessToken).updateUserSettings(
        mergeUserSettings(settingsQuery.data, {
          appearance: {
            locale: values.locale,
          },
          playback: {
            preferred_audio_language: values.preferredAudioLanguage,
            preferred_subtitle_language: values.preferredSubtitleLanguage,
          },
        })
      )
    },
    onSuccess: (settings) => {
      queryClient.setQueryData(miboQueryKeys.userSettings(queryToken), settings)
      form.reset({
        locale: settings.appearance.locale,
        preferredAudioLanguage: settings.playback.preferred_audio_language,
        preferredSubtitleLanguage:
          settings.playback.preferred_subtitle_language,
      })
      toast.success('账户偏好已更新')
    },
  })

  function onSubmit(values: AccountFormValues) {
    saveMutation.mutate(values)
  }

  if (!hasHydrated || (accessToken && settingsQuery.isLoading)) {
    return (
      <div className='text-sm text-muted-foreground'>正在加载账户偏好...</div>
    )
  }

  if (!accessToken) {
    return (
      <div className='text-sm text-muted-foreground'>
        登录后即可管理账户偏好。
      </div>
    )
  }

  if (settingsQuery.error) {
    return (
      <Alert variant='destructive'>
        <AlertTitle>无法加载账户偏好</AlertTitle>
        <AlertDescription>{settingsQuery.error.message}</AlertDescription>
      </Alert>
    )
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-8'>
        <FormField
          control={form.control}
          name='locale'
          render={({ field }) => (
            <FormItem>
              <FormLabel>界面语言区域</FormLabel>
              <Select
                onValueChange={(value) =>
                  field.onChange(fromSelectValue(value))
                }
                value={selectValue(field.value)}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder='选择语言区域' />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {localeOptions.map((locale) => (
                    <SelectItem
                      key={locale.label}
                      value={selectValue(locale.value)}
                    >
                      {locale.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormDescription>
                设置界面默认语言以及日期、时间等区域格式。
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='preferredAudioLanguage'
          render={({ field }) => (
            <FormItem>
              <FormLabel>首选音频语言</FormLabel>
              <Select
                onValueChange={(value) =>
                  field.onChange(fromSelectValue(value))
                }
                value={selectValue(field.value)}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder='选择音频语言' />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {languageOptions.map((language) => (
                    <SelectItem
                      key={language.label}
                      value={selectValue(language.value)}
                    >
                      {language.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormDescription>
                当存在多条音轨时，Mibo 会优先选择该语言。
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='preferredSubtitleLanguage'
          render={({ field }) => (
            <FormItem>
              <FormLabel>首选字幕语言</FormLabel>
              <Select
                onValueChange={(value) =>
                  field.onChange(fromSelectValue(value))
                }
                value={selectValue(field.value)}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder='选择字幕语言' />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {languageOptions.map((language) => (
                    <SelectItem
                      key={language.label}
                      value={selectValue(language.value)}
                    >
                      {language.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormDescription>
                当系统自动选择字幕时，Mibo 会优先使用该语言。
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type='submit' disabled={saveMutation.isPending}>
          {saveMutation.isPending ? '保存中...' : '保存账户设置'}
        </Button>
      </form>
    </Form>
  )
}
