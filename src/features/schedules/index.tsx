import { Link } from '@tanstack/react-router'
import { LoaderCircleIcon } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SchedulesWorkspace } from './workspace'

export default function SchedulesPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)

  if (!hasHydrated) {
    return (
      <div className='flex items-center gap-3 rounded-[1.5rem] border border-border/60 bg-card/80 px-5 py-4 text-foreground shadow-sm'>
        <LoaderCircleIcon className='size-4 animate-spin' />
        <span className='text-sm text-muted-foreground'>
          正在准备计划任务工作台
        </span>
      </div>
    )
  }

  if (!token) {
    return (
      <div className='rounded-[1.5rem] border border-border/60 bg-card/80 px-6 py-8 text-foreground shadow-sm'>
        <div className='max-w-xl space-y-4'>
          <Badge variant='outline' className='border-border/60 bg-card/80'>
            Scheduled Operations
          </Badge>
          <h1 className='text-2xl font-semibold tracking-tight'>
            登录后进入计划任务工作台
          </h1>
          <p className='text-sm leading-7 text-muted-foreground'>
            该页面需要管理员会话来查看计划任务状态、运行历史和后台维护入口。
          </p>
          <Button asChild>
            <Link to='/sign-in' search={{ redirect: '/settings/schedules' }}>
              前往登录
            </Link>
          </Button>
        </div>
      </div>
    )
  }

  return <SchedulesWorkspace token={token} />
}
