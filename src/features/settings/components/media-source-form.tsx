import { useState } from 'react'
import type {
  MediaSource,
  PluginProviderInstance,
  StorageBrowseResult,
} from '@/lib/mibo-api'
import type { createAuthedMiboApi } from '@/lib/mibo-query'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { PathPicker } from '@/components/path-picker'
import { OpenListConnectionCard } from './openlist-connection-card'
import { PluginConfigurationForm } from './plugin-configuration-form'

export type SourceFormState = {
  provider: string
  name: string
  rootPath: string
  baseUrl: string
  username: string
  password: string
  scanInterval: string
}

type StorageProviderOption = {
  value: string
  label: string
  description: string
  examplePath: string
  pluginProvider?: PluginProviderInstance
}

export const DEFAULT_OPENLIST_BASE_URL = 'http://127.0.0.1:5244'

const STORAGE_PROVIDER_OPTIONS: readonly StorageProviderOption[] = [
  {
    value: 'local',
    label: '本地目录',
    description: '直接接入本机上的媒体目录。',
    examplePath: '/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media',
  },
  {
    value: 'openlist',
    label: 'OpenList',
    description: '通过 OpenList HTTP 服务接入远程目录。',
    examplePath: '/',
  },
]

export const EMPTY_SOURCE_FORM: SourceFormState = {
  provider: 'local',
  name: '',
  rootPath: '',
  baseUrl: '',
  username: '',
  password: '',
  scanInterval: '1m',
}

export function buildStorageProviderOptions(
  pluginProviderInstances: PluginProviderInstance[]
): StorageProviderOption[] {
  return [
    ...STORAGE_PROVIDER_OPTIONS,
    ...pluginProviderInstances
      .filter(
        (provider) =>
          provider.enabled && provider.capabilities.includes('storage.resolve')
      )
      .map((provider) => ({
        value: `plugin:${provider.id}`,
        label: provider.name,
        description: `${provider.plugin_name} · ${provider.deployment_kind}`,
        examplePath: '/',
        pluginProvider: provider,
      })),
  ]
}

export function deriveLocalSourceName(rootPath: string) {
  const normalizedPath = rootPath.trim().replace(/[\\/]+$/, '')
  const segments = normalizedPath.split(/[\\/]/).filter(Boolean)
  return segments[segments.length - 1] || '本地媒体'
}

export function buildMediaSourceDraft(source: MediaSource): SourceFormState {
  return {
    provider: source.provider,
    name: source.name,
    rootPath: source.root_path,
    baseUrl: source.config?.openlist?.base_url ?? '',
    username: source.config?.openlist?.username ?? '',
    password: '',
    scanInterval: source.config?.openlist?.scan_interval ?? '1m',
  }
}

