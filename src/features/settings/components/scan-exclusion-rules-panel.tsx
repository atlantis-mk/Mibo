import { useState, type FormEvent, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangleIcon,
  Loader2Icon,
  PencilIcon,
  PlusIcon,
  ShieldCheckIcon,
  Trash2Icon,
} from 'lucide-react'
import type {
  Library,
  ScanExclusionRule,
  ScanExclusionRuleInput,
} from '@/lib/mibo-api'
import {
  createAuthedMiboApi,
  librariesQueryOptions,
  miboQueryKeys,
  scanExclusionRulesQueryOptions,
} from '@/lib/mibo-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'

type RuleFormState = ScanExclusionRuleInput

const emptyRuleForm: RuleFormState = {
  library_id: undefined,
  name: '',
  description: '',
  rule_type: 'filename_token',
  value: '',
  reason: 'advertisement',
  enabled: true,
}

export function ScanExclusionRulesPanel({ token }: { token: string | null }) {
  const queryClient = useQueryClient()
  const queryToken = token ?? 'guest'
  const [form, setForm] = useState<RuleFormState>(emptyRuleForm)
  const [editingRuleId, setEditingRuleId] = useState<number | null>(null)
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const rulesQuery = useQuery({
    ...scanExclusionRulesQueryOptions(queryToken),
    enabled: Boolean(token),
  })

  const librariesQuery = useQuery({
    ...librariesQueryOptions(queryToken),
    enabled: Boolean(token),
  })

  const invalidateRules = async () => {
    if (!token) return
    await queryClient.invalidateQueries({
      queryKey: miboQueryKeys.scanExclusionRules(queryToken),
    })
  }

  const saveMutation = useMutation({
    mutationFn: async (input: RuleFormState) => {
      if (!token) throw new Error('当前未登录，无法保存自动规则。')
      const api = createAuthedMiboApi(token)
      const payload = normalizeRuleInput(input)
      if (editingRuleId) {
        return api.updateScanExclusionRule(editingRuleId, payload)
      }
      return api.createScanExclusionRule(payload)
    },
    onSuccess: async () => {
      setForm(emptyRuleForm)
      setEditingRuleId(null)
      setRuleDialogOpen(false)
      setFormError(null)
      await invalidateRules()
    },
  })

  const toggleMutation = useMutation({
    mutationFn: async (input: { id: number; enabled: boolean }) => {
      if (!token) throw new Error('当前未登录，无法更新自动规则。')
      return createAuthedMiboApi(token).setScanExclusionRuleEnabled(
        input.id,
        input.enabled
      )
    },
    onSuccess: invalidateRules,
  })

  const deleteMutation = useMutation({
    mutationFn: async (ruleId: number) => {
      if (!token) throw new Error('当前未登录，无法删除自动规则。')
      return createAuthedMiboApi(token).deleteScanExclusionRule(ruleId)
    },
    onSuccess: invalidateRules,
  })

  const rules = rulesQuery.data ?? []
  const libraries = librariesQuery.data ?? []
  const editingRule = editingRuleId
    ? rules.find((rule) => rule.id === editingRuleId)
    : null

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const validation = validateRuleForm(form)
    if (validation) {
      setFormError(validation)
      return
    }
    saveMutation.mutate(form)
  }

  const startEditing = (rule: ScanExclusionRule) => {
    setEditingRuleId(rule.id)
    setForm({
      library_id: rule.library_id,
      name: rule.name,
      description: rule.description,
      rule_type: rule.rule_type,
      value: rule.value,
      reason: rule.reason,
      enabled: rule.enabled,
    })
    setFormError(null)
    setRuleDialogOpen(true)
  }

  const startCreating = () => {
    setEditingRuleId(null)
    setForm(emptyRuleForm)
    setFormError(null)
    saveMutation.reset()
    setRuleDialogOpen(true)
  }

  const cancelEditing = () => {
    setEditingRuleId(null)
    setForm(emptyRuleForm)
    setFormError(null)
    saveMutation.reset()
    setRuleDialogOpen(false)
  }

  return (
    <section className='flex min-h-0 flex-1 flex-col overflow-hidden'>
      <div className='mb-5 flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between'>
        <div className='space-y-2'>
          <div className='flex flex-wrap items-center gap-2'>
            <Badge variant='outline' className='gap-1.5 bg-background/70'>
              <ShieldCheckIcon className='size-3.5 text-emerald-500' />
              {rules.length} 条自动规则
            </Badge>
            <Badge variant='outline' className='bg-background/70'>
              {rules.filter((rule) => rule.enabled).length} 条生效
            </Badge>
          </div>
          <div>
            <h3 className='text-base font-medium'>自动扫描规则</h3>
          </div>
        </div>
        <Button onClick={startCreating} disabled={!token}>
          <PlusIcon className='size-4' />
          新增规则
        </Button>
      </div>

      <Dialog
        open={ruleDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            cancelEditing()
            return
          }
          setRuleDialogOpen(true)
        }}
      >
        <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-2xl'>
          <DialogHeader>
            <DialogTitle>{editingRuleId ? '编辑规则' : '新增规则'}</DialogTitle>
            <DialogDescription>
              规则保存后会立即用于后续新扫描，无需重启服务。
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className='space-y-4'>
            <div className='grid gap-3 lg:grid-cols-2'>
              <RuleField label='名称'>
                <Input
                  value={form.name}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      name: event.currentTarget.value,
                    }))
                  }
                  placeholder='例如：跳过 promo 文件'
                />
              </RuleField>
              <RuleField label='作用范围'>
                <NativeSelect
                  value={form.library_id ? String(form.library_id) : 'global'}
                  disabled={Boolean(editingRule?.system)}
                  onChange={(event) => {
                    const value = event.currentTarget.value
                    setForm((current) => ({
                      ...current,
                      library_id:
                        value === 'global' ? undefined : Number.parseInt(value),
                    }))
                  }}
                >
                  <NativeSelectOption value='global'>全局</NativeSelectOption>
                  {libraries.map((library) => (
                    <NativeSelectOption
                      key={library.id}
                      value={String(library.id)}
                    >
                      {library.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </RuleField>
              <RuleField label='规则类型'>
                <NativeSelect
                  value={form.rule_type}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      rule_type: event.currentTarget
                        .value as ScanExclusionRule['rule_type'],
                    }))
                  }
                >
                  <NativeSelectOption value='filename_token'>
                    文件名 token
                  </NativeSelectOption>
                  <NativeSelectOption value='directory_segment'>
                    目录段
                  </NativeSelectOption>
                  <NativeSelectOption value='path_pattern'>
                    路径模式
                  </NativeSelectOption>
                </NativeSelect>
              </RuleField>
              <RuleField label='匹配值'>
                <Input
                  value={form.value}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      value: event.currentTarget.value,
                    }))
                  }
                  placeholder={ruleValuePlaceholder(form.rule_type)}
                />
              </RuleField>
              <RuleField label='原因'>
                <NativeSelect
                  value={form.reason}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      reason: event.currentTarget.value,
                    }))
                  }
                >
                  <NativeSelectOption value='advertisement'>
                    广告
                  </NativeSelectOption>
                  <NativeSelectOption value='unwanted'>
                    不需要
                  </NativeSelectOption>
                  <NativeSelectOption value='duplicate'>
                    重复导入
                  </NativeSelectOption>
                  <NativeSelectOption value='wrong_import'>
                    误导入
                  </NativeSelectOption>
                  <NativeSelectOption value='other'>其他</NativeSelectOption>
                </NativeSelect>
              </RuleField>
              <RuleField label='描述'>
                <Textarea
                  value={form.description}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      description: event.currentTarget.value,
                    }))
                  }
                  placeholder='说明这条规则适用的命名习惯'
                />
              </RuleField>
              <RuleField label='状态'>
                <div className='flex items-center justify-between rounded-[1rem] border border-border/60 bg-background/60 px-4 py-3 text-sm'>
                  <span className='font-medium text-foreground'>
                    {form.enabled ? '生效' : '停用'}
                  </span>
                  <Switch
                    checked={form.enabled}
                    onCheckedChange={(enabled) =>
                      setForm((current) => ({
                        ...current,
                        enabled,
                      }))
                    }
                  />
                </div>
              </RuleField>
            </div>
            {formError || saveMutation.error ? (
              <RuleError
                message={formError ?? errorMessage(saveMutation.error)}
              />
            ) : null}
            <DialogFooter className='mt-4'>
              <Button type='button' variant='outline' onClick={cancelEditing}>
                取消
              </Button>
              <Button type='submit' disabled={!token || saveMutation.isPending}>
                {saveMutation.isPending ? (
                  <Loader2Icon className='size-4 animate-spin' />
                ) : null}
                {editingRuleId ? '保存规则' : '创建规则'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {toggleMutation.error || deleteMutation.error ? (
        <RuleError
          message={errorMessage(toggleMutation.error ?? deleteMutation.error)}
        />
      ) : null}

      {rulesQuery.isLoading ? (
        <div className='grid gap-4 xl:grid-cols-2'>
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className='h-48 rounded-[1.35rem]' />
          ))}
        </div>
      ) : rulesQuery.isError ? (
        <RuleError message='无法加载自动规则，请稍后重试。' />
      ) : rules.length === 0 ? (
        <div className='rounded-[1.35rem] border border-dashed border-border/70 bg-background/60 p-8 text-center text-sm text-muted-foreground'>
          暂无自动规则。新增规则后，后续扫描会使用这些规则跳过匹配视频。
        </div>
      ) : (
        <RulesTable
          rules={rules}
          libraries={libraries}
          pending={toggleMutation.isPending || deleteMutation.isPending}
          onEdit={startEditing}
          onToggle={(rule) =>
            toggleMutation.mutate({ id: rule.id, enabled: !rule.enabled })
          }
          onDelete={(rule) => deleteMutation.mutate(rule.id)}
        />
      )}
    </section>
  )
}

