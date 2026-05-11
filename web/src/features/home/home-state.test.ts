import { describe, expect, it } from "vitest"

import type { CatalogListItem, HealthIssue } from "#/lib/mibo-api"

import { getHomeDashboardState, type HomeDashboardData } from "./home-state"

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

function item(overrides: Partial<CatalogListItem> = {}): CatalogListItem {
  return {
    id: 1,
    library_id: 1,
    type: "movie",
    title: "Movie",
    availability_status: "available",
    governance_status: "ok",
    ...overrides,
  }
}

function blockingIssue(overrides: Partial<HealthIssue> = {}): HealthIssue {
  return {
    id: "storage-auth",
    severity: "blocking",
    reason_code: "storage_auth_expired",
    scope: "media_source",
    title: "Storage auth expired",
    message: "Repair provider auth",
    impact: {
      blocks_scan: true,
      blocks_home_visibility: true,
      blocks_playback: false,
      blocks_metadata: false,
      affected_metadata_items: 10,
      affected_files: 20,
    },
    affected: {
      media_sources: [],
      libraries: [
        {
          id: 1,
          name: "Movies",
          type: "movie",
          status: "error",
          media_source_id: 1,
          root_path: "/movies",
        },
      ],
      jobs: [],
    },
    actions: [],
    technical_detail: {},
    ...overrides,
  }
}

describe("getHomeDashboardState", () => {
  it("covers the empty setup state", () => {
    const state = getHomeDashboardState(data())

    expect(state.hasEmptySetupState).toBe(true)
    expect(state.hasDisplayableHomeContent).toBe(false)
    expect(state.isHealthBlocked).toBe(false)
  })

  it("covers the normal populated state", () => {
    const state = getHomeDashboardState(
      data({
        contentSections: [{ key: "movies", title: "电影", items: [item()] }],
        mediaOverview: {
          sections: [{ key: "movies", title: "电影", count: 7, items: [item()] }],
        },
      })
    )

    expect(state.hasDisplayableHomeContent).toBe(true)
    expect(state.movieCount).toBe(7)
    expect(state.showCount).toBe(0)
    expect(state.isPartiallyDegraded).toBe(false)
  })

  it("covers the fully health-blocked state", () => {
    const state = getHomeDashboardState(
      data({
        healthIssues: [blockingIssue()],
      })
    )

    expect(state.hasEmptySetupState).toBe(true)
    expect(state.hasDisplayableHomeContent).toBe(false)
    expect(state.isHealthBlocked).toBe(true)
    expect(state.homeBlockingIssue?.id).toBe("storage-auth")
  })

  it("covers the partially degraded state", () => {
    const state = getHomeDashboardState(
      data({ items: [item()], healthIssues: [blockingIssue()] })
    )

    expect(state.hasDisplayableHomeContent).toBe(true)
    expect(state.isHealthBlocked).toBe(false)
    expect(state.isPartiallyDegraded).toBe(true)
  })
})
