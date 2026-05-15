import * as React from "react"

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from "@/components/ui/sidebar"
import { Button } from "#/components/ui/button.tsx"
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "#/components/ui/tabs.tsx"

type PlaybackSidebarItem = {
  id: number
  title: string
  label?: string
  runtime_seconds?: number
  selected_images?: { image_type: string; url: string }[]
  progress?: { watched?: boolean; played_percentage?: number }
}
type PlaybackSidebarProps = {
  currentItemId: number
  episodeItems: PlaybackSidebarItem[]
  item: {
    title: string
    overview?: string
    first_air_date?: string
    release_date?: string
    selected_images?: { image_type: string; url: string }[]
  } | null
  playbackTitle: string
  progressPercent: number
  onEpisodeSelect: (episode: PlaybackSidebarItem) => void
} & React.ComponentProps<typeof Sidebar>
function formatDateTime(value?: string) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}
function catalogImageUrl(
  item: { selected_images?: { image_type: string; url: string }[] },
  imageType: string
) {
  return item.selected_images?.find((image) => image.image_type === imageType)
    ?.url
}
function formatClock(seconds?: number) {
  if (!seconds || seconds <= 0) return "00:00"

  const total = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const remainder = total % 60

  if (hours > 0) {
    return [hours, minutes, remainder]
      .map((value) => String(value).padStart(2, "0"))
      .join(":")
  }

  return [minutes, remainder]
    .map((value) => String(value).padStart(2, "0"))
    .join(":")
}
export function AppSidebar({
  item,
  playbackTitle,
  episodeItems,
  currentItemId,
  progressPercent,
  onEpisodeSelect,
  ...props
}: PlaybackSidebarProps) {
  const listItems = episodeItems
  const hasEpisodes = listItems.length > 0
  const dateText = formatDateTime(item?.first_air_date ?? item?.release_date)
  return (
    <Sidebar className="border-white/10 bg-[#0f0f10]" {...props}>
      <SidebarContent className="bg-[#0f0f10] text-white">
        <SidebarGroup className="px-5 pt-3 pb-0 md:px-10 md:pt-6">
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  className="h-auto items-start rounded-xl px-0 py-0 text-white hover:bg-transparent hover:text-white"
                >
                  <div className="flex justify-between">
                    <div className="text-xl leading-tight font-bold tracking-[-0.04em]">
                      {item?.title || playbackTitle}
                      <br />
                      <span className={"text-base text-white/55"}>
                        {dateText || "时间未知"}
                      </span>
                    </div>
                    <div className="mt-7 flex items-center justify-between gap-5 text-[20px] font-semibold text-white/55">
                      <Button>
                        打开文件位置
                      </Button>
                    </div>
                  </div>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroup className="mt-9 px-0 pb-0">
          <SidebarGroupContent>
            <Tabs defaultValue={hasEpisodes ? "overview" : "analytics"}>
              <TabsList>
                {hasEpisodes ? (
                  <TabsTrigger value="overview">选集</TabsTrigger>
                ) : null}
                <TabsTrigger value="analytics">章节</TabsTrigger>
                <TabsTrigger value="reports">信息</TabsTrigger>
              </TabsList>
              {hasEpisodes ? (
                <TabsContent value="overview">
                  {listItems.map((episode) => {
                    const imageUrl =
                      catalogImageUrl(episode, "backdrop") ||
                      catalogImageUrl(episode, "poster") ||
                      catalogImageUrl(item ?? {}, "backdrop")
                    const isCurrent = episode.id === currentItemId
                    const watchedText = episode.progress?.watched
                      ? "已观看"
                      : episode.progress?.played_percentage
                        ? `已观看 ${Math.round(episode.progress.played_percentage)}%`
                        : "未观看"

                    return (
                      <SidebarMenuSubItem key={episode.id}>
                        <SidebarMenuSubButton
                          asChild
                          isActive={isCurrent}
                          className="h-auto rounded-none px-0 py-0 text-white hover:bg-transparent hover:text-white data-active:bg-[#18191c] data-active:text-white"
                        >
                          <button
                            type="button"
                            onClick={() => {
                              if (!isCurrent) {
                                onEpisodeSelect(episode)
                              }
                            }}
                            className="group/episode grid w-full grid-cols-[7.75rem_1fr] gap-6 px-5 py-3 text-left md:px-6"
                          >
                            <div className="relative aspect-video overflow-hidden rounded-md bg-white/10">
                              {imageUrl ? (
                                <img
                                  src={imageUrl}
                                  alt=""
                                  className="h-full w-full object-cover transition-transform duration-300 group-hover/episode:scale-105"
                                />
                              ) : null}
                              {episode.runtime_seconds ? (
                                <div className="absolute right-2 bottom-2 text-[15px] font-bold text-white drop-shadow">
                                  {formatClock(episode.runtime_seconds)}
                                </div>
                              ) : null}
                            </div>
                            <div className="min-w-0 pt-1">
                              <div
                                className={`line-clamp-2 text-[20px] leading-snug font-medium tracking-[-0.04em] ${isCurrent ? "text-[#1768ff]" : "text-white/85"}`}
                              >
                                {isCurrent ? <PlayingGlyph /> : null}
                                {episode.label || episode.title}
                              </div>
                              <div className="mt-6 text-[18px] font-semibold text-white/45">
                                {watchedText}
                              </div>
                            </div>
                          </button>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    )
                  })}
                </TabsContent>
              ) : null}
            </Tabs>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}

function PlayingGlyph() {
  return (
    <span className="mr-3 inline-flex h-5 translate-y-0.5 items-center gap-1 align-baseline text-[#1768ff]">
      <span className="h-4 w-1 rounded-full bg-current" />
      <span className="h-5 w-1 rounded-full bg-current" />
      <span className="h-3 w-1 rounded-full bg-current" />
    </span>
  )
}
