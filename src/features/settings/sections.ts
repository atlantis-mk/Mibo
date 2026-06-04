import type { ComponentType } from 'react'
import {
  BellIcon,
  CalendarClockIcon,
  ClipboardListIcon,
  DatabaseIcon,
  BlocksIcon,
  CaptionsIcon,
  FileX2Icon,
  GlobeIcon,
  LayoutDashboardIcon,
  MonitorSmartphoneIcon,
  KeyRoundIcon,
  PaletteIcon,
  PlayCircleIcon,
  RefreshCwIcon,
  RadioIcon,
  SettingsIcon,
  ShieldCheckIcon,
  ShieldAlertIcon,
  SparklesIcon,
  ShieldEllipsisIcon,
  UserCogIcon,
  UserIcon,
  WrenchIcon,
} from 'lucide-react'
import type { User } from '@/lib/mibo-api'

type SettingsSectionPath =
  | '/settings/profile'
  | '/settings/account'
  | '/settings/appearance'
  | '/settings/notifications'
  | '/settings/display'
  | '/settings/general'
  | '/settings/operations'
  | '/settings/users'
  | '/settings/roles'
  | '/settings/devices'
  | '/settings/dlna'
  | '/settings/console'
  | '/settings/library'
  | '/settings/scan-exclusions'
  | '/settings/network'
  | '/settings/live-tv'
  | '/settings/metadata-sources'
  | '/settings/plugins'
  | '/settings/subtitles'
  | '/settings/playback'
  | '/settings/security'
  | '/settings/schedules'
  | '/settings/jobs'
  | '/settings/logs'
  | '/settings/metadata'

export type SettingsSection = {
  key: string
  group: string
  title: string
  description: string
  status: string
  to: SettingsSectionPath
  matchPrefix: string
  icon: ComponentType<{ className?: string }>
}

export function isAdminUser(user?: Pick<User, 'role'> | null) {
  return user?.role === 'admin'
}

