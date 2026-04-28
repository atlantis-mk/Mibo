import { useEffect, useState } from 'react'
import {
  CalendarDaysIcon,
  FilterIcon,
  FolderIcon,
  InfoIcon,
  PlusIcon,
  RadioIcon,
  RefreshCwIcon,
  SaveIcon,
  SlidersHorizontalIcon,
} from 'lucide-react'
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
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '#/components/ui/empty'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '#/components/ui/tabs'
import { Textarea } from '#/components/ui/textarea'

const LIVE_TV_SETTINGS_STORAGE_KEY = 'mibo-web-live-tv-settings'

type LiveTvAdvancedSettings = {
  bufferLimit: string
  guideDays: string
  defaultRecordingFolder: string
  movieRecordingFolder: string
  seriesRecordingFolder: string
  startPaddingMinutes: string
  stopPaddingMinutes: string
  postProcessorApp: string
  postProcessorArguments: string
}

const defaultAdvancedSettings: LiveTvAdvancedSettings = {
  bufferLimit: 'unlimited',
  guideDays: 'auto',
  defaultRecordingFolder: '',
  movieRecordingFolder: '',
  seriesRecordingFolder: '',
  startPaddingMinutes: '0',
  stopPaddingMinutes: '0',
  postProcessorApp: '',
  postProcessorArguments: '',
}

const liveSourceActions = [
  {
    title: '电视源',
    description: '接入 IPTV、HDHomeRun 或其他直播输入源。',
    action: '添加电视输入源',
    icon: RadioIcon,
  },
  {
    title: '指南数据源',
    description: '配置 XMLTV、服务商 EPG 或自定义节目指南。',
    action: '添加节目指南数据源',
    icon: CalendarDaysIcon,
  },
]

