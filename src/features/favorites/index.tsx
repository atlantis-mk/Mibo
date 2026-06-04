import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { HeartIcon, LoaderCircleIcon } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import { favoritesQueryOptions } from '@/lib/mibo-query'
import { Button } from '@/components/ui/button'
import { MediaPosterCard } from '@/components/media-poster-card'

export default function FavoritesPage() {
  const token = useAuthStore((state) => state.auth.accessToken)
  const user = useAuthStore((state) => state.auth.user)
  const hasHydrated = useAuthStore((state) => state.auth.hasHydrated)
  const queryToken = token ?? 'guest'

  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: hasHydrated && !!token,
  })

  if (!hasHydrated || (token && favoritesQuery.isLoading)) {
    return (
      <div className='flex min-h-svh items-center justify-center bg-background text-foreground'>
        <LoaderCircleIcon className='size-4 animate-spin' />
      </div>
    )
  }

  if (!token || !user) {
    return (
      <div className='flex min-h-svh items-center justify-center px-6 text-center'>
        <div>
          <h1 className='text-3xl font-semibold'>登录后查看收藏</h1>
          <Button asChild className='mt-4 rounded-full'>
            <Link to='/sign-in' search={{ redirect: '/favorites' }}>
              前往登录
            </Link>
          </Button>
        </div>
      </div>
    )
  }

  const favorites = favoritesQuery.data ?? []

  return (
    <div className='relative min-w-0 flex-1 bg-background text-foreground'>
      <section className='px-4 py-10 sm:px-6 sm:py-12 lg:px-8'>
        <div className='mx-auto max-w-[1600px]'>
          {favorites.length > 0 ? (
            <div className='grid gap-5 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6'>
              {favorites.map((entry) => (
                <MediaPosterCard
                  key={entry.item.metadata_item_id}
                  item={entry.item}
                  progress={entry}
                  layout='grid'
                />
              ))}
            </div>
          ) : (
            <div className='rounded-[2rem] border border-border/40 bg-card/70 px-6 py-10 text-center text-sm text-muted-foreground backdrop-blur-sm'>
              <HeartIcon className='mx-auto mb-4 size-8 text-muted-foreground' />
              还没有收藏。打开媒体详情或首页卡片，将喜欢的作品加入收藏。
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
