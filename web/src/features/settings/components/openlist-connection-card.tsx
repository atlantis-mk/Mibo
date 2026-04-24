import { useState } from 'react'
import { CheckCircle2Icon, LoaderCircleIcon } from 'lucide-react'

import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '#/components/ui/dialog'
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
  const [isConnectionDialogOpen, setIsConnectionDialogOpen] = useState(false)
  const [connectionDraft, setConnectionDraft] = useState({
    baseUrl: draft.baseUrl || defaultBaseUrl,
    username: draft.username,
    password: '',
  })
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

  return (
    <>
      <Card className="border-border/70 shadow-none">
        <CardHeader className="space-y-1 px-4 pt-4 pb-0">
          <CardTitle className="text-base">OpenList 连接</CardTitle>
          <CardDescription>
            先测试连通性与登录状态，成功后再继续选择路径。
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 px-4 py-4">
          <div className="rounded-xl border border-border/70 bg-background/70 p-4">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">连接地址</div>
                <div className="text-sm font-medium break-all">
                  {draft.baseUrl || defaultBaseUrl}
                </div>
              </div>
              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">认证方式</div>
                <div className="text-sm font-medium">
                  {draft.username.trim() !== ''
                    ? `账号 ${draft.username}`
                    : '匿名访问'}
                </div>
              </div>
            </div>
            <div className="mt-3 flex justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setConnectionDraft({
                    baseUrl: draft.baseUrl || defaultBaseUrl,
                    username: draft.username,
                    password: '',
                  })
                  onConnectionVerifiedChange(false)
                  setIsConnectionDialogOpen(true)
                }}
              >
                修改连接
              </Button>
            </div>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-muted/20 p-4">
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
          <div className="grid gap-4 md:grid-cols-3">
            <Field>
              <FieldLabel>OpenList 地址</FieldLabel>
              <Input
                value={draft.baseUrl}
                onChange={(event) =>
                  onChange({ ...draft, baseUrl: event.target.value })
                }
                placeholder={defaultBaseUrl}
              />
            </Field>
            <Field>
              <FieldLabel>用户名</FieldLabel>
              <Input
                value={draft.username}
                onChange={(event) =>
                  onChange({ ...draft, username: event.target.value })
                }
                placeholder="OpenList 用户名，可留空"
              />
            </Field>
            <Field>
              <FieldLabel>密码</FieldLabel>
              <Input
                type="password"
                value={draft.password}
                onChange={(event) =>
                  onChange({ ...draft, password: event.target.value })
                }
                placeholder={
                  isEditing ? '留空则保持原密码' : 'OpenList 密码，可留空'
                }
              />
            </Field>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={isConnectionDialogOpen}
        onOpenChange={setIsConnectionDialogOpen}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>修改 OpenList 连接</DialogTitle>
            <DialogDescription>
              默认使用本机
              OpenList。只有需要调整地址或认证方式时，再修改连接设置。
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-3">
            <Input
              value={connectionDraft.baseUrl}
              onChange={(event) =>
                setConnectionDraft((current) => ({
                  ...current,
                  baseUrl: event.target.value,
                }))
              }
              placeholder={defaultBaseUrl}
            />
            <Input
              value={connectionDraft.username}
              onChange={(event) =>
                setConnectionDraft((current) => ({
                  ...current,
                  username: event.target.value,
                }))
              }
              placeholder="OpenList 用户名，可留空"
            />
            <Input
              value={connectionDraft.password}
              type="password"
              onChange={(event) =>
                setConnectionDraft((current) => ({
                  ...current,
                  password: event.target.value,
                }))
              }
              placeholder={
                isEditing ? '留空则保持原密码' : 'OpenList 密码，可留空'
              }
            />
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsConnectionDialogOpen(false)}
            >
              取消
            </Button>
            <Button
              onClick={() => {
                onChange({
                  ...draft,
                  baseUrl: connectionDraft.baseUrl,
                  username: connectionDraft.username,
                  password: connectionDraft.password,
                })
                onConnectionVerifiedChange(false)
                setIsConnectionDialogOpen(false)
              }}
            >
              保存连接设置
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
