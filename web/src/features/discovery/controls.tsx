import { Input } from '#/components/ui/input'

export type DiscoveryFilters = {
  q: string
  type: 'all' | 'movie' | 'show'
  genre: string
  region: string
  year: string
  minRating: string
  watchedState: 'all' | 'unwatched' | 'in_progress' | 'watched'
  sort: 'recent' | 'title' | 'year' | 'watch_status'
  sortDirection: 'asc' | 'desc'
}

type Props = {
  filters: DiscoveryFilters
  showSearch?: boolean
  onChange: (next: DiscoveryFilters) => void
}

export function DiscoveryControls({
  filters,
  showSearch = true,
  onChange,
}: Props) {
  return (
    <div className="grid gap-3 rounded-[1.5rem] border border-border/40 bg-card/70 p-4 backdrop-blur-sm lg:grid-cols-4">
      {showSearch ? (
        <Input
          value={filters.q}
          onChange={(event) => onChange({ ...filters, q: event.target.value })}
          placeholder="搜索标题、原始标题、演员或导演"
          className="lg:col-span-2"
        />
      ) : null}
      <select
        value={filters.type}
        onChange={(event) =>
          onChange({
            ...filters,
            type: event.target.value as DiscoveryFilters['type'],
          })
        }
        className="h-10 rounded-xl border border-input bg-background px-3 text-sm"
      >
        <option value="all">全部类型</option>
        <option value="movie">电影</option>
        <option value="show">剧集</option>
      </select>
      <select
        value={filters.watchedState}
        onChange={(event) =>
          onChange({
            ...filters,
            watchedState: event.target
              .value as DiscoveryFilters['watchedState'],
          })
        }
        className="h-10 rounded-xl border border-input bg-background px-3 text-sm"
      >
        <option value="all">全部进度</option>
        <option value="unwatched">未看</option>
        <option value="in_progress">观看中</option>
        <option value="watched">已看</option>
      </select>
      <select
        value={filters.sort}
        onChange={(event) =>
          onChange({
            ...filters,
            sort: event.target.value as DiscoveryFilters['sort'],
          })
        }
        className="h-10 rounded-xl border border-input bg-background px-3 text-sm"
      >
        <option value="recent">最近加入</option>
        <option value="title">标题</option>
        <option value="year">年份</option>
        <option value="watch_status">观看状态</option>
      </select>
      <select
        value={filters.sortDirection}
        onChange={(event) =>
          onChange({
            ...filters,
            sortDirection: event.target
              .value as DiscoveryFilters['sortDirection'],
          })
        }
        className="h-10 rounded-xl border border-input bg-background px-3 text-sm"
      >
        <option value="asc">升序</option>
        <option value="desc">降序</option>
      </select>
      <Input
        value={filters.genre}
        onChange={(event) =>
          onChange({ ...filters, genre: event.target.value })
        }
        placeholder="类型，例如 Drama"
      />
      <Input
        value={filters.region}
        onChange={(event) =>
          onChange({ ...filters, region: event.target.value })
        }
        placeholder="地区，例如 US"
      />
      <Input
        value={filters.year}
        onChange={(event) => onChange({ ...filters, year: event.target.value })}
        placeholder="年份"
        inputMode="numeric"
      />
      <Input
        value={filters.minRating}
        onChange={(event) =>
          onChange({ ...filters, minRating: event.target.value })
        }
        placeholder="最低评分 0-10"
        inputMode="decimal"
      />
    </div>
  )
}

export function createDefaultDiscoveryFilters(
  initial?: Partial<DiscoveryFilters>,
): DiscoveryFilters {
  return {
    q: initial?.q ?? '',
    type: initial?.type ?? 'all',
    genre: initial?.genre ?? '',
    region: initial?.region ?? '',
    year: initial?.year ?? '',
    minRating: initial?.minRating ?? '',
    watchedState: initial?.watchedState ?? 'all',
    sort: initial?.sort ?? 'recent',
    sortDirection: initial?.sortDirection ?? 'desc',
  }
}
