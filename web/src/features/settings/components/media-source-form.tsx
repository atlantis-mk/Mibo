import { useState } from 'react'

import { PathPicker } from '#/components/path-picker'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
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
    examplePath: '/media',
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
    <div className="grid gap-5">
      <Card className="border-border/70 shadow-none">
        <CardHeader className="space-y-1 px-4 pt-4 pb-0">
          <CardTitle className="text-base">基本信息</CardTitle>
          <CardDescription>先确认媒体源名称与存储类型。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 px-4 py-4">
          <div className="grid gap-2">
            <div className="text-sm font-medium">存储类型</div>
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
            <div className="text-xs leading-5 text-muted-foreground">
              {providerOption.description}
            </div>
          </div>

          {draft.provider === 'local' ? (
            <div className="rounded-xl border border-dashed border-border/70 bg-muted/20 p-4">
              <div className="text-sm font-medium">媒体源名称将自动生成</div>
              <div className="mt-1 text-sm text-muted-foreground">
                本地目录无需额外命名，保存时会使用根路径名称。
              </div>
              <div className="mt-3 text-sm font-medium text-foreground">
                {derivedLocalSourceName}
              </div>
            </div>
          ) : (
            <Field>
              <FieldLabel>媒体源名称</FieldLabel>
              <Input
                value={draft.name}
                onChange={(event) =>
                  onChange({ ...draft, name: event.target.value })
                }
                placeholder="家庭媒体入口"
              />
            </Field>
          )}
        </CardContent>
      </Card>

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

      <Card className="border-border/70 shadow-none">
        <CardHeader className="space-y-1 px-4 pt-4 pb-0">
          <CardTitle className="text-base">根路径</CardTitle>
          <CardDescription>
            {draft.provider === 'openlist'
              ? '连接通过后，再浏览并选择这个媒体源的起始路径。'
              : '本地目录只需要确认要接入的根路径。'}
          </CardDescription>
        </CardHeader>
        <CardContent className="px-4 py-4">
          <PathPicker
            browse={api ? browseSourcePath : null}
            browseKey={`${draft.provider}:${draft.baseUrl}:${draft.username}`}
            browseLabel="当前浏览目录"
            value={draft.rootPath}
            placeholder={providerOption.examplePath}
            onValueChange={(rootPath) => onChange({ ...draft, rootPath })}
            ready={draft.provider === 'local' || isOpenListConnectionVerified}
            lockedMessage={
              draft.provider === 'openlist'
                ? '请先完成 OpenList 连接测试，测试成功后才能浏览和选择路径。'
                : undefined
            }
          />
        </CardContent>
      </Card>
    </div>
  )
}
