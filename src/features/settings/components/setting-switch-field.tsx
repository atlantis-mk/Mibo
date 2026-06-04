import {
  Field,
  FieldContent,
  FieldDescription,
  FieldTitle,
} from '@/components/ui/field'
import { Switch } from '@/components/ui/switch'

export function SettingSwitchField({
  title,
  description,
  defaultChecked = false,
}: {
  title: string
  description: string
  defaultChecked?: boolean
}) {
  return (
    <Field
      orientation='horizontal'
      className='items-start rounded-[1.25rem] border border-border/60 bg-muted/30 p-3.5'
    >
      <Switch defaultChecked={defaultChecked} className='mt-0.5' />
      <FieldContent>
        <FieldTitle className='text-foreground'>{title}</FieldTitle>
        <FieldDescription>{description}</FieldDescription>
      </FieldContent>
    </Field>
  )
}
