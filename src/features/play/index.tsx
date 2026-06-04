import {
  useCallback,
  useEffect,
  useEffectEvent,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
  type RefObject,
} from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import Artplayer from 'artplayer'
import {
  CameraIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ExternalLinkIcon,
  InfoIcon,
  LoaderCircleIcon,
  MaximizeIcon,
  PauseIcon,
  PictureInPicture2Icon,
  PlayIcon,
  SettingsIcon,
  SkipBackIcon,
  SkipForwardIcon,
  Volume2Icon,
  VolumeXIcon,
  XIcon,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { formatLanguageCode, normalizeLanguageCode } from '@/lib/language-code'
import type {
  CatalogItemDetail,
  LiveTVPlaybackSource,
  MetadataResourceDetail,
  PlaybackSource,
  PlaybackVariant,
  SubtitleSearchProvider,
  Track,
} from '@/lib/mibo-api'
import {
  catalogPlaybackQueryOptions,
  createAuthedMiboApi,
  inventoryFilePlaybackQueryOptions,
  liveTVChannelsQueryOptions,
  liveTVPlaybackQueryOptions,
  metadataItemDetailQueryOptions,
  metadataItemProgressQueryOptions,
  metadataItemResourcesQueryOptions,
  miboQueryKeys,
  userSettingsQueryOptions,
} from '@/lib/mibo-query'
import { useProtectedSessionRedirect } from '@/lib/use-protected-session-redirect'
import { useIsMobile } from '@/hooks/use-mobile'
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
} from '@/components/ui/drawer'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { Slider } from '@/components/ui/slider'
import { Spinner } from '@/components/ui/spinner'
import { SidebarEdgeTrigger } from '@/components/layout/sidebar-edge-trigger'
import {
  attachLiveTVStream,
  destroyLiveTVStream,
} from '@/features/live-tv/live-tv-stream'
import { formatComputedResourceVariantLabel } from '@/features/media/components/standalone-media-detail-utils'
import { openConfiguredExternalPlayer } from '@/features/play/external-player'
import { AppSidebar } from './components/app-sidebar'
import { formatSubtitleTrackMenuLabel } from './subtitle-track-label'

Artplayer.DEBUG = true
Artplayer.DBCLICK_FULLSCREEN = false

type ArtPlayerRef = RefObject<Artplayer | null>

const PLAYBACK_RATE_OPTIONS = [5, 3, 2, 1.5, 1, 0.8]

const PLAYBACK_MODE_OPTIONS = [
  '自动连播',
  '单集循环',
  '播放列表循环',
  '播完停止',
] as const

const MAX_SKIP_EDGE_SECONDS = 300
const PROGRESS_SAVE_INTERVAL_MS = 30_000
const PROGRESS_MIN_SAVE_INTERVAL_MS = 15_000
const PROGRESS_FORCE_SAVE_MIN_INTERVAL_MS = 15_000
const PROGRESS_SAVE_SEEK_DELTA_SECONDS = 90
const PROGRESS_FRAME_SAVE_INTERVAL_MS = 120_000
const TRANSCODE_PLAYLIST_REFRESH_INTERVAL_MS = 3_000

const SUBTITLE_COLOR_OPTIONS = [
  { label: '白色（默认）', value: '#ffffff' },
  { label: '黄色', value: '#ffe66d' },
  { label: '蓝色', value: '#8ec5ff' },
  { label: '绿色', value: '#a7f3a7' },
] as const

const SUBTITLE_POSITION_OPTIONS = [
  { label: '中间下（默认）', bottom: 8 },
  { label: '底部', bottom: 4 },
  { label: '中间', bottom: 42 },
  { label: '顶部', bottom: 78 },
] as const

const FOREGROUND_SOLID = 'hsl(var(--foreground))'

