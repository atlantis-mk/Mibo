import { useEffect, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { InfoIcon, LoaderCircleIcon } from 'lucide-react'
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
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import { Separator } from '#/components/ui/separator'
import type { MetadataSettings, MetadataSettingsInput } from '#/lib/mibo-api'
import {
  createAuthedMiboApi,
  metadataSettingsQueryOptions,
  miboQueryKeys,
} from '#/lib/mibo-query'

type ProviderKey = 'tmdb' | 'tvdb'

type MetadataProviderFormState = {
  apiKey: string
  clearApiKey: boolean
  baseURL: string
  imageBaseURL: string
  language: string
  timeout: string
}

type MetadataSettingsFormState = {
  tmdb: MetadataProviderFormState
  tvdb: MetadataProviderFormState
}

const EMPTY_PROVIDER_FORM: MetadataProviderFormState = {
  apiKey: '',
  clearApiKey: false,
  baseURL: '',
  imageBaseURL: '',
  language: '',
  timeout: '',
}

export function MetadataProviderSettingsPanel({
  token,
}: {
  token: string | null
}) {
  const queryClient = useQueryClient()
  const metadataQuery = useQuery({
    ...metadataSettingsQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const [draft, setDraft] = useState<MetadataSettingsFormState | null>(null)

  useEffect(() => {
    if (metadataQuery.data && draft === null) {
      setDraft(buildFormState(metadataQuery.data))
    }
  }, [draft, metadataQuery.data])

  const saveMutation = useMutation({
    mutationFn: async (nextDraft: MetadataSettingsFormState) => {
      if (!token) {
        throw new Error('当前未登录，无法保存元数据源设置。')
      }

      return createAuthedMiboApi(token).updateMetadataSettings(
        buildUpdateInput(nextDraft),
      )
    },
    onSuccess: (settings) => {
      if (!token) {
        return
      }

      queryClient.setQueryData(miboQueryKeys.metadataSettings(token), settings)
      setDraft(buildFormState(settings))
      toast.success('元数据源设置已保存')
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  if (!token) {
    return (
      <Alert>
        <InfoIcon className="size-4" />
        <AlertTitle>登录后可管理元数据源</AlertTitle>
        <AlertDescription className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <span>当前页面需要管理员会话来读取和更新 TMDB / TVDB 配置。</span>
          <Button asChild variant="outline" className="w-fit">
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

  if (metadataQuery.isLoading && draft === null) {
    return (
      <div className="flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm">
        <LoaderCircleIcon className="size-4 animate-spin" />
        正在加载元数据源设置
      </div>
    )
  }

  if (metadataQuery.error && draft === null) {
    return (
      <Alert variant="destructive">
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{metadataQuery.error.message}</AlertDescription>
      </Alert>
    )
  }

  if (!draft || !metadataQuery.data) {
    return null
  }

  return (
    <div className="space-y-4">
      <Alert>
        <InfoIcon className="size-4" />
        <AlertTitle>密钥不会在界面中明文回显</AlertTitle>
        <AlertDescription>
          已配置的 API Key 只显示状态。留空表示保留当前值，勾选“清除已保存
          key”后保存则会删除数据库中的密钥。
        </AlertDescription>
      </Alert>

      <div className="grid gap-4 xl:grid-cols-2">
        <MetadataProviderCard
          title="TMDB"
          description="当前元数据抓取实际使用的 provider，支持 API Key 和 Bearer Token。"
          note="如果你要调整电影、剧集匹配和海报图来源，优先维护这里。"
          settings={metadataQuery.data.tmdb}
          draft={draft.tmdb}
          onChange={(field, value) => {
            setDraft((current) => {
              if (!current) {
                return current
              }

              return {
                ...current,
                tmdb: {
                  ...current.tmdb,
                  [field]: value,
                },
              }
            })
          }}
          includeImageBaseURL
        />

        <MetadataProviderCard
          title="TVDB"
          description="当前后端已经支持配置存储，但抓取流程仍处于 planned 状态。"
          note="可以先录入 key 和默认参数，为后续接入留出管理入口。"
          settings={metadataQuery.data.tvdb}
          draft={draft.tvdb}
          onChange={(field, value) => {
            setDraft((current) => {
              if (!current) {
                return current
              }

              return {
                ...current,
                tvdb: {
                  ...current.tvdb,
                  [field]: value,
                },
              }
            })
          }}
        />
      </div>

      <div className="flex flex-col gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-4 shadow-sm sm:flex-row sm:items-center sm:justify-between">
        <div className="text-sm text-muted-foreground">
          保存后将立即更新后端读取到的元数据源配置。
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => setDraft(buildFormState(metadataQuery.data))}
            disabled={saveMutation.isPending}
          >
            重置表单
          </Button>
          <Button
            type="button"
            onClick={() => saveMutation.mutate(draft)}
            disabled={saveMutation.isPending}
          >
            {saveMutation.isPending ? '保存中…' : '保存元数据源设置'}
          </Button>
        </div>
      </div>
    </div>
  )
}

function MetadataProviderCard({
  title,
  description,
  note,
  settings,
  draft,
  onChange,
  includeImageBaseURL = false,
}: {
  title: string
  description: string
  note: string
  settings: MetadataSettings[ProviderKey]
  draft: MetadataProviderFormState
  onChange: <FieldKey extends keyof MetadataProviderFormState>(
    field: FieldKey,
    value: MetadataProviderFormState[FieldKey],
  ) => void
  includeImageBaseURL?: boolean
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="space-y-3 px-5 py-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="text-xl">{title}</CardTitle>
            <CardDescription className="mt-1">{description}</CardDescription>
          </div>

          <div className="flex flex-wrap gap-2">
            <Badge variant={settings.configured ? 'secondary' : 'outline'}>
              {settings.configured ? '已配置' : '未配置'}
            </Badge>
            <Badge variant="outline">
              {formatSourceLabel(settings.source)}
            </Badge>
            <Badge variant="outline">
              {formatImplementationLabel(settings.implementation)}
            </Badge>
          </div>
        </div>

        <div className="rounded-[1.1rem] border border-border/60 bg-muted/30 px-4 py-3 text-sm leading-6 text-muted-foreground">
          {note}
        </div>
      </CardHeader>

      <Separator className="bg-border" />

      <CardContent className="space-y-5 px-5 py-5">
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor={`${title}-api-key`}>
              API Key / Token
            </FieldLabel>
            <Input
              id={`${title}-api-key`}
              type="password"
              value={draft.apiKey}
              disabled={draft.clearApiKey}
              placeholder={
                settings.api_key_masked
                  ? '已配置，留空则保持当前 key'
                  : '输入新的 API Key 或 Token'
              }
              className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
              onChange={(event) => onChange('apiKey', event.target.value)}
            />
            <FieldDescription>
              前端不会回显已保存的 key。只有重新输入时才会更新；留空则保持现状。
            </FieldDescription>
          </Field>

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

          <div className="grid gap-4 md:grid-cols-2">
            <Field>
              <FieldLabel htmlFor={`${title}-base-url`}>Base URL</FieldLabel>
              <Input
                id={`${title}-base-url`}
                value={draft.baseURL}
                className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
                onChange={(event) => onChange('baseURL', event.target.value)}
              />
              <FieldDescription>
                当前显示的是后端实际解析后的地址，保存后会写入数据库覆盖默认值。
              </FieldDescription>
            </Field>

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
          </div>

          {includeImageBaseURL ? (
            <Field>
              <FieldLabel htmlFor={`${title}-image-base-url`}>
                Image Base URL
              </FieldLabel>
              <Input
                id={`${title}-image-base-url`}
                value={draft.imageBaseURL}
                className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
                onChange={(event) =>
                  onChange('imageBaseURL', event.target.value)
                }
              />
              <FieldDescription>
                仅 TMDB 使用，用于拼接海报、剧照和头像地址。
              </FieldDescription>
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
      </CardContent>
    </Card>
  )
}

function buildFormState(settings: MetadataSettings): MetadataSettingsFormState {
  return {
    tmdb: {
      ...EMPTY_PROVIDER_FORM,
      baseURL: settings.tmdb.base_url,
      imageBaseURL: settings.tmdb.image_base_url ?? '',
      language: settings.tmdb.language,
      timeout: settings.tmdb.timeout,
    },
    tvdb: {
      ...EMPTY_PROVIDER_FORM,
      baseURL: settings.tvdb.base_url,
      language: settings.tvdb.language,
      timeout: settings.tvdb.timeout,
    },
  }
}

function buildUpdateInput(
  draft: MetadataSettingsFormState,
): MetadataSettingsInput {
  return {
    tmdb: {
      api_key: draft.tmdb.apiKey || undefined,
      clear_api_key: draft.tmdb.clearApiKey || undefined,
      base_url: draft.tmdb.baseURL,
      image_base_url: draft.tmdb.imageBaseURL,
      language: draft.tmdb.language,
      timeout: draft.tmdb.timeout,
    },
    tvdb: {
      api_key: draft.tvdb.apiKey || undefined,
      clear_api_key: draft.tvdb.clearApiKey || undefined,
      base_url: draft.tvdb.baseURL,
      language: draft.tvdb.language,
      timeout: draft.tvdb.timeout,
    },
  }
}

function formatSourceLabel(source: string) {
  switch (source) {
    case 'database':
      return '数据库'
    case 'env':
      return '环境变量'
    default:
      return '未设置'
  }
}

function formatImplementationLabel(implementation: string) {
  switch (implementation) {
    case 'active':
      return '已接入'
    case 'planned':
      return '预留'
    default:
      return implementation || '未知'
  }
}
