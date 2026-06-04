import type {
  LibraryAccessTag,
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
} from '@/lib/mibo-api'
import type { createAuthedMiboApi } from '@/lib/mibo-query'
import { Button } from '@/components/ui/button'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { PathPicker } from '@/components/path-picker'
import { FileSizeField } from './file-size-field'
import {
  LibraryScanExclusionRulesEditor,
  normalizeScanExclusionRuleDrafts,
  type LibraryScanExclusionRuleDraft,
} from './library-scan-exclusion-rules-editor'
import { StrategyStageField } from './library-settings-drawer'

export type LibraryFormState = {
  name: string
  mediaSourceId: string
  rootPath: string
  visibilityMode: 'default_open' | 'allow_list_only'
  accessTags: string[]
  scan: LibraryScanPolicy
  metadata: LibraryMetadataPolicy
  metadataStrategy: LibraryMetadataStrategyInput
  playback: LibraryPlaybackPolicy
  subtitle: LibrarySubtitlePolicy
  scanExclusionRules: LibraryScanExclusionRuleDraft[]
}

export const EMPTY_LIBRARY_FORM: LibraryFormState = {
  name: '',
  mediaSourceId: '',
  rootPath: '',
  visibilityMode: 'default_open',
  accessTags: [],
  scan: {
    scanner_enabled: true,
    realtime_monitor_enabled: true,
    scheduled_refresh_enabled: true,
    refresh_interval_hours: 24,
    ignore_hidden_files: true,
    ignore_file_extensions: [],
    min_file_size_bytes: 0,
    sample_ignore_size_bytes: 0,
    inventory_probe_batch_enabled: false,
    configurable_exclusion_rules: true,
  },
  metadata: {
    preferred_metadata_language: '',
    local_metadata_enabled: true,
  },
  metadataStrategy: {
    search_provider_ids: [],
    detail_provider_ids: [],
  },
  playback: {
    resume_enabled: true,
    max_resume_pct: 90,
    min_resume_duration_seconds: 60,
  },
  subtitle: {
    external_sidecars_enabled: true,
    preferred_languages: [],
    tolerate_unavailable_subtitles: true,
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
  }
}

function deriveLibraryNameFromPath(path: string): string {
  const trimmedPath = path.trim().replace(/[\\/]+$/, '')
  if (!trimmedPath) return ''

  const segments = trimmedPath.split(/[\\/]+/).filter(Boolean)

  return segments[segments.length - 1] ?? ''
}

function applyMetadataProfileToDraft(
  draft: LibraryFormState,
  metadataProfiles: MetadataProfile[],
  value: string
): LibraryFormState {
  const profileId = Number(value)
  const profile = metadataProfiles.find((item) => item.id === profileId)
  const previousProfile = metadataProfiles.find(
    (item) => item.id === draft.metadata.metadata_profile_id
  )
  const preferredMetadataLanguage =
    !draft.metadata.preferred_metadata_language ||
    draft.metadata.preferred_metadata_language ===
      (previousProfile?.preferred_metadata_language || '')
      ? profile?.preferred_metadata_language || ''
      : draft.metadata.preferred_metadata_language
  return {
    ...draft,
    metadata: {
      ...draft.metadata,
      metadata_profile_id: profileId,
      metadata_profile_name: profile?.name || '',
      preferred_metadata_language: preferredMetadataLanguage,
    },
    metadataStrategy: {
      ...draft.metadataStrategy,
      template_profile_id: profileId,
      search_provider_ids: profile?.search_provider_ids || [],
      detail_provider_ids: profile?.detail_provider_ids || [],
      preferred_metadata_language: preferredMetadataLanguage,
    },
  }
}

