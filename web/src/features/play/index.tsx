import { useEffect, useEffectEvent, useRef, useState } from "react"
import type { RefObject } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "@tanstack/react-router"
import Artplayer from "artplayer"
import {
  CameraIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  InfoIcon,
  MaximizeIcon,
  PauseIcon,
  PictureInPicture2Icon,
  PlayIcon,
  SettingsIcon,
  SkipForwardIcon,
  Volume2Icon,
  VolumeXIcon,
  XIcon,
} from "lucide-react"
import { toast } from "sonner"
import { Slider } from "#/components/ui/slider"
import type { CatalogItemDetail, Track } from "#/lib/mibo-api"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "#/components/ui/sidebar"
import { Spinner } from "#/components/ui/spinner"
import {
	catalogPlaybackQueryOptions,
	createAuthedMiboApi,
	inventoryFilePlaybackQueryOptions,
	metadataItemDetailQueryOptions,
	metadataItemProgressQueryOptions,
	miboQueryKeys,
} from "#/lib/mibo-query"
import { useAuthStore } from "#/stores/auth-store"

import { AppSidebar } from "./components/AppSidebar"
import * as React from "react"

Artplayer.DEBUG = true
Artplayer.DBCLICK_FULLSCREEN = false

type ArtPlayerRef = RefObject<Artplayer | null>

const PLAYBACK_RATE_OPTIONS = [5, 3, 2, 1.5, 1, 0.8]

const PLAYBACK_MODE_OPTIONS = [
  "自动连播",
  "单集循环",
  "播放列表循环",
  "播完停止",
] as const

const MAX_SKIP_EDGE_SECONDS = 300

const SUBTITLE_COLOR_OPTIONS = [
  { label: "白色（默认）", value: "#ffffff" },
  { label: "黄色", value: "#ffe66d" },
  { label: "蓝色", value: "#8ec5ff" },
  { label: "绿色", value: "#a7f3a7" },
] as const

const SUBTITLE_POSITION_OPTIONS = [
  { label: "中间下（默认）", bottom: 8 },
  { label: "底部", bottom: 4 },
  { label: "中间", bottom: 42 },
  { label: "顶部", bottom: 78 },
] as const

const SUBTITLE_SIZE_OPTIONS = [
  { label: "小", fontSize: 28 },
  { label: "标准", fontSize: 36 },
  { label: "大", fontSize: 44 },
  { label: "极大", fontSize: 52 },
] as const

type PlaybackMode = (typeof PLAYBACK_MODE_OPTIONS)[number]
type SubtitleColorIndex = 0 | 1 | 2 | 3
type SubtitlePositionIndex = 0 | 1 | 2 | 3
type SubtitleSizeIndex = 0 | 1 | 2 | 3

type PlayQueueEpisode = {
  id: number
}

type PlayExperienceProps = {
  itemId: number
  resourceId?: number
  inventoryFileId?: number
  fromStart?: boolean
}

export default function PlayPage(props: PlayExperienceProps) {
  return (
    <SidebarProvider defaultOpen={false}>
      <PlayExperience {...props} />
    </SidebarProvider>
  )
}

