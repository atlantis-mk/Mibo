import { useQuery } from "@tanstack/react-query"
import { Link, useLocation } from "@tanstack/react-router"
import { DatabaseIcon, HeartIcon, HomeIcon, SearchIcon, SettingsIcon, SparklesIcon } from "lucide-react"

import { SearchForm } from "#/components/search-form"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "#/components/ui/sidebar"
import {
  healthSummaryQueryOptions,
  librariesQueryOptions,
} from "#/lib/mibo-query"
import { useAuthStore } from "#/stores/auth-store"

const primaryNav = [
  { title: "首页", to: "/", icon: HomeIcon },
  { title: "收藏", to: "/favorites", icon: HeartIcon },
  { title: "搜索", to: "/search", icon: SearchIcon },
  { title: "设置", to: "/settings", icon: SettingsIcon },
] as const

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const token = useAuthStore((state) => state.token)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const location = useLocation()
  const queryToken = token ?? "guest"
  const librariesQuery = useQuery({
    ...librariesQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const healthQuery = useQuery({
    ...healthSummaryQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  const libraries = librariesQuery.data ?? []
  const issues = healthQuery.data?.issues ?? []
  const hasGlobalHealthIssue =
    (healthQuery.data?.blocking_count ?? 0) +
      (healthQuery.data?.error_count ?? 0) >
    0
  const affectedLibraryIds = new Set(
    issues.flatMap((issue) =>
      issue.severity === "blocking" || issue.severity === "error"
        ? (issue.affected?.libraries ?? []).map((library) => library.id)
        : []
    )
  )

  return (
    <Sidebar {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" asChild>
              <Link to="/">
                <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                  <SparklesIcon className="size-4" />
                </div>
                <div className="flex flex-col gap-0.5 leading-none">
                  <span className="font-medium">Mibo</span>
                  <span className="text-xs text-sidebar-foreground/70">
                    媒体中心
                  </span>
                </div>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
        <SearchForm />
      </SidebarHeader>
      <SidebarContent className="gap-2">
        <SidebarGroup>
          <SidebarGroupLabel>导航</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {primaryNav.map((item) => {
                const Icon = item.icon
                return (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton
                      asChild
                      isActive={location.pathname === item.to}
                    >
                      <Link to={item.to}>
                        <Icon className="size-4" />
                        {item.title}
                        {item.to === "/settings" && hasGlobalHealthIssue ? (
                          <span className="ml-auto size-2 rounded-full bg-destructive" />
                        ) : null}
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>媒体内容</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={location.pathname === "/library"}>
                  <Link to="/library">
                    <DatabaseIcon className="size-4" />
                    内容库
                    {Array.from(affectedLibraryIds).length > 0 ? (
                      <span className="ml-auto shrink-0 rounded-full bg-destructive/15 px-1.5 py-0.5 text-[10px] font-medium text-destructive">
                        处理
                      </span>
                    ) : null}
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              {libraries.length === 0 ? (
                <SidebarMenuItem>
                  <SidebarMenuButton asChild>
                    <Link to="/settings/library">
                      <DatabaseIcon className="size-4" />
                      添加来源
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ) : null}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}
