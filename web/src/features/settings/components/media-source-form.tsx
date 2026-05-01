import { useState } from 'react'

import { PathPicker } from '#/components/path-picker'
import { Field, FieldLabel } from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import type { MediaSource, StorageBrowseResult } from '#/lib/mibo-api'
import { createAuthedMiboApi } from '#/lib/mibo-query'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '#/components/ui/select'
import { Separator } from '#/components/ui/separator'

import { OpenListConnectionCard } from './openlist-connection-card'

export type SourceFormState = {
  provider: string
  name: string
  rootPath: string
  baseUrl: string
  username: string
  password: string
}

type StorageProviderOption = {
  value: string
  label: string
  description: string
  examplePath: string
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
}

export function deriveLocalSourceName(rootPath: string) {
  const normalizedPath = rootPath.trim().replace(/[\\/]+$/, '')
  const segments = normalizedPath.split(/[\\/]/).filter(Boolean)
  return segments.at(-1) || '本地媒体'
}

export function buildMediaSourceDraft(source: MediaSource): SourceFormState {
  return {
    provider: source.provider,
    name: source.name,
    rootPath: source.root_path,
    baseUrl: source.config?.openlist?.base_url ?? '',
    username: source.config?.openlist?.username ?? '',
    password: '',
  }
}

export function SourceForm({
  draft,
  onChange,
  api,
  isEditing = false,
}: {
  draft: SourceFormState
  onChange: (nextDraft: SourceFormState) => void
  api: ReturnType<typeof createAuthedMiboApi> | null
  isEditing?: boolean
}) {
  const providerOption =
    STORAGE_PROVIDER_OPTIONS.find(
      (option) => option.value === draft.provider,
    ) ?? STORAGE_PROVIDER_OPTIONS[0]
  const derivedLocalSourceName = deriveLocalSourceName(draft.rootPath)
  const [isOpenListConnectionVerified, setIsOpenListConnectionVerified] =
    useState(false)

  async function browseSourcePath(path?: string): Promise<StorageBrowseResult> {
    if (!api) {
      throw new Error('当前未登录，无法浏览路径。')
    }

    if (draft.provider === 'openlist') {
      return api.browseOpenList({
        path,
        config: {
          base_url: draft.baseUrl || DEFAULT_OPENLIST_BASE_URL,
          username: draft.username || undefined,
          password: draft.password || undefined,
        },
      })
    }

    return api.browseStorageProvider('local', path)
  }

  return (
    <div className="grid gap-6">
      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">基本信息</h3>
          <p className="text-sm text-muted-foreground">
            选择存储类型，确认保存时显示的媒体源名称。
          </p>
        </div>
        <div className="grid gap-4 md:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
          <Field>
            <FieldLabel>存储类型</FieldLabel>
            <Select
              value={draft.provider}
              disabled={isEditing}
              onValueChange={(value) => {
                const option = STORAGE_PROVIDER_OPTIONS.find(
                  (item) => item.value === value,
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
              <SelectTrigger className="w-full">
                <SelectValue placeholder="选择存储类型" />
              </SelectTrigger>
              <SelectContent>
                {STORAGE_PROVIDER_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs leading-5 text-muted-foreground">
              {providerOption.description}
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
              placeholder="家庭媒体入口"
            />
            {draft.provider === 'local' ? (
              <p className="text-xs leading-5 text-muted-foreground">
                本地目录会使用根路径名称，无需手动填写。
              </p>
            ) : null}
          </Field>
        </div>
      </section>

      {draft.provider === 'openlist' ? (
        <>
          <Separator />
          <OpenListConnectionCard
            defaultBaseUrl={DEFAULT_OPENLIST_BASE_URL}
            draft={draft}
            onChange={onChange}
            api={api}
            isEditing={isEditing}
            onConnectionVerifiedChange={setIsOpenListConnectionVerified}
          />
        </>
      ) : null}

      <Separator />

      <section className="grid gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-medium">根路径</h3>
          <p className="text-sm text-muted-foreground">
            {draft.provider === 'openlist'
              ? '连接通过后，浏览并选择这个媒体源的起始路径。'
              : '确认要接入的本地媒体目录。'}
          </p>
        </div>
        <PathPicker
          browse={api ? browseSourcePath : null}
          browseKey={`${draft.provider}:${draft.baseUrl}:${draft.username}`}
          browseLabel="当前浏览目录"
          value={draft.rootPath}
          placeholder={providerOption.examplePath}
          onValueChange={(rootPath) => onChange({ ...draft, rootPath })}
          selectCurrentOnBrowse
          ready={draft.provider === 'local' || isOpenListConnectionVerified}
          lockedMessage={
            draft.provider === 'openlist'
              ? '请先完成 OpenList 连接测试，测试成功后才能浏览和选择路径。'
              : undefined
          }
        />
      </section>
    </div>
  )
}
