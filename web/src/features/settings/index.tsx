import { Link, Outlet } from '@tanstack/react-router'
import { ArrowLeftIcon } from 'lucide-react'

import { AppTopBar } from '#/components/app-top-bar'
import { Button } from '#/components/ui/button'

import { SettingsSectionNav } from './components/settings-section-nav'
import { SETTINGS_SECTIONS } from './sections'

export default function SettingsLayout() {
  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      <AppTopBar
        leftSlot={
          <>
            <Button
              asChild
              variant="ghost"
              size="icon-sm"
              className="size-9 rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            >
              <Link to="/">
                <ArrowLeftIcon className="size-4.5" />
                <span className="sr-only">返回首页</span>
              </Link>
            </Button>
            <div className="min-w-0">
              <div className="truncate text-sm font-medium">设置</div>
              <div className="truncate text-xs text-muted-foreground">
                媒体源、媒体库与系统偏好
              </div>
            </div>
          </>
        }
      />

      <div className="px-4 pb-10 pt-24 sm:px-6 lg:px-8 xl:px-10">
        <div className="mx-auto max-w-7xl space-y-5">
          <div className="grid gap-5 lg:grid-cols-[280px_minmax(0,1fr)] lg:items-start">
            <aside className="lg:sticky lg:top-24">
              <SettingsSectionNav sections={SETTINGS_SECTIONS} />
            </aside>

            <main className="min-w-0">
              <Outlet />
            </main>
          </div>
        </div>
      </div>
    </div>
  )
}
