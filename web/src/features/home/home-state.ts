import { findBlockingHomeIssue } from "#/lib/health-presentation"
import type {
  CatalogLatestByLibrarySection,
  CatalogListItem,
  CatalogUserItemEntry,
  HealthIssue,
  Library,
} from "#/lib/mibo-api"

export type HomeDashboardData = {
  items: CatalogListItem[]
  continueWatching: CatalogUserItemEntry[]
  continueWatchingCount: number
  libraries: Library[]
  libraryCount: number
  latestByLibrary: CatalogLatestByLibrarySection[]
  healthIssues: HealthIssue[]
}

export function getHomeDashboardState(data: HomeDashboardData) {
  const latestLibrarySections = data.latestByLibrary.filter(
    (section) => section.items.length > 0
  )
  const homeBlockingIssue = findBlockingHomeIssue(data.healthIssues)
  const hasDisplayableHomeContent =
    data.items.length > 0 ||
    latestLibrarySections.length > 0 ||
    data.continueWatching.length > 0
  const hasEmptySetupState =
    data.items.length === 0 &&
    data.libraries.length === 0 &&
    latestLibrarySections.length === 0 &&
    data.continueWatching.length === 0

  return {
    latestLibrarySections,
    homeBlockingIssue,
    hasDisplayableHomeContent,
    hasEmptySetupState,
    isHealthBlocked: !hasDisplayableHomeContent && !!homeBlockingIssue,
    isPartiallyDegraded:
      hasDisplayableHomeContent && data.healthIssues.length > 0,
    movieCount: data.items.filter((item) => item.type === "movie").length,
    showCount: data.items.filter(
      (item) => item.type === "show" || item.type === "series"
    ).length,
  }
}
