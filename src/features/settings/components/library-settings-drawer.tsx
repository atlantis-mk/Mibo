import { useEffect, useState, type ReactNode } from 'react'
import { LoaderCircleIcon, PlusIcon } from 'lucide-react'
import { toast } from 'sonner'
import type {
  LibraryAccessTag,
  Library,
  LibraryMetadataPolicy,
  LibraryMetadataStrategy,
  LibraryPath,
  LibraryPlaybackPolicy,
  LibraryScanPolicy,
  MetadataProviderInstance,
  MetadataProfile,
  LibrarySubtitlePolicy,
  MediaSource,
  ScanExclusionRule,
} from '@/lib/mibo-api'
import type { createAuthedMiboApi } from '@/lib/mibo-query'
import { Button } from '@/components/ui/button'
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from '@/components/ui/drawer'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { FileSizeField } from './file-size-field'
import {
  buildScanExclusionRuleDraft,
  LibraryScanExclusionRulesEditor,
  normalizeScanExclusionRuleDrafts,
  type LibraryScanExclusionRuleDraft,
} from './library-scan-exclusion-rules-editor'

const DRAWER_CLASS_NAME =
  'h-[100vh] max-h-[100vh] data-[vaul-drawer-direction=right]:w-[720px] data-[vaul-drawer-direction=right]:max-w-[720px] data-[vaul-drawer-direction=right]:sm:max-w-[720px] max-sm:data-[vaul-drawer-direction=right]:w-full max-sm:data-[vaul-drawer-direction=right]:max-w-[100vw]'

const HIDDEN_METADATA_PROVIDER_TYPES = new Set(['local_scan'])

type Api = ReturnType<typeof createAuthedMiboApi> | null

function hasAnyMetadataStageProvider(strategy: LibraryMetadataStrategy) {
  return [strategy.search_provider_ids, strategy.detail_provider_ids].some(
    (ids) => ids.length > 0
  )
}

function applyMetadataProfileToStrategy(
  strategy: LibraryMetadataStrategy,
  metadataProfiles: MetadataProfile[],
  profileId: number,
  options: { onlyWhenStagesEmpty?: boolean } = {}
) {
  if (options.onlyWhenStagesEmpty && hasAnyMetadataStageProvider(strategy)) {
    return strategy
  }

  const profile = metadataProfiles.find((item) => item.id === profileId)
  const previousProfile = metadataProfiles.find(
    (item) => item.id === strategy.template_profile_id
  )
  if (!profile) return strategy

  const preferredMetadataLanguage =
    !strategy.preferred_metadata_language ||
    strategy.preferred_metadata_language ===
      (previousProfile?.preferred_metadata_language || '')
      ? profile.preferred_metadata_language || ''
      : strategy.preferred_metadata_language

  return {
    ...strategy,
    template_profile_id: profileId,
    search_provider_ids: profile.search_provider_ids,
    detail_provider_ids: profile.detail_provider_ids,
    preferred_metadata_language: preferredMetadataLanguage,
  }
}

