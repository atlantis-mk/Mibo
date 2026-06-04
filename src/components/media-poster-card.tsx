import {
  useState,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react'
import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
} from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  FileX2Icon,
  HeartIcon,
  InfoIcon,
  MoreHorizontalIcon,
  ShieldCheckIcon,
} from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import {
  formatMediaCardTitle,
  getMediaCardOrganizingLabel,
  getMediaCardPosterUrl,
  getMediaCardType,
} from '@/lib/media-presentation'
import {
  type CatalogListItem,
  type FilenameExclusionPreview,
  type ProgressState,
} from '@/lib/mibo-api'
import {
  catalogPlaybackQueryOptions,
  createAuthedMiboApi,
  favoritesQueryOptions,
  homeDataQueryOptions,
  inventoryFilePlaybackQueryOptions,
  miboQueryKeys,
} from '@/lib/mibo-query'
import { cn } from '@/lib/utils'
import { useIsMobile } from '@/hooks/use-mobile'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  getPlaybackLaunchPreferences,
  openConfiguredExternalPlayer,
} from '@/features/play/external-player'

type MediaPosterCardProps = {
  item: CatalogListItem
  playbackItem?: CatalogListItem
  progress?: ProgressState | null
  progressMeta?: string
  progressDescription?: string
  favorite?: boolean
  libraryName?: string
  layout?: 'rail' | 'grid'
  imageAspect?: 'poster' | 'landscape'
  className?: string
  allowInventoryActions?: boolean
  displaySettings?: MediaPosterDisplaySettings
}

export type MediaPosterImageType =
  | 'primary'
  | 'banner'
  | 'disc'
  | 'logo'
  | 'thumb'
  | 'list'
  | 'datagrid'

export type MediaPosterCardSize =
  | 'extrasmall'
  | 'smaller'
  | 'small'
  | 'normal'
  | 'default'
  | 'large'
  | 'larger'
  | 'extralarge'

export type MediaPosterDisplayField =
  | 'Name'
  | 'OriginalTitle'
  | 'SortName'
  | 'CommunityRating'
  | 'CriticRating'
  | 'OfficialRating'
  | 'ProductionYear'
  | 'PremiereDate'
  | 'Runtime'
  | 'Genres'
  | 'Director'
  | 'Tags'
  | 'Studios'
  | 'Tagline'
  | 'Overview'
  | 'DatePlayed'
  | 'Played'
  | 'DateCreated'
  | 'IsFavorite'

export type MediaPosterDisplaySettings = {
  imageType: MediaPosterImageType
  cardSize: MediaPosterCardSize
  fields: Record<MediaPosterDisplayField, boolean>
}

export const DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS: MediaPosterDisplaySettings =
  {
    imageType: 'primary',
    cardSize: 'default',
    fields: {
      Name: true,
      OriginalTitle: false,
      SortName: false,
      CommunityRating: true,
      CriticRating: false,
      OfficialRating: false,
      ProductionYear: true,
      PremiereDate: false,
      Runtime: true,
      Genres: false,
      Director: false,
      Tags: false,
      Studios: false,
      Tagline: false,
      Overview: false,
      DatePlayed: false,
      Played: false,
      DateCreated: false,
      IsFavorite: false,
    },
  }

type MediaLandscapeCardProps = {
  itemId?: number
  imageUrl?: string
  fallbackImageUrl?: string
  title: string
  subtitle?: string
  meta?: string
  status?: string
  description?: string
  current?: boolean
  actionSlot?: ReactNode
  className?: string
}

