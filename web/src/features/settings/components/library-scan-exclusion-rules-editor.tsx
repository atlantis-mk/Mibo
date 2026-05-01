import { PlusIcon, Trash2Icon } from 'lucide-react'

import { Button } from '#/components/ui/button'
import { Field, FieldLabel } from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '#/components/ui/select'
import { Switch } from '#/components/ui/switch'
import type { ScanExclusionRule, ScanExclusionRuleInput } from '#/lib/mibo-api'

export type LibraryScanExclusionRuleDraft = ScanExclusionRuleInput & {
  id?: number
}

export const EMPTY_SCAN_EXCLUSION_RULE_DRAFT: LibraryScanExclusionRuleDraft = {
  name: '',
  description: '',
  rule_type: 'filename_token',
  value: '',
  reason: 'advertisement',
  enabled: true,
}

export function buildScanExclusionRuleDraft(
  rule: ScanExclusionRule,
): LibraryScanExclusionRuleDraft {
  return {
    id: rule.id,
    library_id: rule.library_id,
    name: rule.name,
    description: rule.description,
    rule_type: rule.rule_type,
    value: rule.value,
    reason: rule.reason,
    enabled: rule.enabled,
  }
}

export function normalizeScanExclusionRuleDrafts(
  rules: LibraryScanExclusionRuleDraft[],
): ScanExclusionRuleInput[] {
  return rules
    .map((rule) => ({
      library_id: rule.library_id,
      name: rule.name.trim(),
      description: rule.description?.trim(),
      rule_type: rule.rule_type,
      value: rule.value.trim(),
      reason: rule.reason,
      enabled: rule.enabled,
    }))
    .filter((rule) => rule.name && rule.value)
}

export function LibraryScanExclusionRulesEditor({
  rules,
  onChange,
  disabled = false,
}: {
  rules: LibraryScanExclusionRuleDraft[]
  onChange: (rules: LibraryScanExclusionRuleDraft[]) => void
  disabled?: boolean
}) {
  const addRule = () => {
    onChange([...rules, { ...EMPTY_SCAN_EXCLUSION_RULE_DRAFT }])
  }

  const updateRule = (
    index: number,
    patch: Partial<LibraryScanExclusionRuleDraft>,
  ) => {
    onChange(
      rules.map((rule, currentIndex) =>
        currentIndex === index ? { ...rule, ...patch } : rule,
      ),
    )
  }

  const removeRule = (index: number) => {
    onChange(rules.filter((_, currentIndex) => currentIndex !== index))
  }

  return (
    <div className="grid gap-3">
      {rules.length ? (
        rules.map((rule, index) => (
          <div
            key={rule.id ?? index}
            className="grid gap-3 rounded-xl border border-border/60 p-3"
          >
            <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_160px]">
              <Field>
                <FieldLabel>规则名称</FieldLabel>
                <Input
                  value={rule.name}
                  disabled={disabled}
                  onChange={(event) =>
                    updateRule(index, { name: event.target.value })
                  }
                  placeholder="例如：跳过 promo 文件"
                />
              </Field>
              <Field>
                <FieldLabel>状态</FieldLabel>
                <div className="flex h-9 items-center justify-between rounded-md border border-input bg-background px-3 text-sm">
                  <span>{rule.enabled ? '生效' : '停用'}</span>
                  <Switch
                    checked={rule.enabled ?? true}
                    disabled={disabled}
                    onCheckedChange={(enabled) =>
                      updateRule(index, { enabled })
                    }
                  />
                </div>
              </Field>
            </div>
            <div className="grid gap-3 md:grid-cols-3">
              <Field>
                <FieldLabel>规则类型</FieldLabel>
                <Select
                  value={rule.rule_type}
                  disabled={disabled}
                  onValueChange={(ruleType) =>
                    updateRule(index, {
                      rule_type: ruleType as ScanExclusionRule['rule_type'],
                    })
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="filename_token">文件名 token</SelectItem>
                    <SelectItem value="directory_segment">目录段</SelectItem>
                    <SelectItem value="path_pattern">路径模式</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
              <Field>
                <FieldLabel>匹配值</FieldLabel>
                <Input
                  value={rule.value}
                  disabled={disabled}
                  onChange={(event) =>
                    updateRule(index, { value: event.target.value })
                  }
                  placeholder={ruleValuePlaceholder(rule.rule_type)}
                />
              </Field>
              <Field>
                <FieldLabel>原因</FieldLabel>
                <Select
                  value={rule.reason}
                  disabled={disabled}
                  onValueChange={(reason) => updateRule(index, { reason })}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="advertisement">广告</SelectItem>
                    <SelectItem value="unwanted">不需要</SelectItem>
                    <SelectItem value="duplicate">重复导入</SelectItem>
                    <SelectItem value="wrong_import">误导入</SelectItem>
                    <SelectItem value="other">其他</SelectItem>
                  </SelectContent>
                </Select>
              </Field>
            </div>
            <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
              <Field>
                <FieldLabel>描述</FieldLabel>
                <Input
                  value={rule.description ?? ''}
                  disabled={disabled}
                  onChange={(event) =>
                    updateRule(index, { description: event.target.value })
                  }
                  placeholder="说明这条规则适用的命名习惯"
                />
              </Field>
              <Button
                type="button"
                variant="outline"
                disabled={disabled}
                onClick={() => removeRule(index)}
              >
                <Trash2Icon className="size-4" />
                删除
              </Button>
            </div>
          </div>
        ))
      ) : (
        <div className="rounded-xl border border-dashed border-border/70 p-4 text-sm text-muted-foreground">
          暂无媒体库专属排除规则。
        </div>
      )}
      <Button
        type="button"
        variant="outline"
        className="justify-self-start"
        disabled={disabled}
        onClick={addRule}
      >
        <PlusIcon className="size-4" />
        添加排除规则
      </Button>
    </div>
  )
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
