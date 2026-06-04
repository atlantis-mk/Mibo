export function normalizeLanguageCode(value?: string) {
  const normalized = value?.trim().toLowerCase().replace(/_/g, '-')
  if (!normalized) return ''

  switch (normalized) {
    case 'eng':
      return 'en'
    case 'chi':
    case 'zho':
    case 'cmn':
      return 'zh'
    case 'zht':
    case 'cht':
    case 'zh-hant':
    case 'zh-hk':
      return 'zh-tw'
    case 'zh-hans':
      return 'zh-cn'
    case 'jpn':
      return 'ja'
    case 'kor':
      return 'ko'
    case 'fre':
    case 'fra':
      return 'fr'
    case 'ger':
    case 'deu':
      return 'de'
    case 'spa':
      return 'es'
    case 'por':
      return 'pt'
    case 'rus':
      return 'ru'
    default:
      return normalized
  }
}

export function formatLanguageCode(value?: string) {
  return normalizeLanguageCode(value)
}
