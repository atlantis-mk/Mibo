import { describe, expect, it } from "vitest"

import {
  blocksMediaCardCatalogActions,
  formatMediaCardYearRange,
  getMediaCardOrganizingLabel,
  type MediaCardItem,
} from "#/lib/media-presentation"

function item(overrides: Partial<MediaCardItem> = {}): MediaCardItem {
  return {
    id: 1,
    library_id: 1,
    type: "movie",
    title: "Movie",
    availability_status: "available",
    governance_status: "pending",
    source_kind: "catalog",
    ...overrides,
  }
}

describe("media card organizing presentation", () => {
  it("uses organizing summary copy before generic year text", () => {
    const card = item({
      organizing: true,
      organizing_summary: {
        state: "organizing",
        stage: "probed",
        message: "Analyzing video streams",
      },
    })

    expect(formatMediaCardYearRange(card)).toBe("Analyzing video streams")
    expect(getMediaCardOrganizingLabel(card)).toBe("整理中")
  })

  it("labels review and failed states", () => {
    expect(
      getMediaCardOrganizingLabel(
        item({
          organizing_summary: {
            state: "review_required",
            message: "Review needed",
          },
        })
      )
    ).toBe("待确认")
    expect(
      getMediaCardOrganizingLabel(
        item({
          organizing_summary: { state: "failed", message: "Probe failed" },
        })
      )
    ).toBe("整理失败")
  })

  it("blocks final catalog actions for inventory-only and organizing cards", () => {
    expect(
      blocksMediaCardCatalogActions(item({ source_kind: "inventory_file" }))
    ).toBe(true)
    expect(blocksMediaCardCatalogActions(item({ organizing: true }))).toBe(true)
    expect(
      blocksMediaCardCatalogActions(
        item({ source_kind: "catalog", organizing: false })
      )
    ).toBe(false)
  })

  it("keeps review-required catalog items distinguishable from inventory-only cards", () => {
    const reviewRequired = item({
      organizing: true,
      organizing_summary: {
        state: "review_required",
        message: "需要人工确认",
      },
    })

    expect(reviewRequired.source_kind).toBe("catalog")
    expect(getMediaCardOrganizingLabel(reviewRequired)).toBe("待确认")
    expect(blocksMediaCardCatalogActions(reviewRequired)).toBe(true)
  })
})
