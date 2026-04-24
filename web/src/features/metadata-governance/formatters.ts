export function formatMediaType(value: string) {
  if (value === 'movie') {
    return '电影'
  }
  if (value === 'episode') {
    return '剧集'
  }
  if (value === 'show') {
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
    default:
      return value || '未知状态'
  }
}
