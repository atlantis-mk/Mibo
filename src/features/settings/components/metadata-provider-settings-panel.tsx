import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  GripVerticalIcon,
  InfoIcon,
  LoaderCircleIcon,
  PencilIcon,
  PlusIcon,
} from 'lucide-react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import type {
  MetadataProfile,
  MetadataProfileInput,
  MetadataProviderInput,
  MetadataProviderInstance,
  MetadataProviderInstanceInput,
  MetadataProviderSettings,
  PluginManifest,
  PluginProviderInstance,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  metadataProfilesQueryOptions,
  metadataProviderInstancesQueryOptions,
  miboQueryKeys,
  pluginProviderInstancesQueryOptions,
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
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  buildPluginConfigurationDefaults,
  PluginConfigurationForm,
} from './plugin-configuration-form'

type MetadataProviderFormState = {
  apiKey: string
  clearApiKey: boolean
  baseURL: string
  imageBaseURL: string
  language: string
  timeout: string
  retryCount: string
  upstreamProviderFilter: string
  fallbackEnabled: boolean
}

type EditableProviderType = 'tmdb' | 'tvdb' | 'metatube'

type ProviderInstanceDraft = {
  name: string
  providerType: string
  enabled: boolean
  availabilityStatus: string
  failureReason: string
  cooldownDuration: string
  tmdb: MetadataProviderFormState
  tvdb: MetadataProviderFormState
  metatube: MetadataProviderFormState
}

type MetadataProfileDraft = {
  name: string
  description: string
  searchProviderRefs: string[]
  detailProviderRefs: string[]
  preferredMetadataLanguage: string
  fallbackEnabled: boolean
}

export type PluginProviderDraft = {
  name: string
  endpoint: string
  enabled: boolean
  manifest: PluginManifest | null
  configuration: Record<string, unknown>
}

type ProviderOption = {
  ref: string
  id: number
  label: string
  description: string
}

const EMPTY_PROVIDER_FORM: MetadataProviderFormState = {
  apiKey: '',
  clearApiKey: false,
  baseURL: '',
  imageBaseURL: '',
  language: '',
  timeout: '',
  retryCount: '',
  upstreamProviderFilter: '',
  fallbackEnabled: true,
}

const DEFAULT_PROVIDER_FORMS: Record<
  EditableProviderType,
  MetadataProviderFormState
> = {
  tmdb: {
    ...EMPTY_PROVIDER_FORM,
    baseURL: 'https://api.themoviedb.org/3',
    imageBaseURL: 'https://image.tmdb.org/t/p/original',
    language: 'zh-CN',
    timeout: '30s',
    retryCount: '2',
  },
  tvdb: {
    ...EMPTY_PROVIDER_FORM,
    baseURL: 'https://api4.thetvdb.com/v4',
    language: 'en',
    timeout: '10s',
    retryCount: '0',
  },
  metatube: {
    ...EMPTY_PROVIDER_FORM,
    baseURL: 'http://127.0.0.1:8081',
    timeout: '10s',
    retryCount: '0',
    fallbackEnabled: true,
  },
}

const EMPTY_PROVIDER_INSTANCE_DRAFT: ProviderInstanceDraft = {
  name: '',
  providerType: 'tmdb',
  enabled: true,
  availabilityStatus: 'available',
  failureReason: '',
  cooldownDuration: '',
  tmdb: DEFAULT_PROVIDER_FORMS.tmdb,
  tvdb: DEFAULT_PROVIDER_FORMS.tvdb,
  metatube: DEFAULT_PROVIDER_FORMS.metatube,
}

const EMPTY_METADATA_PROFILE_DRAFT: MetadataProfileDraft = {
  name: '',
  description: '',
  searchProviderRefs: [],
  detailProviderRefs: [],
  preferredMetadataLanguage: '',
  fallbackEnabled: true,
}

export const EMPTY_PLUGIN_PROVIDER_DRAFT: PluginProviderDraft = {
  name: '',
  endpoint: '',
  enabled: true,
  manifest: null,
  configuration: {},
}

const HIDDEN_PROVIDER_TYPES = new Set(['local_scan'])

