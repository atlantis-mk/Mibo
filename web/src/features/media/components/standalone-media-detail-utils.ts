import type {
  CatalogAssetDetail,
  CatalogAssetFileSummary,
  CatalogSourceEvidence,
  Track,
} from '#/lib/mibo-api'

export function getPrimaryCatalogAsset(
  item: Pick<{ assets?: CatalogAssetDetail[] }, 'assets'>,
) {
  return item.assets?.[0]
}

export function getDisplayDatabaseLinks(
  item: Pick<
    { metadata_provider: string; external_id: string },
    'metadata_provider' | 'external_id'
  >,
) {
  return [
    item.metadata_provider?.toUpperCase() || null,
    item.external_id || null,
  ]
    .filter(Boolean)
    .join('，')
}

export function getDisplayMatchStatus(
  item: Pick<{ governance_status: string }, 'governance_status'>,
) {
  return item.governance_status || 'pending'
}

export function getDisplaySourcePath(
  item: Pick<{ source_evidence: CatalogSourceEvidence[] }, 'source_evidence'>,
) {
  for (const evidence of item.source_evidence ?? []) {
    if (!evidence.summary || typeof evidence.summary !== 'object') {
      continue
    }
    const storagePath = (evidence.summary as { storage_path?: unknown })
      .storage_path
    if (typeof storagePath === 'string' && storagePath.trim()) {
      return storagePath.trim()
    }
  }
  return 'catalog item'
}

export function formatAudioTrackLabel(track?: Track) {
  if (!track) return '未知'
  return [
    track.language || null,
    track.title || track.codec || null,
    formatChannels(track),
  ]
    .filter(Boolean)
    .join(' ')
}

export function formatChannels(track?: Pick<Track, 'channels'>) {
  if (!track?.channels || track.channels <= 0) return '立体声'
  if (track.channels === 1) return '单声道'
  if (track.channels === 2) return 'stereo'
  return `${track.channels} ch`
}

export function formatChannelsCompact(track?: Pick<Track, 'channels'>) {
  if (!track?.channels || track.channels <= 0) return '未知'
  return `${track.channels} ch`
}

export function formatStreamLanguage(language?: string) {
  const value = formatTechnicalValue(language)
  if (!value) return ''

  const normalized = value.toLowerCase()
  const fallbackLabels: Record<string, string> = {
    jpn: 'Japanese',
    ja: 'Japanese',
    eng: 'English',
    en: 'English',
    zho: 'Chinese',
    chi: 'Chinese',
    zh: 'Chinese',
  }
  if (fallbackLabels[normalized]) {
    return fallbackLabels[normalized]
  }

  try {
    const label = new Intl.DisplayNames(['en'], { type: 'language' }).of(value)
    return label || value
  } catch {
    return value
  }
}

export function formatCodecLabel(codec?: string) {
  return formatTechnicalValue(codec).toUpperCase()
}

export function formatAudioLayout(layout?: string, channels?: number) {
  const value = formatTechnicalValue(layout)
  if (value) return value
  if (channels === 1) return 'mono'
  if (channels === 2) return 'stereo'
  return ''
}

export function formatSampleRate(value?: number) {
  if (!value || value <= 0) return ''
  return `${new Intl.NumberFormat('en-US').format(value)} Hz`
}

export function formatAudioBitDepth(value?: number) {
  if (!value || value <= 0) return ''
  return `${value} bit`
}

export function formatBooleanFlag(value?: boolean) {
  return value ? '是' : '否'
}

export function fileNameFromStoragePath(path?: string) {
  const value = formatTechnicalValue(path)
  if (!value) return ''
  const segments = value.split('/')
  return segments[segments.length - 1] || value
}

export function findAssetFileName(
  files: CatalogAssetFileSummary[] | undefined,
  fileID: number,
) {
  const match = files?.find((file) => file.file_id === fileID)
  return fileNameFromStoragePath(match?.storage_path)
}

