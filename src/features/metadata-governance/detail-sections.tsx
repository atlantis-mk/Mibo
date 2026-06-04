export function ArtworkPreview({
  label,
  imageUrl,
  wide,
}: {
  label: string
  imageUrl: string
  wide?: boolean
}) {
  return (
    <div className='space-y-2'>
      <div className='text-xs font-medium tracking-[0.18em] text-muted-foreground uppercase'>
        {label}
      </div>
      <div
        className={
          wide
            ? 'aspect-[16/7] overflow-hidden rounded-xl bg-muted'
            : 'aspect-[2/3] max-w-42 overflow-hidden rounded-xl bg-muted'
        }
      >
        {imageUrl ? (
          <img
            src={imageUrl}
            alt={label}
            className='h-full w-full object-cover'
          />
        ) : null}
      </div>
    </div>
  )
}

export function SummaryRow({
  label,
  value,
  multiline,
}: {
  label: string
  value: string
  multiline?: boolean
}) {
  return (
    <div className='space-y-1'>
      <div className='text-xs font-medium tracking-[0.18em] text-muted-foreground uppercase'>
        {label}
      </div>
      <div
        className={
          multiline
            ? 'text-sm whitespace-pre-wrap text-foreground'
            : 'text-sm text-foreground'
        }
      >
        {value}
      </div>
    </div>
  )
}
