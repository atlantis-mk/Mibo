import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '#/components/ui/card'
import { Separator } from '#/components/ui/separator'

export function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[1rem] border border-border/60 bg-muted/30 px-3.5 py-3">
      <div className="text-xs uppercase tracking-[0.16em] text-muted-foreground">
        {label}
      </div>
      <div className="mt-2 break-all text-sm text-foreground">{value}</div>
    </div>
  )
}

export function EmptyCard({ text }: { text: string }) {
  return (
    <Card className="rounded-[1.5rem] border border-dashed border-border/60 bg-card/60 py-0">
      <CardContent className="px-5 py-8 text-sm leading-7 text-muted-foreground">
        {text}
      </CardContent>
    </Card>
  )
}

export function SettingsAsideCard({
  title,
  description,
  items,
}: {
  title: string
  description: string
  items: Array<[string, string]>
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/60 bg-card/80 py-0 shadow-sm backdrop-blur-sm">
      <CardHeader className="px-5 py-5">
        <CardTitle className="text-xl">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <Separator className="bg-border" />
      <CardContent className="px-5 py-5">
        <div className="space-y-3">
          {items.map(([label, value]) => (
            <div
              key={label}
              className="rounded-[1.1rem] border border-border/60 bg-muted/30 px-3.5 py-3"
            >
              <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                {label}
              </div>
              <div className="mt-2 text-sm font-medium text-foreground">
                {value}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
