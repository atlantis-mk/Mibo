import type { MediaFile, Track } from '#/lib/mibo-api'

export function formatVideoTrackLabel(file?: MediaFile) {
  if (!file) return '未知'
  return (
    [file.height ? `${file.height}p` : null, file.video_codec || null]
      .filter(Boolean)
      .join(' ') || '未知'
  )
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

export function formatChannels(track?: Track) {
  if (!track?.channels || track.channels <= 0) return '立体声'
  if (track.channels === 1) return '单声道'
  if (track.channels === 2) return 'stereo'
  return `${track.channels} ch`
}

export function formatChannelsCompact(track?: Track) {
  if (!track?.channels || track.channels <= 0) return '未知'
  return `${track.channels} ch`
}

export function formatBitRate(value?: number) {
  if (!value || value <= 0) return '未知'
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)} Mbps`
  if (value >= 1_000) return `${Math.round(value / 1_000)} kbps`
  return `${value} bps`
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
  if (type === 'show') return '剧集'
  return '媒体'
}

export function formatProbeStatus(status: string) {
  switch (status) {
    case 'done':
      return '已分析'
    case 'failed':
      return '分析失败'
    case 'processing':
      return '分析中'
    default:
      return '等待分析'
  }
}

export function describeMatchStatus(status: string) {
  switch (status) {
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
