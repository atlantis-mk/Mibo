import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  ArrowLeftIcon,
  ChevronRightIcon,
  CopyIcon,
  ExternalLinkIcon,
  HeartIcon,
  LoaderCircleIcon,
  MoreHorizontalIcon,
  Share2Icon,
  SearchIcon,
  Settings2Icon,
} from "lucide-react"

import { AppTopBar } from "#/components/app-top-bar"
import { Badge } from "#/components/ui/badge"
import { Button } from "#/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu"
import { SidebarTrigger } from "#/components/ui/sidebar"
import type { CatalogListItem, CatalogPersonPageDetail } from "#/lib/mibo-api"
import { ApiError } from "#/lib/mibo-api"
import { catalogPersonDetailQueryOptions } from "#/lib/mibo-query"
import {
  formatMediaCardYearRange,
  getExternalIdentityUrl,
  getMediaCardBackdropUrl,
  getMediaCardPosterUrl,
} from "#/lib/media-presentation"
import { cn } from "#/lib/utils"
import { useAuthStore } from "#/stores/auth-store"

const personFavoritesStorageKey = "mibo-web-favorite-people"

export default function PersonDetailPage({ personId }: { personId: number }) {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const hasHydrated = useAuthStore((state) => state.hasHydrated)
  const navigate = useNavigate()
  const queryToken = token ?? "guest"
  const hasValidPersonId = Number.isFinite(personId) && personId > 0
  const [isFavorite, setIsFavorite] = useState(false)
  const [overviewExpanded, setOverviewExpanded] = useState(false)

  const personQuery = useQuery({
    ...catalogPersonDetailQueryOptions(queryToken, personId),
    enabled: hasHydrated && !!token && hasValidPersonId,
  })

  useEffect(() => {
    if (!hasHydrated || !hasValidPersonId) return
    setIsFavorite(readFavoritePeople().includes(personId))
  }, [hasHydrated, hasValidPersonId, personId])

  useEffect(() => {
    setOverviewExpanded(false)
  }, [personId])

  if (!hasHydrated || (token && personQuery.isLoading)) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3 rounded-full border border-border/40 bg-background/80 px-5 py-3 backdrop-blur-xl">
          <LoaderCircleIcon className="size-4 animate-spin" />
          <span className="text-sm text-muted-foreground">
            正在加载人物详情
          </span>
        </div>
      </div>
    )
  }

  if (!token || !user) {
    void navigate({
      to: "/login",
      search: { redirect: `/person/${personId}` },
      replace: true,
    })
    return null
  }

  if (!hasValidPersonId) {
    return (
      <PersonDetailError
        title="无效的人物 ID"
        message="请从媒体详情页重新进入人物页面。"
      />
    )
  }

  if (personQuery.error) {
    if (
      personQuery.error instanceof ApiError &&
      personQuery.error.status === 404
    ) {
      return (
        <PersonDetailError
          title="未找到该人物"
          message="这个人物可能尚未和本地媒体建立关联，或者已经不再存在。"
          missing
        />
      )
    }

    return (
      <PersonDetailError
        title="人物详情暂时不可用"
        message={personQuery.error.message}
      />
    )
  }

  if (!personQuery.data) {
    return (
      <PersonDetailError
        title="人物详情暂时不可用"
        message="未返回人物详情数据。"
      />
    )
  }

  const person = personQuery.data
  const relatedItems = person.related_items ?? []
  const externalIdentities = person.external_identities ?? []
  const heroBackdrop = getPersonHeroBackdropUrl(relatedItems)
  const primaryFacts = buildPrimaryFacts(person)
  const secondaryFacts = buildSecondaryFacts(person)
  const externalLinks = externalIdentities
    .map((identity) => ({
      identity,
      href: getExternalIdentityUrl(identity),
    }))
    .filter((entry) => entry.href)

  const toggleFavorite = () => {
    const next = toggleFavoritePerson(person.id)
    setIsFavorite(next)
  }

  const copyPageLink = async () => {
    if (typeof window === "undefined" || !navigator.clipboard) return
    await navigator.clipboard.writeText(window.location.href)
  }

  return (
    <div className="relative min-w-0 flex-1 overflow-x-hidden bg-background text-foreground">
      <div
        className="absolute inset-0 bg-cover bg-center"
        style={{
          backgroundImage: heroBackdrop
            ? `url(${heroBackdrop})`
            : "linear-gradient(135deg, rgba(47,67,98,0.92), rgba(20,24,37,0.98))",
        }}
      />
      <div className="absolute inset-0 bg-gradient-to-b from-background/82 via-background/74 to-background" />
      <div className="absolute inset-0 bg-gradient-to-r from-background via-background/88 to-background/96" />

      <AppTopBar
        leftSlot={
          <>
            <SidebarTrigger />
            <Button
              type="button"
              size="icon-sm"
              variant="outline"
              onClick={() => {
                if (window.history.length > 1) {
                  window.history.back()
                  return
                }
                void navigate({ to: "/" })
              }}
            >
              <ArrowLeftIcon className="size-4" />
              <span className="sr-only">返回</span>
            </Button>
            <div className="min-w-0">
              <div className="truncate text-sm font-medium">人物详情</div>
              <div className="truncate text-xs text-muted-foreground">
                {person.name}
              </div>
            </div>
          </>
        }
        rightSlot={
          <div className="hidden items-center gap-2 sm:flex">
            <Button asChild size="icon-sm" variant="outline">
              <Link to="/search" search={{ q: undefined }}>
                <SearchIcon className="size-4" />
                <span className="sr-only">搜索</span>
              </Link>
            </Button>
            <Button asChild size="icon-sm" variant="outline">
              <Link to="/settings">
                <Settings2Icon className="size-4" />
                <span className="sr-only">设置</span>
              </Link>
            </Button>
          </div>
        }
      />

      <main className="relative px-4 pt-24 pb-16 sm:px-6 lg:px-8">
        <div className="mx-auto grid max-w-[1680px] gap-8 lg:grid-cols-[280px_minmax(0,1fr)] xl:grid-cols-[300px_minmax(0,1fr)]">
          <aside className="space-y-10 lg:pt-1">
            <div className="overflow-hidden rounded-xl border border-border/30 bg-card/30 shadow-2xl backdrop-blur-sm">
              <div className="relative aspect-[2/3] bg-muted">
                {person.avatar_url ? (
                  <img
                    src={person.avatar_url}
                    alt={person.name}
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-muted via-muted/70 to-background text-7xl font-semibold text-muted-foreground">
                    {getPersonInitial(person.name)}
                  </div>
                )}
              </div>
            </div>

            <div className="space-y-5">
              <div className="flex items-center gap-2 text-2xl font-semibold tracking-tight text-foreground">
                <span>影片</span>
                <ChevronRightIcon className="size-5 text-muted-foreground" />
              </div>

              {relatedItems.length > 0 ? (
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-1">
                  {relatedItems.map((item) => (
                    <PersonRelatedWorkCard key={item.id} item={item} />
                  ))}
                </div>
              ) : (
                <div className="text-sm leading-7 text-muted-foreground">
                  当前媒体库里还没有和 {person.name} 建立关联的影片作品。
                </div>
              )}
            </div>

            {(externalLinks.length > 0 || secondaryFacts.length > 0) && (
              <div className="space-y-5 pb-8">
                <div className="text-2xl font-semibold tracking-tight text-foreground">
                  其它信息
                </div>

                {secondaryFacts.length > 0 ? (
                  <div className="space-y-4 text-sm text-foreground/90">
                    {secondaryFacts.map((fact) => (
                      <div key={fact.label} className="space-y-1">
                        <div className="font-medium text-muted-foreground">
                          {fact.label}
                        </div>
                        <div>{fact.value}</div>
                      </div>
                    ))}
                  </div>
                ) : null}

                {externalLinks.length > 0 ? (
                  <div className="space-y-3">
                    <div className="font-medium text-foreground">
                      数据库链接
                    </div>
                    <div className="flex flex-wrap gap-x-3 gap-y-2 text-sm text-muted-foreground">
                      {externalLinks.map(({ identity, href }) => (
                        <a
                          key={`${identity.provider}-${identity.external_id}`}
                          href={href}
                          target="_blank"
                          rel="noreferrer"
                          className="transition hover:text-foreground"
                        >
                          {formatExternalLinkLabel(identity.provider)}
                        </a>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            )}
          </aside>

          <section className="min-w-0 space-y-6 pt-1 lg:pl-2 xl:pl-4">
            <div className="max-w-4xl space-y-6">
              <div className="space-y-5">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-3">
                      <h1 className="text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
                        {person.name}
                      </h1>
                      {person.known_for_department ? (
                        <Badge
                          className="border-border/50 bg-background/70 text-foreground"
                          variant="outline"
                        >
                          {person.known_for_department}
                        </Badge>
                      ) : null}
                    </div>
                    {person.sort_name && person.sort_name !== person.name ? (
                      <p className="text-sm text-muted-foreground sm:text-base">
                        {person.sort_name}
                      </p>
                    ) : null}
                  </div>

                  <div className="flex shrink-0 items-center gap-3">
                    <Button
                      type="button"
                      size="icon"
                      variant="outline"
                      className={cn(
                        "size-13 rounded-full border-border/50 bg-background/80",
                        isFavorite
                          ? "text-rose-400 hover:text-rose-300"
                          : "text-muted-foreground hover:text-foreground"
                      )}
                      onClick={toggleFavorite}
                    >
                      <HeartIcon
                        className={cn("size-5", isFavorite && "fill-current")}
                      />
                      <span className="sr-only">
                        {isFavorite ? "取消喜欢" : "喜欢人物"}
                      </span>
                    </Button>

                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          type="button"
                          size="icon"
                          variant="outline"
                          className="size-13 rounded-full border-border/50 bg-background/80 text-muted-foreground hover:text-foreground"
                        >
                          <MoreHorizontalIcon className="size-5" />
                          <span className="sr-only">更多操作</span>
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-56">
                        <DropdownMenuLabel>更多操作</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onSelect={() => void copyPageLink()}>
                          <CopyIcon className="size-4" />
                          复制人物页面链接
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onSelect={() => void toggleFavorite()}
                        >
                          <HeartIcon className="size-4" />
                          {isFavorite ? "取消喜欢" : "加入喜欢"}
                        </DropdownMenuItem>
                        <DropdownMenuItem onSelect={() => void copyPageLink()}>
                          <Share2Icon className="size-4" />
                          分享页面
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        {externalLinks.map(({ identity, href }) => (
                          <DropdownMenuItem asChild key={href}>
                            <a href={href} target="_blank" rel="noreferrer">
                              <ExternalLinkIcon className="size-4" />
                              打开 {formatExternalLinkLabel(identity.provider)}
                            </a>
                          </DropdownMenuItem>
                        ))}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>

                <div className="space-y-3 text-base text-foreground/92">
                  {primaryFacts.map((fact) => (
                    <div key={fact.label} className="leading-7">
                      {fact.label}：{fact.value}
                    </div>
                  ))}
                </div>
              </div>

              <div className="max-w-4xl">
                <p
                  className={cn(
                    "text-lg leading-10 text-foreground/92",
                    !overviewExpanded && "line-clamp-3"
                  )}
                >
                  {person.biography ||
                    `${person.name} 当前只有基础人物资料，后续可通过元数据刷新补充更完整的生平与背景介绍。`}
                </p>
                {person.biography && person.biography.length > 220 ? (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => setOverviewExpanded((value) => !value)}
                  >
                    {overviewExpanded ? "收起" : "更多"}
                  </Button>
                ) : null}
              </div>
            </div>
          </section>
        </div>
      </main>
    </div>
  )
}

function PersonDetailError({
  title,
  message,
  missing = false,
}: {
  title: string
  message: string
  missing?: boolean
}) {
  return (
    <div className="flex min-h-svh items-center justify-center bg-background px-6 text-foreground">
      <div className="max-w-lg rounded-[2rem] border border-border/40 bg-card/80 p-8 text-center backdrop-blur-xl">
        <Badge className="border-border/60 bg-background/80" variant="outline">
          {missing ? "未找到" : "加载失败"}
        </Badge>
        <h1 className="mt-4 text-3xl font-semibold tracking-tight">{title}</h1>
        <p className="mt-3 text-sm leading-7 text-muted-foreground">
          {message}
        </p>
        <div className="mt-6 flex justify-center gap-3">
          <Button asChild variant="outline">
            <Link to="/">返回首页</Link>
          </Button>
          <Button asChild>
            <Link to="/search" search={{ q: undefined }}>
              继续浏览
            </Link>
          </Button>
        </div>
      </div>
    </div>
  )
}

function getPersonHeroBackdropUrl(relatedItems: CatalogListItem[]) {
  for (const item of relatedItems) {
    const backdrop = getMediaCardBackdropUrl(item)
    if (backdrop) return backdrop
    const poster = getMediaCardPosterUrl(item)
    if (poster) return poster
  }
  return ""
}

function buildPrimaryFacts(person: CatalogPersonPageDetail) {
  const facts: Array<{ label: string; value: string }> = []
  const birthday = formatPersonDate(person.birthday)
  if (birthday) {
    const age = formatPersonAge(person.birthday, person.deathday)
    facts.push({
      label: "出生",
      value: age ? `${birthday} · ${age}` : birthday,
    })
  }
  const deathday = formatPersonDate(person.deathday)
  if (deathday) {
    facts.push({ label: "逝世", value: deathday })
  }
  if (person.place_of_birth?.trim()) {
    facts.push({ label: "出生地", value: person.place_of_birth.trim() })
  }
  return facts
}

function buildSecondaryFacts(person: CatalogPersonPageDetail) {
  const facts: Array<{ label: string; value: string }> = []
  if (person.known_for_department?.trim()) {
    facts.push({ label: "领域", value: person.known_for_department.trim() })
  }
  if ((person.related_items?.length ?? 0) > 0) {
    const lead = person.related_items?.[0]
    if (lead) {
      facts.push({
        label: "代表作品",
        value: `${lead.title} · ${formatMediaCardYearRange(lead)}`,
      })
    }
  }
  return facts
}

function formatExternalLinkLabel(provider: string) {
  switch (provider.trim().toLowerCase()) {
    case "imdb":
      return "IMDb"
    case "tmdb":
      return "TheMovieDb"
    default:
      return provider.toUpperCase()
  }
}

function formatPersonDate(value?: string) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ""
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "long",
    day: "numeric",
  }).format(date)
}

