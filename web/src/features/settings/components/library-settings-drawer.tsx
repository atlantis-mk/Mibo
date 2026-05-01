import { useEffect, useState, type ReactNode } from "react"
import { LoaderCircleIcon, PlusIcon } from "lucide-react"

import { Button } from "#/components/ui/button"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "#/components/ui/drawer"
import { Field, FieldLabel } from "#/components/ui/field"
import { Input } from "#/components/ui/input"
import { ScrollArea } from "#/components/ui/scroll-area"
import { Separator } from "#/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "#/components/ui/select"
import { Switch } from "#/components/ui/switch"
import type {
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
} from "#/lib/mibo-api"
import type { createAuthedMiboApi } from "#/lib/mibo-query"

import {
  buildScanExclusionRuleDraft,
  LibraryScanExclusionRulesEditor,
  normalizeScanExclusionRuleDrafts,
  type LibraryScanExclusionRuleDraft,
} from "./library-scan-exclusion-rules-editor"

const DRAWER_CLASS_NAME =
  "h-[100vh] max-h-[100vh] data-[vaul-drawer-direction=right]:w-[960px] data-[vaul-drawer-direction=right]:max-w-[960px] data-[vaul-drawer-direction=right]:sm:max-w-[960px] max-sm:data-[vaul-drawer-direction=right]:w-full max-sm:data-[vaul-drawer-direction=right]:max-w-[100vw]"

const HIDDEN_METADATA_PROVIDER_TYPES = new Set(["local_scan"])

