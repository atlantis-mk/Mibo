import * as React from 'react'
import { useIsMobile } from '@/hooks/use-mobile'
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
} from '@/components/ui/sidebar'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  DescriptionGrid,
  DescriptionItem,
} from '@/features/settings/components/settings-aside-card'

type PlaybackSidebarItem = {
  id: number
  sourceId?: number
  title: string
  label?: string
  runtime_seconds?: number
  selected_images?: { image_type: string; url: string }[]
  progress?: { watched?: boolean; played_percentage?: number }
}

type PlaybackSidebarProps = {
  currentOverviewItemId: number
  overviewItems: PlaybackSidebarItem[]
  overviewTabLabel: string
  playbackFacts: Array<{ label: string; value: string }>
  item: {
    title: string
    overview?: string
    first_air_date?: string
    release_date?: string
    cast?: { id?: number; name: string; role?: string; avatar_url?: string }[]
    directors?: {
      id?: number
      name: string
      role?: string
      avatar_url?: string
    }[]
    selected_images?: { image_type: string; url: string }[]
  } | null
  playbackTitle: string
  progressPercent: number
  onOverviewItemSelect: (item: PlaybackSidebarItem) => void
  inlineOnMobile?: boolean
} & React.ComponentProps<typeof Sidebar>

function formatDateTime(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
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
  if (!seconds || seconds <= 0) return '00:00'

  const total = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const remainder = total % 60

  if (hours > 0) {
    return [hours, minutes, remainder]
      .map((value) => String(value).padStart(2, '0'))
      .join(':')
  }

  return [minutes, remainder]
    .map((value) => String(value).padStart(2, '0'))
    .join(':')
}