function RulesTable({
  rules,
  libraries,
  pending,
  onEdit,
  onToggle,
  onDelete,
}: {
  rules: ScanExclusionRule[]
  libraries: Library[]
  pending: boolean
  onEdit: (rule: ScanExclusionRule) => void
  onToggle: (rule: ScanExclusionRule) => void
  onDelete: (rule: ScanExclusionRule) => void
}) {
  return (
    <div className='flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.35rem] border border-border/60 bg-background/80 shadow-sm'>
      <div className='min-h-0 flex-1 overflow-auto'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className='min-w-52'>规则</TableHead>
              <TableHead>类型</TableHead>
              <TableHead className='min-w-44'>匹配值</TableHead>
              <TableHead>原因</TableHead>
              <TableHead>范围</TableHead>
              <TableHead>来源</TableHead>
              <TableHead>状态</TableHead>
              <TableHead className='text-right'>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rules.map((rule) => (
              <TableRow key={rule.id}>
                <TableCell className='max-w-72'>
                  <div className='space-y-1'>
                    <div className='truncate font-medium' title={rule.name}>
                      {rule.name}
                    </div>
                    <div
                      className='truncate text-xs text-muted-foreground'
                      title={rule.description || rule.key}
                    >
                      {rule.description || rule.key}
                    </div>
                  </div>
                </TableCell>
                <TableCell>{ruleTypeLabel(rule.rule_type)}</TableCell>
                <TableCell className='max-w-64 truncate font-mono text-xs'>
                  {rule.value}
                </TableCell>
                <TableCell>{reasonLabel(rule.reason)}</TableCell>
                <TableCell>
                  <Badge variant={rule.library_id ? 'secondary' : 'outline'}>
                    {ruleScopeLabel(rule, libraries)}
                  </Badge>
                </TableCell>
                <TableCell>
                  {rule.system ? (
                    <Badge variant='outline'>系统</Badge>
                  ) : (
                    <Badge variant='secondary'>自定义</Badge>
                  )}
                </TableCell>
                <TableCell>
                  <Badge variant={rule.enabled ? 'default' : 'outline'}>
                    {rule.enabled ? '生效中' : '已停用'}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className='flex justify-end gap-2'>
                    <Button
                      size='sm'
                      variant='outline'
                      disabled={pending}
                      onClick={() => onEdit(rule)}
                    >
                      <PencilIcon className='size-4' />
                      编辑
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      disabled={pending}
                      onClick={() => onToggle(rule)}
                    >
                      {rule.enabled ? '停用' : '启用'}
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      disabled={pending || rule.system}
                      onClick={() => onDelete(rule)}
                    >
                      <Trash2Icon className='size-4' />
                      删除
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function RuleField({
  label,
  children,
}: {
  label: string
  children: ReactNode
}) {
  return (
    <div className='space-y-2'>
      <Label>{label}</Label>
      {children}
    </div>
  )
}

function RuleError({ message }: { message: string }) {
  return (
    <div className='mt-3 flex items-start gap-3 rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive'>
      <AlertTriangleIcon className='mt-0.5 size-4 shrink-0' />
      <span>{message}</span>
    </div>
  )
}

function normalizeRuleInput(input: RuleFormState): ScanExclusionRuleInput {
  return {
    library_id: input.library_id,
    name: input.name.trim(),
    description: input.description?.trim(),
    rule_type: input.rule_type,
    value: input.value.trim(),
    reason: input.reason,
    enabled: input.enabled,
  }
}

function ruleScopeLabel(rule: ScanExclusionRule, libraries: Library[]) {
  if (!rule.library_id) return '全局'
  const library = libraries.find((item) => item.id === rule.library_id)
  return library?.name ?? `媒体库 #${rule.library_id}`
}

function validateRuleForm(input: RuleFormState) {
  if (!input.name.trim()) return '请填写规则名称。'
  if (!input.value.trim()) return '请填写匹配值。'
  if (
    input.rule_type === 'path_pattern' &&
    ['*', '/*', '/**', '**'].includes(input.value.trim())
  ) {
    return '路径模式过于宽泛，请缩小匹配范围。'
  }
  return null
}

function ruleValuePlaceholder(ruleType: ScanExclusionRule['rule_type']) {
  switch (ruleType) {
    case 'filename_token':
      return '例如：promo'
    case 'directory_segment':
      return '例如：ads'
    case 'path_pattern':
      return '例如：/movies/*/promo.mkv'
  }
}

function ruleTypeLabel(ruleType: ScanExclusionRule['rule_type']) {
  switch (ruleType) {
    case 'filename_token':
      return '文件名 token'
    case 'directory_segment':
      return '目录段'
    case 'path_pattern':
      return '路径模式'
  }
}

function reasonLabel(reason: string) {
  switch (reason) {
    case 'advertisement':
      return '广告'
    case 'unwanted':
      return '不需要'
    case 'duplicate':
      return '重复导入'
    case 'wrong_import':
      return '误导入'
    case 'other':
      return '其他'
    default:
      return reason || '未知'
  }
}

function errorMessage(error: unknown) {
  if (error instanceof Error) return error.message
  return '操作失败，请稍后重试。'
}
