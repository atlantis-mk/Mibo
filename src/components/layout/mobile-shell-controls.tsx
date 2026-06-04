import { Link, useLocation, useNavigate } from '@tanstack/react-router'
import { ChevronLeft, Home } from 'lucide-react'
import type { User } from '@/lib/mibo-api'
import { useAuthStore } from '@/stores/auth-store'
import { ConfigDrawer } from '@/components/config-drawer'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { getSettingsNavGroups, sidebarData } from './data/sidebar-data'

type MobileShellMeta = {
  eyebrow: string
  title: string
  description?: string
}

const defaultMeta: MobileShellMeta = {
  eyebrow: 'Mibo',
  title: '媒体中心',
  description: '浏览内容与系统状态',
}

export function MobileShellControls() {
  const navigate = useNavigate()
  const href = useLocation({ select: (location) => location.href })
  const pathname = useLocation({ select: (location) => location.pathname })
  const authUser = useAuthStore((state) => state.auth.user)
  const meta = getMobileShellMeta(href, pathname, authUser)
  const showBackButton = pathname.startsWith('/media/')
  const showHomeButton = pathname !== '/' && !showBackButton
  const shouldReserveSpace = pathname !== '/'

  const handleBack = () => {
    if (typeof window !== 'undefined' && window.history.length > 1) {
      window.history.back()
      return
    }

    void navigate({ to: '/' })
  }

  return (
    <>
      <div className='fixed inset-x-0 top-0 z-40 border-b border-border/60 bg-background/82 supports-[backdrop-filter]:bg-background/72 backdrop-blur-xl md:hidden'>
        <div className='px-4 pt-[max(env(safe-area-inset-top),0.75rem)] pb-3'>
          <div className='flex items-center gap-3'>
            <SidebarTrigger className='size-10 shrink-0 rounded-2xl border border-border/70 bg-background/90 shadow-sm backdrop-blur transition-colors hover:bg-accent/70' />
            {showBackButton ? (
              <button
                type='button'
                aria-label='返回上一页'
                className='inline-flex size-10 shrink-0 items-center justify-center rounded-2xl border border-border/70 bg-background/90 shadow-sm backdrop-blur transition-colors hover:bg-accent/70'
                onClick={handleBack}
              >
                <ChevronLeft className='size-4' aria-hidden='true' />
              </button>
            ) : null}
            {showHomeButton ? (
              <Link
                to='/'
                aria-label='返回首页'
                className='inline-flex size-10 shrink-0 items-center justify-center rounded-2xl border border-border/70 bg-background/90 shadow-sm backdrop-blur transition-colors hover:bg-accent/70'
              >
                <Home className='size-4' aria-hidden='true' />
              </Link>
            ) : null}
            <div className='min-w-0 flex-1'>
              <p className='truncate text-[0.65rem] font-semibold tracking-[0.28em] text-muted-foreground uppercase'>
                {meta.eyebrow}
              </p>
              <div className='min-w-0'>
                <h1 className='truncate text-base font-semibold text-foreground'>
                  {meta.title}
                </h1>
                {meta.description ? (
                  <p className='truncate text-xs text-muted-foreground'>
                    {meta.description}
                  </p>
                ) : null}
              </div>
            </div>
            <div className='shrink-0 rounded-2xl border border-border/70 bg-background/90 shadow-sm backdrop-blur'>
              <ConfigDrawer />
            </div>
          </div>
        </div>
      </div>
      {shouldReserveSpace ? (
        <div className='h-[calc(max(env(safe-area-inset-top),0.75rem)+3.25rem)] md:hidden' />
      ) : null}
    </>
  )
}

function getMobileShellMeta(
  href: string,
  pathname: string,
  user?: Pick<User, 'role'> | null
): MobileShellMeta {
  if (pathname === '/') {
    return {
      eyebrow: 'Mibo',
      title: '首页',
    }
  }

  if (pathname === '/library') {
    const searchParams = getSearchParams(href)
    const libraryType = searchParams.get('type')

    if (libraryType === 'movie') {
      return {
        eyebrow: '媒体内容',
        title: '电影',
        description: '浏览电影收藏与近期更新',
      }
    }

    if (libraryType === 'show') {
      return {
        eyebrow: '媒体内容',
        title: '剧集',
        description: '继续追剧并查看最新入库',
      }
    }

    return {
      eyebrow: '媒体内容',
      title: '媒体库',
      description: '筛选、整理并发现更多内容',
    }
  }

  if (pathname === '/library-browser') {
    return {
      eyebrow: '媒体内容',
      title: '目录浏览',
      description: '按媒体库与文件夹层级浏览内容',
    }
  }

  const navMatch = [...sidebarData.navGroups, ...getSettingsNavGroups(user)]
    .flatMap((group) => group.items.map((item) => ({ group: group.title, item })))
    .find(({ item }) =>
      ('url' in item && pathname === item.url) ||
      (!!item.matchPrefix && pathname.startsWith(item.matchPrefix))
    )

  if (navMatch) {
    return {
      eyebrow: navMatch.group,
      title: navMatch.item.title,
      description: getDescriptionForPath(pathname),
    }
  }

  return defaultMeta
}

function getDescriptionForPath(pathname: string) {
  if (pathname.startsWith('/settings')) {
    return '查看系统状态并调整偏好配置'
  }

  if (pathname.startsWith('/search')) {
    return '快速查找影片、剧集与人物'
  }

  if (pathname.startsWith('/favorites')) {
    return '集中查看你标记的重要内容'
  }

  if (pathname.startsWith('/play')) {
    return '当前正在播放的媒体内容'
  }

  return defaultMeta.description
}

function getSearchParams(href: string) {
  try {
    return new URL(href, window.location.origin).searchParams
  } catch {
    return new URLSearchParams()
  }
}
