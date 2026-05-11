import { describe, expect, it } from "vitest"

import type { CatalogListItem, HealthIssue } from "#/lib/mibo-api"

import { getHomeDashboardState, type HomeDashboardData } from "./home-state"

function homeItem(overrides: Partial<CatalogListItem> = {}): CatalogListItem {
  return {
    id: 1,
    library_id: 1,
    type: "movie",
    title: "Movie",
    availability_status: "available",
    governance_status: "accepted",
    ...overrides,
  }
}

function issue(overrides: Partial<HealthIssue> = {}): HealthIssue {
  return {
    id: "blocking-1",
    severity: "blocking",
    reason_code: "storage_auth_expired",
    scope: "media_source",
    title: "Storage auth expired",
    message: "Repair provider auth",
    impact: {
      blocks_scan: true,
      blocks_home_visibility: true,
      blocks_playback: true,
      blocks_metadata: false,
      affected_metadata_items: 10,
      affected_files: 20,
    },
    affected: { media_sources: [], libraries: [], jobs: [] },
    actions: [],
    technical_detail: {},
    ...overrides,
  }
}

function data(overrides: Partial<HomeDashboardData> = {}): HomeDashboardData {
  return {
    items: [],
    continueWatching: [],
    continueWatchingCount: 0,
    contentSections: [],
    mediaOverview: { sections: [] },
    healthIssues: [],
    ...overrides,
  }
}

describe("home regression guardrails", () => {
  it("keeps visible content renderable while only partially degraded", () => {
    const state = getHomeDashboardState(
      data({
        contentSections: [
          {
            key: "mixed",
            title: "最近内容",
            items: [homeItem({ id: 1 }), homeItem({ id: 2, type: "show" })],
          },
        ],
        mediaOverview: {
          sections: [
            { key: "movies", title: "电影", count: 9, items: [homeItem()] },
            {
              key: "series",
              title: "剧集",
              count: 4,
              items: [homeItem({ id: 2, type: "show" })],
            },
          ],
        },
        healthIssues: [issue()],
      })
    )

    expect(state.hasDisplayableHomeContent).toBe(true)
    expect(state.isHealthBlocked).toBe(false)
    expect(state.isPartiallyDegraded).toBe(true)
    expect(state.movieCount).toBe(9)
    expect(state.showCount).toBe(4)
  })

  it("treats empty visible content plus blocking health as a hard block", () => {
    const state = getHomeDashboardState(
      data({
        healthIssues: [issue()],
      })
    )

    expect(state.hasDisplayableHomeContent).toBe(false)
    expect(state.isHealthBlocked).toBe(true)
    expect(state.homeBlockingIssue?.impact.blocks_home_visibility).toBe(true)
  })
})
