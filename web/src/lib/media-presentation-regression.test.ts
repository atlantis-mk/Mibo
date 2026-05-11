import { describe, expect, it } from "vitest"

import {
  blocksMediaCardCatalogActions,
  formatMediaCardYearRange,
  getMediaCardOrganizingLabel,
  type MediaCardItem,
} from "#/lib/media-presentation"

function card(overrides: Partial<MediaCardItem> = {}): MediaCardItem {
  return {
    id: 1,
    library_id: 1,
    type: "movie",
    title: "Movie",
    availability_status: "available",
    governance_status: "accepted",
    source_kind: "catalog",
    ...overrides,
  }
}

describe("media presentation regression guardrails", () => {
  it("preserves review-required cards as catalog items blocked for final actions", () => {
    const reviewCard = card({
      organizing: true,
      organizing_summary: {
        state: "review_required",
        message: "Needs review",
      },
    })

    expect(reviewCard.source_kind).toBe("catalog")
    expect(getMediaCardOrganizingLabel(reviewCard)).toBe("待确认")
    expect(blocksMediaCardCatalogActions(reviewCard)).toBe(true)
  })

  it("prefers organizing message over year formatting while files are still being processed", () => {
    const processingCard = card({
      year: 2024,
      organizing: true,
      organizing_summary: {
        state: "organizing",
        stage: "materialized",
        message: "Refreshing projections",
      },
    })

    expect(formatMediaCardYearRange(processingCard)).toBe("Refreshing projections")
  })
})