export function LibraryForm({
  draft,
  onChange,
  mediaSources,
  availableAccessTags,
  metadataProfiles,
  metadataProviderInstances,
  api,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  mediaSources: MediaSource[]
  availableAccessTags: LibraryAccessTag[]
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
    (provider) => provider.provider_type !== 'local_scan'
  )

  async function browseExistingLibraryPath(
    path?: string,
    options?: { refresh?: boolean }
  ): Promise<StorageBrowseResult> {
    if (!api || !selectedSource) {
      throw new Error('请先选择媒体源。')
    }

    return api.browseMediaSource(selectedSource.id, path, options?.refresh)
  }

  return (
    <Tabs defaultValue='simple' className='grid gap-3'>
      <div className='w-fit rounded-xl border border-border/60 bg-background p-1'>
        <TabsList className='grid w-full grid-cols-2'>
          <TabsTrigger value='simple'>简单模式</TabsTrigger>
          <TabsTrigger value='advanced'>高级模式</TabsTrigger>
        </TabsList>
      </div>

      <TabsContent value='simple' className='mt-0'>
        <div className='grid gap-5'>
          <LibraryStorageSection
            draft={draft}
            onChange={onChange}
            mediaSources={mediaSources}
            selectedSource={selectedSource}
          />
          <LibraryNameSection
            draft={draft}
            onChange={onChange}
            recommendedName={recommendedName}
          />
          <LibraryRootPathSection
            draft={draft}
            onChange={onChange}
            selectedSource={selectedSource}
            browseExistingLibraryPath={browseExistingLibraryPath}
          />
          <LibraryAccessTagsSection
            draft={draft}
            onChange={onChange}
            availableAccessTags={availableAccessTags}
          />
          <LibraryMetadataTemplateSection
            draft={draft}
            onChange={onChange}
            metadataProfiles={metadataProfiles}
            selectedMetadataProfile={selectedMetadataProfile}
          />
        </div>
      </TabsContent>

      <TabsContent value='advanced' className='mt-0'>
        <div className='grid gap-5'>
          <LibraryStorageSection
            draft={draft}
            onChange={onChange}
            mediaSources={mediaSources}
            selectedSource={selectedSource}
          />
          <LibraryNameSection
            draft={draft}
            onChange={onChange}
            recommendedName={recommendedName}
          />
          <LibraryRootPathSection
            draft={draft}
            onChange={onChange}
            selectedSource={selectedSource}
            browseExistingLibraryPath={browseExistingLibraryPath}
          />
          <LibraryAccessTagsSection
            draft={draft}
            onChange={onChange}
            availableAccessTags={availableAccessTags}
          />
          <LibraryScanSection draft={draft} onChange={onChange} />
          <LibraryMetadataSection
            draft={draft}
            onChange={onChange}
            metadataProfiles={metadataProfiles}
            selectedMetadataProfile={selectedMetadataProfile}
            configurableProviderInstances={configurableProviderInstances}
          />
          <LibraryPlaybackSection draft={draft} onChange={onChange} />
          <LibrarySubtitleSection draft={draft} onChange={onChange} />
        </div>
      </TabsContent>
    </Tabs>
  )
}

function normalizeAccessTagsInput(value: string) {
  return Array.from(
    new Set(
      value
        .split(',')
        .map((item) => item.trim().toLowerCase())
        .filter(Boolean)
    )
  )
}

function LibraryAccessTagsSection({
  draft,
  onChange,
  availableAccessTags,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  availableAccessTags: LibraryAccessTag[]
}) {
  const value = draft.accessTags.join(', ')

  return (
    <Field>
      <FieldLabel>访问标签</FieldLabel>
      <Input
        value={value}
        onChange={(event) =>
          onChange({
            ...draft,
            accessTags: normalizeAccessTagsInput(event.target.value),
          })
        }
        placeholder='例如 kids, family'
      />
      <p className='text-sm leading-6 text-muted-foreground'>
        留空时该媒体库默认对所有已登录用户可见。
      </p>
      <label className='flex items-center gap-3 rounded-xl border border-border/60 bg-background/60 px-3 py-2 text-sm text-foreground'>
        <Switch
          checked={draft.visibilityMode === 'allow_list_only'}
          onCheckedChange={(checked) =>
            onChange({
              ...draft,
              visibilityMode: checked ? 'allow_list_only' : 'default_open',
            })
          }
        />
        <span>仅对命中 allow 标签规则的角色显示</span>
      </label>
      <p className='text-sm leading-6 text-muted-foreground'>
        开启后，只有命中该库访问标签 allow
        规则的角色才能看到这个库；关闭时仍按默认开放策略处理。
      </p>
      {availableAccessTags.length ? (
        <div className='flex flex-wrap gap-2'>
          {availableAccessTags.map((tag) => {
            const active = draft.accessTags.includes(tag.name)
            return (
              <Button
                key={tag.id}
                type='button'
                variant={active ? 'default' : 'outline'}
                size='sm'
                onClick={() =>
                  onChange({
                    ...draft,
                    accessTags: active
                      ? draft.accessTags.filter((item) => item !== tag.name)
                      : [...draft.accessTags, tag.name].sort(),
                  })
                }
              >
                {tag.name}
              </Button>
            )
          })}
        </div>
      ) : null}
    </Field>
  )
}