export function LibrarySettingsDrawer({
  open,
  library,
  mediaSources,
  api,
  onOpenChange,
  onSaved,
}: {
  open: boolean
  library: Library | null
  mediaSources: MediaSource[]
  api: Api
  onOpenChange: (open: boolean) => void
  onSaved: () => Promise<void>
}) {
  const [paths, setPaths] = useState<LibraryPath[]>([])
  const [visibilityMode, setVisibilityMode] = useState<
    'default_open' | 'allow_list_only'
  >('default_open')
  const [accessTags, setAccessTags] = useState<string[]>([])
  const [availableAccessTags, setAvailableAccessTags] = useState<
    LibraryAccessTag[]
  >([])
  const [scan, setScan] = useState<LibraryScanPolicy | null>(null)
  const [metadata, setMetadata] = useState<LibraryMetadataPolicy | null>(null)
  const [metadataStrategy, setMetadataStrategy] =
    useState<LibraryMetadataStrategy | null>(null)
  const [metadataProfiles, setMetadataProfiles] = useState<MetadataProfile[]>(
    []
  )
  const [providerInstances, setProviderInstances] = useState<
    MetadataProviderInstance[]
  >([])
  const [playback, setPlayback] = useState<LibraryPlaybackPolicy | null>(null)
  const [subtitle, setSubtitle] = useState<LibrarySubtitlePolicy | null>(null)
  const [scanExclusionRules, setScanExclusionRules] = useState<
    LibraryScanExclusionRuleDraft[]
  >([])
  const [pending, setPending] = useState(false)
  const [newPath, setNewPath] = useState({ mediaSourceId: '', rootPath: '' })

  useEffect(() => {
    if (!open || !library || !api) return
    let cancelled = false
    setPending(true)
    setNewPath({ mediaSourceId: String(library.media_source_id), rootPath: '' })
    Promise.all([
      api.listLibraryPaths(library.id),
      api.getLibraryPolicies(library.id),
      api.getLibraryMetadataStrategy(library.id),
      api.listMetadataProfiles(),
      api.listMetadataProviderInstances(),
      api.listScanExclusionRules(),
      api.listLibraryAccessTags(),
    ])
      .then(
        ([
          nextPaths,
          policies,
          strategy,
          profiles,
          providers,
          rules,
          libraryAccessTags,
        ]) => {
          if (cancelled) return
          setPaths(nextPaths)
          setVisibilityMode(library.visibility_mode ?? 'default_open')
          setAccessTags((library.access_tags ?? []).map((tag) => tag.name))
          setAvailableAccessTags(libraryAccessTags)
          setScan(policies.scan)
          setMetadata(policies.metadata)
          setMetadataStrategy(
            strategy.template_profile_id
              ? applyMetadataProfileToStrategy(
                  strategy,
                  profiles,
                  strategy.template_profile_id,
                  { onlyWhenStagesEmpty: true }
                )
              : strategy
          )
          setMetadataProfiles(profiles)
          setProviderInstances(providers)
          setPlayback(policies.playback)
          setSubtitle(policies.subtitle)
          setNewPath({
            mediaSourceId: String(library.media_source_id),
            rootPath: '',
          })
          setScanExclusionRules(
            rules
              .filter(
                (rule: ScanExclusionRule) =>
                  rule.library_id === library.id && !rule.system
              )
              .map(buildScanExclusionRuleDraft)
          )
        }
      )
      .catch((error) => {
        if (!cancelled) {
          toast.error(
            error instanceof Error ? error.message : '加载媒体库配置失败。'
          )
        }
      })
      .finally(() => {
        if (!cancelled) setPending(false)
      })
    return () => {
      cancelled = true
    }
  }, [api, library, open])

  async function runAction(action: () => Promise<void>, success: string) {
    if (!api || !library) return
    setPending(true)
    try {
      await action()
      toast.success(success)
      await onSaved()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '操作失败。')
    } finally {
      setPending(false)
    }
  }

  async function addPath() {
    if (!api || !library) return
    const mediaSourceId = Number(newPath.mediaSourceId)
    await runAction(async () => {
      const created = await api.addLibraryPath(library.id, {
        media_source_id: mediaSourceId,
        root_path: newPath.rootPath,
      })
      setPaths((current) => [...current, created])
      setNewPath({ mediaSourceId: String(mediaSourceId), rootPath: '' })
    }, '路径已添加。')
  }

  async function saveAccessTags() {
    if (!api || !library) return
    await runAction(async () => {
      const result = await api.updateLibraryAccessTags(library.id, accessTags)
      setAccessTags(result.access_tags.map((tag) => tag.name))
    }, '访问标签已更新。')
  }

  async function saveVisibilityMode() {
    if (!api || !library) return
    await runAction(async () => {
      const result = await api.updateLibraryVisibilityMode(
        library.id,
        visibilityMode
      )
      setVisibilityMode(
        result.visibility_mode as 'default_open' | 'allow_list_only'
      )
    }, '可见性模式已更新。')
  }

  async function togglePath(path: LibraryPath, enabled: boolean) {
    if (!api || !library) return
    await runAction(
      async () => {
        const updated = await api.updateLibraryPath(library.id, path.id, {
          enabled,
        })
        setPaths((current) =>
          current.map((item) => (item.id === updated.id ? updated : item))
        )
      },
      enabled ? '路径已启用。' : '路径已停用。'
    )
  }

  function applyMetadataProfile(value: string) {
    if (!metadataStrategy || !metadata) return
    const profileId = Number(value)
    const profile = metadataProfiles.find((item) => item.id === profileId)
    const previousProfile = metadataProfiles.find(
      (item) => item.id === metadata.metadata_profile_id
    )
    const preferredMetadataLanguage =
      !metadata.preferred_metadata_language ||
      metadata.preferred_metadata_language ===
        (previousProfile?.preferred_metadata_language || '')
        ? profile?.preferred_metadata_language || ''
        : metadata.preferred_metadata_language
    setMetadataStrategy(
      applyMetadataProfileToStrategy(
        metadataStrategy,
        metadataProfiles,
        profileId
      )
    )
    setMetadata({
      ...metadata,
      metadata_profile_id: profileId,
      metadata_profile_name: profile?.name || '',
      preferred_metadata_language: preferredMetadataLanguage,
    })
  }

  if (!library) return null

  const configurableProviderInstances = providerInstances.filter(
    (provider) => !HIDDEN_METADATA_PROVIDER_TYPES.has(provider.provider_type)
  )

  return (
    <Drawer direction='right' open={open} onOpenChange={onOpenChange}>
      <DrawerContent className={DRAWER_CLASS_NAME}>
        <DrawerHeader className='border-b border-border/70 text-left'>
          <DrawerTitle>{library.name} 设置</DrawerTitle>
          <DrawerDescription>调整路径与策略。</DrawerDescription>
        </DrawerHeader>
        <ScrollArea className='min-h-0 flex-1'>
          <Tabs defaultValue='simple' className='grid gap-3 px-4 py-4'>
            <div className='w-fit rounded-xl border border-border/60 bg-background p-1'>
              <TabsList className='grid w-full grid-cols-2'>
                <TabsTrigger value='simple'>简单模式</TabsTrigger>
                <TabsTrigger value='advanced'>高级模式</TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value='simple' className='mt-0'>
              <div className='grid gap-5'>
                <PolicySection
                  title='访问标签'
                  description='未打标签的媒体库默认对所有已登录用户可见。'
                >
                  <Field>
                    <FieldLabel>可见性模式</FieldLabel>
                    <label className='flex items-center gap-3 rounded-xl border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground'>
                      <Switch
                        checked={visibilityMode === 'allow_list_only'}
                        onCheckedChange={(checked) =>
                          setVisibilityMode(
                            checked ? 'allow_list_only' : 'default_open'
                          )
                        }
                      />
                      <span>仅对命中 allow 标签规则的角色显示</span>
                    </label>
                  </Field>
                  <Field>
                    <FieldLabel>标签列表</FieldLabel>
                    <Input
                      value={accessTags.join(', ')}
                      onChange={(event) =>
                        setAccessTags(
                          Array.from(
                            new Set(
                              event.target.value
                                .split(',')
                                .map((item) => item.trim().toLowerCase())
                                .filter(Boolean)
                            )
                          )
                        )
                      }
                      placeholder='例如 kids, family'
                    />
                  </Field>
                  {accessTags.length === 0 ? (
                    <div className='rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-900'>
                      {visibilityMode === 'allow_list_only'
                        ? '当前库启用了“仅 allow 角色可见”，但还没有配置访问标签，这会导致没有角色能看到这个库。'
                        : '当前未设置访问标签，默认对所有已登录用户可见。'}
                    </div>
                  ) : null}
                  {availableAccessTags.length ? (
                    <div className='flex flex-wrap gap-2'>
                      {availableAccessTags.map((tag) => {
                        const active = accessTags.includes(tag.name)
                        return (
                          <Button
                            key={tag.id}
                            type='button'
                            variant={active ? 'default' : 'outline'}
                            size='sm'
                            onClick={() =>
                              setAccessTags((current) =>
                                active
                                  ? current.filter((item) => item !== tag.name)
                                  : [...current, tag.name].sort()
                              )
                            }
                          >
                            {tag.name}
                          </Button>
                        )
                      })}
                    </div>
                  ) : null}
                  <div className='flex gap-2'>
                    <Button
                      onClick={() => void saveVisibilityMode()}
                      disabled={pending}
                    >
                      保存可见性模式
                    </Button>
                    <Button
                      onClick={() => void saveAccessTags()}
                      disabled={pending}
                    >
                      保存访问标签
                    </Button>
                  </div>
                </PolicySection>
                <PolicySection title='源路径' description='增减扫描路径。'>
                  <div className='grid gap-2'>
                    {paths.map((path) => (
                      <div
                        key={path.id}
                        className='flex flex-col gap-2 rounded-lg border border-border/50 p-3 sm:flex-row sm:items-center sm:justify-between'
                      >
                        <div className='min-w-0'>
                          <div className='truncate text-sm font-medium'>
                            {path.root_path}
                          </div>
                          <div className='text-xs text-muted-foreground'>
                            媒体源 #{path.media_source_id}
                          </div>
                        </div>
                        <div className='flex items-center gap-2 text-sm'>
                          <span>{path.enabled ? '启用' : '停用'}</span>
                          <Switch
                            checked={path.enabled}
                            disabled={pending}
                            onCheckedChange={(checked) =>
                              void togglePath(path, checked)
                            }
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                  <div className='grid gap-3 rounded-lg border border-border/50 p-3 md:grid-cols-[180px_1fr_auto]'>
                    <Select
                      value={newPath.mediaSourceId}
                      onValueChange={(value) =>
                        setNewPath((current) => ({
                          ...current,
                          mediaSourceId: value,
                        }))
                      }
                    >
                      <SelectTrigger className='w-full'>
                        <SelectValue placeholder='媒体源' />
                      </SelectTrigger>
                      <SelectContent>
                        {mediaSources.map((source) => (
                          <SelectItem key={source.id} value={String(source.id)}>
                            {source.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Input
                      value={newPath.rootPath}
                      onChange={(event) =>
                        setNewPath((current) => ({
                          ...current,
                          rootPath: event.target.value,
                        }))
                      }
                      placeholder='输入要添加的绝对路径'
                    />
                    <Button
                      onClick={() => void addPath()}
                      disabled={
                        pending || !newPath.mediaSourceId || !newPath.rootPath
                      }
                    >
                      <PlusIcon className='size-4' />
                      添加
                    </Button>
                  </div>
                </PolicySection>

                {scan ? (
                  <PolicySection title='扫描策略' description='常用扫描项。'>
                    <ToggleRow
                      label='扫描启用'
                      checked={scan.scanner_enabled}
                      onChange={(checked) =>
                        setScan({ ...scan, scanner_enabled: checked })
                      }
                    />
                    <ToggleRow
                      label='实时监听'
                      checked={scan.realtime_monitor_enabled}
                      onChange={(checked) =>
                        setScan({ ...scan, realtime_monitor_enabled: checked })
                      }
                    />
                    <ToggleRow
                      label='定时刷新'
                      checked={scan.scheduled_refresh_enabled}
                      onChange={(checked) =>
                        setScan({ ...scan, scheduled_refresh_enabled: checked })
                      }
                    />
                    <NumberField
                      label='刷新间隔（小时）'
                      value={scan.refresh_interval_hours}
                      onChange={(value) =>
                        setScan({
                          ...scan,
                          refresh_interval_hours: Math.max(1, value),
                        })
                      }
                    />
                    <Button
                      disabled={pending}
                      onClick={() =>
                        void runAction(async () => {
                          if (!api || !scan) return
                          setScan(
                            await api.updateLibraryScanPolicy(library.id, scan)
                          )
                        }, '扫描策略已保存。')
                      }
                    >
                      保存扫描策略
                    </Button>
                  </PolicySection>
                ) : null}

                {metadata && metadataStrategy ? (
                  <PolicySection title='元数据策略' description='模板与语言。'>
                    <ToggleRow
                      label='读取本地元数据'
                      description='开启后会读取同目录下的本地 NFO/JSON 文件。'
                      checked={metadata.local_metadata_enabled}
                      onChange={(checked) =>
                        setMetadata({
                          ...metadata,
                          local_metadata_enabled: checked,
                        })
                      }
                    />
                    <Field>
                      <FieldLabel>Metadata Template</FieldLabel>
                      <Select
                        value={String(
                          metadataStrategy.template_profile_id || ''
                        )}
                        onValueChange={applyMetadataProfile}
                      >
                        <SelectTrigger className='w-full'>
                          <SelectValue placeholder='选择 metadata template' />
                        </SelectTrigger>
                        <SelectContent>
                          {metadataProfiles.map((profile) => (
                            <SelectItem
                              key={profile.id}
                              value={String(profile.id)}
                            >
                              {profile.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </Field>
                    <Field>
                      <FieldLabel>元数据语言</FieldLabel>
                      <Input
                        value={
                          metadataStrategy.preferred_metadata_language || ''
                        }
                        onChange={(event) => {
                          setMetadata({
                            ...metadata,
                            preferred_metadata_language: event.target.value,
                          })
                          setMetadataStrategy({
                            ...metadataStrategy,
                            preferred_metadata_language: event.target.value,
                          })
                        }}
                        placeholder='zh-CN'
                      />
                    </Field>
                    <div className='flex flex-wrap gap-2'>
                      <Button
                        disabled={pending}
                        onClick={() =>
                          void runAction(async () => {
                            if (!api || !metadata) return
                            setMetadata(
                              await api.updateLibraryMetadataPolicy(
                                library.id,
                                metadata
                              )
                            )
                          }, '元数据设置已保存。')
                        }
                      >
                        保存元数据设置
                      </Button>
                      <Button
                        disabled={pending}
                        onClick={() =>
                          void runAction(async () => {
                            if (!api || !metadataStrategy) return
                            setMetadataStrategy(
                              await api.updateLibraryMetadataStrategy(
                                library.id,
                                {
                                  template_profile_id:
                                    metadataStrategy.template_profile_id,
                                  search_provider_ids:
                                    metadataStrategy.search_provider_ids,
                                  detail_provider_ids:
                                    metadataStrategy.detail_provider_ids,
                                  preferred_metadata_language:
                                    metadataStrategy.preferred_metadata_language,
                                }
                              )
                            )
                          }, '元数据执行策略已保存。')
                        }
                      >
                        保存元数据执行策略
                      </Button>
                    </div>
                  </PolicySection>
                ) : null}

                {playback ? (
                  <PolicySection title='播放策略' description='常用播放项。'>
                    <ToggleRow
                      label='记录播放进度'
                      checked={playback.resume_enabled}
                      onChange={(checked) =>
                        setPlayback({ ...playback, resume_enabled: checked })
                      }
                    />
                    <Button
                      disabled={pending}
                      onClick={() =>
                        void runAction(async () => {
                          if (!api || !playback) return
                          setPlayback(
                            await api.updateLibraryPlaybackPolicy(
                              library.id,
                              playback
                            )
                          )
                        }, '播放策略已保存。')
                      }
                    >
                      保存播放策略
                    </Button>
                  </PolicySection>
                ) : null}

                {subtitle ? (
                  <PolicySection title='字幕策略' description='字幕偏好。'>
                    <ToggleRow
                      label='启用外置字幕'
                      checked={subtitle.external_sidecars_enabled}
                      onChange={(checked) =>
                        setSubtitle({
                          ...subtitle,
                          external_sidecars_enabled: checked,
                        })
                      }
                    />
                    <Field>
                      <FieldLabel>首选字幕语言</FieldLabel>
                      <Input
                        value={subtitle.preferred_languages.join(',')}
                        onChange={(event) =>
                          setSubtitle({
                            ...subtitle,
                            preferred_languages: splitList(event.target.value),
                          })
                        }
                        placeholder='zh,en'
                      />
                    </Field>
                    <Button
                      disabled={pending}
                      onClick={() =>
                        void runAction(async () => {
                          if (!api || !subtitle) return
                          setSubtitle(
                            await api.updateLibrarySubtitlePolicy(
                              library.id,
                              subtitle
                            )
                          )
                        }, '字幕策略已保存。')
                      }
                    >
                      保存字幕策略
                    </Button>
                  </PolicySection>
                ) : null}
              </div>
            </TabsContent>

            <TabsContent value='advanced' className='mt-0'>
              <div className='grid gap-5'>
                <PolicySection
                  title='源路径'
                  description='启用路径会参与扫描。'
                >
                  <div className='grid gap-2'>
                    {paths.map((path) => (
                      <div
                        key={path.id}
                        className='flex flex-col gap-2 rounded-lg border border-border/50 p-3 sm:flex-row sm:items-center sm:justify-between'
                      >
                        <div className='min-w-0'>
                          <div className='truncate text-sm font-medium'>
                            {path.root_path}
                          </div>
                          <div className='text-xs text-muted-foreground'>
                            媒体源 #{path.media_source_id}
                          </div>
                        </div>
                        <div className='flex items-center gap-2 text-sm'>
                          <span>{path.enabled ? '启用' : '停用'}</span>
                          <Switch
                            checked={path.enabled}
                            disabled={pending}
                            onCheckedChange={(checked) =>
                              void togglePath(path, checked)
                            }
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                  <div className='grid gap-3 rounded-lg border border-border/50 p-3 md:grid-cols-[180px_1fr_auto]'>
                    <Select
                      value={newPath.mediaSourceId}
                      onValueChange={(value) =>
                        setNewPath((current) => ({
                          ...current,
                          mediaSourceId: value,
                        }))
                      }
                    >
                      <SelectTrigger className='w-full'>
                        <SelectValue placeholder='媒体源' />
                      </SelectTrigger>
                      <SelectContent>
                        {mediaSources.map((source) => (
                          <SelectItem key={source.id} value={String(source.id)}>
                            {source.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Input
                      value={newPath.rootPath}
                      onChange={(event) =>
                        setNewPath((current) => ({
                          ...current,
                          rootPath: event.target.value,
                        }))
                      }
                      placeholder='输入要添加的绝对路径'
                    />
                    <Button
                      onClick={() => void addPath()}
                      disabled={
                        pending || !newPath.mediaSourceId || !newPath.rootPath
                      }
                    >
                      <PlusIcon className='size-4' />
                      添加
                    </Button>
                  </div>
                </PolicySection>

                {scan ? (
                  <PolicySection title='扫描策略' description='扫描与监听。'>
                    <ToggleRow
                      label='扫描启用'
                      checked={scan.scanner_enabled}
                      onChange={(checked) =>
                        setScan({ ...scan, scanner_enabled: checked })
                      }
                    />
                    <ToggleRow
                      label='实时监听'
                      checked={scan.realtime_monitor_enabled}
                      onChange={(checked) =>
                        setScan({ ...scan, realtime_monitor_enabled: checked })
                      }
                    />
                    <ToggleRow
                      label='批量探测库存'
                      description='关闭后扫描不会创建 inventory_probe_batch 批量探测任务。'
                      checked={scan.inventory_probe_batch_enabled}
                      onChange={(checked) =>
                        setScan({
                          ...scan,
                          inventory_probe_batch_enabled: checked,
                        })
                      }
                    />
                    <ToggleRow
                      label='定时刷新'
                      checked={scan.scheduled_refresh_enabled}
                      onChange={(checked) =>
                        setScan({ ...scan, scheduled_refresh_enabled: checked })
                      }
                    />
                    <NumberField
                      label='刷新间隔（小时）'
                      value={scan.refresh_interval_hours}
                      onChange={(value) =>
                        setScan({
                          ...scan,
                          refresh_interval_hours: Math.max(1, value),
                        })
                      }
                    />
                    <ToggleRow
                      label='隐藏文件忽略'
                      checked={scan.ignore_hidden_files}
                      onChange={(checked) =>
                        setScan({ ...scan, ignore_hidden_files: checked })
                      }
                    />
                    <Field>
                      <FieldLabel>忽略扩展名</FieldLabel>
                      <Input
                        value={scan.ignore_file_extensions.join(',')}
                        onChange={(event) =>
                          setScan({
                            ...scan,
                            ignore_file_extensions: splitList(
                              event.target.value
                            ),
                          })
                        }
                        placeholder='.txt,.jpg'
                      />
                    </Field>
                    <FileSizeField
                      label='最小文件大小（0 不限制）'
                      value={scan.min_file_size_bytes}
                      onChange={(value) =>
                        setScan({
                          ...scan,
                          min_file_size_bytes: Math.max(0, value),
                        })
                      }
                    />
                    <div className='grid gap-2'>
                      <div>
                        <h4 className='text-sm font-medium'>排除规则</h4>
                        <p className='text-xs leading-5 text-muted-foreground'>
                          这些规则仅作用于当前媒体库，会随扫描策略一起保存。
                        </p>
                      </div>
                      <LibraryScanExclusionRulesEditor
                        rules={scanExclusionRules}
                        onChange={setScanExclusionRules}
                        disabled={pending}
                      />
                    </div>
                    <Button
                      disabled={pending}
                      onClick={() =>
                        void runAction(async () => {
                          if (!api || !scan) return
                          setScan(
                            await api.updateLibraryScanPolicy(library.id, scan)
                          )
                          setScanExclusionRules(
                            (
                              await api.replaceLibraryScanExclusionRules(
                                library.id,
                                normalizeScanExclusionRuleDrafts(
                                  scanExclusionRules
                                )
                              )
                            ).map(buildScanExclusionRuleDraft)
                          )
                        }, '扫描策略已保存。')
                      }
                    >
                      保存扫描策略
                    </Button>
                  </PolicySection>
                ) : null}

                {metadata && metadataStrategy ? (
                  <PolicySection
                    title='元数据策略'
                    description='模板、语言与 provider 顺序。'
                  >
                    <ToggleRow
                      label='读取本地元数据'
                      description='开启后，metadata 阶段才会读取同目录下的本地 NFO/JSON 元数据文件。'
                      checked={metadata.local_metadata_enabled}
                      onChange={(checked) =>
                        setMetadata({
                          ...metadata,
                          local_metadata_enabled: checked,
                        })
                      }
                    />
                    <Field>
                      <FieldLabel>Metadata Template</FieldLabel>
                      <Select
                        value={String(
                          metadataStrategy.template_profile_id || ''
                        )}
                        onValueChange={applyMetadataProfile}
                      >
                        <SelectTrigger className='w-full'>
                          <SelectValue placeholder='选择 metadata template' />
                        </SelectTrigger>
                        <SelectContent>
                          {metadataProfiles.map((profile) => (
                            <SelectItem
                              key={profile.id}
                              value={String(profile.id)}
                            >
                              {profile.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <p className='text-xs leading-5 text-muted-foreground'>
                        模板用于初始化策略。
                      </p>
                    </Field>
                    <div className='grid gap-3 md:grid-cols-1'>
                      <Field>
                        <FieldLabel>元数据语言</FieldLabel>
                        <Input
                          value={
                            metadataStrategy.preferred_metadata_language || ''
                          }
                          onChange={(event) => {
                            setMetadata({
                              ...metadata,
                              preferred_metadata_language: event.target.value,
                            })
                            setMetadataStrategy({
                              ...metadataStrategy,
                              preferred_metadata_language: event.target.value,
                            })
                          }}
                          placeholder='zh-CN'
                        />
                      </Field>
                    </div>
                    <StrategyStageField
                      label='搜索阶段'
                      value={metadataStrategy.search_provider_ids}
                      providers={configurableProviderInstances}
                      onChange={(search_provider_ids) =>
                        setMetadataStrategy({
                          ...metadataStrategy,
                          search_provider_ids,
                        })
                      }
                    />
                    <StrategyStageField
                      label='详情阶段'
                      value={metadataStrategy.detail_provider_ids.filter((id) =>
                        configurableProviderInstances.some(
                          (provider) => provider.id === id
                        )
                      )}
                      providers={configurableProviderInstances}
                      onChange={(detail_provider_ids) =>
                        setMetadataStrategy({
                          ...metadataStrategy,
                          detail_provider_ids,
                        })
                      }
                    />
                    <div className='flex flex-wrap gap-2'>
                      <Button
                        disabled={pending}
                        onClick={() =>
                          void runAction(async () => {
                            if (!api || !metadata) return
                            setMetadata(
                              await api.updateLibraryMetadataPolicy(
                                library.id,
                                metadata
                              )
                            )
                          }, '元数据设置已保存。')
                        }
                      >
                        保存元数据设置
                      </Button>
                      <Button
                        disabled={pending}
                        onClick={() =>
                          void runAction(async () => {
                            if (!api || !metadataStrategy) return
                            setMetadataStrategy(
                              await api.updateLibraryMetadataStrategy(
                                library.id,
                                {
                                  template_profile_id:
                                    metadataStrategy.template_profile_id,
                                  search_provider_ids:
                                    metadataStrategy.search_provider_ids,
                                  detail_provider_ids:
                                    metadataStrategy.detail_provider_ids,
                                  preferred_metadata_language:
                                    metadataStrategy.preferred_metadata_language,
                                }
                              )
                            )
                          }, '元数据执行策略已保存。')
                        }
                      >
                        保存元数据执行策略
                      </Button>
                    </div>
                  </PolicySection>
                ) : null}

                {playback ? (
                  <PolicySection title='播放策略' description='播放进度阈值。'>
                    <ToggleRow
                      label='记录播放进度'
                      checked={playback.resume_enabled}
                      onChange={(checked) =>
                        setPlayback({ ...playback, resume_enabled: checked })
                      }
                    />
                    <div className='grid gap-3 md:grid-cols-2'>
                      <NumberField
                        label='完成百分比'
                        value={playback.max_resume_pct}
                        onChange={(value) =>
                          setPlayback({ ...playback, max_resume_pct: value })
                        }
                      />
                      <NumberField
                        label='最小时长秒数'
                        value={playback.min_resume_duration_seconds}
                        onChange={(value) =>
                          setPlayback({
                            ...playback,
                            min_resume_duration_seconds: value,
                          })
                        }
                      />
                    </div>
                    <Button
                      disabled={pending}
                      onClick={() =>
                        void runAction(async () => {
                          if (!api || !playback) return
                          setPlayback(
                            await api.updateLibraryPlaybackPolicy(
                              library.id,
                              playback
                            )
                          )
                        }, '播放策略已保存。')
                      }
                    >
                      保存播放策略
                    </Button>
                  </PolicySection>
                ) : null}

                {subtitle ? (
                  <PolicySection
                    title='字幕策略'
                    description='外置字幕与偏好。'
                  >
                    <ToggleRow
                      label='启用外置字幕'
                      checked={subtitle.external_sidecars_enabled}
                      onChange={(checked) =>
                        setSubtitle({
                          ...subtitle,
                          external_sidecars_enabled: checked,
                        })
                      }
                    />
                    <ToggleRow
                      label='容忍不可用字幕'
                      checked={subtitle.tolerate_unavailable_subtitles}
                      onChange={(checked) =>
                        setSubtitle({
                          ...subtitle,
                          tolerate_unavailable_subtitles: checked,
                        })
                      }
                    />
                    <Field>
                      <FieldLabel>首选字幕语言</FieldLabel>
                      <Input
                        value={subtitle.preferred_languages.join(',')}
                        onChange={(event) =>
                          setSubtitle({
                            ...subtitle,
                            preferred_languages: splitList(event.target.value),
                          })
                        }
                        placeholder='zh,en'
                      />
                    </Field>
                    <Button
                      disabled={pending}
                      onClick={() =>
                        void runAction(async () => {
                          if (!api || !subtitle) return
                          setSubtitle(
                            await api.updateLibrarySubtitlePolicy(
                              library.id,
                              subtitle
                            )
                          )
                        }, '字幕策略已保存。')
                      }
                    >
                      保存字幕策略
                    </Button>
                  </PolicySection>
                ) : null}
              </div>
            </TabsContent>

            {pending ? (
              <div className='flex items-center gap-2 text-sm text-muted-foreground'>
                <LoaderCircleIcon className='size-4 animate-spin' /> 正在处理...
              </div>
            ) : null}
          </Tabs>
        </ScrollArea>
      </DrawerContent>
    </Drawer>
  )
}

export function StrategyStageField({
  label,
  value,
  providers,
  onChange,
}: {
  label: string
  value: number[]
  providers: MetadataProviderInstance[]
  onChange: (value: number[]) => void
}) {
  const providerIds = new Set(providers.map((provider) => provider.id))
  const visibleValue = value.filter((id) => providerIds.has(id))

  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <div className='grid gap-2 rounded-lg border border-border/50 p-3'>
        <div className='text-xs text-muted-foreground'>
          通过勾选决定启用顺序。移除再重新勾选可以把 provider 放到更靠后的位置。
        </div>
        <div className='grid gap-2'>
          {providers.map((provider) => {
            const checked = visibleValue.includes(provider.id)
            return (
              <label
                key={provider.id}
                className='flex items-center justify-between rounded-md border border-border/50 px-3 py-2 text-sm'
              >
                <div className='min-w-0'>
                  <div className='font-medium'>{provider.name}</div>
                  <div className='text-xs text-muted-foreground'>
                    {provider.provider_type}
                    {provider.locked ? ' · read-only provider' : ''}
                  </div>
                </div>
                <Switch
                  checked={checked}
                  onCheckedChange={(nextChecked) => {
                    if (nextChecked) {
                      if (!checked) {
                        onChange([...visibleValue, provider.id])
                      }
                      return
                    }
                    onChange(
                      visibleValue.filter(
                        (selected) => selected !== provider.id
                      )
                    )
                  }}
                />
              </label>
            )
          })}
        </div>
        <div className='text-xs text-muted-foreground'>
          当前顺序：
          {visibleValue.length > 0
            ? ` ${visibleValue
                .map(
                  (id) =>
                    providers.find((provider) => provider.id === id)?.name ||
                    `#${id}`
                )
                .join(' -> ')}`
            : ' 未配置'}
        </div>
      </div>
    </Field>
  )
}

function PolicySection({
  title,
  description,
  children,
}: {
  title: string
  description: string
  children: ReactNode
}) {
  return (
    <section className='grid gap-3 py-1'>
      <div className='space-y-0.5'>
        <h3 className='text-sm font-medium'>{title}</h3>
        <p className='text-xs text-muted-foreground'>{description}</p>
      </div>
      {children}
    </section>
  )
}

function ToggleRow({
  label,
  description,
  checked,
  onChange,
}: {
  label: string
  description?: string
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <div className='flex items-center justify-between gap-3 rounded-lg border border-border/50 px-3 py-2 text-sm'>
      <span className='grid gap-0.5'>
        <span>{label}</span>
        {description ? (
          <span className='text-xs leading-5 text-muted-foreground'>
            {description}
          </span>
        ) : null}
      </span>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}

function NumberField({
  label,
  value,
  onChange,
}: {
  label: string
  value: number
  onChange: (value: number) => void
}) {
  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <Input
        type='number'
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </Field>
  )
}

function splitList(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}
