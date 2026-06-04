import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  CaptionsIcon,
  CloudIcon,
  InfoIcon,
  LoaderCircleIcon,
  PencilIcon,
  PlusIcon,
  PowerIcon,
  SaveIcon,
  SlidersHorizontalIcon,
} from 'lucide-react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import type {
  InternalPlugin,
  OpenSubtitlesProviderSettingsInput,
  SubtitleProviderInstance,
  SubtitleProviderInstanceInput,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  internalPluginsQueryOptions,
  miboQueryKeys,
  subtitleProviderInstancesQueryOptions,
} from '@/lib/mibo-query'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { EmptyManagementCard } from './metadata-provider-settings-panel'

type SubtitleProviderDraft = {
  id: number | null
  name: string
  providerType: 'opensubtitles'
  enabled: boolean
  availabilityStatus: string
  failureReason: string
  cooldownUntil: string
  opensubtitles: {
    apiKey: string
    baseURL: string
    languages: string
    timeout: string
    clearAPIKey: boolean
  }
}

const EMPTY_SUBTITLE_PROVIDER_DRAFT: SubtitleProviderDraft = {
  id: null,
  name: 'OpenSubtitles',
  providerType: 'opensubtitles',
  enabled: true,
  availabilityStatus: 'available',
  failureReason: '',
  cooldownUntil: '',
  opensubtitles: {
    apiKey: '',
    baseURL: 'https://api.opensubtitles.com/api/v1',
    languages: 'zh-cn,zh-tw,en',
    timeout: '10s',
    clearAPIKey: false,
  },
}

