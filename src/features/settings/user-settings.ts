import type { UserSettings, UserSettingsInput } from '@/lib/mibo-api'

export const defaultUserSettings: UserSettings = {
  appearance: {
    theme: 'system',
    locale: '',
  },
  playback: {
    autoplay_next_episode: true,
    prefer_direct_play: true,
    default_subtitle_mode: 'auto',
    preferred_audio_language: '',
    preferred_subtitle_language: '',
  },
  security: {
    session_timeout: '24h',
    login_protection_level: 'standard',
    auto_clear_invalid_token: true,
    require_dangerous_action_confirmation: true,
  },
}

export const localeOptions = [
  { label: '跟随系统默认', value: '' },
  { label: '英语（美国）', value: 'en-US' },
  { label: '法语（法国）', value: 'fr-FR' },
  { label: '德语（德国）', value: 'de-DE' },
  { label: '西班牙语（西班牙）', value: 'es-ES' },
  { label: '葡萄牙语（巴西）', value: 'pt-BR' },
  { label: '俄语（俄罗斯）', value: 'ru-RU' },
  { label: '日语（日本）', value: 'ja-JP' },
  { label: '韩语（韩国）', value: 'ko-KR' },
  { label: '简体中文', value: 'zh-CN' },
] as const

export const languageOptions = [
  { label: '自动', value: '' },
  { label: '英语', value: 'en' },
  { label: '法语', value: 'fr' },
  { label: '德语', value: 'de' },
  { label: '西班牙语', value: 'es' },
  { label: '葡萄牙语', value: 'pt' },
  { label: '俄语', value: 'ru' },
  { label: '日语', value: 'ja' },
  { label: '韩语', value: 'ko' },
  { label: '中文', value: 'zh' },
] as const

export const subtitleModeOptions = [
  { label: '自动', value: 'auto' },
  { label: '始终显示', value: 'always' },
  { label: '默认不显示', value: 'never' },
] as const

export const themeOptions = [
  { label: '跟随系统', value: 'system' },
  { label: '浅色', value: 'light' },
  { label: '深色', value: 'dark' },
] as const

export function mergeUserSettings(
  current: UserSettings | undefined,
  updates: {
    appearance?: Partial<UserSettings['appearance']>
    playback?: Partial<UserSettings['playback']>
    security?: Partial<UserSettings['security']>
  }
): UserSettingsInput {
  const base = current ?? defaultUserSettings

  return {
    appearance: {
      ...base.appearance,
      ...updates.appearance,
    },
    playback: {
      ...base.playback,
      ...updates.playback,
    },
    security: {
      ...base.security,
      ...updates.security,
    },
  }
}

export function selectValue(value: string) {
  return value || '__default__'
}

export function fromSelectValue(value: string) {
  return value === '__default__' ? '' : value
}