export function MediaPosterCard({
  item,
  playbackItem,
  progress,
  progressDescription,
  favorite,
  layout = 'rail',
  imageAspect = 'poster',
  className,
  allowInventoryActions = false,
  displaySettings = DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS,
}: MediaPosterCardProps) {
  const isMobile = useIsMobile()
  const token = useAuthStore((state) => state.auth.accessToken)
  const queryClient = useQueryClient()
  const [ignoreDialogOpen, setIgnoreDialogOpen] = useState(false)
  const [ignorePreview, setIgnorePreview] =
    useState<FilenameExclusionPreview | null>(null)
  const [moreMenuOpen, setMoreMenuOpen] = useState(false)
  const queryToken = token || 'guest'
  const title = formatMediaCardTitle(item)
  const progressFrameUrl = progress?.progress_frame_url
  const posterUrl = getMediaCardPosterUrl(item)
  const displayPosterUrl = progressFrameUrl || posterUrl
  const hasProgress = Boolean(progress && progress.position_seconds > 0)
  const progressPercent = progress ? getProgressPercent(progress) : 0
  const mediaType = getMediaCardType(item)
  const playTarget = playbackItem ?? item
  const isInventoryOnly = item.source_kind === 'inventory_file'
  const playInventoryFileId = isInventoryOnly
    ? item.inventory_file_id
    : undefined
  const ignoreInventoryFileId =
    playTarget.inventory_file_id ?? item.inventory_file_id
  const organizingState = item.organizing_summary?.state
  const isOrganizing = Boolean(item.organizing || isInventoryOnly)
  const canOpenDetails = !isInventoryOnly
  const canApplyIgnore =
    typeof ignoreInventoryFileId === 'number' &&
    ignoreInventoryFileId > 0 &&
    (!item.organizing || organizingState === 'review_required')
  const organizingLabel = getMediaCardOrganizingLabel(item)

  const favoritesQuery = useQuery({
    ...favoritesQueryOptions(queryToken),
    enabled: Boolean(token) && favorite === undefined,
    staleTime: 60_000,
  })

  const isFavorite = isOrganizing
    ? Boolean(
        favorite ??
        favoritesQuery.data?.some((entry) => sameFavoriteItem(entry.item, item))
      )
    : Boolean(
        favorite ??
        favoritesQuery.data?.some((entry) => sameFavoriteItem(entry.item, item))
      )

  const visibleMetadataRows = getVisibleMediaPosterMetadataRows(
    item,
    displaySettings,
    progress,
    isFavorite
  )

  const favoriteMutation = useMutation({
    mutationFn: async (nextFavorite: boolean) => {
      if (!token) throw new Error('当前未登录，无法更新收藏。')
      const api = createAuthedMiboApi(token)
      if (isInventoryOnly) throw new Error('生成条目后可收藏。')
      const metadataItemId = item.metadata_item_id
      if (typeof metadataItemId !== 'number') {
        throw new Error('生成条目后可收藏。')
      }
      return nextFavorite
        ? api.addFavorite(metadataItemId)
        : api.removeFavorite(metadataItemId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: miboQueryKeys.favorites(queryToken),
        }),
        queryClient.invalidateQueries({
          queryKey: homeDataQueryOptions(queryToken).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: ['library', 'browse'] }),
      ])
    },
  })

  const ignoreMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('当前未登录，无法标记忽略。')
      if (!canApplyIgnore) throw new Error('当前条目暂不支持标记忽略。')
      if (typeof ignoreInventoryFileId !== 'number') {
        throw new Error('当前条目缺少文件锚点，无法标记忽略。')
      }
      return createAuthedMiboApi(token).markInventoryFileScanExclusion(
        ignoreInventoryFileId,
        'advertisement'
      )
    },
    onSuccess: async () => {
      await Promise.all([
        invalidateMediaCardQueries(queryClient, queryToken),
        queryClient.invalidateQueries({
          queryKey: ['settings', 'scan-exclusions'],
        }),
      ])
    },
  })

  const previewIgnoreMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('当前未登录，无法预览忽略影响。')
      if (!canApplyIgnore) throw new Error('当前条目暂不支持标记忽略。')
      if (typeof ignoreInventoryFileId !== 'number') {
        throw new Error('当前条目缺少文件锚点，无法预览忽略影响。')
      }
      return createAuthedMiboApi(token).previewInventoryFileScanExclusion(
        ignoreInventoryFileId
      )
    },
    onSuccess: (preview) => {
      setIgnorePreview(preview)
      setIgnoreDialogOpen(true)
    },
  })

  const filenameGroupMutation = useMutation({
    mutationFn: async () => {
      if (!token) throw new Error('当前未登录，无法标记同名忽略。')
      if (!canApplyIgnore) throw new Error('当前条目暂不支持同名忽略。')
      if (typeof ignoreInventoryFileId !== 'number') {
        throw new Error('当前条目缺少文件锚点，无法标记同名忽略。')
      }
      return createAuthedMiboApi(
        token
      ).createInventoryFileFilenameExclusionRule(
        ignoreInventoryFileId,
        'advertisement'
      )
    },
    onSuccess: async () => {
      setIgnoreDialogOpen(false)
      await Promise.all([
        invalidateMediaCardQueries(queryClient, queryToken),
        queryClient.invalidateQueries({
          queryKey: ['settings', 'scan-exclusions'],
        }),
        queryClient.invalidateQueries({ queryKey: ['home'] }),
        queryClient.invalidateQueries({ queryKey: ['library'] }),
        queryClient.invalidateQueries({ queryKey: ['catalog'] }),
      ])
    },
  })

  const actionsPending =
    favoriteMutation.isPending ||
    ignoreMutation.isPending ||
    previewIgnoreMutation.isPending ||
    filenameGroupMutation.isPending
  const canIgnore =
    canApplyIgnore &&
    playTarget.type !== 'series' &&
    playTarget.type !== 'show' &&
    (!isInventoryOnly || allowInventoryActions)
  const canOpenMoreMenu =
    Boolean(token) && (!isInventoryOnly || allowInventoryActions)

  const handleMoreTriggerPointerDown = (
    event: ReactPointerEvent<HTMLButtonElement>
  ) => {
    if (!isMobile) return
    if (event.pointerType === 'mouse') return

    // Radix opens the menu on pointerdown, which makes mobile scrolling
    // across the action row feel sticky. Let click/tap open it instead.
    event.preventDefault()
  }

  const handlePlayCardClick = async (event: { preventDefault(): void }) => {
    const launchPreferences = getPlaybackLaunchPreferences()
    if (launchPreferences.mode !== 'external') {
      return
    }

    event.preventDefault()

    if (!token) {
      toast.error('当前未登录，无法获取外部播放器播放链接。')
      return
    }

    try {
      const playbackSource =
        typeof playInventoryFileId === 'number' && playInventoryFileId > 0
          ? await queryClient.fetchQuery(
              inventoryFilePlaybackQueryOptions(queryToken, playInventoryFileId)
            )
          : await queryClient.fetchQuery(
              catalogPlaybackQueryOptions(queryToken, playTarget.id)
            )

      const playbackUrl = playbackSource.parts?.[0]?.url || playbackSource.url
      const launchResult = openConfiguredExternalPlayer({
        playbackUrl,
        title: playbackSource.title || title,
      })

      if (!launchResult.ok) {
        toast.error(launchResult.message)
      }
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : '无法获取外部播放器播放链接'
      )
    }
  }

  return (
    <article
      className={cn(
        'group relative transition-transform duration-200 [content-visibility:auto] hover:-translate-y-1',
        layout === 'grid'
          ? 'w-full min-w-0 [contain-intrinsic-size:220px_400px]'
          : 'w-[172px] shrink-0 [contain-intrinsic-size:204px_380px] sm:w-[204px]',
        className
      )}
    >
      <div className='relative overflow-hidden rounded-[1.35rem] border border-border/40 bg-card/75 shadow-lg'>
        <Link
          to='/play/$id'
          params={{ id: String(playInventoryFileId ?? playTarget.id) }}
          search={{
            fromStart: !hasProgress,
            inventoryFileId: playInventoryFileId,
            resourceId: undefined,
            liveChannelId: undefined,
            liveSourceId: undefined,
          }}
          onClick={(event) => {
            void handlePlayCardClick(event)
          }}
          preload='intent'
          aria-label={`${hasProgress ? '继续播放' : '播放'} ${title}`}
          className='absolute inset-0 z-10 rounded-[1.35rem] focus:outline-none focus-visible:ring-2 focus-visible:ring-primary'
        />
        <div
          className={cn(
            'relative overflow-hidden bg-muted',
            imageAspect === 'landscape' ? 'aspect-video' : 'aspect-[2/3]'
          )}
        >
          {displayPosterUrl ? (
            <img
              src={displayPosterUrl}
              alt=''
              loading='lazy'
              decoding='async'
              fetchPriority='low'
              sizes='(min-width: 1536px) 12vw, (min-width: 1280px) 18vw, (min-width: 1024px) 23vw, (min-width: 640px) 31vw, 47vw'
              className='h-full w-full object-cover'
            />
          ) : (
            <div className='h-full w-full bg-linear-to-b from-indigo-500/35 to-teal-700/35' />
          )}
          {isOrganizing ? (
            <Badge
              variant='secondary'
              className={cn(
                'absolute top-2 left-2 rounded-full px-2 py-1 shadow-lg',
                organizingState === 'failed'
                  ? 'text-destructive-foreground border-destructive/30 bg-destructive'
                  : organizingState === 'review_required'
                    ? 'border-border bg-secondary text-secondary-foreground'
                    : 'bg-secondary text-secondary-foreground'
              )}
            >
              {organizingLabel}
            </Badge>
          ) : null}
          {hasProgress ? (
            <div className='absolute right-0 bottom-0 left-0 h-1.5 bg-white/25'>
              <div
                className='h-full bg-white shadow-[0_0_12px_rgba(255,255,255,0.6)]'
                style={{ width: `${progressPercent}%` }}
              />
            </div>
          ) : null}
        </div>
        <div className='space-y-3 px-3 pt-3 pb-3'>
          <div className='space-y-1.5'>
            {displaySettings.fields.Name ? (
              <div className='line-clamp-1 text-sm font-semibold tracking-tight text-foreground sm:text-base'>
                {title}
              </div>
            ) : null}
            {progressDescription ? (
              <div className='mt-1 line-clamp-1 text-xs text-muted-foreground'>
                {progressDescription}
              </div>
            ) : null}
            {visibleMetadataRows.length > 0 ? (
              <div className='space-y-1 text-xs text-muted-foreground'>
                {visibleMetadataRows.map((row) => (
                  <div key={row.key} className='line-clamp-2'>
                    <span className='text-foreground/80'>{row.label}</span>
                    <span className='mx-1'>·</span>
                    <span>{row.value}</span>
                  </div>
                ))}
              </div>
            ) : null}
          </div>
          <div className='relative z-20 flex items-center gap-2'>
            {canOpenDetails ? (
              <Button asChild size='icon-sm' variant='outline'>
                <Link
                  to='/media/$id'
                  params={{ id: String(item.id) }}
                  search={{
                    view: mediaType === 'show' ? 'series' : undefined,
                    episodePage: undefined,
                  }}
                  preload='intent'
                >
                  <InfoIcon />
                  <span className='sr-only'>详情</span>
                </Link>
              </Button>
            ) : (
              <Button size='icon-sm' variant='outline' disabled>
                <InfoIcon />
                <span className='sr-only'>详情</span>
              </Button>
            )}
            <Button
              type='button'
              size='icon-sm'
              variant='outline'
              disabled={isInventoryOnly || !token || favoriteMutation.isPending}
              onClick={() => favoriteMutation.mutate(!isFavorite)}
            >
              <HeartIcon className={cn(isFavorite ? 'fill-current' : '')} />
              <span className='sr-only'>
                {isFavorite ? '取消收藏' : '加入收藏'}
              </span>
            </Button>
            <DropdownMenu open={moreMenuOpen} onOpenChange={setMoreMenuOpen}>
              <DropdownMenuTrigger asChild>
                <Button
                  type='button'
                  size='icon-sm'
                  variant='outline'
                  disabled={!canOpenMoreMenu}
                  onPointerDown={handleMoreTriggerPointerDown}
                  onClick={() => setMoreMenuOpen(true)}
                >
                  <MoreHorizontalIcon />
                  <span className='sr-only'>更多操作</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align='end' className='w-48'>
                <DropdownMenuLabel className='truncate'>
                  {title}
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuGroup>
                  {isInventoryOnly && !allowInventoryActions ? (
                    <DropdownMenuItem disabled>
                      生成条目后可操作
                    </DropdownMenuItem>
                  ) : null}
                  {!isInventoryOnly ? (
                    <DropdownMenuItem asChild disabled={actionsPending}>
                      <Link
                        to='/settings/metadata/$id'
                        params={{ id: String(item.id) }}
                      >
                        <ShieldCheckIcon className='size-4' />
                        治理元数据
                      </Link>
                    </DropdownMenuItem>
                  ) : null}
                  {canIgnore ? (
                    <DropdownMenuItem
                      variant='destructive'
                      disabled={actionsPending}
                      onSelect={() => previewIgnoreMutation.mutate()}
                    >
                      <FileX2Icon className='size-4' />
                      标记忽略
                    </DropdownMenuItem>
                  ) : null}
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
      <Dialog open={ignoreDialogOpen} onOpenChange={setIgnoreDialogOpen}>
        <DialogContent className='grid max-h-[85vh] w-[calc(100vw-2rem)] max-w-2xl grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden p-0'>
          <DialogHeader>
            <div className='space-y-2 px-6 pt-6'>
              <DialogTitle>选择忽略范围</DialogTitle>
              <DialogDescription>
                先确认同名文件影响范围，再选择只忽略当前文件或忽略所有同名文件。
              </DialogDescription>
            </div>
          </DialogHeader>
          <div className='min-h-0 overflow-y-auto px-6 py-4'>
            {ignorePreview ? (
              <div className='min-w-0 space-y-3'>
                <div className='min-w-0 rounded-xl border border-border/60 bg-muted/40 p-3 text-sm'>
                  <div className='font-medium break-all'>
                    {ignorePreview.normalized_filename}
                  </div>
                  <div className='mt-1 break-all text-muted-foreground'>
                    {ignorePreview.library_name ||
                      `#${ignorePreview.library_id}`}{' '}
                    / {ignorePreview.storage_provider}，共影响{' '}
                    {ignorePreview.affected_count} 个文件
                  </div>
                </div>
                <div className='max-h-64 min-w-0 space-y-2 overflow-y-auto rounded-xl border border-border/60 p-3'>
                  {ignorePreview.affected_files.map((file) => (
                    <div
                      key={file.id}
                      className='text-xs break-all text-muted-foreground'
                      title={file.storage_path}
                    >
                      {file.storage_path}
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
          <div className='flex flex-col gap-2 border-t border-border/60 bg-muted/30 px-6 py-4 sm:flex-row sm:justify-end'>
            <Button
              variant='outline'
              className='w-full sm:w-auto'
              disabled={
                ignoreMutation.isPending || filenameGroupMutation.isPending
              }
              onClick={() => ignoreMutation.mutate()}
            >
              仅忽略当前文件
            </Button>
            <Button
              variant='destructive'
              className='w-full sm:w-auto'
              disabled={
                ignoreMutation.isPending || filenameGroupMutation.isPending
              }
              onClick={() => filenameGroupMutation.mutate()}
            >
              忽略所有同名文件
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </article>
  )
}

function getProgressPercent(progress: ProgressState) {
  if (typeof progress.played_percentage === 'number') {
    return clampProgressPercent(progress.played_percentage)
  }

  if (progress.duration_seconds && progress.duration_seconds > 0) {
    return clampProgressPercent(
      (progress.position_seconds / progress.duration_seconds) * 100
    )
  }

  return 0
}

function getVisibleMediaPosterMetadataRows(
  item: CatalogListItem,
  settings: MediaPosterDisplaySettings,
  progress: ProgressState | null | undefined,
  isFavorite: boolean
) {
  const rows: Array<{
    key: MediaPosterDisplayField
    label: string
    value: string
  }> = []
  const addRow = (
    key: MediaPosterDisplayField,
    label: string,
    value: string | number | undefined | null
  ) => {
    if (!settings.fields[key]) return
    const normalized = String(value ?? '').trim()
    if (!normalized) return
    rows.push({ key, label, value: normalized })
  }

  addRow('OriginalTitle', '原标题', item.original_title)
  addRow('SortName', '排序标题', item.local_title || item.title)
  addRow('CommunityRating', 'IMDb 评分', formatRating(item.community_rating))
  addRow('CriticRating', '影评人评分', formatRating(item.community_rating))
  addRow('OfficialRating', '家长评分', item.official_rating)
  addRow('ProductionYear', '年份', item.year)
  addRow(
    'PremiereDate',
    '发行日期',
    formatDate(item.release_date || item.first_air_date)
  )
  addRow('Runtime', '播放时长', formatRuntime(item.runtime_seconds))
  addRow('Genres', '类型', item.genres?.join(' / '))
  addRow(
    'Director',
    '导演',
    item.directors?.map((director) => director.name).join(' / ')
  )
  addRow(
    'Tags',
    '标签',
    item.tags
      ?.filter((tag) => tag.kind !== 'genre')
      .map((tag) => tag.name)
      .join(' / ')
  )
  addRow('Overview', '概要', item.overview)
  addRow('DatePlayed', '播放日期', formatDate(progress?.last_played_at))
  addRow('Played', '已播放', progress?.completed_at ? '是' : undefined)
  addRow(
    'DateCreated',
    '创建日期',
    formatDate(item.child_summary?.latest_added_at)
  )
  addRow('IsFavorite', '收藏', isFavorite ? '已收藏' : undefined)

  return rows
}

function formatRating(value: number | undefined) {
  return typeof value === 'number' ? value.toFixed(1) : undefined
}

function formatDate(value: string | undefined) {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString()
}

function formatRuntime(value: number | undefined) {
  if (typeof value !== 'number' || value <= 0) return undefined
  const minutes = Math.round(value / 60)
  if (minutes < 60) return `${minutes} 分钟`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return remainingMinutes > 0
    ? `${hours} 小时 ${remainingMinutes} 分钟`
    : `${hours} 小时`
}

function clampProgressPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
}

function sameFavoriteItem(candidate: CatalogListItem, item: CatalogListItem) {
  return candidate.metadata_item_id === item.metadata_item_id
}

async function invalidateMediaCardQueries(
  queryClient: QueryClient,
  queryToken: string
) {
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: homeDataQueryOptions(queryToken).queryKey,
    }),
    queryClient.invalidateQueries({ queryKey: ['library', 'browse'] }),
    queryClient.invalidateQueries({ queryKey: ['catalog', 'detail'] }),
    queryClient.invalidateQueries({ queryKey: ['catalog', 'series-seasons'] }),
  ])
}

