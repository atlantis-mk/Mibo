import { PathPicker } from "#/components/path-picker"
import { Button } from "#/components/ui/button"
import { Field, FieldLabel } from "#/components/ui/field"
import { Input } from "#/components/ui/input"
import { Switch } from "#/components/ui/switch"
import type {
  LibraryMetadataPolicy,
  LibraryMetadataStrategyInput,
  LibraryPlaybackPolicy,
  LibraryScanPolicy,
  LibrarySubtitlePolicy,
  MediaSource,
  MetadataProviderInstance,
  MetadataProfile,
  ScanExclusionRuleInput,
  StorageBrowseResult,
} from "#/lib/mibo-api"
import { createAuthedMiboApi } from "#/lib/mibo-query"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "#/components/ui/select"
import { Separator } from "#/components/ui/separator"

import {
  LibraryScanExclusionRulesEditor,
  normalizeScanExclusionRuleDrafts,
  type LibraryScanExclusionRuleDraft,
} from "./library-scan-exclusion-rules-editor"
import { StrategyStageField } from "./library-settings-drawer"

export type LibraryFormState = {
  name: string
  mediaSourceId: string
  rootPath: string
  scan: LibraryScanPolicy
  metadata: LibraryMetadataPolicy
  metadataStrategy: LibraryMetadataStrategyInput
  playback: LibraryPlaybackPolicy
  subtitle: LibrarySubtitlePolicy
  scanExclusionRules: LibraryScanExclusionRuleDraft[]
}

export const EMPTY_LIBRARY_FORM: LibraryFormState = {
  name: "",
  mediaSourceId: "",
  rootPath: "",
  scan: {
    scanner_enabled: true,
    realtime_monitor_enabled: true,
    scheduled_refresh_enabled: true,
    refresh_interval_hours: 24,
    ignore_hidden_files: true,
    ignore_file_extensions: [],
    min_file_size_bytes: 0,
    sample_ignore_size_bytes: 0,
    inventory_probe_batch_enabled: true,
    configurable_exclusion_rules: true,
  },
  metadata: {
    preferred_metadata_language: "",
    preferred_image_language: "",
    metadata_country_code: "",
  },
  metadataStrategy: {
    search_provider_ids: [],
    detail_provider_ids: [],
    image_provider_ids: [],
    people_provider_ids: [],
    hierarchy_provider_ids: [],
  },
  playback: {
    resume_enabled: true,
    min_resume_pct: 5,
    max_resume_pct: 90,
    min_resume_duration_seconds: 300,
  },
  subtitle: {
    external_sidecars_enabled: true,
    preferred_languages: [],
    require_perfect_match: false,
    save_with_media: false,
    tolerate_unavailable_subtitles: true,
    skip_if_embedded_subtitles_present: false,
    skip_if_audio_track_matches: false,
  },
  scanExclusionRules: [],
}

export function libraryFormScanExclusionRuleInputs(
  draft: LibraryFormState
): ScanExclusionRuleInput[] {
  return normalizeScanExclusionRuleDrafts(draft.scanExclusionRules)
}

export function libraryFormMetadataStrategyInput(
  draft: LibraryFormState
): LibraryMetadataStrategyInput {
  return {
    ...draft.metadataStrategy,
    template_profile_id: draft.metadata.metadata_profile_id,
    preferred_metadata_language: draft.metadata.preferred_metadata_language,
    preferred_image_language: draft.metadata.preferred_image_language,
    metadata_country_code: draft.metadata.metadata_country_code,
  }
}

export function deriveLibraryNameFromPath(path: string): string {
  const trimmedPath = path.trim().replace(/[\\/]+$/, "")
  if (!trimmedPath) return ""

  return (
    trimmedPath
      .split(/[\\/]+/)
      .filter(Boolean)
      .at(-1) ?? ""
  )
}

function applyMetadataProfileToDraft(
  draft: LibraryFormState,
  metadataProfiles: MetadataProfile[],
  value: string
): LibraryFormState {
  const profileId = Number(value)
  const profile = metadataProfiles.find((item) => item.id === profileId)
  return {
    ...draft,
    metadata: {
      ...draft.metadata,
      metadata_profile_id: profileId,
      metadata_profile_name: profile?.name || "",
      preferred_metadata_language:
        draft.metadata.preferred_metadata_language ||
        profile?.preferred_metadata_language ||
        "",
      preferred_image_language:
        draft.metadata.preferred_image_language ||
        profile?.preferred_image_language ||
        "",
    },
    metadataStrategy: {
      ...draft.metadataStrategy,
      template_profile_id: profileId,
      search_provider_ids: profile?.search_provider_ids || [],
      detail_provider_ids: profile?.detail_provider_ids || [],
      image_provider_ids: profile?.image_provider_ids || [],
      people_provider_ids: profile?.people_provider_ids || [],
      hierarchy_provider_ids: profile?.hierarchy_provider_ids || [],
      preferred_metadata_language:
        draft.metadata.preferred_metadata_language ||
        profile?.preferred_metadata_language ||
        "",
      preferred_image_language:
        draft.metadata.preferred_image_language ||
        profile?.preferred_image_language ||
        "",
      metadata_country_code: draft.metadata.metadata_country_code,
    },
  }
}

