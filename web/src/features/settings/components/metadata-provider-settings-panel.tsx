import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  GripVerticalIcon,
  InfoIcon,
  LoaderCircleIcon,
  PencilIcon,
  PlusIcon,
} from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import { Checkbox } from '#/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '#/components/ui/dialog'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import { Separator } from '#/components/ui/separator'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '#/components/ui/select'
import { Switch } from '#/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '#/components/ui/tabs'
import { Textarea } from '#/components/ui/textarea'
import type {
  MetadataProfile,
  MetadataProfileInput,
  MetadataProviderInput,
  MetadataProviderInstance,
  MetadataProviderInstanceInput,
  MetadataProviderSettings,
} from '#/lib/mibo-api'
import {
  createAuthedMiboApi,
  metadataProfilesQueryOptions,
  metadataProviderInstancesQueryOptions,
  miboQueryKeys,
} from '#/lib/mibo-query'

type MetadataProviderFormState = {
  apiKey: string
  clearApiKey: boolean
  baseURL: string
  imageBaseURL: string
  language: string
  timeout: string
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
  cooldownUntil: string
  tmdb: MetadataProviderFormState
  tvdb: MetadataProviderFormState
  metatube: MetadataProviderFormState
}

type MetadataProfileDraft = {
  name: string
  description: string
  searchProviderIds: number[]
  detailProviderIds: number[]
  imageProviderIds: number[]
  peopleProviderIds: number[]
  hierarchyProviderIds: number[]
  preferredMetadataLanguage: string
  preferredImageLanguage: string
  fallbackEnabled: boolean
}

const EMPTY_PROVIDER_FORM: MetadataProviderFormState = {
  apiKey: '',
  clearApiKey: false,
  baseURL: '',
  imageBaseURL: '',
  language: '',
  timeout: '',
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
    timeout: '10s',
  },
  tvdb: {
    ...EMPTY_PROVIDER_FORM,
    baseURL: 'https://api4.thetvdb.com/v4',
    language: 'en',
    timeout: '10s',
  },
  metatube: {
    ...EMPTY_PROVIDER_FORM,
    baseURL: 'http://127.0.0.1:8081',
    timeout: '10s',
    fallbackEnabled: true,
  },
}

const EMPTY_PROVIDER_INSTANCE_DRAFT: ProviderInstanceDraft = {
  name: '',
  providerType: 'tmdb',
  enabled: true,
  availabilityStatus: 'available',
  failureReason: '',
  cooldownUntil: '',
  tmdb: DEFAULT_PROVIDER_FORMS.tmdb,
  tvdb: DEFAULT_PROVIDER_FORMS.tvdb,
  metatube: DEFAULT_PROVIDER_FORMS.metatube,
}

const EMPTY_METADATA_PROFILE_DRAFT: MetadataProfileDraft = {
  name: '',
  description: '',
  searchProviderIds: [],
  detailProviderIds: [],
  imageProviderIds: [],
  peopleProviderIds: [],
  hierarchyProviderIds: [],
  preferredMetadataLanguage: '',
  preferredImageLanguage: '',
  fallbackEnabled: true,
}

const HIDDEN_PROVIDER_TYPES = new Set(['local_scan'])

