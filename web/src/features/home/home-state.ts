import { findBlockingHomeIssue } from "#/lib/health-presentation"
import type {
  CatalogListItem,
  CatalogUserItemEntry,
  HealthIssue,
  HomeContentSection,
  HomeMediaOverview,
} from "#/lib/mibo-api"

export type HomeDashboardData = {
  items: CatalogListItem[]
  continueWatching: CatalogUserItemEntry[]
  continueWatchingCount: number
  contentSections: HomeContentSection[]
  mediaOverview: HomeMediaOverview
  healthIssues: HealthIssue[]
}

export function getHomeDashboardState(data: HomeDashboardData) {
  const contentSections = data.contentSections.filter(
    (section) => section.items.length > 0
  )
  const homeBlockingIssue = findBlockingHomeIssue(data.healthIssues)
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
    homeBlockingIssue,
    hasDisplayableHomeContent,
    hasEmptySetupState,
    isHealthBlocked: !hasDisplayableHomeContent && !!homeBlockingIssue,
    isPartiallyDegraded:
      hasDisplayableHomeContent && data.healthIssues.length > 0,
    mediaOverviewSections: data.mediaOverview.sections.filter(
      (section) => section.count > 0 || section.items.length > 0
    ),
    movieCount: getHomeMediaCount(data, "movies", (item) => item.type === "movie"),
    showCount: getHomeMediaCount(
      data,
      "series",
      (item) => item.type === "show" || item.type === "series"
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
