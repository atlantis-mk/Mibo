import type {
  CatalogListItem,
  CatalogUserItemEntry,
  HomeContentSection,
  HomeMediaOverview,
  OperationsTask,
} from '@/lib/mibo-api'
import { findBlockingHomeTask } from '@/lib/operations-presentation'

export type HomeDashboardData = {
  items: CatalogListItem[]
  continueWatching: CatalogUserItemEntry[]
  continueWatchingCount: number
  contentSections: HomeContentSection[]
  mediaOverview: HomeMediaOverview
  operationsTasks: OperationsTask[]
}

export function getHomeDashboardState(data: HomeDashboardData) {
  const activeOperationsTasks = data.operationsTasks.filter(
    (task) => (task.lifecycle_status ?? 'active') !== 'resolved'
  )
  const contentSections = data.contentSections.filter(
    (section) => section.items.length > 0
  )
  const homeBlockingTask = findBlockingHomeTask(activeOperationsTasks)
  const hasDisplayableHomeContent =
    data.items.length > 0 ||
    contentSections.length > 0 ||
    data.continueWatching.length > 0
  const hasEmptySetupState =
    data.items.length === 0 &&
    contentSections.length === 0 &&
    data.continueWatching.length === 0

  return {
    contentSections,
    activeOperationsTasks,
    homeBlockingTask,
    hasDisplayableHomeContent,
    hasEmptySetupState,
    isHealthBlocked: !hasDisplayableHomeContent && !!homeBlockingTask,
    isPartiallyDegraded:
      hasDisplayableHomeContent && activeOperationsTasks.length > 0,
    mediaOverviewSections: data.mediaOverview.sections.filter(
      (section) => section.count > 0 || section.items.length > 0
    ),
    movieCount: getHomeMediaCount(
      data,
      'movies',
      (item) => item.type === 'movie'
    ),
    showCount: getHomeMediaCount(
      data,
      'series',
      (item) => item.type === 'show' || item.type === 'series'
    ),
  }
}

function getHomeMediaCount(
  data: HomeDashboardData,
  key: string,
  predicate: (item: CatalogListItem) => boolean
) {
  const overviewSection = data.mediaOverview.sections.find(
    (section) => section.key === key
  )
  if (overviewSection) {
    return overviewSection.count
  }

  const items = data.contentSections.flatMap((section) => section.items)
  const sourceItems = items.length > 0 ? items : data.items
  return sourceItems.filter(predicate).length
}