export function MetadataProviderSettingsPanel({
  token,
}: {
  token: string | null
}) {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<'providers' | 'profiles'>(
    'providers',
  )
  const [providerDialogOpen, setProviderDialogOpen] = useState(false)
  const [providerInstanceDraft, setProviderInstanceDraft] =
    useState<ProviderInstanceDraft>(EMPTY_PROVIDER_INSTANCE_DRAFT)
  const [editingProviderInstanceId, setEditingProviderInstanceId] = useState<
    number | null
  >(null)
  const [profileDialogOpen, setProfileDialogOpen] = useState(false)
  const [profileDraft, setProfileDraft] = useState<MetadataProfileDraft>(
    EMPTY_METADATA_PROFILE_DRAFT,
  )
  const [editingProfileId, setEditingProfileId] = useState<number | null>(null)

  const providerInstancesQuery = useQuery({
    ...metadataProviderInstancesQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const profilesQuery = useQuery({
    ...metadataProfilesQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })

  const providerInstances = providerInstancesQuery.data ?? []
  const configurableProviderInstances = providerInstances.filter(
    (item) => !HIDDEN_PROVIDER_TYPES.has(item.provider_type),
  )
  const metadataProfiles = profilesQuery.data ?? []

  const saveProviderInstanceMutation = useMutation({
    mutationFn: async (draft: ProviderInstanceDraft) => {
      if (!token) {
        throw new Error('当前未登录，无法保存 provider instance。')
      }
      const api = createAuthedMiboApi(token)
      const input = buildProviderInstanceInput(draft)
      if (editingProviderInstanceId) {
        return api.updateMetadataProviderInstance(
          editingProviderInstanceId,
          input,
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
          queryKey: miboQueryKeys.metadataProfiles(token),
        }),
      ])
      resetProviderInstanceDialog()
      toast.success('Metadata provider instance 已保存')
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  const saveProfileMutation = useMutation({
    mutationFn: async (draft: MetadataProfileDraft) => {
      if (!token) {
        throw new Error('当前未登录，无法保存 metadata profile。')
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
      toast.success('Metadata profile 已保存')
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  const configurableProviderInstanceOptions = useMemo(
    () =>
      configurableProviderInstances.map((item) => ({
        id: item.id,
        label: `${item.name} (#${item.id})`,
      })),
    [configurableProviderInstances],
  )

  if (!token) {
    return (
      <Alert>
        <InfoIcon className="size-4" />
        <AlertTitle>登录后可管理元数据配置</AlertTitle>
        <AlertDescription className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <span>
            当前页面需要管理员会话来读取和更新 provider instances 与 metadata
            profiles。
          </span>
          <Button asChild variant="outline">
            <Link
              to="/login"
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

  if (providerInstancesQuery.isLoading || profilesQuery.isLoading) {
    return (
      <div className="flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm">
        <LoaderCircleIcon className="size-4 animate-spin" />
        正在加载元数据管理配置
      </div>
    )
  }

  const error = providerInstancesQuery.error || profilesQuery.error
  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{error.message}</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Tabs
          value={activeTab}
          onValueChange={(value) =>
            setActiveTab(value as 'providers' | 'profiles')
          }
        >
          <TabsList>
            <TabsTrigger value="providers">Provider Instances</TabsTrigger>
            <TabsTrigger value="profiles">Metadata Templates</TabsTrigger>
          </TabsList>
        </Tabs>
        <Button
          type="button"
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
          <PlusIcon className="size-4" />
          {activeTab === 'providers'
            ? '新建 Provider Instance'
            : '新建 Metadata Template'}
        </Button>
      </div>

      {activeTab === 'providers' ? (
        <div className="space-y-4">
          <div className="grid gap-4 xl:grid-cols-2">
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
            {configurableProviderInstances.length === 0 ? (
              <EmptyManagementCard text="还没有 metadata provider instance。先创建一个 TMDB 或 TVDB instance。" />
            ) : null}
          </div>
        </div>
      ) : null}

      {activeTab === 'profiles' ? (
        <div className="space-y-4">
          <div className="grid gap-4 xl:grid-cols-2">
            {metadataProfiles.map((profile) => (
              <MetadataProfileCard
                key={profile.id}
                profile={profile}
                providerLookup={configurableProviderInstanceOptions}
                onEdit={() => {
                  setEditingProfileId(profile.id)
                  setProfileDraft(buildMetadataProfileDraft(profile))
                  setProfileDialogOpen(true)
                }}
              />
            ))}
            {metadataProfiles.length === 0 ? (
              <EmptyManagementCard text="还没有 metadata template。创建模板后，媒体库可以复制默认 provider 顺序与语言设置。" />
            ) : null}
          </div>
        </div>
      ) : null}

      <Dialog open={providerDialogOpen} onOpenChange={setProviderDialogOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {editingProviderInstanceId ? '编辑' : '新建'} Provider Instance
            </DialogTitle>
            <DialogDescription>
              每个 instance 都是一个可单独启用、绑定和降级的运行时元数据提供方。
            </DialogDescription>
          </DialogHeader>

          <ProviderInstanceForm
            draft={providerInstanceDraft}
            providerTypeLocked={editingProviderInstanceId !== null}
            onChange={setProviderInstanceDraft}
          />

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={resetProviderInstanceDialog}
              disabled={saveProviderInstanceMutation.isPending}
            >
              取消
            </Button>
            <Button
              type="button"
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

      <Dialog open={profileDialogOpen} onOpenChange={setProfileDialogOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>
              {editingProfileId ? '编辑' : '新建'} Metadata Template
            </DialogTitle>
            <DialogDescription>
              为搜索、详情、图片、人物、层级同步分别指定 provider instance
              顺序。应用到媒体库时会复制为该库自己的可执行策略。
            </DialogDescription>
          </DialogHeader>

          <MetadataProfileForm
            draft={profileDraft}
            providerOptions={configurableProviderInstanceOptions}
            onChange={setProfileDraft}
          />

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={resetProfileDialog}
              disabled={saveProfileMutation.isPending}
            >
              取消
            </Button>
            <Button
              type="button"
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
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="space-y-3 px-5 py-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="text-xl">{provider.name}</CardTitle>
            <CardDescription className="mt-1">
              {provider.provider_type.toUpperCase()} instance #{provider.id}
            </CardDescription>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {provider.system_managed ? (
              <Badge variant="outline">System</Badge>
            ) : null}
            <Badge variant={provider.enabled ? 'secondary' : 'outline'}>
              {provider.enabled ? '已启用' : '已禁用'}
            </Badge>
            <Badge variant="outline">
              {formatAvailabilityLabel(provider.availability_status)}
            </Badge>
            <Badge variant={provider.configured ? 'secondary' : 'outline'}>
              {provider.configured ? '已配置' : '未配置'}
            </Badge>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onEdit}
              disabled={provider.locked}
            >
              <PencilIcon className="size-4" />
              {provider.locked ? '只读' : '编辑'}
            </Button>
          </div>
        </div>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-3 px-5 py-5 text-sm text-muted-foreground">
        {provider.provider_type === 'local_scan' ? (
          <div className="rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3 text-sm text-foreground">
            这是系统内置的本地扫描 provider。它会复用扫描器已经记录的 sidecar
            metadata evidence，且不需要任何额外配置。
          </div>
        ) : (
          <>
            <MetadataValue
              label="Base URL"
              value={providerSettings(provider)?.base_url || '未配置'}
            />
            <MetadataValue
              label="Language"
              value={
                provider.provider_type === 'metatube'
                  ? '不适用'
                  : providerSettings(provider)?.language || '未配置'
              }
            />
            <MetadataValue
              label="Timeout"
              value={providerSettings(provider)?.timeout || '未配置'}
            />
            {provider.provider_type === 'tmdb' ? (
              <MetadataValue
                label="Image Base URL"
                value={provider.tmdb?.image_base_url || '未配置'}
              />
            ) : null}
            {provider.provider_type === 'tvdb' ? (
              <MetadataValue
                label="执行状态"
                value="已配置，暂未接入抓取执行管线"
              />
            ) : null}
            {provider.provider_type === 'metatube' ? (
              <>
                <MetadataValue
                  label="Upstream Provider"
                  value={
                    provider.metatube?.upstream_provider_filter || '未指定'
                  }
                />
                <MetadataValue
                  label="Fallback"
                  value={
                    provider.metatube?.fallback_enabled ? '允许' : '不允许'
                  }
                />
                <MetadataValue
                  label="Token"
                  value={
                    provider.metatube?.api_key_masked ? '已配置' : '未配置'
                  }
                />
              </>
            ) : null}
          </>
        )}
        {provider.failure_reason ? (
          <div className="rounded-[1rem] border border-amber-500/20 bg-amber-500/5 px-4 py-3 text-sm text-amber-700 dark:text-amber-300">
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
  providerLookup: Array<{ id: number; label: string }>
  onEdit: () => void
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="space-y-3 px-5 py-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="text-xl">{profile.name}</CardTitle>
            {!profile.locked ? (
              <CardDescription className="mt-1">
                {profile.description || '未填写描述'}
              </CardDescription>
            ) : null}
            {profile.locked ? (
              <div className="mt-2 text-xs text-muted-foreground">
                这是系统迁移时期保留的只读模板，用于兼容旧的 local-only 行为。
              </div>
            ) : null}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline">Template</Badge>
            {profile.system ? <Badge variant="outline">System</Badge> : null}
            <Badge variant={profile.fallback_enabled ? 'secondary' : 'outline'}>
              {profile.fallback_enabled ? '允许回退' : '禁止回退'}
            </Badge>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onEdit}
              disabled={profile.locked}
            >
              <PencilIcon className="size-4" />
              {profile.locked ? '只读' : '编辑'}
            </Button>
          </div>
        </div>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="grid gap-3 px-5 py-5 text-sm text-muted-foreground">
        <MetadataValue
          label="搜索阶段"
          value={formatProviderIds(profile.search_provider_ids, providerLookup)}
        />
        <MetadataValue
          label="详情阶段"
          value={formatProviderIds(profile.detail_provider_ids, providerLookup)}
        />
        <MetadataValue
          label="图片阶段"
          value={formatProviderIds(profile.image_provider_ids, providerLookup)}
        />
        <MetadataValue
          label="人物阶段"
          value={formatProviderIds(profile.people_provider_ids, providerLookup)}
        />
        <MetadataValue
          label="层级阶段"
          value={formatProviderIds(
            profile.hierarchy_provider_ids,
            providerLookup,
          )}
        />
        <div className="grid gap-3 md:grid-cols-2">
          <MetadataValue
            label="元数据语言"
            value={profile.preferred_metadata_language || '未指定'}
          />
          <MetadataValue
            label="图片语言"
            value={profile.preferred_image_language || '未指定'}
          />
        </div>
      </CardContent>
    </Card>
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
    <div className="space-y-5 py-2">
      <FieldGroup>
        <div className="grid gap-4 md:grid-cols-2">
          <Field>
            <FieldLabel>名称</FieldLabel>
            <Input
              value={draft.name}
              onChange={(event) =>
                onChange({ ...draft, name: event.target.value })
              }
              placeholder="anime-tmdb-primary"
            />
          </Field>
          <Field>
            <FieldLabel>Provider Type</FieldLabel>
            <Select
              value={draft.providerType}
              disabled={providerTypeLocked}
              onValueChange={(providerType) =>
                onChange(applyProviderTypeDefaults(draft, providerType))
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="tmdb">TMDB</SelectItem>
                <SelectItem value="tvdb">TVDB</SelectItem>
                <SelectItem value="metatube">MetaTube</SelectItem>
              </SelectContent>
            </Select>
            <FieldDescription>
              MetaTube 支持电影搜索/详情；TVDB 当前仅保存配置。
            </FieldDescription>
          </Field>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <ToggleField
            label="启用 instance"
            checked={draft.enabled}
            onChange={(checked) => onChange({ ...draft, enabled: checked })}
          />
          <Field>
            <FieldLabel>Availability Status</FieldLabel>
            <Input
              value={draft.availabilityStatus}
              onChange={(event) =>
                onChange({ ...draft, availabilityStatus: event.target.value })
              }
              placeholder="available / cooldown / unavailable"
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
            placeholder="可选，记录认证失败或限流原因"
          />
        </Field>

        <Field>
          <FieldLabel>Cooldown Until</FieldLabel>
          <Input
            value={draft.cooldownUntil}
            onChange={(event) =>
              onChange({ ...draft, cooldownUntil: event.target.value })
            }
            placeholder="2026-04-29T12:00:00Z"
          />
        </Field>

        <Separator />

        {draft.providerType === 'tvdb' ? (
          <MetadataProviderEditFields
            title="TVDB Instance"
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
            title="MetaTube Instance"
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
            title="TMDB Instance"
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
  providerOptions,
  onChange,
}: {
  draft: MetadataProfileDraft
  providerOptions: Array<{ id: number; label: string }>
  onChange: (nextDraft: MetadataProfileDraft) => void
}) {
  return (
    <div className="space-y-5 py-2">
      <FieldGroup>
        <div className="grid gap-4 md:grid-cols-2">
          <Field>
            <FieldLabel>Template 名称</FieldLabel>
            <Input
              value={draft.name}
              onChange={(event) =>
                onChange({ ...draft, name: event.target.value })
              }
              placeholder="movie-default"
            />
          </Field>
        </div>

        <ToggleField
          label="允许 fallback"
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
            placeholder="说明这个模板适合哪些库、使用哪些 provider 顺序"
          />
        </Field>

        <StageField
          label="搜索阶段"
          value={draft.searchProviderIds}
          providerOptions={providerOptions}
          onChange={(value) => onChange({ ...draft, searchProviderIds: value })}
        />
        <StageField
          label="详情阶段"
          value={draft.detailProviderIds}
          providerOptions={providerOptions}
          onChange={(value) => onChange({ ...draft, detailProviderIds: value })}
        />
        <StageField
          label="图片阶段"
          value={draft.imageProviderIds}
          providerOptions={providerOptions}
          onChange={(value) => onChange({ ...draft, imageProviderIds: value })}
        />
        <StageField
          label="人物阶段"
          value={draft.peopleProviderIds}
          providerOptions={providerOptions}
          onChange={(value) => onChange({ ...draft, peopleProviderIds: value })}
        />
        <StageField
          label="层级阶段"
          value={draft.hierarchyProviderIds}
          providerOptions={providerOptions}
          onChange={(value) =>
            onChange({ ...draft, hierarchyProviderIds: value })
          }
        />

        <div className="grid gap-4 md:grid-cols-2">
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
              placeholder="zh-CN"
            />
          </Field>
          <Field>
            <FieldLabel>默认图片语言</FieldLabel>
            <Input
              value={draft.preferredImageLanguage}
              onChange={(event) =>
                onChange({
                  ...draft,
                  preferredImageLanguage: event.target.value,
                })
              }
              placeholder="zh-CN"
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
  value: number[]
  providerOptions: Array<{ id: number; label: string }>
  onChange: (value: number[]) => void
}) {
  const [draggingId, setDraggingId] = useState<number | null>(null)
  const providerOptionIds = new Set(providerOptions.map((item) => item.id))
  const visibleValue = value.filter((id) => providerOptionIds.has(id))

  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <div className="grid gap-3 rounded-[1rem] border border-border/60 bg-background/60 p-3">
        <div className="text-xs text-muted-foreground">
          勾选 provider 进入当前阶段，拖动已选项可以调整执行顺序。
        </div>

        <div className="grid gap-2">
          {visibleValue.length > 0 ? (
            visibleValue.map((id) => {
              const provider = providerOptions.find((item) => item.id === id)
              return (
                <div
                  key={id}
                  draggable
                  onDragStart={() => setDraggingId(id)}
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={() => {
                    if (draggingId === null || draggingId === id) {
                      return
                    }
                    const next = [...visibleValue]
                    const fromIndex = next.indexOf(draggingId)
                    const toIndex = next.indexOf(id)
                    if (fromIndex < 0 || toIndex < 0) {
                      return
                    }
                    next.splice(fromIndex, 1)
                    next.splice(toIndex, 0, draggingId)
                    onChange(next)
                    setDraggingId(null)
                  }}
                  onDragEnd={() => setDraggingId(null)}
                  className="flex items-center justify-between rounded-lg border border-border/60 bg-card px-3 py-2 text-sm"
                >
                  <div className="flex items-center gap-2">
                    <GripVerticalIcon className="size-4 text-muted-foreground" />
                    <span>{provider?.label || `#${id}`}</span>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                      onChange(visibleValue.filter((item) => item !== id))
                    }
                  >
                    移除
                  </Button>
                </div>
              )
            })
          ) : (
            <div className="rounded-lg border border-dashed border-border/60 px-3 py-4 text-sm text-muted-foreground">
              当前阶段还没有选中的 provider。
            </div>
          )}
        </div>

        <div className="grid gap-2 md:grid-cols-2">
          {providerOptions.map((item) => {
            const checked = visibleValue.includes(item.id)
            return (
              <label
                key={item.id}
                className="flex items-center gap-3 rounded-lg border border-border/60 px-3 py-2 text-sm"
              >
                <Checkbox
                  checked={checked}
                  onCheckedChange={(nextChecked) => {
                    if (nextChecked === true) {
                      if (!checked) {
                        onChange([...visibleValue, item.id])
                      }
                      return
                    }
                    onChange(
                      visibleValue.filter((selected) => selected !== item.id),
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
          ? '每个阶段都可以选择不同的 provider，并用拖拽调整优先级。'
          : '还没有可用实例。请先创建 provider instance。'}
      </FieldDescription>
    </Field>
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
    <div className="flex items-center justify-between rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3 text-sm">
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
    value: MetadataProviderFormState[FieldKey],
  ) => void
  includeImageBaseURL?: boolean
  includeLanguage?: boolean
  includeMetaTubeFields?: boolean
  showClearKey?: boolean
}) {
  return (
    <div className="space-y-5">
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor={`${title}-api-key`}>API Key / Token</FieldLabel>
          <Input
            id={`${title}-api-key`}
            type="password"
            value={draft.apiKey}
            disabled={draft.clearApiKey}
            placeholder={
              settings?.api_key_masked
                ? '已配置，留空则保持当前 key'
                : '输入新的 API Key 或 Token'
            }
            className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
            onChange={(event) => onChange('apiKey', event.target.value)}
          />
        </Field>

        {showClearKey ? (
          <label className="flex items-start gap-3 rounded-[1.1rem] border border-border/60 bg-background/60 px-4 py-3">
            <Checkbox
              checked={draft.clearApiKey}
              onCheckedChange={(checked) =>
                onChange('clearApiKey', checked === true)
              }
              className="mt-1"
            />
            <div className="space-y-1">
              <div className="text-sm font-medium text-foreground">
                清除已保存 key
              </div>
              <div className="text-sm text-muted-foreground">
                保存时会删除数据库中的密钥记录。若当前值来自环境变量，运行中的
                env 配置仍会继续生效。
              </div>
            </div>
          </label>
        ) : null}

        <div className="grid gap-4 md:grid-cols-2">
          <Field>
            <FieldLabel htmlFor={`${title}-base-url`}>Base URL</FieldLabel>
            <Input
              id={`${title}-base-url`}
              value={draft.baseURL}
              className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
              onChange={(event) => onChange('baseURL', event.target.value)}
            />
          </Field>

          {includeLanguage ? (
            <Field>
              <FieldLabel htmlFor={`${title}-language`}>Language</FieldLabel>
              <Input
                id={`${title}-language`}
                value={draft.language}
                className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
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
                Default Upstream Provider
              </FieldLabel>
              <Input
                id={`${title}-upstream-provider`}
                value={draft.upstreamProviderFilter}
                placeholder="例如 fanza，可留空"
                className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
                onChange={(event) =>
                  onChange('upstreamProviderFilter', event.target.value)
                }
              />
              <FieldDescription>
                手动指定 MetaTube 上游 provider filter；留空则由 MetaTube server
                决定。
              </FieldDescription>
            </Field>
            <ToggleField
              label="允许 MetaTube fallback"
              checked={draft.fallbackEnabled}
              onChange={(checked) => onChange('fallbackEnabled', checked)}
            />
          </>
        ) : null}

        {includeImageBaseURL ? (
          <Field>
            <FieldLabel htmlFor={`${title}-image-base-url`}>
              Image Base URL
            </FieldLabel>
            <Input
              id={`${title}-image-base-url`}
              value={draft.imageBaseURL}
              className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
              onChange={(event) => onChange('imageBaseURL', event.target.value)}
            />
          </Field>
        ) : null}

        <Field>
          <FieldLabel htmlFor={`${title}-timeout`}>Timeout</FieldLabel>
          <Input
            id={`${title}-timeout`}
            value={draft.timeout}
            className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
            onChange={(event) => onChange('timeout', event.target.value)}
          />
          <FieldDescription>
            使用 Go duration 格式，例如 `10s`、`30s`、`1m`。
          </FieldDescription>
        </Field>
      </FieldGroup>
    </div>
  )
}

function MetadataValue({
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
      <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </div>
      <div className="mt-1 break-all text-sm text-foreground">{value}</div>
    </div>
  )
}

function EmptyManagementCard({ text }: { text: string }) {
  return (
    <Card className="rounded-[1.5rem] border-dashed border-border/60 bg-card/50 py-0 shadow-none">
      <CardContent className="px-5 py-10 text-sm text-muted-foreground">
        {text}
      </CardContent>
    </Card>
  )
}

function buildProviderInstanceDraft(
  provider: MetadataProviderInstance,
): ProviderInstanceDraft {
  return {
    name: provider.name,
    providerType: provider.provider_type,
    enabled: provider.enabled,
    availabilityStatus: provider.availability_status,
    failureReason: provider.failure_reason || '',
    cooldownUntil: provider.cooldown_until || '',
    tmdb: {
      ...DEFAULT_PROVIDER_FORMS.tmdb,
      baseURL: provider.tmdb?.base_url || '',
      imageBaseURL: provider.tmdb?.image_base_url || '',
      language: provider.tmdb?.language || '',
      timeout: provider.tmdb?.timeout || '',
    },
    tvdb: {
      ...DEFAULT_PROVIDER_FORMS.tvdb,
      baseURL: provider.tvdb?.base_url || '',
      language: provider.tvdb?.language || '',
      timeout: provider.tvdb?.timeout || '',
    },
    metatube: {
      ...DEFAULT_PROVIDER_FORMS.metatube,
      baseURL: provider.metatube?.base_url || '',
      timeout: provider.metatube?.timeout || '',
      upstreamProviderFilter: provider.metatube?.upstream_provider_filter || '',
      fallbackEnabled:
        provider.metatube?.fallback_enabled ??
        DEFAULT_PROVIDER_FORMS.metatube.fallbackEnabled,
    },
  }
}

function applyProviderTypeDefaults(
  draft: ProviderInstanceDraft,
  providerType: string,
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
      upstreamProviderFilter:
        draft[providerType].upstreamProviderFilter ||
        DEFAULT_PROVIDER_FORMS[providerType].upstreamProviderFilter,
    },
  }
}

function buildProviderInstanceInput(
  draft: ProviderInstanceDraft,
): MetadataProviderInstanceInput {
  const input: MetadataProviderInstanceInput = {
    name: draft.name.trim(),
    provider_type: draft.providerType,
    enabled: draft.enabled,
    availability_status: draft.availabilityStatus.trim() || undefined,
    failure_reason: draft.failureReason.trim() || undefined,
    cooldown_until: draft.cooldownUntil.trim() || undefined,
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
  draft: MetadataProviderFormState,
): MetadataProviderInput {
  return {
    api_key: draft.apiKey || undefined,
    clear_api_key: draft.clearApiKey || undefined,
    base_url: draft.baseURL || undefined,
    image_base_url: draft.imageBaseURL || undefined,
    language: draft.language || undefined,
    timeout: draft.timeout || undefined,
    upstream_provider_filter: draft.upstreamProviderFilter || undefined,
    fallback_enabled: draft.fallbackEnabled,
  }
}

function isEditableProviderType(value: string): value is EditableProviderType {
  return value === 'tmdb' || value === 'tvdb' || value === 'metatube'
}

function buildMetadataProfileDraft(
  profile: MetadataProfile,
): MetadataProfileDraft {
  return {
    name: profile.name,
    description: profile.description || '',
    searchProviderIds: profile.search_provider_ids,
    detailProviderIds: profile.detail_provider_ids,
    imageProviderIds: profile.image_provider_ids,
    peopleProviderIds: profile.people_provider_ids,
    hierarchyProviderIds: profile.hierarchy_provider_ids,
    preferredMetadataLanguage: profile.preferred_metadata_language || '',
    preferredImageLanguage: profile.preferred_image_language || '',
    fallbackEnabled: profile.fallback_enabled,
  }
}

function buildMetadataProfileInput(
  draft: MetadataProfileDraft,
): MetadataProfileInput {
  return {
    name: draft.name.trim(),
    description: draft.description.trim() || undefined,
    search_provider_ids: draft.searchProviderIds,
    detail_provider_ids: draft.detailProviderIds,
    image_provider_ids: draft.imageProviderIds,
    people_provider_ids: draft.peopleProviderIds,
    hierarchy_provider_ids: draft.hierarchyProviderIds,
    preferred_metadata_language:
      draft.preferredMetadataLanguage.trim() || undefined,
    preferred_image_language: draft.preferredImageLanguage.trim() || undefined,
    fallback_enabled: draft.fallbackEnabled,
  }
}

function formatProviderIds(
  ids: number[],
  lookup: Array<{ id: number; label: string }>,
) {
  const visibleIds = ids.filter((id) => lookup.some((item) => item.id === id))
  if (!visibleIds.length) {
    return '未配置'
  }
  return visibleIds
    .map((id) => lookup.find((item) => item.id === id)?.label || `#${id}`)
    .join(' -> ')
}

function formatAvailabilityLabel(value: string) {
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
