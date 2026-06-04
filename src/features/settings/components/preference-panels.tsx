import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { InfoIcon, LoaderCircleIcon, ShieldCheckIcon } from 'lucide-react'
import { createPortal } from 'react-dom'
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
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
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
import { mergeUserSettings } from '../user-settings'
import { SettingSwitchField } from './setting-switch-field'

export function PlaybackSettingsPanel() {
  return (
    <SettingsPanelCard
      title='播放体验'
      description='集中调整默认画质、设备档案和自动续播策略。'
      note='这些偏好目前用于界面预设展示，后续会接入服务端持久化。'
    >
      <FieldGroup>
        <div className='grid gap-4 md:grid-cols-2'>
          <Field>
            <FieldLabel>默认画质</FieldLabel>
            <Select defaultValue='original'>
              <SelectTrigger className='w-full border-border/60 bg-background text-foreground'>
                <SelectValue placeholder='选择画质' />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='original'>原始质量</SelectItem>
                <SelectItem value='1080p'>1080p</SelectItem>
                <SelectItem value='720p'>720p</SelectItem>
              </SelectContent>
            </Select>
            <FieldDescription>
              优先使用原始文件，无法直放时再降级。
            </FieldDescription>
          </Field>

          <Field>
            <FieldLabel>设备档案</FieldLabel>
            <Select defaultValue='web'>
              <SelectTrigger className='w-full border-border/60 bg-background text-foreground'>
                <SelectValue placeholder='选择设备档案' />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='web'>Web</SelectItem>
                <SelectItem value='tv'>TV</SelectItem>
                <SelectItem value='mobile'>Mobile</SelectItem>
              </SelectContent>
            </Select>
            <FieldDescription>
              根据播放端选择更合适的兼容策略。
            </FieldDescription>
          </Field>
        </div>

        <SettingSwitchField
          title='自动续播'
          description='打开媒体详情时优先恢复到上次播放进度。'
          defaultChecked
        />
        <SettingSwitchField
          title='优先转码兼容格式'
          description='当原始文件不可直放时优先选择兼容性更高的输出格式。'
        />
      </FieldGroup>
    </SettingsPanelCard>
  )
}

const securityFormSchema = z.object({
  sessionTimeout: z.enum(['12h', '24h', '7d', '30d']),
  loginProtectionLevel: z.enum(['standard', 'strict', 'local_only']),
  autoClearInvalidToken: z.boolean(),
  requireDangerousActionConfirmation: z.boolean(),
})

type SecurityFormValues = z.infer<typeof securityFormSchema>

const securityDefaults: SecurityFormValues = {
  sessionTimeout: '24h',
  loginProtectionLevel: 'standard',
  autoClearInvalidToken: true,
  requireDangerousActionConfirmation: true,
}

