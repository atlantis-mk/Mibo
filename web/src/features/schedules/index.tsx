import { Link } from '@tanstack/react-router'
import { LoaderCircleIcon } from 'lucide-react'

import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import { useAuthStore } from '#/stores/auth-store'

import { SchedulesWorkspace } from './workspace'

export default function SchedulesPage() {
  const token = useAuthStore((state) => state.token)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)

  if (!hasHydrated) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-full border border-border/50 bg-card/85 px-5 py-3">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-muted-foreground">正在准备计划任务工作台</span>
        </div>
      </div>
    )
  }

  if (!token) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background px-6 text-foreground">
        <div className="max-w-xl space-y-4 text-center">
          <Badge variant="outline" className="border-border/60 bg-card/80">
            Scheduled Operations
          </Badge>
          <h1 className="text-3xl font-semibold tracking-tight">
            登录后进入计划任务工作台
          </h1>
          <p className="text-sm leading-7 text-muted-foreground">
            该页面需要管理员会话来查看计划任务状态、运行历史和后台维护入口。
          </p>
          <Button asChild>
            <Link to="/login" search={{ redirect: '/schedules' }}>
              前往登录
            </Link>
          </Button>
        </div>
      </div>
    )
  }

  return <SchedulesWorkspace token={token} />
}
