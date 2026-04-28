import { useState } from 'react'
import type { ComponentType } from 'react'
import {
  ArrowUpIcon,
  BadgeCheckIcon,
  CalendarClockIcon,
  ChevronRightIcon,
  Clock3Icon,
  MoreVerticalIcon,
  PlusIcon,
  ShieldCheckIcon,
  UserIcon,
  UsersIcon,
} from 'lucide-react'

import { Avatar, AvatarFallback } from '#/components/ui/avatar'
import { Button } from '#/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '#/components/ui/dropdown-menu'
import type { User } from '#/lib/mibo-api'
import { cn } from '#/lib/utils'
import { useAuthStore } from '#/stores/auth-store'

type SettingsUser = User & {
  lastActiveAt: string
}

export function UserManagementPanel() {
  const authUser = useAuthStore((state) => state.user)
  const fallbackUser: SettingsUser = {
    id: 0,
    username: 'root',
    role: 'admin',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    lastActiveAt: new Date().toISOString(),
  }
  const users: SettingsUser[] = [
    {
      ...fallbackUser,
      ...(authUser ?? {}),
      lastActiveAt: authUser?.updated_at ?? fallbackUser.lastActiveAt,
    },
  ].sort((left, right) => left.username.localeCompare(right.username))
  const [selectedUserId, setSelectedUserId] = useState(users[0]?.id ?? 0)
  const selectedUser =
    users.find((user) => user.id === selectedUserId) ?? users[0]

  return (
    <div className="space-y-4">
      <section className="rounded-[1.5rem] border border-border/60 bg-card/70 p-4 shadow-sm backdrop-blur-sm">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
          <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
            <div className="inline-flex items-center gap-2 rounded-full border border-border/60 bg-background/70 px-3 py-1.5 text-foreground">
              <UsersIcon className="size-4 text-emerald-500" />共 {users.length}{' '}
              个用户
            </div>
            <div className="inline-flex items-center gap-1.5 rounded-full border border-border/60 bg-background/50 px-3 py-1.5">
              <ArrowUpIcon className="size-3.5 text-emerald-500" />
              按标题升序
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button disabled title="后端用户创建接口接入后启用">
              <PlusIcon className="size-4" />
              新增用户
            </Button>
            <Button variant="outline">
              <ArrowUpIcon className="size-4" />
              标题
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon" aria-label="更多用户操作">
                  <MoreVerticalIcon className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-52">
                <DropdownMenuLabel>更多操作</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem disabled>批量编辑用户</DropdownMenuItem>
                <DropdownMenuItem disabled>导出用户列表</DropdownMenuItem>
                <DropdownMenuItem disabled>登录策略</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </section>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
        <section className="min-h-[420px] rounded-[1.5rem] border border-border/60 bg-gradient-to-br from-card/80 via-card/60 to-emerald-500/5 p-5 shadow-sm backdrop-blur-sm">
          <div className="mb-5 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-base font-medium">服务器用户</h3>
              <p className="text-sm text-muted-foreground">
                点击用户卡片查看账号详情与权限概览。
              </p>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-3">
            {users.map((user) => {
              const selected = user.id === selectedUser?.id
              return (
                <button
                  key={user.id}
                  type="button"
                  onClick={() => setSelectedUserId(user.id)}
                  className={cn(
                    'group rounded-[1.25rem] border bg-background/75 p-4 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:border-emerald-500/50 hover:shadow-md focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-emerald-500/25',
                    selected
                      ? 'border-emerald-500/60 ring-3 ring-emerald-500/15'
                      : 'border-border/60',
                  )}
                >
                  <div className="flex items-start justify-between gap-3">
                    <Avatar className="size-16 rounded-2xl" size="lg">
                      <AvatarFallback className="rounded-2xl bg-muted text-lg font-semibold text-muted-foreground">
                        {getUserInitial(user.username)}
                      </AvatarFallback>
                    </Avatar>
                    <ChevronRightIcon className="mt-2 size-4 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-emerald-500" />
                  </div>
                  <div className="mt-4 min-w-0">
                    <div className="truncate text-base font-semibold">
                      {user.username}
                    </div>
                    <div className="mt-1 flex items-center gap-1.5 text-sm text-muted-foreground">
                      <Clock3Icon className="size-3.5" />
                      最近活动 {formatRelativeTime(user.lastActiveAt)}
                    </div>
                  </div>
                </button>
              )
            })}
          </div>
        </section>

        <UserDetailCard user={selectedUser} />
      </div>
    </div>
  )
}

function UserDetailCard({ user }: { user?: SettingsUser }) {
  if (!user) {
    return null
  }

  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 shadow-sm backdrop-blur-sm">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <UserIcon className="size-4 text-emerald-500" />
          用户详情
        </CardTitle>
        <CardDescription>当前展示本地会话用户资料。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center gap-3 rounded-2xl border border-border/60 bg-background/70 p-3">
          <Avatar className="size-12" size="lg">
            <AvatarFallback className="bg-muted font-semibold">
              {getUserInitial(user.username)}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0">
            <div className="truncate font-medium">{user.username}</div>
            <div className="text-xs text-muted-foreground">ID #{user.id}</div>
          </div>
        </div>

        <DetailRow
          icon={ShieldCheckIcon}
          label="角色"
          value={formatRole(user.role)}
        />
        <DetailRow
          icon={Clock3Icon}
          label="最近活动"
          value={formatRelativeTime(user.lastActiveAt)}
        />
        <DetailRow
          icon={CalendarClockIcon}
          label="创建时间"
          value={formatDateTime(user.created_at)}
        />
        <DetailRow icon={BadgeCheckIcon} label="账号状态" value="可登录" />

        <div className="rounded-2xl border border-dashed border-border/70 bg-muted/30 p-3 text-sm leading-6 text-muted-foreground">
          用户新增、密码重置、媒体库访问控制等操作需要后端用户管理接口后启用。
        </div>
      </CardContent>
    </Card>
  )
}

function DetailRow({
  icon: Icon,
  label,
  value,
}: {
  icon: ComponentType<{ className?: string }>
  label: string
  value: string
}) {
  return (
    <div className="flex items-center gap-3 rounded-2xl border border-border/50 bg-background/50 px-3 py-2.5">
      <div className="flex size-8 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
        <Icon className="size-4" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="truncate text-sm font-medium">{value}</div>
      </div>
    </div>
  )
}

function getUserInitial(username: string) {
  return username.trim().slice(0, 1).toUpperCase() || 'U'
}

function formatRole(role: string) {
  return role === 'admin' ? '管理员' : role || '普通用户'
}

function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '未知'
  }

  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function formatRelativeTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '未知'
  }

  const diffSeconds = Math.max(
    1,
    Math.floor((Date.now() - date.getTime()) / 1000),
  )
  if (diffSeconds < 60) {
    return `${diffSeconds}秒钟前`
  }

  const diffMinutes = Math.floor(diffSeconds / 60)
  if (diffMinutes < 60) {
    return `${diffMinutes}分钟前`
  }

  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) {
    return `${diffHours}小时前`
  }

  return `${Math.floor(diffHours / 24)}天前`
}
