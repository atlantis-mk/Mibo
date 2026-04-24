import { useEffect, useEffectEvent, useRef, useState } from 'react'
import type { ReactNode, RefObject } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import {
  AlertCircleIcon,
  ArrowLeftIcon,
  LoaderCircleIcon,
  MaximizeIcon,
  MonitorUpIcon,
  PauseIcon,
  PictureInPicture2Icon,
  PlayIcon,
  SkipBackIcon,
  SkipForwardIcon,
  Volume2Icon,
  VolumeXIcon,
} from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '#/components/ui/alert'
import { Badge } from '#/components/ui/badge'
import { Button } from '#/components/ui/button'
import {
  createAuthedMiboApi,
  mediaItemDetailQueryOptions,
  mediaItemProgressQueryOptions,
  miboQueryKeys,
} from '#/lib/mibo-query'
import { cn } from '#/lib/utils'
import { useAuthStore } from '#/stores/auth-store'

type PlayExperienceProps = {
  mediaItemId: number
  fromStart?: boolean
}

export default function PlayExperience({
  mediaItemId,
  fromStart = false,
}: PlayExperienceProps) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const queryToken = token ?? 'guest'
  const hasValidMediaItemId = Number.isFinite(mediaItemId) && mediaItemId > 0
  const playerRef = useRef<HTMLVideoElement | null>(null)
  const hideChromeTimerRef = useRef<number | null>(null)
  const restoreAppliedRef = useRef(false)
  const saveInFlightRef = useRef(false)
  const lastSavedPositionRef = useRef(0)
  const lastSavedAtRef = useRef(0)
  const [videoError, setVideoError] = useState<string | null>(null)
  const [duration, setDuration] = useState(0)
  const [currentTime, setCurrentTime] = useState(0)
  const [isPaused, setIsPaused] = useState(false)
  const [volume, setVolume] = useState(1)
  const [isMuted, setIsMuted] = useState(false)
  const [isChromeVisible, setIsChromeVisible] = useState(true)

  const itemQuery = useQuery({
    ...mediaItemDetailQueryOptions(queryToken, mediaItemId),
    enabled: hasHydrated && !!token && hasValidMediaItemId,
  })
  const progressQuery = useQuery({
    ...mediaItemProgressQueryOptions(queryToken, mediaItemId),
    enabled: hasHydrated && !!token && hasValidMediaItemId,
  })
  const playbackQuery = useQuery({
    queryKey: ['media', 'playback', queryToken, mediaItemId],
    enabled: hasHydrated && !!token && hasValidMediaItemId,
    queryFn: async () => {
      if (!token) {
        throw new Error('当前未登录，无法获取播放信息。')
      }

      return createAuthedMiboApi(token).getPlayback(mediaItemId, {
        clientProfile: 'web',
      })
    },
  })

  const item = itemQuery.data ?? null
  const progress = progressQuery.data ?? null
  const playback = playbackQuery.data ?? null
  const displayDuration =
    duration || playback?.runtime_seconds || item?.runtime_seconds || 0
  const progressPercent =
    displayDuration > 0
      ? Math.min(100, (currentTime / displayDuration) * 100)
      : 0
  const yearLabel =
    item?.year ?? (item?.release_date ? item.release_date.slice(0, 4) : null)

  const persistProgress = useEffectEvent(
    async ({ force = false, completed = false } = {}) => {
      if (
        !token ||
        !item ||
        !playback ||
        !playerRef.current ||
        saveInFlightRef.current
      ) {
        return
      }

      const player = playerRef.current
      const rawDuration = Number.isFinite(player.duration)
        ? player.duration
        : (playback.runtime_seconds ?? item.runtime_seconds ?? 0)
      const durationSeconds =
        rawDuration > 0 ? Math.round(rawDuration) : undefined
      const positionSeconds = Math.max(0, Math.round(player.currentTime || 0))
      const now = Date.now()
      const positionDelta = Math.abs(
        positionSeconds - lastSavedPositionRef.current,
      )

      if (!force && !completed) {
        if (positionSeconds <= 0) {
          return
        }

        if (positionDelta < 10 && now - lastSavedAtRef.current < 15000) {
          return
        }
      }

      saveInFlightRef.current = true

      try {
        const nextProgress = await createAuthedMiboApi(token).updateProgress({
          media_item_id: item.id,
          media_file_id: playback.media_file_id,
          position_seconds:
            completed && durationSeconds ? durationSeconds : positionSeconds,
          duration_seconds: durationSeconds,
          completed,
        })

        lastSavedPositionRef.current = nextProgress.position_seconds
        lastSavedAtRef.current = now
        queryClient.setQueryData(
          miboQueryKeys.mediaItemProgress(queryToken, mediaItemId),
          nextProgress,
        )
      } finally {
        saveInFlightRef.current = false
      }
    },
  )

  useEffect(() => {
    restoreAppliedRef.current = false
    lastSavedPositionRef.current = progress?.position_seconds ?? 0
    lastSavedAtRef.current = 0
    setVideoError(null)
  }, [mediaItemId, progress?.position_seconds, playback?.url])

  useEffect(() => {
    const player = playerRef.current
    if (
      !player ||
      !playback ||
      !progress ||
      fromStart ||
      restoreAppliedRef.current
    ) {
      return
    }

    const savedPosition = Math.round(progress.position_seconds)
    if (savedPosition <= 0) {
      return
    }

    const restorePosition = () => {
      if (restoreAppliedRef.current) {
        return
      }

      const playerDuration = Number.isFinite(player.duration)
        ? player.duration
        : Infinity
      const target = Math.min(savedPosition, Math.max(0, playerDuration - 3))
      if (target <= 0) {
        return
      }

      player.currentTime = target
      setCurrentTime(target)
      restoreAppliedRef.current = true
    }

    if (player.readyState >= 1) {
      restorePosition()
      return
    }

    player.addEventListener('loadedmetadata', restorePosition)

    return () => {
      player.removeEventListener('loadedmetadata', restorePosition)
    }
  }, [fromStart, playback, progress])

  useEffect(() => {
    const player = playerRef.current
    if (!player || !playback) {
      return
    }

    const syncState = () => {
      setCurrentTime(player.currentTime || 0)
      setDuration(Number.isFinite(player.duration) ? player.duration : 0)
      setIsPaused(player.paused)
      setVolume(player.volume)
      setIsMuted(player.muted)
    }

    const handlePause = () => {
      syncState()
      void persistProgress({ force: true })
    }
    const handlePlay = () => {
      syncState()
    }
    const handleTimeUpdate = () => {
      syncState()
      void persistProgress()
    }
    const handleLoadedMetadata = () => {
      syncState()
    }
    const handleVolumeChange = () => {
      syncState()
    }
    const handleEnded = () => {
      syncState()
      void persistProgress({ force: true, completed: true })
    }
    const handleError = () => {
      setVideoError('视频流加载失败，请稍后重试或返回详情页切换播放方式。')
    }

    syncState()
    player.addEventListener('pause', handlePause)
    player.addEventListener('play', handlePlay)
    player.addEventListener('timeupdate', handleTimeUpdate)
    player.addEventListener('loadedmetadata', handleLoadedMetadata)
    player.addEventListener('volumechange', handleVolumeChange)
    player.addEventListener('ended', handleEnded)
    player.addEventListener('error', handleError)

    return () => {
      player.removeEventListener('pause', handlePause)
      player.removeEventListener('play', handlePlay)
      player.removeEventListener('timeupdate', handleTimeUpdate)
      player.removeEventListener('loadedmetadata', handleLoadedMetadata)
      player.removeEventListener('volumechange', handleVolumeChange)
      player.removeEventListener('ended', handleEnded)
      player.removeEventListener('error', handleError)
    }
  }, [playback])

  useEffect(() => {
    if (!playback) {
      return
    }

    const handlePageHide = () => {
      void persistProgress({ force: true })
    }
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'hidden') {
        void persistProgress({ force: true })
      }
    }

    window.addEventListener('pagehide', handlePageHide)
    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      window.removeEventListener('pagehide', handlePageHide)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [playback])

  const revealChrome = useEffectEvent(() => {
    setIsChromeVisible(true)

    if (hideChromeTimerRef.current) {
      window.clearTimeout(hideChromeTimerRef.current)
    }

    if (playerRef.current?.paused) {
      return
    }

    hideChromeTimerRef.current = window.setTimeout(() => {
      setIsChromeVisible(false)
    }, 2200)
  })

  useEffect(() => {
    if (hideChromeTimerRef.current) {
      window.clearTimeout(hideChromeTimerRef.current)
    }

    if (isPaused) {
      setIsChromeVisible(true)
      return
    }

    hideChromeTimerRef.current = window.setTimeout(() => {
      setIsChromeVisible(false)
    }, 2200)

    return () => {
      if (hideChromeTimerRef.current) {
        window.clearTimeout(hideChromeTimerRef.current)
      }
    }
  }, [isPaused])

  if (
    !hasHydrated ||
    (token && (itemQuery.isLoading || playbackQuery.isLoading))
  ) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-black text-white">
        <div className="flex items-center gap-3 rounded-full border border-white/10 bg-white/5 px-5 py-3 backdrop-blur-xl">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-white/70">正在准备播放内容</span>
        </div>
      </div>
    )
  }

  if (!token || !user) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-black px-6 text-white">
        <div className="max-w-xl space-y-4 text-center">
          <Badge
            variant="outline"
            className="border-white/15 bg-white/5 text-white/80"
          >
            Mibo Theater
          </Badge>
          <h1 className="text-4xl font-semibold tracking-tight">
            登录后才能播放媒体
          </h1>
          <p className="text-sm leading-7 text-white/60 sm:text-base">
            当前播放页需要已登录会话来请求后端播放地址和同步观看进度。
          </p>
          <Button asChild size="lg" className="min-w-36">
            <Link to="/login" search={{ redirect: `/play/${mediaItemId}` }}>
              前往登录
            </Link>
          </Button>
        </div>
      </div>
    )
  }

  if (!hasValidMediaItemId) {
    return <PlayError message="无效的媒体 ID。" />
  }

  if (itemQuery.error || playbackQuery.error) {
    return (
      <PlayError
        message={
          itemQuery.error?.message ??
          playbackQuery.error?.message ??
          '播放信息加载失败。'
        }
      />
    )
  }

  if (!item || !playback) {
    return <PlayError message="未找到可播放的媒体内容。" />
  }

  return (
    <div
      className={cn(
        'group/player relative h-svh w-screen overflow-hidden bg-black text-white',
        !isPaused && !isChromeVisible && 'cursor-none [&_*]:cursor-none',
      )}
      onMouseEnter={() => {
        revealChrome()
      }}
      onMouseMove={() => {
        revealChrome()
      }}
      onMouseLeave={() => {
        if (!isPaused) {
          setIsChromeVisible(false)
        }
      }}
      onTouchStart={() => {
        revealChrome()
      }}
    >
      <button
        type="button"
        aria-label={isPaused ? '播放视频' : '暂停视频'}
        className="absolute inset-0 z-0 block bg-transparent"
        onClick={() => {
          void togglePlayback(playerRef)
        }}
      >
        <video
          ref={playerRef}
          key={playback.url}
          className="h-full w-full bg-black object-contain"
          src={playback.url}
          poster={item.backdrop_url || item.poster_url || undefined}
          autoPlay
          playsInline
          preload="metadata"
          crossOrigin="anonymous"
          controls={false}
        />
      </button>

      <div className="pointer-events-none absolute inset-0 bg-black/10" />

      <div
        className={cn(
          'pointer-events-none absolute inset-x-0 top-0 z-20 bg-gradient-to-b from-black/78 via-black/26 to-transparent px-4 py-5 transition-all duration-300 sm:px-7',
          isPaused || isChromeVisible
            ? 'translate-y-0 opacity-100'
            : '-translate-y-6 opacity-0',
        )}
      >
        <div
          className="flex items-center justify-between gap-4"
          onClick={(event) => {
            event.stopPropagation()
          }}
        >
          <button
            type="button"
            className="pointer-events-auto inline-flex min-w-0 items-center gap-3 text-left text-white transition hover:text-white/80"
            onClick={() => {
              if (window.history.length > 1) {
                window.history.back()
                return
              }

              void navigate({
                to: '/media/$id',
                params: { id: String(item.id) },
              })
            }}
          >
            <ArrowLeftIcon className="size-6 shrink-0" />
            <span className="truncate text-2xl font-semibold tracking-tight sm:text-3xl">
              {item.title}
            </span>
          </button>

          <div className="pointer-events-auto hidden items-center gap-3 lg:flex">
            <div className="flex items-center gap-3 rounded-full bg-white/6 px-4 py-2 backdrop-blur-md">
              <button
                type="button"
                className="text-white/80 transition hover:text-white"
                onClick={toggleMute.bind(null, playerRef)}
              >
                {isMuted || volume <= 0 ? (
                  <VolumeXIcon className="size-5" />
                ) : (
                  <Volume2Icon className="size-5" />
                )}
              </button>
              <input
                type="range"
                min="0"
                max="1"
                step="0.01"
                value={isMuted ? 0 : volume}
                onChange={(event) => {
                  const nextVolume = Number(event.target.value)
                  setPlayerVolume(playerRef, nextVolume)
                }}
                className="h-1.5 w-36 accent-white"
              />
            </div>

            <PlayerIconButton
              onClick={() => void requestPictureInPicture(playerRef)}
            >
              <PictureInPicture2Icon className="size-5" />
            </PlayerIconButton>
            <PlayerIconButton onClick={() => void requestFullscreen(playerRef)}>
              <MonitorUpIcon className="size-5" />
            </PlayerIconButton>
          </div>
        </div>
      </div>

      {videoError ? (
        <div className="absolute inset-x-4 top-24 z-30 sm:inset-x-8">
          <Alert className="border-red-400/20 bg-red-500/10 text-white backdrop-blur-md">
            <AlertCircleIcon className="size-4" />
            <AlertTitle>播放失败</AlertTitle>
            <AlertDescription>{videoError}</AlertDescription>
          </Alert>
        </div>
      ) : null}

      <div
        className={cn(
          'absolute inset-x-0 bottom-0 z-20 bg-gradient-to-t from-black via-black/78 to-transparent px-4 pb-4 pt-16 transition-all duration-300 sm:px-7 sm:pb-6',
          isPaused || isChromeVisible
            ? 'translate-y-0 opacity-100'
            : 'translate-y-10 opacity-0',
        )}
      >
        <div
          className="pointer-events-auto"
          onClick={(event) => {
            event.stopPropagation()
          }}
        >
          <div className="mb-2 flex items-end justify-between gap-4">
            <div className="min-w-0">
              {yearLabel ? (
                <div className="text-sm text-white/60">{yearLabel}</div>
              ) : null}
              <div className="truncate text-3xl font-semibold tracking-tight sm:text-4xl">
                {item.title}
              </div>
            </div>

            <div className="hidden items-center gap-2 sm:flex">
              <PlayerIconButton
                onClick={() => void requestPictureInPicture(playerRef)}
              >
                <PictureInPicture2Icon className="size-4.5" />
              </PlayerIconButton>
              <PlayerIconButton
                onClick={() => void requestFullscreen(playerRef)}
              >
                <MaximizeIcon className="size-4.5" />
              </PlayerIconButton>
            </div>
          </div>

          <div className="space-y-3">
            <input
              type="range"
              min="0"
              max={Math.max(displayDuration, 0)}
              step="1"
              value={Math.min(currentTime, displayDuration || currentTime)}
              onChange={(event) => {
                const nextTime = Number(event.target.value)
                seekTo(playerRef, nextTime)
                setCurrentTime(nextTime)
              }}
              style={{
                background: `linear-gradient(to right, rgba(255,255,255,0.92) 0%, rgba(255,255,255,0.92) ${progressPercent}%, rgba(255,255,255,0.24) ${progressPercent}%, rgba(255,255,255,0.24) 100%)`,
              }}
              className="h-1.5 w-full cursor-pointer appearance-none rounded-full bg-white/20"
            />

            <div className="flex flex-wrap items-center justify-between gap-4 text-white/90">
              <div className="flex items-center gap-5">
                <ControlButton
                  onClick={() => {
                    seekBy(playerRef, -10)
                  }}
                  label="后退 10 秒"
                >
                  <SkipBackIcon className="size-5" />
                  <span className="text-[10px] font-semibold">10</span>
                </ControlButton>
                <ControlButton
                  onClick={() => {
                    void togglePlayback(playerRef)
                  }}
                  label={isPaused ? '播放' : '暂停'}
                >
                  {isPaused ? (
                    <PlayIcon className="size-7 fill-current" />
                  ) : (
                    <PauseIcon className="size-7 fill-current" />
                  )}
                </ControlButton>
                <ControlButton
                  onClick={() => {
                    seekBy(playerRef, 10)
                  }}
                  label="前进 10 秒"
                >
                  <SkipForwardIcon className="size-5" />
                  <span className="text-[10px] font-semibold">10</span>
                </ControlButton>
              </div>

              <div className="text-sm tabular-nums text-white/70">
                {formatClock(currentTime)} / {formatClock(displayDuration)}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function PlayError({ message }: { message: string }) {
  return (
    <div className="flex min-h-svh items-center justify-center bg-black px-6 text-white">
      <div className="max-w-lg rounded-[2rem] border border-white/10 bg-white/5 p-8 text-center backdrop-blur-xl">
        <Badge
          variant="outline"
          className="border-white/15 bg-white/5 text-white/70"
        >
          播放不可用
        </Badge>
        <h1 className="mt-4 text-3xl font-semibold tracking-tight">
          当前媒体暂时无法播放
        </h1>
        <p className="mt-3 text-sm leading-7 text-white/60">{message}</p>
      </div>
    </div>
  )
}

function PlayerIconButton({
  children,
  onClick,
}: {
  children: ReactNode
  onClick?: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex size-10 items-center justify-center rounded-full bg-white/6 text-white/82 transition hover:bg-white/12 hover:text-white"
    >
      {children}
    </button>
  )
}

function ControlButton({
  children,
  onClick,
  label,
}: {
  children: ReactNode
  onClick: () => void
  label: string
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="inline-flex items-center gap-1 text-white transition hover:text-white/70"
    >
      {children}
    </button>
  )
}

async function togglePlayback(playerRef: RefObject<HTMLVideoElement | null>) {
  const player = playerRef.current
  if (!player) {
    return
  }

  if (player.paused) {
    await player.play()
    return
  }

  player.pause()
}

function seekBy(
  playerRef: RefObject<HTMLVideoElement | null>,
  seconds: number,
) {
  const player = playerRef.current
  if (!player) {
    return
  }

  const nextTime = Math.max(0, player.currentTime + seconds)
  player.currentTime = nextTime
}

function seekTo(
  playerRef: RefObject<HTMLVideoElement | null>,
  seconds: number,
) {
  const player = playerRef.current
  if (!player) {
    return
  }

  player.currentTime = Math.max(0, seconds)
}

function setPlayerVolume(
  playerRef: RefObject<HTMLVideoElement | null>,
  nextVolume: number,
) {
  const player = playerRef.current
  if (!player) {
    return
  }

  player.volume = Math.min(1, Math.max(0, nextVolume))
  player.muted = nextVolume <= 0
}

function toggleMute(playerRef: RefObject<HTMLVideoElement | null>) {
  const player = playerRef.current
  if (!player) {
    return
  }

  player.muted = !player.muted
}

async function requestFullscreen(
  playerRef: RefObject<HTMLVideoElement | null>,
) {
  const player = playerRef.current
  if (!player) {
    return
  }

  const host = player.parentElement
  if (!host || !host.requestFullscreen) {
    return
  }

  await host.requestFullscreen()
}

async function requestPictureInPicture(
  playerRef: RefObject<HTMLVideoElement | null>,
) {
  const player = playerRef.current
  if (!player || typeof player.requestPictureInPicture !== 'function') {
    return
  }

  await player.requestPictureInPicture()
}
function formatClock(seconds?: number) {
  if (!seconds || seconds <= 0) {
    return '00:00'
  }

  const total = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const remainder = total % 60

  if (hours > 0) {
    return [hours, minutes, remainder]
      .map((value) => String(value).padStart(2, '0'))
      .join(':')
  }

  return [minutes, remainder]
    .map((value) => String(value).padStart(2, '0'))
    .join(':')
}