function PlayExperience({
  itemId,
  resourceId,
  inventoryFileId,
  fromStart = false,
}: PlayExperienceProps) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const queryToken = token ?? "guest"
  const hasValidItemId = Number.isFinite(itemId) && itemId > 0
  const hasInventoryFilePlayback =
    Number.isFinite(inventoryFileId) && (inventoryFileId ?? 0) > 0
  const playerRef = useRef<Artplayer | null>(null)
  const playerRootRef = useRef<HTMLDivElement | null>(null)
  const playerContainerRef = useRef<HTMLDivElement | null>(null)
  const restoreAppliedRef = useRef(false)
  const saveInFlightRef = useRef(false)
  const lastSavedPositionRef = useRef(0)
  const lastSavedAtRef = useRef(0)
  const lastFrameSavedAtRef = useRef(0)
  const skipOutroSecondsRef = useRef(0)
  const controlsHideTimerRef = useRef<number | null>(null)
  const playbackFeedbackTimerRef = useRef<number | null>(null)
  const [duration, setDuration] = useState(0)
  const [currentTime, setCurrentTime] = useState(0)
  const [isPaused, setIsPaused] = useState(true)
  const [isMuted, setIsMuted] = useState(false)
  const [volumePercent, setVolumePercent] = useState(100)
  const [playbackRate, setPlaybackRate] = useState(1)
  const [playbackMode, setPlaybackMode] = useState<PlaybackMode>("自动连播")
  const [restorePositionEnabled, setRestorePositionEnabled] =
    useState(!fromStart)
  const [skipIntroSeconds, setSkipIntroSeconds] = useState(0)
  const [skipOutroSeconds, setSkipOutroSeconds] = useState(0)
  const [skipSettingsOpen, setSkipSettingsOpen] = useState(false)
  const [controlsVisible, setControlsVisible] = useState(true)
  const [controlsInteracting, setControlsInteracting] = useState(false)
  const [isVideoLoading, setIsVideoLoading] = useState(true)
  const [playbackFeedback, setPlaybackFeedback] = useState<
    "play" | "pause" | null
  >(null)

	const itemQuery = useQuery({
		...metadataItemDetailQueryOptions(queryToken, itemId),
		enabled:
			hasHydrated && !!token && hasValidItemId && !hasInventoryFilePlayback,
	})
	const progressQuery = useQuery({
		...metadataItemProgressQueryOptions(queryToken, itemId),
		enabled:
			hasHydrated && !!token && hasValidItemId && !hasInventoryFilePlayback,
	})
  const playbackQuery = useQuery({
    ...catalogPlaybackQueryOptions(queryToken, itemId, { resourceId }),
    enabled:
      hasHydrated && !!token && hasValidItemId && !hasInventoryFilePlayback,
  })
  const inventoryPlaybackQuery = useQuery({
    ...inventoryFilePlaybackQueryOptions(queryToken, inventoryFileId ?? 0),
    enabled: hasHydrated && !!token && hasInventoryFilePlayback,
  })
  const item = itemQuery.data ?? null
  const progress = progressQuery.data ?? null
  const playback = hasInventoryFilePlayback
    ? (inventoryPlaybackQuery.data ?? null)
    : (playbackQuery.data ?? null)
  const posterUrl = item
    ? catalogImageUrl(item, "backdrop") || catalogImageUrl(item, "poster")
    : undefined
  const playbackTitle = item?.title ?? playback?.title ?? "整理中媒体"
  const playbackHeader = buildPlaybackHeader(item, playbackTitle)
  const displayDuration =
    duration || playback?.runtime_seconds || item?.runtime_seconds || 0
  const progressPercent =
    displayDuration > 0
      ? Math.min(100, (currentTime / displayDuration) * 100)
      : 0
  const episodeItems = item?.same_season_episodes?.length
    ? item.same_season_episodes
    : (item?.seasons?.flatMap((season) => season.episodes ?? []) ?? [])
  const currentEpisodeIndex = episodeItems.findIndex(
    (episode) => episode.id === item?.id
  )
  const nextEpisode =
    currentEpisodeIndex >= 0 ? episodeItems[currentEpisodeIndex + 1] : undefined
  const showNextEpisodeButton = item?.type === "episode"

  useEffect(() => {
    skipOutroSecondsRef.current = skipOutroSeconds
  }, [skipOutroSeconds])

  useEffect(() => {
    document.title = buildPlaybackDocumentTitle(item, playbackTitle)
  }, [item, playbackTitle])

  const persistProgress = useEffectEvent(
    async ({ force = false, completed = false } = {}) => {
      if (
        !token ||
        (!item && !hasInventoryFilePlayback) ||
        !playback ||
        !playerRef.current ||
        saveInFlightRef.current
      ) {
        return
      }

      const player = playerRef.current
      const rawDuration = Number.isFinite(player.duration)
        ? player.duration
        : (playback.runtime_seconds ?? item?.runtime_seconds ?? 0)
      const durationSeconds =
        rawDuration > 0 ? Math.round(rawDuration) : undefined
      const positionSeconds = Math.max(0, Math.round(player.currentTime || 0))
      const now = Date.now()
      const positionDelta = Math.abs(
        positionSeconds - lastSavedPositionRef.current
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
        if (hasInventoryFilePlayback || !item) return
		const progressMetadataItemId = playback.metadata_item_id
		if (!progressMetadataItemId) return
		const progressResourceId =
          typeof playback.resource_id === "number" && playback.resource_id > 0
            ? playback.resource_id
            : undefined
        const shouldCaptureFrame =
          !completed &&
          positionSeconds > 0 &&
          now - lastFrameSavedAtRef.current > 30000
        const progressFrameData = shouldCaptureFrame
          ? captureProgressFrame(player.video)
          : undefined
		const nextProgress = await createAuthedMiboApi(token).updateProgress({
			metadata_item_id: progressMetadataItemId,
			...(progressResourceId ? { resource_id: progressResourceId } : {}),
          position_seconds:
            completed && durationSeconds ? durationSeconds : positionSeconds,
          duration_seconds: durationSeconds,
          completed,
          ...(progressFrameData
            ? { progress_frame_data: progressFrameData }
            : {}),
        })

        lastSavedPositionRef.current = nextProgress.position_seconds
        lastSavedAtRef.current = now
        if (progressFrameData) {
          lastFrameSavedAtRef.current = now
        }
		queryClient.setQueryData(
			miboQueryKeys.catalogItemProgress(queryToken, progressMetadataItemId),
			nextProgress
		)
      } finally {
        saveInFlightRef.current = false
      }
    }
  )

  useEffect(() => {
    restoreAppliedRef.current = false
    lastSavedAtRef.current = 0
    lastFrameSavedAtRef.current = 0
    setIsVideoLoading(true)
  }, [itemId, resourceId, playback?.url])

  useEffect(() => {
    lastSavedPositionRef.current = progress?.position_seconds ?? 0
  }, [progress?.position_seconds])

  const restoreProgress = useEffectEvent(() => {
    const player = playerRef.current
    if (
      !player ||
      !progress ||
      fromStart ||
      !restorePositionEnabled ||
      restoreAppliedRef.current
    ) {
      return
    }

    const savedPosition = Math.round(progress.position_seconds)
    if (savedPosition <= 0) {
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
    restoreAppliedRef.current = true
  })

  useEffect(() => {
    restoreProgress()
  }, [progress?.position_seconds, fromStart, restorePositionEnabled])

  useEffect(() => {
    const container = playerContainerRef.current
    if (!container || !playback || (!item && !hasInventoryFilePlayback)) {
      return
    }

    const player = new Artplayer({
      container,
      url: playback.url,
      ...(posterUrl ? { poster: posterUrl } : {}),
      autoplay: true,
      playsInline: true,
      theme: "#ffffff",
      setting: false,
      playbackRate: false,
      pip: false,
      fullscreen: false,
      fullscreenWeb: false,
      miniProgressBar: false,
      hotkey: true,
      lock: true,
      controls: [],
      settings: [],
      layers: [],
      contextmenu: [],
      moreVideoAttr: {
        crossOrigin: "anonymous",
      },
    })

    playerRef.current = player

    const syncState = () => {
      setCurrentTime(player.currentTime || 0)
      setDuration(Number.isFinite(player.duration) ? player.duration : 0)
      setIsPaused(!player.playing)
      setIsMuted(player.muted)
      setVolumePercent(Math.round((player.volume ?? 1) * 100))
      setPlaybackRate(player.playbackRate)
    }

    const handlePause = () => {
      syncState()
      showPlaybackFeedback("pause")
      void persistProgress({ force: true })
    }
    const handlePlay = () => {
      syncState()
      showPlaybackFeedback("play")
    }
    const handleVideoReady = () => {
      syncState()
      setIsVideoLoading(false)
    }
    const handleVideoLoading = () => {
      setIsVideoLoading(true)
    }
    const handleTimeUpdate = () => {
      syncState()
      if (
        skipOutroSecondsRef.current > 0 &&
        Number.isFinite(player.duration) &&
        player.duration > skipOutroSecondsRef.current &&
        player.duration - player.currentTime <= skipOutroSecondsRef.current
      ) {
        player.currentTime = player.duration
        return
      }

      void persistProgress()
    }
    const handleLoadedMetadata = () => {
      syncState()
      restoreProgress()
    }
    const handleEnded = () => {
      syncState()
      void persistProgress({ force: true, completed: true })
      handlePlaybackEnded()
    }
    const handleVolumeChange = () => {
      syncState()
    }
    const handleRateChange = () => {
      syncState()
    }

    syncState()
    player.on("ready", handleLoadedMetadata)
    player.on("video:pause", handlePause)
    player.on("video:play", handlePlay)
    player.on("video:playing", handleVideoReady)
    player.on("video:timeupdate", handleTimeUpdate)
    player.on("video:loadedmetadata", handleLoadedMetadata)
    player.on("video:loadeddata", handleVideoReady)
    player.on("video:canplay", handleVideoReady)
    player.on("video:waiting", handleVideoLoading)
    player.on("video:seeking", handleVideoLoading)
    player.on("video:seeked", handleVideoReady)
    player.on("video:ended", handleEnded)
    player.on("video:volumechange", handleVolumeChange)
    player.on("video:ratechange", handleRateChange)

    return () => {
      playerRef.current = null
      player.destroy(false)
    }
  }, [item?.id, playback?.url, playbackTitle, posterUrl])

  useEffect(() => {
    if (!playback) {
      return
    }

    const handlePageHide = () => {
      void persistProgress({ force: true })
    }
    const handleVisibilityChange = () => {
      if (document.visibilityState === "hidden") {
        void persistProgress({ force: true })
      }
    }

    window.addEventListener("pagehide", handlePageHide)
    document.addEventListener("visibilitychange", handleVisibilityChange)

    return () => {
      window.removeEventListener("pagehide", handlePageHide)
      document.removeEventListener("visibilitychange", handleVisibilityChange)
    }
  }, [playback])

  useEffect(() => {
    const player = playerRef.current
    if (!player || skipIntroSeconds <= 0 || restoreAppliedRef.current) {
      return
    }

    const current = Math.round(player.currentTime || 0)
    if (current < skipIntroSeconds) {
      player.currentTime = skipIntroSeconds
    }
  }, [skipIntroSeconds, playback?.url])

  useEffect(() => {
    return () => {
      if (controlsHideTimerRef.current) {
        window.clearTimeout(controlsHideTimerRef.current)
      }
      if (playbackFeedbackTimerRef.current) {
        window.clearTimeout(playbackFeedbackTimerRef.current)
      }
    }
  }, [])

  const showPlaybackFeedback = (feedback: "play" | "pause") => {
    if (playbackFeedbackTimerRef.current) {
      window.clearTimeout(playbackFeedbackTimerRef.current)
    }

    setPlaybackFeedback(feedback)
    playbackFeedbackTimerRef.current = window.setTimeout(() => {
      setPlaybackFeedback(null)
      playbackFeedbackTimerRef.current = null
    }, 520)
  }

  const scheduleControlsHide = () => {
    if (controlsHideTimerRef.current) {
      window.clearTimeout(controlsHideTimerRef.current)
    }

    controlsHideTimerRef.current = window.setTimeout(() => {
      if (!controlsInteracting) {
        setControlsVisible(false)
      }
    }, 2200)
  }

  const showControls = () => {
    setControlsVisible(true)
    scheduleControlsHide()
  }

  const hideControls = () => {
    if (controlsHideTimerRef.current) {
      window.clearTimeout(controlsHideTimerRef.current)
    }
    setControlsVisible(false)
  }

  const keepControlsVisible = () => {
    if (controlsHideTimerRef.current) {
      window.clearTimeout(controlsHideTimerRef.current)
    }
    setControlsInteracting(true)
    setControlsVisible(true)
  }

  const releaseControls = () => {
    setControlsInteracting(false)
    scheduleControlsHide()
  }

  const playEpisode = (episode?: PlayQueueEpisode) => {
    if (!episode) {
      toast.info("当前已经是最后一集")
      return
    }

    void navigate({
      to: "/play/$id",
      params: { id: String(episode.id) },
      search: {
        fromStart: false,
        resourceId: undefined,
        inventoryFileId: undefined,
      },
      replace: true,
    })
  }

  const handlePlaybackEnded = useEffectEvent(() => {
    if (playbackMode === "单集循环") {
      seekTo(playerRef, skipIntroSeconds)
      void playerRef.current?.play()
      return
    }

    if (playbackMode === "播完停止") {
      return
    }

    if (playbackMode === "播放列表循环" && !nextEpisode) {
      playEpisode(episodeItems[0])
      return
    }

    playEpisode(nextEpisode)
  })

  if (
    !hasHydrated ||
    (token &&
      (hasInventoryFilePlayback
        ? inventoryPlaybackQuery.isLoading
        : itemQuery.isLoading || playbackQuery.isLoading))
  ) {
    return <div className="min-h-svh bg-black" />
  }

  if (!token || !user) {
    return <div className="min-h-svh bg-black" />
  }

  if (!hasValidItemId) {
    return <div className="min-h-svh bg-black" />
  }

  if (itemQuery.error || playbackQuery.error || inventoryPlaybackQuery.error) {
    return <div className="min-h-svh bg-black" />
  }

  if ((!item && !hasInventoryFilePlayback) || !playback) {
    return <div className="min-h-svh bg-black" />
  }

  const controlsVisibilityClass = controlsVisible
    ? "opacity-100"
    : "pointer-events-none opacity-0"

  return (
    <SidebarProvider
      defaultOpen={false}
      style={{ "--sidebar-width": "36rem" } as React.CSSProperties}
    >
      <SidebarInset>
        <div
          ref={playerRootRef}
          className={`relative h-full w-full overflow-hidden bg-black text-white ${controlsVisible ? "" : "cursor-none"}`}
          onMouseEnter={showControls}
          onMouseLeave={hideControls}
          onMouseMove={showControls}
          onDoubleClickCapture={(event) => {
            event.preventDefault()
            event.stopPropagation()
            void requestFullscreen(playerRef, playerRootRef)
          }}
        >
          <div
            ref={playerContainerRef}
            className="mibo-custom-player absolute inset-0 z-0"
          />

          {isVideoLoading ? (
            <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center bg-black/10">
              <div className="rounded-full bg-black/45 p-5 text-white shadow-2xl ring-1 ring-white/10 backdrop-blur-sm">
                <Spinner className="size-10" />
              </div>
            </div>
          ) : null}

          {playbackFeedback ? (
            <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center">
              <div className="mibo-playback-feedback flex size-24 items-center justify-center rounded-full bg-black/45 text-white shadow-2xl ring-1 ring-white/10 backdrop-blur-sm">
                {playbackFeedback === "play" ? (
                  <PlayIcon className="ml-1 size-11 fill-current stroke-[2.5]" />
                ) : (
                  <PauseIcon className="size-11 fill-current stroke-[2.5]" />
                )}
              </div>
            </div>
          ) : null}

          <div
            className={`pointer-events-none absolute inset-x-0 top-0 z-20 flex h-[13svh] min-h-24 items-start justify-between bg-linear-to-b from-black/80 to-transparent px-10 pt-9 transition-opacity duration-200 ${controlsVisibilityClass}`}
            onMouseEnter={keepControlsVisible}
            onMouseLeave={releaseControls}
          >
            <div className="flex min-w-0 items-center gap-7">
              <button
                type="button"
                aria-label="返回"
                onClick={() => window.history.back()}
                className="pointer-events-auto flex shrink-0 items-center justify-center text-white transition-opacity hover:opacity-80"
              >
                <ChevronLeftIcon className="size-6 stroke-[2.4]" />
              </button>
              <div className="min-w-0">
                <div className="truncate text-[22px] leading-none font-semibold tracking-[-0.03em]">
                  {playbackHeader.title}
                </div>
                {playbackHeader.subtitle ? (
                  <div className="mt-2 truncate text-sm leading-none font-semibold tracking-[-0.02em] text-white/58">
                    {playbackHeader.subtitle}
                  </div>
                ) : null}
              </div>
            </div>

            <div className="flex shrink-0 items-center gap-8 text-white">
              <button
                type="button"
                aria-label="截图"
                onClick={() => void captureScreenshot(playerRef, playbackHeader)}
                className="pointer-events-auto flex shrink-0 items-center justify-center text-white transition-opacity hover:opacity-80"
              >
                <CameraIcon className="size-6 stroke-[2.4]" />
              </button>
            </div>
          </div>

          <SidebarTrigger
            className={`absolute top-1/2 right-0 z-30 flex h-20 w-14 items-center justify-center rounded-l-2xl bg-black/45 text-white/95 hover:bg-black/60 ${controlsVisibilityClass}`}
          />

          <div
            className={`absolute inset-x-0 bottom-0 z-20 h-[13svh] min-h-24 bg-linear-to-t from-black/80 to-transparent px-10 pt-7 transition-opacity duration-200 ${controlsVisibilityClass}`}
            onMouseEnter={keepControlsVisible}
            onMouseLeave={releaseControls}
            onPointerDown={keepControlsVisible}
            onPointerUp={releaseControls}
            onPointerCancel={releaseControls}
          >
            <div className="flex items-center gap-7">
              <div className="text-[17px] leading-none font-semibold tabular-nums">
                {formatClock(currentTime)}
              </div>
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
                  background: `linear-gradient(to right, #ffffff 0%, #ffffff ${progressPercent}%, rgba(255,255,255,0.42) ${progressPercent}%, rgba(255,255,255,0.42) 100%)`,
                }}
                className="h-1.5 min-w-0 flex-1 cursor-pointer appearance-none rounded-full accent-white [&::-moz-range-thumb]:size-0 [&::-webkit-slider-thumb]:size-0 [&::-webkit-slider-thumb]:appearance-none"
              />
              <div className="text-[17px] leading-none font-semibold tabular-nums">
                {formatClock(displayDuration)}
              </div>
            </div>

            <div className="mt-6 flex items-center justify-between">
              <div className="flex items-center gap-7">
                <button
                  type="button"
                  aria-label={isPaused ? "播放" : "暂停"}
                  onClick={() => void togglePlayback(playerRef)}
                  className="flex size-7 items-center justify-center text-white transition-opacity hover:opacity-80"
                >
                  {isPaused ? (
                    <PlayIcon className="size-7 fill-current stroke-[2.5]" />
                  ) : (
                    <PauseIcon className="size-7 fill-current stroke-[2.5]" />
                  )}
                </button>
                {showNextEpisodeButton ? (
                  <button
                    type="button"
                    aria-label="下一集"
                    onClick={() => playEpisode(nextEpisode)}
                    className="flex size-7 items-center justify-center text-white transition-opacity hover:opacity-80"
                  >
                    <SkipForwardIcon className="size-7 fill-current stroke-[2.5]" />
                  </button>
                ) : null}
              </div>

              <div className="flex items-center gap-7 text-[18px] font-semibold tracking-[-0.03em]">
                <SubtitleHoverMenu
                  playerRef={playerRef}
                  subtitleTracks={playback.subtitle_tracks}
                />
                <PlaybackRateHoverMenu
                  playerRef={playerRef}
                  playbackRate={playbackRate}
                  onPlaybackRateChange={setPlaybackRate}
                />
                <VolumeHoverMenu
                  playerRef={playerRef}
                  isMuted={isMuted}
                  volumePercent={volumePercent}
                  onMutedChange={setIsMuted}
                  onVolumePercentChange={setVolumePercent}
                />
                <SettingsHoverMenu
                  restorePositionEnabled={restorePositionEnabled}
                  skipIntroSeconds={skipIntroSeconds}
                  skipOutroSeconds={skipOutroSeconds}
                  playbackMode={playbackMode}
                  onSkipSettingsOpenChange={setSkipSettingsOpen}
                  onRestorePositionEnabledChange={setRestorePositionEnabled}
                  onPlaybackModeChange={setPlaybackMode}
                />
                <button
                  type="button"
                  aria-label="画中画"
                  onClick={() => void requestPictureInPicture(playerRef)}
                  className="transition-opacity hover:opacity-80"
                >
                  <PictureInPicture2Icon className="size-7 stroke-[2.4]" />
                </button>
                <button
                  type="button"
                  aria-label="全屏"
                  onClick={() =>
                    void requestFullscreen(playerRef, playerRootRef)
                  }
                  className="transition-opacity hover:opacity-80"
                >
                  <MaximizeIcon className="size-7 stroke-[2.4]" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </SidebarInset>
      <AppSidebar
        episodeItems={episodeItems}
        progressPercent={progressPercent}
        currentItemId={item?.id || 0}
        item={item}
        playbackTitle={playbackTitle}
        onEpisodeSelect={playEpisode}
        side="right"
      />
      {skipSettingsOpen ? (
        <SkipEdgeSettingsDialog
          posterUrl={posterUrl}
          playbackUrl={playback.url}
          duration={displayDuration}
          skipIntroSeconds={skipIntroSeconds}
          skipOutroSeconds={skipOutroSeconds}
          onClose={() => setSkipSettingsOpen(false)}
          onConfirm={(nextIntroSeconds, nextOutroSeconds) => {
            setSkipIntroSeconds(nextIntroSeconds)
            setSkipOutroSeconds(nextOutroSeconds)
            setSkipSettingsOpen(false)
          }}
        />
      ) : null}
    </SidebarProvider>
  )
}

