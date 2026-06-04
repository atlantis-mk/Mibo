import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { ChevronDownIcon } from '@radix-ui/react-icons'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fonts } from '@/config/fonts'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import {
  createAuthedMiboApi,
  miboQueryKeys,
  userSettingsQueryOptions,
} from '@/lib/mibo-query'
import { cn } from '@/lib/utils'
import { useFont } from '@/context/font-provider'
import { useTheme } from '@/context/theme-provider'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { mergeUserSettings, themeOptions } from '../user-settings'

const appearanceFormSchema = z.object({
  theme: z.enum(['light', 'dark', 'system']),
  font: z.enum(fonts),
})

type AppearanceFormValues = z.infer<typeof appearanceFormSchema>

export function AppearanceForm() {
  const queryClient = useQueryClient()
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryToken = accessToken || 'guest'
  const { font, setFont } = useFont()
  const { theme, setTheme } = useTheme()

  const settingsQuery = useQuery({
    ...userSettingsQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
  })

  const form = useForm<AppearanceFormValues>({
    resolver: zodResolver(appearanceFormSchema),
    defaultValues: {
      theme,
      font,
    },
  })

  useEffect(() => {
    if (!settingsQuery.data || form.formState.isDirty) {
      return
    }

    const nextTheme = settingsQuery.data.appearance.theme
    if (nextTheme !== theme) {
      setTheme(nextTheme)
    }

    form.reset({
      theme: nextTheme,
      font,
    })
  }, [font, form, form.formState.isDirty, setTheme, settingsQuery.data, theme])

  const saveMutation = useMutation({
    mutationFn: async (values: AppearanceFormValues) => {
      if (!accessToken) {
        throw new Error('缺少登录会话')
      }

      return createAuthedMiboApi(accessToken).updateUserSettings(
        mergeUserSettings(settingsQuery.data, {
          appearance: {
            theme: values.theme,
          },
        })
      )
    },
    onSuccess: (settings, values) => {
      queryClient.setQueryData(miboQueryKeys.userSettings(queryToken), settings)
      if (values.font !== font) {
        setFont(values.font)
      }
      setTheme(settings.appearance.theme)
      form.reset({
        theme: settings.appearance.theme,
        font: values.font,
      })
      toast.success('外观设置已更新')
    },
  })

  function onSubmit(values: AppearanceFormValues) {
    saveMutation.mutate(values)
  }

  if (!hasHydrated || (accessToken && settingsQuery.isLoading)) {
    return (
      <div className='text-sm text-muted-foreground'>正在加载外观设置...</div>
    )
  }

  if (!accessToken) {
    return (
      <div className='text-sm text-muted-foreground'>
        登录后即可管理外观设置。
      </div>
    )
  }

  if (settingsQuery.error) {
    return (
      <Alert variant='destructive'>
        <AlertTitle>无法加载外观设置</AlertTitle>
        <AlertDescription>{settingsQuery.error.message}</AlertDescription>
      </Alert>
    )
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-8'>
        <FormField
          control={form.control}
          name='font'
          render={({ field }) => (
            <FormItem>
              <FormLabel>字体</FormLabel>
              <div className='relative w-max'>
                <FormControl>
                  <select
                    className={cn(
                      buttonVariants({ variant: 'outline' }),
                      'w-50 appearance-none font-normal capitalize',
                      'dark:bg-background dark:hover:bg-background'
                    )}
                    {...field}
                  >
                    {fonts.map((option) => (
                      <option key={option} value={option}>
                        {option}
                      </option>
                    ))}
                  </select>
                </FormControl>
                <ChevronDownIcon className='absolute inset-e-3 top-2.5 h-4 w-4 opacity-50' />
              </div>
              <FormDescription>
                字体选择仅保存在当前浏览器和当前设备中。
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='theme'
          render={({ field }) => (
            <FormItem>
              <FormLabel>主题</FormLabel>
              <FormDescription>选择登录后默认恢复的界面主题。</FormDescription>
              <FormMessage />
              <RadioGroup
                onValueChange={field.onChange}
                value={field.value}
                className='grid max-w-xl grid-cols-1 gap-4 pt-2 sm:grid-cols-3'
              >
                {themeOptions.map((option) => (
                  <FormItem key={option.value}>
                    <FormLabel className='[&:has([data-state=checked])>div]:border-primary'>
                      <FormControl>
                        <RadioGroupItem
                          value={option.value}
                          className='sr-only'
                        />
                      </FormControl>
                      <div className='rounded-md border-2 border-muted p-4 hover:border-accent'>
                        <div className='space-y-2'>
                          <div className='text-sm font-medium'>
                            {option.label}
                          </div>
                          <p className='text-sm text-muted-foreground'>
                            {option.value === 'system'
                              ? '自动跟随当前设备的系统主题。'
                              : option.value === 'light'
                                ? '默认使用浅色主题。'
                                : '默认使用深色主题。'}
                          </p>
                        </div>
                      </div>
                    </FormLabel>
                  </FormItem>
                ))}
              </RadioGroup>
            </FormItem>
          )}
        />

        <Button type='submit' disabled={saveMutation.isPending}>
          {saveMutation.isPending ? '保存中...' : '保存外观设置'}
        </Button>
      </form>
    </Form>
  )
}
