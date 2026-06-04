import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  BoxesIcon,
  CirclePlayIcon,
  LoaderCircleIcon,
  PlusIcon,
  PowerIcon,
  RefreshCwIcon,
  SquareIcon,
  Trash2Icon,
} from 'lucide-react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import type {
  InternalPlugin,
  LocalPluginInstallInput,
  PluginProviderInstance,
  PluginUsageReference,
  RemotePluginProviderInput,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  internalPluginsQueryOptions,
  localPluginInstallationsQueryOptions,
  miboQueryKeys,
  pluginCatalogOverviewQueryOptions,
  pluginProviderDetailQueryOptions,
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  EMPTY_PLUGIN_PROVIDER_DRAFT,
  EmptyManagementCard,
  MetadataValue,
  PluginProviderForm,
  buildPluginProviderDraft,
  formatAvailabilityLabel,
  type PluginProviderDraft,
} from '@/features/settings/components/metadata-provider-settings-panel'
import { buildPluginConfigurationDefaults } from '@/features/settings/components/plugin-configuration-form'

type PluginCenterTab =
  | 'overview'
  | 'system'
  | 'instances'
  | 'detail'
  | 'local'
  | 'catalog'

const EMPTY_LOCAL_DRAFT: LocalPluginInstallInput = {
  plugin_id: '',
  name: '',
  version: '',
  source_kind: 'filesystem',
  source: '',
  install_path: '',
  enabled: true,
}

