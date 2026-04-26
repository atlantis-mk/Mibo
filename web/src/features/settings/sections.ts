import {
  BellIcon,
  CalendarClockIcon,
  DatabaseIcon,
  KeyRoundIcon,
  PlayCircleIcon,
  ShieldCheckIcon,
  SparklesIcon,
} from 'lucide-react'
import type { ComponentType } from 'react'

export type SettingsSectionPath =
  | '/settings/library'
  | '/settings/metadata-sources'
  | '/settings/playback'
  | '/settings/notifications'
  | '/settings/security'
  | '/settings/schedules'
  | '/settings/metadata'

export type SettingsSection = {
  key: string
  title: string
  description: string
  status: string
  to: SettingsSectionPath
  matchPrefix: string
  icon: ComponentType<{ className?: string }>
}

export const SETTINGS_SECTIONS: SettingsSection[] = [
  {
    key: 'library',
    icon: DatabaseIcon,
    title: '媒体管理',
    description: '媒体源、媒体库、扫描和目录挂载。',
    status: '核心',
    to: '/settings/library',
    matchPrefix: '/settings/library',
  },
  {
    key: 'metadata-sources',
    icon: KeyRoundIcon,
    title: '元数据源',
    description: '管理 TMDB、TVDB 等 provider 的 API Key 与连接参数。',
    status: '配置',
    to: '/settings/metadata-sources',
    matchPrefix: '/settings/metadata-sources',
  },
  {
    key: 'playback',
    icon: PlayCircleIcon,
    title: '播放体验',
    description: '默认画质、设备档案和续播策略。',
    status: '偏好',
    to: '/settings/playback',
    matchPrefix: '/settings/playback',
  },
  {
    key: 'notifications',
    icon: BellIcon,
    title: '任务通知',
    description: '任务提醒、失败提醒和通知邮箱。',
    status: '提醒',
    to: '/settings/notifications',
    matchPrefix: '/settings/notifications',
  },
  {
    key: 'security',
    icon: ShieldCheckIcon,
    title: '账号安全',
    description: '会话时长、登录保护和高危操作确认。',
    status: '策略',
    to: '/settings/security',
    matchPrefix: '/settings/security',
  },
  {
    key: 'schedules',
    icon: CalendarClockIcon,
    title: '任务工作台',
    description: '管理启停、立即运行和最近执行历史。',
    status: '工作台',
    to: '/settings/schedules',
    matchPrefix: '/settings/schedules',
  },
  {
    key: 'metadata',
    icon: SparklesIcon,
    title: '元数据治理',
    description: '进入治理工作台和单条目治理流程。',
    status: '治理',
    to: '/settings/metadata',
    matchPrefix: '/settings/metadata',
  },
]
