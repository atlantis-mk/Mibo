import { Link } from '@tanstack/react-router'
import {
  ArrowLeftIcon,
  BellIcon,
  CalendarClockIcon,
  DatabaseIcon,
  SparklesIcon,
  PlayCircleIcon,
  Settings2Icon,
  ShieldCheckIcon,
} from 'lucide-react'

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
} from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '#/components/ui/select'
import { Separator } from '#/components/ui/separator'
import { Tabs, TabsContent, TabsList } from '#/components/ui/tabs'
import { useAuthStore } from '#/stores/auth-store'

import { LibraryManagementPanel } from './components/library-management-panel'
import { SettingSwitchField } from './components/setting-switch-field'
import { SettingsAsideCard } from './components/settings-aside-card'
import { SettingsMenuTrigger } from './components/settings-menu-trigger'

export default function SettingsPage() {
  const token = useAuthStore((state) => state.token)

  return (
    <div className="min-h-svh bg-background px-4 py-6 text-foreground sm:px-6 lg:px-8 xl:px-10">
      <div className="mx-auto max-w-450">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="flex size-11 items-center justify-center rounded-2xl border border-border/60 bg-card/80">
                <Settings2Icon className="size-5 text-muted-foreground" />
              </div>
              <div>
                <h1 className="text-3xl font-semibold tracking-tight">设置</h1>
                <p className="mt-1 text-sm text-muted-foreground">
                  左侧切换设置分类，右侧查看对应的系统配置与偏好项。
                </p>
              </div>
            </div>
          </div>

          <Button
            asChild
            variant="outline"
            className="border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground"
          >
            <Link to="/">
              <ArrowLeftIcon className="size-4" />
              返回首页
            </Link>
          </Button>

          <Button
            asChild
            variant="outline"
            className="border-border/60 bg-card/80 text-foreground hover:bg-muted hover:text-foreground"
          >
            <Link to="/metadata">
              <SparklesIcon className="size-4" />
              元数据治理
            </Link>
          </Button>
        </div>

        <Tabs
          defaultValue="library"
          orientation="vertical"
          className="mt-6 gap-4 lg:grid lg:grid-cols-[240px_minmax(0,1fr)] lg:items-start xl:grid-cols-[260px_minmax(0,1fr)]"
        >
          <aside className="lg:sticky lg:top-6">
            <div className="rounded-[1.5rem] border border-border/60 bg-card/80 p-2.5 shadow-sm backdrop-blur-sm">
              <div className="px-2.5 pb-2.5 pt-1.5">
                <div className="text-sm font-medium text-foreground">
                  设置菜单
                </div>
                <div className="mt-1 text-sm text-muted-foreground">
                  选择要查看或调整的配置分组。
                </div>
              </div>
              <TabsList
                variant="line"
                className="flex w-full flex-col items-stretch gap-1 rounded-[1.25rem] bg-transparent p-0"
              >
                <SettingsMenuTrigger
                  value="library"
                  icon={DatabaseIcon}
                  title="媒体库"
                  description="媒体源、扫描和目录"
                />
                <SettingsMenuTrigger
                  value="playback"
                  icon={PlayCircleIcon}
                  title="播放与设备"
                  description="播放偏好和质量策略"
                />
                <SettingsMenuTrigger
                  value="notifications"
                  icon={BellIcon}
                  title="通知与任务"
                  description="同步提醒和后台任务"
                />
                <SettingsMenuTrigger
                  value="security"
                  icon={ShieldCheckIcon}
                  title="账号与安全"
                  description="访问控制与登录保护"
                />
              </TabsList>
            </div>
          </aside>

          <div className="space-y-4">
            <TabsContent value="library" className="mt-0">
              <LibraryManagementPanel token={token} />
            </TabsContent>

            <TabsContent value="playback" className="mt-0">
              <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px] 2xl:grid-cols-[minmax(0,1fr)_320px]">
                <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
                  <CardHeader className="px-5 py-5">
                    <CardTitle className="text-xl">播放与设备</CardTitle>
                    <CardDescription>
                      调整播放清晰度、自动续播和设备兼容策略。
                    </CardDescription>
                  </CardHeader>
                  <Separator className="bg-border" />
                  <CardContent className="px-5 py-5">
                    <FieldGroup>
                      <div className="grid gap-4 md:grid-cols-2">
                        <Field>
                          <FieldLabel>默认画质</FieldLabel>
                          <Select defaultValue="original">
                            <SelectTrigger className="w-full border-border/60 bg-background text-foreground">
                              <SelectValue placeholder="选择画质" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="original">原始质量</SelectItem>
                              <SelectItem value="1080p">1080p</SelectItem>
                              <SelectItem value="720p">720p</SelectItem>
                            </SelectContent>
                          </Select>
                        </Field>

                        <Field>
                          <FieldLabel>设备档案</FieldLabel>
                          <Select defaultValue="web">
                            <SelectTrigger className="w-full border-border/60 bg-background text-foreground">
                              <SelectValue placeholder="选择设备档案" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="web">Web</SelectItem>
                              <SelectItem value="tv">TV</SelectItem>
                              <SelectItem value="mobile">Mobile</SelectItem>
                            </SelectContent>
                          </Select>
                        </Field>
                      </div>

                      <SettingSwitchField
                        title="自动续播"
                        description="打开媒体详情时优先恢复到上次播放进度。"
                        defaultChecked
                      />
                      <SettingSwitchField
                        title="优先转码兼容格式"
                        description="当原始文件不可直放时优先选择兼容性更高的输出格式。"
                      />
                    </FieldGroup>
                  </CardContent>
                </Card>

                <SettingsAsideCard
                  title="播放摘要"
                  description="当前播放体验的默认策略。"
                  items={[
                    ['默认画质', '原始质量'],
                    ['设备档案', 'Web'],
                    ['自动续播', '已开启'],
                  ]}
                />
              </div>
            </TabsContent>

            <TabsContent value="notifications" className="mt-0">
              <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px] 2xl:grid-cols-[minmax(0,1fr)_320px]">
                <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
                  <CardHeader className="px-5 py-5">
                    <CardTitle className="text-xl">通知与任务</CardTitle>
                    <CardDescription>
                      管理后台任务提醒、失败通知和完成回执。
                    </CardDescription>
                  </CardHeader>
                  <Separator className="bg-border" />
                  <CardContent className="px-5 py-5">
                    <FieldGroup>
                      <div className="rounded-[1.25rem] border border-border/60 bg-background/60 px-4 py-4">
                        <div className="flex items-start justify-between gap-4">
                          <div className="space-y-1">
                            <div className="flex items-center gap-2 text-sm font-medium text-foreground">
                              <CalendarClockIcon className="size-4" />
                              计划任务工作台
                            </div>
                            <div className="text-sm text-muted-foreground">
                              在独立工作台中统一管理计划任务的启停、立即运行和最近运行历史。设置页仅保留摘要入口，不承载完整管理流。
                            </div>
                          </div>

                          <Button asChild variant="outline" className="border-border/60 bg-card/80">
                            <Link to="/schedules">进入工作台</Link>
                          </Button>
                        </div>
                      </div>

                      <Field>
                        <FieldLabel htmlFor="notification-email">
                          通知邮箱
                        </FieldLabel>
                        <Input
                          id="notification-email"
                          defaultValue="admin@mibo.local"
                          className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
                        />
                        <FieldDescription>
                          后台扫描或识别失败时用于接收摘要通知。
                        </FieldDescription>
                      </Field>

                      <SettingSwitchField
                        title="任务完成提醒"
                        description="扫描、识别和刷新任务完成后显示通知。"
                        defaultChecked
                      />
                      <SettingSwitchField
                        title="仅提醒失败任务"
                        description="减少干扰，只在出错时发送重点提醒。"
                        defaultChecked
                      />
                    </FieldGroup>
                  </CardContent>
                </Card>

                <SettingsAsideCard
                  title="通知摘要"
                  description="当前任务与提醒相关的输出方式。"
                  items={[
                    ['计划任务', '独立工作台管理'],
                    ['通知邮箱', 'admin@mibo.local'],
                    ['完成提醒', '已开启'],
                    ['失败提醒', '仅失败'],
                  ]}
                />
              </div>
            </TabsContent>

            <TabsContent value="security" className="mt-0">
              <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px] 2xl:grid-cols-[minmax(0,1fr)_320px]">
                <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
                  <CardHeader className="px-5 py-5">
                    <CardTitle className="text-xl">账号与安全</CardTitle>
                    <CardDescription>
                      配置管理员账号保护、会话时长与访问收敛策略。
                    </CardDescription>
                  </CardHeader>
                  <Separator className="bg-border" />
                  <CardContent className="px-5 py-5">
                    <FieldGroup>
                      <div className="grid gap-4 md:grid-cols-2">
                        <Field>
                          <FieldLabel htmlFor="session-timeout">
                            会话时长
                          </FieldLabel>
                          <Input
                            id="session-timeout"
                            defaultValue="24h"
                            className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
                          />
                        </Field>

                        <Field>
                          <FieldLabel>登录保护级别</FieldLabel>
                          <Select defaultValue="standard">
                            <SelectTrigger className="w-full border-border/60 bg-background text-foreground">
                              <SelectValue placeholder="选择级别" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="standard">标准</SelectItem>
                              <SelectItem value="strict">严格</SelectItem>
                              <SelectItem value="local-only">
                                仅本地网络
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </Field>
                      </div>

                      <SettingSwitchField
                        title="自动清理失效 token"
                        description="当服务端判定 token 失效时立即移除本地会话。"
                        defaultChecked
                      />
                      <SettingSwitchField
                        title="限制高危操作二次确认"
                        description="对删除库、重扫等高风险动作增加额外确认步骤。"
                      />
                    </FieldGroup>
                  </CardContent>
                </Card>

                <SettingsAsideCard
                  title="安全摘要"
                  description="当前账号和会话保护的核心状态。"
                  items={[
                    ['会话时长', '24h'],
                    ['保护级别', '标准'],
                    ['失效 token 清理', '已开启'],
                  ]}
                />
              </div>
            </TabsContent>
          </div>
        </Tabs>
      </div>
    </div>
  )
}