export function PluginManagementCenter({ token }: { token: string | null }) {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<PluginCenterTab>('overview')
  const [selectedProviderId, setSelectedProviderId] = useState<number | null>(
    null
  )
  const [pluginDialogOpen, setPluginDialogOpen] = useState(false)
  const [localDialogOpen, setLocalDialogOpen] = useState(false)
  const [editingPluginProviderId, setEditingPluginProviderId] = useState<
    number | null
  >(null)
  const [pluginProviderDraft, setPluginProviderDraft] =
    useState<PluginProviderDraft>(EMPTY_PLUGIN_PROVIDER_DRAFT)
  const [localDraft, setLocalDraft] =
    useState<LocalPluginInstallInput>(EMPTY_LOCAL_DRAFT)

  const pluginProvidersQuery = useQuery({
    ...pluginProviderInstancesQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const internalPluginsQuery = useQuery({
    ...internalPluginsQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const localInstallationsQuery = useQuery({
    ...localPluginInstallationsQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const catalogQuery = useQuery({
    ...pluginCatalogOverviewQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const selectedDetailQuery = useQuery({
    ...pluginProviderDetailQueryOptions(
      token ?? 'guest',
      selectedProviderId ?? 0
    ),
    enabled: !!token && !!selectedProviderId,
  })

  const providers = useMemo(
    () => pluginProvidersQuery.data ?? [],
    [pluginProvidersQuery.data]
  )
  const selectedProvider =
    providers.find((provider) => provider.id === selectedProviderId) ??
    providers[0] ??
    null
  const internalPlugins = useMemo(
    () => internalPluginsQuery.data ?? [],
    [internalPluginsQuery.data]
  )

  const overview = useMemo(() => {
    const unhealthy =
      providers.filter(
        (provider) => provider.availability_status !== 'available'
      ).length +
      internalPlugins.filter(
        (plugin) => plugin.availability_status !== 'available'
      ).length
    const localCount = providers.filter(
      (provider) => provider.deployment_kind === 'local_companion'
    ).length
    const capabilities = new Map<string, number>()
    providers.forEach((provider) =>
      (provider.capabilities ?? []).forEach((capability) =>
        capabilities.set(capability, (capabilities.get(capability) ?? 0) + 1)
      )
    )
    internalPlugins.forEach((plugin) =>
      (plugin.capabilities ?? []).forEach((capability) =>
        capabilities.set(capability, (capabilities.get(capability) ?? 0) + 1)
      )
    )
    return { unhealthy, localCount, capabilities: [...capabilities.entries()] }
  }, [internalPlugins, providers])

  const previewMutation = useMutation({
    mutationFn: async (endpoint: string) => {
      if (!token) throw new Error('当前未登录，无法获取插件清单。')
      return createAuthedMiboApi(token).previewRemotePluginManifest(endpoint)
    },
    onSuccess: (manifest) => {
      setPluginProviderDraft((current) => ({
        ...current,
        name: current.name.trim() || manifest.name,
        manifest,
        configuration: buildPluginConfigurationDefaults(
          manifest.configuration_schema,
          current.configuration
        ),
      }))
      toast.success('插件清单已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const savePluginMutation = useMutation({
    mutationFn: async (draft: PluginProviderDraft) => {
      if (!token) throw new Error('当前未登录，无法保存插件实例。')
      const input: RemotePluginProviderInput = {
        name: draft.name.trim() || undefined,
        endpoint: draft.endpoint.trim(),
        configuration: draft.configuration,
        enabled: draft.enabled,
      }
      const api = createAuthedMiboApi(token)
      if (editingPluginProviderId) {
        return api.updateRemotePluginProviderInstance(
          editingPluginProviderId,
          input
        )
      }
      return api.createRemotePluginProviderInstance(input)
    },
    onSuccess: async (instance) => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.pluginProviderInstances(token),
      })
      setSelectedProviderId(instance.id)
      setActiveTab('detail')
      resetPluginDialog()
      toast.success('插件实例已保存')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const refreshHealthMutation = useMutation({
    mutationFn: async (providerId: number) => {
      if (!token) throw new Error('当前未登录，无法刷新插件状态。')
      return createAuthedMiboApi(token).refreshPluginProviderHealth(providerId)
    },
    onSuccess: async (instance) => {
      if (!token) return
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.pluginProviderInstances(token),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.pluginProviderDetail(token, instance.id),
        }),
      ])
      toast.success('插件健康状态已刷新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const disableMutation = useMutation({
    mutationFn: async (provider: PluginProviderInstance) => {
      if (!token) throw new Error('当前未登录，无法禁用插件实例。')
      const detail = await createAuthedMiboApi(token).getPluginProviderDetail(
        provider.id
      )
      if (
        detail.usage.active_reference_count > 0 &&
        !window.confirm(
          `该插件仍有 ${detail.usage.active_reference_count} 个引用，禁用后可能影响元数据模板或媒体源。继续禁用？`
        )
      ) {
        return provider
      }
      return createAuthedMiboApi(token).disablePluginProviderInstance(
        provider.id
      )
    },
    onSuccess: async (instance) => {
      if (!token) return
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.pluginProviderInstances(token),
        }),
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.pluginProviderDetail(token, instance.id),
        }),
      ])
      toast.success('插件实例已禁用')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const saveLocalMutation = useMutation({
    mutationFn: async (input: LocalPluginInstallInput) => {
      if (!token) throw new Error('当前未登录，无法注册本地插件。')
      return createAuthedMiboApi(token).installLocalPlugin(input)
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.localPluginInstallations(token),
      })
      setLocalDialogOpen(false)
      setLocalDraft(EMPTY_LOCAL_DRAFT)
      toast.success('本地插件来源已注册')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const updateInternalPluginMutation = useMutation({
    mutationFn: async ({
      plugin,
      enabled,
    }: {
      plugin: InternalPlugin
      enabled: boolean
    }) => {
      if (!token) throw new Error('当前未登录，无法更新内部插件。')
      if (
        !enabled &&
        (plugin.usage?.length ?? 0) > 0 &&
        !window.confirm(
          `该内部插件仍有 ${plugin.usage?.length ?? 0} 个引用，禁用后可能影响元数据匹配或媒体源访问。继续禁用？`
        )
      ) {
        return plugin
      }
      return createAuthedMiboApi(token).updateInternalPlugin(plugin.id, {
        enabled,
      })
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.internalPlugins(token),
      })
      toast.success('内部插件状态已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const localActionMutation = useMutation({
    mutationFn: async ({
      id,
      action,
    }: {
      id: number
      action: 'start' | 'stop' | 'restart' | 'uninstall'
    }) => {
      if (!token) throw new Error('当前未登录，无法操作本地插件。')
      const api = createAuthedMiboApi(token)
      if (action === 'start') return api.startLocalPluginInstallation(id)
      if (action === 'stop') return api.stopLocalPluginInstallation(id)
      if (action === 'restart') return api.restartLocalPluginInstallation(id)
      return api.uninstallLocalPluginInstallation(id)
    },
    onSuccess: async () => {
      if (!token) return
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.localPluginInstallations(token),
      })
      toast.success('本地插件状态已更新')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  if (!token) {
    return (
      <Alert>
        <AlertTitle>登录后可管理插件</AlertTitle>
        <AlertDescription>
          插件中心需要管理员会话来读取和操作插件实例、生命周期与目录信息。
        </AlertDescription>
      </Alert>
    )
  }

  if (
    pluginProvidersQuery.isLoading ||
    internalPluginsQuery.isLoading ||
    localInstallationsQuery.isLoading ||
    catalogQuery.isLoading
  ) {
    return (
      <div className='flex items-center gap-3 rounded-lg border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm'>
        <LoaderCircleIcon className='size-4 animate-spin' />
        正在加载插件中心
      </div>
    )
  }

  const error =
    pluginProvidersQuery.error ||
    internalPluginsQuery.error ||
    localInstallationsQuery.error ||
    catalogQuery.error
  if (error) {
    return (
      <Alert variant='destructive'>
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{error.message}</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className='flex min-h-0 flex-1 flex-col gap-4 pb-20'>
      <div className='flex shrink-0 justify-center'>
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as PluginCenterTab)}
        >
          <TabsList className='flex flex-wrap'>
            <TabsTrigger value='overview'>概览</TabsTrigger>
            <TabsTrigger value='system'>系统内部</TabsTrigger>
            <TabsTrigger value='instances'>实例</TabsTrigger>
            <TabsTrigger value='detail'>详情</TabsTrigger>
            <TabsTrigger value='local'>本地</TabsTrigger>
            <TabsTrigger value='catalog'>目录</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <div className='min-h-0 flex-1 overflow-y-auto'>
        {activeTab === 'overview' ? (
          <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
            <MetricCard label='已注册实例' value={String(providers.length)} />
            <MetricCard
              label='内部插件'
              value={String(internalPlugins.length)}
            />
            <MetricCard
              label='本地伴随实例'
              value={String(overview.localCount)}
            />
            <MetricCard label='异常或冷却' value={String(overview.unhealthy)} />
            <MetricCard
              label='能力类型'
              value={String(overview.capabilities.length)}
            />
            <Card className='border-border/60 py-0 shadow-sm md:col-span-2 xl:col-span-4'>
              <CardHeader className='py-5'>
                <CardTitle className='text-base'>能力分布</CardTitle>
                <CardDescription>按插件声明能力聚合。</CardDescription>
              </CardHeader>
              <CardContent className='flex flex-wrap gap-2 pb-5'>
                {overview.capabilities.length ? (
                  overview.capabilities.map(([capability, count]) => (
                    <Badge key={capability} variant='outline'>
                      {capability} · {count}
                    </Badge>
                  ))
                ) : (
                  <span className='text-sm text-muted-foreground'>
                    还没有注册插件能力。
                  </span>
                )}
              </CardContent>
            </Card>
          </div>
        ) : null}

        {activeTab === 'system' ? (
          <div className='grid gap-4 xl:grid-cols-2'>
            {internalPlugins.map((plugin) => (
              <InternalPluginCard
                key={plugin.id}
                plugin={plugin}
                pending={updateInternalPluginMutation.isPending}
                onToggle={(enabled) =>
                  updateInternalPluginMutation.mutate({ plugin, enabled })
                }
              />
            ))}
            {internalPlugins.length === 0 ? (
              <EmptyManagementCard text='还没有可展示的内部插件能力。' />
            ) : null}
          </div>
        ) : null}

        {activeTab === 'instances' ? (
          <div className='grid gap-4 xl:grid-cols-2'>
            {providers.map((provider) => (
              <RemotePluginCard
                key={provider.id}
                provider={provider}
                pending={refreshHealthMutation.isPending}
                onEdit={() => {
                  setEditingPluginProviderId(provider.id)
                  setPluginProviderDraft(buildPluginProviderDraft(provider))
                  setPluginDialogOpen(true)
                }}
                onRefreshHealth={() =>
                  refreshHealthMutation.mutate(provider.id)
                }
                onDisable={() => disableMutation.mutate(provider)}
                onViewDetail={() => {
                  setSelectedProviderId(provider.id)
                  setActiveTab('detail')
                }}
              />
            ))}
            {providers.length === 0 ? (
              <EmptyManagementCard text='还没有注册插件实例。先注册远程插件，或在本地区域登记伴随插件来源。' />
            ) : null}
          </div>
        ) : null}

        {activeTab === 'detail' ? (
          <PluginDetailPanel
            provider={selectedProvider}
            detail={selectedDetailQuery.data}
            loading={selectedDetailQuery.isLoading}
            onSelect={(providerId) => setSelectedProviderId(providerId)}
            providers={providers}
          />
        ) : null}

        {activeTab === 'local' ? (
          <div className='space-y-4'>
            <div className='grid gap-4 xl:grid-cols-2'>
              {(localInstallationsQuery.data ?? []).map((item) => (
                <Card
                  key={item.id}
                  className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'
                >
                  <CardHeader className='gap-3 px-4 py-3'>
                    <CardTitle className='truncate text-base'>
                      {item.name}
                    </CardTitle>
                    <CardDescription className='truncate'>
                      {item.plugin_id} · {item.version} · {item.source_kind}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='space-y-3 px-4 pb-4'>
                    <div className='grid gap-2'>
                      <PluginSummaryLine label='来源' value={item.source} />
                      <PluginSummaryLine
                        label='端点'
                        value={item.resolved_endpoint || '尚未解析'}
                      />
                    </div>
                    <div className='flex flex-wrap gap-2'>
                      <PluginMetaPill label='安装' value={item.install_state} />
                      <PluginMetaPill label='进程' value={item.process_state} />
                    </div>
                    {item.failure_reason ? (
                      <Alert variant='destructive'>
                        <AlertTitle>最近失败原因</AlertTitle>
                        <AlertDescription>
                          {item.failure_reason}
                        </AlertDescription>
                      </Alert>
                    ) : null}
                    <div className='flex flex-wrap gap-2'>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          localActionMutation.mutate({
                            id: item.id,
                            action: 'start',
                          })
                        }
                      >
                        <CirclePlayIcon className='size-4' />
                        启动
                      </Button>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          localActionMutation.mutate({
                            id: item.id,
                            action: 'stop',
                          })
                        }
                      >
                        <SquareIcon className='size-4' />
                        停止
                      </Button>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          localActionMutation.mutate({
                            id: item.id,
                            action: 'restart',
                          })
                        }
                      >
                        <RefreshCwIcon className='size-4' />
                        重启
                      </Button>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          localActionMutation.mutate({
                            id: item.id,
                            action: 'uninstall',
                          })
                        }
                      >
                        <Trash2Icon className='size-4' />
                        卸载
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
              {(localInstallationsQuery.data ?? []).length === 0 ? (
                <EmptyManagementCard text='尚未注册本地伴随插件。当前支持先登记本地路径或端点，启动后再解析为现有插件协议端点。' />
              ) : null}
            </div>
          </div>
        ) : null}

        {activeTab === 'catalog' ? (
          <CatalogPanel
            sources={catalogQuery.data?.sources ?? []}
            entries={catalogQuery.data?.entries ?? []}
          />
        ) : null}
      </div>

      {createPortal(
        <div className='fixed right-6 bottom-6 z-[100] flex justify-end'>
          <Button
            type='button'
            onClick={() => {
              if (activeTab === 'local') {
                setLocalDialogOpen(true)
                return
              }
              setEditingPluginProviderId(null)
              setPluginProviderDraft(EMPTY_PLUGIN_PROVIDER_DRAFT)
              setPluginDialogOpen(true)
            }}
          >
            <PlusIcon className='size-4' />
            {activeTab === 'local' ? '注册本地来源' : '注册远程插件'}
          </Button>
        </div>,
        document.body
      )}

      <Dialog open={pluginDialogOpen} onOpenChange={setPluginDialogOpen}>
        <DialogContent className='flex max-h-[90vh] flex-col overflow-hidden sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>
              {editingPluginProviderId ? '编辑' : '注册'}远程插件实例
            </DialogTitle>
            <DialogDescription>
              输入插件端点后获取清单，再根据 manifest 声明的 schema 配置实例。
            </DialogDescription>
          </DialogHeader>
          <div className='-mx-4 min-h-0 flex-1 overflow-y-auto px-4'>
            <PluginProviderForm
              draft={pluginProviderDraft}
              pendingPreview={previewMutation.isPending}
              onPreview={() =>
                previewMutation.mutate(pluginProviderDraft.endpoint)
              }
              onChange={setPluginProviderDraft}
            />
          </div>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={resetPluginDialog}
              disabled={savePluginMutation.isPending}
            >
              取消
            </Button>
            <Button
              type='button'
              onClick={() => savePluginMutation.mutate(pluginProviderDraft)}
              disabled={
                savePluginMutation.isPending ||
                !pluginProviderDraft.endpoint.trim()
              }
            >
              {savePluginMutation.isPending ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={localDialogOpen} onOpenChange={setLocalDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>注册本地伴随插件来源</DialogTitle>
            <DialogDescription>
              先登记本地路径或本地端点；启动时会解析端点并接入现有插件协议。
            </DialogDescription>
          </DialogHeader>
          <FieldGroup>
            <Field>
              <FieldLabel>插件 ID</FieldLabel>
              <Input
                value={localDraft.plugin_id}
                onChange={(event) =>
                  setLocalDraft({
                    ...localDraft,
                    plugin_id: event.target.value,
                  })
                }
              />
            </Field>
            <Field>
              <FieldLabel>名称</FieldLabel>
              <Input
                value={localDraft.name}
                onChange={(event) =>
                  setLocalDraft({ ...localDraft, name: event.target.value })
                }
              />
            </Field>
            <Field>
              <FieldLabel>来源</FieldLabel>
              <Input
                value={localDraft.source}
                placeholder='/plugins/demo 或 http://127.0.0.1:9001'
                onChange={(event) =>
                  setLocalDraft({ ...localDraft, source: event.target.value })
                }
              />
            </Field>
          </FieldGroup>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => setLocalDialogOpen(false)}
            >
              取消
            </Button>
            <Button
              type='button'
              onClick={() => saveLocalMutation.mutate(localDraft)}
              disabled={
                saveLocalMutation.isPending ||
                !localDraft.plugin_id.trim() ||
                !localDraft.source.trim()
              }
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )

  function resetPluginDialog() {
    setPluginDialogOpen(false)
    setEditingPluginProviderId(null)
    setPluginProviderDraft(EMPTY_PLUGIN_PROVIDER_DRAFT)
  }
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='px-4 py-3'>
        <CardDescription>{label}</CardDescription>
        <CardTitle className='text-2xl'>{value}</CardTitle>
      </CardHeader>
    </Card>
  )
}

function RemotePluginCard({
  provider,
  pending,
  onEdit,
  onRefreshHealth,
  onDisable,
  onViewDetail,
}: {
  provider: PluginProviderInstance
  pending: boolean
  onEdit: () => void
  onRefreshHealth: () => void
  onDisable: () => void
  onViewDetail: () => void
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
              {provider.plugin_name} · {provider.plugin_id} · #{provider.id}
            </CardDescription>
          </div>
          <div className='flex shrink-0 flex-wrap items-center gap-2'>
            <Badge variant={provider.enabled ? 'secondary' : 'outline'}>
              {provider.enabled ? '已启用' : '已禁用'}
            </Badge>
            <Button type='button' variant='outline' size='sm' onClick={onEdit}>
              编辑
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4 text-sm text-muted-foreground'>
        <PluginSummaryLine label='端点' value={provider.endpoint} />
        <div className='flex flex-wrap gap-2'>
          <PluginMetaPill
            label='可用'
            value={formatAvailabilityLabel(provider.availability_status)}
          />
          <PluginMetaPill label='部署' value={provider.deployment_kind} />
          <PluginMetaPill label='版本' value={provider.plugin_version} />
          <PluginMetaPill
            label='检查'
            value={provider.last_checked_at || '尚未检查'}
          />
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
        <div className='flex flex-wrap justify-end gap-2 pt-1'>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            onClick={onViewDetail}
          >
            查看诊断
          </Button>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={pending}
            onClick={onRefreshHealth}
          >
            <RefreshCwIcon className='size-4' />
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
      </CardContent>
    </Card>
  )
}

function PluginSummaryLine({ label, value }: { label: string; value: string }) {
  return (
    <div className='rounded-lg border border-border/60 bg-muted/15 px-3 py-2'>
      <div className='text-[11px] text-muted-foreground'>{label}</div>
      <div className='mt-1 truncate font-mono text-xs text-foreground'>
        {value}
      </div>
    </div>
  )
}

function PluginMetaPill({ label, value }: { label: string; value: string }) {
  return (
    <span className='inline-flex h-6 min-w-0 items-center rounded-md border border-border/60 bg-muted/20 px-2 text-xs text-muted-foreground'>
      <span className='text-muted-foreground/80'>{label}</span>
      <span className='mx-1 text-border'>/</span>
      <span className='truncate text-foreground'>{value}</span>
    </span>
  )
}

function InternalPluginCard({
  plugin,
  pending,
  onToggle,
}: {
  plugin: InternalPlugin
  pending: boolean
  onToggle: (enabled: boolean) => void
}) {
  const usage = plugin.usage ?? []
  return (
    <Card className='rounded-xl border-border/60 bg-background/60 py-0 shadow-none'>
      <CardHeader className='gap-3 px-4 py-3'>
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <CardTitle className='truncate text-base'>{plugin.name}</CardTitle>
            <CardDescription className='mt-1 truncate'>
              {plugin.kind} · {plugin.provider_key}
              {plugin.provider_ref ? ` · ${plugin.provider_ref}` : ''}
            </CardDescription>
          </div>
          <div className='flex shrink-0 flex-wrap items-center gap-2'>
            <Badge variant={plugin.enabled ? 'secondary' : 'outline'}>
              {plugin.enabled ? '已启用' : '已禁用'}
            </Badge>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={pending}
              onClick={() => onToggle(!plugin.enabled)}
            >
              <PowerIcon className='size-4' />
              {plugin.enabled ? '禁用' : '启用'}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3 px-4 pb-4'>
        {plugin.description ? (
          <p className='line-clamp-2 text-sm leading-6 text-muted-foreground'>
            {plugin.description}
          </p>
        ) : null}
        <div className='flex flex-wrap gap-2'>
          <PluginMetaPill
            label='配置'
            value={plugin.configured ? '已配置' : '未配置'}
          />
          <PluginMetaPill
            label='可用'
            value={formatAvailabilityLabel(plugin.availability_status)}
          />
          <PluginMetaPill label='引用' value={String(usage.length)} />
        </div>
        <div className='flex flex-wrap gap-2'>
          {(plugin.capabilities ?? []).map((capability) => (
            <Badge key={capability} variant='outline'>
              {capability}
            </Badge>
          ))}
        </div>
        {usage.length ? (
          <div className='space-y-2'>
            <div className='flex flex-wrap gap-2'>
              {usage.map((item) => (
                <Badge
                  key={`${item.kind}-${item.id}-${item.stage ?? ''}`}
                  variant='outline'
                >
                  {item.name}
                  {item.stage ? ` · ${item.stage}` : ''}
                </Badge>
              ))}
            </div>
          </div>
        ) : (
          <div className='text-sm text-muted-foreground'>暂无活动引用</div>
        )}
      </CardContent>
    </Card>
  )
}

function PluginDetailPanel({
  provider,
  providers,
  detail,
  loading,
  onSelect,
}: {
  provider: PluginProviderInstance | null
  providers: PluginProviderInstance[]
  detail?: {
    usage: {
      metadata_profiles: PluginUsageReference[]
      library_metadata_strategies: PluginUsageReference[]
      media_sources: PluginUsageReference[]
      active_reference_count: number
    }
  }
  loading: boolean
  onSelect: (providerId: number) => void
}) {
  if (!provider) {
    return (
      <EmptyManagementCard text='选择或注册插件实例后，可查看 manifest、运行状态、能力和引用关系。' />
    )
  }

  const usage = detail?.usage

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap gap-2'>
        {providers.map((item) => (
          <Button
            key={item.id}
            type='button'
            size='sm'
            variant={item.id === provider.id ? 'default' : 'outline'}
            onClick={() => onSelect(item.id)}
          >
            {item.name}
          </Button>
        ))}
      </div>
      <Card className='border-border/60 py-0 shadow-sm'>
        <CardHeader className='py-5'>
          <CardTitle>{provider.name}</CardTitle>
          <CardDescription>
            {provider.plugin_name} · {provider.plugin_id}
          </CardDescription>
        </CardHeader>
        <Separator />
        <CardContent className='space-y-5 py-5'>
          <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
            <MetadataValue label='端点' value={provider.endpoint} />
            <MetadataValue label='部署类型' value={provider.deployment_kind} />
            <MetadataValue label='协议版本' value={provider.protocol_version} />
            <MetadataValue label='插件版本' value={provider.plugin_version} />
            <MetadataValue
              label='最近检查'
              value={provider.last_checked_at || '尚未检查'}
            />
            <MetadataValue
              label='冷却状态'
              value={provider.cooldown_until || '无冷却'}
            />
          </div>
          <div className='flex flex-wrap gap-2'>
            {(provider.capabilities ?? []).map((capability) => (
              <Badge key={capability} variant='outline'>
                {capability}
              </Badge>
            ))}
          </div>
          {provider.failure_reason ? (
            <Alert variant='destructive'>
              <AlertTitle>最近失败原因</AlertTitle>
              <AlertDescription>{provider.failure_reason}</AlertDescription>
            </Alert>
          ) : null}
          <ReferenceSummary usage={usage} loading={loading} />
        </CardContent>
      </Card>
    </div>
  )
}

function ReferenceSummary({
  usage,
  loading,
}: {
  usage?: {
    metadata_profiles: PluginUsageReference[]
    library_metadata_strategies: PluginUsageReference[]
    media_sources: PluginUsageReference[]
    active_reference_count: number
  }
  loading: boolean
}) {
  if (loading) {
    return (
      <div className='text-sm text-muted-foreground'>正在加载引用关系...</div>
    )
  }
  if (!usage || usage.active_reference_count === 0) {
    return (
      <Alert>
        <AlertTitle>暂无活动引用</AlertTitle>
        <AlertDescription>
          禁用、更新或卸载前仍会再次检查引用关系。
        </AlertDescription>
      </Alert>
    )
  }
  return (
    <Alert>
      <BoxesIcon className='size-4' />
      <AlertTitle>存在活动引用</AlertTitle>
      <AlertDescription className='space-y-3'>
        <div>
          共有 {usage.active_reference_count}{' '}
          个元数据模板、媒体库策略或媒体源引用该插件。
        </div>
        <ReferenceList title='元数据模板' items={usage.metadata_profiles} />
        <ReferenceList
          title='媒体库策略'
          items={usage.library_metadata_strategies}
        />
        <ReferenceList title='媒体源' items={usage.media_sources} />
      </AlertDescription>
    </Alert>
  )
}

function ReferenceList({
  title,
  items,
}: {
  title: string
  items: PluginUsageReference[]
}) {
  if (!items.length) return null
  return (
    <div>
      <div className='font-medium text-foreground'>{title}</div>
      <div className='mt-1 flex flex-wrap gap-2'>
        {items.map((item) => (
          <Badge
            key={`${item.kind}-${item.id}-${item.stage}`}
            variant='outline'
          >
            {item.name}
            {item.stage ? ` · ${item.stage}` : ''}
          </Badge>
        ))}
      </div>
    </div>
  )
}

function CatalogPanel({
  sources,
  entries,
}: {
  sources: Array<{ id: number; name: string; trust_level: string }>
  entries: Array<{
    id: number
    name: string
    version: string
    compatibility?: { compatible: boolean; reasons: string[] }
    signature_status?: string
    release_notes?: string
  }>
}) {
  if (!sources.length) {
    return (
      <EmptyManagementCard text='尚未配置插件目录来源。目录、更新和回滚能力已预留，配置可信来源后才会启用安装或更新。' />
    )
  }
  return (
    <div className='grid gap-4 xl:grid-cols-2'>
      {entries.map((entry) => (
        <Card key={entry.id} className='border-border/60 py-0 shadow-sm'>
          <CardHeader className='py-5'>
            <CardTitle>{entry.name}</CardTitle>
            <CardDescription>
              {entry.version} · {entry.signature_status || '未声明签名'}
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-3 py-5'>
            <Badge
              variant={
                entry.compatibility?.compatible ? 'secondary' : 'outline'
              }
            >
              {entry.compatibility?.compatible ? '兼容' : '暂不可安装'}
            </Badge>
            {entry.compatibility?.reasons?.length ? (
              <div className='text-sm text-muted-foreground'>
                {entry.compatibility.reasons.join('；')}
              </div>
            ) : null}
            {entry.release_notes ? (
              <div className='text-sm text-muted-foreground'>
                {entry.release_notes}
              </div>
            ) : null}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
