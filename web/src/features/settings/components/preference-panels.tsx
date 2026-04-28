import { useEffect, useState } from 'react'
import { InfoIcon } from 'lucide-react'
import { toast } from 'sonner'

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
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldTitle,
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
import { Switch } from '#/components/ui/switch'
import { Textarea } from '#/components/ui/textarea'

import { SettingSwitchField } from './setting-switch-field'

const GENERAL_SETTINGS_STORAGE_KEY = 'mibo-web-general-settings'

type GeneralSettingsForm = {
  language: string
  maintenanceMode: boolean
  cachePath: string
  automaticRestart: boolean
  loginDisclaimer: string
  customCSS: string
}

const defaultGeneralSettings: GeneralSettingsForm = {
  language: 'zh-Hans',
  maintenanceMode: false,
  cachePath: '',
  automaticRestart: true,
  loginDisclaimer: '',
  customCSS: '',
}

export function GeneralSettingsPanel() {
  const [draft, setDraft] = useState<GeneralSettingsForm>(
    defaultGeneralSettings,
  )

  useEffect(() => {
    const savedSettings = window.localStorage.getItem(
      GENERAL_SETTINGS_STORAGE_KEY,
    )

    if (!savedSettings) {
      return
    }

    try {
      setDraft({
        ...defaultGeneralSettings,
        ...(JSON.parse(savedSettings) as Partial<GeneralSettingsForm>),
      })
    } catch {
      window.localStorage.removeItem(GENERAL_SETTINGS_STORAGE_KEY)
    }
  }, [])

  function updateDraft<Value extends keyof GeneralSettingsForm>(
    key: Value,
    value: GeneralSettingsForm[Value],
  ) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    window.localStorage.setItem(
      GENERAL_SETTINGS_STORAGE_KEY,
      JSON.stringify(draft),
    )
    toast.success('通用设置已保存')
  }

  return (
    <SettingsPanelCard
      title="通用"
      description="管理服务器基础行为、Web 界面语言和全局登录页展示。"
      note="当前通用设置先保存在本机浏览器，后续可与服务端设置表打通。"
    >
      <form onSubmit={handleSubmit} className="space-y-6">
        <FieldGroup>
          <Field>
            <FieldLabel>首选显示语言</FieldLabel>
            <Select
              value={draft.language}
              onValueChange={(value) => updateDraft('language', value)}
            >
              <SelectTrigger className="w-full border-border/60 bg-background text-foreground md:max-w-md">
                <SelectValue placeholder="选择界面语言" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="zh-Hans">Chinese Simplified</SelectItem>
                <SelectItem value="en-US">English</SelectItem>
                <SelectItem value="ja-JP">Japanese</SelectItem>
              </SelectContent>
            </Select>
            <FieldDescription>
              控制 Mibo Web
              的默认显示语言。翻译会持续完善，欢迎在项目中提交翻译改进。
            </FieldDescription>
          </Field>

          <FormSwitchField
            title="维护模式"
            description="开启后，普通用户只会看到维护提示，管理员仍可进入设置中心检查状态。"
            checked={draft.maintenanceMode}
            onCheckedChange={(checked) =>
              updateDraft('maintenanceMode', checked)
            }
          />

          <Field>
            <FieldLabel htmlFor="general-cache-path">高级：缓存路径</FieldLabel>
            <Input
              id="general-cache-path"
              value={draft.cachePath}
              onChange={(event) => updateDraft('cachePath', event.target.value)}
              placeholder="留空则使用服务器默认缓存位置"
              className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
            />
            <FieldDescription>
              用于图片缓存、临时转码产物等服务器缓存文件。建议使用服务器可写的绝对路径。
            </FieldDescription>
          </Field>

          <FormSwitchField
            title="自动更新"
            description="允许服务器在空闲期间自动重启，以便插件或后台组件更新生效。"
            checked={draft.automaticRestart}
            onCheckedChange={(checked) =>
              updateDraft('automaticRestart', checked)
            }
          />

          <Field>
            <FieldLabel htmlFor="general-login-disclaimer">
              登录免责声明
            </FieldLabel>
            <Textarea
              id="general-login-disclaimer"
              value={draft.loginDisclaimer}
              onChange={(event) =>
                updateDraft('loginDisclaimer', event.target.value)
              }
              placeholder="显示在登录页底部的提示文字"
              className="min-h-28 border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
            />
            <FieldDescription>
              可用于展示家庭媒体库使用说明、访问边界或隐私提示。
            </FieldDescription>
          </Field>

          <Field>
            <FieldLabel htmlFor="general-custom-css">自定义 CSS</FieldLabel>
            <Textarea
              id="general-custom-css"
              value={draft.customCSS}
              onChange={(event) => updateDraft('customCSS', event.target.value)}
              placeholder=".mibo-login { backdrop-filter: blur(16px); }"
              className="min-h-40 border-border/60 bg-background font-mono text-sm text-foreground placeholder:text-muted-foreground"
            />
            <FieldDescription>
              用 CSS 代码微调 Mibo Web 外观。保存后会在后续全局样式接入中生效。
            </FieldDescription>
          </Field>
        </FieldGroup>

        <Button
          type="submit"
          size="lg"
          className="w-full bg-emerald-600 text-white hover:bg-emerald-700"
        >
          保存
        </Button>
      </form>
    </SettingsPanelCard>
  )
}

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

function FormSwitchField({
  title,
  description,
  checked,
  onCheckedChange,
}: {
  title: string
  description: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <Field
      orientation="horizontal"
      className="items-start rounded-[1.25rem] border border-border/60 bg-muted/30 p-3.5"
    >
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        className="mt-0.5 data-[state=checked]:bg-emerald-600"
      />
      <FieldContent>
        <FieldTitle className="text-foreground">{title}</FieldTitle>
        <FieldDescription>{description}</FieldDescription>
      </FieldContent>
    </Field>
  )
}