export function MetadataProviderSettingsPanel({
  token,
}: {
  token: string | null
}) {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<'providers' | 'profiles'>(
    'providers'
  )
  const [providerDialogOpen, setProviderDialogOpen] = useState(false)
  const [providerInstanceDraft, setProviderInstanceDraft] =
    useState<ProviderInstanceDraft>(EMPTY_PROVIDER_INSTANCE_DRAFT)
  const [editingProviderInstanceId, setEditingProviderInstanceId] = useState<
    number | null
  >(null)
  const [profileDialogOpen, setProfileDialogOpen] = useState(false)
  const [profileDraft, setProfileDraft] = useState<MetadataProfileDraft>(
    EMPTY_METADATA_PROFILE_DRAFT
  )
  const [editingProfileId, setEditingProfileId] = useState<number | null>(null)

  const providerInstancesQuery = useQuery({
    ...metadataProviderInstancesQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const pluginProviderInstancesQuery = useQuery({
    ...pluginProviderInstancesQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const profilesQuery = useQuery({
    ...metadataProfilesQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })

  const providerInstances = useMemo(
    () => providerInstancesQuery.data ?? [],
    [providerInstancesQuery.data]
  )
  const configurableProviderInstances = useMemo(
    () =>
      providerInstances.filter(
        (item) => !HIDDEN_PROVIDER_TYPES.has(item.provider_type)
      ),
    [providerInstances]
  )
  const pluginProviderInstances = useMemo(
    () => pluginProviderInstancesQuery.data ?? [],
    [pluginProviderInstancesQuery.data]
  )
  const metadataProfiles = useMemo(
    () => profilesQuery.data ?? [],
    [profilesQuery.data]
  )

  const saveProviderInstanceMutation = useMutation({
    mutationFn: async (draft: ProviderInstanceDraft) => {
      if (!token) {
        throw new Error('当前未登录，无法保存提供方实例。')
      }
      const api = createAuthedMiboApi(token)
      const input = buildProviderInstanceInput(draft)
      if (editingProviderInstanceId) {
        return api.updateMetadataProviderInstance(
          editingProviderInstanceId,
          input
        )
      }
      return api.createMetadataProviderInstance(input)
    },
    onSuccess: async () => {
      if (!token) return
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.metadataProviderInstances(token),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.pluginProviderInstances(token),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.metadataProfiles(token),
        }),
      ])
      resetProviderInstanceDialog()
      toast.success('元数据提供方实例已保存')
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  const saveProfileMutation = useMutation({
    mutationFn: async (draft: MetadataProfileDraft) => {
      if (!token) {
        throw new Error('当前未登录，无法保存元数据模板。')
      }
      const api = createAuthedMiboApi(token)
      const input = buildMetadataProfileInput(draft)
      if (editingProfileId) {
        return api.updateMetadataProfile(editingProfileId, input)
      }
      return api.createMetadataProfile(input)
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.metadataProfiles(token),
      })
      resetProfileDialog()
      toast.success('元数据模板已保存')
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  const searchProviderOptions = useMemo(
    () =>
      buildStageProviderOptions(
        configurableProviderInstances,
        pluginProviderInstances,
        'metadata.search'
      ),
    [configurableProviderInstances, pluginProviderInstances]
  )
  const detailProviderOptions = useMemo(
    () =>
      buildStageProviderOptions(
        configurableProviderInstances,
        pluginProviderInstances,
        'metadata.detail'
      ),
    [configurableProviderInstances, pluginProviderInstances]
  )
  const metadataProfileProviderLookup = useMemo(
    () =>
      Array.from(
        new Map(
          [...searchProviderOptions, ...detailProviderOptions].map((item) => [
            item.ref,
            item,
          ])
        ).values()
      ),
    [searchProviderOptions, detailProviderOptions]
  )

  if (!token) {
    return (
      <Alert>
        <InfoIcon className='size-4' />
        <AlertTitle>登录后可管理元数据配置</AlertTitle>
        <AlertDescription className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
          <span>
            当前页面需要管理员会话来读取和更新元数据提供方实例与元数据模板。
          </span>
          <Button asChild variant='outline'>
            <Link
              to='/sign-in'
              search={{
                redirect: '/settings/metadata-sources',
              }}
            >
              前往登录
            </Link>
          </Button>
        </AlertDescription>
      </Alert>
    )
  }

  if (
    providerInstancesQuery.isLoading ||
    pluginProviderInstancesQuery.isLoading ||
    profilesQuery.isLoading
  ) {
    return (
      <div className='flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm'>
        <LoaderCircleIcon className='size-4 animate-spin' />
        正在加载元数据管理配置
      </div>
    )
  }

  const error =
    providerInstancesQuery.error ||
    pluginProviderInstancesQuery.error ||
    profilesQuery.error
  if (error) {
    return (
      <Alert variant='destructive'>
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{error.message}</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className='space-y-4 pb-20'>
      <div className='flex justify-center'>
        <Tabs
          value={activeTab}
          onValueChange={(value) =>
            setActiveTab(value as 'providers' | 'profiles')
          }
        >
          <TabsList>
            <TabsTrigger value='providers'>提供方实例</TabsTrigger>
            <TabsTrigger value='profiles'>元数据模板</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {activeTab === 'providers' ? (
        <div className='space-y-4'>
          <div className='grid gap-4 md:grid-cols-2 2xl:grid-cols-3'>
            {configurableProviderInstances.map((provider) => (
              <ProviderInstanceCard
                key={provider.id}
                provider={provider}
                onEdit={() => {
                  setEditingProviderInstanceId(provider.id)
                  setProviderInstanceDraft(buildProviderInstanceDraft(provider))
                  setProviderDialogOpen(true)
                }}
              />
            ))}
            {pluginProviderInstances.map((provider) => (
              <PluginProviderSummaryCard
                key={`plugin-${provider.id}`}
                provider={provider}
              />
            ))}
            {configurableProviderInstances.length === 0 ? (
              pluginProviderInstances.length === 0 ? (
                <EmptyManagementCard text='还没有元数据提供方实例。' />
              ) : null
            ) : null}
          </div>
        </div>
      ) : null}

      {activeTab === 'profiles' ? (
        <div className='space-y-4'>
          <div className='grid gap-4 md:grid-cols-2 2xl:grid-cols-3'>
            {metadataProfiles.map((profile) => (
              <MetadataProfileCard
                key={profile.id}
                profile={profile}
                providerLookup={metadataProfileProviderLookup}
                onEdit={() => {
                  setEditingProfileId(profile.id)
                  setProfileDraft(buildMetadataProfileDraft(profile))
                  setProfileDialogOpen(true)
                }}
              />
            ))}
            {metadataProfiles.length === 0 ? (
              <EmptyManagementCard text='还没有元数据模板。创建模板后，媒体库可以复制默认提供方顺序与语言设置。' />
            ) : null}
          </div>
        </div>
      ) : null}

      <Dialog open={providerDialogOpen} onOpenChange={setProviderDialogOpen}>
        <DialogContent className='flex max-h-[90vh] flex-col overflow-hidden sm:max-w-2xl'>
          <DialogHeader>
            <DialogTitle>
              {editingProviderInstanceId ? '编辑' : '新建'}提供方实例
            </DialogTitle>
            <DialogDescription>
              每个实例都是一个可单独启用、绑定和降级的运行时元数据提供方。
            </DialogDescription>
          </DialogHeader>

          <div className='-mx-4 min-h-0 flex-1 overflow-y-auto px-4'>
            <ProviderInstanceForm
              draft={providerInstanceDraft}
              providerTypeLocked={editingProviderInstanceId !== null}
              onChange={setProviderInstanceDraft}
            />
          </div>

          <DialogFooter className='shrink-0'>
            <Button
              type='button'
              variant='outline'
              onClick={resetProviderInstanceDialog}
              disabled={saveProviderInstanceMutation.isPending}
            >
              取消
            </Button>
            <Button
              type='button'
              onClick={() =>
                saveProviderInstanceMutation.mutate(providerInstanceDraft)
              }
              disabled={
                saveProviderInstanceMutation.isPending ||
                !providerInstanceDraft.name.trim()
              }
            >
              {saveProviderInstanceMutation.isPending ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {createPortal(
        <div className='fixed right-6 bottom-6 z-[100] flex justify-end'>
          <Button
            type='button'
            onClick={() => {
              if (activeTab === 'providers') {
                setEditingProviderInstanceId(null)
                setProviderInstanceDraft(EMPTY_PROVIDER_INSTANCE_DRAFT)
                setProviderDialogOpen(true)
                return
              }
              setEditingProfileId(null)
              setProfileDraft(EMPTY_METADATA_PROFILE_DRAFT)
              setProfileDialogOpen(true)
            }}
          >
            <PlusIcon className='size-4' />
            {activeTab === 'providers' ? '新建提供方实例' : '新建元数据模板'}
          </Button>
        </div>,
        document.body
      )}

      <Dialog open={profileDialogOpen} onOpenChange={setProfileDialogOpen}>
        <DialogContent className='flex max-h-[90vh] flex-col overflow-hidden sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>
              {editingProfileId ? '编辑' : '新建'}元数据模板
            </DialogTitle>
            <DialogDescription>
              为搜索、详情、图片、人物、层级同步分别指定提供方实例
              顺序。应用到媒体库时会复制为该库自己的可执行策略。
            </DialogDescription>
          </DialogHeader>

          <div className='-mx-4 min-h-0 flex-1 overflow-y-auto px-4'>
            <MetadataProfileForm
              draft={profileDraft}
              searchProviderOptions={searchProviderOptions}
              detailProviderOptions={detailProviderOptions}
              onChange={setProfileDraft}
            />
          </div>

          <DialogFooter className='shrink-0'>
            <Button
              type='button'
              variant='outline'
              onClick={resetProfileDialog}
              disabled={saveProfileMutation.isPending}
            >
              取消
            </Button>
            <Button
              type='button'
              onClick={() => saveProfileMutation.mutate(profileDraft)}
              disabled={
                saveProfileMutation.isPending || !profileDraft.name.trim()
              }
            >
              {saveProfileMutation.isPending ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )

  function resetProviderInstanceDialog() {
    setProviderDialogOpen(false)
    setEditingProviderInstanceId(null)
    setProviderInstanceDraft(EMPTY_PROVIDER_INSTANCE_DRAFT)
  }

  function resetProfileDialog() {
    setProfileDialogOpen(false)
    setEditingProfileId(null)
    setProfileDraft(EMPTY_METADATA_PROFILE_DRAFT)
  }
}

function ProviderInstanceCard({
  provider,
  onEdit,
}: {
  provider: MetadataProviderInstance
  onEdit: () => void
}) {
  const settings = providerSettings(provider)
  const providerType = provider.provider_type.toUpperCase()
  const baseValue =
    provider.provider_type === 'local_scan'
      ? '复用扫描器记录的旁挂元数据证据'
      : provider.provider_type === 'metatube'
        ? provider.metatube?.upstream_provider_filter || '未指定上游'
        : settings?.base_url || '未配置'
  const secondaryValue =
    provider.provider_type === 'metatube'
      ? provider.metatube?.api_key_masked
        ? '令牌已配置'
        : '令牌未配置'
      : provider.provider_type === 'local_scan'
        ? '无需额外配置'
        : settings?.language || '未配置'

  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='gap-3 px-4 py-3'>
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <CardTitle className='truncate text-base'>
              {provider.name}
            </CardTitle>
            <CardDescription className='mt-1 truncate'>
              {providerType} · 实例 #{provider.id}
            </CardDescription>
          </div>
          <div className='flex shrink-0 items-center gap-2'>
            {provider.system_managed ? (
              <Badge variant='outline'>系统</Badge>
            ) : null}
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={onEdit}
              disabled={provider.locked}
            >
              <PencilIcon className='size-4' />
              {provider.locked ? '只读' : '编辑'}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4 text-sm text-muted-foreground'>
        <div className='grid gap-2'>
          <ProviderSummaryLine
            label={
              provider.provider_type === 'metatube'
                ? '上游提供方'
                : provider.provider_type === 'local_scan'
                  ? '内置能力'
                  : '基础 URL'
            }
            value={baseValue}
          />
          <ProviderSummaryLine
            label={
              provider.provider_type === 'metatube'
                ? '认证状态'
                : provider.provider_type === 'local_scan'
                  ? '配置'
                  : '语言'
            }
            value={secondaryValue}
          />
        </div>
        <div className='flex flex-wrap gap-2'>
          <ProviderMetaPill
            label='状态'
            value={provider.enabled ? '已启用' : '已禁用'}
          />
          <ProviderMetaPill
            label='可用'
            value={formatAvailabilityLabel(provider.availability_status)}
          />
          <ProviderMetaPill
            label='配置'
            value={provider.configured ? '已配置' : '未配置'}
          />
          {provider.provider_type !== 'local_scan' ? (
            <>
              <ProviderMetaPill
                label='超时'
                value={settings?.timeout || '未配置'}
              />
              <ProviderMetaPill
                label='重试'
                value={String(settings?.retry_count ?? 0)}
              />
            </>
          ) : null}
          {provider.provider_type === 'metatube' ? (
            <ProviderMetaPill
              label='回退'
              value={provider.metatube?.fallback_enabled ? '允许' : '不允许'}
            />
          ) : null}
          {provider.provider_type === 'tvdb' ? (
            <ProviderMetaPill label='执行' value='暂未接入' />
          ) : null}
        </div>
        {provider.provider_type === 'tmdb' ? (
          <ProviderSummaryLine
            label='图片基础 URL'
            value={provider.tmdb?.image_base_url || '未配置'}
          />
        ) : null}
        {provider.failure_reason ? (
          <div className='rounded-lg border border-amber-500/20 bg-amber-500/5 px-3 py-2 text-sm [overflow-wrap:anywhere] text-amber-700 dark:text-amber-300'>
            最近失败原因：{provider.failure_reason}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

function MetadataProfileCard({
  profile,
  providerLookup,
  onEdit,
}: {
  profile: MetadataProfile
  providerLookup: ProviderOption[]
  onEdit: () => void
}) {
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='gap-3 px-4 py-3'>
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <CardTitle className='truncate text-base'>{profile.name}</CardTitle>
            {!profile.locked ? (
              <CardDescription className='mt-1 truncate'>
                {profile.description || '未填写描述'}
              </CardDescription>
            ) : null}
            {profile.locked ? (
              <div className='mt-2 text-xs text-muted-foreground'>
                这是系统迁移时期保留的只读模板，用于兼容旧的仅本地行为。
              </div>
            ) : null}
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <Badge variant='outline'>模板</Badge>
            {profile.system ? <Badge variant='outline'>系统</Badge> : null}
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={onEdit}
              disabled={profile.locked}
            >
              <PencilIcon className='size-4' />
              {profile.locked ? '只读' : '编辑'}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4 text-sm text-muted-foreground'>
        <div className='grid gap-2'>
          <ProviderSummaryLine
            label='搜索阶段'
            value={formatProviderRefs(
              profile.search_provider_refs,
              providerLookup
            )}
          />
          <ProviderSummaryLine
            label='详情阶段'
            value={formatProviderRefs(
              profile.detail_provider_refs,
              providerLookup
            )}
          />
        </div>
        <div className='flex flex-wrap gap-2'>
          <ProviderMetaPill
            label='语言'
            value={profile.preferred_metadata_language || '未指定'}
          />
          <ProviderMetaPill
            label='回退'
            value={profile.fallback_enabled ? '允许' : '禁止'}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function PluginProviderSummaryCard({
  provider,
}: {
  provider: PluginProviderInstance
}) {
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='gap-3 px-4 py-3'>
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <CardTitle className='truncate text-base'>
              {provider.name}
            </CardTitle>
            <CardDescription className='mt-1 truncate'>
              {provider.plugin_name} · {provider.deployment_kind} · #
              {provider.id}
            </CardDescription>
          </div>
          <div className='flex shrink-0 flex-wrap items-center gap-2'>
            <Badge variant='outline'>插件</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4 text-sm text-muted-foreground'>
        <ProviderSummaryLine label='端点' value={provider.endpoint} />
        <div className='flex flex-wrap gap-2'>
          <ProviderMetaPill
            label='状态'
            value={provider.enabled ? '已启用' : '已禁用'}
          />
          <ProviderMetaPill
            label='可用'
            value={formatAvailabilityLabel(provider.availability_status)}
          />
          <ProviderMetaPill label='部署' value={provider.deployment_kind} />
        </div>
        <div className='flex flex-wrap gap-2'>
          {(provider.capabilities ?? []).map((capability) => (
            <Badge key={capability} variant='outline'>
              {capability}
            </Badge>
          ))}
        </div>
        {provider.failure_reason ? (
          <div className='rounded-lg border border-amber-500/20 bg-amber-500/5 px-3 py-2 text-sm [overflow-wrap:anywhere] text-amber-700 dark:text-amber-300'>
            最近失败原因：{provider.failure_reason}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

function ProviderSummaryLine({
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

function ProviderMetaPill({ label, value }: { label: string; value: string }) {
  return (
    <span className='inline-flex h-6 min-w-0 items-center rounded-md border border-border/60 bg-muted/20 px-2 text-xs text-muted-foreground'>
      <span className='text-muted-foreground/80'>{label}</span>
      <span className='mx-1 text-border'>/</span>
      <span className='truncate text-foreground'>{value}</span>
    </span>
  )
}

function ProviderInstanceForm({
  draft,
  providerTypeLocked,
  onChange,
}: {
  draft: ProviderInstanceDraft
  providerTypeLocked: boolean
  onChange: (nextDraft: ProviderInstanceDraft) => void
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
              placeholder='例如：anime-tmdb-primary'
            />
          </Field>
          <Field>
            <FieldLabel>提供方类型</FieldLabel>
            <Select
              value={draft.providerType}
              disabled={providerTypeLocked}
              onValueChange={(providerType) =>
                onChange(applyProviderTypeDefaults(draft, providerType))
              }
            >
              <SelectTrigger className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='tmdb'>TMDB</SelectItem>
                <SelectItem value='tvdb'>TVDB</SelectItem>
                <SelectItem value='metatube'>MetaTube</SelectItem>
              </SelectContent>
            </Select>
            <FieldDescription>
              MetaTube 支持电影搜索和详情；TVDB 当前仅保存配置。
            </FieldDescription>
          </Field>
        </div>

        <div className='grid gap-4 md:grid-cols-2'>
          <ToggleField
            label='启用实例'
            checked={draft.enabled}
            onChange={(checked) => onChange({ ...draft, enabled: checked })}
          />
          <Field>
            <FieldLabel>可用状态</FieldLabel>
            <Input
              value={draft.availabilityStatus}
              onChange={(event) =>
                onChange({ ...draft, availabilityStatus: event.target.value })
              }
              placeholder='例如：available / cooldown / unavailable'
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
          <FieldLabel>冷却时长</FieldLabel>
          <Input
            value={draft.cooldownDuration}
            onChange={(event) =>
              onChange({ ...draft, cooldownDuration: event.target.value })
            }
            placeholder='例如：15m / 1h'
          />
          <FieldDescription>
            保存时会自动换算成冷却截止时间；留空表示不设置冷却。
          </FieldDescription>
        </Field>

        <Separator />

        {draft.providerType === 'tvdb' ? (
          <MetadataProviderEditFields
            title='TVDB 实例'
            draft={draft.tvdb}
            onChange={(field, value) =>
              onChange({
                ...draft,
                tvdb: {
                  ...draft.tvdb,
                  [field]: value,
                },
              })
            }
            showClearKey={false}
          />
        ) : draft.providerType === 'metatube' ? (
          <MetadataProviderEditFields
            title='MetaTube 实例'
            draft={draft.metatube}
            onChange={(field, value) =>
              onChange({
                ...draft,
                metatube: {
                  ...draft.metatube,
                  [field]: value,
                },
              })
            }
            includeLanguage={false}
            includeMetaTubeFields
            showClearKey={false}
          />
        ) : (
          <MetadataProviderEditFields
            title='TMDB 实例'
            draft={draft.tmdb}
            onChange={(field, value) =>
              onChange({
                ...draft,
                tmdb: {
                  ...draft.tmdb,
                  [field]: value,
                },
              })
            }
            includeImageBaseURL
            showClearKey={false}
          />
        )}
      </FieldGroup>
    </div>
  )
}

function MetadataProfileForm({
  draft,
  searchProviderOptions,
  detailProviderOptions,
  onChange,
}: {
  draft: MetadataProfileDraft
  searchProviderOptions: ProviderOption[]
  detailProviderOptions: ProviderOption[]
  onChange: (nextDraft: MetadataProfileDraft) => void
}) {
  return (
    <div className='space-y-5 py-2'>
      <FieldGroup>
        <div className='grid gap-4 md:grid-cols-2'>
          <Field>
            <FieldLabel>模板名称</FieldLabel>
            <Input
              value={draft.name}
              onChange={(event) =>
                onChange({ ...draft, name: event.target.value })
              }
              placeholder='movie-default'
            />
          </Field>
        </div>

        <ToggleField
          label='允许回退'
          checked={draft.fallbackEnabled}
          onChange={(checked) =>
            onChange({ ...draft, fallbackEnabled: checked })
          }
        />

        <Field>
          <FieldLabel>描述</FieldLabel>
          <Textarea
            value={draft.description}
            onChange={(event) =>
              onChange({ ...draft, description: event.target.value })
            }
            placeholder='说明这个模板适合哪些媒体库、使用哪些提供方顺序'
          />
        </Field>

        <StageField
          label='搜索阶段'
          value={draft.searchProviderRefs}
          providerOptions={searchProviderOptions}
          onChange={(value) =>
            onChange({ ...draft, searchProviderRefs: value })
          }
        />
        <StageField
          label='详情阶段'
          value={draft.detailProviderRefs}
          providerOptions={detailProviderOptions}
          onChange={(value) =>
            onChange({ ...draft, detailProviderRefs: value })
          }
        />

        <div className='grid gap-4 md:grid-cols-1'>
          <Field>
            <FieldLabel>默认元数据语言</FieldLabel>
            <Input
              value={draft.preferredMetadataLanguage}
              onChange={(event) =>
                onChange({
                  ...draft,
                  preferredMetadataLanguage: event.target.value,
                })
              }
              placeholder='zh-CN'
            />
          </Field>
        </div>
      </FieldGroup>
    </div>
  )
}

function StageField({
  label,
  value,
  providerOptions,
  onChange,
}: {
  label: string
  value: string[]
  providerOptions: ProviderOption[]
  onChange: (value: string[]) => void
}) {
  const [draggingRef, setDraggingRef] = useState<string | null>(null)
  const providerOptionRefs = new Set(providerOptions.map((item) => item.ref))
  const visibleValue = value.filter((ref) => providerOptionRefs.has(ref))

  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <div className='grid gap-3 rounded-[1rem] border border-border/60 bg-background/60 p-3'>
        <div className='text-xs text-muted-foreground'>
          勾选提供方进入当前阶段，拖动已选项可以调整执行顺序。
        </div>

        <div className='grid gap-2'>
          {visibleValue.length > 0 ? (
            visibleValue.map((ref) => {
              const provider = providerOptions.find((item) => item.ref === ref)
              return (
                <div
                  key={ref}
                  draggable
                  onDragStart={() => setDraggingRef(ref)}
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={() => {
                    if (draggingRef === null || draggingRef === ref) {
                      return
                    }
                    const next = [...visibleValue]
                    const fromIndex = next.indexOf(draggingRef)
                    const toIndex = next.indexOf(ref)
                    if (fromIndex < 0 || toIndex < 0) {
                      return
                    }
                    next.splice(fromIndex, 1)
                    next.splice(toIndex, 0, draggingRef)
                    onChange(next)
                    setDraggingRef(null)
                  }}
                  onDragEnd={() => setDraggingRef(null)}
                  className='flex items-center justify-between rounded-lg border border-border/60 bg-card px-3 py-2 text-sm'
                >
                  <div className='flex items-center gap-2'>
                    <GripVerticalIcon className='size-4 text-muted-foreground' />
                    <span>{provider?.label || ref}</span>
                  </div>
                  <Button
                    type='button'
                    variant='ghost'
                    size='sm'
                    onClick={() =>
                      onChange(visibleValue.filter((item) => item !== ref))
                    }
                  >
                    移除
                  </Button>
                </div>
              )
            })
          ) : (
            <div className='rounded-lg border border-dashed border-border/60 px-3 py-4 text-sm text-muted-foreground'>
              当前阶段还没有选中的提供方。
            </div>
          )}
        </div>

        <div className='grid gap-2 md:grid-cols-2'>
          {providerOptions.map((item) => {
            const checked = visibleValue.includes(item.ref)
            return (
              <label
                key={item.ref}
                className='flex items-center gap-3 rounded-lg border border-border/60 px-3 py-2 text-sm'
              >
                <Checkbox
                  checked={checked}
                  onCheckedChange={(nextChecked) => {
                    if (nextChecked === true) {
                      if (!checked) {
                        onChange([...visibleValue, item.ref])
                      }
                      return
                    }
                    onChange(
                      visibleValue.filter((selected) => selected !== item.ref)
                    )
                  }}
                />
                <span>{item.label}</span>
              </label>
            )
          })}
        </div>
      </div>
      <FieldDescription>
        {providerOptions.length > 0
          ? '每个阶段都可以选择不同的提供方，并用拖拽调整优先级。'
          : '还没有可用实例。请先创建提供方实例。'}
      </FieldDescription>
    </Field>
  )
}

export function PluginProviderForm({
  draft,
  pendingPreview,
  onPreview,
  onChange,
  requiredCapabilities,
  requiredCapabilityLabel,
}: {
  draft: PluginProviderDraft
  pendingPreview: boolean
  onPreview: () => void
  onChange: (nextDraft: PluginProviderDraft) => void
  requiredCapabilities?: string[]
  requiredCapabilityLabel?: string
}) {
  const missingRequiredCapability =
    draft.manifest !== null &&
    requiredCapabilities !== undefined &&
    !pluginManifestSupportsAny(draft.manifest, requiredCapabilities)

  return (
    <div className='space-y-5 py-2'>
      <FieldGroup>
        <Field>
          <FieldLabel>端点地址</FieldLabel>
          <div className='flex gap-2'>
            <Input
              value={draft.endpoint}
              onChange={(event) =>
                onChange({
                  ...draft,
                  endpoint: event.target.value,
                })
              }
              placeholder='https://plugin.example.com'
            />
            <Button
              type='button'
              variant='outline'
              onClick={onPreview}
              disabled={pendingPreview || !draft.endpoint.trim()}
            >
              {pendingPreview ? '获取中…' : '获取清单'}
            </Button>
          </div>
        </Field>

        <Field>
          <FieldLabel>实例名称</FieldLabel>
          <Input
            value={draft.name}
            onChange={(event) =>
              onChange({
                ...draft,
                name: event.target.value,
              })
            }
            placeholder='例如 anime-plugin-primary'
          />
        </Field>

        <ToggleField
          label='启用实例'
          checked={draft.enabled}
          onChange={(checked) => onChange({ ...draft, enabled: checked })}
        />
      </FieldGroup>

      {draft.manifest ? (
        <div className='space-y-4'>
          <div className='grid gap-3 md:grid-cols-2'>
            <MetadataValue label='插件 ID' value={draft.manifest.id} />
            <MetadataValue
              label='协议版本'
              value={draft.manifest.protocol_version}
            />
            <MetadataValue label='插件名称' value={draft.manifest.name} />
            <MetadataValue label='版本' value={draft.manifest.version} />
          </div>

          <Field>
            <FieldLabel>能力</FieldLabel>
            <div className='flex flex-wrap gap-2 rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3'>
              {(draft.manifest.capabilities ?? []).map((capability) => (
                <Badge key={capability.capability} variant='outline'>
                  {capability.capability}
                </Badge>
              ))}
            </div>
          </Field>

          {missingRequiredCapability ? (
            <Alert variant='destructive'>
              <AlertTitle>能力不匹配</AlertTitle>
              <AlertDescription>
                这个入口需要
                {requiredCapabilityLabel ?? requiredCapabilities?.join(' / ')}
                ，请换一个插件端点。
              </AlertDescription>
            </Alert>
          ) : null}

          <Field>
            <FieldLabel>插件配置</FieldLabel>
            <PluginConfigurationForm
              schema={draft.manifest.configuration_schema}
              value={draft.configuration}
              onChange={(configuration) =>
                onChange({
                  ...draft,
                  configuration,
                })
              }
            />
          </Field>
        </div>
      ) : (
        <div className='rounded-[1rem] border border-dashed border-border/60 bg-muted/30 px-4 py-4 text-sm text-muted-foreground'>
          先获取插件清单，才能查看 manifest 和可配置字段。
        </div>
      )}
    </div>
  )
}

function ToggleField({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <div className='flex items-center justify-between rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3 text-sm'>
      <span>{label}</span>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}

function MetadataProviderEditFields({
  title,
  settings,
  draft,
  onChange,
  includeImageBaseURL = false,
  includeLanguage = true,
  includeMetaTubeFields = false,
  showClearKey = true,
}: {
  title: string
  settings?: MetadataProviderSettings
  draft: MetadataProviderFormState
  onChange: <FieldKey extends keyof MetadataProviderFormState>(
    field: FieldKey,
    value: MetadataProviderFormState[FieldKey]
  ) => void
  includeImageBaseURL?: boolean
  includeLanguage?: boolean
  includeMetaTubeFields?: boolean
  showClearKey?: boolean
}) {
  return (
    <div className='space-y-5'>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor={`${title}-api-key`}>API 密钥 / 令牌</FieldLabel>
          <Input
            id={`${title}-api-key`}
            type='password'
            value={draft.apiKey}
            disabled={draft.clearApiKey}
            placeholder={
              settings?.api_key_masked
                ? '已配置，留空则保持当前密钥'
                : '输入新的 API 密钥或令牌'
            }
            className='border-border/60 bg-background text-foreground placeholder:text-muted-foreground'
            onChange={(event) => onChange('apiKey', event.target.value)}
          />
        </Field>

        {showClearKey ? (
          <label className='flex items-start gap-3 rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3'>
            <Checkbox
              checked={draft.clearApiKey}
              onCheckedChange={(checked) =>
                onChange('clearApiKey', checked === true)
              }
              className='mt-1'
            />
            <div className='space-y-1'>
              <div className='text-sm font-medium text-foreground'>
                清除已保存密钥
              </div>
              <div className='text-sm text-muted-foreground'>
                保存时会删除数据库中的密钥记录。若当前值来自环境变量，运行中的环境配置仍会继续生效。
              </div>
            </div>
          </label>
        ) : null}

        <div className='grid gap-4 md:grid-cols-2'>
          <Field>
            <FieldLabel htmlFor={`${title}-base-url`}>基础 URL</FieldLabel>
            <Input
              id={`${title}-base-url`}
              value={draft.baseURL}
              className='border-border/60 bg-background text-foreground placeholder:text-muted-foreground'
              onChange={(event) => onChange('baseURL', event.target.value)}
            />
          </Field>

          {includeLanguage ? (
            <Field>
              <FieldLabel htmlFor={`${title}-language`}>语言</FieldLabel>
              <Input
                id={`${title}-language`}
                value={draft.language}
                className='border-border/60 bg-background text-foreground placeholder:text-muted-foreground'
                onChange={(event) => onChange('language', event.target.value)}
              />
              <FieldDescription>例如 `zh-CN`、`en-US`、`zh`。</FieldDescription>
            </Field>
          ) : null}
        </div>

        {includeMetaTubeFields ? (
          <>
            <Field>
              <FieldLabel htmlFor={`${title}-upstream-provider`}>
                默认上游提供方
              </FieldLabel>
              <Input
                id={`${title}-upstream-provider`}
                value={draft.upstreamProviderFilter}
                placeholder='例如 fanza，可留空'
                className='border-border/60 bg-background text-foreground placeholder:text-muted-foreground'
                onChange={(event) =>
                  onChange('upstreamProviderFilter', event.target.value)
                }
              />
              <FieldDescription>
                手动指定 MetaTube 上游提供方过滤器；留空则由 MetaTube
                服务端决定。
              </FieldDescription>
            </Field>
            <ToggleField
              label='允许 MetaTube 回退'
              checked={draft.fallbackEnabled}
              onChange={(checked) => onChange('fallbackEnabled', checked)}
            />
          </>
        ) : null}

        {includeImageBaseURL ? (
          <Field>
            <FieldLabel htmlFor={`${title}-image-base-url`}>
              图片基础 URL
            </FieldLabel>
            <Input
              id={`${title}-image-base-url`}
              value={draft.imageBaseURL}
              className='border-border/60 bg-background text-foreground placeholder:text-muted-foreground'
              onChange={(event) => onChange('imageBaseURL', event.target.value)}
            />
          </Field>
        ) : null}

        <Field>
          <FieldLabel htmlFor={`${title}-timeout`}>超时时间</FieldLabel>
          <Input
            id={`${title}-timeout`}
            value={draft.timeout}
            className='border-border/60 bg-background text-foreground placeholder:text-muted-foreground'
            onChange={(event) => onChange('timeout', event.target.value)}
          />
          <FieldDescription>
            使用 Go 时长格式，例如 `10s`、`30s`、`1m`。
          </FieldDescription>
        </Field>

        <Field>
          <FieldLabel htmlFor={`${title}-retry-count`}>重试次数</FieldLabel>
          <Input
            id={`${title}-retry-count`}
            type='number'
            min='0'
            max='10'
            value={draft.retryCount}
            className='border-border/60 bg-background text-foreground placeholder:text-muted-foreground'
            onChange={(event) => onChange('retryCount', event.target.value)}
          />
          <FieldDescription>
            仅在临时网络错误或限流时重试，范围 `0-10`。
          </FieldDescription>
        </Field>
      </FieldGroup>
    </div>
  )
}

export function MetadataValue({
  label,
  value,
  className = '',
}: {
  label: string
  value: string
  className?: string
}) {
  return (
    <div
      className={`rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3 ${className}`}
    >
      <div className='text-xs font-medium tracking-wide text-muted-foreground uppercase'>
        {label}
      </div>
      <div className='mt-1 text-sm break-all text-foreground'>{value}</div>
    </div>
  )
}

export function EmptyManagementCard({ text }: { text: string }) {
  return (
    <Card className='rounded-[1.5rem] border-dashed border-border/60 bg-card/50 py-0 shadow-none'>
      <CardContent className='px-5 py-10 text-sm text-muted-foreground'>
        {text}
      </CardContent>
    </Card>
  )
}

export function buildProviderInstanceDraft(
  provider: MetadataProviderInstance
): ProviderInstanceDraft {
  return {
    name: provider.name,
    providerType: provider.provider_type,
    enabled: provider.enabled,
    availabilityStatus: provider.availability_status,
    failureReason: provider.failure_reason || '',
    cooldownDuration: formatCooldownDuration(provider.cooldown_until),
    tmdb: {
      ...DEFAULT_PROVIDER_FORMS.tmdb,
      baseURL: provider.tmdb?.base_url || '',
      imageBaseURL: provider.tmdb?.image_base_url || '',
      language: provider.tmdb?.language || '',
      timeout: provider.tmdb?.timeout || '',
      retryCount: String(provider.tmdb?.retry_count ?? ''),
    },
    tvdb: {
      ...DEFAULT_PROVIDER_FORMS.tvdb,
      baseURL: provider.tvdb?.base_url || '',
      language: provider.tvdb?.language || '',
      timeout: provider.tvdb?.timeout || '',
      retryCount: String(provider.tvdb?.retry_count ?? ''),
    },
    metatube: {
      ...DEFAULT_PROVIDER_FORMS.metatube,
      baseURL: provider.metatube?.base_url || '',
      timeout: provider.metatube?.timeout || '',
      retryCount: String(provider.metatube?.retry_count ?? ''),
      upstreamProviderFilter: provider.metatube?.upstream_provider_filter || '',
      fallbackEnabled:
        provider.metatube?.fallback_enabled ??
        DEFAULT_PROVIDER_FORMS.metatube.fallbackEnabled,
    },
  }
}

function applyProviderTypeDefaults(
  draft: ProviderInstanceDraft,
  providerType: string
): ProviderInstanceDraft {
  if (!isEditableProviderType(providerType)) {
    return { ...draft, providerType }
  }

  return {
    ...draft,
    providerType,
    [providerType]: {
      ...DEFAULT_PROVIDER_FORMS[providerType],
      ...draft[providerType],
      baseURL:
        draft[providerType].baseURL ||
        DEFAULT_PROVIDER_FORMS[providerType].baseURL,
      imageBaseURL:
        draft[providerType].imageBaseURL ||
        DEFAULT_PROVIDER_FORMS[providerType].imageBaseURL,
      language:
        draft[providerType].language ||
        DEFAULT_PROVIDER_FORMS[providerType].language,
      timeout:
        draft[providerType].timeout ||
        DEFAULT_PROVIDER_FORMS[providerType].timeout,
      retryCount:
        draft[providerType].retryCount ||
        DEFAULT_PROVIDER_FORMS[providerType].retryCount,
      upstreamProviderFilter:
        draft[providerType].upstreamProviderFilter ||
        DEFAULT_PROVIDER_FORMS[providerType].upstreamProviderFilter,
    },
  }
}

export function buildProviderInstanceInput(
  draft: ProviderInstanceDraft
): MetadataProviderInstanceInput {
  const input: MetadataProviderInstanceInput = {
    name: draft.name.trim(),
    provider_type: draft.providerType,
    enabled: draft.enabled,
    availability_status: draft.availabilityStatus.trim() || undefined,
    failure_reason: draft.failureReason.trim() || undefined,
    cooldown_until: buildCooldownUntil(draft.cooldownDuration),
  }
  switch (draft.providerType) {
    case 'tvdb':
      input.tvdb = buildProviderInput(draft.tvdb)
      break
    case 'metatube':
      input.metatube = buildProviderInput(draft.metatube)
      break
    default:
      input.tmdb = buildProviderInput(draft.tmdb)
  }
  return input
}

function buildProviderInput(
  draft: MetadataProviderFormState
): MetadataProviderInput {
  const retryCount = Number.parseInt(draft.retryCount, 10)
  return {
    api_key: draft.apiKey || undefined,
    clear_api_key: draft.clearApiKey || undefined,
    base_url: draft.baseURL || undefined,
    image_base_url: draft.imageBaseURL || undefined,
    language: draft.language || undefined,
    timeout: draft.timeout || undefined,
    retry_count: Number.isNaN(retryCount) ? undefined : retryCount,
    upstream_provider_filter: draft.upstreamProviderFilter || undefined,
    fallback_enabled: draft.fallbackEnabled,
  }
}

function isEditableProviderType(value: string): value is EditableProviderType {
  return value === 'tmdb' || value === 'tvdb' || value === 'metatube'
}

function buildCooldownUntil(value: string): string | undefined {
  const duration = parseGoDurationMs(value)
  if (duration == null || duration <= 0) {
    return undefined
  }
  return new Date(Date.now() + duration).toISOString()
}

function formatCooldownDuration(value?: string): string {
  if (!value) {
    return ''
  }
  const target = Date.parse(value)
  if (Number.isNaN(target)) {
    return ''
  }
  const remaining = Math.max(0, target - Date.now())
  if (remaining === 0) {
    return ''
  }
  const seconds = Math.ceil(remaining / 1000)
  if (seconds % 3600 === 0) {
    return `${seconds / 3600}h`
  }
  if (seconds % 60 === 0) {
    return `${seconds / 60}m`
  }
  return `${seconds}s`
}

function parseGoDurationMs(value: string): number | null {
  const trimmed = value.trim()
  if (!trimmed) {
    return null
  }
  const matches = [...trimmed.matchAll(/(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)/g)]
  if (matches.length === 0) {
    return null
  }
  const consumed = matches.map((match) => match[0]).join('')
  if (consumed !== trimmed) {
    return null
  }
  let total = 0
  for (const [, rawAmount, unit] of matches) {
    const amount = Number.parseFloat(rawAmount)
    if (Number.isNaN(amount)) {
      return null
    }
    switch (unit) {
      case 'h':
        total += amount * 60 * 60 * 1000
        break
      case 'm':
        total += amount * 60 * 1000
        break
      case 's':
        total += amount * 1000
        break
      case 'ms':
        total += amount
        break
      case 'us':
      case 'µs':
        total += amount / 1000
        break
      case 'ns':
        total += amount / 1_000_000
        break
      default:
        return null
    }
  }
  return total
}

export function buildMetadataProfileDraft(
  profile: MetadataProfile
): MetadataProfileDraft {
  return {
    name: profile.name,
    description: profile.description || '',
    searchProviderRefs: profile.search_provider_refs?.length
      ? profile.search_provider_refs
      : profile.search_provider_ids.map((id) => `builtin:${id}`),
    detailProviderRefs: profile.detail_provider_refs?.length
      ? profile.detail_provider_refs
      : profile.detail_provider_ids.map((id) => `builtin:${id}`),
    preferredMetadataLanguage: profile.preferred_metadata_language || '',
    fallbackEnabled: profile.fallback_enabled,
  }
}

export function buildMetadataProfileInput(
  draft: MetadataProfileDraft
): MetadataProfileInput {
  return {
    name: draft.name.trim(),
    description: draft.description.trim() || undefined,
    search_provider_ids: [],
    search_provider_refs: draft.searchProviderRefs,
    detail_provider_ids: [],
    detail_provider_refs: draft.detailProviderRefs,
    preferred_metadata_language:
      draft.preferredMetadataLanguage.trim() || undefined,
    fallback_enabled: draft.fallbackEnabled,
  }
}

export function buildPluginProviderDraft(
  provider: PluginProviderInstance
): PluginProviderDraft {
  return {
    name: provider.name,
    endpoint: provider.endpoint,
    enabled: provider.enabled,
    manifest: provider.manifest,
    configuration: buildPluginConfigurationDefaults(
      provider.manifest.configuration_schema,
      provider.configuration ?? {}
    ),
  }
}

function pluginManifestSupportsAny(
  manifest: PluginManifest | null,
  capabilities: string[]
) {
  if (!manifest) return false
  const declaredCapabilities = new Set(
    (manifest.capabilities ?? []).map((item) => item.capability)
  )
  return capabilities.some((capability) => declaredCapabilities.has(capability))
}

export function buildStageProviderOptions(
  builtins: MetadataProviderInstance[],
  plugins: PluginProviderInstance[],
  capability: 'metadata.search' | 'metadata.detail'
) {
  const builtinOptions: ProviderOption[] = builtins
    .filter(
      (provider) =>
        provider.enabled && supportsBuiltinCapability(provider, capability)
    )
    .map((provider) => ({
      ref: `builtin:${provider.id}`,
      id: provider.id,
      label: `${provider.name} (#${provider.id})`,
      description: provider.provider_type,
    }))

  const pluginOptions: ProviderOption[] = plugins
    .filter(
      (provider) =>
        provider.enabled &&
        (provider.capabilities ?? []).includes(capability) &&
        provider.availability_status !== 'unavailable'
    )
    .map((provider) => ({
      ref: `plugin:${provider.id}`,
      id: provider.id,
      label: `${provider.name} (#${provider.id})`,
      description: `${provider.plugin_name} · ${provider.deployment_kind}`,
    }))

  return [...builtinOptions, ...pluginOptions]
}

function supportsBuiltinCapability(
  provider: MetadataProviderInstance,
  capability: 'metadata.search' | 'metadata.detail'
) {
  if (provider.provider_type === 'tvdb') {
    return false
  }
  if (
    capability === 'metadata.detail' &&
    provider.provider_type === 'local_scan'
  ) {
    return true
  }
  return (
    provider.provider_type === 'tmdb' || provider.provider_type === 'metatube'
  )
}

function formatProviderRefs(
  refs: string[] | undefined,
  lookup: ProviderOption[]
) {
  const visibleRefs = (refs ?? []).filter((ref) =>
    lookup.some((item) => item.ref === ref)
  )
  if (!visibleRefs.length) {
    return '未配置'
  }
  return visibleRefs
    .map((ref) => lookup.find((item) => item.ref === ref)?.label || ref)
    .join(' -> ')
}

export function PluginProviderInstanceCard({
  provider,
  pending,
  onEdit,
  onRefreshHealth,
  onDisable,
}: {
  provider: PluginProviderInstance
  pending: boolean
  onEdit: () => void
  onRefreshHealth: () => void
  onDisable: () => void
}) {
  return (
    <Card className='rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm'>
      <CardHeader className='space-y-3 px-5 py-5'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <CardTitle className='text-xl'>{provider.name}</CardTitle>
            <CardDescription className='mt-1'>
              {provider.plugin_name} · {provider.deployment_kind} · #
              {provider.id}
            </CardDescription>
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <Badge variant={provider.enabled ? 'secondary' : 'outline'}>
              {provider.enabled ? '已启用' : '已禁用'}
            </Badge>
            <Badge variant='outline'>
              {formatAvailabilityLabel(provider.availability_status)}
            </Badge>
            <Button type='button' variant='outline' size='sm' onClick={onEdit}>
              <PencilIcon className='size-4' />
              编辑
            </Button>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={pending}
              onClick={onRefreshHealth}
            >
              刷新健康
            </Button>
            {provider.enabled ? (
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={onDisable}
              >
                禁用
              </Button>
            ) : null}
          </div>
        </div>
      </CardHeader>
      <Separator className='bg-border' />
      <CardContent className='space-y-3 px-5 py-5 text-sm text-muted-foreground'>
        <MetadataValue label='端点' value={provider.endpoint} />
        <MetadataValue label='插件 ID' value={provider.plugin_id} />
        <MetadataValue label='版本' value={provider.plugin_version} />
        <div className='flex flex-wrap gap-2'>
          {(provider.capabilities ?? []).map((capability) => (
            <Badge key={capability} variant='outline'>
              {capability}
            </Badge>
          ))}
        </div>
        {provider.failure_reason ? (
          <div className='rounded-[1rem] border border-amber-500/20 bg-amber-500/5 px-4 py-3 text-sm [overflow-wrap:anywhere] text-amber-700 dark:text-amber-300'>
            最近失败原因：{provider.failure_reason}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

export function formatAvailabilityLabel(value: string) {
  switch (value) {
    case 'available':
      return '可用'
    case 'cooldown':
      return '冷却中'
    case 'unavailable':
      return '不可用'
    default:
      return value || '未知'
  }
}

function providerSettings(provider: MetadataProviderInstance) {
  if (provider.provider_type === 'tvdb') {
    return provider.tvdb
  }
  if (provider.provider_type === 'metatube') {
    return provider.metatube
  }
  return provider.tmdb
}
