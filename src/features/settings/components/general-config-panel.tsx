import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { InfoIcon, LoaderCircleIcon } from 'lucide-react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import type { GeneralConfigInput, GeneralConfigSettings } from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  generalConfigQueryOptions,
  miboQueryKeys,
} from '@/lib/mibo-query'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

type GeneralConfigForm = GeneralConfigInput

const defaultGeneralConfig: GeneralConfigForm = {
  http: {
    shutdown_timeout: '10s',
  },
  web: {
    dist_dir: '',
  },
  cors: {
    allowed_origins: '*',
  },
  access: {
    disable_library_visibility_enforcement: false,
  },
  ffmpeg: {
    enabled: true,
    path: 'ffmpeg',
    timeout: '2m0s',
    artwork_root_path: 'tmp/artwork',
    transcode_root_path: 'tmp/transcode',
    transcode_idle_timeout: '10m0s',
  },
  ffprobe: {
    enabled: true,
    path: 'ffprobe',
    timeout: '30s',
  },
  worker: {
    enabled: true,
    poll_interval: '2s',
    probe_workers: 2,
    scan_directory_workers: 10,
    workflow_poll_interval: '2s',
    workflow_lease_duration: '1m0s',
    workflow_task_timeout: '10m0s',
    workflow_max_concurrent: 4,
  },
}