async function togglePlayback(playerRef: ArtPlayerRef) {
  const player = playerRef.current
  if (!player) return

  if (!player.playing) {
    await player.play()
    return
  }

  player.pause()
}

function seekTo(playerRef: ArtPlayerRef, seconds: number) {
  const player = playerRef.current
  if (!player) return

  player.currentTime = Math.max(0, seconds)
}

function buildPlaybackHeader(
  item: CatalogItemDetail | null,
  fallbackTitle: string
) {
  if (!item || item.type !== "episode") {
    return { title: fallbackTitle, subtitle: "" }
  }

  const context = item.episode_context
  const seriesTitle = context?.series?.title?.trim()
  const seasonNumber = context?.season_number ?? context?.season?.number
  const episodeNumber = context?.episode_number
  const episodeTitle = item.title?.trim()
  const seasonEpisodeText = formatSeasonEpisodeCode(seasonNumber, episodeNumber)
  const subtitle = [seasonEpisodeText, episodeTitle]
    .filter(Boolean)
    .join("-")

  return {
    title: seriesTitle || fallbackTitle,
    subtitle,
  }
}

function formatSeasonEpisodeCode(seasonNumber?: number, episodeNumber?: number) {
  if (
    typeof seasonNumber !== "number" ||
    seasonNumber <= 0 ||
    typeof episodeNumber !== "number" ||
    episodeNumber <= 0
  ) {
    return ""
  }

  return `S${seasonNumber}:E${episodeNumber}`
}

