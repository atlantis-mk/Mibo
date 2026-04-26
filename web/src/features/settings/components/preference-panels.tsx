import { InfoIcon } from 'lucide-react'

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

import { SettingSwitchField } from './setting-switch-field'

export function PlaybackSettingsPanel() {
  return (
    <SettingsPanelCard
      title="播放体验"
      description="集中调整默认画质、设备档案和自动续播策略。"
      note="这些偏好目前用于界面预设展示，后续会接入服务端持久化。"
    >
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
            <FieldDescription>
              优先使用原始文件，无法直放时再降级。
            </FieldDescription>
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
            <FieldDescription>
              根据播放端选择更合适的兼容策略。
            </FieldDescription>
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
    </SettingsPanelCard>
  )
}

export function NotificationSettingsPanel() {
  return (
    <SettingsPanelCard
      title="任务通知"
      description="控制后台任务的完成提醒、失败提醒和通知邮箱。"
      note="通知配置会在后续版本与服务端设置表打通。"
    >
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="notification-email">通知邮箱</FieldLabel>
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
    </SettingsPanelCard>
  )
}

export function SecuritySettingsPanel() {
  return (
    <SettingsPanelCard
      title="账号安全"
      description="收敛登录会话、token 清理和高风险操作确认策略。"
      note="当前会话保护由认证模块执行，这里先保留管理入口和默认策略说明。"
    >
      <FieldGroup>
        <div className="grid gap-4 md:grid-cols-2">
          <Field>
            <FieldLabel htmlFor="session-timeout">会话时长</FieldLabel>
            <Input
              id="session-timeout"
              defaultValue="24h"
              className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
            />
            <FieldDescription>超过该时长后需要重新登录。</FieldDescription>
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
                <SelectItem value="local-only">仅本地网络</SelectItem>
              </SelectContent>
            </Select>
            <FieldDescription>控制登录保护和访问范围。</FieldDescription>
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
    </SettingsPanelCard>
  )
}

function SettingsPanelCard({
  title,
  description,
  note,
  children,
}: {
  title: string
  description: string
  note: string
  children: React.ReactNode
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle className="text-xl">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-5 px-5 py-5">
        <div className="flex items-start gap-3 rounded-[1.15rem] border border-border/60 bg-muted/30 px-4 py-3 text-sm leading-6 text-muted-foreground">
          <InfoIcon className="mt-0.5 size-4 shrink-0" />
          <span>{note}</span>
        </div>
        {children}
      </CardContent>
    </Card>
  )
}