export const SETTINGS_SECTIONS: SettingsSection[] = [
  {
    key: 'console',
    group: '总览与监控',
    icon: LayoutDashboardIcon,
    title: '控制台',
    description: '服务器状态、媒体指标、活动和维护快捷入口。',
    status: '总览',
    to: '/settings/console',
    matchPrefix: '/settings/console',
  },
  {
    key: 'operations',
    group: '总览与监控',
    icon: ShieldAlertIcon,
    title: '媒体库运营',
    description: '聚焦媒体源、扫描同步、整理流水线和人工确认。',
    status: '运营',
    to: '/settings/operations',
    matchPrefix: '/settings/operations',
  },
  {
    key: 'jobs',
    group: '总览与监控',
    icon: RefreshCwIcon,
    title: '后台任务',
    description: '查看后台队列、失败原因和重试任务。',
    status: '队列',
    to: '/settings/jobs',
    matchPrefix: '/settings/jobs',
  },
  {
    key: 'schedules',
    group: '总览与监控',
    icon: CalendarClockIcon,
    title: '计划任务',
    description: '查看维护任务状态，并手动触发后台执行。',
    status: '维护',
    to: '/settings/schedules',
    matchPrefix: '/settings/schedules',
  },
  {
    key: 'logs',
    group: '总览与监控',
    icon: ClipboardListIcon,
    title: '日志',
    description: '查看和管理服务器日志、转码日志与历史日志文件。',
    status: '维护',
    to: '/settings/logs',
    matchPrefix: '/settings/logs',
  },
  {
    key: 'profile',
    group: '用户偏好',
    icon: UserCogIcon,
    title: '个人资料',
    description: '管理当前用户的公开资料和基础信息。',
    status: '账户',
    to: '/settings/profile',
    matchPrefix: '/settings/profile',
  },
  {
    key: 'account',
    group: '用户偏好',
    icon: WrenchIcon,
    title: '账户',
    description: '设置语言、音频和字幕等个人默认偏好。',
    status: '偏好',
    to: '/settings/account',
    matchPrefix: '/settings/account',
  },
  {
    key: 'appearance',
    group: '用户偏好',
    icon: PaletteIcon,
    title: '外观',
    description: '调整主题、字体和界面显示风格。',
    status: '界面',
    to: '/settings/appearance',
    matchPrefix: '/settings/appearance',
  },
  {
    key: 'notifications',
    group: '用户偏好',
    icon: BellIcon,
    title: '通知',
    description: '控制应用提醒、通知渠道与提示方式。',
    status: '提醒',
    to: '/settings/notifications',
    matchPrefix: '/settings/notifications',
  },
  {
    key: 'display',
    group: '用户偏好',
    icon: MonitorSmartphoneIcon,
    title: '播放',
    description: '设置个人播放体验和默认行为。',
    status: '播放',
    to: '/settings/display',
    matchPrefix: '/settings/display',
  },
  {
    key: 'general',
    group: '基础与访问',
    icon: SettingsIcon,
    title: '通用配置',
    description: '管理静态资源、跨域、转码和后台执行参数。',
    status: '运行',
    to: '/settings/general',
    matchPrefix: '/settings/general',
  },
  {
    key: 'network',
    group: '基础与访问',
    icon: GlobeIcon,
    title: '网络',
    description: '配置局域网、远程访问、端口映射、TLS 和代理识别策略。',
    status: '服务器',
    to: '/settings/network',
    matchPrefix: '/settings/network',
  },
  {
    key: 'users',
    group: '基础与访问',
    icon: UserIcon,
    title: '用户',
    description: '管理服务器账号、角色、最近活动与访问权限。',
    status: '账号',
    to: '/settings/users',
    matchPrefix: '/settings/users',
  },
  {
    key: 'roles',
    group: '基础与访问',
    icon: ShieldEllipsisIcon,
    title: '角色',
    description: '管理角色定义，供用户分配和权限控制使用。',
    status: '权限',
    to: '/settings/roles',
    matchPrefix: '/settings/roles',
  },
  {
    key: 'devices',
    group: '基础与访问',
    icon: MonitorSmartphoneIcon,
    title: '设备',
    description: '查看当前账号的登录设备、最近活动和会话撤销。',
    status: '登录',
    to: '/settings/devices',
    matchPrefix: '/settings/devices',
  },
  {
    key: 'security',
    group: '基础与访问',
    icon: ShieldCheckIcon,
    title: '账号安全',
    description: '会话时长、登录保护和高危操作确认。',
    status: '策略',
    to: '/settings/security',
    matchPrefix: '/settings/security',
  },
  {
    key: 'library',
    group: '媒体库',
    icon: DatabaseIcon,
    title: '媒体管理',
    description: '媒体源、媒体库、扫描和目录挂载。',
    status: '核心',
    to: '/settings/library',
    matchPrefix: '/settings/library',
  },
  {
    key: 'scan-exclusions',
    group: '媒体库',
    icon: FileX2Icon,
    title: '扫描排除',
    description: '查看广告和误导入文件排除项，恢复或重新启用扫描过滤。',
    status: '过滤',
    to: '/settings/scan-exclusions',
    matchPrefix: '/settings/scan-exclusions',
  },
  {
    key: 'metadata-sources',
    group: '元数据',
    icon: KeyRoundIcon,
    title: '元数据策略',
    description: '管理元数据提供方实例、模板以及媒体库绑定策略。',
    status: '配置',
    to: '/settings/metadata-sources',
    matchPrefix: '/settings/metadata-sources',
  },
  {
    key: 'plugins',
    group: '元数据',
    icon: BlocksIcon,
    title: '插件中心',
    description:
      '集中管理插件实例、健康状态、引用关系、本地伴随插件和目录能力。',
    status: '扩展',
    to: '/settings/plugins',
    matchPrefix: '/settings/plugins',
  },
  {
    key: 'subtitles',
    group: '元数据',
    icon: CaptionsIcon,
    title: '字幕',
    description: '管理第三方字幕源、本地外挂字幕策略和播放时字幕搜索能力。',
    status: '字幕',
    to: '/settings/subtitles',
    matchPrefix: '/settings/subtitles',
  },
  {
    key: 'metadata',
    group: '元数据',
    icon: SparklesIcon,
    title: '元数据治理',
    description: '进入治理工作台和单条目治理流程。',
    status: '治理',
    to: '/settings/metadata',
    matchPrefix: '/settings/metadata',
  },
  {
    key: 'playback',
    group: '播放与设备',
    icon: PlayCircleIcon,
    title: '转码',
    description: '硬件加速、编码器、临时路径、字幕处理和 HDR 色调映射。',
    status: '服务器',
    to: '/settings/playback',
    matchPrefix: '/settings/playback',
  },
  {
    key: 'live-tv',
    group: '播放与设备',
    icon: RadioIcon,
    title: '直播电视',
    description: '配置电视输入源、节目指南、频道列表和录制行为。',
    status: '直播',
    to: '/settings/live-tv',
    matchPrefix: '/settings/live-tv',
  },
  {
    key: 'dlna',
    group: '播放与设备',
    icon: RadioIcon,
    title: 'DLNA',
    description: '管理 DLNA 播放、服务器发现、默认用户和设备 Profiles。',
    status: '设备',
    to: '/settings/dlna',
    matchPrefix: '/settings/dlna',
  },
]

export function getVisibleSettingsSections(user?: Pick<User, 'role'> | null) {
  if (isAdminUser(user)) {
    return SETTINGS_SECTIONS
  }

  return SETTINGS_SECTIONS.filter((section) => section.group === '用户偏好')
}

export function canAccessSettingsPath(
  pathname: string,
  user?: Pick<User, 'role'> | null
) {
  if (pathname === '/settings') {
    return true
  }

  return getVisibleSettingsSections(user).some(
    (section) =>
      pathname === section.to || pathname.startsWith(`${section.matchPrefix}/`)
  )
}