function buildPlaybackDocumentTitle(
  item: CatalogItemDetail | null,
  fallbackTitle: string
) {
  if (!item || item.type !== "episode") {
    return fallbackTitle
  }

  const context = item.episode_context
  const seriesTitle = context?.series?.title?.trim()
  const seasonNumber = context?.season_number ?? context?.season?.number
  const episodeNumber = context?.episode_number
  const episodeTitle = item.title?.trim()
  const seasonEpisodeText = formatSeasonEpisodeCode(seasonNumber, episodeNumber)

  return [seriesTitle || fallbackTitle, seasonEpisodeText, episodeTitle]
    .filter(Boolean)
    .join("-")
}

function setPlayerVolume(playerRef: ArtPlayerRef, volumePercent: number) {
  const player = playerRef.current
  if (!player) return

  const nextPercent = Math.min(100, Math.max(0, volumePercent))
  player.volume = nextPercent / 100
  player.muted = nextPercent === 0
}

function VolumeHoverMenu({
  playerRef,
  isMuted,
  volumePercent,
  onMutedChange,
  onVolumePercentChange,
}: {
  playerRef: ArtPlayerRef
  isMuted: boolean
  volumePercent: number
  onMutedChange: (isMuted: boolean) => void
  onVolumePercentChange: (volumePercent: number) => void
}) {
  const [open, setOpen] = useState(false)
  const panelRef = useRef<HTMLDivElement | null>(null)
  const displayPercent = isMuted ? 0 : volumePercent

  const changeVolume = (nextPercent: number) => {
    const normalizedPercent = Math.min(100, Math.max(0, nextPercent))
    setPlayerVolume(playerRef, normalizedPercent)
    onVolumePercentChange(normalizedPercent)
    onMutedChange(normalizedPercent === 0)
  }

  useEffect(() => {
    const panel = panelRef.current
    if (!open || !panel) return

    const handleWheel = (event: WheelEvent) => {
      event.preventDefault()
      event.stopPropagation()

      const direction = event.deltaY < 0 ? 1 : -1
      changeVolume(displayPercent + direction * 5)
    }

    panel.addEventListener("wheel", handleWheel, { passive: false })

    return () => {
      panel.removeEventListener("wheel", handleWheel)
    }
  }, [displayPercent, open])

  const handleMuteToggle = () => {
    const player = playerRef.current
    if (!player) return

    const nextMuted = !player.muted
    player.muted = nextMuted
    onMutedChange(nextMuted)
  }

  return (
    <div
      className="group/volume relative flex w-8 justify-center"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type="button"
        aria-label="音量"
        onClick={handleMuteToggle}
        className="transition-opacity hover:opacity-80"
      >
        {isMuted || volumePercent === 0 ? (
          <VolumeXIcon className="size-7 stroke-[2.4]" />
        ) : (
          <Volume2Icon className="size-7 stroke-[2.4]" />
        )}
      </button>

      {open ? (
        <>
          <div className="absolute bottom-full left-1/2 z-40 h-4 w-16 -translate-x-1/2" />
          <div
            ref={panelRef}
            className="absolute bottom-full left-1/2 z-50 mb-4 flex w-16 -translate-x-1/2 flex-col items-center rounded-xl bg-black/80 px-3 py-4 text-white shadow-2xl ring-1 ring-white/10 backdrop-blur-xl"
          >
            <div className="mb-3 text-[13px] leading-none font-semibold tabular-nums">
              {displayPercent}%
            </div>
            <Slider
              orientation="vertical"
              min={0}
              max={100}
              step={1}
              value={[displayPercent]}
              onValueChange={([nextPercent]) => {
                if (typeof nextPercent === "number") {
                  changeVolume(nextPercent)
                }
              }}
              className="h-28 min-h-0 [&_[data-slot=slider-range]]:bg-white [&_[data-slot=slider-thumb]]:size-4 [&_[data-slot=slider-thumb]]:border-white [&_[data-slot=slider-thumb]]:bg-white [&_[data-slot=slider-track]]:bg-white/20"
            />
          </div>
        </>
      ) : null}
    </div>
  )
}

function setPlayerPlaybackRate(playerRef: ArtPlayerRef, playbackRate: number) {
  const player = playerRef.current
  if (!player) return

  player.playbackRate = playbackRate
}

function PlaybackRateHoverMenu({
  playerRef,
  playbackRate,
  onPlaybackRateChange,
}: {
  playerRef: ArtPlayerRef
  playbackRate: number
  onPlaybackRateChange: (playbackRate: number) => void
}) {
  const [open, setOpen] = useState(false)
  const [customOpen, setCustomOpen] = useState(false)
  const selectPlaybackRate = (nextRate: number) => {
    setPlayerPlaybackRate(playerRef, nextRate)
    onPlaybackRateChange(nextRate)
  }

  return (
    <div
      className="group/rate relative w-12 text-center"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => {
        setOpen(false)
        setCustomOpen(false)
      }}
    >
      <button
        type="button"
        className="w-full text-center transition-opacity hover:opacity-80"
      >
        {formatPlaybackRate(playbackRate)}
      </button>

      {open ? (
        <>
          <div className="absolute bottom-full left-1/2 z-40 h-4 w-52 -translate-x-1/2" />
          <div
            className={`absolute bottom-full left-1/2 z-50 mb-4 w-52 -translate-x-1/2 rounded-lg bg-black/80 p-2 text-white shadow-2xl ring-1 ring-white/10 backdrop-blur-xl ${customOpen ? "min-h-24" : "min-h-68"}`}
          >
            {customOpen ? (
              <CustomPlaybackRatePanel
                playbackRate={playbackRate}
                onPlaybackRateChange={selectPlaybackRate}
              />
            ) : (
              <div className="grid gap-1">
                {PLAYBACK_RATE_OPTIONS.map((rate) => {
                  const isActive = Math.abs(playbackRate - rate) < 0.01

                  return (
                    <button
                      key={rate}
                      type="button"
                      onClick={() => selectPlaybackRate(rate)}
                      className={`flex h-9 items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold transition-colors hover:bg-white/12 ${isActive ? "bg-white/16 text-white" : "text-white/72"}`}
                    >
                      <span>{formatPlaybackRate(rate)}</span>
                      {isActive ? (
                        <span className="size-1.5 rounded-full bg-white" />
                      ) : null}
                    </button>
                  )
                })}

                <button
                  type="button"
                  onPointerDown={(event) => {
                    event.preventDefault()
                    event.stopPropagation()
                    setCustomOpen(true)
                  }}
                  className="flex h-9 items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
                >
                  <span>自定义</span>
                  <span className="text-xs text-white/45">0.1-5.0</span>
                </button>
              </div>
            )}
          </div>
        </>
      ) : null}
    </div>
  )
}

