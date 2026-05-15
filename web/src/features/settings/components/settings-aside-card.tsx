import {
  Card,
  CardContent,
} from '#/components/ui/card'

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
