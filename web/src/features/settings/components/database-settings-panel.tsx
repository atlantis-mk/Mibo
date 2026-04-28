import { useEffect, useState } from 'react'
import { AlertTriangleIcon } from 'lucide-react'
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
import { Separator } from '#/components/ui/separator'
import { Switch } from '#/components/ui/switch'

const DATABASE_SETTINGS_STORAGE_KEY = 'mibo-web-database-settings'

type DatabaseSettingsForm = {
  cacheSizeMb: string
  optimizeOnShutdown: boolean
  analyzeRowLimit: string
  cleanupOnNextStartup: boolean
}

const defaultDatabaseSettings: DatabaseSettingsForm = {
  cacheSizeMb: '128',
  optimizeOnShutdown: true,
  analyzeRowLimit: '400',
  cleanupOnNextStartup: true,
}

export function DatabaseSettingsPanel() {
  const [draft, setDraft] = useState<DatabaseSettingsForm>(
    defaultDatabaseSettings,
  )

  useEffect(() => {
    const savedSettings = window.localStorage.getItem(
      DATABASE_SETTINGS_STORAGE_KEY,
    )

    if (!savedSettings) {
      return
    }

    try {
      setDraft({
        ...defaultDatabaseSettings,
        ...(JSON.parse(savedSettings) as Partial<DatabaseSettingsForm>),
      })
    } catch {
      window.localStorage.removeItem(DATABASE_SETTINGS_STORAGE_KEY)
    }
  }, [])

  function updateDraft<Value extends keyof DatabaseSettingsForm>(
    key: Value,
    value: DatabaseSettingsForm[Value],
  ) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    window.localStorage.setItem(
      DATABASE_SETTINGS_STORAGE_KEY,
      JSON.stringify(draft),
    )
    toast.success('数据库设置已保存')
  }

  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle className="text-xl">数据库</CardTitle>
        <CardDescription>
          调整数据库缓存、关闭时优化和下次启动清理策略。
        </CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="space-y-5 px-5 py-5">
        <div className="flex items-start gap-3 rounded-[1.15rem] border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm leading-6 text-amber-900 dark:text-amber-200">
          <AlertTriangleIcon className="mt-0.5 size-4 shrink-0" />
          <span>
            这些是高级数据库选项。除非你正在排查性能、数据库体积或维护窗口问题，否则建议保留默认值。
          </span>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <FieldGroup>
            <NumberField
              id="database-cache-size"
              label="数据库缓存尺寸（MB）"
              value={draft.cacheSizeMb}
              min="1"
              onChange={(value) => updateDraft('cacheSizeMb', value)}
              description="控制每个数据库文件在内存中一次性保留的数据量。值越大可能提升服务器性能，需要下次重启后生效。"
            />

            <FormSwitchField
              title="尝试在服务器关闭时优化数据库"
              description="服务器关闭时尝试执行数据库优化。通常无需人工干预；如优化有利于查询规划器，可能会执行分析任务，并让关闭耗时变长。"
              checked={draft.optimizeOnShutdown}
              onCheckedChange={(checked) =>
                updateDraft('optimizeOnShutdown', checked)
              }
            />

            <NumberField
              id="database-analyze-row-limit"
              label="分析行限制数"
              value={draft.analyzeRowLimit}
              min="0"
              onChange={(value) => updateDraft('analyzeRowLimit', value)}
              description="控制数据库优化时 ANALYZE 命令在每个索引中检查的大致行数。值越大，优化可能更有效，但也可能让关闭更慢。"
            />

            <FormSwitchField
              title="下次服务器启动时清理数据库"
              description="下次启动时重建并压缩数据库文件，把数据库重新打包到更小磁盘空间中。清理期间服务器不可用，也无法查看进度；不要强制关闭服务器，否则可能损坏数据库。该操作执行一次后会自动恢复为未选中。"
              checked={draft.cleanupOnNextStartup}
              onCheckedChange={(checked) =>
                updateDraft('cleanupOnNextStartup', checked)
              }
            />
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
  )
}

function NumberField({
  id,
  label,
  value,
  min,
  onChange,
  description,
}: {
  id: string
  label: string
  value: string
  min: string
  onChange: (value: string) => void
  description: string
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        type="number"
        min={min}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="border-border/60 bg-background text-foreground placeholder:text-muted-foreground md:max-w-md"
      />
      <FieldDescription>{description}</FieldDescription>
    </Field>
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