export function AppSidebar({
  item,
  playbackTitle,
  overviewItems,
  currentOverviewItemId,
  overviewTabLabel,
  playbackFacts,
  progressPercent,
  onOverviewItemSelect,
  inlineOnMobile = false,
  className,
  ...props
}: PlaybackSidebarProps) {
  const isMobile = useIsMobile()
  const listItems = overviewItems
  const hasOverviewItems = listItems.length > 0
  const dateText = formatDateTime(item?.first_air_date ?? item?.release_date)
  const infoText = item?.overview?.trim() || '暂无媒体介绍'
  const directorNames = uniquePeople(item?.directors).slice(0, 4)
  const castPeople = uniquePeople(item?.cast).slice(0, 8)
  const renderInline = inlineOnMobile && isMobile
  const sidebarClassName = renderInline
    ? `w-full border-t border-border bg-background ${className ?? ''}`.trim()
    : className

  return (
    <Sidebar
      collapsible={renderInline ? 'none' : 'offcanvas'}
      className={sidebarClassName}
      {...props}
    >
      <SidebarContent className='bg-background text-foreground'>
        <SidebarGroup className='px-5 pt-3 pb-0 md:px-10 md:pt-6'>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  className='h-auto items-start rounded-xl px-0 py-0 text-foreground hover:bg-transparent hover:text-foreground'
                >
                  <div className='flex justify-between'>
                    <div className='text-xl leading-tight font-bold tracking-[-0.04em]'>
                      {item?.title || playbackTitle}
                      <br />
                      <span className='text-base text-muted-foreground'>
                        {dateText || '时间未知'}
                      </span>
                    </div>
                    <div className='mt-7 flex items-center justify-between gap-5 text-sm font-semibold text-muted-foreground'>
                      已播放 {Math.round(progressPercent)}%
                    </div>
                  </div>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroup className='mt-9 px-0 pb-0'>
          <SidebarGroupContent>
            <Tabs defaultValue={hasOverviewItems ? 'overview' : 'reports'}>
              <TabsList className='mx-5 md:mx-6'>
                {hasOverviewItems ? (
                  <TabsTrigger value='overview'>{overviewTabLabel}</TabsTrigger>
                ) : null}
                <TabsTrigger value='reports'>信息</TabsTrigger>
              </TabsList>
              {hasOverviewItems ? (
                <TabsContent value='overview'>
                  {listItems.map((listItem) => {
                    const imageUrl =
                      catalogImageUrl(listItem, 'backdrop') ||
                      catalogImageUrl(listItem, 'poster') ||
                      catalogImageUrl(item ?? {}, 'backdrop')
                    const isCurrent = listItem.id === currentOverviewItemId
                    const secondaryText = listItem.progress?.watched
                      ? '已观看'
                      : listItem.progress?.played_percentage
                        ? `已观看 ${Math.round(listItem.progress.played_percentage)}%`
                        : undefined

                    return (
                      <SidebarMenuSubItem key={listItem.id}>
                        <SidebarMenuSubButton
                          asChild
                          isActive={isCurrent}
                          className='h-auto rounded-none px-0 py-0 text-foreground hover:bg-transparent hover:text-foreground data-active:bg-accent data-active:text-accent-foreground'
                        >
                          <button
                            type='button'
                            onClick={() => {
                              if (!isCurrent) {
                                onOverviewItemSelect(listItem)
                              }
                            }}
                            className='group/episode grid w-full grid-cols-[7.75rem_1fr] gap-6 px-5 py-3 text-left md:px-6'
                          >
                            <div className='relative aspect-video overflow-hidden rounded-md bg-muted/40'>
                              {imageUrl ? (
                                <img
                                  src={imageUrl}
                                  alt=''
                                  className='h-full w-full object-cover transition-transform duration-300 group-hover/episode:scale-105'
                                />
                              ) : null}
                              {listItem.runtime_seconds ? (
                                <div className='absolute right-2 bottom-2 text-[15px] font-bold text-foreground drop-shadow'>
                                  {formatClock(listItem.runtime_seconds)}
                                </div>
                              ) : null}
                            </div>
                            <div className='min-w-0 pt-1'>
                              <div
                                className={`line-clamp-2 text-[20px] leading-snug font-medium tracking-[-0.04em] ${isCurrent ? 'text-primary' : 'text-foreground/85'}`}
                              >
                                {isCurrent ? <PlayingGlyph /> : null}
                                {listItem.label || listItem.title}
                              </div>
                              {secondaryText ? (
                                <div className='mt-6 text-[18px] font-semibold text-muted-foreground'>
                                  {secondaryText}
                                </div>
                              ) : null}
                            </div>
                          </button>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    )
                  })}
                </TabsContent>
              ) : null}
              <TabsContent value='reports'>
                <div className='space-y-5 px-5 py-4 md:px-6'>
                  <section>
                    <div className='mb-2 text-xs font-semibold tracking-[0.18em] text-muted-foreground uppercase'>
                      内容简介
                    </div>
                    <div className='text-sm leading-7 text-muted-foreground'>
                      {infoText}
                    </div>
                  </section>

                  {playbackFacts.length ? (
                    <section>
                      <div className='mb-2 text-xs font-semibold tracking-[0.18em] text-muted-foreground uppercase'>
                        播放详情
                      </div>
                      <DescriptionGrid className='grid-cols-2 sm:grid-cols-2 xl:grid-cols-2'>
                        {playbackFacts.map((fact) => (
                          <DescriptionItem
                            key={fact.label}
                            label={fact.label}
                            value={fact.value}
                            compact
                          />
                        ))}
                      </DescriptionGrid>
                    </section>
                  ) : null}

                  {directorNames.length || castPeople.length ? (
                    <section>
                      <div className='mb-2 text-xs font-semibold tracking-[0.18em] text-muted-foreground uppercase'>
                        演职人员
                      </div>
                      <div className='space-y-3'>
                        {directorNames.length ? (
                          <div>
                            <div className='mb-1.5 text-[11px] font-semibold tracking-[0.14em] text-muted-foreground uppercase'>
                              导演
                            </div>
                            <div className='flex flex-wrap gap-2'>
                              {directorNames.map((person) => (
                                <span
                                  key={`director-${person.name}`}
                                  className='rounded-full border border-border/60 bg-muted/30 px-3 py-1.5 text-sm text-foreground/82'
                                >
                                  {person.name}
                                </span>
                              ))}
                            </div>
                          </div>
                        ) : null}

                        {castPeople.length ? (
                          <div>
                            <div className='mb-1.5 text-[11px] font-semibold tracking-[0.14em] text-muted-foreground uppercase'>
                              主演
                            </div>
                            <div className='flex flex-wrap gap-2'>
                              {castPeople.map((person) => (
                                <span
                                  key={`cast-${person.name}-${person.role ?? ''}`}
                                  className='rounded-full border border-border/60 bg-muted/30 px-3 py-1.5 text-sm text-foreground/82'
                                >
                                  {person.name}
                                  {person.role ? (
                                    <span className='text-muted-foreground'>
                                      {' / '}
                                      {person.role}
                                    </span>
                                  ) : null}
                                </span>
                              ))}
                            </div>
                          </div>
                        ) : null}
                      </div>
                    </section>
                  ) : null}
                </div>
              </TabsContent>
            </Tabs>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      {renderInline ? null : <SidebarRail />}
    </Sidebar>
  )
}

function PlayingGlyph() {
  return (
    <span className='mr-3 inline-flex h-5 translate-y-0.5 items-center gap-1 align-baseline text-primary'>
      <span className='h-4 w-1 rounded-full bg-current' />
      <span className='h-5 w-1 rounded-full bg-current' />
      <span className='h-3 w-1 rounded-full bg-current' />
    </span>
  )
}

function uniquePeople(
  people?: { name: string; role?: string }[] | null
): Array<{ name: string; role?: string }> {
  if (!people?.length) return []

  const seen = new Set<string>()
  return people.filter((person) => {
    const name = person.name?.trim()
    if (!name) return false
    if (seen.has(name)) return false
    seen.add(name)
    return true
  })
}
