import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { CalendarDaysIcon, ClapperboardIcon, MapPinIcon } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import {
  formatProviderLabel,
  getExternalIdentityUrl,
} from '@/lib/media-presentation'
import type { CatalogPersonPageDetail } from '@/lib/mibo-api'
import { catalogPersonDetailQueryOptions } from '@/lib/mibo-query'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Main } from '@/components/layout/main'
import { MediaPosterCard } from '@/components/media-poster-card'

export const Route = createFileRoute('/_authenticated/person/$id')({
  component: PersonDetailPage,
})

function PersonDetailPage() {
  const { id } = Route.useParams()
  const personId = Number(id)
  const token = useAuthStore((state) => state.auth.accessToken)
  const personQuery = useQuery({
    ...catalogPersonDetailQueryOptions(token ?? '', personId),
    enabled: Boolean(token) && Number.isFinite(personId) && personId > 0,
  })

  if (!Number.isFinite(personId) || personId <= 0) {
    return (
      <PersonPageFrame>
        <PersonEmptyState
          title='人物 ID 无效'
          description='请返回上一页重新选择人物。'
        />
      </PersonPageFrame>
    )
  }

  if (personQuery.isPending) {
    return (
      <PersonPageFrame>
        <PersonDetailSkeleton />
      </PersonPageFrame>
    )
  }

  if (personQuery.isError || !personQuery.data) {
    return (
      <PersonPageFrame>
        <PersonEmptyState
          title='人物信息加载失败'
          description='暂时无法读取这个人物的资料，请稍后重试。'
        />
      </PersonPageFrame>
    )
  }

  return (
    <PersonPageFrame>
      <div className='grid gap-8 lg:grid-cols-[minmax(280px,360px)_minmax(0,1fr)] xl:gap-10'>
        <PersonProfileColumn person={personQuery.data} />
        <PersonWorksColumn person={personQuery.data} />
      </div>
    </PersonPageFrame>
  )
}

function PersonPageFrame({ children }: { children: ReactNode }) {
  return (
    <Main className='min-h-svh bg-[radial-gradient(circle_at_top_left,hsl(var(--primary)/0.16),transparent_32rem)] px-4 py-6 sm:px-6 lg:px-8'>
      {children}
    </Main>
  )
}

function PersonProfileColumn({ person }: { person: CatalogPersonPageDetail }) {
  const lifeSpan = formatPersonLifeSpan(person)
  const externalIdentities = person.external_identities ?? []

  return (
    <aside className='lg:sticky lg:top-6 lg:self-start'>
      <div className='overflow-hidden rounded-[2rem] border border-border/60 bg-card/80 shadow-2xl shadow-black/10 backdrop-blur'>
        <div className='relative aspect-[3/4] bg-muted'>
          {person.avatar_url ? (
            <img
              src={person.avatar_url}
              alt={person.name}
              className='h-full w-full object-cover'
            />
          ) : (
            <div className='flex h-full w-full items-center justify-center bg-gradient-to-br from-muted via-muted/80 to-background text-8xl font-semibold text-muted-foreground'>
              {getPersonInitial(person.name)}
            </div>
          )}
          <div className='absolute inset-x-0 bottom-0 bg-gradient-to-t from-background via-background/70 to-transparent p-6 pt-24'>
            <Badge variant='secondary' className='mb-3 rounded-full'>
              {person.known_for_department || '人物'}
            </Badge>
            <h1 className='text-3xl leading-tight font-semibold tracking-tight'>
              {person.name}
            </h1>
          </div>
        </div>

        <div className='space-y-6 p-6'>
          <div className='grid gap-3 text-sm'>
            <PersonFact
              icon={<CalendarDaysIcon className='size-4' />}
              label='生卒'
              value={lifeSpan || '暂无资料'}
            />
            <PersonFact
              icon={<MapPinIcon className='size-4' />}
              label='出生地'
              value={person.place_of_birth || '暂无资料'}
            />
          </div>

          {person.biography ? (
            <section className='space-y-2'>
              <h2 className='text-sm font-medium tracking-wide text-muted-foreground'>
                简介
              </h2>
              <p className='text-sm leading-7 text-foreground/85'>
                {person.biography}
              </p>
            </section>
          ) : (
            <div className='rounded-2xl border border-dashed border-border/70 p-4 text-sm leading-6 text-muted-foreground'>
              还没有同步到人物简介。作品关系已经会在右侧展示。
            </div>
          )}

          {externalIdentities.length ? (
            <section className='space-y-3'>
              <h2 className='text-sm font-medium tracking-wide text-muted-foreground'>
                外部资料
              </h2>
              <div className='flex flex-wrap gap-2'>
                {externalIdentities.map((identity) => {
                  const href = getExternalIdentityUrl(identity)
                  const label = formatProviderLabel(identity.provider)
                  if (!href) {
                    return (
                      <Badge
                        key={`${identity.provider}-${identity.external_id}`}
                        variant='outline'
                      >
                        {label}
                      </Badge>
                    )
                  }
                  return (
                    <Button
                      key={`${identity.provider}-${identity.external_id}`}
                      asChild
                      size='sm'
                      variant='outline'
                      className='rounded-full'
                    >
                      <a href={href} target='_blank' rel='noreferrer'>
                        {label}
                      </a>
                    </Button>
                  )
                })}
              </div>
            </section>
          ) : null}
        </div>
      </div>
    </aside>
  )
}

