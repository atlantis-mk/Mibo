import { describe, expect, it } from 'vitest'
import {
  SETTINGS_SECTIONS,
  canAccessSettingsPath,
  getVisibleSettingsSections,
} from './sections'

describe('settings sections', () => {
  it('exposes the plugin center only to administrators', () => {
    expect(
      SETTINGS_SECTIONS.some((section) => section.to === '/settings/plugins')
    ).toBe(true)
    expect(
      getVisibleSettingsSections({ role: 'admin' }).some(
        (section) => section.to === '/settings/plugins'
      )
    ).toBe(true)
    expect(
      getVisibleSettingsSections({ role: 'user' }).some(
        (section) => section.to === '/settings/plugins'
      )
    ).toBe(false)
    expect(canAccessSettingsPath('/settings/plugins', { role: 'user' })).toBe(
      false
    )
    expect(canAccessSettingsPath('/settings/plugins', { role: 'admin' })).toBe(
      true
    )
  })
})
