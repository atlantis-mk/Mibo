import { useEffect, useState } from 'react'
import {
  ExternalLinkIcon,
  FolderOpenIcon,
  HelpCircleIcon,
  InfoIcon,
  Settings2Icon,
} from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '#/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '#/components/ui/card'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '#/components/ui/tabs'

const TRANSCODING_SETTINGS_STORAGE_KEY = 'mibo-web-transcoding-settings'

type TranscodingSettingsForm = {
  hardwareAcceleration: string
  transcodingTempPath: string
  throttlingEnabled: boolean
  audioBoost: string
  subtitleExtractionEnabled: boolean
  subtitleFontExtractionEnabled: boolean
  hevcEncodingEnabled: boolean
  maxResolution: string
  softwareToneMappingEnabled: boolean
}

const defaultTranscodingSettings: TranscodingSettingsForm = {
  hardwareAcceleration: 'yes',
  transcodingTempPath: '',
  throttlingEnabled: true,
  audioBoost: '2',
  subtitleExtractionEnabled: true,
  subtitleFontExtractionEnabled: true,
  hevcEncodingEnabled: false,
  maxResolution: 'unlimited',
  softwareToneMappingEnabled: false,
}

const softwareEncoders = [
  {
    name: 'H.264 (AVC)',
    description: '兼容性最高的视频输出格式，适合 Web、移动端和旧设备。',
  },
  {
    name: 'H.265 (HEVC)',
    description: '在同等画质下减少带宽占用，适合支持 HEVC 的客户端。',
  },
]