const SUBTITLE_SIZE_OPTIONS = [
  { label: '小', fontSize: 28 },
  { label: '标准', fontSize: 36 },
  { label: '大', fontSize: 44 },
  { label: '极大', fontSize: 52 },
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
  liveTVChannelId?: number
  liveTVSourceId?: number
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
  liveTVChannelId,
  liveTVSourceId,
}: PlayExperienceProps) {
  const token = useAuthStore((state) => state.auth.accessToken)
  const user = useAuthStore((state) => state.auth.user)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const isCheckingSession = useProtectedSessionRedirect()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const queryToken = token ?? 'guest'
  const hasValidItemId = Number.isFinite(itemId) && itemId > 0
  const isLiveTVPlayback =
    Number.isFinite(liveTVChannelId) && (liveTVChannelId ?? 0) > 0
  const hasInventoryFilePlayback =
    Number.isFinite(inventoryFileId) && (inventoryFileId ?? 0) > 0
  const playerRef = useRef<Artplayer | null>(null)
  const playerRootRef = useRef<HTMLDivElement | null>(null)
  const playerContainerRef = useRef<HTMLDivElement | null>(null)
  const restoreAppliedRef = useRef(false)
  const saveInFlightRef = useRef(false)
  const lastSavedPositionRef = useRef(0)
  const lastSavedAtRef = useRef(0)
  const lastSaveAttemptAtRef = useRef(0)
  const lastFrameSavedAtRef = useRef(0)
  const skipOutroSecondsRef = useRef(0)
  const pendingSeekSecondsRef = useRef<number | null>(null)
  const pendingTimelineSeekRef = useRef<number | null>(null)
  const activeTranscodeSessionRef = useRef<string | null>(null)
  const currentPlaybackStreamUrlRef = useRef<string | null>(null)
  const controlsHideTimerRef = useRef<number | null>(null)
  const playbackFeedbackTimerRef = useRef<number | null>(null)
  const wasPausedRef = useRef(true)
  const [duration, setDuration] = useState(0)
  const [currentTime, setCurrentTime] = useState(0)
  const [isPaused, setIsPaused] = useState(true)
  const [isMuted, setIsMuted] = useState(false)
  const [volumePercent, setVolumePercent] = useState(100)
  const [playbackRate, setPlaybackRate] = useState(1)
  const [playbackMode, setPlaybackMode] = useState<PlaybackMode>('自动连播')
  const [selectedVariantId, setSelectedVariantId] = useState('original')
  const [selectedAudioStreamIndex, setSelectedAudioStreamIndex] = useState<
    number | undefined
  >(undefined)
  const [variantStartSeconds, setVariantStartSeconds] = useState(0)
  const [restorePositionEnabled, setRestorePositionEnabled] =
    useState(!fromStart)
  const [skipIntroSeconds, setSkipIntroSeconds] = useState(0)
  const [skipOutroSeconds, setSkipOutroSeconds] = useState(0)
  const [skipSettingsOpen, setSkipSettingsOpen] = useState(false)
  const [controlsVisible, setControlsVisible] = useState(true)
  const [controlsInteracting, setControlsInteracting] = useState(false)
  const [isVideoLoading, setIsVideoLoading] = useState(true)
  const [playbackFeedback, setPlaybackFeedback] = useState<
    'play' | 'pause' | null
  >(null)

  const itemQuery = useQuery({
    ...metadataItemDetailQueryOptions(queryToken, itemId),
    enabled:
      hasHydrated &&
      !!token &&
      hasValidItemId &&
      !hasInventoryFilePlayback &&
      !isLiveTVPlayback,
  })
  const progressQuery = useQuery({
    ...metadataItemProgressQueryOptions(queryToken, itemId),
    enabled:
      hasHydrated &&
      !!token &&
      hasValidItemId &&
      !hasInventoryFilePlayback &&
      !isLiveTVPlayback,
  })
  const playbackQuery = useQuery({
    ...catalogPlaybackQueryOptions(queryToken, itemId, {
      resourceId,
      variant: selectedVariantId,
      startSeconds: variantStartSeconds,
      audioStreamIndex: selectedAudioStreamIndex,
    }),
    enabled:
      hasHydrated &&
      !!token &&
      hasValidItemId &&
      !hasInventoryFilePlayback &&
      !isLiveTVPlayback,
  })
  const resourcesQuery = useQuery({
    ...metadataItemResourcesQueryOptions(queryToken, itemId),
    enabled:
      hasHydrated &&
      !!token &&
      hasValidItemId &&
      !hasInventoryFilePlayback &&
      !isLiveTVPlayback,
  })
  const inventoryPlaybackQuery = useQuery({
    ...inventoryFilePlaybackQueryOptions(queryToken, inventoryFileId ?? 0, {
      variant: selectedVariantId,
      startSeconds: variantStartSeconds,
      audioStreamIndex: selectedAudioStreamIndex,
    }),
    enabled:
      hasHydrated && !!token && hasInventoryFilePlayback && !isLiveTVPlayback,
  })
  const liveTVPlaybackQuery = useQuery({
    ...liveTVPlaybackQueryOptions(queryToken, liveTVChannelId ?? 0),
    enabled: hasHydrated && !!token && isLiveTVPlayback,
  })
  const liveTVSourceChannelsQuery = useQuery({
    ...liveTVChannelsQueryOptions(queryToken, {
      source_id: liveTVSourceId,
      enabled: true,
    }),
    enabled:
      hasHydrated &&
      !!token &&
      isLiveTVPlayback &&
      typeof liveTVSourceId === 'number' &&
      liveTVSourceId > 0,
  })
  const userSettingsQuery = useQuery({
    ...userSettingsQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const probePlaybackFileMutation = useMutation({
    mutationFn: async (fileId: number) => {
      if (!token) {
        throw new Error('当前未登录，无法探测媒体资源')
      }
      return createAuthedMiboApi(token).reprobeInventoryFile(fileId)
    },
    onSuccess: () => {
      toast.success('已加入探测队列，稍后会刷新音轨和字幕信息')
      refreshPlaybackAfterProbe()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '探测资源失败')
    },
  })
  const item = itemQuery.data ?? null
  const progress = progressQuery.data ?? null
  const liveTVPlayback = useMemo(
    () => normalizeLiveTVPlayback(liveTVPlaybackQuery.data),
    [liveTVPlaybackQuery.data]
  )
  const playback = hasInventoryFilePlayback
    ? (inventoryPlaybackQuery.data ?? null)
    : isLiveTVPlayback
      ? liveTVPlayback
      : (playbackQuery.data ?? null)
  const isPlaybackSourceLoading = hasInventoryFilePlayback
    ? inventoryPlaybackQuery.isLoading
    : isLiveTVPlayback
      ? liveTVPlaybackQuery.isLoading
      : itemQuery.isLoading || playbackQuery.isLoading
  const isPlaybackSourceFetching = hasInventoryFilePlayback
    ? inventoryPlaybackQuery.isFetching
    : isLiveTVPlayback
      ? liveTVPlaybackQuery.isFetching
      : playbackQuery.isFetching

  const [activePartIndex, setActivePartIndex] = useState(0)
  const playbackParts = useMemo(
    () =>
      playback?.parts?.length
        ? playback.parts
        : playback
          ? [
              {
                part_index: 1,
                file_id: playback.file_id ?? 0,
                title: playback.title,
                container: playback.container,
                url: playback.url,
                direct: playback.direct,
                size_bytes: playback.size_bytes,
              },
            ]
          : [],
    [playback]
  )
  const activePlaybackPart = playbackParts[activePartIndex] ?? playbackParts[0]
  const multipartDurationSeconds = totalPlaybackPartsDuration(playbackParts)
  const posterUrl = item
    ? catalogImageUrl(item, 'backdrop') || catalogImageUrl(item, 'poster')
    : undefined
  const playbackTitle = item?.title ?? playback?.title ?? '整理中媒体'
  const playbackHeader = buildPlaybackHeader(item, playbackTitle)
  const openCurrentPlaybackInExternalPlayer = useCallback(() => {
    const playbackUrl = activePlaybackPart?.url || playback?.url || ''
    const launchResult = openConfiguredExternalPlayer({
      playbackUrl,
      title: playbackTitle,
    })

    if (!launchResult.ok) {
      toast.error(launchResult.message)
    }
  }, [activePlaybackPart?.url, playback?.url, playbackTitle])
  const playbackInfoPanel = useMemo(
    () =>
      buildPlaybackInfoPanel({
        item,
        playback,
        activePlaybackPart,
        partCount: playbackParts.length,
      }),
    [activePlaybackPart, item, playback, playbackParts.length]
  )
  const displayDuration =
    multipartDurationSeconds ||
    playback?.runtime_seconds ||
    item?.runtime_seconds ||
    duration ||
    0
  const progressPercent =
    displayDuration > 0
      ? Math.min(100, (currentTime / displayDuration) * 100)
      : 0
  const playbackTimelineOffset =
    playback?.selected_variant?.hls && !isLiveTVPlayback
      ? variantStartSeconds
      : 0
  const selectedPlaybackVariantId = playback?.selected_variant?.id ?? 'original'
  const selectedPlaybackAudioStreamIndex =
    playback?.selected_audio_stream_index ?? selectedAudioStreamIndex
  const isAwaitingSelectedAudioStream =
    !isLiveTVPlayback &&
    typeof selectedAudioStreamIndex === 'number' &&
    playback?.selected_audio_stream_index !== selectedAudioStreamIndex
  const canProbePlaybackResource =
    !isLiveTVPlayback &&
    typeof playback?.file_id === 'number' &&
    playback.file_id > 0
  const shouldShowProbePlaybackButton =
    canProbePlaybackResource &&
    (playback?.probe_status !== 'ready' ||
      (playback?.audio_tracks?.length ?? 0) === 0)
  const isAwaitingSelectedVariant =
    !isLiveTVPlayback &&
    selectedVariantId !== 'original' &&
    selectedPlaybackVariantId !== selectedVariantId
  const selectedPlaybackManifestUrl =
    playback?.hls_manifest_url || playback?.selected_variant?.manifest || ''
  const playbackStreamUrl =
    isAwaitingSelectedVariant || isAwaitingSelectedAudioStream
      ? ''
      : playback?.selected_variant?.hls
        ? selectedPlaybackManifestUrl || playback.url
        : (activePlaybackPart?.url ?? playback?.url ?? '')
  const episodeItems = (
    item?.same_season_episodes?.length
      ? item.same_season_episodes
      : (item?.seasons?.flatMap((season) => season.episodes ?? []) ?? [])
  ).filter((episode) => {
    if (episode.id === item?.id) {
      return true
    }

    return (
      Boolean(episode.inventory_file_id) ||
      episode.availability_status === 'available' ||
      episode.availability_status === 'partial'
    )
  })
  const playbackPartItems = playbackParts.map((part) => ({
    id: part.file_id,
    title: part.title?.trim() || `第 ${part.part_index} 段`,
    label: `第 ${part.part_index} 段`,
    runtime_seconds: part.duration_seconds,
  }))
  const showPlaybackPartsOverview =
    !isLiveTVPlayback && item?.type === 'movie' && playbackPartItems.length > 1
  const liveChannelItems = isLiveTVPlayback
    ? (liveTVSourceChannelsQuery.data ?? []).map((channel) => ({
        id: channel.id,
        sourceId: channel.source_id,
        title: channel.name,
        label: channel.name,
      }))
    : []
  const overviewItems = episodeItems.length
    ? episodeItems
    : liveChannelItems.length
      ? liveChannelItems
      : showPlaybackPartsOverview
        ? playbackPartItems
        : []
  const overviewTabLabel = episodeItems.length
    ? '选集'
    : liveChannelItems.length
      ? '频道'
      : '分段'
  const currentOverviewItemId = episodeItems.length
    ? item?.id || 0
    : liveChannelItems.length
      ? (liveTVChannelId ?? 0)
      : showPlaybackPartsOverview
        ? (activePlaybackPart?.file_id ?? playbackPartItems[0]?.id ?? 0)
        : 0
  const currentEpisodeIndex = episodeItems.findIndex(
    (episode) => episode.id === item?.id
  )
  const previousEpisode =
    currentEpisodeIndex > 0 ? episodeItems[currentEpisodeIndex - 1] : undefined
  const nextEpisode =
    currentEpisodeIndex >= 0 ? episodeItems[currentEpisodeIndex + 1] : undefined
  const previousPlaybackPart = playbackParts[activePartIndex - 1]
  const nextPlaybackPart = playbackParts[activePartIndex + 1]
  const currentLiveChannelIndex = liveChannelItems.findIndex(
    (channel) => channel.id === liveTVChannelId
  )
  const previousLiveChannel =
    currentLiveChannelIndex > 0
      ? liveChannelItems[currentLiveChannelIndex - 1]
      : undefined
  const nextLiveChannel =
    currentLiveChannelIndex >= 0
      ? liveChannelItems[currentLiveChannelIndex + 1]
      : undefined
  const showPreviousEpisodeButton = isLiveTVPlayback
    ? Boolean(previousLiveChannel)
    : Boolean(previousPlaybackPart) || Boolean(previousEpisode)
  const showNextEpisodeButton = isLiveTVPlayback
    ? Boolean(nextLiveChannel)
    : Boolean(nextPlaybackPart) || Boolean(nextEpisode)
  const skipBackLabel = isLiveTVPlayback
    ? '上一频道'
    : previousPlaybackPart
      ? '上一段'
      : item?.type === 'episode'
        ? '上一集'
        : '上一段'
  const skipForwardLabel = isLiveTVPlayback
    ? '下一频道'
    : nextPlaybackPart
      ? '下一段'
      : item?.type === 'episode'
        ? '下一集'
        : '下一段'

  useEffect(() => {
    skipOutroSecondsRef.current = skipOutroSeconds
  }, [skipOutroSeconds])

  useEffect(() => {
    setRestorePositionEnabled(!fromStart)
  }, [fromStart, itemId, resourceId, liveTVChannelId])

  useEffect(() => {
    document.title = buildPlaybackDocumentTitle(item, playbackTitle)
  }, [item, playbackTitle])

  const persistProgress = useEffectEvent(
    async ({ force = false, completed = false } = {}) => {
      if (
        !token ||
        isLiveTVPlayback ||
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
        : (activePlaybackPart?.duration_seconds ??
          playback.runtime_seconds ??
          item?.runtime_seconds ??
          0)
      const durationSeconds = Math.round(
        multipartDurationSeconds || rawDuration || 0
      )
      const localPositionSeconds = Math.max(
        0,
        Math.round(player.currentTime || 0)
      )
      const positionSeconds = aggregatedPlaybackPosition(
        playbackParts,
        activePartIndex,
        localPositionSeconds
      )
      const now = Date.now()
      const positionDelta = Math.abs(
        positionSeconds - lastSavedPositionRef.current
      )
      const lastSaveAt = Math.max(
        lastSavedAtRef.current,
        lastSaveAttemptAtRef.current
      )
      const elapsedSinceLastSave = now - lastSaveAt

      if (!completed && positionSeconds <= 0) {
        return
      }

      if (
        !completed &&
        lastSaveAt > 0 &&
        elapsedSinceLastSave < PROGRESS_MIN_SAVE_INTERVAL_MS
      ) {
        return
      }

      if (
        force &&
        !completed &&
        lastSaveAt > 0 &&
        elapsedSinceLastSave < PROGRESS_FORCE_SAVE_MIN_INTERVAL_MS &&
        positionDelta < PROGRESS_SAVE_SEEK_DELTA_SECONDS
      ) {
        return
      }

      if (!force && !completed) {
        if (positionSeconds <= 0) {
          return
        }

        const shouldSaveByInterval =
          elapsedSinceLastSave >= PROGRESS_SAVE_INTERVAL_MS
        const shouldSaveBySeek =
          positionDelta >= PROGRESS_SAVE_SEEK_DELTA_SECONDS

        if (!shouldSaveByInterval && !shouldSaveBySeek) {
          return
        }
      }

      lastSaveAttemptAtRef.current = now
      saveInFlightRef.current = true

      try {
        if (hasInventoryFilePlayback || !item) return
        const progressMetadataItemId = playback.metadata_item_id
        if (!progressMetadataItemId) return
        const progressResourceId =
          typeof playback.resource_id === 'number' && playback.resource_id > 0
            ? playback.resource_id
            : undefined
        const shouldCaptureFrame =
          !completed &&
          positionSeconds > 0 &&
          now - lastFrameSavedAtRef.current > PROGRESS_FRAME_SAVE_INTERVAL_MS
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

  const releasePlaybackTranscode = useEffectEvent(
    (
      streamUrl?: string,
      options?: { keepalive?: boolean; delayMs?: number; force?: boolean }
    ) => {
      const sessionId = extractTranscodeSessionId(streamUrl)
      if (!token || !sessionId) {
        return
      }
      const release = () => {
        if (
          !options?.force &&
          activeTranscodeSessionRef.current === sessionId
        ) {
          return
        }
        void createAuthedMiboApi(token)
          .releaseTranscodeSession(sessionId, {
            keepalive: options?.keepalive,
          })
          .catch(() => {
            // Best-effort cleanup: the session may already be stopped or expired.
          })
      }
      if (options?.delayMs && options.delayMs > 0) {
        window.setTimeout(release, options.delayMs)
        return
      }
      release()
    }
  )

  useEffect(() => {
    currentPlaybackStreamUrlRef.current = playbackStreamUrl || null
    activeTranscodeSessionRef.current =
      extractTranscodeSessionId(playbackStreamUrl) ?? null
  }, [playbackStreamUrl])

  useEffect(() => {
    return () => {
      void persistProgress({ force: true })
      releasePlaybackTranscode(
        currentPlaybackStreamUrlRef.current ?? undefined,
        {
          keepalive: true,
          force: true,
        }
      )
    }
  }, [])

  useEffect(() => {
    restoreAppliedRef.current = false
    lastSavedAtRef.current = 0
    lastSaveAttemptAtRef.current = 0
    lastFrameSavedAtRef.current = 0
    setSelectedVariantId('original')
    setSelectedAudioStreamIndex(undefined)
    setVariantStartSeconds(0)
    if (activePartIndex !== 0) {
      setActivePartIndex(0)
    }
  }, [activePartIndex, itemId, resourceId, liveTVChannelId, liveTVSourceId])

  useEffect(() => {
    lastSavedPositionRef.current = progress?.position_seconds ?? 0
  }, [progress?.position_seconds])

  const showPlaybackFeedback = useEffectEvent((feedback: 'play' | 'pause') => {
    if (playbackFeedbackTimerRef.current) {
      window.clearTimeout(playbackFeedbackTimerRef.current)
    }

    setPlaybackFeedback(feedback)
    playbackFeedbackTimerRef.current = window.setTimeout(() => {
      setPlaybackFeedback(null)
      playbackFeedbackTimerRef.current = null
    }, 520)
  })

  function playEpisode(episode?: PlayQueueEpisode) {
    if (!episode) {
      toast.info('当前已经是最后一集')
      return
    }

    void navigate({
      to: '/play/$id',
      params: { id: String(episode.id) },
      search: {
        fromStart: false,
        resourceId: undefined,
        inventoryFileId: undefined,
        liveChannelId: undefined,
        liveSourceId: undefined,
      },
      replace: true,
    })
  }

  function playLiveChannel(channel?: { id: number; sourceId?: number }) {
    if (!channel) {
      toast.info('没有可切换的频道')
      return
    }

    void navigate({
      to: '/play/$id',
      params: { id: String(channel.id) },
      search: {
        fromStart: undefined,
        inventoryFileId: undefined,
        resourceId: undefined,
        liveChannelId: channel.id,
        liveSourceId: channel.sourceId,
      },
      replace: true,
    })
  }

  function selectPlaybackResource(nextResourceId: number) {
    if (!Number.isFinite(nextResourceId) || nextResourceId <= 0) {
      return
    }

    void navigate({
      to: '/play/$id',
      params: { id: String(itemId) },
      search: {
        fromStart: false,
        resourceId: nextResourceId,
        inventoryFileId: undefined,
        liveChannelId: undefined,
        liveSourceId: undefined,
      },
      replace: true,
    })
  }

  function playPlaybackPartByFileId(fileId: number) {
    const nextPartIndex = playbackParts.findIndex(
      (part) => part.file_id === fileId
    )
    if (nextPartIndex < 0) {
      toast.info('未找到对应分段')
      return
    }

    if (nextPartIndex === activePartIndex) {
      return
    }

    setActivePartIndex(nextPartIndex)
  }

  function selectPlaybackVariant(variantId: string) {
    const normalizedVariant = variantId.trim() || 'original'
    if (normalizedVariant === selectedVariantId) {
      return
    }
    const timelineSeconds = currentPlaybackTimelineSeconds()
    if (normalizedVariant === 'original') {
      pendingSeekSecondsRef.current = Math.max(0, Math.floor(timelineSeconds))
    }
    setVariantStartSeconds(Math.max(0, Math.floor(timelineSeconds)))
    setSelectedVariantId(normalizedVariant)
    setIsVideoLoading(true)
  }

  function selectAudioTrack(track?: Track) {
    if (!track || typeof track.stream_index !== 'number') {
      return
    }
    if (track.stream_index === selectedAudioStreamIndex) {
      return
    }
    const timelineSeconds = currentPlaybackTimelineSeconds()
    setVariantStartSeconds(Math.max(0, Math.floor(timelineSeconds)))
    setSelectedAudioStreamIndex(track.stream_index)
    if (selectedVariantId === 'original') {
      setSelectedVariantId('audio-repair')
    }
    setIsVideoLoading(true)
  }

  function refreshPlaybackAfterProbe() {
    const refresh = () => {
      if (hasInventoryFilePlayback && inventoryFileId) {
        void queryClient.invalidateQueries({
          queryKey: ['inventory-file', 'playback', queryToken, inventoryFileId],
        })
        return
      }
      void queryClient.invalidateQueries({
        queryKey: ['catalog', 'playback', queryToken, itemId],
      })
      void queryClient.invalidateQueries({
        queryKey: miboQueryKeys.catalogItemDetail(queryToken, itemId),
      })
    }

    refresh()
    window.setTimeout(refresh, 2500)
    window.setTimeout(refresh, 7000)
  }

  function probePlaybackResource() {
    const fileId = playback?.file_id
    if (!fileId) {
      toast.info('当前播放源没有可探测的文件')
      return
    }
    probePlaybackFileMutation.mutate(fileId)
  }

  function currentPlaybackTimelineSeconds() {
    if (playback?.selected_variant?.hls) {
      return Math.max(0, currentTime)
    }
    const player = playerRef.current
    const currentPartSeconds =
      player && Number.isFinite(player.currentTime) ? player.currentTime : 0
    return currentPartSeconds + playbackTimelineOffset
  }

  function playPreviousEpisode(episode?: PlayQueueEpisode) {
    if (!episode) {
      toast.info('当前已经是第一集')
      return
    }

    void navigate({
      to: '/play/$id',
      params: { id: String(episode.id) },
      search: {
        fromStart: false,
        resourceId: undefined,
        inventoryFileId: undefined,
        liveChannelId: undefined,
        liveSourceId: undefined,
      },
      replace: true,
    })
  }

  function playPreviousPlaybackPart() {
    if (!previousPlaybackPart) {
      toast.info('当前已经是第一段')
      return
    }

    setActivePartIndex((current) => current - 1)
  }

  function playNextPlaybackPart() {
    if (!nextPlaybackPart) {
      toast.info('当前已经是最后一段')
      return
    }

    setActivePartIndex((current) => current + 1)
  }

  function handleSkipForward() {
    if (isLiveTVPlayback) {
      playLiveChannel(nextLiveChannel)
      return
    }

    if (nextPlaybackPart) {
      playNextPlaybackPart()
      return
    }

    if (item?.type === 'episode') {
      playEpisode(nextEpisode)
      return
    }

    toast.info('当前已经是最后一段')
  }

  function handleSkipBack() {
    if (isLiveTVPlayback) {
      playLiveChannel(previousLiveChannel)
      return
    }

    if (previousPlaybackPart) {
      playPreviousPlaybackPart()
      return
    }

    if (item?.type === 'episode') {
      playPreviousEpisode(previousEpisode)
      return
    }

    toast.info('当前已经是第一段')
  }

  const handlePlaybackEnded = useEffectEvent(() => {
    if (isLiveTVPlayback) {
      return
    }

    if (
      playbackParts.length > 1 &&
      activePartIndex < playbackParts.length - 1
    ) {
      setActivePartIndex((current) => current + 1)
      return
    }

    if (playbackMode === '单集循环') {
      seekTo(playerRef, skipIntroSeconds)
      void playerRef.current?.play()
      return
    }

    if (playbackMode === '播完停止') {
      return
    }

    if (playbackMode === '播放列表循环' && !nextEpisode) {
      playEpisode(episodeItems[0])
      return
    }

    playEpisode(nextEpisode)
  })

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

    const resolvedPosition = resolvePlaybackPartPosition(
      playbackParts,
      savedPosition
    )
    if (resolvedPosition.partIndex !== activePartIndex) {
      pendingSeekSecondsRef.current = resolvedPosition.localSeconds
      setActivePartIndex(resolvedPosition.partIndex)
      return
    }

    const playerDuration = Number.isFinite(player.duration)
      ? player.duration
      : Infinity
    const target = Math.min(
      resolvedPosition.localSeconds,
      Math.max(0, playerDuration - 3)
    )
    if (target <= 0) {
      return
    }

    player.currentTime = target
    pendingSeekSecondsRef.current = null
    restoreAppliedRef.current = true
  })

  const seekToTimelinePosition = useCallback(
    (seconds: number) => {
      const targetSeconds = Math.max(0, seconds)
      const player = playerRef.current
      if (!player) {
        return
      }
      const resolvedPosition = resolvePlaybackPartPosition(
        playbackParts,
        targetSeconds
      )
      if (playback?.selected_variant?.hls) {
        setVariantStartSeconds(
          Math.max(0, Math.floor(resolvedPosition.localSeconds))
        )
        setCurrentTime(targetSeconds)
        setIsVideoLoading(true)
        return
      }
      if (resolvedPosition.partIndex !== activePartIndex) {
        pendingSeekSecondsRef.current = resolvedPosition.localSeconds
        restoreAppliedRef.current = true
        setActivePartIndex(resolvedPosition.partIndex)
        setCurrentTime(targetSeconds)
        return
      }
      player.currentTime = Math.max(0, resolvedPosition.localSeconds)
      setCurrentTime(targetSeconds)
    },
    [activePartIndex, playback?.selected_variant?.hls, playbackParts]
  )

  const updateTimelineSeekPreview = useCallback(
    (seconds: number) => {
      const targetSeconds = Math.max(0, seconds)
      if (playback?.selected_variant?.hls) {
        pendingTimelineSeekRef.current = targetSeconds
        setCurrentTime(targetSeconds)
        return
      }
      seekToTimelinePosition(targetSeconds)
    },
    [playback?.selected_variant?.hls, seekToTimelinePosition]
  )

  const commitTimelineSeekPreview = useCallback(() => {
    const targetSeconds = pendingTimelineSeekRef.current
    pendingTimelineSeekRef.current = null
    if (targetSeconds == null) {
      return
    }
    seekToTimelinePosition(targetSeconds)
  }, [seekToTimelinePosition])

  useEffect(() => {
    restoreProgress()
  }, [
    activePartIndex,
    progress?.position_seconds,
    fromStart,
    restorePositionEnabled,
  ])

  useEffect(() => {
    const container = playerContainerRef.current
    if (
      !container ||
      !playback ||
      !playbackStreamUrl ||
      (!item && !hasInventoryFilePlayback && !isLiveTVPlayback)
    ) {
      return
    }

    const player = new Artplayer({
      container,
      url: playbackStreamUrl,
      ...(resolveArtPlayerType(playback, playbackStreamUrl)
        ? { type: resolveArtPlayerType(playback, playbackStreamUrl) }
        : {}),
      ...(posterUrl ? { poster: posterUrl } : {}),
      autoplay: true,
      isLive: isLiveTVPlayback,
      playsInline: true,
      theme: FOREGROUND_SOLID,
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
      customType: {
        m3u8: (video, url) => {
          if (
            attachLiveTVStream(video, url, {
              token,
              playlistRefreshIntervalMs: isTranscodeStreamUrl(url)
                ? TRANSCODE_PLAYLIST_REFRESH_INTERVAL_MS
                : undefined,
            })
          ) {
            return
          }

          throw new Error('当前浏览器不支持 HLS 直播播放')
        },
      },
    })

    playerRef.current = player

    const syncState = () => {
      const currentPartSeconds = player.currentTime || 0
      const timelineSeconds =
        playbackTimelineOffset > 0 &&
        currentPartSeconds >= playbackTimelineOffset
          ? currentPartSeconds
          : currentPartSeconds + playbackTimelineOffset
      setCurrentTime(
        playbackParts.length > 1
          ? aggregatedPlaybackPosition(
              playbackParts,
              activePartIndex,
              Math.round(timelineSeconds)
            )
          : timelineSeconds
      )
      setDuration(Number.isFinite(player.duration) ? player.duration : 0)
      setIsPaused(!player.playing)
      setIsMuted(player.muted)
      setVolumePercent(Math.round((player.volume ?? 1) * 100))
      setPlaybackRate(player.playbackRate)
    }

    const handlePause = () => {
      syncState()
      showPlaybackFeedback('pause')
      void persistProgress({ force: true })
    }
    const handlePlay = () => {
      syncState()
      showPlaybackFeedback('play')
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
      if (pendingSeekSecondsRef.current != null) {
        player.currentTime = Math.max(0, pendingSeekSecondsRef.current)
        pendingSeekSecondsRef.current = null
        restoreAppliedRef.current = true
        return
      }
      restoreProgress()
    }
    const handleEnded = () => {
      syncState()
      void persistProgress({
        force: true,
        completed: activePartIndex >= playbackParts.length - 1,
      })
      handlePlaybackEnded()
    }
    const handleVolumeChange = () => {
      syncState()
    }
    const handleRateChange = () => {
      syncState()
    }

    syncState()
    player.on('ready', handleLoadedMetadata)
    player.on('video:pause', handlePause)
    player.on('video:play', handlePlay)
    player.on('video:playing', handleVideoReady)
    player.on('video:timeupdate', handleTimeUpdate)
    player.on('video:loadedmetadata', handleLoadedMetadata)
    player.on('video:loadeddata', handleVideoReady)
    player.on('video:canplay', handleVideoReady)
    player.on('video:waiting', handleVideoLoading)
    player.on('video:seeking', handleVideoLoading)
    player.on('video:seeked', handleVideoReady)
    player.on('video:ended', handleEnded)
    player.on('video:volumechange', handleVolumeChange)
    player.on('video:ratechange', handleRateChange)

    return () => {
      void persistProgress({ force: true })
      destroyLiveTVStream(player.video)
      playerRef.current = null
      player.destroy(false)
    }
  }, [
    activePartIndex,
    activePlaybackPart?.url,
    hasInventoryFilePlayback,
    isLiveTVPlayback,
    item,
    playback,
    playbackParts,
    playbackStreamUrl,
    playbackTimelineOffset,
    playbackTitle,
    posterUrl,
    token,
  ])

  useEffect(() => {
    if (!playback) {
      return
    }

    const handlePageHide = () => {
      void persistProgress({ force: true })
      releasePlaybackTranscode(playbackStreamUrl, { keepalive: true })
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
  }, [playback, playbackStreamUrl])

  useEffect(() => {
    const player = playerRef.current
    if (!player || skipIntroSeconds <= 0 || restoreAppliedRef.current) {
      return
    }

    const current = Math.round(player.currentTime || 0)
    if (current < skipIntroSeconds) {
      player.currentTime = skipIntroSeconds
    }
  }, [
    activePartIndex,
    skipIntroSeconds,
    activePlaybackPart?.url,
    playback?.url,
  ])

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

  const scheduleControlsHide = useCallback(() => {
    if (isPaused) {
      setControlsVisible(true)
      return
    }

    if (controlsHideTimerRef.current) {
      window.clearTimeout(controlsHideTimerRef.current)
    }

    controlsHideTimerRef.current = window.setTimeout(() => {
      if (!controlsInteracting && !isPaused) {
        setControlsVisible(false)
      }
    }, 2200)
  }, [controlsInteracting, isPaused])

  useEffect(() => {
    const wasPaused = wasPausedRef.current
    wasPausedRef.current = isPaused

    if (controlsHideTimerRef.current) {
      window.clearTimeout(controlsHideTimerRef.current)
      controlsHideTimerRef.current = null
    }

    if (isPaused) {
      setControlsVisible(true)
      return
    }

    if (wasPaused || !controlsInteracting) {
      scheduleControlsHide()
    }
  }, [controlsInteracting, isPaused, scheduleControlsHide])

  const showControls = () => {
    setControlsVisible(true)
    scheduleControlsHide()
  }

  const hideControls = () => {
    if (controlsHideTimerRef.current) {
      window.clearTimeout(controlsHideTimerRef.current)
    }
    if (isPaused) {
      setControlsVisible(true)
      return
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
    if (isPaused) {
      setControlsVisible(true)
      return
    }
    scheduleControlsHide()
  }

  if (isCheckingSession) {
    return (
      <div className='flex h-svh w-full items-center justify-center bg-background text-foreground'>
        <div className='flex items-center gap-3 rounded-full border border-border/60 bg-card px-5 py-3 backdrop-blur-xl'>
          <LoaderCircleIcon className='size-4 animate-spin' />
          <span className='text-sm text-muted-foreground'>
            正在验证播放权限
          </span>
        </div>
      </div>
    )
  }

  if (!hasHydrated || (token && isPlaybackSourceLoading)) {
    return <PlaybackLoadingScreen label='正在加载播放源' />
  }

  if (!token || !user) {
    return <div className='min-h-svh bg-background' />
  }

  if (!hasValidItemId && !isLiveTVPlayback) {
    return <div className='min-h-svh bg-background' />
  }

  if (
    itemQuery.error ||
    playbackQuery.error ||
    inventoryPlaybackQuery.error ||
    liveTVPlaybackQuery.error
  ) {
    return <div className='min-h-svh bg-background' />
  }

  if ((!item && !hasInventoryFilePlayback && !isLiveTVPlayback) || !playback) {
    return <div className='min-h-svh bg-background' />
  }

  const controlsVisibilityClass = controlsVisible
    ? 'opacity-100'
    : 'pointer-events-none opacity-0'

  return (
    <SidebarProvider
      defaultOpen={false}
      style={{ '--sidebar-width': '36rem' } as CSSProperties}
    >
      <SidebarInset className='bg-background'>
        <div
          ref={playerRootRef}
          className={`relative aspect-video w-full overflow-hidden bg-background text-foreground md:aspect-auto md:h-full ${controlsVisible ? '' : 'cursor-none'}`}
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
            className='mibo-custom-player absolute inset-0 z-0'
          />

          {isVideoLoading ||
          isAwaitingSelectedVariant ||
          isAwaitingSelectedAudioStream ||
          isPlaybackSourceFetching ? (
            <div className='pointer-events-none absolute inset-0 z-10 flex items-center justify-center bg-background/10'>
              <div className='rounded-full border border-border/60 bg-card/80 p-4 text-foreground shadow-2xl backdrop-blur-sm md:p-5'>
                <Spinner className='size-8 md:size-10' />
              </div>
            </div>
          ) : null}

          {playbackFeedback ? (
            <div className='pointer-events-none absolute inset-0 z-10 flex items-center justify-center'>
              <div className='mibo-playback-feedback flex size-20 items-center justify-center rounded-full border border-border/60 bg-card/80 text-foreground shadow-2xl backdrop-blur-sm md:size-24'>
                {playbackFeedback === 'play' ? (
                  <PlayIcon className='ml-1 size-9 fill-current stroke-[2.5] md:size-11' />
                ) : (
                  <PauseIcon className='size-9 fill-current stroke-[2.5] md:size-11' />
                )}
              </div>
            </div>
          ) : null}

          <div
            className={`pointer-events-none absolute inset-x-0 top-0 z-20 flex h-[12svh] min-h-20 items-start justify-between bg-linear-to-b from-background/85 to-transparent px-5 pt-5 transition-opacity duration-200 md:h-[13svh] md:min-h-24 md:px-10 md:pt-9 ${controlsVisibilityClass}`}
            onMouseEnter={keepControlsVisible}
            onMouseLeave={releaseControls}
          >
            <div className='flex min-w-0 items-center gap-4 md:gap-7'>
              <button
                type='button'
                aria-label='返回'
                onClick={() => window.history.back()}
                className='pointer-events-auto flex shrink-0 items-center justify-center text-foreground transition-opacity hover:opacity-80'
              >
                <ChevronLeftIcon className='size-5 stroke-[2.4] md:size-6' />
              </button>
              <div className='min-w-0'>
                <div className='truncate text-lg leading-none font-semibold tracking-[-0.03em] md:text-[22px]'>
                  {playbackHeader.title}
                </div>
                {playbackHeader.subtitle ? (
                  <div className='mt-1.5 truncate text-xs leading-none font-semibold tracking-[-0.02em] text-foreground/58 md:mt-2 md:text-sm'>
                    {playbackHeader.subtitle}
                  </div>
                ) : null}
              </div>
            </div>

            <div className='flex shrink-0 items-center gap-4 text-foreground md:gap-8'>
              <button
                type='button'
                aria-label='截图'
                onClick={() =>
                  void captureScreenshot(playerRef, playbackHeader)
                }
                className='pointer-events-auto flex shrink-0 items-center justify-center text-foreground transition-opacity hover:opacity-80'
              >
                <CameraIcon className='size-5 stroke-[2.4] md:size-6' />
              </button>
            </div>
          </div>

          <SidebarEdgeTrigger
            side='right'
            className={`!right-0 z-30 hidden text-foreground transition-opacity duration-200 md:block ${controlsVisibilityClass}`}
          />

          <div
            className={`absolute inset-x-0 bottom-0 z-20 bg-linear-to-t from-background/85 to-transparent px-5 pt-2.5 pb-2 transition-opacity duration-200 md:h-[13svh] md:min-h-24 md:px-10 md:pt-7 md:pb-0 ${controlsVisibilityClass}`}
            onMouseEnter={keepControlsVisible}
            onMouseLeave={releaseControls}
            onPointerDown={keepControlsVisible}
            onPointerUp={releaseControls}
            onPointerCancel={releaseControls}
          >
            {isLiveTVPlayback ? (
              <div className='flex items-center justify-between gap-3 text-sm font-semibold md:text-[17px]'>
                <div className='flex items-center gap-2'>
                  <span className='inline-flex size-2 rounded-full bg-red-500' />
                  <span>直播中</span>
                </div>
                <div className='truncate text-xs text-foreground/58 md:text-sm'>
                  直播频道不提供时间轴拖动与进度保存
                </div>
              </div>
            ) : (
              <div className='flex items-center gap-3 md:gap-7'>
                <div className='text-sm leading-none font-semibold tabular-nums md:text-[17px]'>
                  {formatClock(currentTime)}
                </div>
                <input
                  type='range'
                  min='0'
                  max={Math.max(displayDuration, 0)}
                  step='1'
                  value={Math.min(currentTime, displayDuration || currentTime)}
                  onChange={(event) => {
                    const nextTime = Number(event.target.value)
                    updateTimelineSeekPreview(nextTime)
                  }}
                  onPointerUp={commitTimelineSeekPreview}
                  onPointerCancel={commitTimelineSeekPreview}
                  onKeyUp={commitTimelineSeekPreview}
                  onBlur={commitTimelineSeekPreview}
                  style={{
                    background: `linear-gradient(to right, #ffffff 0%, #ffffff ${progressPercent}%, rgba(255,255,255,0.42) ${progressPercent}%, rgba(255,255,255,0.42) 100%)`,
                  }}
                  className='h-1 min-w-0 flex-1 cursor-pointer appearance-none rounded-full accent-white md:h-1.5 [&::-moz-range-thumb]:size-0 [&::-webkit-slider-thumb]:size-0 [&::-webkit-slider-thumb]:appearance-none'
                />
                <div className='text-sm leading-none font-semibold tabular-nums md:text-[17px]'>
                  {formatClock(displayDuration)}
                </div>
              </div>
            )}

            <div className='mt-3 flex items-center justify-between md:mt-6'>
              <div className='flex items-center gap-4 md:gap-7'>
                <button
                  type='button'
                  aria-label={isPaused ? '播放' : '暂停'}
                  onClick={() => void togglePlayback(playerRef)}
                  className='flex size-5 items-center justify-center text-foreground transition-opacity hover:opacity-80 md:size-7'
                >
                  {isPaused ? (
                    <PlayIcon className='size-5 fill-current stroke-[2.5] md:size-7' />
                  ) : (
                    <PauseIcon className='size-5 fill-current stroke-[2.5] md:size-7' />
                  )}
                </button>
                {showPreviousEpisodeButton ? (
                  <button
                    type='button'
                    aria-label={skipBackLabel}
                    onClick={handleSkipBack}
                    className='flex size-5 items-center justify-center text-foreground transition-opacity hover:opacity-80 md:size-7'
                  >
                    <SkipBackIcon className='size-5 fill-current stroke-[2.5] md:size-7' />
                  </button>
                ) : null}
                {showNextEpisodeButton ? (
                  <button
                    type='button'
                    aria-label={skipForwardLabel}
                    onClick={handleSkipForward}
                    className='flex size-5 items-center justify-center text-foreground transition-opacity hover:opacity-80 md:size-7'
                  >
                    <SkipForwardIcon className='size-5 fill-current stroke-[2.5] md:size-7' />
                  </button>
                ) : null}
              </div>

              <div className='flex items-center gap-3 text-xs font-semibold tracking-[-0.03em] md:gap-7 md:text-[18px]'>
                <SubtitleHoverMenu
                  playerRef={playerRef}
                  itemId={itemId}
                  resourceId={playback.resource_id}
                  inventoryFileId={
                    hasInventoryFilePlayback ? inventoryFileId : undefined
                  }
                  subtitleTracks={playback.subtitle_tracks}
                  subtitleSearchProviders={playback.subtitle_search_providers}
                  defaultSubtitleMode={
                    userSettingsQuery.data?.playback.default_subtitle_mode
                  }
                  preferredSubtitleLanguage={
                    userSettingsQuery.data?.playback.preferred_subtitle_language
                  }
                />
                {!isLiveTVPlayback ? (
                  <AudioTrackHoverMenu
                    tracks={playback.audio_tracks}
                    selectedAudioStreamIndex={selectedPlaybackAudioStreamIndex}
                    onSelectAudioTrack={selectAudioTrack}
                  />
                ) : null}
                {!isLiveTVPlayback ? (
                  <VersionHoverMenu
                    resources={resourcesQuery.data}
                    selectedResourceId={playback.resource_id ?? resourceId}
                    onSelectResource={selectPlaybackResource}
                  />
                ) : null}
                {!isLiveTVPlayback ? (
                  <QualityHoverMenu
                    variants={playback.variants}
                    selectedVariantId={
                      playback.selected_variant?.id ?? selectedVariantId
                    }
                    onSelectVariant={selectPlaybackVariant}
                  />
                ) : null}
                {!isLiveTVPlayback ? (
                  <PlaybackRateHoverMenu
                    playerRef={playerRef}
                    playbackRate={playbackRate}
                    onPlaybackRateChange={setPlaybackRate}
                  />
                ) : null}
                <VolumeHoverMenu
                  playerRef={playerRef}
                  isMuted={isMuted}
                  volumePercent={volumePercent}
                  onMutedChange={setIsMuted}
                  onVolumePercentChange={setVolumePercent}
                />
                {!isLiveTVPlayback ? (
                  <SettingsHoverMenu
                    restorePositionEnabled={restorePositionEnabled}
                    skipIntroSeconds={skipIntroSeconds}
                    skipOutroSeconds={skipOutroSeconds}
                    playbackMode={playbackMode}
                    canProbePlaybackResource={shouldShowProbePlaybackButton}
                    isProbingPlaybackResource={
                      probePlaybackFileMutation.isPending
                    }
                    onOpenExternalPlayer={openCurrentPlaybackInExternalPlayer}
                    onProbePlaybackResource={probePlaybackResource}
                    onSkipSettingsOpenChange={setSkipSettingsOpen}
                    onRestorePositionEnabledChange={setRestorePositionEnabled}
                    onPlaybackModeChange={setPlaybackMode}
                  />
                ) : null}
                <button
                  type='button'
                  aria-label='画中画'
                  onClick={() => void requestPictureInPicture(playerRef)}
                  className='transition-opacity hover:opacity-80'
                >
                  <PictureInPicture2Icon className='size-5 stroke-[2.4] md:size-7' />
                </button>
                <button
                  type='button'
                  aria-label='全屏'
                  onClick={() =>
                    void requestFullscreen(playerRef, playerRootRef)
                  }
                  className='transition-opacity hover:opacity-80'
                >
                  <MaximizeIcon className='size-5 stroke-[2.4] md:size-7' />
                </button>
              </div>
            </div>
          </div>
        </div>
        <div className='md:hidden'>
          <AppSidebar
            inlineOnMobile
            overviewItems={overviewItems}
            overviewTabLabel={overviewTabLabel}
            progressPercent={progressPercent}
            currentOverviewItemId={currentOverviewItemId}
            item={item}
            playbackTitle={playbackTitle}
            playbackFacts={playbackInfoPanel.facts}
            onOverviewItemSelect={(selectedItem) => {
              if (isLiveTVPlayback) {
                playLiveChannel(selectedItem)
                return
              }

              if (episodeItems.length) {
                playEpisode(selectedItem)
                return
              }

              if (showPlaybackPartsOverview) {
                playPlaybackPartByFileId(selectedItem.id)
              }
            }}
            side='right'
          />
        </div>
      </SidebarInset>
      <div className='hidden md:block'>
        <AppSidebar
          overviewItems={overviewItems}
          overviewTabLabel={overviewTabLabel}
          progressPercent={progressPercent}
          currentOverviewItemId={currentOverviewItemId}
          item={item}
          playbackTitle={playbackTitle}
          playbackFacts={playbackInfoPanel.facts}
          onOverviewItemSelect={(selectedItem) => {
            if (isLiveTVPlayback) {
              playLiveChannel(selectedItem)
              return
            }

            if (episodeItems.length) {
              playEpisode(selectedItem)
              return
            }

            if (showPlaybackPartsOverview) {
              playPlaybackPartByFileId(selectedItem.id)
            }
          }}
          side='right'
        />
      </div>
      {!isLiveTVPlayback && skipSettingsOpen ? (
        <SkipEdgeSettingsDialog
          posterUrl={posterUrl}
          playbackUrl={playbackStreamUrl}
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
function resolveArtPlayerType(playback: PlaybackSource | null, url?: string) {
  if (!playback) {
    return undefined
  }

  const lowerURL = (url || playback.url).toLowerCase()
  const lowerContainer = playback.container.toLowerCase()

  if (lowerContainer === 'hls' || lowerURL.includes('.m3u8')) {
    return 'm3u8'
  }

  if (playback.type === 'live_tv_channel' && lowerContainer === 'stream') {
    return 'm3u8'
  }

  return undefined
}
function normalizeLiveTVPlayback(
  playback: LiveTVPlaybackSource | undefined
): PlaybackSource | null {
  if (!playback) {
    return null
  }

  return {
    metadata_item_id: undefined,
    resource_id: undefined,
    file_id: playback.channel_id,
    title: playback.title,
    type: playback.type,
    container: playback.container || 'ts',
    url: playback.url,
    direct: playback.direct,
    size_bytes: 0,
    runtime_seconds: undefined,
    segment_index: undefined,
    start_seconds: undefined,
    end_seconds: undefined,
    edition: playback.group_name || '直播',
    video_codec: '',
    width: undefined,
    height: undefined,
    audio_tracks: [],
    subtitle_tracks: [],
    subtitle_search_providers: [],
    parts: undefined,
    checks: [],
    playable: playback.playable,
    decision: {
      kind: playback.direct ? 'direct' : 'fallback',
      client_profile: 'web',
      selected_by: `live-tv:${playback.stream_mode || 'proxy'}`,
      reasons: [],
    },
  }
}
function buildPlaybackHeader(
  item: CatalogItemDetail | null,
  fallbackTitle: string
) {
  if (!item || item.type !== 'episode')
    return { title: fallbackTitle, subtitle: '' }
  const context = item.episode_context
  const seriesTitle = context?.series?.title?.trim()
  const seasonNumber = context?.season_number ?? context?.season?.number
  const episodeNumber = context?.episode_number
  const episodeTitle = item.title?.trim()
  const seasonEpisodeText = formatSeasonEpisodeCode(seasonNumber, episodeNumber)
  const subtitle = [seasonEpisodeText, episodeTitle].filter(Boolean).join('-')
  return { title: seriesTitle || fallbackTitle, subtitle }
}
function buildPlaybackInfoPanel({
  item,
  playback,
  activePlaybackPart,
  partCount,
}: {
  item: CatalogItemDetail | null
  playback: PlaybackSource | null
  activePlaybackPart?: { container?: string } | null
  partCount: number
}) {
  if (!playback) {
    return { badges: [], facts: [] }
  }

  const edition = playback.edition?.trim() || ''
  const year = formatReleaseYear(item)
  const officialRating = formatOfficialRating(item?.official_rating)
  const genre = formatGenreLabel(item?.genres)
  const resolution = formatResolutionLabel(playback.width, playback.height)
  const codec = formatCodecLabel(playback.video_codec)
  const audio = formatAudioTrackLabel(playback.audio_tracks?.[0])
  const subtitles = formatSubtitleCountLabel(playback.subtitle_tracks)
  const mode = formatPlaybackModeBadge(playback)
  const container = formatContainerLabel(
    activePlaybackPart?.container || playback.container
  )
  const parts = partCount > 1 ? `${partCount} 段` : '单文件'

  return {
    badges: [
      edition,
      resolution,
      codec,
      audio,
      subtitles,
      mode,
      partCount > 1 ? parts : '',
    ].filter(Boolean),
    facts: [
      { label: '版本', value: edition },
      { label: '年份', value: year },
      { label: '分级', value: officialRating.replace(/^分级\s*/, '') },
      { label: '类型', value: genre },
      { label: '画质', value: resolution },
      { label: '编码', value: codec },
      { label: '音频', value: audio },
      { label: '字幕', value: subtitles },
      { label: '播放方式', value: mode },
      { label: '封装', value: container },
      { label: '文件结构', value: parts },
    ].filter((fact) => fact.value),
  }
}
function formatSeasonEpisodeCode(
  seasonNumber?: number,
  episodeNumber?: number
) {
  if (
    typeof seasonNumber !== 'number' ||
    seasonNumber <= 0 ||
    typeof episodeNumber !== 'number' ||
    episodeNumber <= 0
  )
    return ''
  return `S${seasonNumber}:E${episodeNumber}`
}
function buildPlaybackDocumentTitle(
  item: CatalogItemDetail | null,
  fallbackTitle: string
) {
  if (!item || item.type !== 'episode') return fallbackTitle
  const context = item.episode_context
  const seriesTitle = context?.series?.title?.trim()
  const seasonNumber = context?.season_number ?? context?.season?.number
  const episodeNumber = context?.episode_number
  const episodeTitle = item.title?.trim()
  const seasonEpisodeText = formatSeasonEpisodeCode(seasonNumber, episodeNumber)
  return [seriesTitle || fallbackTitle, seasonEpisodeText, episodeTitle]
    .filter(Boolean)
    .join('-')
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
  const isMobile = useIsMobile()
  const [open, setOpen] = useState(false)
  const panelRef = useRef<HTMLDivElement | null>(null)
  const displayPercent = isMuted ? 0 : volumePercent
  const changeVolume = useCallback(
    (nextPercent: number) => {
      const normalizedPercent = Math.min(100, Math.max(0, nextPercent))
      setPlayerVolume(playerRef, normalizedPercent)
      onVolumePercentChange(normalizedPercent)
      onMutedChange(normalizedPercent === 0)
    },
    [onMutedChange, onVolumePercentChange, playerRef]
  )
  useEffect(() => {
    const panel = panelRef.current
    if (!open || !panel) return
    const handleWheel = (event: WheelEvent) => {
      event.preventDefault()
      event.stopPropagation()
      const direction = event.deltaY < 0 ? 1 : -1
      changeVolume(displayPercent + direction * 5)
    }
    panel.addEventListener('wheel', handleWheel, { passive: false })
    return () => {
      panel.removeEventListener('wheel', handleWheel)
    }
  }, [changeVolume, displayPercent, open])
  const handleMuteToggle = () => {
    const player = playerRef.current
    if (!player) return
    const nextMuted = !player.muted
    player.muted = nextMuted
    onMutedChange(nextMuted)
  }

  if (isMobile) {
    return (
      <>
        <button
          type='button'
          aria-label='音量'
          onClick={() => setOpen(true)}
          className='transition-opacity hover:opacity-80'
        >
          {isMuted || volumePercent === 0 ? (
            <VolumeXIcon className='size-5 stroke-[2.4] md:size-7' />
          ) : (
            <Volume2Icon className='size-5 stroke-[2.4] md:size-7' />
          )}
        </button>
        <Drawer open={open} onOpenChange={setOpen}>
          <DrawerContent className='border-border bg-background text-foreground'>
            <DrawerHeader className='px-5 pt-4 pb-2 text-left'>
              <DrawerTitle>音量</DrawerTitle>
            </DrawerHeader>
            <div className='px-5 pb-6'>
              <button
                type='button'
                onClick={handleMuteToggle}
                className='mb-4 flex h-11 w-full items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-sm font-semibold text-foreground/82'
              >
                <span>
                  {isMuted || volumePercent === 0 ? '取消静音' : '静音'}
                </span>
                <span className='text-muted-foreground'>{displayPercent}%</span>
              </button>
              <Slider
                min={0}
                max={100}
                step={1}
                value={[displayPercent]}
                onValueChange={([nextPercent]) => {
                  if (typeof nextPercent === 'number') {
                    changeVolume(nextPercent)
                  }
                }}
                className='[&_[data-slot=slider-range]]:bg-foreground [&_[data-slot=slider-thumb]]:size-4 [&_[data-slot=slider-thumb]]:border-foreground [&_[data-slot=slider-thumb]]:bg-foreground [&_[data-slot=slider-track]]:bg-muted'
              />
            </div>
          </DrawerContent>
        </Drawer>
      </>
    )
  }
  return (
    <div
      className='group/volume relative flex w-7 justify-center md:w-8'
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type='button'
        aria-label='音量'
        onClick={handleMuteToggle}
        className='transition-opacity hover:opacity-80'
      >
        {isMuted || volumePercent === 0 ? (
          <VolumeXIcon className='size-5 stroke-[2.4] md:size-7' />
        ) : (
          <Volume2Icon className='size-5 stroke-[2.4] md:size-7' />
        )}
      </button>
      {open ? (
        <>
          <div className='absolute bottom-full left-1/2 z-40 h-4 w-16 -translate-x-1/2' />
          <div
            ref={panelRef}
            className='absolute bottom-full left-1/2 z-50 mb-4 flex w-16 -translate-x-1/2 flex-col items-center rounded-xl border border-border/60 bg-popover px-3 py-4 text-popover-foreground shadow-2xl backdrop-blur-xl'
          >
            <div className='mb-3 text-[13px] leading-none font-semibold tabular-nums'>
              {displayPercent}%
            </div>
            <Slider
              orientation='vertical'
              min={0}
              max={100}
              step={1}
              value={[displayPercent]}
              onValueChange={([nextPercent]) => {
                if (typeof nextPercent === 'number') {
                  changeVolume(nextPercent)
                }
              }}
              className='h-28 min-h-0 [&_[data-slot=slider-range]]:bg-foreground [&_[data-slot=slider-thumb]]:size-4 [&_[data-slot=slider-thumb]]:border-foreground [&_[data-slot=slider-thumb]]:bg-foreground [&_[data-slot=slider-track]]:bg-muted'
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

function AudioTrackHoverMenu({
  tracks,
  selectedAudioStreamIndex,
  onSelectAudioTrack,
}: {
  tracks?: Track[]
  selectedAudioStreamIndex?: number
  onSelectAudioTrack: (track: Track) => void
}) {
  const isMobile = useIsMobile()
  const [open, setOpen] = useState(false)
  const audioTracks = (tracks ?? []).filter(
    (track) => typeof track.stream_index === 'number'
  )
  const selectedTrack =
    audioTracks.find(
      (track) => track.stream_index === selectedAudioStreamIndex
    ) ?? audioTracks[0]

  if (audioTracks.length <= 1) {
    return null
  }

  const selectTrack = (track: Track) => {
    onSelectAudioTrack(track)
    setOpen(false)
  }

  if (isMobile) {
    return (
      <>
        <button
          type='button'
          onClick={() => setOpen(true)}
          className='w-full text-center whitespace-nowrap transition-opacity hover:opacity-80'
        >
          音轨
        </button>
        <Drawer open={open} onOpenChange={setOpen}>
          <DrawerContent className='border-border bg-background text-foreground'>
            <DrawerHeader className='px-5 pt-4 pb-2 text-left'>
              <DrawerTitle>音轨</DrawerTitle>
            </DrawerHeader>
            <div className='grid gap-2 px-5 pb-6'>
              {audioTracks.map((track, index) => {
                const isActive =
                  track.stream_index === selectedTrack?.stream_index
                return (
                  <button
                    key={getAudioTrackKey(track, index)}
                    type='button'
                    onClick={() => selectTrack(track)}
                    className={`flex min-h-11 w-full items-center justify-between gap-3 overflow-hidden rounded-xl border px-4 py-2 text-left text-sm font-semibold transition-colors ${isActive ? 'border-border bg-accent text-accent-foreground' : 'border-border/60 bg-muted/30 text-muted-foreground'}`}
                  >
                    <span className='min-w-0 truncate'>
                      {formatAudioTrackMenuLabel(track, index)}
                    </span>
                    <span className='shrink-0 text-xs'>
                      {formatAudioTrackLabel(track) || '音频'}
                    </span>
                  </button>
                )
              })}
            </div>
          </DrawerContent>
        </Drawer>
      </>
    )
  }

  return (
    <div
      className='group/audio relative w-10 text-center md:w-12'
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type='button'
        className='w-full text-center whitespace-nowrap transition-opacity hover:opacity-80'
      >
        音轨
      </button>
      {open ? (
        <>
          <div className='absolute bottom-full left-1/2 z-40 h-4 w-72 -translate-x-1/2' />
          <div className='absolute bottom-full left-1/2 z-50 mb-4 w-72 -translate-x-1/2 rounded-lg border border-border/60 bg-popover p-2 text-popover-foreground shadow-2xl backdrop-blur-xl'>
            <div className='grid max-h-72 gap-1 overflow-y-auto pr-1'>
              {audioTracks.map((track, index) => {
                const isActive =
                  track.stream_index === selectedTrack?.stream_index
                return (
                  <button
                    key={getAudioTrackKey(track, index)}
                    type='button'
                    onClick={() => selectTrack(track)}
                    className={`w-full overflow-hidden rounded-lg px-3 py-2 text-left transition-colors hover:bg-accent ${isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}`}
                  >
                    <div className='flex min-w-0 items-center justify-between gap-3 text-[15px] font-semibold'>
                      <span className='min-w-0 truncate'>
                        {formatAudioTrackMenuLabel(track, index)}
                      </span>
                      <span className='shrink-0 text-xs text-muted-foreground'>
                        {formatAudioTrackLabel(track) || '音频'}
                      </span>
                    </div>
                    <div className='mt-1 text-xs text-muted-foreground'>
                      流 {track.stream_index}
                    </div>
                  </button>
                )
              })}
            </div>
          </div>
        </>
      ) : null}
    </div>
  )
}

function VersionHoverMenu({
  resources,
  selectedResourceId,
  onSelectResource,
}: {
  resources?: MetadataResourceDetail[]
  selectedResourceId?: number
  onSelectResource: (resourceId: number) => void
}) {
  const isMobile = useIsMobile()
  const [open, setOpen] = useState(false)
  const playbackResources = (resources ?? []).filter(isPlaybackVersionResource)
  const selectedResource =
    playbackResources.find((resource) => resource.id === selectedResourceId) ??
    playbackResources[0]

  if (playbackResources.length <= 1) {
    return null
  }

  const selectResource = (resource: MetadataResourceDetail) => {
    if (resource.id === selectedResource?.id) {
      setOpen(false)
      return
    }
    onSelectResource(resource.id)
    setOpen(false)
  }

  if (isMobile) {
    return (
      <>
        <button
          type='button'
          onClick={() => setOpen(true)}
          className='w-full text-center whitespace-nowrap transition-opacity hover:opacity-80'
        >
          版本
        </button>
        <Drawer open={open} onOpenChange={setOpen}>
          <DrawerContent className='border-border bg-background text-foreground'>
            <DrawerHeader className='px-5 pt-4 pb-2 text-left'>
              <DrawerTitle>播放版本</DrawerTitle>
            </DrawerHeader>
            <div className='grid gap-2 px-5 pb-6'>
              {playbackResources.map((resource, index) => {
                const isActive = resource.id === selectedResource?.id
                return (
                  <button
                    key={resource.id}
                    type='button'
                    onClick={() => selectResource(resource)}
                    className={`flex min-h-11 w-full items-center overflow-hidden rounded-xl border px-4 py-2 text-left text-sm font-semibold transition-colors ${isActive ? 'border-border bg-accent text-accent-foreground' : 'border-border/60 bg-muted/30 text-muted-foreground'}`}
                  >
                    <span className='min-w-0 truncate'>
                      {formatComputedResourceVariantLabel(
                        resource,
                        playbackResources,
                        index
                      )}
                    </span>
                  </button>
                )
              })}
            </div>
          </DrawerContent>
        </Drawer>
      </>
    )
  }

  return (
    <div
      className='group/version relative w-10 text-center md:w-12'
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type='button'
        className='w-full text-center whitespace-nowrap transition-opacity hover:opacity-80'
      >
        版本
      </button>
      {open ? (
        <>
          <div className='absolute bottom-full left-1/2 z-40 h-4 w-72 -translate-x-1/2' />
          <div className='absolute bottom-full left-1/2 z-50 mb-4 w-72 -translate-x-1/2 rounded-lg border border-border/60 bg-popover p-2 text-popover-foreground shadow-2xl backdrop-blur-xl'>
            <div className='grid max-h-72 gap-1 overflow-y-auto pr-1'>
              {playbackResources.map((resource, index) => {
                const isActive = resource.id === selectedResource?.id
                return (
                  <button
                    key={resource.id}
                    type='button'
                    onClick={() => selectResource(resource)}
                    className={`w-full overflow-hidden rounded-lg px-3 py-2 text-left transition-colors hover:bg-accent ${isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}`}
                  >
                    <div className='truncate text-[15px] font-semibold'>
                      {formatComputedResourceVariantLabel(
                        resource,
                        playbackResources,
                        index
                      )}
                    </div>
                  </button>
                )
              })}
            </div>
          </div>
        </>
      ) : null}
    </div>
  )
}

function QualityHoverMenu({
  variants,
  selectedVariantId,
  onSelectVariant,
}: {
  variants?: PlaybackVariant[]
  selectedVariantId: string
  onSelectVariant: (variantId: string) => void
}) {
  const isMobile = useIsMobile()
  const [open, setOpen] = useState(false)
  const availableVariants = (variants ?? []).filter(
    (variant) => variant.available
  )
  const selectedVariant =
    availableVariants.find((variant) => variant.id === selectedVariantId) ??
    availableVariants[0]

  if (availableVariants.length <= 1) {
    return null
  }

  const selectVariant = (variant: PlaybackVariant) => {
    onSelectVariant(variant.id)
    setOpen(false)
  }

  if (isMobile) {
    return (
      <>
        <button
          type='button'
          onClick={() => setOpen(true)}
          className='min-w-10 text-center transition-opacity hover:opacity-80'
        >
          {selectedVariant?.label ?? '原画'}
        </button>
        <Drawer open={open} onOpenChange={setOpen}>
          <DrawerContent className='border-border bg-background text-foreground'>
            <DrawerHeader className='px-5 pt-4 pb-2 text-left'>
              <DrawerTitle>清晰度</DrawerTitle>
            </DrawerHeader>
            <div className='grid gap-2 px-5 pb-6'>
              {availableVariants.map((variant) => {
                const isActive = variant.id === selectedVariantId
                return (
                  <button
                    key={variant.id}
                    type='button'
                    onClick={() => selectVariant(variant)}
                    className={`flex h-11 items-center justify-between rounded-xl border px-4 text-left text-sm font-semibold transition-colors ${isActive ? 'border-border bg-accent text-accent-foreground' : 'border-border/60 bg-muted/30 text-muted-foreground'}`}
                  >
                    <span>{variant.label}</span>
                    <span className='text-xs'>
                      {qualityVariantDescription(variant)}
                    </span>
                  </button>
                )
              })}
            </div>
          </DrawerContent>
        </Drawer>
      </>
    )
  }

  return (
    <div
      className='group/quality relative min-w-10 text-center md:min-w-12'
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type='button'
        className='w-full text-center transition-opacity hover:opacity-80'
      >
        {selectedVariant?.label ?? '原画'}
      </button>
      {open ? (
        <>
          <div className='absolute bottom-full left-1/2 z-40 h-4 w-56 -translate-x-1/2' />
          <div className='absolute bottom-full left-1/2 z-50 mb-4 w-56 -translate-x-1/2 rounded-lg border border-border/60 bg-popover p-2 text-popover-foreground shadow-2xl backdrop-blur-xl'>
            <div className='grid gap-1'>
              {availableVariants.map((variant) => {
                const isActive = variant.id === selectedVariantId
                return (
                  <button
                    key={variant.id}
                    type='button'
                    onClick={() => selectVariant(variant)}
                    className={`flex h-10 items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold transition-colors hover:bg-accent ${isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}`}
                  >
                    <span>{variant.label}</span>
                    <span className='text-xs text-muted-foreground'>
                      {qualityVariantDescription(variant)}
                    </span>
                  </button>
                )
              })}
            </div>
          </div>
        </>
      ) : null}
    </div>
  )
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
  const isMobile = useIsMobile()
  const [open, setOpen] = useState(false)
  const [customOpen, setCustomOpen] = useState(false)
  const selectPlaybackRate = (nextRate: number) => {
    setPlayerPlaybackRate(playerRef, nextRate)
    onPlaybackRateChange(nextRate)
  }

  if (isMobile) {
    return (
      <>
        <button
          type='button'
          onClick={() => setOpen(true)}
          className='w-full text-center transition-opacity hover:opacity-80'
        >
          {formatPlaybackRate(playbackRate)}
        </button>
        <Drawer
          open={open}
          onOpenChange={(nextOpen) => {
            setOpen(nextOpen)
            if (!nextOpen) {
              setCustomOpen(false)
            }
          }}
        >
          <DrawerContent className='border-border bg-background text-foreground'>
            <DrawerHeader className='px-5 pt-4 pb-2 text-left'>
              <DrawerTitle>播放倍速</DrawerTitle>
            </DrawerHeader>
            <div className='grid gap-2 px-5 pb-6'>
              {PLAYBACK_RATE_OPTIONS.map((rate) => {
                const isActive = Math.abs(playbackRate - rate) < 0.01
                return (
                  <button
                    key={rate}
                    type='button'
                    onClick={() => selectPlaybackRate(rate)}
                    className={`flex h-11 items-center justify-between rounded-xl border px-4 text-left text-sm font-semibold transition-colors ${isActive ? 'border-border bg-accent text-accent-foreground' : 'border-border/60 bg-muted/30 text-muted-foreground'}`}
                  >
                    <span>{formatPlaybackRate(rate)}</span>
                    {isActive ? (
                      <span className='size-1.5 rounded-full bg-foreground' />
                    ) : null}
                  </button>
                )
              })}
              <div className='rounded-xl border border-border/60 bg-muted/30 px-4 py-4'>
                <div className='mb-3 flex items-center justify-between text-sm font-semibold'>
                  <span className='text-muted-foreground'>自定义倍速</span>
                  <span>{playbackRate.toFixed(1)}x</span>
                </div>
                <Slider
                  min={0.1}
                  max={5}
                  step={0.1}
                  value={[playbackRate]}
                  onValueChange={([nextRate]) => {
                    if (typeof nextRate === 'number') {
                      onPlaybackRateChange(Number(nextRate.toFixed(1)))
                      setPlayerPlaybackRate(
                        playerRef,
                        Number(nextRate.toFixed(1))
                      )
                    }
                  }}
                  className='[&_[data-slot=slider-range]]:bg-foreground [&_[data-slot=slider-thumb]]:size-4 [&_[data-slot=slider-thumb]]:border-foreground [&_[data-slot=slider-thumb]]:bg-foreground [&_[data-slot=slider-track]]:bg-muted'
                />
              </div>
            </div>
          </DrawerContent>
        </Drawer>
      </>
    )
  }
  return (
    <div
      className='group/rate relative w-10 text-center md:w-12'
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => {
        setOpen(false)
        setCustomOpen(false)
      }}
    >
      <button
        type='button'
        className='w-full text-center transition-opacity hover:opacity-80'
      >
        {formatPlaybackRate(playbackRate)}
      </button>
      {open ? (
        <>
          <div className='absolute bottom-full left-1/2 z-40 h-4 w-52 -translate-x-1/2' />
          <div
            className={`absolute bottom-full left-1/2 z-50 mb-4 w-52 -translate-x-1/2 rounded-lg border border-border/60 bg-popover p-2 text-popover-foreground shadow-2xl backdrop-blur-xl ${customOpen ? 'min-h-24' : 'min-h-68'}`}
          >
            {customOpen ? (
              <CustomPlaybackRatePanel
                playbackRate={playbackRate}
                onPlaybackRateChange={selectPlaybackRate}
              />
            ) : (
              <div className='grid gap-1'>
                {PLAYBACK_RATE_OPTIONS.map((rate) => {
                  const isActive = Math.abs(playbackRate - rate) < 0.01
                  return (
                    <button
                      key={rate}
                      type='button'
                      onClick={() => selectPlaybackRate(rate)}
                      className={`flex h-9 items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold transition-colors hover:bg-accent ${isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}`}
                    >
                      <span>{formatPlaybackRate(rate)}</span>
                      {isActive ? (
                        <span className='size-1.5 rounded-full bg-foreground' />
                      ) : null}
                    </button>
                  )
                })}
                <button
                  type='button'
                  onPointerDown={(event) => {
                    event.preventDefault()
                    event.stopPropagation()
                    setCustomOpen(true)
                  }}
                  className='flex h-9 items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
                >
                  <span>自定义</span>
                  <span className='text-xs text-muted-foreground'>0.1-5.0</span>
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
    <div className='px-2 pt-2 pb-3'>
      <div className='mb-4 flex items-center justify-between text-sm font-semibold'>
        <span className='text-muted-foreground'>自定义倍速</span>
        <span>{playbackRate.toFixed(1)}x</span>
      </div>
      <div className='px-1'>
        <Slider
          min={0.1}
          max={5}
          step={0.1}
          value={[playbackRate]}
          onValueChange={([nextRate]) => {
            if (typeof nextRate === 'number') {
              onPlaybackRateChange(Number(nextRate.toFixed(1)))
            }
          }}
          className='[&_[data-slot=slider-range]]:bg-foreground [&_[data-slot=slider-thumb]]:size-4 [&_[data-slot=slider-thumb]]:border-foreground [&_[data-slot=slider-thumb]]:bg-foreground [&_[data-slot=slider-track]]:bg-muted'
        />
      </div>
    </div>
  )
}

function SubtitleHoverMenu({
  playerRef,
  itemId,
  resourceId,
  inventoryFileId,
  subtitleTracks,
  subtitleSearchProviders,
  defaultSubtitleMode = 'auto',
  preferredSubtitleLanguage,
}: {
  playerRef: ArtPlayerRef
  itemId?: number
  resourceId?: number
  inventoryFileId?: number
  subtitleTracks?: Track[]
  subtitleSearchProviders?: SubtitleSearchProvider[]
  defaultSubtitleMode?: 'auto' | 'always' | 'never'
  preferredSubtitleLanguage?: string
}) {
  const isMobile = useIsMobile()
  const token = useAuthStore((state) => state.auth.accessToken)
  const [open, setOpen] = useState(false)
  const [externalSubtitleName, setExternalSubtitleName] = useState('')
  const [selectedSubtitleKey, setSelectedSubtitleKey] = useState<string | null>(
    null
  )
  const [subtitlesVisible, setSubtitlesVisible] = useState(true)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [subtitleColorIndex, setSubtitleColorIndex] =
    useState<SubtitleColorIndex>(0)
  const [subtitlePositionIndex, setSubtitlePositionIndex] =
    useState<SubtitlePositionIndex>(0)
  const [subtitleSizeIndex, setSubtitleSizeIndex] =
    useState<SubtitleSizeIndex>(1)
  const [subtitleBackgroundOpacity, setSubtitleBackgroundOpacity] = useState(50)
  const [subtitleOffsetSeconds, setSubtitleOffsetSeconds] = useState(0)
  const [activeSearchProviderId, setActiveSearchProviderId] = useState<
    number | null
  >(null)
  const [availableTracks, setAvailableTracks] = useState<Track[]>(
    subtitleTracks ?? []
  )
  const externalSubtitleUrlRef = useRef<string | null>(null)
  const externalSubtitleInputRef = useRef<HTMLInputElement | null>(null)
  const tracks = availableTracks
  const searchProviders = subtitleSearchProviders ?? []
  const embeddedSubtitleSummary = tracks.length ? `${tracks.length} 条` : '无'
  const externalSubtitleSummary = externalSubtitleName || '选择本地文件'
  const selectableTrackCount = tracks.filter((track) =>
    isSubtitleTrackSelectable(track)
  ).length
  const searchProviderMutation = useMutation({
    mutationFn: async (provider: SubtitleSearchProvider) => {
      if (!token) {
        throw new Error('当前未登录，无法搜索字幕')
      }
      setActiveSearchProviderId(provider.id)
      if (typeof inventoryFileId === 'number' && inventoryFileId > 0) {
        return createAuthedMiboApi(token).searchInventoryFilePlaybackSubtitles(
          inventoryFileId,
          provider.id
        )
      }
      if (typeof itemId === 'number' && itemId > 0) {
        return createAuthedMiboApi(token).searchCatalogPlaybackSubtitles(
          itemId,
          {
            providerId: provider.id,
            resourceId,
          }
        )
      }
      throw new Error('当前播放上下文不支持字幕搜索')
    },
    onSuccess: (result) => {
      setAvailableTracks((currentTracks) =>
        mergeSubtitleTracks(currentTracks, result.tracks)
      )
      toast.success(
        result.tracks.length
          ? `${result.provider.name} 已添加 ${result.tracks.length} 条字幕`
          : `${result.provider.name} 没有找到可用字幕`
      )
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '字幕搜索失败')
    },
    onSettled: () => {
      setActiveSearchProviderId(null)
    },
  })
  const toggleSubtitlesVisible = () => {
    const player = playerRef.current
    const nextVisible = !subtitlesVisible
    setSubtitlesVisible(nextVisible)
    if (player) {
      player.subtitle.show = nextVisible
    }
  }
  useEffect(
    () => () => {
      if (externalSubtitleUrlRef.current) {
        URL.revokeObjectURL(externalSubtitleUrlRef.current)
      }
    },
    []
  )
  useEffect(() => {
    setAvailableTracks(subtitleTracks ?? [])
    setSelectedSubtitleKey(null)
    setExternalSubtitleName('')
    setSubtitlesVisible(defaultSubtitleMode !== 'never')
  }, [defaultSubtitleMode, subtitleTracks])

  const selectServerSubtitle = useCallback(
    async (track: Track, index: number, options?: { silent?: boolean }) => {
      const player = playerRef.current
      const subtitleUrl = track.url?.trim()
      if (!player || !subtitleUrl) {
        toast.info('这个字幕轨道当前还不能直接切换')
        return
      }

      try {
        await player.subtitle.switch(subtitleUrl, {
          name: formatSubtitleTrackLabel(track, index),
          type: getSubtitleTrackType(track),
        })
        player.subtitle.show = true
        setSubtitlesVisible(true)
        setSelectedSubtitleKey(getSubtitleTrackKey(track, index))
        setExternalSubtitleName('')
        if (!options?.silent) {
          toast.success('字幕已切换')
        }
      } catch (error) {
        if (!options?.silent) {
          toast.error(error instanceof Error ? error.message : '字幕切换失败')
        }
      }
    },
    [playerRef]
  )

  const loadExternalSubtitle = async (file: File) => {
    const player = playerRef.current
    if (!player) {
      toast.error('播放器尚未准备好')
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
      setSelectedSubtitleKey('local-upload')
      player.subtitle.show = true
      setSubtitlesVisible(true)
      toast.success('外挂字幕已加载')
    } catch (error) {
      URL.revokeObjectURL(subtitleUrl)
      toast.error(error instanceof Error ? error.message : '外挂字幕加载失败')
    }
  }

  useEffect(() => {
    if (defaultSubtitleMode === 'never') {
      const player = playerRef.current
      if (player) {
        player.subtitle.show = false
      }
      setSubtitlesVisible(false)
      return
    }
    if (selectedSubtitleKey) {
      return
    }
    const selectableTracks = tracks
      .map((track, index) => ({ track, index }))
      .filter(({ track }) => isSubtitleTrackSelectable(track))
    if (selectableTracks.length === 0) {
      return
    }

    const preferredLanguage = normalizeSubtitleLanguage(
      preferredSubtitleLanguage
    )
    const preferredTrack = preferredLanguage
      ? selectableTracks.find(({ track }) =>
          subtitleTrackMatchesLanguage(track, preferredLanguage)
        )
      : undefined
    const fallbackTrack = selectableTracks[0]
    const trackToSelect =
      defaultSubtitleMode === 'always'
        ? (preferredTrack ?? fallbackTrack)
        : preferredTrack

    if (!trackToSelect) {
      return
    }

    void selectServerSubtitle(trackToSelect.track, trackToSelect.index, {
      silent: true,
    })
  }, [
    defaultSubtitleMode,
    preferredSubtitleLanguage,
    playerRef,
    selectedSubtitleKey,
    selectServerSubtitle,
    tracks,
  ])

  const searchWithProvider = (provider: SubtitleSearchProvider) => {
    searchProviderMutation.mutate(provider)
  }

  if (isMobile) {
    return (
      <>
        <button
          type='button'
          onClick={() => setOpen(true)}
          className='w-full text-center whitespace-nowrap transition-opacity hover:opacity-80'
        >
          字幕
        </button>
        <input
          ref={externalSubtitleInputRef}
          type='file'
          accept='.srt,.vtt,.ass,text/vtt,application/x-subrip'
          className='hidden'
          onChange={(event) => {
            const file = event.target.files?.[0]
            event.currentTarget.value = ''
            if (file) {
              void loadExternalSubtitle(file)
            }
          }}
        />
        <Drawer
          open={open}
          onOpenChange={(nextOpen) => {
            setOpen(nextOpen)
            if (!nextOpen) {
              setSettingsOpen(false)
            }
          }}
        >
          <DrawerContent className='border-border bg-background text-foreground'>
            <DrawerHeader className='px-5 pt-4 pb-2 text-left'>
              <DrawerTitle>{settingsOpen ? '字幕设置' : '字幕'}</DrawerTitle>
            </DrawerHeader>
            <div className='px-5 pb-6'>
              {settingsOpen ? (
                <div className='overflow-x-auto'>
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
                    onSubtitleBackgroundOpacityChange={
                      setSubtitleBackgroundOpacity
                    }
                    onSubtitleOffsetSecondsChange={setSubtitleOffsetSeconds}
                  />
                </div>
              ) : (
                <div className='grid gap-2'>
                  <button
                    type='button'
                    onClick={toggleSubtitlesVisible}
                    className='w-full rounded-xl border border-border/60 bg-muted/30 px-4 py-3 text-left'
                  >
                    <div className='flex items-center justify-between text-[15px] font-semibold'>
                      <span>显示字幕</span>
                      <span
                        className={`flex h-6 w-11 items-center rounded-full p-1 transition-colors ${subtitlesVisible ? 'justify-end bg-accent' : 'justify-start bg-muted'}`}
                      >
                        <span className='size-4 rounded-full bg-foreground shadow transition-transform' />
                      </span>
                    </div>
                    <div className='mt-1 text-xs leading-5 text-muted-foreground'>
                      当前共有 {embeddedSubtitleSummary} 条字幕，可直接切换{' '}
                      {selectableTrackCount} 条
                    </div>
                  </button>
                  {tracks.length ? (
                    <div className='rounded-xl border border-border/60 bg-muted/30 p-2'>
                      <div className='mb-1 px-2 text-[11px] font-semibold tracking-[0.14em] text-muted-foreground uppercase'>
                        可用字幕
                      </div>
                      <div className='grid gap-1'>
                        {tracks.map((track, index) => (
                          <button
                            key={`${track.language}-${track.title}-${index}`}
                            type='button'
                            disabled={!isSubtitleTrackSelectable(track)}
                            onClick={() =>
                              void selectServerSubtitle(track, index)
                            }
                            className={`rounded-lg px-3 py-2 text-left transition-colors ${isSubtitleTrackSelectable(track) ? 'hover:bg-accent' : 'cursor-not-allowed opacity-60'} ${selectedSubtitleKey === getSubtitleTrackKey(track, index) ? 'bg-accent/80' : ''}`}
                          >
                            <div className='flex items-center justify-between gap-3 text-[15px] font-semibold text-foreground/80'>
                              <span className='truncate'>
                                {formatSubtitleTrackLabel(track, index)}
                              </span>
                              <span className='shrink-0 text-xs text-muted-foreground'>
                                {track.codec || '字幕'}
                              </span>
                            </div>
                            <div className='mt-1 flex items-center gap-2 text-xs text-muted-foreground'>
                              <span>
                                {formatLanguageCode(track.language) ||
                                  '未知语言'}
                              </span>
                              <span>{track.external ? '外挂' : '内封'}</span>
                              {track.provider_name ? (
                                <span>{track.provider_name}</span>
                              ) : null}
                              <span>
                                {isSubtitleTrackSelectable(track)
                                  ? '可切换'
                                  : '网页直放暂不可切换'}
                              </span>
                            </div>
                          </button>
                        ))}
                      </div>
                    </div>
                  ) : null}
                  {searchProviders.length ? (
                    <div className='rounded-xl border border-border/60 bg-muted/30 p-2'>
                      <div className='mb-1 px-2 text-[11px] font-semibold tracking-[0.14em] text-muted-foreground uppercase'>
                        字幕提供者
                      </div>
                      <div className='grid gap-1'>
                        {searchProviders.map((provider) => {
                          const searching =
                            searchProviderMutation.isPending &&
                            activeSearchProviderId === provider.id
                          return (
                            <button
                              key={provider.id}
                              type='button'
                              onClick={() => searchWithProvider(provider)}
                              disabled={searchProviderMutation.isPending}
                              className='flex min-h-11 items-center justify-between rounded-lg px-3 py-2 text-left transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60'
                            >
                              <div>
                                <div className='text-[15px] font-semibold text-foreground/80'>
                                  {provider.name}
                                </div>
                                <div className='text-xs text-muted-foreground'>
                                  点击搜索字幕
                                </div>
                              </div>
                              <div className='text-xs text-muted-foreground'>
                                {searching ? '搜索中...' : '搜索'}
                              </div>
                            </button>
                          )
                        })}
                      </div>
                    </div>
                  ) : null}
                  <button
                    type='button'
                    onClick={() => externalSubtitleInputRef.current?.click()}
                    className='flex h-11 w-full items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-[15px] font-semibold text-muted-foreground'
                  >
                    <span>外挂字幕</span>
                    <span className='max-w-32 truncate text-xs text-muted-foreground'>
                      {externalSubtitleSummary}
                    </span>
                  </button>
                  <button
                    type='button'
                    onClick={() => setSettingsOpen(true)}
                    className='flex h-11 w-full items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-[15px] font-semibold text-muted-foreground'
                  >
                    <span>字幕设置</span>
                    <ChevronRightIcon className='size-4 stroke-[2.8] text-muted-foreground' />
                  </button>
                </div>
              )}
            </div>
          </DrawerContent>
        </Drawer>
      </>
    )
  }
  return (
    <div
      className='group/subtitle relative w-10 text-center md:w-12'
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => {
        setOpen(false)
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
        type='button'
        className='w-full text-center whitespace-nowrap transition-opacity hover:opacity-80'
      >
        字幕
      </button>
      <input
        ref={externalSubtitleInputRef}
        type='file'
        accept='.srt,.vtt,.ass,text/vtt,application/x-subrip'
        className='hidden'
        onChange={(event) => {
          const file = event.target.files?.[0]
          event.currentTarget.value = ''
          if (file) {
            void loadExternalSubtitle(file)
          }
        }}
      />
      {open ? (
        <>
          <div
            className={`absolute bottom-full left-1/2 z-40 h-4 -translate-x-1/2 ${settingsOpen ? 'w-[40rem] max-w-[calc(100vw-2rem)]' : 'w-64'}`}
          />
          <div
            className={`absolute bottom-full left-1/2 z-50 mb-4 -translate-x-1/2 overflow-hidden rounded-lg border border-border/60 bg-popover text-popover-foreground shadow-2xl backdrop-blur-xl ${settingsOpen ? 'w-max max-w-[calc(100vw-2rem)]' : 'w-64 p-2'}`}
          >
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
              <div className='grid gap-1'>
                <button
                  type='button'
                  onClick={toggleSubtitlesVisible}
                  className='w-full rounded-lg px-3 py-2 text-left transition-colors hover:bg-accent'
                >
                  <div className='flex items-center justify-between text-[15px] font-semibold'>
                    <span>显示字幕</span>
                    <span
                      className={`flex h-6 w-11 items-center rounded-full p-1 transition-colors ${subtitlesVisible ? 'justify-end bg-accent' : 'justify-start bg-muted'}`}
                    >
                      <span className='size-4 rounded-full bg-foreground shadow transition-transform' />
                    </span>
                  </div>
                  <div className='mt-1 text-xs leading-5 text-muted-foreground'>
                    当前共有 {embeddedSubtitleSummary} 条字幕，可直接切换{' '}
                    {selectableTrackCount} 条
                  </div>
                </button>
                <div className='my-1 h-px bg-border' />
                <div className='rounded-lg px-3 py-2'>
                  <div className='flex items-center justify-between text-left text-[15px] font-semibold text-muted-foreground'>
                    <span>可用字幕</span>
                    <span className='text-xs text-muted-foreground'>
                      {embeddedSubtitleSummary}
                    </span>
                  </div>
                  <div className='mt-2 grid max-h-64 gap-1 overflow-y-auto pr-1'>
                    {tracks.length ? (
                      tracks.map((track, index) => (
                        <button
                          key={`${track.language}-${track.title}-${index}`}
                          type='button'
                          disabled={!isSubtitleTrackSelectable(track)}
                          onClick={() =>
                            void selectServerSubtitle(track, index)
                          }
                          className={`rounded-lg px-3 py-2 text-left transition-colors ${isSubtitleTrackSelectable(track) ? 'hover:bg-accent' : 'cursor-not-allowed opacity-60'} ${selectedSubtitleKey === getSubtitleTrackKey(track, index) ? 'bg-accent/80' : ''}`}
                        >
                          <div className='flex items-center justify-between gap-3 text-[15px] font-semibold text-foreground/80'>
                            <span className='truncate'>
                              {formatSubtitleTrackLabel(track, index)}
                            </span>
                            <span className='shrink-0 text-xs text-muted-foreground'>
                              {track.codec || '字幕'}
                            </span>
                          </div>
                          <div className='mt-1 flex items-center gap-2 text-xs text-muted-foreground'>
                            <span>
                              {formatLanguageCode(track.language) || '未知语言'}
                            </span>
                            <span>{track.external ? '外挂' : '内封'}</span>
                            {track.provider_name ? (
                              <span>{track.provider_name}</span>
                            ) : null}
                            <span>
                              {isSubtitleTrackSelectable(track)
                                ? '可切换'
                                : '网页直放暂不可切换'}
                            </span>
                          </div>
                        </button>
                      ))
                    ) : (
                      <div className='rounded-lg px-3 py-2 text-left text-xs text-muted-foreground'>
                        暂无可用字幕
                      </div>
                    )}
                  </div>
                </div>
                {searchProviders.length ? (
                  <div className='rounded-lg px-3 py-2'>
                    <div className='mb-2 text-left text-[15px] font-semibold text-muted-foreground'>
                      字幕提供者
                    </div>
                    <div className='grid gap-1'>
                      {searchProviders.map((provider) => {
                        const searching =
                          searchProviderMutation.isPending &&
                          activeSearchProviderId === provider.id
                        return (
                          <button
                            key={provider.id}
                            type='button'
                            onClick={() => searchWithProvider(provider)}
                            disabled={searchProviderMutation.isPending}
                            className='flex min-h-9 items-center justify-between rounded-lg px-3 py-2 text-left transition-colors hover:bg-accent hover:text-accent-foreground disabled:cursor-not-allowed disabled:opacity-60'
                          >
                            <span>{provider.name}</span>
                            <span className='text-xs text-muted-foreground'>
                              {searching ? '搜索中...' : '搜索'}
                            </span>
                          </button>
                        )
                      })}
                    </div>
                  </div>
                ) : null}
                <button
                  type='button'
                  onClick={() => externalSubtitleInputRef.current?.click()}
                  className='flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
                >
                  <span>外挂字幕</span>
                  <span className='max-w-32 truncate text-xs text-muted-foreground'>
                    {externalSubtitleSummary}
                  </span>
                </button>
                <button
                  type='button'
                  onClick={() => {
                    setSettingsOpen(true)
                  }}
                  className='flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
                >
                  <span>字幕设置</span>
                  <ChevronRightIcon className='size-4 stroke-[2.8] text-muted-foreground' />
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
    <div className='text-foreground'>
      <div className='flex h-14 items-center justify-between border-b border-border px-4'>
        <button
          type='button'
          onClick={onBack}
          className='flex items-center gap-1.5 text-[15px] font-semibold transition-opacity hover:opacity-80'
        >
          <ChevronLeftIcon className='size-4 stroke-[2.8]' />
          字幕设置
        </button>
        <button
          type='button'
          onClick={resetSettings}
          className='text-xs font-semibold text-muted-foreground transition-colors hover:text-foreground'
        >
          恢复默认设置
        </button>
      </div>
      <div className='grid gap-4 px-4 py-4'>
        <SubtitleSettingRow label='字幕颜色'>
          <div className='relative w-72 shrink-0'>
            <select
              value={subtitleColorIndex}
              onChange={(event) =>
                onSubtitleColorIndexChange(
                  Number(event.target.value) as SubtitleColorIndex
                )
              }
              className='h-9 w-full appearance-none rounded-lg border border-border/60 bg-muted/30 px-11 pr-10 text-[15px] font-semibold text-foreground transition-colors outline-none hover:bg-accent'
            >
              {SUBTITLE_COLOR_OPTIONS.map((option, index) => (
                <option key={option.value} value={index}>
                  {option.label}
                </option>
              ))}
            </select>
            <span
              className='absolute top-1/2 left-3 size-5 -translate-y-1/2 rounded-sm border border-border shadow'
              style={{
                backgroundColor:
                  SUBTITLE_COLOR_OPTIONS[subtitleColorIndex].value,
              }}
            />
            <ChevronRightIcon className='pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 rotate-90 stroke-[2.8] text-muted-foreground' />
          </div>
        </SubtitleSettingRow>
        <SubtitleSettingRow label='字幕位置'>
          <div className='relative w-72 shrink-0'>
            <select
              value={subtitlePositionIndex}
              onChange={(event) =>
                onSubtitlePositionIndexChange(
                  Number(event.target.value) as SubtitlePositionIndex
                )
              }
              className='h-9 w-full appearance-none rounded-lg border border-border/60 bg-muted/30 px-11 pr-10 text-[15px] font-semibold text-foreground transition-colors outline-none hover:bg-accent'
            >
              {SUBTITLE_POSITION_OPTIONS.map((option, index) => (
                <option key={option.label} value={index}>
                  {option.label}
                </option>
              ))}
            </select>
            <span className='absolute top-1/2 left-3 flex size-5 -translate-y-1/2 items-end justify-center rounded-sm border border-border pb-0.5'>
              <span className='h-0.5 w-2.5 rounded-full bg-foreground' />
            </span>
            <ChevronRightIcon className='pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 rotate-90 stroke-[2.8] text-muted-foreground' />
          </div>
        </SubtitleSettingRow>
        <SubtitleSettingRow label='字幕大小'>
          <div className='grid h-9 w-72 shrink-0 grid-cols-4 rounded-lg bg-muted/30 p-0.5'>
            {SUBTITLE_SIZE_OPTIONS.map((option, index) => {
              const isActive = index === subtitleSizeIndex
              return (
                <button
                  key={option.label}
                  type='button'
                  onClick={() =>
                    onSubtitleSizeIndexChange(index as SubtitleSizeIndex)
                  }
                  className={`rounded-md text-[15px] font-semibold transition-colors ${isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:text-foreground'}`}
                >
                  {option.label}
                </button>
              )
            })}
          </div>
        </SubtitleSettingRow>
        <SubtitleSettingRow label='背景透明度'>
          <div className='flex w-72 shrink-0 items-center gap-3'>
            <Slider
              min={0}
              max={100}
              step={1}
              value={[subtitleBackgroundOpacity]}
              onValueChange={([nextOpacity]) => {
                if (typeof nextOpacity === 'number') {
                  onSubtitleBackgroundOpacityChange(nextOpacity)
                }
              }}
              className='[&_[data-slot=slider-range]]:bg-foreground [&_[data-slot=slider-thumb]]:size-5 [&_[data-slot=slider-thumb]]:border-foreground [&_[data-slot=slider-thumb]]:bg-foreground [&_[data-slot=slider-track]]:bg-muted'
            />
            <span className='w-10 text-right text-[15px] font-semibold text-muted-foreground'>
              {subtitleBackgroundOpacity}%
            </span>
          </div>
        </SubtitleSettingRow>
        <SubtitleSettingRow label='偏移时间' hint='仅对当前视频生效'>
          <div className='flex w-72 shrink-0 items-center justify-between'>
            <button
              type='button'
              onClick={() =>
                onSubtitleOffsetSecondsChange(
                  Number((subtitleOffsetSeconds - 0.25).toFixed(2))
                )
              }
              className='flex size-9 items-center justify-center rounded-full bg-muted/50 text-xl font-semibold text-foreground/80 transition-colors hover:bg-accent'
            >
              -
            </button>
            <span className='text-[15px] font-semibold text-muted-foreground tabular-nums'>
              {subtitleOffsetSeconds >= 0 ? '+' : ''}
              {subtitleOffsetSeconds.toFixed(2)}s
            </span>
            <button
              type='button'
              onClick={() =>
                onSubtitleOffsetSecondsChange(
                  Number((subtitleOffsetSeconds + 0.25).toFixed(2))
                )
              }
              className='flex size-9 items-center justify-center rounded-full bg-muted/50 text-xl font-semibold text-foreground/80 transition-colors hover:bg-accent'
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
  children: ReactNode
}) {
  return (
    <div className='grid grid-cols-[7rem_1fr] items-center gap-4'>
      <div className='text-left'>
        <div className='text-[15px] font-semibold'>{label}</div>
        {hint ? (
          <div className='mt-0.5 text-xs font-semibold text-muted-foreground'>
            {hint}
          </div>
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
    lineHeight: '1.25',
    padding: '0.2em 0.45em',
    borderRadius: '0.28em',
    backgroundColor: `rgba(0, 0, 0, ${backgroundOpacity})`,
    textShadow: '0 2px 4px rgba(0, 0, 0, 0.7)',
  })
  player.subtitleOffset = settings.offsetSeconds
}
function formatSubtitleTrackLabel(track: Track, index: number) {
  return formatSubtitleTrackMenuLabel(track.title, track.language, index)
}

function getSubtitleTrackKey(track: Track, index: number) {
  if (typeof track.stream_index === 'number' && track.stream_index >= 0) {
    return `stream-${track.stream_index}`
  }
  if (typeof track.file_id === 'number' && track.file_id > 0) {
    return `file-${track.file_id}`
  }
  if (track.url?.trim()) {
    return `url-${track.url.trim()}`
  }
  return `${track.language}-${track.title}-${index}`
}

function mergeSubtitleTracks(currentTracks: Track[], nextTracks: Track[]) {
  if (!nextTracks.length) {
    return currentTracks
  }
  const merged = [...currentTracks]
  const seen = new Set(
    merged.map((track, index) => getSubtitleTrackKey(track, index))
  )
  for (const track of nextTracks) {
    const key = getSubtitleTrackKey(track, merged.length)
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    merged.push(track)
  }
  return merged
}

function isSubtitleTrackSelectable(track: Track) {
  return Boolean(track.url?.trim()) && track.available !== false
}

function getSubtitleTrackType(track: Track) {
  if (track.codec) {
    return getSubtitleFileType(`subtitle.${track.codec}`)
  }
  if (track.url) {
    try {
      const path = new URL(track.url, window.location.origin).pathname
      return getSubtitleFileType(path)
    } catch {
      return getSubtitleFileType(track.url)
    }
  }
  return 'vtt'
}

function normalizeSubtitleLanguage(language?: string) {
  return normalizeLanguageCode(language)
}

function subtitleTrackMatchesLanguage(track: Track, preferredLanguage: string) {
  const language = normalizeSubtitleLanguage(track.language)
  if (!language) return false
  if (language === preferredLanguage) return true
  if (language.startsWith(`${preferredLanguage}-`)) return true
  if (preferredLanguage.startsWith(`${language}-`)) return true
  return false
}

function getSubtitleFileType(fileName: string) {
  const extension = fileName.split('.').pop()?.toLowerCase()
  if (extension === 'srt' || extension === 'ass' || extension === 'vtt')
    return extension
  return 'vtt'
}

function SettingsHoverMenu({
  restorePositionEnabled,
  skipIntroSeconds,
  skipOutroSeconds,
  playbackMode,
  canProbePlaybackResource,
  isProbingPlaybackResource,
  onOpenExternalPlayer,
  onProbePlaybackResource,
  onSkipSettingsOpenChange,
  onRestorePositionEnabledChange,
  onPlaybackModeChange,
}: {
  restorePositionEnabled: boolean
  skipIntroSeconds: number
  skipOutroSeconds: number
  playbackMode: PlaybackMode
  canProbePlaybackResource: boolean
  isProbingPlaybackResource: boolean
  onOpenExternalPlayer: () => void
  onProbePlaybackResource: () => void
  onSkipSettingsOpenChange: (open: boolean) => void
  onRestorePositionEnabledChange: (enabled: boolean) => void
  onPlaybackModeChange: (playbackMode: PlaybackMode) => void
}) {
  const isMobile = useIsMobile()
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
      : '未设置'

  if (isMobile) {
    return (
      <>
        <button
          type='button'
          aria-label='设置'
          onClick={() => setOpen(true)}
          className='transition-opacity hover:opacity-80'
        >
          <SettingsIcon className='size-5 stroke-[2.4] md:size-7' />
        </button>
        <Drawer
          open={open}
          onOpenChange={(nextOpen) => {
            setOpen(nextOpen)
            if (!nextOpen) {
              setModeMenuOpen(false)
            }
          }}
        >
          <DrawerContent className='border-border bg-background text-foreground'>
            <DrawerHeader className='px-5 pt-4 pb-2 text-left'>
              <DrawerTitle>播放设置</DrawerTitle>
            </DrawerHeader>
            <div className='grid gap-2 px-5 pb-6'>
              <button
                type='button'
                onClick={() =>
                  onRestorePositionEnabledChange(!restorePositionEnabled)
                }
                className='flex h-11 w-full items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-[15px] font-semibold text-muted-foreground'
              >
                <span>自动定位上次观看位置</span>
                <span
                  className={`flex h-6 w-11 items-center rounded-full p-1 transition-colors ${restorePositionEnabled ? 'justify-end bg-accent' : 'justify-start bg-muted'}`}
                >
                  <span className='size-4 rounded-full bg-foreground shadow transition-transform' />
                </span>
              </button>
              <div className='rounded-xl border border-border/60 bg-muted/30 p-3'>
                <div className='mb-2 text-[11px] font-semibold tracking-[0.14em] text-muted-foreground uppercase'>
                  播放模式
                </div>
                <div className='grid gap-1'>
                  {PLAYBACK_MODE_OPTIONS.map((mode) => {
                    const isActive = mode === playbackMode
                    return (
                      <button
                        key={mode}
                        type='button'
                        onClick={() => onPlaybackModeChange(mode)}
                        className={`flex h-10 items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold transition-colors ${isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent/70'}`}
                      >
                        <span>{mode}</span>
                        {isActive ? (
                          <span className='size-1.5 rounded-full bg-foreground' />
                        ) : null}
                      </button>
                    )
                  })}
                </div>
              </div>
              <button
                type='button'
                onClick={() => {
                  setOpen(false)
                  onOpenExternalPlayer()
                }}
                className='flex h-11 w-full items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-[15px] font-semibold text-muted-foreground'
              >
                <span>用外部播放器打开</span>
                <ExternalLinkIcon className='size-4 stroke-[2.5]' />
              </button>
              {canProbePlaybackResource ? (
                <button
                  type='button'
                  onClick={() => {
                    setOpen(false)
                    onProbePlaybackResource()
                  }}
                  disabled={isProbingPlaybackResource}
                  className='flex h-11 w-full items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-[15px] font-semibold text-muted-foreground disabled:cursor-not-allowed disabled:opacity-55'
                >
                  <span>探测资源</span>
                  <span className='text-xs'>
                    {isProbingPlaybackResource ? '探测中' : '刷新音轨字幕'}
                  </span>
                </button>
              ) : null}
              <button
                type='button'
                onClick={() => {
                  setOpen(false)
                  onSkipSettingsOpenChange(true)
                }}
                className='flex h-11 w-full items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-[15px] font-semibold text-muted-foreground'
              >
                <span>设置片头片尾</span>
                <span className='flex items-center gap-1.5 text-xs text-muted-foreground'>
                  {skipSummary}
                  <ChevronRightIcon className='size-4 stroke-[2.8]' />
                </span>
              </button>
              <button
                type='button'
                onClick={() => void navigate({ to: '/settings/display' })}
                className='flex h-11 w-full items-center justify-between rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-[15px] font-semibold text-muted-foreground'
              >
                <span>更多设置</span>
                <span className='rounded-md bg-accent px-1.5 py-0.5 text-xs font-bold tracking-normal text-accent-foreground'>
                  NEW
                </span>
              </button>
              <button
                type='button'
                onClick={() =>
                  toast.info(
                    '反馈入口准备中，请先在 GitHub Issue 或项目讨论区提交。'
                  )
                }
                className='flex h-11 w-full items-center rounded-xl border border-border/60 bg-muted/30 px-4 text-left text-[15px] font-semibold text-muted-foreground'
              >
                意见反馈
              </button>
            </div>
          </DrawerContent>
        </Drawer>
      </>
    )
  }
  return (
    <div
      className='relative -m-2 flex items-center p-2'
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
        type='button'
        aria-label='设置'
        className='transition-opacity hover:opacity-80'
      >
        <SettingsIcon className='size-5 stroke-[2.4] md:size-7' />
      </button>
      {open ? (
        <>
          <div className='absolute right-0 bottom-full z-40 h-4 w-80' />
          <div className='absolute right-0 bottom-full z-50 mb-4 w-80 rounded-lg border border-border/60 bg-popover p-2 text-popover-foreground shadow-2xl backdrop-blur-xl'>
            <button
              type='button'
              onClick={() =>
                onRestorePositionEnabledChange(!restorePositionEnabled)
              }
              className='flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
            >
              <span>自动定位上次观看位置</span>
              <span
                className={`flex h-6 w-11 items-center rounded-full p-1 transition-colors ${restorePositionEnabled ? 'justify-end bg-accent' : 'justify-start bg-muted'}`}
              >
                <span className='size-4 rounded-full bg-foreground shadow transition-transform' />
              </span>
            </button>
            <div
              className='relative mt-1'
              onMouseEnter={() => setModeMenuOpen(true)}
              onMouseLeave={() => setModeMenuOpen(false)}
              onFocus={() => setModeMenuOpen(true)}
            >
              <button
                type='button'
                className='flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
              >
                <span>播放模式</span>
                <span className='flex items-center gap-1.5 text-xs text-muted-foreground'>
                  {playbackMode}
                  <ChevronRightIcon className='size-4 stroke-[2.8]' />
                </span>
              </button>
              {modeMenuOpen ? (
                <>
                  <div className='absolute top-[-4.75rem] right-full z-50 h-56 w-4' />
                  <div className='absolute top-[-4.75rem] right-full z-50 mr-4 w-52 rounded-lg border border-border/60 bg-popover p-2 text-popover-foreground shadow-2xl backdrop-blur-xl'>
                    <div className='grid gap-1'>
                      {PLAYBACK_MODE_OPTIONS.map((mode) => {
                        const isActive = mode === playbackMode
                        return (
                          <button
                            key={mode}
                            type='button'
                            onClick={() => onPlaybackModeChange(mode)}
                            className={`flex h-9 items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold transition-colors hover:bg-accent ${isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}`}
                          >
                            <span>{mode}</span>
                            {isActive ? (
                              <span className='size-1.5 rounded-full bg-foreground' />
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
              type='button'
              onClick={() => {
                setOpen(false)
                setModeMenuOpen(false)
                onOpenExternalPlayer()
              }}
              className='mt-1 flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
            >
              <span>用外部播放器打开</span>
              <ExternalLinkIcon className='size-4 stroke-[2.5]' />
            </button>
            {canProbePlaybackResource ? (
              <button
                type='button'
                onClick={() => {
                  setOpen(false)
                  setModeMenuOpen(false)
                  onProbePlaybackResource()
                }}
                disabled={isProbingPlaybackResource}
                className='mt-1 flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:cursor-not-allowed disabled:opacity-55 disabled:hover:bg-transparent disabled:hover:text-muted-foreground'
              >
                <span>探测资源</span>
                <span className='text-xs'>
                  {isProbingPlaybackResource ? '探测中' : '刷新音轨字幕'}
                </span>
              </button>
            ) : null}
            <button
              type='button'
              onClick={() => {
                setOpen(false)
                setModeMenuOpen(false)
                onSkipSettingsOpenChange(true)
              }}
              className='mt-1 flex h-9 w-full items-center justify-between rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
            >
              <span>设置片头片尾</span>
              <span className='flex items-center gap-1.5 text-xs text-muted-foreground'>
                {skipSummary}
                <ChevronRightIcon className='size-4 stroke-[2.8]' />
              </span>
            </button>
            <div className='my-2 h-px bg-border' />
            <button
              type='button'
              onClick={() => void navigate({ to: '/settings/display' })}
              className='flex h-9 w-full items-center gap-2.5 rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
            >
              更多设置
              <span className='rounded-md bg-accent px-1.5 py-0.5 text-xs font-bold tracking-normal text-accent-foreground'>
                NEW
              </span>
            </button>
            <button
              type='button'
              onClick={() =>
                toast.info(
                  '反馈入口准备中，请先在 GitHub Issue 或项目讨论区提交。'
                )
              }
              className='flex h-9 w-full items-center rounded-lg px-3 text-left text-[15px] font-semibold text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
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
  const isMobile = useIsMobile()
  const outroStartSeconds = Math.max(0, safeDuration - draftOutroSeconds)
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
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

  const previewSection = (
    <div className='min-h-0 flex-1'>
      <div className='relative h-full min-h-48 overflow-hidden rounded-xl bg-muted/30 sm:min-h-72'>
        {posterUrl && !videoPreviewReady ? (
          <img
            src={posterUrl}
            alt=''
            className='absolute inset-0 h-full w-full object-contain opacity-80'
          />
        ) : null}
        <video
          ref={previewVideoRef}
          src={playbackUrl}
          muted
          playsInline
          preload='metadata'
          crossOrigin='anonymous'
          onLoadedMetadata={() => setVideoPreviewReady(true)}
          onCanPlay={() => setVideoPreviewReady(true)}
          onError={() => setVideoPreviewReady(false)}
          className={`h-full w-full object-contain ${videoPreviewReady ? 'opacity-80' : 'opacity-0'}`}
        />
        {!posterUrl && !videoPreviewReady ? (
          <div className='flex h-full items-center justify-center bg-muted/30 text-muted-foreground'>
            暂无预览画面
          </div>
        ) : null}
      </div>
    </div>
  )

  const sliderSection = (
    <div className='mt-5'>
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
        className='w-full [&_[data-slot=slider-range]]:bg-foreground/35 [&_[data-slot=slider-thumb]]:size-4 [&_[data-slot=slider-thumb]]:border-2 [&_[data-slot=slider-thumb]]:border-foreground [&_[data-slot=slider-thumb]]:bg-foreground sm:[&_[data-slot=slider-thumb]]:size-5 [&_[data-slot=slider-track]]:h-1.5 [&_[data-slot=slider-track]]:bg-muted sm:[&_[data-slot=slider-track]]:h-2'
      />
      <div className='mt-4 grid gap-2 text-base font-bold tracking-[-0.04em] sm:mt-5 sm:flex sm:items-start sm:justify-between sm:gap-4 sm:text-xl'>
        <div className='min-w-0'>
          片头 {formatTimelineTime(0)} - {formatTimelineTime(draftIntroSeconds)}
        </div>
        <div className='min-w-0 text-right sm:text-left'>
          <span className='text-primary'>
            {formatTimelineTime(outroStartSeconds)}
          </span>{' '}
          - {formatTimelineTime(safeDuration)} 片尾
        </div>
      </div>
    </div>
  )

  const footerSection = (
    <>
      <div className='flex items-start gap-2 text-sm font-semibold text-muted-foreground sm:text-lg'>
        <InfoIcon className='mt-0.5 size-4 sm:mt-0 sm:size-6' />
        <span>仅针对同一文件夹选集生效</span>
      </div>
      <div className='grid grid-cols-2 gap-3 sm:flex sm:items-center sm:gap-5'>
        <button
          type='button'
          onClick={() => {
            setDraftIntroSeconds(0)
            setDraftOutroSeconds(0)
          }}
          className='h-12 min-w-0 rounded-lg bg-muted/40 px-4 text-base font-bold transition-colors hover:bg-accent sm:h-15 sm:min-w-36 sm:px-8 sm:text-xl'
        >
          重置
        </button>
        <button
          type='button'
          onClick={() => onConfirm(draftIntroSeconds, draftOutroSeconds)}
          className='h-12 min-w-0 rounded-lg bg-primary px-4 text-base font-bold text-primary-foreground transition-colors hover:bg-primary/90 sm:h-15 sm:min-w-44 sm:px-8 sm:text-xl'
        >
          确认设置
        </button>
      </div>
    </>
  )

  if (isMobile) {
    return (
      <Drawer open onOpenChange={(open) => !open && onClose()}>
        <DrawerContent className='border-border bg-background text-foreground'>
          <DrawerHeader className='px-4 pt-4 pb-2 text-left'>
            <DrawerTitle className='pr-10 text-lg leading-snug tracking-[-0.04em]'>
              片头时长{formatSkipDuration(draftIntroSeconds)}，片尾时长
              {formatSkipDuration(draftOutroSeconds)}
            </DrawerTitle>
          </DrawerHeader>
          <div className='grid max-h-[78svh] gap-4 overflow-y-auto px-4 pb-4'>
            <div className='h-52'>{previewSection}</div>
            {sliderSection}
            <div className='grid gap-4'>{footerSection}</div>
          </div>
        </DrawerContent>
      </Drawer>
    )
  }
  return (
    <div className='fixed inset-0 z-100 flex bg-background text-foreground'>
      <div className='flex min-h-0 w-full flex-col px-4 pt-4 pb-5 sm:px-8 sm:pt-7 sm:pb-8 md:px-14'>
        <div className='mb-5 flex items-start justify-between gap-4 sm:mb-8 sm:items-center'>
          <div className='pr-2 text-lg leading-snug font-bold tracking-[-0.04em] sm:text-2xl sm:leading-none'>
            片头时长{formatSkipDuration(draftIntroSeconds)}，片尾时长
            {formatSkipDuration(draftOutroSeconds)}
          </div>
          <button
            type='button'
            aria-label='关闭片头片尾设置'
            onClick={onClose}
            className='flex size-10 shrink-0 items-center justify-center rounded-full text-foreground transition-colors hover:bg-accent sm:size-12'
          >
            <XIcon className='size-6 stroke-[2.7] sm:size-8' />
          </button>
        </div>
        {previewSection}
        {sliderSection}
        <div className='mt-6 flex flex-col gap-4 sm:mt-10 sm:flex-row sm:items-center sm:justify-between sm:gap-6'>
          {footerSection}
        </div>
      </div>
    </div>
  )
}

function formatSkipDuration(seconds: number) {
  if (seconds <= 0) return '0秒'
  const minutes = Math.floor(seconds / 60)
  const remainder = seconds % 60
  if (minutes > 0 && remainder > 0) return `${minutes}分${remainder}秒`
  if (minutes > 0) return `${minutes}分`
  return `${remainder}秒`
}
function formatTimelineTime(seconds: number) {
  const total = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const remainder = total % 60
  return [hours, minutes, remainder]
    .map((value) => String(value).padStart(2, '0'))
    .join(':')
}

function aggregatedPlaybackPosition(
  parts: Array<{ duration_seconds?: number }>,
  activePartIndex: number,
  localSeconds: number
) {
  let offset = 0
  for (let index = 0; index < activePartIndex; index += 1) {
    offset += parts[index]?.duration_seconds ?? 0
  }
  return Math.max(0, Math.round(offset + localSeconds))
}

function totalPlaybackPartsDuration(
  parts: Array<{ duration_seconds?: number }>
) {
  const total = parts.reduce(
    (sum, part) => sum + (part.duration_seconds ?? 0),
    0
  )
  return total > 0 ? total : 0
}

function resolvePlaybackPartPosition(
  parts: Array<{ duration_seconds?: number }>,
  aggregateSeconds: number
) {
  if (parts.length === 0 || aggregateSeconds <= 0) {
    return { partIndex: 0, localSeconds: aggregateSeconds }
  }
  let remaining = aggregateSeconds
  for (let index = 0; index < parts.length; index += 1) {
    const duration = parts[index]?.duration_seconds ?? 0
    if (duration <= 0) {
      break
    }
    if (remaining < duration) {
      return { partIndex: index, localSeconds: remaining }
    }
    remaining -= duration
  }
  return {
    partIndex: Math.max(0, parts.length - 1),
    localSeconds: remaining,
  }
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
function PlaybackLoadingScreen({ label }: { label: string }) {
  return (
    <div className='flex h-svh w-full items-center justify-center bg-background text-foreground'>
      <div className='flex items-center gap-3 rounded-full border border-border/60 bg-card px-5 py-3 backdrop-blur-xl'>
        <LoaderCircleIcon className='size-4 animate-spin' />
        <span className='text-sm text-muted-foreground'>{label}</span>
      </div>
    </div>
  )
}
async function captureScreenshot(
  playerRef: ArtPlayerRef,
  playbackHeader: { title: string; subtitle: string }
) {
  const player = playerRef.current
  if (!player) {
    toast.error('当前画面无法截图')
    return
  }
  try {
    await player.screenshot(
      safeFilename(
        [playbackHeader.title, playbackHeader.subtitle]
          .filter(Boolean)
          .join('-') || 'mibo-screenshot'
      )
    )
    toast.success('截图已保存')
  } catch {
    toast.error('当前画面无法截图')
  }
}
function formatClock(seconds?: number) {
  if (!seconds || seconds <= 0) return '00:00'
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
function formatPlaybackRate(rate: number) {
  return rate === 1 ? '倍速' : `${rate}x`
}
function formatReleaseYear(item: CatalogItemDetail | null) {
  if (typeof item?.year === 'number' && item.year > 0) {
    return String(item.year)
  }
  const value = item?.release_date ?? item?.first_air_date
  if (!value) return ''
  const match = value.match(/^(\d{4})/)
  return match?.[1] ?? ''
}
function formatOfficialRating(value?: string) {
  const rating = value?.trim()
  return rating ? `分级 ${rating}` : ''
}
function formatGenreLabel(genres?: string[]) {
  const primaryGenre = genres?.find((genre) => genre.trim())
  return primaryGenre?.trim() ?? ''
}
function formatResolutionLabel(width?: number, height?: number) {
  if (
    typeof width !== 'number' ||
    width <= 0 ||
    typeof height !== 'number' ||
    height <= 0
  ) {
    return ''
  }
  const longEdge = Math.max(width, height)
  if (longEdge >= 3800) return '4K'
  if (longEdge >= 2500) return '1440p'
  if (longEdge >= 1800) return '1080p'
  if (longEdge >= 1200) return '720p'
  return `${width}x${height}`
}
function formatCodecLabel(codec?: string) {
  const value = codec?.trim()
  return value ? value.toUpperCase() : ''
}
function formatAudioTrackLabel(track?: Track) {
  if (!track) return ''
  const codec = track.codec?.trim().toUpperCase()
  const channels = formatAudioChannels(track.channels)
  if (codec && channels) return `${codec} ${channels}`
  return codec || channels
}
function formatAudioTrackMenuLabel(track: Track, index: number) {
  const title = track.title?.trim()
  const language = formatLanguageCode(track.language)
  if (title && language) return `${title} · ${language}`
  if (title) return title
  if (language) return language
  return `音轨 ${index + 1}`
}
function isPlaybackVersionResource(resource: MetadataResourceDetail) {
  if (resource.segment_index) return false
  if (!resource.role) return true
  return resource.role === 'primary' || resource.role === 'version'
}
function getAudioTrackKey(track: Track, index: number) {
  if (typeof track.stream_index === 'number' && track.stream_index >= 0) {
    return `stream-${track.stream_index}`
  }
  return `${track.language}-${track.title}-${index}`
}
function formatAudioChannels(channels?: number) {
  if (typeof channels !== 'number' || channels <= 0) return ''
  if (channels === 1) return '1.0'
  if (channels === 2) return '2.0'
  return `${Math.max(1, Math.floor(channels - 1))}.1`
}
function formatSubtitleCountLabel(subtitleTracks?: Track[]) {
  const count = subtitleTracks?.length ?? 0
  return count > 0 ? `${count} 字幕` : '无字幕'
}
function formatPlaybackModeBadge(playback: PlaybackSource) {
  if (playback.selected_variant?.kind === 'audio-repair') {
    return '音频修复'
  }
  if (playback.selected_variant?.kind === 'quality') {
    return playback.selected_variant.label
  }
  if (playback.decision.kind === 'direct' || playback.direct) {
    return '直连'
  }
  if (playback.decision.kind === 'fallback') {
    return '转码'
  }
  return ''
}
function qualityVariantDescription(variant: PlaybackVariant) {
  if (variant.kind === 'original') return '原始码流'
  if (variant.kind === 'audio-repair') return 'AAC'
  if (variant.height) return `${variant.height}p`
  return variant.requires_ffmpeg ? '转码' : ''
}
function extractTranscodeSessionId(streamUrl?: string) {
  const value = streamUrl?.trim()
  if (!value) return ''
  const match = value.match(/\/api\/v1\/transcodes\/([^/?#]+)(?:\/|$)/)
  return match?.[1] ? decodeURIComponent(match[1]) : ''
}
function isTranscodeStreamUrl(streamUrl?: string) {
  return Boolean(extractTranscodeSessionId(streamUrl))
}
function formatContainerLabel(container?: string) {
  const value = container?.trim()
  return value ? value.toUpperCase() : ''
}
function captureProgressFrame(video?: HTMLVideoElement | null) {
  if (!video || video.readyState < HTMLMediaElement.HAVE_CURRENT_DATA)
    return undefined
  const sourceWidth = video.videoWidth
  const sourceHeight = video.videoHeight
  if (sourceWidth <= 0 || sourceHeight <= 0) return undefined
  try {
    const width = Math.min(1280, sourceWidth)
    const height = Math.max(1, Math.round((sourceHeight / sourceWidth) * width))
    const canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    const context = canvas.getContext('2d')
    if (!context) return undefined
    context.drawImage(video, 0, 0, width, height)
    return canvas.toDataURL('image/webp', 0.88)
  } catch {
    return undefined
  }
}
function safeFilename(value: string) {
  return value
    .trim()
    .replace(/[\\/:*?"<>|]/g, '-')
    .replace(/\s+/g, ' ')
    .slice(0, 120)
}
function catalogImageUrl(
  item: { selected_images?: { image_type: string; url: string }[] },
  imageType: string
) {
  return item.selected_images?.find((image) => image.image_type === imageType)
    ?.url
}
