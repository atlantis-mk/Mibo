import { useEffect, useRef, type VideoHTMLAttributes } from 'react'
import { attachLiveTVStream, destroyLiveTVStream } from './live-tv-stream'

type LiveTVVideoProps = Omit<
  VideoHTMLAttributes<HTMLVideoElement>,
  'children' | 'src'
> & {
  src: string
}

export function LiveTVVideo({ src, ...props }: LiveTVVideoProps) {
  const videoRef = useRef<HTMLVideoElement | null>(null)

  useEffect(() => {
    const video = videoRef.current
    if (!video) {
      return
    }
    attachLiveTVStream(video, src)
    return () => {
      destroyLiveTVStream(video)
    }
  }, [src])

  return <video ref={videoRef} {...props} />
}