export function LibraryForm({
  draft,
  onChange,
  mediaSources,
  metadataProfiles,
  metadataProviderInstances,
  api,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  mediaSources: MediaSource[]
  metadataProfiles: MetadataProfile[]
  metadataProviderInstances: MetadataProviderInstance[]
  api: ReturnType<typeof createAuthedMiboApi> | null
}) {
  const selectedSource =
    mediaSources.find((source) => String(source.id) === draft.mediaSourceId) ??
    null
  const selectedMetadataProfile =
    metadataProfiles.find(
      (profile) => profile.id === draft.metadata.metadata_profile_id
    ) ?? null
  const recommendedName = deriveLibraryNameFromPath(draft.rootPath)
  const configurableProviderInstances = metadataProviderInstances.filter(
    (provider) => provider.provider_type !== "local_scan"
  )

  async function browseExistingLibraryPath(
    path?: string,
    options?: { refresh?: boolean }
  ): Promise<StorageBrowseResult> {
    if (!api || !selectedSource) {
      throw new Error("请先选择媒体源。")
    }

    return api.browseMediaSource(selectedSource.id, path, options?.refresh)
  }

  return (
    <div className="grid gap-6">
      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">存储位置</h3>
          <p className="text-sm text-muted-foreground">
            选择已有媒体源，Mibo 会从这个目录开始探测和扫描内容。
          </p>
        </div>
        <Field>
          <FieldLabel>媒体源</FieldLabel>
          <Select
            value={draft.mediaSourceId}
            onValueChange={(value) =>
              onChange({ ...draft, mediaSourceId: value })
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="选择媒体源" />
            </SelectTrigger>
            <SelectContent>
              {mediaSources.map((source) => (
                <SelectItem key={source.id} value={String(source.id)}>
                  {source.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-xs leading-5 text-muted-foreground">
            {selectedSource
              ? `当前媒体源：#${selectedSource.id} · ${selectedSource.name} · ${selectedSource.provider} · 根路径 ${selectedSource.root_path}`
              : "请选择一个可复用的媒体源；如需新增，请先在“媒体源”标签中创建。"}
          </p>
        </Field>
      </section>

      <Separator />

      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">内容来源信息</h3>
          <p className="text-sm text-muted-foreground">
            填写显示名称即可。Mibo
            会在后台自动识别视频、音乐、文本以及电影或剧集语义。
          </p>
        </div>
        <div className="grid gap-4">
          <Field>
            <FieldLabel>来源名称</FieldLabel>
            <Input
              value={draft.name}
              onChange={(event) =>
                onChange({ ...draft, name: event.target.value })
              }
              placeholder="电影"
            />
            {recommendedName && !draft.name.trim() ? (
              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <span>推荐名称：{recommendedName}</span>
                <Button
                  type="button"
                  variant="link"
                  size="sm"
                  className="h-auto px-0 py-0 text-xs"
                  onClick={() => onChange({ ...draft, name: recommendedName })}
                >
                  使用推荐名称
                </Button>
              </div>
            ) : null}
          </Field>
        </div>
      </section>

      <Separator />

      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">挂载路径</h3>
          <p className="text-sm text-muted-foreground">
            浏览媒体源目录，选择这个内容来源的起始路径。
          </p>
        </div>
        <PathPicker
          browse={selectedSource ? browseExistingLibraryPath : null}
          browseKey={`existing:${selectedSource?.id ?? "none"}`}
          browseLabel="当前媒体源子目录"
          value={draft.rootPath}
          onValueChange={(rootPath) => onChange({ ...draft, rootPath })}
          placeholder={selectedSource?.root_path || "/"}
          selectCurrentOnBrowse
          ready={!!selectedSource}
          lockedMessage="先选择媒体源，再选择来源路径。"
        />
      </section>

      <Separator />

      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">扫描策略</h3>
          <p className="text-sm text-muted-foreground">控制扫描与监听行为。</p>
        </div>
        <ToggleRow
          label="扫描启用"
          checked={draft.scan.scanner_enabled}
          onChange={(checked) =>
            onChange({
              ...draft,
              scan: { ...draft.scan, scanner_enabled: checked },
            })
          }
        />
        <ToggleRow
          label="实时监听"
          checked={draft.scan.realtime_monitor_enabled}
          onChange={(checked) =>
            onChange({
              ...draft,
              scan: { ...draft.scan, realtime_monitor_enabled: checked },
            })
          }
        />
        <ToggleRow
          label="批量探测库存"
          description="关闭后扫描不会创建 inventory_probe_batch 批量探测任务。"
          checked={draft.scan.inventory_probe_batch_enabled}
          onChange={(checked) =>
            onChange({
              ...draft,
              scan: { ...draft.scan, inventory_probe_batch_enabled: checked },
            })
          }
        />
        <ToggleRow
          label="隐藏文件忽略"
          checked={draft.scan.ignore_hidden_files}
          onChange={(checked) =>
            onChange({
              ...draft,
              scan: { ...draft.scan, ignore_hidden_files: checked },
            })
          }
        />
        <Field>
          <FieldLabel>忽略扩展名</FieldLabel>
          <Input
            value={draft.scan.ignore_file_extensions.join(",")}
            onChange={(event) =>
              onChange({
                ...draft,
                scan: {
                  ...draft.scan,
                  ignore_file_extensions: splitList(event.target.value),
                },
              })
            }
            placeholder=".txt,.jpg"
          />
        </Field>
        <NumberField
          label="最小文件大小（字节，0 不限制）"
          value={draft.scan.min_file_size_bytes}
          onChange={(value) =>
            onChange({
              ...draft,
              scan: { ...draft.scan, min_file_size_bytes: Math.max(0, value) },
            })
          }
        />
        <div className="grid gap-2">
          <div>
            <h4 className="text-sm font-medium">排除规则</h4>
            <p className="text-xs leading-5 text-muted-foreground">
              规则会随内容来源一起保存，并在扫描时跳过匹配的视频。
            </p>
          </div>
          <LibraryScanExclusionRulesEditor
            rules={draft.scanExclusionRules}
            onChange={(scanExclusionRules) =>
              onChange({ ...draft, scanExclusionRules })
            }
          />
        </div>
      </section>

      <Separator />

      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">元数据策略</h3>
          <p className="text-sm text-muted-foreground">
            先选择模板作为起点，也可以在创建前直接调整每个阶段的 provider 顺序。
          </p>
        </div>
        <Field>
          <FieldLabel>Metadata Template</FieldLabel>
          <Select
            value={
              draft.metadata.metadata_profile_id
                ? String(draft.metadata.metadata_profile_id)
                : ""
            }
            onValueChange={(value) =>
              onChange(
                applyMetadataProfileToDraft(draft, metadataProfiles, value)
              )
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
            创建时会把下方阶段配置保存成内容来源自己的执行策略。
          </p>
          {selectedMetadataProfile?.locked ? (
            <p className="text-xs leading-5 text-muted-foreground">
              当前选择的是系统只读模板，常用于本地扫描优先或兼容迁移场景。
            </p>
          ) : null}
        </Field>
        <StrategyStageField
          label="搜索阶段"
          value={draft.metadataStrategy.search_provider_ids}
          providers={configurableProviderInstances}
          onChange={(search_provider_ids) =>
            onChange({
              ...draft,
              metadataStrategy: {
                ...draft.metadataStrategy,
                search_provider_ids,
              },
            })
          }
        />
        <StrategyStageField
          label="详情阶段"
          value={draft.metadataStrategy.detail_provider_ids}
          providers={configurableProviderInstances}
          onChange={(detail_provider_ids) =>
            onChange({
              ...draft,
              metadataStrategy: {
                ...draft.metadataStrategy,
                detail_provider_ids,
              },
            })
          }
        />
        <StrategyStageField
          label="图片阶段"
          value={draft.metadataStrategy.image_provider_ids}
          providers={configurableProviderInstances}
          onChange={(image_provider_ids) =>
            onChange({
              ...draft,
              metadataStrategy: {
                ...draft.metadataStrategy,
                image_provider_ids,
              },
            })
          }
        />
        <StrategyStageField
          label="人物阶段"
          value={draft.metadataStrategy.people_provider_ids}
          providers={configurableProviderInstances}
          onChange={(people_provider_ids) =>
            onChange({
              ...draft,
              metadataStrategy: {
                ...draft.metadataStrategy,
                people_provider_ids,
              },
            })
          }
        />
        <StrategyStageField
          label="层级阶段"
          value={draft.metadataStrategy.hierarchy_provider_ids}
          providers={configurableProviderInstances}
          onChange={(hierarchy_provider_ids) =>
            onChange({
              ...draft,
              metadataStrategy: {
                ...draft.metadataStrategy,
                hierarchy_provider_ids,
              },
            })
          }
        />
        <div className="grid gap-3 md:grid-cols-3">
          <Field>
            <FieldLabel>元数据语言</FieldLabel>
            <Input
              value={draft.metadata.preferred_metadata_language}
              onChange={(event) =>
                onChange({
                  ...draft,
                  metadata: {
                    ...draft.metadata,
                    preferred_metadata_language: event.target.value,
                  },
                  metadataStrategy: {
                    ...draft.metadataStrategy,
                    preferred_metadata_language: event.target.value,
                  },
                })
              }
              placeholder="zh-CN"
            />
          </Field>
          <Field>
            <FieldLabel>图片语言</FieldLabel>
            <Input
              value={draft.metadata.preferred_image_language}
              onChange={(event) =>
                onChange({
                  ...draft,
                  metadata: {
                    ...draft.metadata,
                    preferred_image_language: event.target.value,
                  },
                  metadataStrategy: {
                    ...draft.metadataStrategy,
                    preferred_image_language: event.target.value,
                  },
                })
              }
              placeholder="zh-CN"
            />
          </Field>
          <Field>
            <FieldLabel>国家/地区</FieldLabel>
            <Input
              value={draft.metadata.metadata_country_code}
              onChange={(event) =>
                onChange({
                  ...draft,
                  metadata: {
                    ...draft.metadata,
                    metadata_country_code: event.target.value,
                  },
                  metadataStrategy: {
                    ...draft.metadataStrategy,
                    metadata_country_code: event.target.value,
                  },
                })
              }
              placeholder="CN"
            />
          </Field>
        </div>
      </section>

      <Separator />

      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">播放策略</h3>
          <p className="text-sm text-muted-foreground">控制播放进度阈值。</p>
        </div>
        <ToggleRow
          label="记录播放进度"
          checked={draft.playback.resume_enabled}
          onChange={(checked) =>
            onChange({
              ...draft,
              playback: { ...draft.playback, resume_enabled: checked },
            })
          }
        />
        <div className="grid gap-3 md:grid-cols-3">
          <NumberField
            label="最小记录百分比"
            value={draft.playback.min_resume_pct}
            onChange={(value) =>
              onChange({
                ...draft,
                playback: { ...draft.playback, min_resume_pct: value },
              })
            }
          />
          <NumberField
            label="完成百分比"
            value={draft.playback.max_resume_pct}
            onChange={(value) =>
              onChange({
                ...draft,
                playback: { ...draft.playback, max_resume_pct: value },
              })
            }
          />
          <NumberField
            label="最小时长秒数"
            value={draft.playback.min_resume_duration_seconds}
            onChange={(value) =>
              onChange({
                ...draft,
                playback: {
                  ...draft.playback,
                  min_resume_duration_seconds: value,
                },
              })
            }
          />
        </div>
      </section>

      <Separator />

      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">字幕策略</h3>
          <p className="text-sm text-muted-foreground">
            控制外置字幕绑定和播放。
          </p>
        </div>
        <ToggleRow
          label="启用外置字幕"
          checked={draft.subtitle.external_sidecars_enabled}
          onChange={(checked) =>
            onChange({
              ...draft,
              subtitle: {
                ...draft.subtitle,
                external_sidecars_enabled: checked,
              },
            })
          }
        />
        <ToggleRow
          label="容忍不可用字幕"
          checked={draft.subtitle.tolerate_unavailable_subtitles}
          onChange={(checked) =>
            onChange({
              ...draft,
              subtitle: {
                ...draft.subtitle,
                tolerate_unavailable_subtitles: checked,
              },
            })
          }
        />
        <Field>
          <FieldLabel>首选字幕语言</FieldLabel>
          <Input
            value={draft.subtitle.preferred_languages.join(",")}
            onChange={(event) =>
              onChange({
                ...draft,
                subtitle: {
                  ...draft.subtitle,
                  preferred_languages: splitList(event.target.value),
                },
              })
            }
            placeholder="chi,eng"
          />
        </Field>
      </section>
    </div>
  )
}

function ToggleRow({
  label,
  description,
  checked,
  onChange,
  disabled = false,
}: {
  label: string
  description?: string
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-border/60 px-3 py-2 text-sm">
      <span className="grid gap-0.5">
        <span>{label}</span>
        {description ? (
          <span className="text-xs leading-5 text-muted-foreground">
            {description}
          </span>
        ) : null}
      </span>
      <Switch
        checked={checked}
        onCheckedChange={onChange}
        disabled={disabled}
      />
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