export function formatBitRate(value?: number) {
  if (!value || value <= 0) return '未知'
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)} Mbps`
  if (value >= 1_000) return `${Math.round(value / 1_000)} kbps`
  return `${value} bps`
}

export function formatTechnicalValue(value?: number | string | null) {
  if (typeof value === 'number') return value > 0 ? String(value) : ''
  return value?.trim() ?? ''
}

export function formatFrameRate(primary?: string, fallback?: string) {
  const raw = formatTechnicalValue(primary) || formatTechnicalValue(fallback)
  if (!raw || raw === '0/0') return ''
  const [numeratorRaw, denominatorRaw] = raw.split('/')
  if (!denominatorRaw) return raw

  const numerator = Number(numeratorRaw)
  const denominator = Number(denominatorRaw)
  if (
    !Number.isFinite(numerator) ||
    !Number.isFinite(denominator) ||
    denominator <= 0
  ) {
    return raw
  }
  const value = numerator / denominator
  if (value <= 0) return ''
  return `${value
    .toFixed(value >= 100 ? 0 : 3)
    .replace(/\.0+$/, '')
    .replace(/(\.\d*?)0+$/, '$1')} fps`
}

export function formatInterlaceState(fieldOrder?: string) {
  switch (formatTechnicalValue(fieldOrder).toLowerCase()) {
    case 'progressive':
      return '否（逐行）'
    case 'tt':
    case 'bb':
    case 'tb':
    case 'bt':
      return '是（隔行）'
    case 'unknown':
      return ''
    default:
      return formatTechnicalValue(fieldOrder)
  }
}

export function formatCodecLevel(value?: number, codec?: string) {
  if (!value || value <= 0) return ''
  const normalizedCodec = codec?.toLowerCase() ?? ''
  if (
    (normalizedCodec.includes('h264') || normalizedCodec.includes('avc')) &&
    value >= 10 &&
    value < 100
  ) {
    return `${Math.floor(value / 10)}.${value % 10}`
  }
  return String(value)
}

export function formatBitDepth(value?: number) {
  if (!value || value <= 0) return ''
  return `${value}-bit`
}

export function formatFileSize(value?: number) {
  if (!value || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = value
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }
  return `${size.toFixed(size >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`
}

export function simplifyAspectRatio(width: number, height: number) {
  const divisor = greatestCommonDivisor(width, height)
  return `${width / divisor}:${height / divisor}`
}

function greatestCommonDivisor(a: number, b: number): number {
  let left = a
  let right = b
  while (right !== 0) {
    const remainder = left % right
    left = right
    right = remainder
  }
  return left || 1
}

export function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

export function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

export function formatMediaType(type: string) {
  if (type === 'movie') return '电影'
  if (type === 'show' || type === 'series' || type === 'episode') return '剧集'
  if (type === 'season') return '季度'
  return '媒体'
}

export function formatProbeStatus(status: string) {
  switch (status) {
    case 'ready':
    case 'complete':
    case 'done':
      return '已就绪'
    case 'failed':
      return '分析失败'
    case 'processing':
    case 'probing':
    case 'processing':
      return '分析中'
    default:
      return '等待分析'
  }
}

export function describeMatchStatus(status: string) {
  switch (status) {
    case 'matched':
      return ''
    case 'needs_review':
      return '该条目的元数据需要人工复核后再确认。'
    case 'manual':
      return '该条目当前使用人工治理结果。'
    case 'locked':
      return '该条目的关键字段已锁定，刷新不会直接覆盖。'
    case 'pending':
      return '该条目还未完成元数据匹配。'
    case 'searching':
      return '系统正在为该条目搜索更准确的元数据。'
    case 'failed':
      return '最近一次元数据匹配失败，可以尝试重新匹配。'
    case 'unmatched':
      return '当前没有找到合适的元数据结果。'
    default:
      return ''
  }
}

export function formatAvailabilityStatus(status: string) {
  switch (status) {
    case 'available':
      return '可播放'
    case 'missing':
      return '缺失'
    case 'unaired':
      return '未播出'
    case 'no_local_media':
      return '无本地资源'
    default:
      return status || '未知状态'
  }
}

export function formatAssetLabel(asset?: CatalogAssetDetail) {
  if (!asset) return '暂无已链接资源'
  return (
    [asset.display_name, asset.edition, asset.quality_label]
      .filter(Boolean)
      .join(' · ') ||
    asset.asset_type ||
    `资源 ${asset.id}`
  )
}

export function formatRuntime(value?: number) {
  if (!value || value <= 0) return ''
  const totalMinutes = Math.round(value / 60)
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours <= 0) return `${minutes} 分钟`
  if (minutes === 0) return `${hours} 小时`
  return `${hours} 小时 ${minutes} 分钟`
}

export function formatSeconds(value?: number) {
  if (!value || value <= 0) return '00:00'
  const totalSeconds = Math.max(0, Math.floor(value))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  if (hours > 0) {
    return [hours, minutes, seconds]
      .map((part) => String(part).padStart(2, '0'))
      .join(':')
  }
  return [minutes, seconds]
    .map((part) => String(part).padStart(2, '0'))
    .join(':')
}
