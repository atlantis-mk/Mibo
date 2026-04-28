import { Settings2Icon } from 'lucide-react'
import { useState } from 'react'

import { Label } from '#/components/ui/label'
import { Switch } from '#/components/ui/switch'

export function LogSettingsPanel() {
  const [debugLogging, setDebugLogging] = useState(false)
  const [transcodeLogging, setTranscodeLogging] = useState(true)

  return (
    <section className="rounded-[1.75rem] border border-border/60 bg-card/70 p-5 shadow-sm backdrop-blur-sm">
      <div className="flex items-start gap-3">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
          <Settings2Icon className="size-4" />
        </div>
        <div>
          <h3 className="font-semibold">日志设置</h3>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">
            这里预留服务器日志级别、转码日志保留与自动清理策略；当前先保存为本页临时偏好，后续可接入服务端配置。
          </p>
        </div>
      </div>

      <div className="mt-6 space-y-4">
        <SettingRow
          label="启用调试日志"
          description="记录更详细的服务器行为，适合排查问题时短期开启。"
          checked={debugLogging}
          onCheckedChange={setDebugLogging}
        />
        <SettingRow
          label="保留转码日志"
          description="为 ffmpeg 转码任务保留独立日志，便于定位播放和字幕问题。"
          checked={transcodeLogging}
          onCheckedChange={setTranscodeLogging}
        />
      </div>
    </section>
  )
}

function SettingRow({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string
  description: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-2xl bg-muted/55 p-4">
      <div className="min-w-0">
        <Label className="font-medium">{label}</Label>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          {description}
        </p>
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  )
}
