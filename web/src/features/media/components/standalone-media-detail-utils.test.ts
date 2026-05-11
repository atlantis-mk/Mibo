import { describe, expect, it } from "vitest"

import { canPlayMediaDetailItem } from "./standalone-media-detail-utils"

describe("standalone media detail playback availability", () => {
  it("allows playback when a movie has available versions even if the item is unavailable", () => {
    expect(
      canPlayMediaDetailItem(
        {
          type: "movie",
          availability_status: "unavailable",
          series_playback_target: undefined,
        },
        { status: "available" },
        undefined
      )
    ).toBe(true)
  })

  it("keeps unaired items locked even with available versions", () => {
    expect(
      canPlayMediaDetailItem(
        {
          type: "episode",
          availability_status: "unaired",
          series_playback_target: undefined,
        },
        { status: "available" },
        undefined
      )
    ).toBe(false)
  })
})