function CustomPlaybackRatePanel({
  playbackRate,
  onPlaybackRateChange,
}: {
  playbackRate: number
  onPlaybackRateChange: (playbackRate: number) => void
}) {
  return (
    <div className="px-2 pt-2 pb-3">
      <div className="mb-4 flex items-center justify-between text-sm font-semibold">
        <span className="text-white/55">自定义倍速</span>
        <span>{playbackRate.toFixed(1)}x</span>
      </div>
      <div className="px-1">
        <Slider
          min={0.1}
          max={5}
          step={0.1}
          value={[playbackRate]}
          onValueChange={([nextRate]) => {
            if (typeof nextRate === "number") {
              onPlaybackRateChange(Number(nextRate.toFixed(1)))
            }
          }}
          className="[&_[data-slot=slider-range]]:bg-white [&_[data-slot=slider-thumb]]:size-4 [&_[data-slot=slider-thumb]]:border-white [&_[data-slot=slider-thumb]]:bg-white [&_[data-slot=slider-track]]:bg-white/20"
        />
      </div>
    </div>
  )
}

function SubtitleHoverMenu({
  playerRef,
  subtitleTracks,
}: {
  playerRef: ArtPlayerRef
  subtitleTracks?: Track[]
}) {
  const [open, setOpen] = useState(false)
  const [embeddedMenuOpen, setEmbeddedMenuOpen] = useState(false)
  const [externalSubtitleName, setExternalSubtitleName] = useState("")
  const [subtitlesVisible, setSubtitlesVisible] = useState(true)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [subtitleColorIndex, setSubtitleColorIndex] =
    useState<SubtitleColorIndex>(0)
  const [subtitlePositionIndex, setSubtitlePositionIndex] =
    useState<SubtitlePositionIndex>(0)
  const [subtitleSizeIndex, setSubtitleSizeIndex] = useState<SubtitleSizeIndex>(1)
  const [subtitleBackgroundOpacity, setSubtitleBackgroundOpacity] = useState(50)
  const [subtitleOffsetSeconds, setSubtitleOffsetSeconds] = useState(0)
  const externalSubtitleUrlRef = useRef<string | null>(null)
  const externalSubtitleInputRef = useRef<HTMLInputElement | null>(null)
  const tracks = subtitleTracks ?? []
  const embeddedSubtitleSummary = tracks.length
    ? `${tracks.length} 条`
    : "无"
  const externalSubtitleSummary = externalSubtitleName || "选择本地文件"

  const toggleSubtitlesVisible = () => {
    const player = playerRef.current
    const nextVisible = !subtitlesVisible

    setSubtitlesVisible(nextVisible)
    if (player) {
      player.subtitle.show = nextVisible
    }
  }

  useEffect(() => {
    return () => {
      if (externalSubtitleUrlRef.current) {
        URL.revokeObjectURL(externalSubtitleUrlRef.current)
      }
    }
  }, [])

  const loadExternalSubtitle = async (file: File) => {
    const player = playerRef.current
    if (!player) {
      toast.error("播放器尚未准备好")
      return
    }

    const subtitleUrl = URL.createObjectURL(file)
    const subtitleType = getSubtitleFileType(file.name)

    try {
      await player.subtitle.switch(subtitleUrl, {
        name: file.name,
        type: subtitleType,
      })

      if (externalSubtitleUrlRef.current) {
        URL.revokeObjectURL(externalSubtitleUrlRef.current)
      }
      externalSubtitleUrlRef.current = subtitleUrl
      setExternalSubtitleName(file.name)
      toast.success("外挂字幕已加载")
    } catch (error) {
      URL.revokeObjectURL(subtitleUrl)
      toast.error(error instanceof Error ? error.message : "外挂字幕加载失败")
    }
  }

  return (
    <div
      className="group/subtitle relative w-12 text-center"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => {
        setOpen(false)
        setEmbeddedMenuOpen(false)
        setSettingsOpen(false)
      }}
      onFocus={() => setOpen(true)}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) {
          setOpen(false)
        }
      }}
    >
      <button
        type="button"
        className="w-full text-center transition-opacity hover:opacity-80"
      >
        字幕
      </button>
      <input
        ref={externalSubtitleInputRef}
        type="file"
        accept=".srt,.vtt,.ass,text/vtt,application/x-subrip"
        className="hidden"
        onChange={(event) => {
          const file = event.target.files?.[0]
          event.currentTarget.value = ""
          if (file) {
            void loadExternalSubtitle(file)
          }
        }}
      />

      {open ? (
        <>
          <div className={`absolute bottom-full left-1/2 z-40 h-4 -translate-x-1/2 ${settingsOpen ? "w-[40rem] max-w-[calc(100vw-2rem)]" : "w-64"}`} />
          <div className={`absolute bottom-full left-1/2 z-50 mb-4 -translate-x-1/2 overflow-hidden rounded-lg bg-black/80 text-white shadow-2xl ring-1 ring-white/10 backdrop-blur-xl ${settingsOpen ? "w-max max-w-[calc(100vw-2rem)]" : "w-64 p-2"}`}>
            {settingsOpen ? (
              <SubtitleSettingsPanel
                playerRef={playerRef}
                subtitleColorIndex={subtitleColorIndex}
                subtitlePositionIndex={subtitlePositionIndex}
                subtitleSizeIndex={subtitleSizeIndex}
                subtitleBackgroundOpacity={subtitleBackgroundOpacity}
                subtitleOffsetSeconds={subtitleOffsetSeconds}
                onBack={() => setSettingsOpen(false)}
                onSubtitleColorIndexChange={setSubtitleColorIndex}
                onSubtitlePositionIndexChange={setSubtitlePositionIndex}
                onSubtitleSizeIndexChange={setSubtitleSizeIndex}
                onSubtitleBackgroundOpacityChange={setSubtitleBackgroundOpacity}
                onSubtitleOffsetSecondsChange={setSubtitleOffsetSeconds}
              />
            ) : (
            <div className="grid gap-1">
              <button
                type="button"
                onClick={toggleSubtitlesVisible}
                className="w-full rounded-lg px-3 py-2 text-left transition-colors hover:bg-white/12"
              >
                <div className="flex items-center justify-between text-[15px] font-semibold">
                  <span>显示字幕</span>
                  <span
                    className={`flex h-6 w-11 items-center rounded-full p-1 transition-colors ${subtitlesVisible ? "justify-end bg-white/28" : "justify-start bg-white/16"}`}
                  >
                    <span className="size-4 rounded-full bg-white shadow transition-transform" />
                  </span>
                </div>
                <div className="mt-1 text-xs leading-5 text-white/45">
                  当前播放源检测到 {embeddedSubtitleSummary} 字幕
                </div>
              </button>

              <div className="my-1 h-px bg-white/10" />

              <div
                className="relative"
                onMouseEnter={() => setEmbeddedMenuOpen(true)}
                onMouseLeave={() => setEmbeddedMenuOpen(false)}
                onFocus={() => setEmbeddedMenuOpen(true)}
              >
                <button
                  type="button"
                  className="flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
                >
                  <span>内嵌字幕</span>
                  <span className="flex items-center gap-1.5 text-xs text-white/45">
                    {embeddedSubtitleSummary}
                    <ChevronRightIcon className="size-4 stroke-[2.8]" />
                  </span>
                </button>

                {embeddedMenuOpen ? (
                  <>
                    <div className="absolute top-[-2.5rem] left-full z-50 h-64 w-4" />
                    <div className="absolute top-[-2.5rem] left-full z-50 ml-4 w-72 rounded-lg bg-black/80 p-2 text-white shadow-2xl ring-1 ring-white/10 backdrop-blur-xl">
                      <div className="grid gap-1">
                        {tracks.length ? (
                          tracks.map((track, index) => (
                            <div
                              key={`${track.language}-${track.title}-${index}`}
                              className="rounded-lg px-3 py-2 text-left transition-colors hover:bg-white/12"
                            >
                              <div className="flex items-center justify-between gap-3 text-[15px] font-semibold text-white/80">
                                <span className="truncate">
                                  {formatSubtitleTrackLabel(track, index)}
                                </span>
                                <span className="shrink-0 text-xs text-white/45">
                                  {track.codec || "字幕"}
                                </span>
                              </div>
                              <div className="mt-1 flex items-center gap-2 text-xs text-white/45">
                                <span>{track.language || "未知语言"}</span>
                                <span>内嵌</span>
                              </div>
                            </div>
                          ))
                        ) : (
                          <div className="rounded-lg px-3 py-2 text-left text-xs text-white/45">
                            未发现内嵌字幕
                          </div>
                        )}
                      </div>
                    </div>
                  </>
                ) : null}
              </div>

              <button
                type="button"
                onClick={() => externalSubtitleInputRef.current?.click()}
                className="flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
              >
                <span>外挂字幕</span>
                <span className="max-w-32 truncate text-xs text-white/45">
                  {externalSubtitleSummary}
                </span>
              </button>

              <button
                type="button"
                onClick={() => {
                  setEmbeddedMenuOpen(false)
                  setSettingsOpen(true)
                }}
                className="flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
              >
                <span>字幕设置</span>
                <ChevronRightIcon className="size-4 stroke-[2.8] text-white/45" />
              </button>
            </div>
            )}
          </div>
        </>
      ) : null}
    </div>
  )
}

