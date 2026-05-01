import { useState } from 'react'
import { CheckCircle2Icon, LoaderCircleIcon } from 'lucide-react'

import { Button } from '#/components/ui/button'
import { Field, FieldLabel } from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import type { OpenListTestResult } from '#/lib/mibo-api'
import { createAuthedMiboApi } from '#/lib/mibo-query'

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
  }
  onChange: (nextDraft: {
    provider: string
    name: string
    rootPath: string
    baseUrl: string
    username: string
    password: string
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
        error instanceof Error ? error.message : 'OpenList 连接测试失败。',
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
    <section className="grid gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <h3 className="text-base font-medium">OpenList 连接</h3>
          <p className="text-sm text-muted-foreground">
            填写连接信息后测试，测试通过即可浏览远程路径。
          </p>
        </div>
        <Button
          type="button"
          onClick={() => void testOpenListConnection()}
          disabled={connectionStatus === 'testing' || !draft.baseUrl}
        >
          {connectionStatus === 'testing' ? (
            <LoaderCircleIcon className="size-4 animate-spin" />
          ) : (
            <CheckCircle2Icon className="size-4" />
          )}
          测试连接
        </Button>
      </div>
      <div className="grid gap-4 md:grid-cols-[minmax(0,1.2fr)_minmax(0,0.9fr)_minmax(0,0.9fr)]">
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
            placeholder="可留空匿名访问"
          />
        </Field>
        <Field>
          <FieldLabel>密码</FieldLabel>
          <Input
            type="password"
            value={draft.password}
            onChange={(event) =>
              updateConnection({ ...draft, password: event.target.value })
            }
            placeholder={isEditing ? '留空则保持原密码' : '可留空'}
          />
        </Field>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-muted/20 px-4 py-3">
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-sm font-medium">
            <CheckCircle2Icon className="size-4 text-muted-foreground" />
            连接状态
          </div>
          <div className="text-sm text-muted-foreground">
            {connectionStatus === 'testing'
              ? '正在测试连接与登录状态...'
              : connectionMessage || '尚未验证 OpenList 连接'}
          </div>
          {lastOpenListTestResult?.root_path ? (
            <div className="text-xs text-muted-foreground">
              已确认服务根路径：{lastOpenListTestResult.root_path}
            </div>
          ) : null}
        </div>
        <div className="text-sm font-medium text-muted-foreground">
          {draft.username.trim() !== '' ? `账号 ${draft.username}` : '匿名访问'}
        </div>
      </div>
    </section>
  )
}
