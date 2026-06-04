import { describe, expect, it } from 'vitest'
import { resolveActiveResourceMetadataItemId } from './resource-selection'

describe('resolveActiveResourceMetadataItemId', () => {
  it('uses the selected episode for series resource queries', () => {
    expect(
      resolveActiveResourceMetadataItemId({
        itemType: 'series',
        itemId: 10,
        selectedEpisodeMetadataItemId: 22,
        seriesPlaybackTargetEpisodeId: 11,
      })
    ).toBe(22)
  })

  it('falls back to the series playback target episode', () => {
    expect(
      resolveActiveResourceMetadataItemId({
        itemType: 'series',
        itemId: 10,
        seriesPlaybackTargetEpisodeId: 11,
      })
    ).toBe(11)
  })

  it('keeps standalone episode details scoped to the current item', () => {
    expect(
      resolveActiveResourceMetadataItemId({
        itemType: 'episode',
        itemId: 33,
        selectedEpisodeMetadataItemId: 22,
      })
    ).toBe(33)
  })
})
