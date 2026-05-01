import {
  BellIcon,
  CalendarClockIcon,
  ClipboardListIcon,
  DatabaseIcon,
  EraserIcon,
  FileX2Icon,
  GlobeIcon,
  LayoutDashboardIcon,
  MonitorSmartphoneIcon,
  KeyRoundIcon,
  PlayCircleIcon,
  RefreshCwIcon,
  RadioIcon,
  ShieldCheckIcon,
  ShieldAlertIcon,
  SparklesIcon,
  UserIcon,
} from "lucide-react"
import type { ComponentType } from "react"

export type SettingsSectionPath =
  | "/settings/general"
  | "/settings/health"
  | "/settings/users"
  | "/settings/devices"
  | "/settings/dlna"
  | "/settings/console"
  | "/settings/library"
  | "/settings/scan-exclusions"
  | "/settings/cleanup"
  | "/settings/database"
  | "/settings/network"
  | "/settings/live-tv"
  | "/settings/metadata-sources"
  | "/settings/playback"
  | "/settings/notifications"
  | "/settings/security"
  | "/settings/schedules"
  | "/settings/jobs"
  | "/settings/logs"
  | "/settings/metadata"

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
    key: "console",
    icon: LayoutDashboardIcon,
    title: "控制台",
    description: "服务器状态、媒体指标、活动和维护快捷入口。",
    status: "总览",
    to: "/settings/console",
    matchPrefix: "/settings/console",
  },
  {
    key: "health",
    icon: ShieldAlertIcon,
    title: "健康中心",
    description: "查看系统问题、影响范围、恢复引导和技术详情。",
    status: "诊断",
    to: "/settings/health",
    matchPrefix: "/settings/health",
  },
  {
    key: "users",
    icon: UserIcon,
    title: "用户",
    description: "管理服务器账号、角色、最近活动与访问权限。",
    status: "账号",
    to: "/settings/users",
    matchPrefix: "/settings/users",
  },
  {
    key: "devices",
    icon: MonitorSmartphoneIcon,
    title: "设备",
    description: "查看当前账号的登录设备、最近活动和会话撤销。",
    status: "登录",
    to: "/settings/devices",
    matchPrefix: "/settings/devices",
  },
  {
    key: "dlna",
    icon: RadioIcon,
    title: "DLNA",
    description: "管理 DLNA 播放、服务器发现、默认用户和设备 Profiles。",
    status: "设备",
    to: "/settings/dlna",
    matchPrefix: "/settings/dlna",
  },
  {
    key: "general",
    icon: GlobeIcon,
    title: "通用",
    description: "服务器基础行为、界面语言、维护模式和全局 Web 外观。",
    status: "基础",
    to: "/settings/general",
    matchPrefix: "/settings/general",
  },
  {
    key: "library",
    icon: DatabaseIcon,
    title: "媒体管理",
    description: "媒体源、媒体库、扫描和目录挂载。",
    status: "核心",
    to: "/settings/library",
    matchPrefix: "/settings/library",
  },
  {
    key: "scan-exclusions",
    icon: FileX2Icon,
    title: "扫描排除",
    description: "查看广告和误导入文件排除项，恢复或重新启用扫描过滤。",
    status: "过滤",
    to: "/settings/scan-exclusions",
    matchPrefix: "/settings/scan-exclusions",
  },
  {
    key: "cleanup",
    icon: EraserIcon,
    title: "清理",
    description: "配置缺失媒体硬删除策略，并主动触发缺失媒体清理任务。",
    status: "危险",
    to: "/settings/cleanup",
    matchPrefix: "/settings/cleanup",
  },
  {
    key: "network",
    icon: GlobeIcon,
    title: "网络",
    description: "配置局域网、远程访问、端口映射、TLS 和代理识别策略。",
    status: "服务器",
    to: "/settings/network",
    matchPrefix: "/settings/network",
  },
  {
    key: "database",
    icon: DatabaseIcon,
    title: "数据库",
    description: "调整缓存尺寸、关闭时优化和下次启动数据库清理行为。",
    status: "高级",
    to: "/settings/database",
    matchPrefix: "/settings/database",
  },
  {
    key: "metadata-sources",
    icon: KeyRoundIcon,
    title: "元数据策略",
    description: "管理 metadata provider instances、profiles 和库绑定策略。",
    status: "配置",
    to: "/settings/metadata-sources",
    matchPrefix: "/settings/metadata-sources",
  },
  {
    key: "playback",
    icon: PlayCircleIcon,
    title: "转码",
    description: "硬件加速、编码器、临时路径、字幕处理和 HDR 色调映射。",
    status: "服务器",
    to: "/settings/playback",
    matchPrefix: "/settings/playback",
  },
  {
    key: "live-tv",
    icon: RadioIcon,
    title: "直播电视",
    description: "配置电视输入源、节目指南、频道列表和录制行为。",
    status: "直播",
    to: "/settings/live-tv",
    matchPrefix: "/settings/live-tv",
  },
  {
    key: "notifications",
    icon: BellIcon,
    title: "任务通知",
    description: "任务提醒、失败提醒和通知邮箱。",
    status: "提醒",
    to: "/settings/notifications",
    matchPrefix: "/settings/notifications",
  },
  {
    key: "security",
    icon: ShieldCheckIcon,
    title: "账号安全",
    description: "会话时长、登录保护和高危操作确认。",
    status: "策略",
    to: "/settings/security",
    matchPrefix: "/settings/security",
  },
  {
    key: "schedules",
    icon: CalendarClockIcon,
    title: "计划任务",
    description: "查看维护任务状态，并手动触发后台执行。",
    status: "维护",
    to: "/settings/schedules",
    matchPrefix: "/settings/schedules",
  },
  {
    key: "jobs",
    icon: RefreshCwIcon,
    title: "后台任务",
    description: "查看后台队列、失败原因和重试任务。",
    status: "队列",
    to: "/settings/jobs",
    matchPrefix: "/settings/jobs",
  },
  {
    key: "logs",
    icon: ClipboardListIcon,
    title: "日志",
    description: "查看和管理服务器日志、转码日志与历史日志文件。",
    status: "维护",
    to: "/settings/logs",
    matchPrefix: "/settings/logs",
  },
  {
    key: "metadata",
    icon: SparklesIcon,
    title: "元数据治理",
    description: "进入治理工作台和单条目治理流程。",
    status: "治理",
    to: "/settings/metadata",
    matchPrefix: "/settings/metadata",
  },
]
