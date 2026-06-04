import {
  Fragment,
  useEffect,
  useState,
  type FormEvent,
  type RefObject,
} from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS,
  MediaPosterCard,
  type MediaPosterCardSize,
  type MediaPosterDisplayField,
  type MediaPosterDisplaySettings,
  type MediaPosterImageType,
} from '#/components/media-poster-card'
import { Badge } from '#/components/ui/badge'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '#/components/ui/breadcrumb'
import { Button } from '#/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '#/components/ui/dialog'
import { Field, FieldLabel } from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import { Label } from '#/components/ui/label'
import { NativeSelect, NativeSelectOption } from '#/components/ui/native-select'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '#/components/ui/select'
import { Switch } from '#/components/ui/switch'
import {
  DiscoveryControls,
  createDefaultDiscoveryFilters,
  type DiscoveryFilters,
} from '#/features/discovery/controls'
import type {
  LibraryHierarchyBreadcrumb,
  LibraryHierarchyContext,
  LibraryHierarchyNode,
} from '#/lib/mibo-api'
import { createAuthedMiboApi, miboQueryKeys } from '#/lib/mibo-query'
import { cn } from '#/lib/utils'
import { useAuthStore } from '#/stores/auth-store'
import {
  ChevronLeftIcon,
  FolderIcon,
  FolderOpenIcon,
  HouseIcon,
  LoaderCircleIcon,
  Settings2Icon,
} from 'lucide-react'

const LIBRARY_PAGE_SIZE_OPTIONS = [24, 48, 60, 96] as const
export const DEFAULT_LIBRARY_BROWSER_PAGE_SIZE = 60
const LIBRARY_BROWSER_DISPLAY_SETTINGS_STORAGE_KEY =
  'mibo:library-browser-display-settings:v1'

export function isLibraryBrowserPageSize(value: number) {
  return LIBRARY_PAGE_SIZE_OPTIONS.some((option) => option === value)
}

type LibraryBrowserProps = {
  libraryId?: number
  browsePath?: string
  page: number
  pageSize: number
  filters: DiscoveryFilters
  scrollContainerRef: RefObject<HTMLDivElement | null>
  onPaginationChange: (next: { page?: number; pageSize?: number }) => void
  onBrowseTargetChange: (next: { libraryId?: number; path?: string }) => void
  onFiltersChange: (
    next: DiscoveryFilters,
    options?: { resetPage?: boolean }
  ) => void
}