export function SourceForm({
  draft,
  onChange,
  api,
  pluginProviderInstances,
  isEditing = false,
}: {
  draft: SourceFormState
  onChange: (nextDraft: SourceFormState) => void
  api: ReturnType<typeof createAuthedMiboApi> | null
  pluginProviderInstances: PluginProviderInstance[]
  isEditing?: boolean
}) {
  const storageProviderOptions = buildStorageProviderOptions(
    pluginProviderInstances
  )
  const providerOption =
    storageProviderOptions.find((option) => option.value === draft.provider) ??
    storageProviderOptions[0]
  const selectedPluginProvider = providerOption?.pluginProvider ?? null
  const derivedLocalSourceName = deriveLocalSourceName(draft.rootPath)
  const [isOpenListConnectionVerified, setIsOpenListConnectionVerified] =
    useState(false)

  async function browseSourcePath(
    path?: string,
    options?: { refresh?: boolean }
  ): Promise<StorageBrowseResult> {
    if (!api) {
      throw new Error('当前未登录，无法浏览路径。')
    }

    if (draft.provider === 'openlist') {
      return api.browseOpenList({
        path,
        refresh: options?.refresh,
        config: {
          base_url: draft.baseUrl || DEFAULT_OPENLIST_BASE_URL,
          username: draft.username || undefined,
          password: draft.password || undefined,
        },
      })
    }

    if (selectedPluginProvider) {
      return api.browsePluginProvider(
        selectedPluginProvider.id,
        path,
        options?.refresh
      )
    }

    return api.browseStorageProvider('local', path, options?.refresh)
  }

  return (
    <div className='grid gap-5'>
      <section className='grid gap-3'>
        <div>
          <h3 className='text-sm font-medium'>基本信息</h3>
        </div>
        <div className='grid gap-4 md:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]'>
          <Field>
            <FieldLabel>存储类型</FieldLabel>
            <Select
              value={draft.provider}
              disabled={isEditing}
              onValueChange={(value) => {
                const option = storageProviderOptions.find(
                  (item) => item.value === value
                )
                onChange({
                  ...draft,
                  provider: value,
                  rootPath: option?.examplePath ?? draft.rootPath,
                  baseUrl:
                    value === 'openlist'
                      ? draft.baseUrl || DEFAULT_OPENLIST_BASE_URL
                      : draft.baseUrl,
                })
              }}
            >
              <SelectTrigger className='w-full'>
                <SelectValue placeholder='选择存储类型' />
              </SelectTrigger>
              <SelectContent>
                {storageProviderOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className='text-xs leading-5 text-muted-foreground'>
              {providerOption.label}
            </p>
          </Field>

          <Field>
            <FieldLabel>媒体源名称</FieldLabel>
            <Input
              value={
                draft.provider === 'local' ? derivedLocalSourceName : draft.name
              }
              disabled={draft.provider === 'local'}
              onChange={(event) =>
                onChange({ ...draft, name: event.target.value })
              }
              placeholder='家庭媒体入口'
            />
            {draft.provider === 'local' ? (
              <p className='text-xs leading-5 text-muted-foreground'>
                使用目录名
              </p>
            ) : null}
          </Field>
        </div>
      </section>

      {draft.provider === 'openlist' ? (
        <OpenListConnectionCard
          defaultBaseUrl={DEFAULT_OPENLIST_BASE_URL}
          draft={draft}
          onChange={onChange}
          api={api}
          isEditing={isEditing}
          onConnectionVerifiedChange={setIsOpenListConnectionVerified}
        />
      ) : null}

      {selectedPluginProvider ? (
        <section className='grid gap-3'>
          <div>
            <h3 className='text-sm font-medium'>插件实例</h3>
          </div>
          <div className='grid gap-3 rounded-[1rem] border border-border/60 bg-background/60 p-4'>
            <div className='grid gap-3 md:grid-cols-2'>
              <Field>
                <FieldLabel>插件</FieldLabel>
                <div className='text-sm text-foreground'>
                  {selectedPluginProvider.plugin_name}
                </div>
              </Field>
              <Field>
                <FieldLabel>端点</FieldLabel>
                <div className='text-sm break-all text-foreground'>
                  {selectedPluginProvider.endpoint}
                </div>
              </Field>
            </div>
            <Field>
              <FieldLabel>能力</FieldLabel>
              <div className='flex flex-wrap gap-2'>
                {selectedPluginProvider.capabilities.map((capability) => (
                  <span
                    key={capability}
                    className='rounded-full border border-border/60 px-2 py-1 text-xs text-foreground'
                  >
                    {capability}
                  </span>
                ))}
              </div>
            </Field>
            <Field>
              <FieldLabel>插件配置</FieldLabel>
              <PluginConfigurationForm
                schema={selectedPluginProvider.manifest.configuration_schema}
                value={selectedPluginProvider.configuration ?? {}}
                onChange={() => {}}
                disabled
              />
            </Field>
          </div>
        </section>
      ) : null}

      <section className='grid gap-3'>
        <div>
          <h3 className='text-sm font-medium'>根路径</h3>
        </div>
        <PathPicker
          browse={api ? browseSourcePath : null}
          browseKey={`${draft.provider}:${draft.baseUrl}:${draft.username}`}
          browseLabel='当前浏览目录'
          value={draft.rootPath}
          placeholder={providerOption.examplePath}
          onValueChange={(rootPath) => onChange({ ...draft, rootPath })}
          selectCurrentOnBrowse
          ready={
            draft.provider === 'local' ||
            isOpenListConnectionVerified ||
            selectedPluginProvider !== null
          }
          lockedMessage={
            draft.provider === 'openlist'
              ? '请先完成 OpenList 连接测试，测试成功后才能浏览和选择路径。'
              : !selectedPluginProvider && draft.provider.startsWith('plugin:')
                ? '请先选择一个可用的插件实例。'
                : undefined
          }
        />
      </section>
    </div>
  )
}