function formatPersonAge(birthday?: string, deathday?: string) {
  if (!birthday) return ""
  const start = new Date(birthday)
  const end = deathday ? new Date(deathday) : new Date()
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return ""

  let age = end.getFullYear() - start.getFullYear()
  const monthDelta = end.getMonth() - start.getMonth()
  if (monthDelta < 0 || (monthDelta === 0 && end.getDate() < start.getDate())) {
    age -= 1
  }
  return age > 0 ? `${age} 岁` : ""
}

function getPersonInitial(name: string) {
  const normalized = name.trim()
  if (!normalized) return "?"
  return Array.from(normalized)[0]?.toUpperCase() ?? "?"
}

function PersonRelatedWorkCard({ item }: { item: CatalogListItem }) {
  const posterUrl = getMediaCardPosterUrl(item) || getMediaCardBackdropUrl(item)

  return (
    <Link
      to="/media/$id"
      params={{ id: String(item.id) }}
      search={{ view: item.type === "series" ? "series" : undefined }}
      className="group block min-w-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
    >
      <div className="overflow-hidden rounded-xl border border-border/30 bg-card/30 shadow-xl backdrop-blur-sm transition group-hover:border-border/60">
        <div className="aspect-[2/3] bg-muted">
          {posterUrl ? (
            <img
              src={posterUrl}
              alt={`${item.title} 海报`}
              className="h-full w-full object-cover transition duration-300 group-hover:scale-[1.02]"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-muted via-muted/70 to-background text-4xl font-semibold text-muted-foreground">
              {item.title.slice(0, 1)}
            </div>
          )}
        </div>
      </div>
      <div className="px-2 pt-3 text-center">
        <div className="line-clamp-2 text-base font-medium text-foreground">
          {item.title}
        </div>
        <div className="mt-1 text-sm text-muted-foreground">
          {formatMediaCardYearRange(item)}
        </div>
      </div>
    </Link>
  )
}

function readFavoritePeople() {
  if (typeof window === "undefined") return [] as number[]
  try {
    const raw = window.localStorage.getItem(personFavoritesStorageKey)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((value): value is number => typeof value === "number")
  } catch {
    return []
  }
}

function toggleFavoritePerson(personId: number) {
  const current = readFavoritePeople()
  const next = current.includes(personId)
    ? current.filter((value) => value !== personId)
    : [...current, personId]
  if (typeof window !== "undefined") {
    window.localStorage.setItem(personFavoritesStorageKey, JSON.stringify(next))
  }
  return next.includes(personId)
}