function SubtitleSettingsPanel({
  playerRef,
  subtitleColorIndex,
  subtitlePositionIndex,
  subtitleSizeIndex,
  subtitleBackgroundOpacity,
  subtitleOffsetSeconds,
  onBack,
  onSubtitleColorIndexChange,
  onSubtitlePositionIndexChange,
  onSubtitleSizeIndexChange,
  onSubtitleBackgroundOpacityChange,
  onSubtitleOffsetSecondsChange,
}: {
  playerRef: ArtPlayerRef
  subtitleColorIndex: SubtitleColorIndex
  subtitlePositionIndex: SubtitlePositionIndex
  subtitleSizeIndex: SubtitleSizeIndex
  subtitleBackgroundOpacity: number
  subtitleOffsetSeconds: number
  onBack: () => void
  onSubtitleColorIndexChange: (index: SubtitleColorIndex) => void
  onSubtitlePositionIndexChange: (index: SubtitlePositionIndex) => void
  onSubtitleSizeIndexChange: (index: SubtitleSizeIndex) => void
  onSubtitleBackgroundOpacityChange: (opacity: number) => void
  onSubtitleOffsetSecondsChange: (seconds: number) => void
}) {
  useEffect(() => {
    applySubtitleSettings(playerRef, {
      colorIndex: subtitleColorIndex,
      positionIndex: subtitlePositionIndex,
      sizeIndex: subtitleSizeIndex,
      backgroundOpacity: subtitleBackgroundOpacity,
      offsetSeconds: subtitleOffsetSeconds,
    })
  }, [
    playerRef,
    subtitleColorIndex,
    subtitlePositionIndex,
    subtitleSizeIndex,
    subtitleBackgroundOpacity,
    subtitleOffsetSeconds,
  ])

  const resetSettings = () => {
    onSubtitleColorIndexChange(0)
    onSubtitlePositionIndexChange(0)
    onSubtitleSizeIndexChange(1)
    onSubtitleBackgroundOpacityChange(50)
    onSubtitleOffsetSecondsChange(0)
  }

  return (
    <div className="text-white">
      <div className="flex h-14 items-center justify-between border-b border-white/10 px-4">
        <button
          type="button"
          onClick={onBack}
          className="flex items-center gap-1.5 text-[15px] font-semibold transition-opacity hover:opacity-80"
        >
          <ChevronLeftIcon className="size-4 stroke-[2.8]" />
          字幕设置
        </button>
        <button
          type="button"
          onClick={resetSettings}
          className="text-xs font-semibold text-white/45 transition-colors hover:text-white/72"
        >
          恢复默认设置
        </button>
      </div>

      <div className="grid gap-4 px-4 py-4">
        <SubtitleSettingRow label="字幕颜色">
          <div className="relative w-72 shrink-0">
            <select
              value={subtitleColorIndex}
              onChange={(event) =>
                onSubtitleColorIndexChange(
                  Number(event.target.value) as SubtitleColorIndex
                )
              }
              className="h-9 w-full appearance-none rounded-lg border border-white/15 bg-white/6 px-11 pr-10 text-[15px] font-semibold text-white/72 outline-none transition-colors hover:bg-white/10"
            >
              {SUBTITLE_COLOR_OPTIONS.map((option, index) => (
                <option key={option.value} value={index}>
                  {option.label}
                </option>
              ))}
            </select>
            <span
              className="absolute top-1/2 left-3 size-5 -translate-y-1/2 rounded-sm border border-white/70 shadow"
              style={{ backgroundColor: SUBTITLE_COLOR_OPTIONS[subtitleColorIndex].value }}
            />
            <ChevronRightIcon className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 rotate-90 stroke-[2.8] text-white/72" />
          </div>
        </SubtitleSettingRow>

        <SubtitleSettingRow label="字幕位置">
          <div className="relative w-72 shrink-0">
            <select
              value={subtitlePositionIndex}
              onChange={(event) =>
                onSubtitlePositionIndexChange(
                  Number(event.target.value) as SubtitlePositionIndex
                )
              }
              className="h-9 w-full appearance-none rounded-lg border border-white/15 bg-white/6 px-11 pr-10 text-[15px] font-semibold text-white/72 outline-none transition-colors hover:bg-white/10"
            >
              {SUBTITLE_POSITION_OPTIONS.map((option, index) => (
                <option key={option.label} value={index}>
                  {option.label}
                </option>
              ))}
            </select>
            <span className="absolute top-1/2 left-3 flex size-5 -translate-y-1/2 items-end justify-center rounded-sm border border-white/70 pb-0.5">
              <span className="h-0.5 w-2.5 rounded-full bg-white" />
            </span>
            <ChevronRightIcon className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 rotate-90 stroke-[2.8] text-white/72" />
          </div>
        </SubtitleSettingRow>

        <SubtitleSettingRow label="字幕大小">
          <div className="grid h-9 w-72 shrink-0 grid-cols-4 rounded-lg bg-white/6 p-0.5">
            {SUBTITLE_SIZE_OPTIONS.map((option, index) => {
              const isActive = index === subtitleSizeIndex
              return (
                <button
                  key={option.label}
                  type="button"
                  onClick={() =>
                    onSubtitleSizeIndexChange(index as SubtitleSizeIndex)
                  }
                  className={`rounded-md text-[15px] font-semibold transition-colors ${isActive ? "bg-white/24 text-white" : "text-white/45 hover:text-white/75"}`}
                >
                  {option.label}
                </button>
              )
            })}
          </div>
        </SubtitleSettingRow>

        <SubtitleSettingRow label="背景透明度">
          <div className="flex w-72 shrink-0 items-center gap-3">
            <Slider
              min={0}
              max={100}
              step={1}
              value={[subtitleBackgroundOpacity]}
              onValueChange={([nextOpacity]) => {
                if (typeof nextOpacity === "number") {
                  onSubtitleBackgroundOpacityChange(nextOpacity)
                }
              }}
              className="[&_[data-slot=slider-range]]:bg-white [&_[data-slot=slider-thumb]]:size-5 [&_[data-slot=slider-thumb]]:border-white [&_[data-slot=slider-thumb]]:bg-white [&_[data-slot=slider-track]]:bg-white/18"
            />
            <span className="w-10 text-right text-[15px] font-semibold text-white/45">
              {subtitleBackgroundOpacity}%
            </span>
          </div>
        </SubtitleSettingRow>

        <SubtitleSettingRow label="偏移时间" hint="仅对当前视频生效">
          <div className="flex w-72 shrink-0 items-center justify-between">
            <button
              type="button"
              onClick={() =>
                onSubtitleOffsetSecondsChange(
                  Number((subtitleOffsetSeconds - 0.25).toFixed(2))
                )
              }
              className="flex size-9 items-center justify-center rounded-full bg-white/14 text-xl font-semibold text-white/80 transition-colors hover:bg-white/22"
            >
              -
            </button>
            <span className="text-[15px] font-semibold text-white/60 tabular-nums">
              {subtitleOffsetSeconds >= 0 ? "+" : ""}
              {subtitleOffsetSeconds.toFixed(2)}s
            </span>
            <button
              type="button"
              onClick={() =>
                onSubtitleOffsetSecondsChange(
                  Number((subtitleOffsetSeconds + 0.25).toFixed(2))
                )
              }
              className="flex size-9 items-center justify-center rounded-full bg-white/14 text-xl font-semibold text-white/80 transition-colors hover:bg-white/22"
            >
              +
            </button>
          </div>
        </SubtitleSettingRow>
      </div>
    </div>
  )
}

function SubtitleSettingRow({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="grid grid-cols-[7rem_1fr] items-center gap-4">
      <div className="text-left">
        <div className="text-[15px] font-semibold">{label}</div>
        {hint ? (
          <div className="mt-0.5 text-xs font-semibold text-white/38">{hint}</div>
        ) : null}
      </div>
      {children}
    </div>
  )
}

