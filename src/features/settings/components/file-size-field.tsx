import { useEffect, useRef, useState } from 'react'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

type FileSizeUnit = {
  value: string
  label: string
  multiplier: number
}

const FILE_SIZE_UNITS: FileSizeUnit[] = [
  { value: 'B', label: 'B', multiplier: 1 },
  { value: 'KB', label: 'KB', multiplier: 1024 },
  { value: 'MB', label: 'MB', multiplier: 1024 * 1024 },
  { value: 'GB', label: 'GB', multiplier: 1024 * 1024 * 1024 },
  { value: 'TB', label: 'TB', multiplier: 1024 * 1024 * 1024 * 1024 },
]

const DEFAULT_FILE_SIZE_UNIT = 'MB'

function clampBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return 0
  }
  return Math.max(0, Math.round(value))
}

function fileSizeUnitByValue(value: string) {
  return (
    FILE_SIZE_UNITS.find((unit) => unit.value === value) ?? FILE_SIZE_UNITS[0]
  )
}

function formatFileSizeAmount(value: number) {
  const rounded = Math.round(value * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : String(rounded)
}

function fileSizePartsFromBytes(bytes: number) {
  const safeBytes = clampBytes(bytes)
  if (safeBytes === 0) {
    return {
      amount: '0',
      unit: DEFAULT_FILE_SIZE_UNIT,
    }
  }
  const unit =
    [...FILE_SIZE_UNITS]
      .reverse()
      .find((candidate) => safeBytes >= candidate.multiplier) ??
    FILE_SIZE_UNITS[0]

  return {
    amount: formatFileSizeAmount(safeBytes / unit.multiplier),
    unit: unit.value,
  }
}

function bytesFromFileSizeParts(amount: string, unitValue: string) {
  const numericAmount = Number(amount)
  if (!Number.isFinite(numericAmount) || numericAmount <= 0) {
    return 0
  }
  return clampBytes(numericAmount * fileSizeUnitByValue(unitValue).multiplier)
}

export function FileSizeField({
  label,
  value,
  onChange,
}: {
  label: string
  value: number
  onChange: (value: number) => void
}) {
  const [amount, setAmount] = useState('0')
  const [unit, setUnit] = useState(DEFAULT_FILE_SIZE_UNIT)
  const lastEmittedBytesRef = useRef<number | null>(null)

  useEffect(() => {
    if (amount !== '' && lastEmittedBytesRef.current === clampBytes(value)) {
      return
    }
    const next = fileSizePartsFromBytes(value)
    setAmount(next.amount)
    setUnit(next.unit)
    lastEmittedBytesRef.current = clampBytes(value)
  }, [amount, value])

  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <div className='flex gap-2'>
        <Input
          type='number'
          min='0'
          step='0.01'
          value={amount}
          onChange={(event) => {
            const nextAmount = event.target.value
            const nextBytes = bytesFromFileSizeParts(nextAmount, unit)
            setAmount(nextAmount)
            lastEmittedBytesRef.current = nextBytes
            onChange(nextBytes)
          }}
        />
        <Select
          value={unit}
          onValueChange={(nextUnit) => {
            const nextBytes = bytesFromFileSizeParts(amount, nextUnit)
            setUnit(nextUnit)
            lastEmittedBytesRef.current = nextBytes
            onChange(nextBytes)
          }}
        >
          <SelectTrigger className='w-28'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {FILE_SIZE_UNITS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </Field>
  )
}
