import {
  BellIcon,
  CalendarClockIcon,
  ClipboardListIcon,
  DatabaseIcon,
  GlobeIcon,
  LayoutDashboardIcon,
  MonitorSmartphoneIcon,
  KeyRoundIcon,
  PlayCircleIcon,
  RadioIcon,
  ShieldCheckIcon,
  SparklesIcon,
  UserIcon,
} from 'lucide-react'
import type { ComponentType } from 'react'

export type SettingsSectionPath =
  | '/settings/general'
  | '/settings/users'
  | '/settings/devices'
  | '/settings/dlna'
  | '/settings/console'
  | '/settings/library'
  | '/settings/database'
  | '/settings/network'
  | '/settings/live-tv'
  | '/settings/metadata-sources'
  | '/settings/playback'
  | '/settings/notifications'
  | '/settings/security'
  | '/settings/schedules'
  | '/settings/logs'
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
    key: 'users',
    icon: UserIcon,
    title: '用户',
    description: '管理服务器账号、角色、最近活动与访问权限。',
    status: '账号',
    to: '/settings/users',
    matchPrefix: '/settings/users',
  },
  {
    key: 'devices',
    icon: MonitorSmartphoneIcon,
    title: '设备',
    description: '查看连接过服务器的客户端、最近活动和使用用户。',
    status: '客户端',
    to: '/settings/devices',
    matchPrefix: '/settings/devices',
  },
  {
    key: 'dlna',
    icon: RadioIcon,
    title: 'DLNA',
    description: '管理 DLNA 播放、服务器发现、默认用户和设备 Profiles。',
    status: '设备',
    to: '/settings/dlna',
    matchPrefix: '/settings/dlna',
  },
  {
    key: 'general',
    icon: GlobeIcon,
    title: '通用',
    description: '服务器基础行为、界面语言、维护模式和全局 Web 外观。',
    status: '基础',
    to: '/settings/general',
    matchPrefix: '/settings/general',
  },
  {
    key: 'console',
    icon: LayoutDashboardIcon,
    title: '控制台',
    description: '服务器状态、媒体指标、活动和维护快捷入口。',
    status: '总览',
    to: '/settings/console',
    matchPrefix: '/settings/console',
  },
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
    key: 'network',
    icon: GlobeIcon,
    title: '网络',
    description: '配置局域网、远程访问、端口映射、TLS 和代理识别策略。',
    status: '服务器',
    to: '/settings/network',
    matchPrefix: '/settings/network',
  },
  {
    key: 'database',
    icon: DatabaseIcon,
    title: '数据库',
    description: '调整缓存尺寸、关闭时优化和下次启动数据库清理行为。',
    status: '高级',
    to: '/settings/database',
    matchPrefix: '/settings/database',
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
    title: '转码',
    description: '硬件加速、编码器、临时路径、字幕处理和 HDR 色调映射。',
    status: '服务器',
    to: '/settings/playback',
    matchPrefix: '/settings/playback',
  },
  {
    key: 'live-tv',
    icon: RadioIcon,
    title: '直播电视',
    description: '配置电视输入源、节目指南、频道列表和录制行为。',
    status: '直播',
    to: '/settings/live-tv',
    matchPrefix: '/settings/live-tv',
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
    title: '计划任务',
    description: '查看维护任务状态，并手动触发后台执行。',
    status: '维护',
    to: '/settings/schedules',
    matchPrefix: '/settings/schedules',
  },
  {
    key: 'logs',
    icon: ClipboardListIcon,
    title: '日志',
    description: '查看和管理服务器日志、转码日志与历史日志文件。',
    status: '维护',
    to: '/settings/logs',
    matchPrefix: '/settings/logs',
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
