export function formatLibraryType(type: string | null | undefined) {
  switch ((type ?? '').trim().toLowerCase()) {
    case 'movies':
    case 'movie':
    case 'films':
      return '电影库'
    case 'shows':
    case 'tv':
    case 'tvshows':
      return '剧集库'
    case 'mixed':
    case 'mixed-content':
    case 'mixed_content':
      return '混合内容'
    default:
      return type || '媒体库'
  }
}