type Api = ReturnType<typeof createAuthedMiboApi> | null

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
  const [message, setMessage] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [newPath, setNewPath] = useState({ mediaSourceId: "", rootPath: "" })

  useEffect(() => {
    if (!open || !library || !api) return
    let cancelled = false
    setPending(true)
    setMessage(null)
    Promise.all([
      api.listLibraryPaths(library.id),
      api.getLibraryPolicies(library.id),
      api.getLibraryMetadataStrategy(library.id),
      api.listMetadataProfiles(),
      api.listMetadataProviderInstances(),
      api.listScanExclusionRules(),
    ])
      .then(([nextPaths, policies, strategy, profiles, providers, rules]) => {
        if (cancelled) return
        setPaths(nextPaths)
        setScan(policies.scan)
        setMetadata(policies.metadata)
        setMetadataStrategy(strategy)
        setMetadataProfiles(profiles)
        setProviderInstances(providers)
        setPlayback(policies.playback)
        setSubtitle(policies.subtitle)
        setNewPath({
          mediaSourceId: String(library.media_source_id),
          rootPath: "",
        })
        setScanExclusionRules(
          rules
            .filter(
              (rule: ScanExclusionRule) =>
                rule.library_id === library.id && !rule.system
            )
            .map(buildScanExclusionRuleDraft)
        )
      })
      .catch((error) => {
        if (!cancelled) {
          setMessage(
            error instanceof Error ? error.message : "加载媒体库配置失败。"
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
    setMessage(null)
    try {
      await action()
      setMessage(success)
      await onSaved()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "操作失败。")
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
      setNewPath({ mediaSourceId: String(mediaSourceId), rootPath: "" })
    }, "路径已添加。")
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
      enabled ? "路径已启用。" : "路径已停用。"
    )
  }

  if (!library) return null

  const configurableProviderInstances = providerInstances.filter(
    (provider) => !HIDDEN_METADATA_PROVIDER_TYPES.has(provider.provider_type)
  )

  return (
    <Drawer direction="right" open={open} onOpenChange={onOpenChange}>
      <DrawerContent className={DRAWER_CLASS_NAME}>
        <DrawerHeader className="border-b border-border/70 text-left">
          <DrawerTitle>{library.name} 设置</DrawerTitle>
          <DrawerDescription>
            管理多路径、扫描、元数据、播放和字幕策略。
          </DrawerDescription>
        </DrawerHeader>
        <ScrollArea className="min-h-0 flex-1">
          <div className="grid gap-5 px-4 py-4">
            {message ? (
              <div className="rounded-xl border border-border bg-muted px-3 py-2 text-sm">
                {message}
              </div>
            ) : null}
            <PolicySection title="源路径" description="启用的路径会参与扫描。">
              <div className="grid gap-2">
                {paths.map((path) => (
                  <div
                    key={path.id}
                    className="flex flex-col gap-2 rounded-xl border border-border/70 p-3 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium">
                        {path.root_path}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        媒体源 #{path.media_source_id}
                      </div>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <span>{path.enabled ? "启用" : "停用"}</span>
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
              <div className="grid gap-3 rounded-xl border border-border/70 p-3 md:grid-cols-[180px_1fr_auto]">
                <Select
                  value={newPath.mediaSourceId}
                  onValueChange={(value) =>
                    setNewPath((current) => ({
                      ...current,
                      mediaSourceId: value,
                    }))
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="媒体源" />
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
                  placeholder="输入要添加的绝对路径"
                />
                <Button
                  onClick={() => void addPath()}
                  disabled={
                    pending || !newPath.mediaSourceId || !newPath.rootPath
                  }
                >
                  <PlusIcon className="size-4" />
                  添加
                </Button>
              </div>
            </PolicySection>

            {scan ? (
              <PolicySection
                title="扫描策略"
                description="控制扫描与监听行为。"
              >
                <ToggleRow
                  label="扫描启用"
                  checked={scan.scanner_enabled}
                  onChange={(checked) =>
                    setScan({ ...scan, scanner_enabled: checked })
                  }
                />
                <ToggleRow
                  label="实时监听"
                  checked={scan.realtime_monitor_enabled}
                  onChange={(checked) =>
                    setScan({ ...scan, realtime_monitor_enabled: checked })
                  }
                />
                <ToggleRow
                  label="隐藏文件忽略"
                  checked={scan.ignore_hidden_files}
                  onChange={(checked) =>
                    setScan({ ...scan, ignore_hidden_files: checked })
                  }
                />
                <Field>
                  <FieldLabel>忽略扩展名</FieldLabel>
                  <Input
                    value={scan.ignore_file_extensions.join(",")}
                    onChange={(event) =>
                      setScan({
                        ...scan,
                        ignore_file_extensions: splitList(event.target.value),
                      })
                    }
                    placeholder=".txt,.jpg"
                  />
                </Field>
                <NumberField
                  label="最小文件大小（字节，0 不限制）"
                  value={scan.min_file_size_bytes}
                  onChange={(value) =>
                    setScan({
                      ...scan,
                      min_file_size_bytes: Math.max(0, value),
                    })
                  }
                />
                <div className="grid gap-2">
                  <div>
                    <h4 className="text-sm font-medium">排除规则</h4>
                    <p className="text-xs leading-5 text-muted-foreground">
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
                            normalizeScanExclusionRuleDrafts(scanExclusionRules)
                          )
                        ).map(buildScanExclusionRuleDraft)
                      )
                    }, "扫描策略已保存。")
                  }
                >
                  保存扫描策略
                </Button>
              </PolicySection>
            ) : null}

            {metadata && metadataStrategy ? (
              <PolicySection
                title="元数据策略"
                description="控制可执行 provider 顺序、模板来源和语言 override。"
              >
                <Field>
                  <FieldLabel>Metadata Template</FieldLabel>
                  <Select
                    value={String(metadataStrategy.template_profile_id || "")}
                    onValueChange={(value) =>
                      setMetadataStrategy({
                        ...metadataStrategy,
                        template_profile_id: Number(value),
                      })
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="选择 metadata template" />
                    </SelectTrigger>
                    <SelectContent>
                      {metadataProfiles.map((profile) => (
                        <SelectItem key={profile.id} value={String(profile.id)}>
                          {profile.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-xs leading-5 text-muted-foreground">
                    模板只用于复制默认策略；保存后，当前媒体库会持有自己的可执行
                    stage 顺序。
                  </p>
                </Field>
                <div className="grid gap-3 md:grid-cols-3">
                  <Field>
                    <FieldLabel>元数据语言</FieldLabel>
                    <Input
                      value={metadataStrategy.preferred_metadata_language || ""}
                      onChange={(event) =>
                        setMetadataStrategy({
                          ...metadataStrategy,
                          preferred_metadata_language: event.target.value,
                        })
                      }
                      placeholder="zh-CN"
                    />
                  </Field>
                  <Field>
                    <FieldLabel>图片语言</FieldLabel>
                    <Input
                      value={metadataStrategy.preferred_image_language || ""}
                      onChange={(event) =>
                        setMetadataStrategy({
                          ...metadataStrategy,
                          preferred_image_language: event.target.value,
                        })
                      }
                      placeholder="zh-CN"
                    />
                  </Field>
                  <Field>
                    <FieldLabel>国家/地区</FieldLabel>
                    <Input
                      value={metadataStrategy.metadata_country_code || ""}
                      onChange={(event) =>
                        setMetadataStrategy({
                          ...metadataStrategy,
                          metadata_country_code: event.target.value,
                        })
                      }
                      placeholder="CN"
                    />
                  </Field>
                </div>
                <StrategyStageField
                  label="搜索阶段"
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
                  label="详情阶段"
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
                <StrategyStageField
                  label="图片阶段"
                  value={metadataStrategy.image_provider_ids}
                  providers={configurableProviderInstances}
                  onChange={(image_provider_ids) =>
                    setMetadataStrategy({
                      ...metadataStrategy,
                      image_provider_ids,
                    })
                  }
                />
                <StrategyStageField
                  label="人物阶段"
                  value={metadataStrategy.people_provider_ids}
                  providers={configurableProviderInstances}
                  onChange={(people_provider_ids) =>
                    setMetadataStrategy({
                      ...metadataStrategy,
                      people_provider_ids,
                    })
                  }
                />
                <StrategyStageField
                  label="层级阶段"
                  value={metadataStrategy.hierarchy_provider_ids}
                  providers={configurableProviderInstances}
                  onChange={(hierarchy_provider_ids) =>
                    setMetadataStrategy({
                      ...metadataStrategy,
                      hierarchy_provider_ids,
                    })
                  }
                />
                <Button
                  disabled={pending}
                  onClick={() =>
                    void runAction(async () => {
                      if (!api || !metadataStrategy) return
                      setMetadataStrategy(
                        await api.updateLibraryMetadataStrategy(library.id, {
                          template_profile_id:
                            metadataStrategy.template_profile_id,
                          search_provider_ids:
                            metadataStrategy.search_provider_ids,
                          detail_provider_ids:
                            metadataStrategy.detail_provider_ids,
                          image_provider_ids:
                            metadataStrategy.image_provider_ids,
                          people_provider_ids:
                            metadataStrategy.people_provider_ids,
                          hierarchy_provider_ids:
                            metadataStrategy.hierarchy_provider_ids,
                          preferred_metadata_language:
                            metadataStrategy.preferred_metadata_language,
                          preferred_image_language:
                            metadataStrategy.preferred_image_language,
                          metadata_country_code:
                            metadataStrategy.metadata_country_code,
                        })
                      )
                    }, "元数据策略已保存。")
                  }
                >
                  保存元数据策略
                </Button>
              </PolicySection>
            ) : null}

            {playback ? (
              <PolicySection title="播放策略" description="控制播放进度阈值。">
                <ToggleRow
                  label="记录播放进度"
                  checked={playback.resume_enabled}
                  onChange={(checked) =>
                    setPlayback({ ...playback, resume_enabled: checked })
                  }
                />
                <div className="grid gap-3 md:grid-cols-3">
                  <NumberField
                    label="最小记录百分比"
                    value={playback.min_resume_pct}
                    onChange={(value) =>
                      setPlayback({ ...playback, min_resume_pct: value })
                    }
                  />
                  <NumberField
                    label="完成百分比"
                    value={playback.max_resume_pct}
                    onChange={(value) =>
                      setPlayback({ ...playback, max_resume_pct: value })
                    }
                  />
                  <NumberField
                    label="最小时长秒数"
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
                    }, "播放策略已保存。")
                  }
                >
                  保存播放策略
                </Button>
              </PolicySection>
            ) : null}

            {subtitle ? (
              <PolicySection
                title="字幕策略"
                description="控制外置字幕绑定和播放。"
              >
                <ToggleRow
                  label="启用外置字幕"
                  checked={subtitle.external_sidecars_enabled}
                  onChange={(checked) =>
                    setSubtitle({
                      ...subtitle,
                      external_sidecars_enabled: checked,
                    })
                  }
                />
                <ToggleRow
                  label="容忍不可用字幕"
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
                    value={subtitle.preferred_languages.join(",")}
                    onChange={(event) =>
                      setSubtitle({
                        ...subtitle,
                        preferred_languages: splitList(event.target.value),
                      })
                    }
                    placeholder="chi,eng"
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
                    }, "字幕策略已保存。")
                  }
                >
                  保存字幕策略
                </Button>
              </PolicySection>
            ) : null}
            {pending ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <LoaderCircleIcon className="size-4 animate-spin" /> 正在处理...
              </div>
            ) : null}
          </div>
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
      <div className="grid gap-2 rounded-xl border border-border/70 p-3">
        <div className="text-xs text-muted-foreground">
          通过勾选决定启用顺序。移除再重新勾选可以把 provider 放到更靠后的位置。
        </div>
        <div className="grid gap-2">
          {providers.map((provider) => {
            const checked = visibleValue.includes(provider.id)
            return (
              <label
                key={provider.id}
                className="flex items-center justify-between rounded-lg border border-border/60 px-3 py-2 text-sm"
              >
                <div className="min-w-0">
                  <div className="font-medium">{provider.name}</div>
                  <div className="text-xs text-muted-foreground">
                    {provider.provider_type}
                    {provider.locked ? " · read-only provider" : ""}
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
        <div className="text-xs text-muted-foreground">
          当前顺序：
          {visibleValue.length > 0
            ? ` ${visibleValue
                .map(
                  (id) =>
                    providers.find((provider) => provider.id === id)?.name ||
                    `#${id}`
                )
                .join(" -> ")}`
            : " 未配置"}
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
    <section className="grid gap-4 rounded-2xl border border-border/70 bg-card/50 p-4">
      <div>
        <h3 className="text-base font-medium">{title}</h3>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
      <Separator />
      {children}
    </section>
  )
}

function ToggleRow({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-border/60 px-3 py-2 text-sm">
      <span>{label}</span>
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
        type="number"
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </Field>
  )
}

function splitList(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
}
