import { useState } from 'react'
import { CheckCircle2Icon, LoaderCircleIcon } from 'lucide-react'
import type { OpenListTestResult } from '@/lib/mibo-api'
import type { createAuthedMiboApi } from '@/lib/mibo-query'
import { Button } from '@/components/ui/button'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'

export function OpenListConnectionCard({
  defaultBaseUrl,
  draft,
  onChange,
  api,
  isEditing,
  onConnectionVerifiedChange,
}: {
  defaultBaseUrl: string
  draft: {
    provider: string
    name: string
    rootPath: string
    baseUrl: string
    username: string
    password: string
    scanInterval: string
  }
  onChange: (nextDraft: {
    provider: string
    name: string
    rootPath: string
    baseUrl: string
    username: string
    password: string
    scanInterval: string
  }) => void
  api: ReturnType<typeof createAuthedMiboApi> | null
  isEditing: boolean
  onConnectionVerifiedChange: (verified: boolean) => void
}) {
  const [connectionStatus, setConnectionStatus] = useState<
    'idle' | 'testing' | 'success' | 'error'
  >('idle')
  const [connectionMessage, setConnectionMessage] = useState('')
  const [lastOpenListTestResult, setLastOpenListTestResult] =
    useState<OpenListTestResult | null>(null)

  async function testOpenListConnection() {
    if (!api) {
      setConnectionStatus('error')
      setConnectionMessage('当前未登录，无法测试 OpenList 连接。')
      onConnectionVerifiedChange(false)
      return
    }

    setConnectionStatus('testing')
    setConnectionMessage('')
    try {
      const result = await api.testOpenListConnection({
        config: {
          base_url: draft.baseUrl || defaultBaseUrl,
          username: draft.username || undefined,
          password: draft.password || undefined,
        },
      })
      setConnectionStatus('success')
      setConnectionMessage(result.message)
      setLastOpenListTestResult(result)
      onConnectionVerifiedChange(true)
    } catch (error) {
      setConnectionStatus('error')
      setConnectionMessage(
        error instanceof Error ? error.message : 'OpenList 连接测试失败。'
      )
      onConnectionVerifiedChange(false)
    }
  }

  function updateConnection(nextDraft: typeof draft) {
    onChange(nextDraft)
    setConnectionStatus('idle')
    setConnectionMessage('连接配置已修改，请重新测试。')
    setLastOpenListTestResult(null)
    onConnectionVerifiedChange(false)
  }

  return (
    <section className='grid gap-3'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <h3 className='text-sm font-medium'>OpenList 连接</h3>
        </div>
        <Button
          type='button'
          onClick={() => void testOpenListConnection()}
          disabled={connectionStatus === 'testing' || !draft.baseUrl}
        >
          {connectionStatus === 'testing' ? (
            <LoaderCircleIcon className='size-4 animate-spin' />
          ) : (
            <CheckCircle2Icon className='size-4' />
          )}
          测试连接
        </Button>
      </div>
      <div className='grid gap-4 md:grid-cols-[minmax(0,1.2fr)_minmax(0,0.9fr)_minmax(0,0.9fr)_minmax(0,0.7fr)]'>
        <Field>
          <FieldLabel>OpenList 地址</FieldLabel>
          <Input
            value={draft.baseUrl}
            onChange={(event) =>
              updateConnection({ ...draft, baseUrl: event.target.value })
            }
            placeholder={defaultBaseUrl}
          />
        </Field>
        <Field>
          <FieldLabel>用户名</FieldLabel>
          <Input
            value={draft.username}
            onChange={(event) =>
              updateConnection({ ...draft, username: event.target.value })
            }
            placeholder='可留空匿名访问'
          />
        </Field>
        <Field>
          <FieldLabel>密码</FieldLabel>
          <Input
            type='password'
            value={draft.password}
            onChange={(event) =>
              updateConnection({ ...draft, password: event.target.value })
            }
            placeholder={isEditing ? '留空则保持原密码' : '可留空'}
          />
        </Field>
        <Field>
          <FieldLabel>扫描间隔</FieldLabel>
          <Input
            value={draft.scanInterval}
            onChange={(event) =>
              onChange({ ...draft, scanInterval: event.target.value })
            }
            placeholder='1m'
          />
        </Field>
      </div>
      <div className='flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border/50 px-3 py-2.5'>
        <div className='space-y-1'>
          <div className='flex items-center gap-2 text-sm font-medium'>
            <CheckCircle2Icon className='size-4 text-muted-foreground' />
            连接状态
          </div>
          <div className='text-sm text-muted-foreground'>
            {connectionStatus === 'testing'
              ? '正在测试...'
              : connectionMessage || '尚未验证'}
          </div>
          {lastOpenListTestResult?.root_path ? (
            <div className='text-xs text-muted-foreground'>
              根路径：{lastOpenListTestResult.root_path}
            </div>
          ) : null}
        </div>
        <div className='text-sm font-medium text-muted-foreground'>
          {draft.username.trim() !== '' ? `账号 ${draft.username}` : '匿名访问'}
        </div>
      </div>
    </section>
  )
}
