import { Link, Outlet } from "@tanstack/react-router"
import {
  ArrowLeftIcon,
  CastIcon,
  CircleUserRoundIcon,
  SettingsIcon,
} from "lucide-react"

import { AppTopBar } from "#/components/app-top-bar"
import { Button } from "#/components/ui/button"

import { SettingsSectionNav } from "./components/settings-section-nav"
import { SETTINGS_SECTIONS } from "./sections"

export default function SettingsLayout() {
  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      <AppTopBar
        leftSlot={
          <>
            <Button asChild variant="ghost" size="icon-sm">
              <Link to="/">
                <ArrowLeftIcon className="size-4" />
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
        rightSlot={
          <>
            <Button variant="ghost" size="icon-sm">
              <CastIcon className="size-4" />
              <span className="sr-only">投屏</span>
            </Button>
            <Button variant="ghost" size="icon-sm">
              <CircleUserRoundIcon className="size-4" />
              <span className="sr-only">用户</span>
            </Button>
            <Button asChild variant="ghost" size="icon-sm">
              <Link to="/settings">
                <SettingsIcon className="size-4" />
                <span className="sr-only">系统设置</span>
              </Link>
            </Button>
          </>
        }
      />

      <div className="px-4 pt-24 pb-10 sm:px-6 lg:px-8 xl:px-10">
        <div className="space-y-5">
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
