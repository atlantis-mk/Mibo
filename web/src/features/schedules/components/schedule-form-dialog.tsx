import { useEffect, useState } from 'react'

import { Button } from '#/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '#/components/ui/dialog'
import { Field, FieldGroup, FieldLabel } from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '#/components/ui/select'
import {
  type Library,
  type Schedule,
  type ScheduleMutationInput,
  type ScheduleFrequencyKind,
  type ScheduleScopeKind,
} from '#/lib/mibo-api'

type Props = {
  libraries: Library[]
  onOpenChange: (open: boolean) => void
  onSubmit: (input: ScheduleMutationInput) => Promise<unknown>
  open: boolean
  schedule?: Schedule | null
}

type FormState = {
  name: string
  kind: string
  scope_kind: ScheduleScopeKind
  library_id: string
  enabled: boolean
  frequency_kind: ScheduleFrequencyKind
  time_of_day: string
  weekday: string
  day_of_month: string
}

const defaultForm: FormState = {
  name: '',
  kind: 'scan',
  scope_kind: 'global',
  library_id: '',
  enabled: true,
  frequency_kind: 'daily',
  time_of_day: '09:00',
  weekday: '1',
  day_of_month: '1',
}

export function ScheduleFormDialog({
  libraries,
  onOpenChange,
  onSubmit,
  open,
  schedule,
}: Props) {
  const [form, setForm] = useState(defaultForm)
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (!schedule) {
      setForm(defaultForm)
      return
    }
    setForm({
      name: schedule.name,
      kind: schedule.kind,
      scope_kind: schedule.scope_kind,
      library_id: schedule.library_id ? String(schedule.library_id) : '',
      enabled: schedule.enabled,
      frequency_kind: schedule.frequency.kind,
      time_of_day: schedule.frequency.time_of_day,
      weekday: String(schedule.frequency.weekday ?? 1),
      day_of_month: String(schedule.frequency.day_of_month ?? 1),
    })
  }, [schedule])

  async function handleSubmit() {
    setIsSubmitting(true)
    try {
      await onSubmit({
        name: form.name,
        kind: form.kind,
        scope_kind: form.scope_kind,
        library_id:
          form.scope_kind === 'library' && form.library_id
            ? Number(form.library_id)
            : undefined,
        enabled: form.enabled,
        frequency: {
          kind: form.frequency_kind,
          time_of_day: form.time_of_day,
          weekday:
            form.frequency_kind === 'weekly' ? Number(form.weekday) : undefined,
          day_of_month:
            form.frequency_kind === 'monthly'
              ? Number(form.day_of_month)
              : undefined,
        },
      })
      onOpenChange(false)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>
            {schedule ? '编辑计划任务' : '创建计划任务'}
          </DialogTitle>
          <DialogDescription>
            使用 daily / weekly / monthly 模板配置 recurring
            maintenance，不直接暴露 cron 文本。
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel>任务名称</FieldLabel>
            <Input
              value={form.name}
              onChange={(event) =>
                setForm((current) => ({ ...current, name: event.target.value }))
              }
            />
          </Field>

          <div className="grid gap-4 md:grid-cols-2">
            <Field>
              <FieldLabel>任务类型</FieldLabel>
              <Select
                value={form.kind}
                onValueChange={(value) =>
                  setForm((current) => ({ ...current, kind: value }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择任务类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="scan">媒体扫描</SelectItem>
                  <SelectItem value="library_cleanup">库清理</SelectItem>
                  <SelectItem value="invalid_link_check">
                    失效链接检查
                  </SelectItem>
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel>目标范围</FieldLabel>
              <Select
                value={form.scope_kind}
                onValueChange={(value: 'global' | 'library') =>
                  setForm((current) => ({ ...current, scope_kind: value }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择范围" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="global">全局范围</SelectItem>
                  <SelectItem value="library">单媒体库</SelectItem>
                </SelectContent>
              </Select>
            </Field>
          </div>

          {form.scope_kind === 'library' ? (
            <Field>
              <FieldLabel>媒体库</FieldLabel>
              <Select
                value={form.library_id}
                onValueChange={(value) =>
                  setForm((current) => ({ ...current, library_id: value }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择媒体库" />
                </SelectTrigger>
                <SelectContent>
                  {libraries.map((library) => (
                    <SelectItem key={library.id} value={String(library.id)}>
                      {library.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
          ) : null}

          <div className="grid gap-4 md:grid-cols-2">
            <Field>
              <FieldLabel>频率模板</FieldLabel>
              <Select
                value={form.frequency_kind}
                onValueChange={(value: 'daily' | 'weekly' | 'monthly') =>
                  setForm((current) => ({ ...current, frequency_kind: value }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择频率" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="daily">每天</SelectItem>
                  <SelectItem value="weekly">每周</SelectItem>
                  <SelectItem value="monthly">每月</SelectItem>
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel>时间</FieldLabel>
              <Input
                type="time"
                value={form.time_of_day}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    time_of_day: event.target.value,
                  }))
                }
              />
            </Field>
          </div>

          {form.frequency_kind === 'weekly' ? (
            <Field>
              <FieldLabel>星期</FieldLabel>
              <Select
                value={form.weekday}
                onValueChange={(value) =>
                  setForm((current) => ({ ...current, weekday: value }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择星期" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="0">周日</SelectItem>
                  <SelectItem value="1">周一</SelectItem>
                  <SelectItem value="2">周二</SelectItem>
                  <SelectItem value="3">周三</SelectItem>
                  <SelectItem value="4">周四</SelectItem>
                  <SelectItem value="5">周五</SelectItem>
                  <SelectItem value="6">周六</SelectItem>
                </SelectContent>
              </Select>
            </Field>
          ) : null}

          {form.frequency_kind === 'monthly' ? (
            <Field>
              <FieldLabel>每月日期</FieldLabel>
              <Input
                type="number"
                min={1}
                max={31}
                value={form.day_of_month}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    day_of_month: event.target.value,
                  }))
                }
              />
            </Field>
          ) : null}
        </FieldGroup>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? '保存中…' : schedule ? '保存修改' : '创建任务'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
