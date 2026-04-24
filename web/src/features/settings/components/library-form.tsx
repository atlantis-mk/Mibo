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

export type LibraryFormState = {
  name: string
  type: string
  mediaSourceId: string
  rootPath: string
}

type LibraryTypeOption = {
  value: string
  label: string
  description: string
}

const LIBRARY_TYPE_OPTIONS: readonly LibraryTypeOption[] = [
  {
    value: 'movies',
    label: '电影库',
    description: '适合电影和独立视频内容。',
  },
  {
    value: 'shows',
    label: '剧集库',
    description: '适合剧集、综艺和分季内容。',
  },
]

export const EMPTY_LIBRARY_FORM: LibraryFormState = {
  name: '',
  type: 'movies',
  mediaSourceId: '',
  rootPath: '',
}

export function LibraryForm({
  draft,
  onChange,
  mediaSources,
  api,
}: {
  draft: LibraryFormState
  onChange: (nextDraft: LibraryFormState) => void
  mediaSources: MediaSource[]
  api: ReturnType<typeof createAuthedMiboApi> | null
}) {
  const selectedLibraryType =
    LIBRARY_TYPE_OPTIONS.find((option) => option.value === draft.type) ??
    LIBRARY_TYPE_OPTIONS[0]
  const selectedSource =
    mediaSources.find((source) => String(source.id) === draft.mediaSourceId) ??
    null

  async function browseExistingLibraryPath(
    path?: string,
  ): Promise<StorageBrowseResult> {
    if (!api || !selectedSource) {
      throw new Error('请先选择媒体源。')
    }

    return api.browseMediaSource(selectedSource.id, path)
  }

  return (
    <div className="grid gap-5">
      <Card className="border-border/70 shadow-none">
        <CardHeader className="space-y-1 px-4 pt-4 pb-0">
          <CardTitle className="text-base">存储位置</CardTitle>
          <CardDescription>创建媒体库时只能绑定已有媒体源。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 px-4 py-4">
          <div className="grid gap-3">
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
            <div className="text-xs leading-6 text-muted-foreground">
              {selectedSource
                ? `当前媒体源：#${selectedSource.id} · ${selectedSource.name} · ${selectedSource.provider} · 根路径 ${selectedSource.root_path}`
                : '请选择一个可复用的媒体源；如需新增，请先在“媒体源”标签中创建。'}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="border-border/70 shadow-none">
        <CardHeader className="space-y-1 px-4 pt-4 pb-0">
          <CardTitle className="text-base">媒体库信息</CardTitle>
          <CardDescription>选择媒体库类型并绑定根路径。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 px-4 py-4">
          <Field>
            <FieldLabel>媒体库名称</FieldLabel>
            <Input
              value={draft.name}
              onChange={(event) =>
                onChange({ ...draft, name: event.target.value })
              }
              placeholder="电影"
            />
          </Field>
          <Field>
            <FieldLabel>媒体库类型</FieldLabel>
            <Select
              value={draft.type}
              onValueChange={(value) => onChange({ ...draft, type: value })}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="选择媒体库类型" />
              </SelectTrigger>
              <SelectContent>
                {LIBRARY_TYPE_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel>挂载路径</FieldLabel>
            <PathPicker
              browse={selectedSource ? browseExistingLibraryPath : null}
              browseKey={`existing:${selectedSource?.id ?? 'none'}`}
              browseLabel="当前媒体源子目录"
              value={draft.rootPath}
              onValueChange={(rootPath) => onChange({ ...draft, rootPath })}
              placeholder={selectedSource?.root_path || '/media'}
              ready={!!selectedSource}
              lockedMessage="先选择媒体源，再选择媒体库路径。"
            />
          </Field>
          <div className="text-xs leading-6 text-muted-foreground">
            类型：{selectedLibraryType.label} ·{' '}
            {selectedLibraryType.description}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
