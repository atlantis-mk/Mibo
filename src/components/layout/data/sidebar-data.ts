import {
  Command,
  FilmIcon,
  FolderTreeIcon,
  Heart,
  Home,
  MonitorPlayIcon,
  Search,
  Settings,
  TvIcon,
} from 'lucide-react'
import type { User } from '@/lib/mibo-api'
import { getVisibleSettingsSections } from '@/features/settings/sections'
import { type NavGroup, type SidebarData } from '../types'

export function getSettingsNavGroups(user?: Pick<User, 'role'> | null): NavGroup[] {
  return [
    {
      title: '返回应用',
      items: [
        {
          title: '首页',
          url: '/',
          icon: Home,
        },
      ],
    },
    ...getVisibleSettingsSections(user).reduce<NavGroup[]>((groups, section) => {
      const currentGroup = groups.find((group) => group.title === section.group)
      const item = {
        title: section.title,
        url: section.to,
        icon: section.icon,
        matchPrefix: section.matchPrefix,
      }

      if (currentGroup) {
        currentGroup.items.push(item)
        return groups
      }

      groups.push({
        title: section.group,
        items: [item],
      })
      return groups
    }, []),
  ]
}

export const sidebarData: SidebarData = {
  user: {
    name: 'Mibo',
    email: '媒体中心',
    avatar: '/avatars/shadcn.jpg',
  },
  teams: [
    {
      name: 'Mibo',
      logo: Command,
      plan: '媒体中心',
    },
  ],
  navGroups: [
    {
      title: '导航',
      items: [
        {
          title: '首页',
          url: '/',
          icon: Home,
        },
        {
          title: '收藏',
          url: '/favorites',
          icon: Heart,
        },
        {
          title: '搜索',
          url: '/search',
          icon: Search,
        },
        {
          title: '设置',
          url: '/settings',
          icon: Settings,
        },
      ],
    },
    {
      title: '媒体内容',
      items: [
        {
          title: '电影',
          url: '/library?type=movie',
          icon: FilmIcon,
        },
        {
          title: '剧集',
          url: '/library?type=show',
          icon: MonitorPlayIcon,
        },
        {
          title: '目录浏览',
          url: '/library-browser',
          icon: FolderTreeIcon,
          matchPrefix: '/library-browser',
        },
        {
          title: '电视直播',
          url: '/live-tv',
          icon: TvIcon,
          matchPrefix: '/live-tv',
        },
      ],
    },
  ],
}