export function LiveTvSettingsPanel() {
  const [draft, setDraft] = useState<LiveTvAdvancedSettings>(
    defaultAdvancedSettings,
  )

  useEffect(() => {
    const savedSettings = window.localStorage.getItem(
      LIVE_TV_SETTINGS_STORAGE_KEY,
    )

    if (!savedSettings) {
      return
    }

    try {
      setDraft({
        ...defaultAdvancedSettings,
        ...(JSON.parse(savedSettings) as Partial<LiveTvAdvancedSettings>),
      })
    } catch {
      window.localStorage.removeItem(LIVE_TV_SETTINGS_STORAGE_KEY)
    }
  }, [])

  function updateDraft<Value extends keyof LiveTvAdvancedSettings>(
    key: Value,
    value: LiveTvAdvancedSettings[Value],
  ) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  function handleSaveAdvanced(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    window.localStorage.setItem(
      LIVE_TV_SETTINGS_STORAGE_KEY,
      JSON.stringify(draft),
    )
    toast.success('电视直播高级设置已保存')
  }

  function handlePlaceholderAction(label: string) {
    toast.info(`${label} 将在后端直播源能力接入后启用`)
  }

  return (
    <Tabs defaultValue="settings" className="space-y-4">
      <div className="flex justify-center">
        <TabsList className="grid w-full max-w-md grid-cols-3">
          <TabsTrigger value="settings">设置</TabsTrigger>
          <TabsTrigger value="channels">频道</TabsTrigger>
          <TabsTrigger value="advanced">高级</TabsTrigger>
        </TabsList>
      </div>

      <TabsContent value="settings" className="mt-0 space-y-4">
        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
          <CardHeader className="px-5 py-5">
            <CardTitle className="text-xl">直播数据来源</CardTitle>
            <CardDescription>
              添加电视输入源与节目指南数据源后，Mibo 会在这里汇总可播放频道。
            </CardDescription>
          </CardHeader>
          <Separator className="bg-border" />
          <CardContent className="space-y-4 px-5 py-5">
            {liveSourceActions.map((item) => {
              const Icon = item.icon

              return (
                <div
                  key={item.title}
                  className="flex flex-col gap-4 rounded-[1.25rem] border border-border/60 bg-background/70 p-4 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex items-start gap-3">
                    <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-emerald-600/10 text-emerald-600">
                      <Icon className="size-5" />
                    </div>
                    <div className="min-w-0">
                      <h3 className="font-medium">{item.title}</h3>
                      <p className="mt-1 text-sm leading-6 text-muted-foreground">
                        {item.description}
                      </p>
                    </div>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    className="justify-start sm:justify-center"
                    onClick={() => handlePlaceholderAction(item.action)}
                  >
                    <PlusIcon className="size-4" />
                    {item.action}
                  </Button>
                </div>
              )
            })}

            <div className="flex flex-col gap-3 rounded-[1.25rem] border border-dashed border-border/70 bg-muted/20 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 className="font-medium">刷新指南数据</h3>
                <p className="mt-1 text-sm leading-6 text-muted-foreground">
                  手动拉取最新频道和 EPG
                  数据，适合新增源或排查节目单缺失时使用。
                </p>
              </div>
              <Button
                type="button"
                className="bg-emerald-600 text-white hover:bg-emerald-700"
                onClick={() => handlePlaceholderAction('刷新指南数据')}
              >
                <RefreshCwIcon className="size-4" />
                刷新指南数据
              </Button>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="channels" className="mt-0 space-y-4">
        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
          <CardHeader className="flex flex-col gap-3 px-5 py-5 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-xl">频道</CardTitle>
              <CardDescription>
                查看和筛选已从直播源识别出的频道列表。
              </CardDescription>
            </div>
            <Button type="button" variant="outline" disabled>
              <FilterIcon className="size-4" />
              筛选
            </Button>
          </CardHeader>
          <Separator className="bg-border" />
          <CardContent className="px-5 py-5">
            <Empty className="min-h-72 border border-dashed border-border/70 bg-muted/20">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <RadioIcon className="size-4" />
                </EmptyMedia>
                <EmptyTitle>未找到项目。</EmptyTitle>
                <EmptyDescription>
                  添加电视输入源并刷新指南数据后，识别出的频道会显示在这里。
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => handlePlaceholderAction('添加电视输入源')}
                >
                  <PlusIcon className="size-4" />
                  添加电视输入源
                </Button>
              </EmptyContent>
            </Empty>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="advanced" className="mt-0 space-y-4">
        <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
          <CardHeader className="px-5 py-5">
            <CardTitle className="text-xl">高级</CardTitle>
            <CardDescription>
              配置直播缓存、指南下载范围、默认录制目录和录制后处理。
            </CardDescription>
          </CardHeader>
          <Separator className="bg-border" />
          <CardContent className="space-y-5 px-5 py-5">
            <div className="flex items-start gap-3 rounded-[1.15rem] border border-border/60 bg-muted/30 px-4 py-3 text-sm leading-6 text-muted-foreground">
              <InfoIcon className="mt-0.5 size-4 shrink-0" />
              <span>
                当前高级设置先保存在本机浏览器，直播源、录制任务和后处理执行会在服务端能力完成后接入。
              </span>
            </div>

            <form onSubmit={handleSaveAdvanced} className="space-y-6">
              <FieldGroup>
                <div className="grid gap-4 md:grid-cols-2">
                  <Field>
                    <FieldLabel>直播流缓冲区尺寸限制</FieldLabel>
                    <Select
                      value={draft.bufferLimit}
                      onValueChange={(value) =>
                        updateDraft('bufferLimit', value)
                      }
                    >
                      <SelectTrigger className="w-full border-border/60 bg-background text-foreground">
                        <SelectValue placeholder="选择缓冲限制" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="unlimited">无限制</SelectItem>
                        <SelectItem value="1">1 小时</SelectItem>
                        <SelectItem value="2">2 小时</SelectItem>
                        <SelectItem value="4">4 小时</SelectItem>
                      </SelectContent>
                    </Select>
                    <FieldDescription>
                      控制直播回看与缓存范围。缓冲越长，占用磁盘越多。
                    </FieldDescription>
                  </Field>

                  <Field>
                    <FieldLabel>指南数据下载天数</FieldLabel>
                    <Select
                      value={draft.guideDays}
                      onValueChange={(value) => updateDraft('guideDays', value)}
                    >
                      <SelectTrigger className="w-full border-border/60 bg-background text-foreground">
                        <SelectValue placeholder="选择下载天数" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="auto">自动</SelectItem>
                        <SelectItem value="1">1 天</SelectItem>
                        <SelectItem value="3">3 天</SelectItem>
                        <SelectItem value="7">7 天</SelectItem>
                        <SelectItem value="14">14 天</SelectItem>
                      </SelectContent>
                    </Select>
                    <FieldDescription>
                      控制 EPG 提前下载范围。部分提供商可能只提供 24 小时。
                    </FieldDescription>
                  </Field>
                </div>

                <RecordingFolderField
                  id="live-tv-default-recording-folder"
                  label="默认录制文件夹"
                  description="保存录制内容的默认媒体库位置，建议使用已创建的混合内容媒体库。"
                  value={draft.defaultRecordingFolder}
                  onChange={(value) =>
                    updateDraft('defaultRecordingFolder', value)
                  }
                />

                <div className="grid gap-4 md:grid-cols-2">
                  <RecordingFolderField
                    id="live-tv-movie-recording-folder"
                    label="影片录制文件夹"
                    description="可选。电影类录制内容会优先保存到这里。"
                    value={draft.movieRecordingFolder}
                    onChange={(value) =>
                      updateDraft('movieRecordingFolder', value)
                    }
                  />
                  <RecordingFolderField
                    id="live-tv-series-recording-folder"
                    label="剧集录制文件夹"
                    description="可选。电视剧和节目类录制内容会优先保存到这里。"
                    value={draft.seriesRecordingFolder}
                    onChange={(value) =>
                      updateDraft('seriesRecordingFolder', value)
                    }
                  />
                </div>

                <div className="rounded-[1.25rem] border border-border/60 bg-muted/20 p-4">
                  <div className="mb-4 flex items-center gap-2">
                    <SlidersHorizontalIcon className="size-4 text-muted-foreground" />
                    <h3 className="font-medium">默认录制设置</h3>
                  </div>
                  <div className="grid gap-4 md:grid-cols-2">
                    <Field>
                      <FieldLabel htmlFor="live-tv-start-padding">
                        随时开始
                      </FieldLabel>
                      <Input
                        id="live-tv-start-padding"
                        type="number"
                        min="0"
                        value={draft.startPaddingMinutes}
                        onChange={(event) =>
                          updateDraft('startPaddingMinutes', event.target.value)
                        }
                        className="border-border/60 bg-background text-foreground"
                      />
                      <FieldDescription>
                        录制开始前提前多少分钟启动。
                      </FieldDescription>
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="live-tv-stop-padding">
                        随时停止
                      </FieldLabel>
                      <Input
                        id="live-tv-stop-padding"
                        type="number"
                        min="0"
                        value={draft.stopPaddingMinutes}
                        onChange={(event) =>
                          updateDraft('stopPaddingMinutes', event.target.value)
                        }
                        className="border-border/60 bg-background text-foreground"
                      />
                      <FieldDescription>
                        录制结束后延后多少分钟停止。
                      </FieldDescription>
                    </Field>
                  </div>
                </div>

                <div className="rounded-[1.25rem] border border-border/60 bg-muted/20 p-4">
                  <div className="mb-4 flex items-center gap-2">
                    <SaveIcon className="size-4 text-muted-foreground" />
                    <h3 className="font-medium">录制后期处理</h3>
                  </div>
                  <FieldGroup>
                    <Field>
                      <FieldLabel htmlFor="live-tv-post-processor-app">
                        后期处理应用程序
                      </FieldLabel>
                      <Input
                        id="live-tv-post-processor-app"
                        value={draft.postProcessorApp}
                        onChange={(event) =>
                          updateDraft('postProcessorApp', event.target.value)
                        }
                        placeholder="/usr/local/bin/process-recording"
                        className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
                      />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="live-tv-post-processor-arguments">
                        后期处理器命令行参数
                      </FieldLabel>
                      <Textarea
                        id="live-tv-post-processor-arguments"
                        value={draft.postProcessorArguments}
                        onChange={(event) =>
                          updateDraft(
                            'postProcessorArguments',
                            event.target.value,
                          )
                        }
                        placeholder="--path {path} --channel {channelname} --number {channelnumber}"
                        className="min-h-28 border-border/60 bg-background font-mono text-sm text-foreground placeholder:text-muted-foreground"
                      />
                      <FieldDescription>
                        支持变量：{'{path}'}、{'{channelname}'}、
                        {'{channelnumber}'}。
                      </FieldDescription>
                    </Field>
                  </FieldGroup>
                </div>
              </FieldGroup>

              <Button
                type="submit"
                size="lg"
                className="w-full bg-emerald-600 text-white hover:bg-emerald-700"
              >
                保存
              </Button>
            </form>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  )
}

function RecordingFolderField({
  id,
  label,
  description,
  value,
  onChange,
}: {
  id: string
  label: string
  description: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <div className="flex gap-2">
        <Input
          id={id}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder="选择或输入录制目录"
          className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground"
        />
        <Button type="button" variant="outline" size="icon" disabled>
          <FolderIcon className="size-4" />
          <span className="sr-only">选择目录</span>
        </Button>
      </div>
      <FieldDescription>{description}</FieldDescription>
    </Field>
  )
}
