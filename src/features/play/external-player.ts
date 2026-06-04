export type PlaybackLaunchMode = 'internal' | 'external'

export type ExternalPlayerId =
  | 'vlc'
  | 'iina'
  | 'potplayer'
  | 'nplayer'
  | 'omniplayer'
  | 'figplayer'
  | 'infuse'
  | 'fileball'
  | 'mxplayer'
  | 'mxplayer-pro'
  | 'iplay'
  | 'mpv'
  | 'android'

export type PlaybackLaunchPreferences = {
  mode: PlaybackLaunchMode
  externalPlayerId: ExternalPlayerId
}

export type ExternalPlayerOption = {
  id: ExternalPlayerId
  name: string
  scheme: string
  platforms: string[]
}

type OpenExternalPlayerInput = {
  playbackUrl: string
  title?: string
  playerId?: ExternalPlayerId
}

const STORAGE_KEY = 'mibo-playback-launch-preferences'

export const EXTERNAL_PLAYER_OPTIONS: ExternalPlayerOption[] = [
  {
    id: 'iina',
    name: 'IINA',
    scheme: 'iina://weblink?url=$edurl',
    platforms: ['macOS'],
  },
  {
    id: 'potplayer',
    name: 'PotPlayer',
    scheme: 'potplayer://$durl',
    platforms: ['Windows'],
  },
  {
    id: 'vlc',
    name: 'VLC',
    scheme: 'vlc://$durl',
    platforms: ['Windows', 'macOS', 'Linux', 'Android', 'iOS'],
  },
  {
    id: 'android',
    name: 'Android Intent',
    scheme: 'intent:$durl#Intent;type=video/*;S.title=$name;end',
    platforms: ['Android'],
  },
  {
    id: 'nplayer',
    name: 'nPlayer',
    scheme: 'nplayer-$durl',
    platforms: ['Android', 'iOS'],
  },
  {
    id: 'omniplayer',
    name: 'OmniPlayer',
    scheme: 'omniplayer://weblink?url=$durl',
    platforms: ['macOS'],
  },
  {
    id: 'figplayer',
    name: 'Fig Player',
    scheme: 'figplayer://weblink?url=$durl',
    platforms: ['Windows', 'macOS'],
  },
  {
    id: 'infuse',
    name: 'Infuse',
    scheme: 'infuse://x-callback-url/play?url=$durl',
    platforms: ['macOS', 'iOS'],
  },
  {
    id: 'fileball',
    name: 'Fileball',
    scheme: 'filebox://play?url=$durl',
    platforms: ['macOS', 'iOS'],
  },
  {
    id: 'mxplayer',
    name: 'MX Player',
    scheme:
      'intent:$durl#Intent;package=com.mxtech.videoplayer.ad;S.title=$name;end',
    platforms: ['Android'],
  },
  {
    id: 'mxplayer-pro',
    name: 'MX Player Pro',
    scheme:
      'intent:$durl#Intent;package=com.mxtech.videoplayer.pro;S.title=$name;end',
    platforms: ['Android'],
  },
  {
    id: 'iplay',
    name: 'iPlay',
    scheme: 'iplay://play/any?type=url&url=$bdurl',
    platforms: ['iOS'],
  },
  {
    id: 'mpv',
    name: 'mpv',
    scheme: 'mpv://$edurl',
    platforms: ['Windows', 'macOS', 'Linux', 'Android'],
  },
]

const DEFAULT_PREFERENCES: PlaybackLaunchPreferences = {
  mode: 'internal',
  externalPlayerId: 'vlc',
}

export function getPlaybackLaunchPreferences(): PlaybackLaunchPreferences {
  if (typeof window === 'undefined') {
    return DEFAULT_PREFERENCES
  }

  const raw = window.localStorage.getItem(STORAGE_KEY)
  if (!raw) {
    return {
      ...DEFAULT_PREFERENCES,
      externalPlayerId: getRecommendedExternalPlayer(),
    }
  }

  try {
    const parsed = JSON.parse(raw) as Partial<PlaybackLaunchPreferences>
    return normalizePlaybackLaunchPreferences(parsed)
  } catch {
    window.localStorage.removeItem(STORAGE_KEY)
    return {
      ...DEFAULT_PREFERENCES,
      externalPlayerId: getRecommendedExternalPlayer(),
    }
  }
}

