import { Field, FieldGroup, FieldLabel } from '#/components/ui/field'
import { Input } from '#/components/ui/input'
import { NativeSelect, NativeSelectOption } from '#/components/ui/native-select'

export type DiscoverySort =
  | 'recent'
  | 'imdb_rating'
  | 'last_episode_release_date'
  | 'last_episode_added_date'
  | 'added_date'
  | 'release_date'
  | 'parental_rating'
  | 'director'
  | 'year'
  | 'critic_rating'
  | 'played_date'
  | 'runtime'
  | 'title'
  | 'random'
  | 'audience_rating'
  | 'watch_status'

export type DiscoveryFilters = {
  q: string
  type: 'all' | 'movie' | 'show'
  genre: string
  region: string
  year: string
  minRating: string
  watchedState: 'all' | 'unwatched' | 'in_progress' | 'watched'
  organizingState: 'all' | 'organized' | 'unorganized'
  sort: DiscoverySort
  sortDirection: 'asc' | 'desc'
}

type Props = {
  filters: DiscoveryFilters
  showSearch?: boolean
  showType?: boolean
  showOrganizingState?: boolean
  onChange: (next: DiscoveryFilters) => void
}

export function DiscoveryControls({
  filters,
  showSearch = true,
  showType = true,
  showOrganizingState = false,
  onChange,
}: Props) {
  return (
    <FieldGroup className='grid gap-3 lg:grid-cols-4'>
      {showSearch ? (
        <Field className='lg:col-span-2'>
          <FieldLabel htmlFor='library-search'>搜索内容库</FieldLabel>
          <Input
            id='library-search'
            value={filters.q}
            onChange={(event) =>
              onChange({ ...filters, q: event.target.value })
            }
            placeholder='搜索标题、原始标题、演员或导演'
          />
        </Field>
      ) : null}
      {showType ? (
        <Field>
          <FieldLabel htmlFor='library-type'>媒体类型</FieldLabel>
          <NativeSelect
            id='library-type'
            value={filters.type}
            onChange={(event) =>
              onChange({
                ...filters,
                type: event.target.value as DiscoveryFilters['type'],
              })
            }
            className='w-full'
          >
            <NativeSelectOption value='all'>全部类型</NativeSelectOption>
            <NativeSelectOption value='movie'>电影</NativeSelectOption>
            <NativeSelectOption value='show'>剧集</NativeSelectOption>
          </NativeSelect>
        </Field>
      ) : null}
      <Field>
        <FieldLabel htmlFor='library-watched-state'>观看进度</FieldLabel>
        <NativeSelect
          id='library-watched-state'
          value={filters.watchedState}
          onChange={(event) =>
            onChange({
              ...filters,
              watchedState: event.target
                .value as DiscoveryFilters['watchedState'],
            })
          }
          className='w-full'
        >
          <NativeSelectOption value='all'>全部进度</NativeSelectOption>
          <NativeSelectOption value='unwatched'>未看</NativeSelectOption>
          <NativeSelectOption value='in_progress'>观看中</NativeSelectOption>
          <NativeSelectOption value='watched'>已看</NativeSelectOption>
        </NativeSelect>
      </Field>
      {showOrganizingState ? (
        <Field>
          <FieldLabel htmlFor='library-organizing-state'>整理状态</FieldLabel>
          <NativeSelect
            id='library-organizing-state'
            value={filters.organizingState}
            onChange={(event) =>
              onChange({
                ...filters,
                organizingState: event.target
                  .value as DiscoveryFilters['organizingState'],
              })
            }
            className='w-full'
          >
            <NativeSelectOption value='all'>全部状态</NativeSelectOption>
            <NativeSelectOption value='organized'>已整理</NativeSelectOption>
            <NativeSelectOption value='unorganized'>未整理</NativeSelectOption>
          </NativeSelect>
        </Field>
      ) : null}
      <Field>
        <FieldLabel htmlFor='library-sort'>排序字段</FieldLabel>
        <NativeSelect
          id='library-sort'
          value={filters.sort}
          onChange={(event) =>
            onChange({
              ...filters,
              sort: event.target.value as DiscoveryFilters['sort'],
            })
          }
          className='w-full'
        >
          <NativeSelectOption value='imdb_rating'>IMDb 评分</NativeSelectOption>
          <NativeSelectOption value='last_episode_release_date'>
            上次发布集日期
          </NativeSelectOption>
          <NativeSelectOption value='last_episode_added_date'>
            上次添加集日期
          </NativeSelectOption>
          <NativeSelectOption value='added_date'>加入日期</NativeSelectOption>
          <NativeSelectOption value='release_date'>发行日期</NativeSelectOption>
          <NativeSelectOption value='parental_rating'>
            家长评分
          </NativeSelectOption>
          <NativeSelectOption value='director'>导演</NativeSelectOption>
          <NativeSelectOption value='title'>标题</NativeSelectOption>
          <NativeSelectOption value='year'>年份</NativeSelectOption>
          <NativeSelectOption value='critic_rating'>
            影评人评分
          </NativeSelectOption>
          <NativeSelectOption value='played_date'>播放日期</NativeSelectOption>
          <NativeSelectOption value='runtime'>播放时长</NativeSelectOption>
          <NativeSelectOption value='random'>随机</NativeSelectOption>
          <NativeSelectOption value='audience_rating'>
            观众评分
          </NativeSelectOption>
          <NativeSelectOption value='watch_status'>观看状态</NativeSelectOption>
          <NativeSelectOption value='recent'>最近加入</NativeSelectOption>
        </NativeSelect>
      </Field>
      <Field>
        <FieldLabel htmlFor='library-sort-direction'>排序方向</FieldLabel>
        <NativeSelect
          id='library-sort-direction'
          value={filters.sortDirection}
          onChange={(event) =>
            onChange({
              ...filters,
              sortDirection: event.target
                .value as DiscoveryFilters['sortDirection'],
            })
          }
          className='w-full'
        >
          <NativeSelectOption value='asc'>升序</NativeSelectOption>
          <NativeSelectOption value='desc'>降序</NativeSelectOption>
        </NativeSelect>
      </Field>
      <Field>
        <FieldLabel htmlFor='library-genre'>类型筛选</FieldLabel>
        <Input
          id='library-genre'
          value={filters.genre}
          onChange={(event) =>
            onChange({ ...filters, genre: event.target.value })
          }
          placeholder='类型，例如 Drama'
        />
      </Field>
      <Field>
        <FieldLabel htmlFor='library-region'>地区筛选</FieldLabel>
        <Input
          id='library-region'
          value={filters.region}
          onChange={(event) =>
            onChange({ ...filters, region: event.target.value })
          }
          placeholder='地区，例如 US'
        />
      </Field>
      <Field>
        <FieldLabel htmlFor='library-year'>年份筛选</FieldLabel>
        <Input
          id='library-year'
          value={filters.year}
          onChange={(event) =>
            onChange({ ...filters, year: event.target.value })
          }
          placeholder='年份'
          inputMode='numeric'
        />
      </Field>
      <Field>
        <FieldLabel htmlFor='library-min-rating'>最低评分筛选</FieldLabel>
        <Input
          id='library-min-rating'
          value={filters.minRating}
          onChange={(event) =>
            onChange({ ...filters, minRating: event.target.value })
          }
          placeholder='最低评分 0-10'
          inputMode='decimal'
        />
      </Field>
    </FieldGroup>
  )
}

export function createDefaultDiscoveryFilters(
  initial?: Partial<DiscoveryFilters>
): DiscoveryFilters {
  return {
    q: initial?.q ?? '',
    type: initial?.type ?? 'all',
    genre: initial?.genre ?? '',
    region: initial?.region ?? '',
    year: initial?.year ?? '',
    minRating: initial?.minRating ?? '',
    watchedState: initial?.watchedState ?? 'all',
    organizingState: initial?.organizingState ?? 'all',
    sort: initial?.sort ?? 'recent',
    sortDirection: initial?.sortDirection ?? 'desc',
  }
}