export function TranscodingSettingsPanel() {
  const [draft, setDraft] = useState<TranscodingSettingsForm>(
    defaultTranscodingSettings,
  )

  useEffect(() => {
    const savedSettings = window.localStorage.getItem(
      TRANSCODING_SETTINGS_STORAGE_KEY,
    )

    if (!savedSettings) {
      return
    }

    try {
      setDraft({
        ...defaultTranscodingSettings,
        ...(JSON.parse(savedSettings) as Partial<TranscodingSettingsForm>),
      })
    } catch {
      window.localStorage.removeItem(TRANSCODING_SETTINGS_STORAGE_KEY)
    }
  }, [])

  function updateDraft<Value extends keyof TranscodingSettingsForm>(
    key: Value,
    value: TranscodingSettingsForm[Value],
  ) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    window.localStorage.setItem(
      TRANSCODING_SETTINGS_STORAGE_KEY,
      JSON.stringify(draft),
    )
    toast.success('转码设置已保存')
  }

  return (
    <form onSubmit={handleSubmit}>
      <Tabs defaultValue="transcoding" className="space-y-4">
        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
          <CardHeader className="gap-4 px-5 py-5">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-start gap-3">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-border/60 bg-background/70">
                  <HelpCircleIcon className="size-4 text-muted-foreground" />
                </div>
                <div>
                  <CardTitle className="text-xl">转码控制中心</CardTitle>
                  <p className="mt-1 text-sm leading-6 text-muted-foreground">
                    管理硬件加速、转码缓存、字幕实时处理、HEVC 输出和 HDR
                    色调映射。
                  </p>
                </div>
              </div>

              <TabsList className="w-full sm:w-fit">
                <TabsTrigger value="transcoding" className="px-4">
                  转码
                </TabsTrigger>
                <TabsTrigger value="tone-mapping" className="px-4">
                  色调映射
                </TabsTrigger>
              </TabsList>
            </div>
          </CardHeader>
        </Card>

        <TabsContent value="transcoding" className="mt-0 space-y-4">
          <SettingsBlock
            title="硬件加速"
            description="在可用时使用 GPU 或系统硬件编解码能力，减少 CPU 压力。"
          >
            <FieldGroup>
              <Field>
                <FieldLabel>启用硬件加速（如果可用）</FieldLabel>
                <Select
                  value={draft.hardwareAcceleration}
                  onValueChange={(value) =>
                    updateDraft('hardwareAcceleration', value)
                  }
                >
                  <SelectTrigger className="w-full border-border/60 bg-background text-foreground md:max-w-sm">
                    <SelectValue placeholder="选择硬件加速策略" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="yes">是</SelectItem>
                    <SelectItem value="no">否</SelectItem>
                    <SelectItem value="advanced">高级</SelectItem>
                  </SelectContent>
                </Select>
                <FieldDescription>
                  让 Mibo 在检测到可用硬件时优先使用硬件编码和解码。
                </FieldDescription>
              </Field>

              <Button
                type="button"
                variant="outline"
                className="w-fit rounded-full"
                onClick={() => toast.info('硬件加速设置指南稍后接入文档页')}
              >
                <ExternalLinkIcon className="size-4" />
                硬件加速设置指南
              </Button>
            </FieldGroup>
          </SettingsBlock>

          <SettingsBlock
            title="软件编码器"
            description="配置软件转码输出编码器。齿轮入口将承载更细的编码参数。"
          >
            <div className="space-y-3">
              {softwareEncoders.map((encoder) => (
                <div
                  key={encoder.name}
                  className="flex items-center justify-between gap-4 rounded-[1.15rem] border border-border/60 bg-muted/30 px-4 py-3"
                >
                  <div className="min-w-0">
                    <div className="font-medium text-foreground">
                      {encoder.name}
                    </div>
                    <div className="mt-1 text-sm leading-6 text-muted-foreground">
                      {encoder.description}
                    </div>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="shrink-0 rounded-full"
                    onClick={() =>
                      toast.info(`${encoder.name} 参数设置稍后接入`)
                    }
                  >
                    <Settings2Icon className="size-4" />
                    <span className="sr-only">配置 {encoder.name}</span>
                  </Button>
                </div>
              ))}
            </div>
          </SettingsBlock>

          <SettingsBlock
            title="高级"
            description="控制转码工作目录、资源占用、字幕实时处理和输出分辨率。"
          >
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="transcoding-temp-path">
                  转码临时路径
                </FieldLabel>
                <div className="flex gap-2">
                  <Input
                    id="transcoding-temp-path"
                    value={draft.transcodingTempPath}
                    onChange={(event) =>
                      updateDraft('transcodingTempPath', event.target.value)
                    }
                    placeholder="留空则使用服务器默认转码目录"
                    className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="shrink-0"
                    onClick={() =>
                      toast.info('路径选择器稍后接入服务端目录浏览')
                    }
                  >
                    <FolderOpenIcon className="size-4" />
                    <span className="sr-only">浏览路径</span>
                  </Button>
                </div>
                <FieldDescription>
                  指定转码工作文件目录。建议使用服务器本地高速磁盘上的可写绝对路径。
                </FieldDescription>
              </Field>

              <TranscodingSwitchField
                title="启用限流"
                description="动态调整转码速度，避免后台转码长时间占满 CPU。"
                checked={draft.throttlingEnabled}
                onCheckedChange={(checked) =>
                  updateDraft('throttlingEnabled', checked)
                }
              />

              <Field>
                <FieldLabel htmlFor="audio-boost">缩混时音频增强</FieldLabel>
                <Input
                  id="audio-boost"
                  value={draft.audioBoost}
                  onChange={(event) =>
                    updateDraft('audioBoost', event.target.value)
                  }
                  inputMode="decimal"
                  className="border-border/60 bg-background text-foreground md:max-w-40"
                />
                <FieldDescription>
                  设置音频降混时的增益。当前建议值为 2。
                </FieldDescription>
              </Field>

              <TranscodingSwitchField
                title="允许实时提取字幕"
                description="从视频中提取内嵌字幕并以文本形式传给客户端，部分视频可避免转码。"
                checked={draft.subtitleExtractionEnabled}
                onCheckedChange={(checked) =>
                  updateDraft('subtitleExtractionEnabled', checked)
                }
              />

              <TranscodingSwitchField
                title="允许实时提取字幕字体"
                description="支持自定义字幕字体；提取过程可能耗时，并在弱设备上造成播放卡顿。"
                checked={draft.subtitleFontExtractionEnabled}
                onCheckedChange={(checked) =>
                  updateDraft('subtitleFontExtractionEnabled', checked)
                }
              />

              <TranscodingSwitchField
                title="启用 HEVC 视频编码（实验性）"
                description="允许使用 HEVC 编码器进行视频转码。开启前请确认客户端兼容性。"
                checked={draft.hevcEncodingEnabled}
                onCheckedChange={(checked) =>
                  updateDraft('hevcEncodingEnabled', checked)
                }
              />

              <Field>
                <FieldLabel>最大转码分辨率</FieldLabel>
                <Select
                  value={draft.maxResolution}
                  onValueChange={(value) => updateDraft('maxResolution', value)}
                >
                  <SelectTrigger className="w-full border-border/60 bg-background text-foreground md:max-w-sm">
                    <SelectValue placeholder="选择最大分辨率" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="unlimited">无限制</SelectItem>
                    <SelectItem value="2160p">4K / 2160p</SelectItem>
                    <SelectItem value="1080p">1080p</SelectItem>
                    <SelectItem value="720p">720p</SelectItem>
                    <SelectItem value="480p">480p</SelectItem>
                  </SelectContent>
                </Select>
                <FieldDescription>
                  限制所有视频转码的最高输出分辨率。
                </FieldDescription>
              </Field>
            </FieldGroup>
          </SettingsBlock>
        </TabsContent>

        <TabsContent value="tone-mapping" className="mt-0 space-y-4">
          <SettingsBlock
            title="色调映射"
            description="在 HDR 转 SDR 或其他色彩空间时保持正确颜色和亮度。"
          >
            <div className="flex items-start gap-3 rounded-[1.15rem] border border-amber-400/25 bg-amber-400/10 px-4 py-3 text-sm leading-6 text-amber-100">
              <InfoIcon className="mt-0.5 size-4 shrink-0" />
              <span>
                如果不执行色调映射，HDR 内容在 SDR
                屏幕上可能显得暗淡，并出现饱和度降低。
              </span>
            </div>

            <TranscodingSwitchField
              title="启用软件色调映射"
              description="让 CPU 在软件中执行色调映射。软件色调映射比硬件加速色调映射更慢，并需要更强 CPU。"
              checked={draft.softwareToneMappingEnabled}
              onCheckedChange={(checked) =>
                updateDraft('softwareToneMappingEnabled', checked)
              }
            />
          </SettingsBlock>
        </TabsContent>

        <div className="sticky bottom-4 z-10 flex justify-end">
          <Button
            type="submit"
            size="lg"
            className="min-w-36 rounded-full bg-emerald-600 px-8 text-white shadow-lg shadow-emerald-950/20 hover:bg-emerald-700"
          >
            保存
          </Button>
        </div>
      </Tabs>
    </form>
  )
}

function SettingsBlock({
  title,
  description,
  children,
}: {
  title: string
  description: string
  children: React.ReactNode
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle className="text-lg">{title}</CardTitle>
        <p className="text-sm leading-6 text-muted-foreground">{description}</p>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-5 px-5 py-5">{children}</CardContent>
    </Card>
  )
}

function TranscodingSwitchField({
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
