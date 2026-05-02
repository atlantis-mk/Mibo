import { describe, expect, it } from "vitest"

import type { HealthIssue } from "#/lib/mibo-api"

import { getHealthCenterState } from "./health-center-state"

function issue(overrides: Partial<HealthIssue>): HealthIssue {
  return {
    id: "issue-1",
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
      affected_catalog_items: 2,
      affected_files: 3,
    },
    affected: {
      media_sources: [],
      libraries: [],
      jobs: [],
    },
    actions: [],
    technical_detail: {},
    ...overrides,
  }
}

describe("getHealthCenterState", () => {
  it("covers the empty Health Center state", () => {
    const state = getHealthCenterState([], {
      validatePending: false,
      rescanPending: false,
      ignorePending: false,
    })

    expect(state.isEmpty).toBe(true)
    expect(state.activeIssueCount).toBe(0)
    expect(state.hasBlockingIssues).toBe(false)
    expect(state.hasOtherIssues).toBe(false)
  })

  it("keeps blocking issues before warning issues", () => {
    const warning = issue({ id: "warning", severity: "warning" })
    const blocking = issue({ id: "blocking", severity: "blocking" })
    const state = getHealthCenterState([warning, blocking], {
      validatePending: false,
      rescanPending: false,
      ignorePending: false,
    })

    expect(state.blockingIssues.map((entry) => entry.id)).toEqual(["blocking"])
    expect(state.otherIssues.map((entry) => entry.id)).toEqual(["warning"])
    expect(state.hasBlockingIssues).toBe(true)
    expect(state.hasOtherIssues).toBe(true)
  })

  it("reports action-loading state while recovery actions are pending", () => {
    const state = getHealthCenterState([issue({})], {
      validatePending: false,
      rescanPending: true,
      ignorePending: false,
    })

    expect(state.actionLoading).toBe(true)
  })
})
