import { useQuery } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  HeartIcon,
  LoaderCircleIcon,
  SearchIcon,
  Settings2Icon,
} from "lucide-react"

import { AppTopBar } from "#/components/app-top-bar"
import { MediaPosterCard } from "#/components/media-poster-card"
import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import { SidebarTrigger } from "#/components/ui/sidebar"
import { favoritesQueryOptions } from "#/lib/mibo-query"
import { useAuthStore } from "#/stores/auth-store"

export default function FavoritesPage() {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const navigate = useNavigate()
  const queryToken = token ?? "guest"

  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })
  if (!hasHydrated || (token && favoritesQuery.isLoading)) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background text-foreground">
        <LoaderCircleIcon className="size-4 animate-spin" />
      </div>
    )
  }

  if (!token || !user) {
    void navigate({
      to: "/login",
      search: { redirect: "/favorites" },
      replace: true,
    })
    return null
  }

  const favorites = favoritesQuery.data ?? []

  return (
    <div className="relative min-w-0 flex-1 bg-background text-foreground">
      <AppTopBar
        leftSlot={
          <>
            <SidebarTrigger className="rounded-full border border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground" />
            <div className="hidden rounded-full border border-border/50 bg-background/80 p-1 sm:flex">
              <Button
                asChild
                size="sm"
                variant="ghost"
                className="h-8 rounded-full px-4 text-muted-foreground"
              >
                <Link to="/">首页</Link>
              </Button>
              <Button asChild size="sm" className="h-8 rounded-full px-4">
                <Link to="/favorites">收藏</Link>
              </Button>
            </div>
            <div className="min-w-0">
              <div className="truncate text-sm font-medium">收藏</div>
              <div className="truncate text-xs text-muted-foreground">
                {favorites.length} 个收藏项目
              </div>
            </div>
          </>
        }
        rightSlot={
          <div className="hidden items-center gap-2 sm:flex">
            <Button
              asChild
              size="icon-sm"
              variant="outline"
              className="rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
            >
              <Link to="/search" search={{ q: undefined }}>
                <SearchIcon className="size-4" />
                <span className="sr-only">搜索</span>
              </Link>
            </Button>
            <Button
              asChild
              size="icon-sm"
              variant="outline"
              className="rounded-full border-border/50 bg-background/80 text-foreground hover:bg-accent hover:text-accent-foreground"
            >
              <Link to="/settings">
                <Settings2Icon className="size-4" />
                <span className="sr-only">设置</span>
              </Link>
            </Button>
          </div>
        }
      />

      <section className="px-4 pt-24 pb-16 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-[1600px]">
          <div className="mb-8">
            <Badge
              className="border-border/60 bg-background/80"
              variant="outline"
            >
              Favorites
            </Badge>
            <h1 className="mt-4 text-4xl font-semibold tracking-tight">
              我的收藏
            </h1>
          </div>
          {favorites.length > 0 ? (
            <div className="grid gap-5 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6">
              {favorites.map((entry) => (
                <MediaPosterCard
                  key={entry.item.id}
                  item={entry.item}
                  progress={entry}
                  layout="grid"
                />
              ))}
            </div>
          ) : (
            <div className="rounded-[2rem] border border-border/40 bg-card/70 px-6 py-10 text-center text-sm text-muted-foreground backdrop-blur-sm">
              <HeartIcon className="mx-auto mb-4 size-8 text-muted-foreground" />
              还没有收藏。打开媒体详情或首页卡片，将喜欢的作品加入收藏。
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
