import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '#/components/ui/dialog'
import type { Trailer } from '#/lib/mibo-api'

type StandaloneMediaDetailTrailerDialogProps = {
  open: boolean
  trailer?: Trailer
  title: string
  onOpenChange: (open: boolean) => void
}

export function StandaloneMediaDetailTrailerDialog({
  open,
  trailer,
  title,
  onOpenChange,
}: StandaloneMediaDetailTrailerDialogProps) {
  const embedUrl = buildAutoplayEmbedUrl(trailer?.embed_url)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        showCloseButton
        className="w-[min(1080px,calc(100%-1.5rem))] max-w-[min(1080px,calc(100%-1.5rem))] gap-3 overflow-hidden border border-border/50 bg-background/95 p-0 shadow-2xl"
      >
        <div className="border-b border-border/50 px-5 py-4 pr-12 sm:px-6">
          <DialogTitle className="text-lg text-foreground">
            预告片 · {title}
          </DialogTitle>
          <DialogDescription className="mt-1 text-sm text-muted-foreground">
            {trailer?.name || '正在播放选定预告片'}
          </DialogDescription>
        </div>

        <div className="bg-black">
          {embedUrl ? (
            <div className="aspect-video w-full">
              <iframe
                key={embedUrl}
                src={embedUrl}
                title={`${title} trailer`}
                className="h-full w-full border-0"
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                allowFullScreen
                referrerPolicy="strict-origin-when-cross-origin"
              />
            </div>
          ) : (
            <div className="flex aspect-video items-center justify-center px-6 text-center text-sm text-muted-foreground">
              当前预告片暂时不可播放，请稍后重试。
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function buildAutoplayEmbedUrl(embedUrl?: string) {
  if (!embedUrl) {
    return ''
  }

  try {
    const url = new URL(embedUrl)
    url.searchParams.set('autoplay', '1')
    url.searchParams.set('rel', '0')
    url.searchParams.set('modestbranding', '1')
    return url.toString()
  } catch {
    return embedUrl
  }
}
