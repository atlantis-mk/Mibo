import type {
  PluginConfigurationField,
  PluginConfigurationSchema,
} from '@/lib/mibo-api'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

const REDACTED_SECRET_SENTINEL = '***REDACTED***'

export function buildPluginConfigurationDefaults(
  schema?: PluginConfigurationSchema,
  current: Record<string, unknown> = {}
) {
  const next = { ...current }

  for (const field of schema?.fields ?? []) {
    if (next[field.key] !== undefined) continue
    if (field.default === undefined) continue
    next[field.key] = field.default
  }

  return next
}

export function PluginConfigurationForm({
  schema,
  value,
  onChange,
  disabled = false,
}: {
  schema?: PluginConfigurationSchema
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
}) {
  const fields = schema?.fields ?? []

  if (!fields.length) {
    return (
      <div className='rounded-[1rem] border border-dashed border-border/60 bg-muted/30 px-4 py-3 text-sm text-muted-foreground'>
        当前插件没有声明可配置字段。
      </div>
    )
  }

  function updateField(key: string, nextValue: unknown) {
    onChange({
      ...value,
      [key]: nextValue,
    })
  }

  return (
    <FieldGroup className='gap-4'>
      {fields.map((field) => (
        <PluginConfigurationFieldControl
          key={field.key}
          field={field}
          value={value[field.key]}
          disabled={disabled}
          onChange={(nextValue) => updateField(field.key, nextValue)}
        />
      ))}
    </FieldGroup>
  )
}

function PluginConfigurationFieldControl({
  field,
  value,
  disabled,
  onChange,
}: {
  field: PluginConfigurationField
  value: unknown
  disabled: boolean
  onChange: (nextValue: unknown) => void
}) {
  const label = field.display?.label?.trim() || field.key
  const description = buildFieldDescription(field)
  const placeholder = field.display?.placeholder?.trim()
  const secretConfigured = isRedactedSecretValue(value)

  return (
    <Field>
      <FieldLabel>
        {label}
        {field.required ? ' *' : ''}
      </FieldLabel>
      {renderFieldControl({
        field,
        value,
        disabled,
        placeholder,
        secretConfigured,
        onChange,
      })}
      {description ? <FieldDescription>{description}</FieldDescription> : null}
      {field.type === 'secret' && secretConfigured ? (
        <FieldDescription>
          已配置的密钥不会回显，留空即可保持现有值。
        </FieldDescription>
      ) : null}
    </Field>
  )
}

function renderFieldControl({
  field,
  value,
  disabled,
  placeholder,
  secretConfigured,
  onChange,
}: {
  field: PluginConfigurationField
  value: unknown
  disabled: boolean
  placeholder?: string
  secretConfigured: boolean
  onChange: (nextValue: unknown) => void
}) {
  switch (field.type) {
    case 'boolean':
      return (
        <div className='flex min-h-10 items-center rounded-md border border-border/60 px-3'>
          <Switch
            checked={Boolean(value)}
            disabled={disabled}
            onCheckedChange={(checked) => onChange(checked)}
          />
        </div>
      )
    case 'select':
      return (
        <Select
          value={typeof value === 'string' ? value : stringValue(field.default)}
          disabled={disabled}
          onValueChange={(nextValue) => onChange(nextValue)}
        >
          <SelectTrigger className='w-full'>
            <SelectValue placeholder={placeholder || '请选择'} />
          </SelectTrigger>
          <SelectContent>
            {(field.options ?? []).map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label?.trim() || option.value}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )
    case 'number':
      return (
        <Input
          type='number'
          inputMode='decimal'
          min={field.minimum}
          max={field.maximum}
          value={numberInputValue(value)}
          disabled={disabled}
          placeholder={placeholder}
          onChange={(event) =>
            onChange(parseOptionalNumber(event.target.value))
          }
        />
      )
    case 'secret':
      return (
        <Input
          type='password'
          value={secretConfigured ? '' : stringValue(value)}
          disabled={disabled}
          placeholder={
            placeholder || (secretConfigured ? '输入新值以替换' : '')
          }
          onChange={(event) => onChange(event.target.value)}
        />
      )
    case 'url':
      return (
        <Input
          type='url'
          value={stringValue(value)}
          disabled={disabled}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
      )
    case 'duration':
    case 'string':
    default:
      return (
        <Input
          type='text'
          value={stringValue(value)}
          disabled={disabled}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
      )
  }
}

function buildFieldDescription(field: PluginConfigurationField) {
  const parts = [
    field.display?.description?.trim(),
    field.display?.help_text?.trim(),
  ].filter(Boolean)

  return parts.join(' ')
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function numberInputValue(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value)
  }
  if (typeof value === 'string') {
    return value
  }
  return ''
}

function parseOptionalNumber(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? parsed : value
}

function isRedactedSecretValue(value: unknown) {
  return typeof value === 'string' && value.trim() === REDACTED_SECRET_SENTINEL
}
