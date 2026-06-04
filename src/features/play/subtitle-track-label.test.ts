import { describe, expect, it } from 'vitest'
import {
  extractSubtitleDisplayTitle,
  formatSubtitleTrackMenuLabel,
} from './subtitle-track-label'

describe('extractSubtitleDisplayTitle', () => {
  it('returns the suffix after the first whitespace boundary', () => {
    expect(extractSubtitleDisplayTitle('Cht/SUP  繁体')).toBe('繁体')
    expect(extractSubtitleDisplayTitle('Chs/SUP  简体')).toBe('简体')
  })

  it('keeps the full title when no whitespace boundary exists', () => {
    expect(extractSubtitleDisplayTitle('Eng/SUP')).toBe('Eng/SUP')
  })
})

describe('formatSubtitleTrackMenuLabel', () => {
  it('prefers the parsed title over language', () => {
    expect(formatSubtitleTrackMenuLabel('Cht/SUP  繁体', 'chi', 0)).toBe('繁体')
    expect(formatSubtitleTrackMenuLabel('Eng/SUP', 'eng', 1)).toBe('Eng/SUP')
  })

  it('falls back to language and index', () => {
    expect(formatSubtitleTrackMenuLabel('', 'eng', 0)).toBe('en')
    expect(formatSubtitleTrackMenuLabel('', 'zho', 0)).toBe('zh')
    expect(formatSubtitleTrackMenuLabel('', '', 1)).toBe('字幕 2')
  })
})
