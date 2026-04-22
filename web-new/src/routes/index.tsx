import { createFileRoute, Link } from '@tanstack/react-router'
import { Loader2, Rocket, Wrench } from 'lucide-react'
import { useEffect, useState } from 'react'

import { Badge } from '~/components/ui/badge'
import { Button } from '~/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '~/components/ui/card'
import { getStoredApiBaseUrl } from '~/lib/client-config'
import { createMiboApi, type SetupStatus } from '~/lib/mibo-api'

export const Route = createFileRoute('/')({
  component: Home,
})

function Home() {
  const [setupStatus, setSetupStatus] = useState<SetupStatus | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      try {
        const status = await createMiboApi({
          baseUrl: getStoredApiBaseUrl(),
        }).getSetupStatus()

        if (!cancelled) {
          setSetupStatus(status)
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(
            loadError instanceof Error ? loadError.message : '无法连接后端服务'
          )
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  }, [])

  return (
    <main className="min-h-screen bg-background px-4 py-8 text-foreground sm:px-6 lg:px-8">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
        <section className="grid gap-6 rounded-[2rem] border border-border/70 bg-card/80 p-6 shadow-sm backdrop-blur lg:grid-cols-[1.2fr_0.8fr] lg:p-8">
          <div className="space-y-4">
            <Badge variant="outline" className="border-primary/30 bg-primary/5 text-primary">
              TanStack Start 迁移中
            </Badge>
            <div className="space-y-3">
              <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
                `web-new/` 已接管 Mibo 的初始化入口。
              </h1>
              <p className="max-w-2xl text-base leading-7 text-muted-foreground">
                当前这一步先迁移最关键的启动骨架：主题与提示层、后端地址与会话存储、setup gate，以及首轮初始化向导。后续业务页可以在这个新壳上继续逐段搬迁。
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Button asChild>
                <Link to="/setup">
                  <Rocket className="size-4" />
                  打开初始化向导
                </Link>
              </Button>
              <Button asChild variant="outline">
                <a href="http://127.0.0.1:8080/api/v1/setup/status" target="_blank" rel="noreferrer">
                  <Wrench className="size-4" />
                  查看后端 setup 状态
                </a>
              </Button>
            </div>
          </div>

          <Card className="border-border/70 bg-background/70">
            <CardHeader>
              <CardTitle>当前接入状态</CardTitle>
              <CardDescription>
                展示新框架壳层是否能正确读到 `mibo-media-server` 的 setup 接口。
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              {setupStatus ? (
                <>
                  <StatusRow label="可进入应用" value={setupStatus.can_enter_app ? '是' : '否'} />
                  <StatusRow label="已创建用户" value={String(setupStatus.user_count)} />
                  <StatusRow label="媒体源数量" value={String(setupStatus.media_source_count)} />
                  <StatusRow label="媒体库数量" value={String(setupStatus.library_count)} />
                </>
              ) : error ? (
                <div className="rounded-2xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-destructive">
                  {error}
                </div>
              ) : (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="size-4 animate-spin" />
                  正在读取 setup 状态...
                </div>
              )}
            </CardContent>
          </Card>
        </section>
      </div>
    </main>
  )
}

function StatusRow(props: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between rounded-2xl border border-border/70 bg-card px-4 py-3">
      <span className="text-muted-foreground">{props.label}</span>
      <span className="font-medium">{props.value}</span>
    </div>
  )
}
