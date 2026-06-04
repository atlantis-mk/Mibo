import { useEffect, useState, type ReactNode } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { LoaderCircleIcon, Settings2Icon } from 'lucide-react'
import { toast } from 'sonner'
import type { AdminLogSettings, AdminLogSettingsInput } from '@/lib/mibo-api'
import { createAuthedMiboApi, miboQueryKeys } from '@/lib/mibo-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { formatBytes } from '../format'

const previewSizeOptions = [
  { label: '64 KB', value: 64 * 1024 },
  { label: '256 KB', value: 256 * 1024 },
  { label: '1 MB', value: 1024 * 1024 },
]

export function LogSettingsPanel({
  tabs,
  token,
  queryToken,
  settings,
  isLoading,
}: {
  tabs: ReactNode
  token: string | null
  queryToken: string
  settings?: AdminLogSettings
  isLoading: boolean
}) {
  const queryClient = useQueryClient()
  const current = settings ?? {
    include_server_logs: true,
    include_transcode_logs: true,
    max_preview_bytes: 256 * 1024,
  }
  const [customPreviewBytes, setCustomPreviewBytes] = useState(
    String(current.max_preview_bytes)
  )
  const updateMutation = useMutation({
    mutationFn: async (input: AdminLogSettingsInput) => {
      if (!token) {
        throw new Error('当前未登录，无法保存日志设置。')
      }
      return createAuthedMiboApi(token).updateAdminLogSettings(input)
    },
    onSuccess: async (nextSettings) => {
      queryClient.setQueryData(
        miboQueryKeys.adminLogSettings(queryToken),
        nextSettings
      )
      await queryClient.invalidateQueries({
        queryKey: miboQueryKeys.adminLogs(queryToken),
      })
      toast.success('日志设置已保存')
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const isSaving = updateMutation.isPending

  useEffect(() => {
    setCustomPreviewBytes(String(current.max_preview_bytes))
  }, [current.max_preview_bytes])

  function updateSettings(input: AdminLogSettingsInput) {
    updateMutation.mutate(input)
  }

  function saveCustomPreviewBytes() {
    const nextValue = Number(customPreviewBytes)
    if (
      !Number.isFinite(nextValue) ||
      nextValue < 4096 ||
      nextValue > 4 * 1024 * 1024
    ) {
      setCustomPreviewBytes(String(current.max_preview_bytes))
      toast.error('预览读取上限需要在 4096 到 4194304 字节之间。')
      return
    }
    if (nextValue !== current.max_preview_bytes) {
      updateSettings({ max_preview_bytes: nextValue })
    }
  }

  return (
    <section>
      <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
        <div className='flex items-start gap-3'>
          <div className='flex size-10 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground'>
            <Settings2Icon className='size-4' />
          </div>
          <div>
            <h3 className='font-semibold'>日志设置</h3>
            <p className='mt-1 text-sm leading-6 text-muted-foreground'>
              控制日志列表包含的来源，以及在线预览单个日志时读取的最大内容。
            </p>
          </div>
        </div>
        <div className='sm:shrink-0'>{tabs}</div>
      </div>

      {isLoading ? (
        <div className='mt-6 flex items-center gap-3 rounded-2xl bg-muted/55 p-4 text-sm text-muted-foreground'>
          <LoaderCircleIcon className='size-4 animate-spin' />
          正在加载日志设置
        </div>
      ) : (
        <div className='mt-6 space-y-4'>
          <SettingRow
            label='显示服务器日志'
            description='读取 data/logs 下的 .log 和 .txt 文件。'
            checked={current.include_server_logs}
            disabled={isSaving || !token}
            onCheckedChange={(checked) =>
              updateSettings({ include_server_logs: checked })
            }
          />
          <SettingRow
            label='显示转码日志'
            description='读取转码目录中每个会话的 ffmpeg.log。'
            checked={current.include_transcode_logs}
            disabled={isSaving || !token}
            onCheckedChange={(checked) =>
              updateSettings({ include_transcode_logs: checked })
            }
          />
          <div className='rounded-2xl bg-muted/55 p-4'>
            <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
              <div className='min-w-0'>
                <Label className='font-medium'>预览读取上限</Label>
                <p className='mt-1 text-sm leading-6 text-muted-foreground'>
                  当前上限 {formatBytes(current.max_preview_bytes)}
                  ，较大的日志会显示文件尾部内容。
                </p>
              </div>
              <div className='flex flex-wrap gap-2 sm:justify-end'>
                {previewSizeOptions.map((option) => (
                  <Button
                    key={option.value}
                    type='button'
                    variant={
                      current.max_preview_bytes === option.value
                        ? 'default'
                        : 'outline'
                    }
                    size='sm'
                    disabled={isSaving || !token}
                    onClick={() =>
                      updateSettings({ max_preview_bytes: option.value })
                    }
                  >
                    {option.label}
                  </Button>
                ))}
              </div>
            </div>
            <div className='mt-4 max-w-xs'>
              <Input
                type='number'
                min={4096}
                max={4 * 1024 * 1024}
                step={4096}
                value={customPreviewBytes}
                disabled={isSaving || !token}
                onBlur={saveCustomPreviewBytes}
                onChange={(event) => setCustomPreviewBytes(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    event.currentTarget.blur()
                  }
                }}
              />
            </div>
          </div>
        </div>
      )}
    </section>
  )
}

function SettingRow({
  label,
  description,
  checked,
  disabled,
  onCheckedChange,
}: {
  label: string
  description: string
  checked: boolean
  disabled: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <div className='flex items-center justify-between gap-4 rounded-2xl bg-muted/55 p-4'>
      <div className='min-w-0'>
        <Label className='font-medium'>{label}</Label>
        <p className='mt-1 text-sm leading-6 text-muted-foreground'>
          {description}
        </p>
      </div>
      <Switch
        checked={checked}
        disabled={disabled}
        onCheckedChange={onCheckedChange}
      />
    </div>
  )
}
