import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { ShieldCheckIcon, SparklesIcon } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Button } from '#/components/ui/button'
import { Card, CardContent } from '#/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import { Spinner } from '#/components/ui/spinner'
import { ApiError, createMiboApi, getApiBaseUrl } from '#/lib/mibo-api'
import { miboQueryKeys } from '#/lib/mibo-query'
import { useAuthStore } from '#/stores/auth-store'

export default function SetupPage({
  redirectTo = '/',
}: {
  redirectTo?: string
}) {
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('admin123')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const setSession = useAuthStore((state) => state.setSession)
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const setupMutation = useMutation({
    mutationFn: async () => {
      const api = createMiboApi({ baseUrl: getApiBaseUrl() })

      await api.register(username, password)

      return api.login(username, password)
    },
    onMutate: () => {
      setErrorMessage(null)
    },
    onSuccess: async (session) => {
      setSession({ token: session.token, user: session.user })
      queryClient.setQueryData(
        miboQueryKeys.authUser(session.token),
        session.user,
      )
      await navigate({ to: redirectTo })
    },
    onError: (error) => {
      setErrorMessage(
        error instanceof ApiError ? error.message : '初始化失败，请稍后重试。',
      )
    },
  })

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setupMutation.mutate()
  }

  return (
    <div className="flex min-h-svh items-center justify-center bg-muted p-6 md:p-10">
      <div className="w-full max-w-sm md:max-w-5xl">
        <Card className="overflow-hidden border-border/60 bg-card/95 p-0 shadow-2xl">
          <CardContent className="grid p-0 md:grid-cols-[1.05fr_0.95fr]">
            <form className="p-6 md:p-8" onSubmit={handleSubmit}>
              <FieldGroup>
                <div className="flex flex-col items-center gap-2 text-center">
                  <h1 className="text-2xl font-bold">初始化 Mibo</h1>
                  <p className="text-balance text-muted-foreground">
                    先创建首个管理员账号，完成后即可进入应用并继续配置媒体源。
                  </p>
                </div>

                {errorMessage ? (
                  <Alert variant="destructive">
                    <AlertTitle>初始化失败</AlertTitle>
                    <AlertDescription>{errorMessage}</AlertDescription>
                  </Alert>
                ) : null}

                <FieldSeparator className="*:data-[slot=field-separator-content]:bg-card">
                  First Admin
                </FieldSeparator>

                <Field>
                  <FieldLabel htmlFor="setup-username">用户名</FieldLabel>
                  <Input
                    id="setup-username"
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                    placeholder="请输入管理员用户名"
                    autoComplete="username"
                    required
                    disabled={setupMutation.isPending}
                  />
                </Field>

                <Field>
                  <FieldLabel htmlFor="setup-password">密码</FieldLabel>
                  <Input
                    id="setup-password"
                    type="password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    autoComplete="new-password"
                    required
                    disabled={setupMutation.isPending}
                  />
                  <FieldDescription>
                    创建成功后会自动登录，并跳转回应用首页。
                  </FieldDescription>
                </Field>

                <Field>
                  <Button
                    type="submit"
                    className="w-full"
                    disabled={setupMutation.isPending}
                  >
                    {setupMutation.isPending ? (
                      <Spinner className="size-4" />
                    ) : null}
                    {setupMutation.isPending
                      ? '初始化中...'
                      : '创建管理员并进入应用'}
                  </Button>
                </Field>

                <FieldDescription className="text-center">
                  默认开发值为{' '}
                  <span className="font-medium text-foreground">admin</span> /{' '}
                  <span className="font-medium text-foreground">admin123</span>
                </FieldDescription>
              </FieldGroup>
            </form>

            <div className="relative hidden overflow-hidden bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 p-8 text-white md:flex md:flex-col md:justify-between">
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(34,211,238,0.26),transparent_32%),radial-gradient(circle_at_bottom_left,rgba(16,185,129,0.22),transparent_30%)]" />

              <div className="relative space-y-3">
                <div className="inline-flex items-center rounded-full border border-white/15 bg-white/10 px-3 py-1 text-xs uppercase tracking-[0.24em] text-white/80">
                  Setup Gate
                </div>
                <div className="space-y-2">
                  <h2 className="text-3xl font-semibold tracking-tight">
                    先建立管理员入口，再进入媒体应用
                  </h2>
                  <p className="max-w-sm text-sm leading-6 text-white/70">
                    当前系统还没有任何管理员账号。创建首个账号后，首页、设置和播放路由才会解锁。
                  </p>
                </div>
              </div>

              <div className="relative space-y-4 rounded-2xl border border-white/10 bg-white/6 p-5 backdrop-blur-sm">
                <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-black/10 px-4 py-3">
                  <ShieldCheckIcon className="size-4 text-cyan-300" />
                  <div>
                    <p className="text-sm font-medium text-white">硬门禁</p>
                    <p className="text-sm text-white/70">
                      没有首个管理员前，非 `/setup` 路由都会被拦回初始化页。
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-black/10 px-4 py-3">
                  <SparklesIcon className="size-4 text-emerald-300" />
                  <div>
                    <p className="text-sm font-medium text-white">最小化开通</p>
                    <p className="text-sm text-white/70">
                      首个管理员创建后即可进应用，媒体源和媒体库继续在设置页完成。
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