function LibraryStorageSection({
  draft,
  onChange,
  mediaSources,
  selectedSource,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  mediaSources: MediaSource[]
  selectedSource: MediaSource | null
}) {
  return (
    <section className='grid gap-3'>
      <div>
        <h3 className='text-sm font-medium'>存储位置</h3>
      </div>
      <Field>
        <FieldLabel>媒体源</FieldLabel>
        <Select
          value={draft.mediaSourceId}
          onValueChange={(value) =>
            onChange({ ...draft, mediaSourceId: value })
          }
        >
          <SelectTrigger className='w-full'>
            <SelectValue placeholder='选择媒体源' />
          </SelectTrigger>
          <SelectContent>
            {mediaSources.map((source) => (
              <SelectItem key={source.id} value={String(source.id)}>
                {source.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className='text-xs leading-5 text-muted-foreground'>
          {selectedSource
            ? `#${selectedSource.id} · ${selectedSource.name} · ${selectedSource.root_path}`
            : '先选择媒体源'}
        </p>
      </Field>
    </section>
  )
}

function LibraryNameSection({
  draft,
  onChange,
  recommendedName,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  recommendedName: string
}) {
  return (
    <section className='grid gap-3'>
      <div>
        <h3 className='text-sm font-medium'>名称</h3>
      </div>
      <Field>
        <FieldLabel>来源名称</FieldLabel>
        <Input
          value={draft.name}
          onChange={(event) => onChange({ ...draft, name: event.target.value })}
          placeholder='电影'
        />
        {recommendedName && !draft.name.trim() ? (
          <div className='flex flex-wrap items-center gap-2 text-xs text-muted-foreground'>
            <span>推荐名称：{recommendedName}</span>
            <Button
              type='button'
              variant='link'
              size='sm'
              className='h-auto px-0 py-0 text-xs'
              onClick={() => onChange({ ...draft, name: recommendedName })}
            >
              使用推荐名称
            </Button>
          </div>
        ) : null}
      </Field>
    </section>
  )
}

function LibraryRootPathSection({
  draft,
  onChange,
  selectedSource,
  browseExistingLibraryPath,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  selectedSource: MediaSource | null
  browseExistingLibraryPath: (
    path?: string,
    options?: { refresh?: boolean }
  ) => Promise<StorageBrowseResult>
}) {
  return (
    <section className='grid gap-3'>
      <div>
        <h3 className='text-sm font-medium'>挂载路径</h3>
      </div>
      <PathPicker
        browse={selectedSource ? browseExistingLibraryPath : null}
        browseKey={`existing:${selectedSource?.id ?? 'none'}`}
        browseLabel='当前媒体源子目录'
        value={draft.rootPath}
        onValueChange={(rootPath) => onChange({ ...draft, rootPath })}
        placeholder={selectedSource?.root_path || '/'}
        selectCurrentOnBrowse
        ready={!!selectedSource}
        lockedMessage='先选择媒体源，再选择来源路径。'
      />
    </section>
  )
}

function LibraryScanSection({
  draft,
  onChange,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
}) {
  return (
    <section className='grid gap-3'>
      <div>
        <h3 className='text-sm font-medium'>扫描策略</h3>
      </div>
      <ToggleRow
        label='扫描启用'
        checked={draft.scan.scanner_enabled}
        onChange={(checked) =>
          onChange({
            ...draft,
            scan: { ...draft.scan, scanner_enabled: checked },
          })
        }
      />
      <ToggleRow
        label='实时监听'
        checked={draft.scan.realtime_monitor_enabled}
        onChange={(checked) =>
          onChange({
            ...draft,
            scan: { ...draft.scan, realtime_monitor_enabled: checked },
          })
        }
      />
      <ToggleRow
        label='定时刷新'
        checked={draft.scan.scheduled_refresh_enabled}
        onChange={(checked) =>
          onChange({
            ...draft,
            scan: { ...draft.scan, scheduled_refresh_enabled: checked },
          })
        }
      />
      <NumberField
        label='刷新间隔（小时）'
        value={draft.scan.refresh_interval_hours}
        onChange={(value) =>
          onChange({
            ...draft,
            scan: {
              ...draft.scan,
              refresh_interval_hours: Math.max(1, value),
            },
          })
        }
      />
      <ToggleRow
        label='批量探测库存'
        description='关闭后扫描不会创建 inventory_probe_batch 批量探测任务。'
        checked={draft.scan.inventory_probe_batch_enabled}
        onChange={(checked) =>
          onChange({
            ...draft,
            scan: { ...draft.scan, inventory_probe_batch_enabled: checked },
          })
        }
      />
      <ToggleRow
        label='隐藏文件忽略'
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
          value={draft.scan.ignore_file_extensions.join(',')}
          onChange={(event) =>
            onChange({
              ...draft,
              scan: {
                ...draft.scan,
                ignore_file_extensions: splitList(event.target.value),
              },
            })
          }
          placeholder='.txt,.jpg'
        />
      </Field>
      <FileSizeField
        label='最小文件大小（0 不限制）'
        value={draft.scan.min_file_size_bytes}
        onChange={(value) =>
          onChange({
            ...draft,
            scan: { ...draft.scan, min_file_size_bytes: Math.max(0, value) },
          })
        }
      />
      <div className='grid gap-2'>
        <div>
          <h4 className='text-sm font-medium'>排除规则</h4>
          <p className='text-xs leading-5 text-muted-foreground'>
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
  )
}

function LibraryMetadataTemplateSection({
  draft,
  onChange,
  metadataProfiles,
  selectedMetadataProfile,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  metadataProfiles: MetadataProfile[]
  selectedMetadataProfile: MetadataProfile | null
}) {
  return (
    <section className='grid gap-3'>
      <div>
        <h3 className='text-sm font-medium'>元数据模板</h3>
      </div>
      <Field>
        <FieldLabel>Metadata Template</FieldLabel>
        <Select
          value={
            draft.metadata.metadata_profile_id
              ? String(draft.metadata.metadata_profile_id)
              : ''
          }
          onValueChange={(value) =>
            onChange(
              applyMetadataProfileToDraft(draft, metadataProfiles, value)
            )
          }
        >
          <SelectTrigger className='w-full'>
            <SelectValue placeholder='选择 metadata template' />
          </SelectTrigger>
          <SelectContent>
            {metadataProfiles.map((profile) => (
              <SelectItem key={profile.id} value={String(profile.id)}>
                {profile.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {selectedMetadataProfile?.locked ? (
          <p className='text-xs leading-5 text-muted-foreground'>
            系统只读模板
          </p>
        ) : null}
      </Field>
    </section>
  )
}

function LibraryMetadataSection({
  draft,
  onChange,
  metadataProfiles,
  selectedMetadataProfile,
  configurableProviderInstances,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  metadataProfiles: MetadataProfile[]
  selectedMetadataProfile: MetadataProfile | null
  configurableProviderInstances: MetadataProviderInstance[]
}) {
  return (
    <section className='grid gap-3'>
      <div>
        <h3 className='text-sm font-medium'>元数据策略</h3>
      </div>
      <Field>
        <FieldLabel>Metadata Template</FieldLabel>
        <Select
          value={
            draft.metadata.metadata_profile_id
              ? String(draft.metadata.metadata_profile_id)
              : ''
          }
          onValueChange={(value) =>
            onChange(
              applyMetadataProfileToDraft(draft, metadataProfiles, value)
            )
          }
        >
          <SelectTrigger className='w-full'>
            <SelectValue placeholder='选择 metadata template' />
          </SelectTrigger>
          <SelectContent>
            {metadataProfiles.map((profile) => (
              <SelectItem key={profile.id} value={String(profile.id)}>
                {profile.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {selectedMetadataProfile?.locked ? (
          <p className='text-xs leading-5 text-muted-foreground'>
            系统只读模板
          </p>
        ) : null}
      </Field>
      <StrategyStageField
        label='搜索阶段'
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
        label='详情阶段'
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
      <div className='grid gap-3 md:grid-cols-1'>
        <ToggleRow
          label='读取本地元数据'
          description='开启后，metadata 阶段才会读取同目录下的本地 NFO/JSON 元数据文件。'
          checked={draft.metadata.local_metadata_enabled}
          onChange={(checked) =>
            onChange({
              ...draft,
              metadata: {
                ...draft.metadata,
                local_metadata_enabled: checked,
              },
            })
          }
        />
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
            placeholder='zh-CN'
          />
        </Field>
      </div>
    </section>
  )
}

function LibraryPlaybackSection({
  draft,
  onChange,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
}) {
  return (
    <section className='grid gap-3'>
      <div>
        <h3 className='text-sm font-medium'>播放策略</h3>
      </div>
      <ToggleRow
        label='记录播放进度'
        checked={draft.playback.resume_enabled}
        onChange={(checked) =>
          onChange({
            ...draft,
            playback: { ...draft.playback, resume_enabled: checked },
          })
        }
      />
      <div className='grid gap-3 md:grid-cols-2'>
        <NumberField
          label='完成百分比'
          value={draft.playback.max_resume_pct}
          onChange={(value) =>
            onChange({
              ...draft,
              playback: { ...draft.playback, max_resume_pct: value },
            })
          }
        />
        <NumberField
          label='最小时长秒数'
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
  )
}

function LibrarySubtitleSection({
  draft,
  onChange,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
}) {
  return (
    <section className='grid gap-3'>
      <div>
        <h3 className='text-sm font-medium'>字幕策略</h3>
      </div>
      <ToggleRow
        label='启用外置字幕'
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
        label='容忍不可用字幕'
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
          value={draft.subtitle.preferred_languages.join(',')}
          onChange={(event) =>
            onChange({
              ...draft,
              subtitle: {
                ...draft.subtitle,
                preferred_languages: splitList(event.target.value),
              },
            })
          }
          placeholder='zh,en'
        />
      </Field>
    </section>
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
    <div className='flex items-center justify-between gap-3 rounded-lg border border-border/50 px-3 py-2 text-sm'>
      <span className='grid gap-0.5'>
        <span>{label}</span>
        {description ? (
          <span className='text-xs leading-5 text-muted-foreground'>
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