export default function LibraryBrowser({
  libraryId,
  browsePath,
  page,
  pageSize,
  filters,
  scrollContainerRef,
  onPaginationChange,
  onBrowseTargetChange,
  onFiltersChange,
}: LibraryBrowserProps) {
  const token = useAuthStore((state) => state.auth.accessToken)
  const [displaySettings, setDisplaySettings] = useState(
    loadLibraryBrowserDisplaySettings
  )

  const browseQuery = useQuery({
    queryKey: miboQueryKeys.libraryHierarchy(
      token ?? 'guest',
      typeof libraryId === 'number' ? libraryId : 'root',
      browsePath ?? '',
      filters,
      page,
      pageSize
    ),
    enabled: !!token,
    queryFn: async () => {
      if (!token) throw new Error('当前未登录，无法加载目录浏览。')
      return createAuthedMiboApi(token).browseLibraryHierarchy({
        library_id: libraryId,
        path: browsePath,
        q: filters.q.trim() || undefined,
        type: filters.type === 'all' ? undefined : filters.type,
        genre: filters.genre.trim() || undefined,
        region: filters.region.trim() || undefined,
        year: parseOptionalInt(filters.year),
        min_rating: parseOptionalFloat(filters.minRating),
        watched_state: filters.watchedState,
        organizing_state: filters.organizingState,
        sort: filters.sort,
        sort_direction: filters.sortDirection,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      })
    },
  })

  const nodes = browseQuery.data?.items ?? []
  const total = browseQuery.data?.total ?? 0
  const currentNode = browseQuery.data?.current_node
  const breadcrumbs = browseQuery.data?.breadcrumbs ?? []
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const pageStart = total === 0 ? 0 : (page - 1) * pageSize + 1
  const pageEnd = Math.min(total, page * pageSize)
  const previousCrumb = breadcrumbs[breadcrumbs.length - 2]
  const hasActiveTitleSorting = filters.sort === 'title'
  const resetFilters = () => {
    onFiltersChange(
      createDefaultDiscoveryFilters({
        type: filters.type,
        organizingState: 'organized',
      }),
      { resetPage: true }
    )
  }
  const clearTitleSorting = () => {
    onFiltersChange(
      {
        ...filters,
        sort: 'recent',
        sortDirection: 'desc',
      },
      { resetPage: true }
    )
  }

  useEffect(() => {
    if (total > 0 && page > pageCount) {
      onPaginationChange({ page: pageCount })
    }
  }, [onPaginationChange, page, pageCount, total])

  useEffect(() => {
    window.localStorage.setItem(
      LIBRARY_BROWSER_DISPLAY_SETTINGS_STORAGE_KEY,
      JSON.stringify(displaySettings)
    )
  }, [displaySettings])

  return (
    <div className='relative min-w-0 flex-1 bg-background text-foreground'>
      <div
        ref={scrollContainerRef}
        className='h-svh overflow-x-hidden overflow-y-auto'
      >
        <section className='min-w-0 px-3 pt-6 pb-16 sm:px-6 lg:px-8'>
          <div className='mx-auto max-w-[1800px] min-w-0 space-y-6'>
            <div className='space-y-3'>
              <DiscoveryControls
                filters={filters}
                showSearch
                showType={false}
                showOrganizingState
                onChange={(next) => onFiltersChange(next, { resetPage: true })}
              />
              <div className='flex flex-wrap items-center gap-2'>
                {previousCrumb ? (
                  <Button
                    type='button'
                    variant='outline'
                    className='rounded-full'
                    disabled={browseQuery.isFetching}
                    onClick={() => {
                      onBrowseTargetChange({
                        libraryId: previousCrumb.library_id,
                        path: previousCrumb.path,
                      })
                    }}
                  >
                    <ChevronLeftIcon className='size-4' />
                    返回上一级
                  </Button>
                ) : null}
                <Button
                  type='button'
                  variant='outline'
                  className='rounded-full'
                  onClick={resetFilters}
                >
                  重置筛选
                </Button>
                {hasActiveTitleSorting ? (
                  <Button
                    type='button'
                    variant='outline'
                    className='rounded-full'
                    disabled={browseQuery.isFetching}
                    onClick={clearTitleSorting}
                  >
                    取消标题排序
                  </Button>
                ) : null}
                <LibraryBrowserDisplaySettingsDialog
                  settings={displaySettings}
                  onSettingsChange={setDisplaySettings}
                />
                <Badge
                  variant='outline'
                  className='rounded-full bg-background/60'
                >
                  {total} 项
                </Badge>
              </div>
              <LibraryBreadcrumbs
                breadcrumbs={breadcrumbs}
                onSelect={(crumb) =>
                  onBrowseTargetChange({
                    libraryId: crumb.library_id,
                    path: crumb.path,
                  })
                }
              />
            </div>

            <LibraryResults
              isLoading={browseQuery.isLoading}
              isFetching={browseQuery.isFetching}
              error={browseQuery.error?.message}
              nodes={nodes}
              currentNode={currentNode}
              displaySettings={displaySettings}
              onNavigate={onBrowseTargetChange}
              onRetry={() => void browseQuery.refetch()}
            />

            {total > 0 ? (
              <div className='flex min-h-16 flex-col gap-4 rounded-[1.5rem] border border-border/45 bg-card/65 p-4 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between'>
                <div className='flex flex-wrap items-center gap-3'>
                  <span>
                    第 {page} / {pageCount} 页，显示 {pageStart}-{pageEnd} /{' '}
                    {total} 项
                  </span>
                  {browseQuery.isFetching ? (
                    <span className='inline-flex items-center gap-1.5'>
                      <LoaderCircleIcon className='size-4 animate-spin' />
                      正在加载
                    </span>
                  ) : null}
                </div>
                <div className='flex flex-wrap items-center gap-2'>
                  <Select
                    value={String(pageSize)}
                    onValueChange={(nextPageSize) =>
                      onPaginationChange({
                        page: 1,
                        pageSize: Number(nextPageSize),
                      })
                    }
                  >
                    <SelectTrigger
                      size='sm'
                      className='rounded-full bg-background/70'
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {LIBRARY_PAGE_SIZE_OPTIONS.map((option) => (
                        <SelectItem key={option} value={String(option)}>
                          {option} 项
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <PageJumpForm
                    key={page}
                    page={page}
                    pageCount={pageCount}
                    onJump={(nextPage) =>
                      onPaginationChange({ page: nextPage })
                    }
                  />
                  <Button
                    type='button'
                    variant='outline'
                    className='rounded-full'
                    disabled={page <= 1 || browseQuery.isFetching}
                    onClick={() => onPaginationChange({ page: page - 1 })}
                  >
                    上一页
                  </Button>
                  <Button
                    type='button'
                    variant='outline'
                    className='rounded-full'
                    disabled={page >= pageCount || browseQuery.isFetching}
                    onClick={() => onPaginationChange({ page: page + 1 })}
                  >
                    下一页
                  </Button>
                </div>
              </div>
            ) : null}
          </div>
        </section>
      </div>
    </div>
  )
}

function PageJumpForm({
  page,
  pageCount,
  onJump,
}: {
  page: number
  pageCount: number
  onJump: (page: number) => void
}) {
  const [pageJumpDraft, setPageJumpDraft] = useState(String(page))

  const submitPageJump = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const nextPage = Number.parseInt(pageJumpDraft, 10)
    if (!Number.isFinite(nextPage)) {
      setPageJumpDraft(String(page))
      return
    }

    const clampedPage = Math.min(pageCount, Math.max(1, nextPage))
    setPageJumpDraft(String(clampedPage))
    if (clampedPage !== page) onJump(clampedPage)
  }

  return (
    <form className='flex items-center gap-2' onSubmit={submitPageJump}>
      <Input
        type='number'
        aria-label='跳转页码'
        min={1}
        max={pageCount}
        inputMode='numeric'
        value={pageJumpDraft}
        onChange={(event) => setPageJumpDraft(event.currentTarget.value)}
        className='w-20 rounded-full bg-background/70 text-center'
      />
      <Button type='submit' variant='outline' className='rounded-full'>
        跳转
      </Button>
    </form>
  )
}

function LibraryResults({
  isLoading,
  isFetching,
  error,
  nodes,
  currentNode,
  displaySettings,
  onNavigate,
  onRetry,
}: {
  isLoading: boolean
  isFetching: boolean
  error?: string
  nodes: LibraryHierarchyNode[]
  currentNode?: LibraryHierarchyContext
  displaySettings: MediaPosterDisplaySettings
  onNavigate: (next: { libraryId?: number; path?: string }) => void
  onRetry: () => void
}) {
  if (isLoading) {
    return (
      <div className='flex min-h-80 items-center justify-center rounded-[2rem] border border-border/50 bg-card/70 shadow-sm backdrop-blur-xl'>
        <div className='flex items-center gap-3 rounded-full border border-border/50 bg-background/70 px-5 py-3 text-sm text-muted-foreground'>
          <LoaderCircleIcon className='size-4 animate-spin' />
          正在加载目录浏览
        </div>
      </div>
    )
  }

  if (error && nodes.length === 0) {
    return (
      <div className='space-y-4 rounded-[2rem] border border-destructive/30 bg-destructive/10 px-6 py-10 text-center text-sm text-destructive shadow-sm backdrop-blur-xl'>
        <p>{error}</p>
        <Button
          type='button'
          variant='outline'
          className='rounded-full'
          onClick={onRetry}
        >
          重新加载
        </Button>
      </div>
    )
  }

  if (nodes.length === 0) {
    return (
      <div className='rounded-[2rem] border border-border/50 bg-card/70 px-6 py-14 text-center shadow-sm backdrop-blur-xl'>
        <h2 className='text-2xl font-semibold tracking-tight'>
          暂无可显示内容
        </h2>
        <p className='mx-auto mt-3 max-w-xl text-sm leading-7 text-muted-foreground'>
          {currentNode?.node_kind === 'library' || currentNode?.path
            ? '当前目录下没有可显示内容，可以调整筛选条件或返回上一级。'
            : '当前筛选条件下没有内容，可以调整类型、进度、排序或关键词后重试。'}
        </p>
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      {error ? (
        <div className='rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive'>
          {error}
        </div>
      ) : null}
      <div
        className={cn(
          'grid gap-4',
          gridClassForDisplaySettings(displaySettings),
          isFetching && 'opacity-80'
        )}
      >
        {nodes.map((node) => (
          <LibraryNodeCard
            key={node.node_id}
            node={node}
            displaySettings={displaySettings}
            onNavigate={onNavigate}
          />
        ))}
      </div>
    </div>
  )
}

function LibraryNodeCard({
  node,
  displaySettings,
  onNavigate,
}: {
  node: LibraryHierarchyNode
  displaySettings: MediaPosterDisplaySettings
  onNavigate: (next: { libraryId?: number; path?: string }) => void
}) {
  if (node.node_kind === 'item' && node.item) {
    return (
      <MediaPosterCard
        item={node.item}
        layout='grid'
        imageAspect={imageAspectForView(displaySettings.imageType)}
        displaySettings={displaySettings}
      />
    )
  }

  const isLibrary = node.node_kind === 'library'
  return (
    <button
      type='button'
      onClick={() =>
        onNavigate({
          libraryId: node.library_id,
          path: isLibrary ? undefined : node.path,
        })
      }
      className='group flex min-h-72 w-full flex-col justify-between overflow-hidden rounded-[1.35rem] border border-border/40 bg-card/75 p-4 text-left shadow-lg transition-transform duration-200 hover:-translate-y-1 hover:border-border/70'
    >
      <div className='space-y-3'>
        <div className='flex items-start justify-between gap-3'>
          <div className='rounded-2xl bg-muted/70 p-3 text-muted-foreground'>
            {isLibrary ? (
              <FolderOpenIcon className='size-6' />
            ) : (
              <FolderIcon className='size-6' />
            )}
          </div>
          <span className='rounded-full border border-border/60 bg-background/60 px-3 py-1 text-xs text-muted-foreground'>
            {isLibrary ? '媒体库' : '文件夹'}
          </span>
        </div>
        <div>
          <div className='line-clamp-2 text-lg font-semibold tracking-tight text-foreground'>
            {node.title}
          </div>
          <div className='mt-2 text-sm leading-6 text-muted-foreground'>
            {isLibrary
              ? node.library_name || `媒体库 #${node.library_id}`
              : node.path || '根目录'}
          </div>
        </div>
      </div>
      <div className='flex flex-wrap items-center gap-2 text-sm text-muted-foreground'>
        {typeof node.child_count === 'number' && node.child_count > 0 ? (
          <span>{node.child_count} 个子文件夹</span>
        ) : null}
        {typeof node.item_count === 'number' && node.item_count > 0 ? (
          <span>{node.item_count} 个内容项</span>
        ) : null}
      </div>
    </button>
  )
}

function LibraryBreadcrumbs({
  breadcrumbs,
  onSelect,
}: {
  breadcrumbs: LibraryHierarchyBreadcrumb[]
  onSelect: (crumb: LibraryHierarchyBreadcrumb) => void
}) {
  if (breadcrumbs.length === 0) return null

  return (
    <Breadcrumb className='min-w-0 px-4 py-3'>
      <BreadcrumbList>
        {breadcrumbs.map((crumb, index) => {
          const isCurrent = index === breadcrumbs.length - 1

          return (
            <Fragment key={crumb.node_id}>
              <BreadcrumbItem className='min-w-0'>
                {isCurrent ? (
                  <BreadcrumbPage className='inline-flex max-w-full min-w-0 items-center gap-2 truncate'>
                    {index === 0 ? (
                      <HouseIcon className='size-4 shrink-0' />
                    ) : null}
                    <span className='truncate'>{crumb.title}</span>
                  </BreadcrumbPage>
                ) : (
                  <BreadcrumbLink asChild>
                    <button
                      type='button'
                      className='inline-flex max-w-full min-w-0 items-center gap-2 truncate'
                      onClick={() => onSelect(crumb)}
                    >
                      {index === 0 ? (
                        <HouseIcon className='size-4 shrink-0' />
                      ) : null}
                      <span className='truncate'>{crumb.title}</span>
                    </button>
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
              {isCurrent ? null : <BreadcrumbSeparator />}
            </Fragment>
          )
        })}
      </BreadcrumbList>
    </Breadcrumb>
  )
}

const IMAGE_TYPE_OPTIONS: Array<{
  value: MediaPosterImageType
  label: string
}> = [
  { value: 'primary', label: '海报' },
  { value: 'banner', label: '横幅图' },
  { value: 'disc', label: '光盘封面' },
  { value: 'logo', label: '徽标' },
  { value: 'thumb', label: '缩略图' },
  { value: 'list', label: '列表' },
  { value: 'datagrid', label: '表格' },
]

const CARD_SIZE_OPTIONS: Array<{
  value: MediaPosterCardSize
  label: string
}> = [
  { value: 'extrasmall', label: '超小' },
  { value: 'smaller', label: '特小' },
  { value: 'small', label: '小' },
  { value: 'normal', label: '中' },
  { value: 'default', label: '默认' },
  { value: 'large', label: '大' },
  { value: 'larger', label: '特大' },
  { value: 'extralarge', label: '超大' },
]

const DISPLAY_FIELD_OPTIONS: Array<{
  field: MediaPosterDisplayField
  label: string
}> = [
  { field: 'Name', label: '标题' },
  { field: 'OriginalTitle', label: '原标题' },
  { field: 'SortName', label: '排序标题' },
  { field: 'CommunityRating', label: 'IMDb 评分' },
  { field: 'CriticRating', label: '影评人评分' },
  { field: 'OfficialRating', label: '家长评分' },
  { field: 'ProductionYear', label: '年份' },
  { field: 'PremiereDate', label: '发行日期' },
  { field: 'Runtime', label: '播放时长' },
  { field: 'Genres', label: '类型' },
  { field: 'Director', label: '导演' },
  { field: 'Tags', label: '标签' },
  { field: 'Studios', label: '工作室' },
  { field: 'Tagline', label: '宣传语' },
  { field: 'Overview', label: '概要' },
  { field: 'DatePlayed', label: '播放日期' },
  { field: 'Played', label: '已播放' },
  { field: 'DateCreated', label: '创建日期' },
  { field: 'IsFavorite', label: '收藏' },
]

function LibraryBrowserDisplaySettingsDialog({
  settings,
  onSettingsChange,
}: {
  settings: MediaPosterDisplaySettings
  onSettingsChange: (next: MediaPosterDisplaySettings) => void
}) {
  const updateField = (field: MediaPosterDisplayField, checked: boolean) => {
    onSettingsChange({
      ...settings,
      fields: {
        ...settings.fields,
        [field]: checked,
      },
    })
  }

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button type='button' variant='outline' className='rounded-full'>
          <Settings2Icon data-icon='inline-start' />
          展示设置
        </Button>
      </DialogTrigger>
      <DialogContent className='grid max-h-[min(820px,calc(100svh-2rem))] grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>展示设置</DialogTitle>
          <DialogDescription>
            控制目录浏览卡片的图像样式、尺寸和可见字段。
          </DialogDescription>
        </DialogHeader>
        <form
          className='min-h-0 space-y-6 overflow-y-auto pr-1'
          onSubmit={(event) => event.preventDefault()}
        >
          <fieldset className='space-y-4'>
            <legend className='sr-only'>设置</legend>
            <div className='grid gap-4 sm:grid-cols-2'>
              <Field>
                <FieldLabel htmlFor='library-browser-display-image-type'>
                  视图
                </FieldLabel>
                <NativeSelect
                  id='library-browser-display-image-type'
                  value={settings.imageType}
                  onChange={(event) =>
                    onSettingsChange({
                      ...settings,
                      imageType: event.target.value as MediaPosterImageType,
                    })
                  }
                  className='w-full'
                >
                  {IMAGE_TYPE_OPTIONS.map((option) => (
                    <NativeSelectOption key={option.value} value={option.value}>
                      {option.label}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor='library-browser-display-card-size'>
                  图像大小
                </FieldLabel>
                <NativeSelect
                  id='library-browser-display-card-size'
                  value={settings.cardSize}
                  onChange={(event) =>
                    onSettingsChange({
                      ...settings,
                      cardSize: event.target.value as MediaPosterCardSize,
                    })
                  }
                  className='w-full'
                >
                  {CARD_SIZE_OPTIONS.map((option) => (
                    <NativeSelectOption key={option.value} value={option.value}>
                      {option.label}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
            </div>
          </fieldset>
          <fieldset className='space-y-3'>
            <legend className='text-sm font-medium'>显示字段</legend>
            <div className='grid gap-2 sm:grid-cols-2'>
              {DISPLAY_FIELD_OPTIONS.map((option) => (
                <DisplaySettingsSwitch
                  key={option.field}
                  label={option.label}
                  checked={settings.fields[option.field]}
                  onCheckedChange={(checked) =>
                    updateField(option.field, checked)
                  }
                />
              ))}
            </div>
          </fieldset>
        </form>
        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() =>
              onSettingsChange(DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS)
            }
          >
            恢复默认
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function DisplaySettingsSwitch({
  label,
  checked,
  onCheckedChange,
}: {
  label: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <Label className='flex min-h-10 justify-between rounded-lg border border-border/60 px-3 py-2'>
      <span>{label}</span>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </Label>
  )
}

function loadLibraryBrowserDisplaySettings(): MediaPosterDisplaySettings {
  if (typeof window === 'undefined') {
    return DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS
  }
  try {
    const raw = window.localStorage.getItem(
      LIBRARY_BROWSER_DISPLAY_SETTINGS_STORAGE_KEY
    )
    if (!raw) return DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS
    return mergeLibraryBrowserDisplaySettings(JSON.parse(raw))
  } catch {
    return DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS
  }
}

function mergeLibraryBrowserDisplaySettings(
  value: Partial<MediaPosterDisplaySettings>
): MediaPosterDisplaySettings {
  return {
    ...DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS,
    ...value,
    fields: {
      ...DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS.fields,
      ...(value.fields ?? {}),
    },
  }
}

function gridClassForDisplaySettings(settings: MediaPosterDisplaySettings) {
  if (settings.imageType === 'list' || settings.imageType === 'datagrid') {
    return 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-3'
  }
  switch (settings.cardSize) {
    case 'extrasmall':
      return 'grid-cols-3 sm:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8 2xl:grid-cols-10'
    case 'smaller':
      return 'grid-cols-3 sm:grid-cols-4 lg:grid-cols-5 xl:grid-cols-7 2xl:grid-cols-8'
    case 'small':
      return 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-7'
    case 'normal':
      return 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6'
    case 'large':
      return 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-4 2xl:grid-cols-5'
    case 'larger':
      return 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-4'
    case 'extralarge':
      return 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-3'
    case 'default':
    default:
      return 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6'
  }
}

function imageAspectForView(
  imageType: MediaPosterImageType
): 'poster' | 'landscape' {
  return imageType === 'banner' ||
    imageType === 'thumb' ||
    imageType === 'list' ||
    imageType === 'datagrid'
    ? 'landscape'
    : 'poster'
}

function parseOptionalInt(value: string) {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) ? parsed : undefined
}

function parseOptionalFloat(value: string) {
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : undefined
}
