export function formatMediaType(value: string) {
  if (value === 'movie') {
    return '电影'
  }
  if (value === 'series' || value === 'show') {
    return '剧集'
  }
  if (value === 'season') {
    return '季度'
  }
  if (value === 'episode') {
    return '剧集'
  }
  return value || '未知类型'
}

export function formatMatchStatus(value: string) {
  switch (value) {
    case 'matched':
      return '已匹配'
    case 'needs_review':
      return '待复核'
    case 'unmatched':
      return '未匹配'
    case 'skipped':
      return '已跳过'
    case 'pending':
      return '待处理'
    case 'manual':
      return '人工治理'
    case 'locked':
      return '字段锁定'
    default:
      return value || '未知状态'
  }
}

export function formatClassificationStatus(value: string) {
  switch (value) {
    case 'confirmed_fast':
      return '快速确认'
    case 'provisional':
      return '待后台验证'
    case 'review_required':
      return '需要复核'
    default:
      return value || '未知状态'
  }
}

export function formatClassificationType(value: string) {
  switch (value) {
    case 'movie':
      return '电影'
    case 'episode':
      return '单集'
    case 'attachment':
      return '附属视频'
    case 'movie_version':
      return '电影版本'
    case 'independent_movie':
      return '独立电影'
    default:
      return value || '未知候选'
  }
}