export function setPlaybackLaunchPreferences(
  preferences: PlaybackLaunchPreferences
) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(
    STORAGE_KEY,
    JSON.stringify(normalizePlaybackLaunchPreferences(preferences))
  )
}

export function normalizePlaybackLaunchPreferences(
  value: Partial<PlaybackLaunchPreferences> | null | undefined
): PlaybackLaunchPreferences {
  const mode = value?.mode === 'external' ? 'external' : 'internal'
  const externalPlayerId = isExternalPlayerId(value?.externalPlayerId)
    ? value.externalPlayerId
    : getRecommendedExternalPlayer()

  return {
    mode,
    externalPlayerId,
  }
}

export function getSupportedExternalPlayerOptions() {
  const platform = getCurrentPlatform()

  return EXTERNAL_PLAYER_OPTIONS.filter((player) =>
    player.platforms.includes(platform)
  )
}

export function getRecommendedExternalPlayer(): ExternalPlayerId {
  const supported = getSupportedExternalPlayerOptions()
  return supported[0]?.id ?? DEFAULT_PREFERENCES.externalPlayerId
}

export function getCurrentPlatform() {
  if (typeof navigator === 'undefined') {
    return 'Unknown'
  }

  const userAgent = navigator.userAgent.toLowerCase()

  if (/android/.test(userAgent)) return 'Android'
  if (/iphone|ipad|ipod/.test(userAgent)) return 'iOS'
  if (/mac os x|macintosh/.test(userAgent)) return 'macOS'
  if (/windows/.test(userAgent)) return 'Windows'
  if (/linux/.test(userAgent)) return 'Linux'

  return 'Unknown'
}

export function openConfiguredExternalPlayer(
  input: OpenExternalPlayerInput
): { ok: true } | { ok: false; message: string } {
  const preferences = getPlaybackLaunchPreferences()

  const playerId = input.playerId ?? preferences.externalPlayerId
  const player = EXTERNAL_PLAYER_OPTIONS.find((item) => item.id === playerId)
  if (!player) {
    return { ok: false, message: '未找到外部播放器配置' }
  }

  const resolvedPlaybackUrl = resolveAbsolutePlaybackUrl(input.playbackUrl)
  if (!resolvedPlaybackUrl) {
    return { ok: false, message: '当前播放地址无效，无法交给外部播放器' }
  }

  const launchUrl = convertExternalPlayerUrl(player.scheme, {
    name: input.title ?? 'Mibo Playback',
    url: resolvedPlaybackUrl,
  })

  window.location.href = launchUrl
  return { ok: true }
}

export function resolveAbsolutePlaybackUrl(url: string) {
  const trimmed = url.trim()
  if (!trimmed) {
    return ''
  }

  try {
    return new URL(trimmed, window.location.origin).toString()
  } catch {
    return ''
  }
}

export function convertExternalPlayerUrl(
  scheme: string,
  args: { url: string; name: string }
) {
  return scheme
    .replace('$name', args.name)
    .replace(/\$[eb_]*durl/g, (placeholder) =>
      applyUrlPlaceholder(placeholder, args.url)
    )
    .replace(/\$[eb_]*url/g, (placeholder) =>
      applyUrlPlaceholder(placeholder, args.url)
    )
}

function applyUrlPlaceholder(placeholder: string, url: string) {
  const operations = placeholder.match(/[eb]/g) ?? []
  let value = url

  for (const operation of operations.reverse()) {
    if (operation === 'e') {
      value = encodeURIComponent(value)
    } else if (operation === 'b') {
      value = window.btoa(value)
    }
  }

  return value
}

function isExternalPlayerId(value: unknown): value is ExternalPlayerId {
  return EXTERNAL_PLAYER_OPTIONS.some((player) => player.id === value)
}