function PersonWorksColumn({ person }: { person: CatalogPersonPageDetail }) {
  const works = person.related_items ?? []

  return (
    <section className='min-w-0 space-y-5'>
      <div className='flex flex-wrap items-end justify-between gap-4'>
        <div>
          <div className='flex items-center gap-3'>
            <span className='flex size-11 items-center justify-center rounded-2xl bg-muted text-muted-foreground'>
              <ClapperboardIcon className='size-4' />
            </span>
            <h2 className='text-3xl font-semibold tracking-tight'>相关作品</h2>
          </div>
        </div>
        <Badge variant='outline' className='rounded-full px-3 py-1'>
          {works.length} 部作品
        </Badge>
      </div>

      {works.length ? (
        <div className='grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5'>
          {works.map((item) => (
            <MediaPosterCard
              key={`${item.library_id}-${item.metadata_item_id ?? item.id}`}
              item={item}
              layout='grid'
              className='h-full'
            />
          ))}
        </div>
      ) : (
        <PersonEmptyState
          title='暂时没有关联作品'
          description='当媒体元数据里的人物关系完成同步后，电影、电视剧和剧集会出现在这里。'
          compact
        />
      )}
    </section>
  )
}

function PersonFact({
  icon,
  label,
  value,
}: {
  icon: ReactNode
  label: string
  value: string
}) {
  return (
    <div className='flex gap-3 rounded-2xl border border-border/60 bg-background/45 p-3'>
      <div className='mt-0.5 text-muted-foreground'>{icon}</div>
      <div className='min-w-0'>
        <div className='text-xs text-muted-foreground'>{label}</div>
        <div className='mt-1 font-medium break-words'>{value}</div>
      </div>
    </div>
  )
}

function PersonEmptyState({
  title,
  description,
  compact = false,
}: {
  title: string
  description: string
  compact?: boolean
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center rounded-[2rem] border border-dashed border-border/70 bg-card/60 p-8 text-center',
        compact ? 'min-h-72' : 'min-h-[60svh]'
      )}
    >
      <div className='mb-4 flex size-14 items-center justify-center rounded-2xl bg-muted text-muted-foreground'>
        <ClapperboardIcon className='size-6' />
      </div>
      <h1 className='text-2xl font-semibold tracking-tight'>{title}</h1>
      <p className='mt-2 max-w-md text-sm leading-6 text-muted-foreground'>
        {description}
      </p>
    </div>
  )
}

function PersonDetailSkeleton() {
  return (
    <div className='grid animate-pulse gap-8 lg:grid-cols-[minmax(280px,360px)_minmax(0,1fr)] xl:gap-10'>
      <div className='h-[720px] rounded-[2rem] bg-muted/70' />
      <div className='space-y-5'>
        <div className='h-20 rounded-3xl bg-muted/70' />
        <div className='grid grid-cols-2 gap-4 sm:grid-cols-3 xl:grid-cols-4'>
          {Array.from({ length: 8 }).map((_, index) => (
            <div
              key={index}
              className='aspect-[2/3] rounded-[18px] bg-muted/70'
            />
          ))}
        </div>
      </div>
    </div>
  )
}

function formatPersonLifeSpan(person: CatalogPersonPageDetail) {
  const birthday = formatDate(person.birthday)
  const deathday = formatDate(person.deathday)
  if (birthday && deathday) return `${birthday} - ${deathday}`
  if (birthday) return birthday
  if (deathday) return `逝世于 ${deathday}`
  return ''
}

function formatDate(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }).format(date)
}

function getPersonInitial(name: string) {
  return name.trim().slice(0, 1).toUpperCase() || '?'
}