export function SecuritySettingsPanel() {
  const queryClient = useQueryClient()
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryToken = accessToken || 'guest'

  const settingsQuery = useQuery({
    ...userSettingsQueryOptions(queryToken),
    enabled: hasHydrated && !!accessToken,
  })

  const form = useForm<SecurityFormValues>({
    resolver: zodResolver(securityFormSchema),
    defaultValues: securityDefaults,
  })

  useEffect(() => {
    if (!settingsQuery.data || form.formState.isDirty) {
      return
    }

    const security = settingsQuery.data.security
    form.reset({
      sessionTimeout: security.session_timeout,
      loginProtectionLevel: security.login_protection_level,
      autoClearInvalidToken: security.auto_clear_invalid_token,
      requireDangerousActionConfirmation:
        security.require_dangerous_action_confirmation,
    })
    syncSecurityLocalPreferences(security.auto_clear_invalid_token)
  }, [form, form.formState.isDirty, settingsQuery.data])

  const saveMutation = useMutation({
    mutationFn: async (values: SecurityFormValues) => {
      if (!accessToken) {
        throw new Error('缺少登录会话')
      }

      return createAuthedMiboApi(accessToken).updateUserSettings(
        mergeUserSettings(settingsQuery.data, {
          security: {
            session_timeout: values.sessionTimeout,
            login_protection_level: values.loginProtectionLevel,
            auto_clear_invalid_token: values.autoClearInvalidToken,
            require_dangerous_action_confirmation:
              values.requireDangerousActionConfirmation,
          },
        })
      )
    },
    onSuccess: (settings) => {
      queryClient.setQueryData(miboQueryKeys.userSettings(queryToken), settings)
      const security = settings.security
      form.reset({
        sessionTimeout: security.session_timeout,
        loginProtectionLevel: security.login_protection_level,
        autoClearInvalidToken: security.auto_clear_invalid_token,
        requireDangerousActionConfirmation:
          security.require_dangerous_action_confirmation,
      })
      syncSecurityLocalPreferences(security.auto_clear_invalid_token)
      toast.success('账号安全设置已更新')
    },
  })

  function onSubmit(values: SecurityFormValues) {
    saveMutation.mutate(values)
  }

  if (!hasHydrated || (accessToken && settingsQuery.isLoading)) {
    return (
      <div className='text-sm text-muted-foreground'>
        正在加载账号安全设置...
      </div>
    )
  }

  if (!accessToken) {
    return (
      <div className='text-sm text-muted-foreground'>
        登录后即可管理账号安全设置。
      </div>
    )
  }

  if (settingsQuery.error) {
    return (
      <Alert variant='destructive'>
        <AlertTitle>无法加载账号安全设置</AlertTitle>
        <AlertDescription>{settingsQuery.error.message}</AlertDescription>
      </Alert>
    )
  }

  return (
    <Form {...form}>
      <form
        id='security-settings-form'
        onSubmit={form.handleSubmit(onSubmit)}
        className='space-y-6 pb-20'
      >
        <SettingsPanelCard
          title='会话策略'
          description='收敛登录时长和访问保护级别。'
          note='新登录会话会使用这里的时长；严格模式会限制为 12 小时并只保留最新会话，仅本地网络会拒绝公网来源登录。'
        >
          <FieldGroup>
            <div className='grid gap-4 md:grid-cols-2'>
              <FormField
                control={form.control}
                name='sessionTimeout'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>会话时长</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className='w-full border-border/60 bg-background text-foreground'>
                          <SelectValue placeholder='选择时长' />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value='12h'>12 小时</SelectItem>
                        <SelectItem value='24h'>24 小时</SelectItem>
                        <SelectItem value='7d'>7 天</SelectItem>
                        <SelectItem value='30d'>30 天</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      超过该时长后需要重新登录。
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='loginProtectionLevel'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>登录保护级别</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className='w-full border-border/60 bg-background text-foreground'>
                          <SelectValue placeholder='选择级别' />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value='standard'>标准</SelectItem>
                        <SelectItem value='strict'>严格</SelectItem>
                        <SelectItem value='local_only'>仅本地网络</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      控制新登录会话的保护强度和来源范围。
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </FieldGroup>
        </SettingsPanelCard>

        <SettingsPanelCard
          title='高危操作'
          description='管理 token 清理和危险动作确认。'
          note='失效 token 清理会立即影响当前浏览器；危险操作确认偏好会随账号保存，供删除、重扫等高风险流程使用。'
        >
          <FieldGroup>
            <FormField
              control={form.control}
              name='autoClearInvalidToken'
              render={({ field }) => (
                <FormItem className='flex items-start gap-3 rounded-[1.25rem] border border-border/60 bg-muted/30 p-3.5'>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                      className='mt-0.5'
                    />
                  </FormControl>
                  <div className='space-y-1'>
                    <FormLabel className='text-foreground'>
                      自动清理失效 token
                    </FormLabel>
                    <FormDescription>
                      当服务端判定 token 失效时立即移除本地会话。
                    </FormDescription>
                  </div>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='requireDangerousActionConfirmation'
              render={({ field }) => (
                <FormItem className='flex items-start gap-3 rounded-[1.25rem] border border-border/60 bg-muted/30 p-3.5'>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                      className='mt-0.5'
                    />
                  </FormControl>
                  <div className='space-y-1'>
                    <FormLabel className='text-foreground'>
                      限制高危操作二次确认
                    </FormLabel>
                    <FormDescription>
                      对删除库、重扫等高风险动作增加额外确认步骤。
                    </FormDescription>
                  </div>
                </FormItem>
              )}
            />
          </FieldGroup>
        </SettingsPanelCard>

        {saveMutation.error ? (
          <Alert variant='destructive'>
            <AlertTitle>保存失败</AlertTitle>
            <AlertDescription>{saveMutation.error.message}</AlertDescription>
          </Alert>
        ) : null}
      </form>

      {createPortal(
        <div className='fixed right-6 bottom-6 z-[100] flex justify-end'>
          <Button
            type='submit'
            form='security-settings-form'
            disabled={!form.formState.isDirty || saveMutation.isPending}
          >
            {saveMutation.isPending ? (
              <LoaderCircleIcon
                data-icon='inline-start'
                className='animate-spin'
              />
            ) : null}
            保存配置
          </Button>
        </div>,
        document.body
      )}
    </Form>
  )
}

function syncSecurityLocalPreferences(autoClearInvalidToken: boolean) {
  if (typeof window === 'undefined') {
    return
  }
  window.localStorage.setItem(
    'mibo-auto-clear-invalid-token',
    autoClearInvalidToken ? 'true' : 'false'
  )
}

function SettingsPanelCard({
  title,
  description,
  note,
  children,
}: {
  title: string
  description: string
  note: string
  children: React.ReactNode
}) {
  return (
    <section className='space-y-5'>
      <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
        <div className='flex items-start gap-3'>
          <div className='flex size-10 shrink-0 items-center justify-center rounded-xl border border-border/60 bg-background/70'>
            <ShieldCheckIcon className='size-4 text-muted-foreground' />
          </div>
          <div className='min-w-0'>
            <h2 className='text-xl font-semibold tracking-tight'>{title}</h2>
            <p className='mt-1 text-sm text-muted-foreground'>{description}</p>
          </div>
        </div>
      </div>
      <div className='flex items-start gap-3 rounded-[1.15rem] border border-border/60 bg-muted/30 px-4 py-3 text-sm leading-6 text-muted-foreground'>
        <InfoIcon className='mt-0.5 size-4 shrink-0' />
        <span>{note}</span>
      </div>
      {children}
    </section>
  )
}
