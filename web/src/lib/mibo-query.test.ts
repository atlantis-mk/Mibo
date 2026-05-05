import { describe, expect, it } from "vitest"

import type { CatalogListItem, CatalogUserItemEntry } from "#/lib/mibo-api"

import { getLatestContinueWatchingEntries } from "./mibo-query"

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

function entry(
  overrides: Partial<CatalogUserItemEntry> = {}
): CatalogUserItemEntry {
  return {
    user_id: 1,
    position_seconds: 120,
    watched: false,
    favorite: false,
    last_played_at: "2026-01-01T00:00:00Z",
    item: item({ id: 10 }),
    ...overrides,
  }
}

describe("getLatestContinueWatchingEntries", () => {
  it("keeps only the latest episode progress per displayed series", () => {
    const series = item({ id: 100, type: "show", title: "Show" })
    const oldEpisode = entry({
      item: item({ id: 101, type: "episode", title: "Episode 1" }),
      display_item: series,
      play_item: item({ id: 101, type: "episode", title: "Episode 1" }),
      last_played_at: "2026-01-01T00:00:00Z",
    })
    const latestEpisode = entry({
      item: item({ id: 102, type: "episode", title: "Episode 2" }),
      display_item: series,
      play_item: item({ id: 102, type: "episode", title: "Episode 2" }),
      last_played_at: "2026-01-02T00:00:00Z",
    })
    const movie = entry({
      item: item({ id: 200, type: "movie", title: "Movie" }),
      last_played_at: "2026-01-03T00:00:00Z",
    })

    const result = getLatestContinueWatchingEntries([
      oldEpisode,
      movie,
      latestEpisode,
    ])

    expect(result).toEqual([movie, latestEpisode])
  })
})