function applySubtitleSettings(
  playerRef: ArtPlayerRef,
  settings: {
    colorIndex: SubtitleColorIndex
    positionIndex: SubtitlePositionIndex
    sizeIndex: SubtitleSizeIndex
    backgroundOpacity: number
    offsetSeconds: number
  }
) {
  const player = playerRef.current
  if (!player) return

  const color = SUBTITLE_COLOR_OPTIONS[settings.colorIndex].value
  const bottom = SUBTITLE_POSITION_OPTIONS[settings.positionIndex].bottom
  const fontSize = SUBTITLE_SIZE_OPTIONS[settings.sizeIndex].fontSize
  const backgroundOpacity = Math.min(
    1,
    Math.max(0, settings.backgroundOpacity / 100)
  )

  player.subtitle.style({
    color,
    bottom: `${bottom}%`,
    fontSize: `${fontSize}px`,
    lineHeight: "1.25",
    padding: "0.2em 0.45em",
    borderRadius: "0.28em",
    backgroundColor: `rgba(0, 0, 0, ${backgroundOpacity})`,
    textShadow: "0 2px 4px rgba(0, 0, 0, 0.7)",
  })
  player.subtitleOffset = settings.offsetSeconds
}

function formatSubtitleTrackLabel(track: Track, index: number) {
  const title = track.title?.trim()
  const language = track.language?.trim()
  if (title && language) return `${title} · ${language}`
  if (title) return title
  if (language) return language
  return `字幕 ${index + 1}`
}

function getSubtitleFileType(fileName: string) {
  const extension = fileName.split(".").pop()?.toLowerCase()
  if (extension === "srt" || extension === "ass" || extension === "vtt") {
    return extension
  }
  return "vtt"
}

function SettingsHoverMenu({
  restorePositionEnabled,
  skipIntroSeconds,
  skipOutroSeconds,
  playbackMode,
  onSkipSettingsOpenChange,
  onRestorePositionEnabledChange,
  onPlaybackModeChange,
}: {
  restorePositionEnabled: boolean
  skipIntroSeconds: number
  skipOutroSeconds: number
  playbackMode: PlaybackMode
  onSkipSettingsOpenChange: (open: boolean) => void
  onRestorePositionEnabledChange: (enabled: boolean) => void
  onPlaybackModeChange: (playbackMode: PlaybackMode) => void
}) {
  const [open, setOpen] = useState(false)
  const [modeMenuOpen, setModeMenuOpen] = useState(false)
  const closeTimerRef = useRef<number | null>(null)
  const navigate = useNavigate()

  const showMenu = () => {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current)
      closeTimerRef.current = null
    }

    setOpen(true)
  }

  const scheduleCloseMenu = () => {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current)
    }

    closeTimerRef.current = window.setTimeout(() => {
      setOpen(false)
      setModeMenuOpen(false)
      closeTimerRef.current = null
    }, 180)
  }

  const skipSummary =
    skipIntroSeconds > 0 || skipOutroSeconds > 0
      ? `${skipIntroSeconds}s / ${skipOutroSeconds}s`
      : "未设置"

  return (
    <div
      className="relative -m-2 flex items-center p-2"
      onMouseEnter={showMenu}
      onMouseLeave={scheduleCloseMenu}
      onFocus={showMenu}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) {
          scheduleCloseMenu()
        }
      }}
    >
      <button
        type="button"
        aria-label="设置"
        className="transition-opacity hover:opacity-80"
      >
        <SettingsIcon className="size-7 stroke-[2.4]" />
      </button>

      {open ? (
        <>
          <div className="absolute right-0 bottom-full z-40 h-4 w-80" />
          <div className="absolute right-0 bottom-full z-50 mb-4 w-80 rounded-lg bg-black/80 p-2 text-white shadow-2xl ring-1 ring-white/10 backdrop-blur-xl">
            <button
              type="button"
              onClick={() =>
                onRestorePositionEnabledChange(!restorePositionEnabled)
              }
              className="flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
            >
              <span>自动定位上次观看位置</span>
              <span
                className={`flex h-6 w-11 items-center rounded-full p-1 transition-colors ${restorePositionEnabled ? "justify-end bg-white/28" : "justify-start bg-white/16"}`}
              >
                <span className="size-4 rounded-full bg-white shadow transition-transform" />
              </span>
            </button>

            <div
              className="relative mt-1"
              onMouseEnter={() => setModeMenuOpen(true)}
              onMouseLeave={() => setModeMenuOpen(false)}
              onFocus={() => setModeMenuOpen(true)}
            >
              <button
                type="button"
                className="flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
              >
                <span>播放模式</span>
                <span className="flex items-center gap-1.5 text-xs text-white/45">
                  {playbackMode}
                  <ChevronRightIcon className="size-4 stroke-[2.8]" />
                </span>
              </button>

              {modeMenuOpen ? (
                <>
                  <div className="absolute top-[-4.75rem] right-full z-50 h-56 w-4" />
                  <div className="absolute top-[-4.75rem] right-full z-50 mr-4 w-52 rounded-lg bg-black/80 p-2 text-white shadow-2xl ring-1 ring-white/10 backdrop-blur-xl">
                    <div className="grid gap-1">
                      {PLAYBACK_MODE_OPTIONS.map((mode) => {
                        const isActive = mode === playbackMode

                        return (
                          <button
                            key={mode}
                            type="button"
                            onClick={() => onPlaybackModeChange(mode)}
                            className={`flex h-9 items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold transition-colors hover:bg-white/12 ${isActive ? "bg-white/16 text-white" : "text-white/72"}`}
                          >
                            <span>{mode}</span>
                            {isActive ? (
                              <span className="size-1.5 rounded-full bg-white" />
                            ) : null}
                          </button>
                        )
                      })}
                    </div>
                  </div>
                </>
              ) : null}
            </div>

            <button
              type="button"
              onClick={() => {
                setOpen(false)
                setModeMenuOpen(false)
                onSkipSettingsOpenChange(true)
              }}
              className="mt-1 flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
            >
              <span>设置片头片尾</span>
              <span className="flex items-center gap-1.5 text-xs text-white/45">
                {skipSummary}
                <ChevronRightIcon className="size-4 stroke-[2.8]" />
              </span>
            </button>

            <div className="my-2 h-px bg-white/10" />

            <button
              type="button"
              onClick={() => void navigate({ to: "/settings/playback" })}
              className="flex h-9 w-full items-center gap-2.5 rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
            >
              更多设置
              <span className="rounded-md bg-white/16 px-1.5 py-0.5 text-xs font-bold tracking-normal text-white/72">
                NEW
              </span>
            </button>

            <button
              type="button"
              onClick={() =>
                toast.info(
                  "反馈入口准备中，请先在 GitHub Issue 或项目讨论区提交。"
                )
              }
              className="flex h-9 w-full items-center rounded-lg px-3 text-left text-[15px] font-semibold text-white/72 transition-colors hover:bg-white/12 hover:text-white"
            >
              意见反馈
            </button>
          </div>
        </>
      ) : null}
    </div>
  )
}