export function GeneralConfigPanel({ token }: { token: string | null }) {
  const queryClient = useQueryClient()
  const configQuery = useQuery({
    ...generalConfigQueryOptions(token ?? 'guest'),
    enabled: !!token,
  })
  const [draft, setDraft] = useState<GeneralConfigForm>(defaultGeneralConfig)

  useEffect(() => {
    if (configQuery.data) {
      setDraft(generalConfigToForm(configQuery.data))
    }
  }, [configQuery.data])

  const saveMutation = useMutation({
    mutationFn: async (nextDraft: GeneralConfigForm) => {
      if (!token) {
        throw new Error('当前未登录，无法保存通用配置。')
      }

      return createAuthedMiboApi(token).updateGeneralConfig(nextDraft)
    },
    onSuccess: (settings) => {
      if (!token) return

      queryClient.setQueryData(miboQueryKeys.generalConfig(token), settings)
      setDraft(generalConfigToForm(settings))
      toast.success('通用配置已保存，服务正在应用新配置')
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  function updateSection<Section extends keyof GeneralConfigForm>(
    section: Section,
    values: Partial<GeneralConfigForm[Section]>
  ) {
    setDraft((current) => ({
      ...current,
      [section]: { ...current[section], ...values },
    }))
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    saveMutation.mutate(draft)
  }

  if (!token) {
    return (
      <Alert>
        <InfoIcon className='size-4' />
        <AlertTitle>登录后可管理通用配置</AlertTitle>
        <AlertDescription className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
          <span>当前页面需要管理员会话来读取和更新服务器运行配置。</span>
          <Button asChild variant='outline'>
            <Link to='/sign-in' search={{ redirect: '/settings/general' }}>
              前往登录
            </Link>
          </Button>
        </AlertDescription>
      </Alert>
    )
  }

  if (configQuery.isLoading) {
    return (
      <div className='flex items-center gap-3 rounded-[1.25rem] border border-border/60 bg-card/80 px-4 py-6 text-sm text-muted-foreground shadow-sm'>
        <LoaderCircleIcon className='size-4 animate-spin' />
        正在加载通用配置
      </div>
    )
  }

  if (configQuery.error) {
    return (
      <Alert variant='destructive'>
        <AlertTitle>加载失败</AlertTitle>
        <AlertDescription>{configQuery.error.message}</AlertDescription>
      </Alert>
    )
  }

  return (
    <>
      <form
        id='general-config-form'
        onSubmit={handleSubmit}
        className='flex flex-col gap-6 pb-20'
      >
        {configQuery.data?.effective_status ? (
          <Alert>
            <InfoIcon className='size-4' />
            <AlertTitle>保存后会自动重启服务</AlertTitle>
            <AlertDescription>
              {configQuery.data.effective_status.message}
            </AlertDescription>
          </Alert>
        ) : null}

        {saveMutation.error ? (
          <Alert variant='destructive'>
            <AlertTitle>保存失败</AlertTitle>
            <AlertDescription>{saveMutation.error.message}</AlertDescription>
          </Alert>
        ) : null}

        <FieldGroup>
          <SectionTitle
            title='访问与静态资源'
            description='网络端口、数据库、元数据源和字幕源由对应设置页管理。'
          />
          <DurationField
            id='general-http-shutdown-timeout'
            label='关闭超时'
            description='服务退出或重启时等待 HTTP 连接关闭的最长时间。'
            value={draft.http.shutdown_timeout}
            onChange={(value) =>
              updateSection('http', { shutdown_timeout: value })
            }
          />
          <TextField
            id='general-web-dist-dir'
            label='Web 静态目录'
            description='留空时使用内嵌 Web UI；填写后使用指定目录。'
            value={draft.web.dist_dir}
            onChange={(value) => updateSection('web', { dist_dir: value })}
          />
          <Field>
            <FieldLabel htmlFor='general-cors-origins'>
              允许的跨域来源
            </FieldLabel>
            <Textarea
              id='general-cors-origins'
              value={draft.cors.allowed_origins}
              onChange={(event) =>
                updateSection('cors', {
                  allowed_origins: event.target.value,
                })
              }
              className='min-h-20 border-border/60 bg-background font-mono text-sm'
            />
            <FieldDescription>
              支持逗号或换行分隔；使用 * 允许所有来源。
            </FieldDescription>
          </Field>
          <SwitchField
            label='关闭媒体库可见性强制校验'
            description='仅用于兼容旧数据或紧急回滚。'
            checked={draft.access.disable_library_visibility_enforcement}
            onCheckedChange={(checked) =>
              updateSection('access', {
                disable_library_visibility_enforcement: checked,
              })
            }
          />
        </FieldGroup>

        <Separator />

        <FieldGroup>
          <SectionTitle title='播放处理' description='控制转码和媒体探测。' />
          <SwitchField
            label='启用 FFmpeg'
            checked={draft.ffmpeg.enabled}
            onCheckedChange={(checked) =>
              updateSection('ffmpeg', { enabled: checked })
            }
          />
          <div className='grid gap-4 md:grid-cols-2'>
            <TextField
              id='general-ffmpeg-path'
              label='FFmpeg 路径'
              value={draft.ffmpeg.path}
              onChange={(value) => updateSection('ffmpeg', { path: value })}
            />
            <DurationField
              id='general-ffmpeg-timeout'
              label='FFmpeg 超时'
              value={draft.ffmpeg.timeout}
              onChange={(value) => updateSection('ffmpeg', { timeout: value })}
            />
          </div>
          <div className='grid gap-4 md:grid-cols-2'>
            <TextField
              id='general-artwork-root-path'
              label='图片缓存目录'
              value={draft.ffmpeg.artwork_root_path}
              onChange={(value) =>
                updateSection('ffmpeg', { artwork_root_path: value })
              }
            />
            <TextField
              id='general-transcode-root-path'
              label='转码临时目录'
              value={draft.ffmpeg.transcode_root_path}
              onChange={(value) =>
                updateSection('ffmpeg', { transcode_root_path: value })
              }
            />
          </div>
          <DurationField
            id='general-transcode-idle-timeout'
            label='转码空闲超时'
            value={draft.ffmpeg.transcode_idle_timeout}
            onChange={(value) =>
              updateSection('ffmpeg', { transcode_idle_timeout: value })
            }
          />
          <SwitchField
            label='启用 FFprobe'
            checked={draft.ffprobe.enabled}
            onCheckedChange={(checked) =>
              updateSection('ffprobe', { enabled: checked })
            }
          />
          <div className='grid gap-4 md:grid-cols-2'>
            <TextField
              id='general-ffprobe-path'
              label='FFprobe 路径'
              value={draft.ffprobe.path}
              onChange={(value) => updateSection('ffprobe', { path: value })}
            />
            <DurationField
              id='general-ffprobe-timeout'
              label='FFprobe 超时'
              value={draft.ffprobe.timeout}
              onChange={(value) => updateSection('ffprobe', { timeout: value })}
            />
          </div>
        </FieldGroup>

        <Separator />

        <FieldGroup>
          <SectionTitle
            title='后台执行'
            description='控制后台循环和 Workflow 并发。'
          />
          <SwitchField
            label='启用后台 Worker'
            checked={draft.worker.enabled}
            onCheckedChange={(checked) =>
              updateSection('worker', { enabled: checked })
            }
          />
          <div className='grid gap-4 md:grid-cols-2'>
            <DurationField
              id='general-worker-poll-interval'
              label='Worker 轮询间隔'
              value={draft.worker.poll_interval}
              onChange={(value) =>
                updateSection('worker', { poll_interval: value })
              }
            />
            <DurationField
              id='general-workflow-poll-interval'
              label='Workflow 轮询间隔'
              value={draft.worker.workflow_poll_interval}
              onChange={(value) =>
                updateSection('worker', {
                  workflow_poll_interval: value,
                })
              }
            />
          </div>
          <div className='grid gap-4 md:grid-cols-2'>
            <NumberField
              id='general-probe-workers'
              label='探测并发'
              min={1}
              max={8}
              value={draft.worker.probe_workers}
              onChange={(value) =>
                updateSection('worker', { probe_workers: value })
              }
            />
            <NumberField
              id='general-scan-directory-workers'
              label='目录扫描并发'
              min={1}
              max={32}
              value={draft.worker.scan_directory_workers}
              onChange={(value) =>
                updateSection('worker', { scan_directory_workers: value })
              }
            />
          </div>
          <div className='grid gap-4 md:grid-cols-3'>
            <DurationField
              id='general-workflow-lease-duration'
              label='任务租约'
              value={draft.worker.workflow_lease_duration}
              onChange={(value) =>
                updateSection('worker', {
                  workflow_lease_duration: value,
                })
              }
            />
            <DurationField
              id='general-workflow-task-timeout'
              label='任务超时'
              value={draft.worker.workflow_task_timeout}
              onChange={(value) =>
                updateSection('worker', { workflow_task_timeout: value })
              }
            />
            <NumberField
              id='general-workflow-max-concurrent'
              label='Workflow 并发'
              min={1}
              max={32}
              value={draft.worker.workflow_max_concurrent}
              onChange={(value) =>
                updateSection('worker', { workflow_max_concurrent: value })
              }
            />
          </div>
        </FieldGroup>
      </form>

      {createPortal(
        <div className='fixed right-6 bottom-6 z-[100] flex justify-end'>
          <Button
            type='submit'
            form='general-config-form'
            disabled={saveMutation.isPending}
          >
            {saveMutation.isPending ? (
              <LoaderCircleIcon
                data-icon='inline-start'
                className='animate-spin'
              />
            ) : null}
            保存配置
          </Button>
        </div>,
        document.body
      )}
    </>
  )
}

function generalConfigToForm(
  settings: GeneralConfigSettings
): GeneralConfigForm {
  return {
    http: settings.http,
    web: settings.web,
    cors: settings.cors,
    access: settings.access,
    ffmpeg: settings.ffmpeg,
    ffprobe: settings.ffprobe,
    worker: settings.worker,
  }
}

function SectionTitle({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <Field>
      <FieldContent>
        <FieldTitle className='text-base'>{title}</FieldTitle>
        <FieldDescription>{description}</FieldDescription>
      </FieldContent>
    </Field>
  )
}

function TextField({
  id,
  label,
  description,
  value,
  onChange,
}: {
  id: string
  label: string
  description?: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className='border-border/60 bg-background'
      />
      {description ? <FieldDescription>{description}</FieldDescription> : null}
    </Field>
  )
}

function DurationField(props: {
  id: string
  label: string
  description?: string
  value: string
  onChange: (value: string) => void
}) {
  return <TextField {...props} />
}

function NumberField({
  id,
  label,
  value,
  min,
  max,
  onChange,
}: {
  id: string
  label: string
  value: number
  min: number
  max: number
  onChange: (value: number) => void
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        type='number'
        min={min}
        max={max}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
        className='border-border/60 bg-background'
      />
      <FieldDescription>
        可用范围：{min} - {max}
      </FieldDescription>
    </Field>
  )
}

function SwitchField({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string
  description?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <Field orientation='horizontal'>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
      <FieldContent>
        <FieldTitle>{label}</FieldTitle>
        {description ? (
          <FieldDescription>{description}</FieldDescription>
        ) : null}
      </FieldContent>
    </Field>
  )
}
