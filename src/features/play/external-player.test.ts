import { describe, expect, it } from 'vitest'
import {
  convertExternalPlayerUrl,
  getRecommendedExternalPlayer,
  normalizePlaybackLaunchPreferences,
  resolveAbsolutePlaybackUrl,
} from './external-player'

describe('normalizePlaybackLaunchPreferences', () => {
  it('falls back to safe defaults for invalid values', () => {
    expect(
      normalizePlaybackLaunchPreferences({
        mode: 'invalid' as never,
        externalPlayerId: 'invalid' as never,
      })
    ).toEqual({
      mode: 'internal',
      externalPlayerId: getRecommendedExternalPlayer(),
    })
  })
})

describe('convertExternalPlayerUrl', () => {
  it('replaces encoded placeholders', () => {
    expect(
      convertExternalPlayerUrl('mpv://$edurl', {
        url: 'https://example.com/video?id=1',
        name: 'Example',
      })
    ).toBe('mpv://https%3A%2F%2Fexample.com%2Fvideo%3Fid%3D1')
  })

  it('replaces base64 placeholders', () => {
    expect(
      convertExternalPlayerUrl('iplay://play?url=$bdurl', {
        url: 'https://example.com/video',
        name: 'Example',
      })
    ).toBe(`iplay://play?url=${window.btoa('https://example.com/video')}`)
  })
})

describe('resolveAbsolutePlaybackUrl', () => {
  it('resolves api playback urls against the real api base url', () => {
    expect(resolveAbsolutePlaybackUrl('/api/v1/access/inventory-files/1')).toBe(
      `${window.location.origin}/api/v1/access/inventory-files/1`
    )
  })
})
