export const DEFAULT_LOCAL_MEDIA_ROOT_PATH =
  '/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media'

export const DEFAULT_OPENLIST_BASE_URL = 'http://127.0.0.1:5244'

export const STORAGE_PROVIDER_OPTIONS = [
  {
    value: 'local',
    label: '本地文件夹',
    description: '直接接入当前机器上的媒体目录',
    examplePath: DEFAULT_LOCAL_MEDIA_ROOT_PATH,
  },
  {
    value: 'openlist',
    label: '统一挂载路径',
    description: '通过统一挂载入口接入 NAS 或云盘目录',
    examplePath: '/media',
  },
] as const

export const LIBRARY_TYPE_OPTIONS = [
  {
    value: 'movies',
    label: '电影',
    description: '适合单文件电影内容',
    pathSuffix: 'Movies',
  },
  {
    value: 'shows',
    label: '剧集',
    description: '适合按剧集扫描目录',
    pathSuffix: 'Shows',
  },
] as const

export function buildSuggestedLibraryRootPath(rootPath: string, pathSuffix: string) {
  const trimmedRootPath = rootPath.trim()

  if (trimmedRootPath === '' || trimmedRootPath === '/') {
    return `/${pathSuffix}`
  }

  return `${trimmedRootPath.replace(/[\\/]+$/, '')}/${pathSuffix}`
}
