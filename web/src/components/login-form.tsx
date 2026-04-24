import { useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { CheckCircle2Icon, LogOutIcon, ShieldCheckIcon } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import { Spinner } from '#/components/ui/spinner'
import { useLoginSession } from '#/hooks/use-login-session'
import { cn } from '#/lib/utils'

export function LoginForm({
  className,
  redirectTo = '/',
  ...props
}: React.ComponentProps<'div'> & { redirectTo?: string }) {
  const {
    username,
    setUsername,
    password,
    setPassword,
    errorMessage,
    isSubmitting,
    hasHydrated,
    user,
    login,
    logout,
  } = useLoginSession()
  const navigate = useNavigate()

  useEffect(() => {
    if (!hasHydrated || !user) {
      return
    }

    void navigate({ to: redirectTo })
  }, [hasHydrated, navigate, redirectTo, user])

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    void login()
  }

  if (!hasHydrated) {
    return (
      <div className={cn('flex flex-col gap-6', className)} {...props}>
        <Card className="border-border/60 bg-card/80 shadow-xl backdrop-blur">
          <CardContent className="flex min-h-72 items-center justify-center">
            <Spinner className="size-5" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (user) {
    return (
      <div className={cn('flex flex-col gap-6', className)} {...props}>
        <Card className="border-border/60 bg-card/90 shadow-xl backdrop-blur">
          <CardHeader className="text-center">
            <div className="mx-auto flex size-12 items-center justify-center rounded-2xl bg-emerald-500/12 text-emerald-700">
              <CheckCircle2Icon className="size-6" />
            </div>
            <CardTitle className="mt-2 text-2xl">登录成功，正在跳转</CardTitle>
            <CardDescription>
              当前账号为{' '}
              <span className="font-semibold text-foreground">
                {user.username}
              </span>
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-2xl border border-emerald-600/15 bg-emerald-500/6 p-4 text-sm">
              <div className="flex items-center gap-2 font-medium text-foreground">
                <ShieldCheckIcon className="size-4 text-emerald-700" />
                会话已建立
              </div>
              <div className="mt-2 space-y-1 text-muted-foreground">
                <p>角色：{user.role}</p>
                <p>ID：{user.id}</p>
              </div>
            </div>
            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={() => {
                void logout()
              }}
              disabled={isSubmitting}
            >
              {isSubmitting ? (
                <Spinner className="size-4" />
              ) : (
                <LogOutIcon className="size-4" />
              )}
              退出登录
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col gap-6', className)} {...props}>
      <Card className="overflow-hidden border-border/60 bg-card/95 p-0 shadow-2xl">
        <CardContent className="grid p-0 md:grid-cols-2">
          <form className="p-6 md:p-8" onSubmit={handleSubmit}>
            <FieldGroup>
              <div className="flex flex-col items-center gap-2 text-center">
                <h1 className="text-2xl font-bold">登录 Mibo</h1>
                <p className="text-balance text-muted-foreground">
                  使用已创建的管理员账号登录媒体库
                </p>
              </div>

              {errorMessage ? (
                <Alert variant="destructive">
                  <AlertTitle>登录失败</AlertTitle>
                  <AlertDescription>{errorMessage}</AlertDescription>
                </Alert>
              ) : null}

              <FieldSeparator className="*:data-[slot=field-separator-content]:bg-card">
                Mibo Media Server
              </FieldSeparator>

              <Field>
                <FieldLabel htmlFor="username">用户名</FieldLabel>
                <Input
                  id="username"
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  placeholder="请输入管理员用户名"
                  autoComplete="username"
                  required
                  disabled={isSubmitting}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="password">密码</FieldLabel>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="current-password"
                  required
                  disabled={isSubmitting}
                />
              </Field>
              <Field>
                <Button
                  type="submit"
                  className="w-full"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? <Spinner className="size-4" /> : null}
                  {isSubmitting ? '登录中...' : '登录'}
                </Button>
              </Field>
              <FieldDescription className="text-center">
                默认开发账号为{' '}
                <span className="font-medium text-foreground">admin</span> /{' '}
                <span className="font-medium text-foreground">admin123</span>
              </FieldDescription>
            </FieldGroup>
          </form>
          <div className="relative hidden overflow-hidden bg-gradient-to-br from-slate-950 via-slate-900 to-emerald-950 p-8 text-white md:flex md:flex-col md:justify-between">
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(16,185,129,0.28),transparent_35%),radial-gradient(circle_at_bottom_left,rgba(59,130,246,0.18),transparent_30%)]" />
            <div className="relative space-y-3">
              <div className="inline-flex items-center rounded-full border border-white/15 bg-white/10 px-3 py-1 text-xs uppercase tracking-[0.24em] text-white/80">
                Mibo Media Server
              </div>
              <div className="space-y-2">
                <h2 className="text-3xl font-semibold tracking-tight">
                  管理你的私人媒体库
                </h2>
                <p className="max-w-sm text-sm leading-6 text-white/70">
                  登录后即可连接媒体源、同步资源，并在一个界面里管理影片与剧集数据。
                </p>
              </div>
            </div>
            <div className="relative space-y-4 rounded-2xl border border-white/10 bg-white/6 p-5 backdrop-blur-sm">
              <div>
                <p className="text-sm font-medium text-white">登录后可用</p>
                <p className="mt-1 text-sm text-white/70">
                  管理员鉴权、会话保持、账号信息校验。
                </p>
              </div>
              <div className="grid gap-3 text-sm text-white/80">
                <div className="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                  连接 Mibo 后端 API
                </div>
                <div className="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                  自动恢复已保存会话
                </div>
                <div className="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                  失效 token 自动清理
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
      <FieldDescription className="px-6 text-center">
        当前页面已接入真实登录接口，不再使用模板占位内容。
      </FieldDescription>
    </div>
  )
}