export function SubtitleSettingsPanel({ token }: { token: string | null }) {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [localDialogOpen, setLocalDialogOpen] = useState(false)
  const [draft, setDraft] = useState<SubtitleProviderDraft>(
    EMPTY_SUBTITLE_PROVIDER_DRAFT
  )
  const providersQuery = useQuery({
    ...subtitleProviderInstancesQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const internalPluginsQuery = useQuery({
    ...internalPluginsQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })

  const saveMutation = useMutation({
    mutationFn: async (input: SubtitleProviderDraft) => {
      if (!token) throw new Error('当前未登录，无法保存字幕提供方实例。')
      const api = createAuthedMiboApi(token)
      const payload = buildSubtitleProviderInput(input)
      if (input.id) {
        return api.updateSubtitleProviderInstance(input.id, payload)
      }
      return api.createSubtitleProviderInstance(payload)
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.subtitleProviderInstances(token),
      })
      setDialogOpen(false)
      setDraft(EMPTY_SUBTITLE_PROVIDER_DRAFT)
      toast.success('字幕提供方实例已保存')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const toggleMutation = useMutation({
    mutationFn: async (provider: SubtitleProviderInstance) => {
      if (!token) throw new Error('当前未登录，无法更新字幕提供方实例。')
      return createAuthedMiboApi(token).updateSubtitleProviderInstance(
        provider.id,
        { enabled: !provider.enabled }
      )
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.subtitleProviderInstances(token),
      })
      toast.success('字幕提供方状态已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const toggleLocalSubtitleMutation = useMutation({
    mutationFn: async (plugin: InternalPlugin) => {
      if (!token) throw new Error('当前未登录，无法更新本地字幕。')
      return createAuthedMiboApi(token).updateInternalPlugin(plugin.id, {
        enabled: !plugin.enabled,
      })
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.internalPlugins(token),
      })
      toast.success('本地字幕状态已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const toggleEmbeddedExtractionMutation = useMutation({
    mutationFn: async (plugin: InternalPlugin) => {
      if (!token) throw new Error('当前未登录，无法更新内嵌字幕抽取。')
      return createAuthedMiboApi(token).updateInternalPlugin(plugin.id, {
        local_subtitle: {
          embedded_extraction_enabled:
            !plugin.local_subtitle?.embedded_extraction_enabled,
        },
      })
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.internalPlugins(token),
      })
      toast.success('内嵌字幕抽取状态已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const toggleExternalFileMutation = useMutation({
    mutationFn: async (plugin: InternalPlugin) => {
      if (!token) throw new Error('当前未登录，无法更新外挂字幕文件。')
      return createAuthedMiboApi(token).updateInternalPlugin(plugin.id, {
        local_subtitle: {
          external_file_enabled: !plugin.local_subtitle?.external_file_enabled,
        },
      })
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.internalPlugins(token),
      })
      toast.success('外挂字幕文件状态已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  if (!token) {
    return (
      <Alert>
        <InfoIcon className='size-4' />
        <AlertTitle>登录后可管理字幕配置</AlertTitle>
        <AlertDescription className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
          <span>当前页面需要管理员会话来读取和更新字幕提供方实例。</span>
          <Button asChild variant='outline'>
            <Link to='/sign-in' search={{ redirect: '/settings/subtitles' }}>
              前往登录
            </Link>
          </Button>
        </AlertDescription>
      </Alert>
    )
  }

  if (providersQuery.isLoading || internalPluginsQuery.isLoading) {
    return (
      <div className='flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm'>
        <LoaderCircleIcon className='size-4 animate-spin' />
        正在加载字幕提供方
      </div>
    )
  }

  if (providersQuery.error || internalPluginsQuery.error) {
    return (
      <Alert variant='destructive'>
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>
          {providersQuery.error?.message || internalPluginsQuery.error?.message}
        </AlertDescription>
      </Alert>
    )
  }

  const providers = providersQuery.data ?? []
  const localSubtitlePlugin = (internalPluginsQuery.data ?? []).find(
    (plugin) => plugin.kind === 'subtitle' && plugin.provider_key === 'local'
  )

  return (
    <div className='space-y-4 pb-20'>
      <div className='grid gap-4 md:grid-cols-2 2xl:grid-cols-3'>
        <LocalSubtitleProviderCard
          plugin={localSubtitlePlugin}
          pending={toggleLocalSubtitleMutation.isPending}
          onOpenSettings={() => setLocalDialogOpen(true)}
          onToggle={() => {
            if (localSubtitlePlugin) {
              toggleLocalSubtitleMutation.mutate(localSubtitlePlugin)
            }
          }}
        />
        {providers.map((provider) => (
          <SubtitleProviderCard
            key={provider.id}
            provider={provider}
            pending={toggleMutation.isPending}
            onToggle={() => toggleMutation.mutate(provider)}
            onEdit={() => {
              setDraft(buildSubtitleProviderDraft(provider))
              setDialogOpen(true)
            }}
          />
        ))}
        {providers.length === 0 ? (
          <EmptyManagementCard text='还没有第三方字幕提供方实例。先新建一个 OpenSubtitles 实例。' />
        ) : null}
      </div>

      {createPortal(
        <div className='fixed right-6 bottom-6 z-[100] flex justify-end'>
          <Button
            type='button'
            onClick={() => {
              setDraft(EMPTY_SUBTITLE_PROVIDER_DRAFT)
              setDialogOpen(true)
            }}
          >
            <PlusIcon className='size-4' />
            新建字幕提供方
          </Button>
        </div>,
        document.body
      )}

      <Dialog open={localDialogOpen} onOpenChange={setLocalDialogOpen}>
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>本地字幕设置</DialogTitle>
            <DialogDescription>
              配置本地字幕在播放时可使用的附加能力。
            </DialogDescription>
          </DialogHeader>
          <LocalSubtitleSettingsForm
            plugin={localSubtitlePlugin}
            pending={
              toggleEmbeddedExtractionMutation.isPending ||
              toggleExternalFileMutation.isPending
            }
            onToggleExternalFiles={() => {
              if (localSubtitlePlugin) {
                toggleExternalFileMutation.mutate(localSubtitlePlugin)
              }
            }}
            onToggleEmbeddedExtraction={() => {
              if (localSubtitlePlugin) {
                toggleEmbeddedExtractionMutation.mutate(localSubtitlePlugin)
              }
            }}
          />
        </DialogContent>
      </Dialog>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className='flex max-h-[90vh] flex-col overflow-hidden sm:max-w-2xl'>
          <DialogHeader>
            <DialogTitle>
              {draft.id ? '编辑' : '新建'}字幕提供方实例
            </DialogTitle>
            <DialogDescription>
              当前字幕提供方只支持 OpenSubtitles 格式；每个实例可以保存独立的
              API Key 和请求参数。
            </DialogDescription>
          </DialogHeader>
          <div className='-mx-4 min-h-0 flex-1 overflow-y-auto px-4'>
            <SubtitleProviderForm draft={draft} onChange={setDraft} />
          </div>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => setDialogOpen(false)}
              disabled={saveMutation.isPending}
            >
              取消
            </Button>
            <Button
              type='button'
              disabled={saveMutation.isPending}
              onClick={() => saveMutation.mutate(draft)}
            >
              {saveMutation.isPending ? (
                <LoaderCircleIcon className='size-4 animate-spin' />
              ) : (
                <SaveIcon className='size-4' />
              )}
              {saveMutation.isPending ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function LocalSubtitleProviderCard({
  plugin,
  pending,
  onOpenSettings,
  onToggle,
}: {
  plugin?: InternalPlugin
  pending: boolean
  onOpenSettings: () => void
  onToggle: () => void
}) {
  const enabled = plugin?.enabled ?? true
  const externalFileEnabled =
    plugin?.local_subtitle?.external_file_enabled ?? true
  const extractionEnabled =
    plugin?.local_subtitle?.embedded_extraction_enabled ?? false
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='gap-3 px-4 py-3'>
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <CardTitle className='flex items-center gap-2 truncate text-base'>
              <CaptionsIcon className='size-5' />
              本地字幕
            </CardTitle>
            <CardDescription className='mt-1 line-clamp-2'>
              使用现有的旁挂字幕、资源字幕和已探测的内嵌字幕轨。
            </CardDescription>
          </div>
          <div className='flex shrink-0 flex-wrap items-center gap-2'>
            <Badge variant='secondary'>系统内置</Badge>
            <Button
              type='button'
              size='sm'
              variant='outline'
              disabled={!plugin}
              onClick={onOpenSettings}
            >
              <SlidersHorizontalIcon className='size-4' />
              设置
            </Button>
            <Button
              type='button'
              size='sm'
              variant='outline'
              disabled={pending || !plugin}
              onClick={onToggle}
            >
              {pending ? (
                <LoaderCircleIcon className='size-4 animate-spin' />
              ) : (
                <PowerIcon className='size-4' />
              )}
              {enabled ? '禁用' : '启用'}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4 text-sm'>
        <div className='flex flex-wrap gap-2'>
          <SubtitleMetaPill
            label='状态'
            value={enabled ? '已启用' : '已禁用'}
          />
          <SubtitleMetaPill
            label='外挂'
            value={externalFileEnabled ? '已启用' : '已关闭'}
          />
          <SubtitleMetaPill
            label='内嵌抽取'
            value={extractionEnabled ? '已启用' : '已关闭'}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function LocalSubtitleSettingsForm({
  plugin,
  pending,
  onToggleExternalFiles,
  onToggleEmbeddedExtraction,
}: {
  plugin?: InternalPlugin
  pending: boolean
  onToggleExternalFiles: () => void
  onToggleEmbeddedExtraction: () => void
}) {
  const enabled = plugin?.enabled ?? true
  const externalFileEnabled =
    plugin?.local_subtitle?.external_file_enabled ?? true
  const extractionEnabled =
    plugin?.local_subtitle?.embedded_extraction_enabled ?? false

  return (
    <div className='space-y-5'>
      <div className='rounded-lg border border-border/60 bg-muted/20 p-4'>
        <div className='space-y-1'>
          <div className='font-medium'>外挂字幕文件</div>
          <div className='text-sm text-muted-foreground'>
            控制是否使用已有的旁挂字幕、资源字幕等外挂字幕文件；关闭后播放页不会显示这类字幕轨。
          </div>
        </div>
        <div className='mt-4 flex flex-wrap items-center justify-between gap-3'>
          <div className='flex flex-wrap items-center gap-2'>
            <Badge variant={externalFileEnabled ? 'secondary' : 'outline'}>
              {externalFileEnabled ? '已启用' : '已关闭'}
            </Badge>
            {!enabled ? (
              <Badge variant='outline'>需先启用本地字幕</Badge>
            ) : null}
          </div>
          <Button
            type='button'
            variant='outline'
            disabled={pending || !plugin || !enabled}
            onClick={onToggleExternalFiles}
          >
            {pending ? (
              <LoaderCircleIcon className='size-4 animate-spin' />
            ) : (
              <PowerIcon className='size-4' />
            )}
            {externalFileEnabled ? '关闭外挂文件' : '开启外挂文件'}
          </Button>
        </div>
      </div>
      <div className='rounded-lg border border-border/60 bg-muted/20 p-4'>
        <div className='space-y-1'>
          <div className='font-medium'>内嵌字幕抽取</div>
          <div className='text-sm text-muted-foreground'>
            将可提取的内嵌字幕轨导出为可直接访问的字幕文件。关闭时不会使用抽取能力。
          </div>
        </div>
        <div className='mt-4 flex flex-wrap items-center justify-between gap-3'>
          <div className='flex flex-wrap items-center gap-2'>
            <Badge variant={extractionEnabled ? 'secondary' : 'outline'}>
              {extractionEnabled ? '已启用' : '已关闭'}
            </Badge>
            {!enabled ? (
              <Badge variant='outline'>需先启用本地字幕</Badge>
            ) : null}
          </div>
          <Button
            type='button'
            variant='outline'
            disabled={pending || !plugin || !enabled}
            onClick={onToggleEmbeddedExtraction}
          >
            {pending ? (
              <LoaderCircleIcon className='size-4 animate-spin' />
            ) : (
              <PowerIcon className='size-4' />
            )}
            {extractionEnabled ? '关闭抽取' : '开启抽取'}
          </Button>
        </div>
      </div>
    </div>
  )
}

function SubtitleProviderCard({
  provider,
  pending,
  onToggle,
  onEdit,
}: {
  provider: SubtitleProviderInstance
  pending: boolean
  onToggle: () => void
  onEdit: () => void
}) {
  const settings = provider.opensubtitles
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='gap-3 px-4 py-3'>
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <CardTitle className='flex items-center gap-2 truncate text-base'>
              <CloudIcon className='size-5' />
              {provider.name}
            </CardTitle>
            <CardDescription className='mt-1 truncate'>
              {provider.provider_type} · 实例 #{provider.id}
            </CardDescription>
          </div>
          <div className='flex shrink-0 flex-wrap items-center gap-2'>
            <Button
              type='button'
              size='sm'
              variant='outline'
              disabled={pending}
              onClick={onToggle}
            >
              {pending ? (
                <LoaderCircleIcon className='size-4 animate-spin' />
              ) : (
                <PowerIcon className='size-4' />
              )}
              {provider.enabled ? '禁用' : '启用'}
            </Button>
            <Button type='button' size='sm' variant='outline' onClick={onEdit}>
              <PencilIcon className='size-4' />
              编辑
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4 text-sm'>
        <SubtitleSummaryLine
          label='Base URL'
          value={settings?.base_url || '未配置'}
        />
        <div className='flex flex-wrap gap-2'>
          <SubtitleMetaPill
            label='状态'
            value={provider.enabled ? '已启用' : '已禁用'}
          />
          <SubtitleMetaPill
            label='配置'
            value={provider.configured ? '已配置' : '未配置'}
          />
          <SubtitleMetaPill
            label='API Key'
            value={`${settings?.api_key_count ?? 0} 个`}
          />
          <SubtitleMetaPill
            label='语言'
            value={settings?.languages || '未配置'}
          />
          <SubtitleMetaPill
            label='超时'
            value={settings?.timeout || '未配置'}
          />
          <SubtitleMetaPill
            label='来源'
            value={settings?.source || 'database'}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function SubtitleSummaryLine({
  label,
  value,
}: {
  label: string
  value: string
}) {
  return (
    <div className='rounded-lg border border-border/60 bg-muted/15 px-3 py-2'>
      <div className='text-[11px] text-muted-foreground'>{label}</div>
      <div className='mt-1 truncate font-mono text-xs text-foreground'>
        {value}
      </div>
    </div>
  )
}

function SubtitleMetaPill({ label, value }: { label: string; value: string }) {
  return (
    <span className='inline-flex h-6 min-w-0 items-center rounded-md border border-border/60 bg-muted/20 px-2 text-xs text-muted-foreground'>
      <span className='text-muted-foreground/80'>{label}</span>
      <span className='mx-1 text-border'>/</span>
      <span className='truncate text-foreground'>{value}</span>
    </span>
  )
}

function SubtitleProviderForm({
  draft,
  onChange,
}: {
  draft: SubtitleProviderDraft
  onChange: (nextDraft: SubtitleProviderDraft) => void
}) {
  return (
    <div className='space-y-5 py-2'>
      <FieldGroup>
        <div className='grid gap-4 md:grid-cols-2'>
          <Field>
            <FieldLabel>名称</FieldLabel>
            <Input
              value={draft.name}
              onChange={(event) =>
                onChange({ ...draft, name: event.target.value })
              }
              placeholder='例如：opensubtitles-primary'
            />
          </Field>
          <Field>
            <FieldLabel>提供方类型</FieldLabel>
            <Select value={draft.providerType} disabled>
              <SelectTrigger className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='opensubtitles'>OpenSubtitles</SelectItem>
              </SelectContent>
            </Select>
            <FieldDescription>
              当前字幕提供方仅支持 OpenSubtitles。
            </FieldDescription>
          </Field>
        </div>

        <div className='grid gap-4 md:grid-cols-2'>
          <label className='flex items-center justify-between gap-3 rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3'>
            <span className='text-sm font-medium text-foreground'>
              启用实例
            </span>
            <Switch
              checked={draft.enabled}
              onCheckedChange={(checked) =>
                onChange({ ...draft, enabled: checked })
              }
            />
          </label>
          <Field>
            <FieldLabel>可用状态</FieldLabel>
            <Input
              value={draft.availabilityStatus}
              onChange={(event) =>
                onChange({ ...draft, availabilityStatus: event.target.value })
              }
              placeholder='available / cooldown / unavailable'
            />
          </Field>
        </div>

        <Field>
          <FieldLabel>失败原因</FieldLabel>
          <Textarea
            value={draft.failureReason}
            onChange={(event) =>
              onChange({ ...draft, failureReason: event.target.value })
            }
            placeholder='可选，记录认证失败或限流原因'
          />
        </Field>

        <Field>
          <FieldLabel>冷却截止时间</FieldLabel>
          <Input
            value={draft.cooldownUntil}
            onChange={(event) =>
              onChange({ ...draft, cooldownUntil: event.target.value })
            }
            placeholder='2026-04-29T12:00:00Z'
          />
        </Field>

        <Separator />

        <div className='space-y-5'>
          <div>
            <div className='text-sm font-medium text-foreground'>
              OpenSubtitles 实例
            </div>
            <div className='mt-1 text-sm text-muted-foreground'>
              保存 OpenSubtitles API 连接参数；每个实例只允许一个 Key。
            </div>
          </div>

          <Field>
            <FieldLabel>API 密钥 / 令牌</FieldLabel>
            <Input
              type='password'
              value={draft.opensubtitles.apiKey}
              disabled={draft.opensubtitles.clearAPIKey}
              placeholder='已配置则留空保持当前密钥'
              onChange={(event) =>
                onChange({
                  ...draft,
                  opensubtitles: {
                    ...draft.opensubtitles,
                    apiKey: event.target.value,
                    clearAPIKey: false,
                  },
                })
              }
            />
            <FieldDescription>
              每个 OpenSubtitles 实例只保存一个 API
              Key。需要轮换时请创建多个实例。
            </FieldDescription>
          </Field>

          <label className='flex items-start gap-3 rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3'>
            <Checkbox
              checked={draft.opensubtitles.clearAPIKey}
              onCheckedChange={(checked) =>
                onChange({
                  ...draft,
                  opensubtitles: {
                    ...draft.opensubtitles,
                    apiKey: '',
                    clearAPIKey: checked === true,
                  },
                })
              }
              className='mt-1'
            />
            <div className='space-y-1'>
              <div className='text-sm font-medium text-foreground'>
                清除已保存密钥
              </div>
              <div className='text-sm text-muted-foreground'>
                保存时会删除该实例数据库中的密钥记录。
              </div>
            </div>
          </label>

          <div className='grid gap-4 md:grid-cols-2'>
            <Field>
              <FieldLabel>基础 URL</FieldLabel>
              <Input
                value={draft.opensubtitles.baseURL}
                placeholder='https://api.opensubtitles.com/api/v1'
                onChange={(event) =>
                  onChange({
                    ...draft,
                    opensubtitles: {
                      ...draft.opensubtitles,
                      baseURL: event.target.value,
                    },
                  })
                }
              />
            </Field>
            <Field>
              <FieldLabel>语言</FieldLabel>
              <Input
                value={draft.opensubtitles.languages}
                placeholder='zh-cn,zh-tw,en'
                onChange={(event) =>
                  onChange({
                    ...draft,
                    opensubtitles: {
                      ...draft.opensubtitles,
                      languages: event.target.value,
                    },
                  })
                }
              />
              <FieldDescription>
                例如 `zh-cn`、`zh-tw`、`en`，多个语言用逗号分隔。
              </FieldDescription>
            </Field>
          </div>

          <Field>
            <FieldLabel>超时时间</FieldLabel>
            <Input
              value={draft.opensubtitles.timeout}
              placeholder='10s'
              onChange={(event) =>
                onChange({
                  ...draft,
                  opensubtitles: {
                    ...draft.opensubtitles,
                    timeout: event.target.value,
                  },
                })
              }
            />
            <FieldDescription>
              使用 Go 时长格式，例如 `10s`、`30s`、`1m`。
            </FieldDescription>
          </Field>
        </div>
      </FieldGroup>
    </div>
  )
}

function buildSubtitleProviderDraft(
  provider: SubtitleProviderInstance
): SubtitleProviderDraft {
  return {
    id: provider.id,
    name: provider.name,
    providerType: 'opensubtitles',
    enabled: provider.enabled,
    availabilityStatus: provider.availability_status || 'available',
    failureReason: provider.failure_reason || '',
    cooldownUntil: provider.cooldown_until || '',
    opensubtitles: {
      apiKey: '',
      baseURL:
        provider.opensubtitles?.base_url ||
        EMPTY_SUBTITLE_PROVIDER_DRAFT.opensubtitles.baseURL,
      languages:
        provider.opensubtitles?.languages ||
        EMPTY_SUBTITLE_PROVIDER_DRAFT.opensubtitles.languages,
      timeout:
        provider.opensubtitles?.timeout ||
        EMPTY_SUBTITLE_PROVIDER_DRAFT.opensubtitles.timeout,
      clearAPIKey: false,
    },
  }
}

function buildSubtitleProviderInput(
  draft: SubtitleProviderDraft
): SubtitleProviderInstanceInput {
  const input: SubtitleProviderInstanceInput = {
    name: draft.name.trim(),
    provider_type: draft.providerType,
    enabled: draft.enabled,
    availability_status: draft.availabilityStatus.trim() || 'available',
    failure_reason: draft.failureReason.trim(),
    opensubtitles: buildOpenSubtitlesInput(draft.opensubtitles),
  }
  if (draft.cooldownUntil.trim()) {
    input.cooldown_until = draft.cooldownUntil.trim()
  }
  return input
}

function buildOpenSubtitlesInput(
  draft: SubtitleProviderDraft['opensubtitles']
): OpenSubtitlesProviderSettingsInput {
  return {
    api_key: draft.apiKey,
    clear_api_key: draft.clearAPIKey,
    base_url: draft.baseURL,
    languages: draft.languages,
    timeout: draft.timeout,
  }
}
