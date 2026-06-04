import { describe, expect, it } from 'vitest'
import {
  canPlayMediaDetailItem,
  formatComputedResourceVariantLabel,
  formatSeriesPlaybackTargetLabel,
} from './standalone-media-detail-utils'

describe('formatSeriesPlaybackTargetLabel', () => {
  it('uses the backend label when present and normalizes season episode formatting', () => {
    expect(
      formatSeriesPlaybackTargetLabel({
        episode_metadata_item_id: 11,
        label: 'S1:E3',
      })
    ).toBe('S01E03')
  })

  it('falls back to the matched season rail episode when the backend label is missing', () => {
    expect(
      formatSeriesPlaybackTargetLabel(
        {
          episode_metadata_item_id: 22,
        },
        [
          {
            season_number: 2,
            name: 'Season 2',
            overview: '',
            poster_url: '',
            episodes: [
              {
                metadata_item_id: 22,
                inventory_file_id: 301,
                season_number: 2,
                episode_number: 4,
                name: 'Episode 4',
                overview: '',
                still_url: '',
                availability_status: 'available',
              },
            ],
          },
        ]
      )
    ).toBe('S02E04')
  })
})

describe('canPlayMediaDetailItem', () => {
  it('returns false when a movie has no accessible playable resource after filtering', () => {
    expect(
      canPlayMediaDetailItem(
        {
          type: 'movie',
          availability_status: 'available',
        },
        { status: 'missing' },
        { status: 'missing' }
      )
    ).toBe(false)
  })

  it('returns true for a series when an accessible playback target remains', () => {
    expect(
      canPlayMediaDetailItem({
        type: 'series',
        availability_status: 'available',
        series_playback_target: {
          episode_metadata_item_id: 10,
          title: 'Episode 1',
          label: 'S01E01',
          selection_reason: 'default',
        },
      })
    ).toBe(true)
  })
})

describe('formatComputedResourceVariantLabel', () => {
  it('uses file title differences when folder-derived work titles are identical', () => {
    const resources = [
      {
        id: 12809,
        file_name: 'hjd2048.com.061519.112.paco.1080p.mkv',
        token_title: 'hjd2048 com 061519 112 paco 1080p mkv',
      },
      {
        id: 12811,
        file_name: 'hjd2048.com.061519.112.paco.720p.mkv',
        token_title: 'hjd2048 com 061519 112 paco 720p mkv',
      },
      {
        id: 12810,
        file_name: 'hjd2048.com.061519.112.paco.5.mkv',
        token_title: 'hjd2048 com 061519 112 paco 5 mkv',
      },
    ]

    expect(formatComputedResourceVariantLabel(resources[0], resources)).toBe(
      '1080p'
    )
    expect(formatComputedResourceVariantLabel(resources[1], resources)).toBe(
      '720p'
    )
    expect(formatComputedResourceVariantLabel(resources[2], resources)).toBe(
      '5'
    )
  })
})