export function MediaLandscapeCard({
  itemId,
  imageUrl,
  fallbackImageUrl,
  title,
  subtitle,
  meta,
  status,
  description,
  current,
  actionSlot,
  className,
}: MediaLandscapeCardProps) {
  const visualUrl = imageUrl || fallbackImageUrl
  const cardContent = (
    <div
      className={cn(
        'group overflow-hidden rounded-[16px] border border-border/40 bg-card/70 shadow-lg backdrop-blur-md transition',
        current && 'border-primary/70 bg-primary/10',
        itemId ? 'hover:border-border/70 hover:bg-card/85' : 'opacity-90',
        className
      )}
    >
      <div className='relative aspect-video overflow-hidden bg-muted'>
        {visualUrl ? (
          <img
            src={visualUrl}
            alt={title}
            className='h-full w-full object-cover transition duration-300 group-hover:scale-[1.03]'
          />
        ) : null}
        <div className='absolute inset-0 bg-gradient-to-t from-background/90 via-background/15 to-transparent' />
      </div>
      <div className='space-y-2 p-4'>
        {actionSlot && itemId ? (
          <Link
            to='/media/$id'
            params={{ id: String(itemId) }}
            search={{ view: undefined, episodePage: undefined }}
            className='line-clamp-1 text-lg text-foreground underline-offset-4 hover:underline'
          >
            {subtitle ? `${subtitle} - ${title}` : title}
          </Link>
        ) : (
          <div className='line-clamp-1 text-lg text-foreground'>
            {subtitle ? `${subtitle} - ${title}` : title}
          </div>
        )}
        {meta ? (
          <div className='text-sm text-muted-foreground'>{meta}</div>
        ) : null}
        {status ? (
          <div className='text-xs text-muted-foreground'>{status}</div>
        ) : null}
        {description ? (
          <p className='line-clamp-3 text-sm leading-6 text-muted-foreground'>
            {description}
          </p>
        ) : null}
        {actionSlot ? <div className='pt-1'>{actionSlot}</div> : null}
      </div>
    </div>
  )

  if (!itemId || actionSlot) {
    return cardContent
  }

  return (
    <Link
      to='/media/$id'
      params={{ id: String(itemId) }}
      search={{ view: undefined, episodePage: undefined }}
    >
      {cardContent}
    </Link>
  )
}