function SkipEdgeSettingsDialog({
  posterUrl,
  playbackUrl,
  duration,
  skipIntroSeconds,
  skipOutroSeconds,
  onClose,
  onConfirm,
}: {
  posterUrl?: string
  playbackUrl: string
  duration: number
  skipIntroSeconds: number
  skipOutroSeconds: number
  onClose: () => void
  onConfirm: (skipIntroSeconds: number, skipOutroSeconds: number) => void
}) {
  const safeDuration = Math.max(0, Math.floor(duration || 0))
  const maxIntroSeconds = Math.min(MAX_SKIP_EDGE_SECONDS, safeDuration)
  const maxOutroSeconds = Math.min(MAX_SKIP_EDGE_SECONDS, safeDuration)
  const [draftIntroSeconds, setDraftIntroSeconds] = useState(
    Math.min(skipIntroSeconds, maxIntroSeconds)
  )
  const [draftOutroSeconds, setDraftOutroSeconds] = useState(
    Math.min(skipOutroSeconds, maxOutroSeconds)
  )
  const [previewSeconds, setPreviewSeconds] = useState(
    Math.min(skipIntroSeconds, maxIntroSeconds)
  )
  const [videoPreviewReady, setVideoPreviewReady] = useState(false)
  const previewVideoRef = useRef<HTMLVideoElement | null>(null)
  const previewSeekTimerRef = useRef<number | null>(null)
  const outroStartSeconds = Math.max(0, safeDuration - draftOutroSeconds)

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose()
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [onClose])

  useEffect(() => {
    const video = previewVideoRef.current
    if (!video || !Number.isFinite(previewSeconds)) {
      return
    }

    if (previewSeekTimerRef.current) {
      window.clearTimeout(previewSeekTimerRef.current)
    }

    const nextTime = Math.min(Math.max(0, previewSeconds), safeDuration)
    previewSeekTimerRef.current = window.setTimeout(() => {
      if (Math.abs(video.currentTime - nextTime) > 0.15) {
        video.currentTime = nextTime
      }
      previewSeekTimerRef.current = null
    }, 180)

    return () => {
      if (previewSeekTimerRef.current) {
        window.clearTimeout(previewSeekTimerRef.current)
        previewSeekTimerRef.current = null
      }
    }
  }, [previewSeconds, safeDuration])

  return (
    <div className="fixed inset-0 z-100 flex bg-[#111] text-white">
      <div className="flex min-h-0 w-full flex-col px-8 pt-7 pb-8 sm:px-14">
        <div className="mb-8 flex items-center justify-between">
          <div className="text-2xl leading-none font-bold tracking-[-0.04em]">
            片头时长{formatSkipDuration(draftIntroSeconds)}，片尾时长
            {formatSkipDuration(draftOutroSeconds)}
          </div>
          <button
            type="button"
            aria-label="关闭片头片尾设置"
            onClick={onClose}
            className="flex size-12 items-center justify-center rounded-full text-white transition-colors hover:bg-white/10"
          >
            <XIcon className="size-8 stroke-[2.7]" />
          </button>
        </div>

        <div className="min-h-0 flex-1">
          <div className="relative h-full min-h-72 overflow-hidden rounded-xl bg-black">
            {posterUrl && !videoPreviewReady ? (
              <img
                src={posterUrl}
                alt=""
                className="absolute inset-0 h-full w-full object-contain opacity-80"
              />
            ) : null}
            <video
              ref={previewVideoRef}
              src={playbackUrl}
              muted
              playsInline
              preload="metadata"
              crossOrigin="anonymous"
              onLoadedMetadata={() => setVideoPreviewReady(true)}
              onCanPlay={() => setVideoPreviewReady(true)}
              onError={() => setVideoPreviewReady(false)}
              className={`h-full w-full object-contain ${videoPreviewReady ? "opacity-80" : "opacity-0"}`}
            />
            {!posterUrl && !videoPreviewReady ? (
              <div className="flex h-full items-center justify-center bg-linear-to-br from-slate-950 via-slate-900 to-black text-white/35">
                暂无预览画面
              </div>
            ) : null}
          </div>
        </div>

        <div className="mt-7">
          <Slider
            min={0}
            max={safeDuration}
            step={1}
            value={[draftIntroSeconds, outroStartSeconds]}
            onValueChange={([nextIntroSeconds, nextOutroStartSeconds]) => {
              const nextIntro = Math.min(
                Math.max(0, nextIntroSeconds ?? 0),
                maxIntroSeconds
              )
              const nextOutroStart = Math.max(
                Math.min(safeDuration, nextOutroStartSeconds ?? safeDuration),
                Math.max(0, safeDuration - maxOutroSeconds),
                nextIntro
              )
              const introDelta = Math.abs(nextIntro - draftIntroSeconds)
              const outroDelta = Math.abs(nextOutroStart - outroStartSeconds)

              setDraftIntroSeconds(nextIntro)
              setDraftOutroSeconds(Math.max(0, safeDuration - nextOutroStart))
              setPreviewSeconds(
                introDelta >= outroDelta ? nextIntro : nextOutroStart
              )
            }}
            className="w-full [&_[data-slot=slider-range]]:bg-white/35 [&_[data-slot=slider-thumb]]:size-5 [&_[data-slot=slider-thumb]]:border-2 [&_[data-slot=slider-thumb]]:border-white [&_[data-slot=slider-thumb]]:bg-white [&_[data-slot=slider-track]]:h-2 [&_[data-slot=slider-track]]:bg-white/18"
          />

          <div className="mt-5 flex items-start justify-between gap-4 text-xl font-bold tracking-[-0.04em]">
            <div>
              片头&nbsp;&nbsp;{formatTimelineTime(0)} -{" "}
              {formatTimelineTime(draftIntroSeconds)}
            </div>
            <div>
              <span className="text-blue-500">
                {formatTimelineTime(outroStartSeconds)}
              </span>{" "}
              - {formatTimelineTime(safeDuration)}&nbsp;&nbsp;片尾
            </div>
          </div>

          <div className="mt-10 flex items-center justify-between gap-6">
            <div className="flex items-center gap-2 text-lg font-semibold text-white/45">
              <InfoIcon className="size-6" />
              <span>仅针对同一文件夹选集生效</span>
            </div>
            <div className="flex items-center gap-5">
              <button
                type="button"
                onClick={() => {
                  setDraftIntroSeconds(0)
                  setDraftOutroSeconds(0)
                }}
                className="h-15 min-w-36 rounded-lg bg-white/10 px-8 text-xl font-bold transition-colors hover:bg-white/15"
              >
                重置
              </button>
              <button
                type="button"
                onClick={() => onConfirm(draftIntroSeconds, draftOutroSeconds)}
                className="h-15 min-w-44 rounded-lg bg-blue-600 px-8 text-xl font-bold transition-colors hover:bg-blue-500"
              >
                确认设置
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function formatSkipDuration(seconds: number) {
  if (seconds <= 0) return "0秒"

  const minutes = Math.floor(seconds / 60)
  const remainder = seconds % 60
  if (minutes > 0 && remainder > 0) {
    return `${minutes}分${remainder}秒`
  }
  if (minutes > 0) {
    return `${minutes}分`
  }
  return `${remainder}秒`
}

function formatTimelineTime(seconds: number) {
  const total = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const remainder = total % 60

  return [hours, minutes, remainder]
    .map((value) => String(value).padStart(2, "0"))
    .join(":")
}

async function requestFullscreen(
  playerRef: ArtPlayerRef,
  playerRootRef: RefObject<HTMLDivElement | null>
) {
  const root = playerRootRef.current
  if (root?.requestFullscreen) {
    if (document.fullscreenElement) {
      await document.exitFullscreen()
      return
    }

    await root.requestFullscreen()
    return
  }

  const player = playerRef.current
  if (!player) return
  player.fullscreen = !player.fullscreen
}

function requestPictureInPicture(playerRef: ArtPlayerRef) {
  const player = playerRef.current
  if (!player) return

  player.pip = !player.pip
}

async function captureScreenshot(
  playerRef: ArtPlayerRef,
  playbackHeader: { title: string; subtitle: string }
) {
  const player = playerRef.current
  if (!player) {
    toast.error("当前画面无法截图")
    return
  }

  try {
    await player.screenshot(
      safeFilename(
        [playbackHeader.title, playbackHeader.subtitle]
          .filter(Boolean)
          .join("-") || "mibo-screenshot"
      )
    )
    toast.success("截图已保存")
  } catch {
    toast.error("当前画面无法截图")
  }
}

function formatClock(seconds?: number) {
  if (!seconds || seconds <= 0) return "00:00"

  const total = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const remainder = total % 60

  if (hours > 0) {
    return [hours, minutes, remainder]
      .map((value) => String(value).padStart(2, "0"))
      .join(":")
  }

  return [minutes, remainder]
    .map((value) => String(value).padStart(2, "0"))
    .join(":")
}

function formatPlaybackRate(rate: number) {
  return rate === 1 ? "倍速" : `${rate}x`
}

function captureProgressFrame(video?: HTMLVideoElement | null) {
  if (!video || video.readyState < HTMLMediaElement.HAVE_CURRENT_DATA) {
    return undefined
  }

  const sourceWidth = video.videoWidth
  const sourceHeight = video.videoHeight
  if (sourceWidth <= 0 || sourceHeight <= 0) {
    return undefined
  }

  try {
    const width = Math.min(1280, sourceWidth)
    const height = Math.max(1, Math.round((sourceHeight / sourceWidth) * width))
    const canvas = document.createElement("canvas")
    canvas.width = width
    canvas.height = height
    const context = canvas.getContext("2d")
    if (!context) {
      return undefined
    }

    context.drawImage(video, 0, 0, width, height)
    return canvas.toDataURL("image/webp", 0.88)
  } catch {
    return undefined
  }
}

function safeFilename(value: string) {
  return value
    .trim()
    .replace(/[\\/:*?"<>|]/g, "-")
    .replace(/\s+/g, " ")
    .slice(0, 120)
}

function catalogImageUrl(
  item: { selected_images?: { image_type: string; url: string }[] },
  imageType: string
) {
  return item.selected_images?.find((image) => image.image_type === imageType)
    ?.url
}
