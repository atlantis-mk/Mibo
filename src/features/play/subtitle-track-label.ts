import { formatLanguageCode } from '@/lib/language-code'

export function extractSubtitleDisplayTitle(title?: string) {
  const normalizedTitle = title?.trim()
  if (!normalizedTitle) return ''

  const parts = normalizedTitle.match(/^\S+\s+(.+)$/)
  return parts?.[1]?.trim() || normalizedTitle
}

export function formatSubtitleTrackMenuLabel(
  title?: string,
  language?: string,
  index?: number
) {
  const displayTitle = extractSubtitleDisplayTitle(title)
  if (displayTitle) return displayTitle

  const normalizedLanguage = formatLanguageCode(language)
  if (normalizedLanguage) return normalizedLanguage

  if (typeof index === 'number') {
    return `字幕 ${index + 1}`
  }

  return '字幕'
}
