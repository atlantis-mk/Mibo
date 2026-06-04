import Hls, { type HlsConfig } from 'hls.js'

export type MiboHlsVideoElement = HTMLVideoElement & {
  __miboHls?: Hls
  __miboHlsPlaylistRefreshTimer?: number
}

const LIVE_TV_HLS_CONFIG: Partial<HlsConfig> = {
  lowLatencyMode: true,
  startFragPrefetch: true,
  liveSyncDurationCount: 3,
  liveMaxLatencyDurationCount: 6,
  maxBufferLength: 45,
  maxMaxBufferLength: 60,
  backBufferLength: 30,
  maxBufferHole: 0.5,
}

const TRANSCODE_HLS_CONFIG: Partial<HlsConfig> = {
  lowLatencyMode: false,
  startFragPrefetch: true,
  liveSyncDuration: 10,
  liveMaxLatencyDuration: 24 * 60 * 60,
  maxLiveSyncPlaybackRate: 1,
  maxBufferLength: 90,
  maxMaxBufferLength: 120,
  backBufferLength: 60,
  maxBufferHole: 0.5,
}

const LIVE_TV_FATAL_NETWORK_RETRY_LIMIT = 3
const LIVE_TV_FATAL_NETWORK_RETRY_DELAY_MS = 1500

export function attachLiveTVStream(
  video: HTMLVideoElement,
  url: string,
  options?: { token?: string; playlistRefreshIntervalMs?: number }
): boolean {
  const hlsVideo = video as MiboHlsVideoElement
  destroyLiveTVStream(hlsVideo)

  if (Hls.isSupported()) {
    const isProgressiveTranscode =
      typeof options?.playlistRefreshIntervalMs === 'number' &&
      options.playlistRefreshIntervalMs > 0
    const hls = new Hls({
      ...(isProgressiveTranscode ? TRANSCODE_HLS_CONFIG : LIVE_TV_HLS_CONFIG),
      xhrSetup: (xhr) => {
        xhr.withCredentials = true
        if (options?.token) {
          xhr.setRequestHeader('Authorization', `Bearer ${options.token}`)
        }
      },
    })
    let fatalNetworkRetryCount = 0

    hls.on(Hls.Events.ERROR, (_event, data) => {
      if (!data.fatal) {
        return
      }
      if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
        if (fatalNetworkRetryCount >= LIVE_TV_FATAL_NETWORK_RETRY_LIMIT) {
          return
        }
        fatalNetworkRetryCount += 1
        window.setTimeout(() => {
          if (hlsVideo.__miboHls === hls) {
            hls.startLoad()
          }
        }, LIVE_TV_FATAL_NETWORK_RETRY_DELAY_MS)
        return
      }
      if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
        hls.recoverMediaError()
      }
    })

    hls.loadSource(url)
    hls.attachMedia(video)
    hlsVideo.__miboHls = hls
    if (isProgressiveTranscode) {
      hlsVideo.__miboHlsPlaylistRefreshTimer = window.setInterval(() => {
        if (video.paused || video.ended) {
          return
        }
        refreshHlsPlaylist(hls, video)
      }, options.playlistRefreshIntervalMs)
    }
    return true
  }

  if (video.canPlayType('application/vnd.apple.mpegurl')) {
    video.src = url
    return true
  }

  return false
}

function refreshHlsPlaylist(hls: Hls, video: HTMLVideoElement) {
  const level = hls.loadLevel
  const levelInfo = hls.loadLevelObj
  if (level < 0 || !levelInfo) {
    hls.startLoad(video.currentTime || -1, true)
    return
  }
  const playbackPosition = video.currentTime
  hls.once(Hls.Events.LEVEL_LOADED, () => {
    if (
      Number.isFinite(playbackPosition) &&
      Number.isFinite(video.currentTime) &&
      video.currentTime - playbackPosition > 3 &&
      !video.seeking
    ) {
      video.currentTime = playbackPosition
    }
  })

  hls.trigger(Hls.Events.LEVEL_LOADING, {
    url: levelInfo.uri,
    level,
    levelInfo,
    pathwayId: levelInfo.attrs['PATHWAY-ID'],
    id: 0,
    deliveryDirectives: null,
  })
}

export function destroyLiveTVStream(video?: HTMLVideoElement | null) {
  if (!video) {
    return
  }
  const hlsVideo = video as MiboHlsVideoElement
  if (hlsVideo.__miboHlsPlaylistRefreshTimer) {
    window.clearInterval(hlsVideo.__miboHlsPlaylistRefreshTimer)
  }
  hlsVideo.__miboHls?.destroy()
  delete hlsVideo.__miboHlsPlaylistRefreshTimer
  delete hlsVideo.__miboHls
  video.removeAttribute('src')
  video.load()
}
