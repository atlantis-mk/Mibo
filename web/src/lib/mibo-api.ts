export type ApiErrorShape = {
  code: string
  message: string
}

type Envelope<T> = {
  request_id: string
  data?: T
  error?: ApiErrorShape
}

export type User = {
  id: number
  username: string
  role: string
  created_at: string
  updated_at: string
}

export type LoginResult = {
  token: string
  expires_at: string
  user: User
}

export type Library = {
  id: number
  name: string
  type: string
  media_source_id: number
  root_path: string
  status: string
  scanner_enabled: boolean
}

export type LibraryDetail = Library & {
  media_items_count: number
  media_files_count: number
}

export type OpenListMediaSourceConfig = {
  base_url: string
  username?: string
  password?: string
  token?: string
  timeout?: string
  insecure_skip?: boolean
}

export type MediaSourceConfigInput = {
  openlist?: OpenListMediaSourceConfig
}

export type OpenListMediaSourceConfigView = {
  base_url: string
  username?: string
  timeout?: string
  insecure_skip: boolean
  has_password: boolean
  has_token: boolean
}

export type MediaSourceConfigView = {
  openlist?: OpenListMediaSourceConfigView
}

export type MediaSource = {
  id: number
  name: string
  provider: string
  storage_ref: string
  root_path: string
  config?: MediaSourceConfigView
  capabilities_json: string
  created_at: string
  updated_at: string
}

export type StorageBrowseItem = {
  name: string
  path: string
  is_dir: boolean
  size: number
  modified?: string
}

export type StorageBrowseResult = {
  provider: string
  root_path: string
  current_path: string
  parent_path?: string
  items: StorageBrowseItem[]
}

export type OpenListTestResult = {
  status: string
  provider: string
  message: string
  root_path: string
}

export type MediaItem = {
  id: number
  library_id: number
  type: string
  title: string
  original_title: string
  series_title: string
  overview: string
  poster_url: string
  logo_url?: string
  backdrop_url: string
  year?: number
  vote_average?: number
  release_date: string
  runtime_seconds?: number
  season_number?: number
  episode_number?: number
  source_path: string
  match_status: string
  metadata_provider: string
  external_id: string
  metadata_confidence?: number
  status: string
  created_at: string
  updated_at: string
}

export type Track = {
  codec: string
  language: string
  title: string
  channels?: number
}

export type MediaFile = {
  id: number
  library_id: number
  media_item_id?: number
  storage_path: string
  container: string
  size_bytes: number
  fingerprint: string
  probe_status: string
  probe_error: string
  duration_seconds?: number
  bit_rate?: number
  width?: number
  height?: number
  video_codec: string
  audio_tracks: Track[]
  subtitle_tracks: Track[]
}

export type Person = {
  name: string
  role: string
  avatar_url: string
}

export type Trailer = {
  provider: string
  site: string
  key: string
  name: string
  type: string
  official: boolean
  language: string
  watch_url: string
  embed_url: string
  thumbnail: string
}

export type MetadataSearchCandidate = {
  provider: string
  media_type: string
  external_id: string
  title: string
  original_title: string
  overview: string
  poster_url: string
  backdrop_url: string
  release_date: string
  year?: number
  confidence: number
}

export type MediaItemDetail = MediaItem & {
  series_tmdb_id?: number
  series_title_display: string
  default_season_number?: number
  genres: string[]
  cast: Person[]
  directors: Person[]
  trailer?: Trailer
  files: MediaFile[]
}

export type ProgressState = {
  user_id: number
  media_item_id: number
  media_file_id?: number
  position_seconds: number
  duration_seconds?: number
  watched: boolean
  completed_at?: string
  last_played_at?: string
}

export type ProgressEntry = ProgressState & {
  media_item: MediaItem
}

export type LatestByLibrarySection = {
  library_id: number
  library_name: string
  items: MediaItem[]
}

export type DiscoveryQuery = {
  scope?: 'all' | 'library'
  library_id?: number
  q?: string
  type?: 'all' | 'movie' | 'show'
  genre?: string
  region?: string
  year?: number
  min_rating?: number
  watched_state?: 'all' | 'unwatched' | 'in_progress' | 'watched'
  sort?: 'recent' | 'title' | 'year' | 'watch_status'
  limit?: number
}

export type DiscoveryItem = {
  item: MediaItem
  watched_state: string
}

export type SearchResult = {
  item: MediaItem
  watched_state: string
  highlight: string
}

export type SearchHistoryEntry = {
  id: number
  query: string
  type_filter: string
  genre: string
  region: string
  year?: number
  min_rating?: number
  watched_state: string
  sort: 'recent' | 'title' | 'year' | 'watch_status'
  last_used_at: string
}

export type HomeDiscovery = {
  continue_watching: ProgressEntry[]
  recently_played: ProgressEntry[]
  latest_by_library: LatestByLibrarySection[]
}

export type ClientProfile = 'web' | 'mobile' | 'tv'

export type PlaybackCheck = {
  code: string
  status: string
  message: string
}

export type DecisionReason = {
  code: string
  category: string
  message: string
}

export type PlaybackDecision = {
  kind: 'direct' | 'fallback' | 'unplayable'
  client_profile: ClientProfile
  selected_by: string
  fallback_kind?: string
  reasons: DecisionReason[]
}

export type PlaybackSource = {
  media_item_id: number
  media_file_id: number
  title: string
  type: string
  container: string
  url: string
  direct: boolean
  size_bytes: number
  runtime_seconds?: number
  video_codec: string
  width?: number
  height?: number
  audio_tracks: Track[]
  subtitle_tracks: Track[]
  checks: PlaybackCheck[]
  playable: boolean
  decision: PlaybackDecision
}

export type Job = {
  id: number
  job_key: string
  kind: string
  status: string
  payload_json: string
  error_message: string
  attempts: number
  available_at: string
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export type ScheduleFrequencyKind = 'daily' | 'weekly' | 'monthly'

export type ScheduleScopeKind = 'global' | 'library'

export type ScheduleRunStatus = 'queued' | 'running' | 'completed' | 'failed'

export type ScheduleFrequency = {
  kind: ScheduleFrequencyKind
  time_of_day: string
  weekday?: number
  day_of_month?: number
}

export type ScheduleRun = {
  id: number
  schedule_id: number
  status: ScheduleRunStatus
  job_id?: number
  error_summary: string
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export type Schedule = {
  id: number
  name: string
  kind: string
  scope_kind: ScheduleScopeKind
  library_id?: number
  frequency: ScheduleFrequency
  enabled: boolean
  next_run_at?: string
  latest_run_status?: ScheduleRunStatus | ''
  latest_run_message: string
  latest_job_id?: number
  latest_run_started_at?: string
  latest_run_finished_at?: string
  recent_runs?: ScheduleRun[]
  created_at: string
  updated_at: string
}

export type ScheduleMutationInput = {
  name: string
  kind: string
  scope_kind: ScheduleScopeKind
  library_id?: number
  enabled?: boolean
  frequency: ScheduleFrequency
}

export type ScheduleRunNowResult = {
  run: ScheduleRun
  job: Job
}

type ApiOptions = {
  baseUrl: string
  token?: string | null
}

export const TOKEN_STORAGE_KEY = 'mibo-web-token'

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, error: ApiErrorShape) {
    super(error.message)
    this.name = 'ApiError'
    this.status = status
    this.code = error.code
  }
}

export function getApiBaseUrl() {
  return (
    (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(
      /\/$/,
      '',
    ) ?? 'http://10.0.0.33:8080'
  )
}

export function createMiboApi(options: ApiOptions) {
  const baseUrl = options.baseUrl.replace(/\/$/, '')

  async function request<T>(pathname: string, init?: RequestInit): Promise<T> {
    const headers = new Headers(init?.headers)

    if (!headers.has('Content-Type') && init?.body !== undefined) {
      headers.set('Content-Type', 'application/json')
    }

    if (options.token) {
      headers.set('Authorization', `Bearer ${options.token}`)
    }

    let response: Response
    try {
      response = await fetch(`${baseUrl}${pathname}`, {
        ...init,
        headers,
      })
    } catch {
      throw new ApiError(0, {
        code: 'network_error',
        message: '无法连接后端服务，请确认 Mibo 服务已启动。',
      })
    }

    let payload: Envelope<T> | null = null
    try {
      payload = (await response.json()) as Envelope<T>
    } catch {
      if (!response.ok) {
        throw new ApiError(response.status, {
          code: 'request_failed',
          message: `请求失败，状态码 ${response.status}`,
        })
      }
    }

    if (!response.ok || payload?.error) {
      throw new ApiError(
        response.status,
        payload?.error ?? {
          code: 'request_failed',
          message: `请求失败，状态码 ${response.status}`,
        },
      )
    }

    if (payload?.data === undefined) {
      throw new ApiError(response.status, {
        code: 'missing_payload',
        message: '服务端返回了空数据',
      })
    }

    return payload.data
  }

  return {
    login(username: string, password: string) {
      return request<LoginResult>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      })
    },
    logout() {
      return request<{ status: string }>('/api/v1/auth/logout', {
        method: 'POST',
      })
    },
    me() {
      return request<User>('/api/v1/me')
    },
    listMediaSources() {
      return request<MediaSource[]>('/api/v1/media-sources')
    },
    browseStorageProvider(provider: string, path?: string) {
      const query = path ? `?path=${encodeURIComponent(path)}` : ''
      return request<StorageBrowseResult>(
        `/api/v1/storage/providers/${provider}/browse${query}`,
      )
    },
    browseOpenList(input: {
      path?: string
      config: OpenListMediaSourceConfig
    }) {
      return request<StorageBrowseResult>('/api/v1/storage/openlist/browse', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    testOpenListConnection(input: { config: OpenListMediaSourceConfig }) {
      return request<OpenListTestResult>('/api/v1/storage/openlist/test', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    createMediaSource(input: {
      provider: string
      name: string
      root_path: string
      storage_ref?: string
      config?: MediaSourceConfigInput
    }) {
      return request<MediaSource>('/api/v1/media-sources', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateMediaSource(
      mediaSourceId: number,
      input: {
        name: string
        root_path: string
        storage_ref?: string
        config?: MediaSourceConfigInput
      },
    ) {
      return request<MediaSource>(`/api/v1/media-sources/${mediaSourceId}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      })
    },
    deleteMediaSource(mediaSourceId: number) {
      return request<{ id: number; status: string; type: string }>(
        `/api/v1/media-sources/${mediaSourceId}`,
        {
          method: 'DELETE',
        },
      )
    },
    browseMediaSource(mediaSourceId: number, path?: string) {
      const query = path ? `?path=${encodeURIComponent(path)}` : ''
      return request<StorageBrowseResult>(
        `/api/v1/media-sources/${mediaSourceId}/browse${query}`,
      )
    },
    listLibraries() {
      return request<Library[]>('/api/v1/libraries')
    },
    getLibrary(libraryId: number) {
      return request<LibraryDetail>(`/api/v1/libraries/${libraryId}`)
    },
    createLibrary(input: {
      name: string
      type: string
      media_source_id: number
      root_path: string
    }) {
      return request<{ library: Library }>('/api/v1/libraries', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    deleteLibrary(libraryId: number) {
      return request<{ id: number; status: string; type: string }>(
        `/api/v1/libraries/${libraryId}`,
        {
          method: 'DELETE',
        },
      )
    },
    scanLibrary(libraryId: number) {
      return request<{ id: number }>(`/api/v1/libraries/${libraryId}/scan`, {
        method: 'POST',
      })
    },
    listLibraryItems(
      libraryId: number,
      queryOptions?: {
        type?: 'all' | 'movie' | 'show'
        year?: number
        sort?: 'recent' | 'title' | 'year' | 'watch_status'
        limit?: number
      },
    ) {
      const query = new URLSearchParams()

      if (queryOptions?.type) {
        query.set('type', queryOptions.type)
      }
      if (typeof queryOptions?.year === 'number') {
        query.set('year', String(queryOptions.year))
      }
      if (queryOptions?.sort) {
        query.set('sort', queryOptions.sort)
      }
      if (typeof queryOptions?.limit === 'number') {
        query.set('limit', String(queryOptions.limit))
      }

      const queryString = query.toString()
      return request<DiscoveryItem[]>(
        `/api/v1/libraries/${libraryId}/items${queryString ? `?${queryString}` : ''}`,
      )
    },
    discoverMedia(queryOptions?: DiscoveryQuery) {
      const query = new URLSearchParams()

      if (queryOptions?.scope) query.set('scope', queryOptions.scope)
      if (typeof queryOptions?.library_id === 'number') {
        query.set('library_id', String(queryOptions.library_id))
      }
      if (queryOptions?.q) query.set('q', queryOptions.q)
      if (queryOptions?.type) query.set('type', queryOptions.type)
      if (queryOptions?.genre) query.set('genre', queryOptions.genre)
      if (queryOptions?.region) query.set('region', queryOptions.region)
      if (typeof queryOptions?.year === 'number') {
        query.set('year', String(queryOptions.year))
      }
      if (typeof queryOptions?.min_rating === 'number') {
        query.set('min_rating', String(queryOptions.min_rating))
      }
      if (queryOptions?.watched_state) {
        query.set('watched_state', queryOptions.watched_state)
      }
      if (queryOptions?.sort) query.set('sort', queryOptions.sort)
      if (typeof queryOptions?.limit === 'number') {
        query.set('limit', String(queryOptions.limit))
      }

      const queryString = query.toString()
      return request<{ items: DiscoveryItem[] | SearchResult[] }>(
        `/api/v1/discovery${queryString ? `?${queryString}` : ''}`,
      )
    },
    listSearchHistory(limit = 8) {
      return request<SearchHistoryEntry[]>(
        `/api/v1/search/history?limit=${limit}`,
      )
    },
    getMediaItem(mediaItemId: number) {
      return request<MediaItemDetail>(`/api/v1/media-items/${mediaItemId}`)
    },
    updateMediaItemMetadata(
      mediaItemId: number,
      input: {
        title: string
        original_title?: string
        year?: number
        overview?: string
        poster_url?: string
        backdrop_url?: string
      },
    ) {
      return request<MediaItemDetail>(
        `/api/v1/media-items/${mediaItemId}/metadata`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        },
      )
    },
    searchMediaItemMetadata(
      mediaItemId: number,
      input: {
        title?: string
        year?: number
        imdb_id?: string
        tmdb_id?: string
        tvdb_id?: string
      },
    ) {
      return request<MetadataSearchCandidate[]>(
        `/api/v1/media-items/${mediaItemId}/metadata/search`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        },
      )
    },
    applyMediaItemMetadataCandidate(
      mediaItemId: number,
      input: {
        external_id: string
      },
    ) {
      return request<MediaItemDetail>(
        `/api/v1/media-items/${mediaItemId}/metadata/apply`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        },
      )
    },
    rematchMediaItem(mediaItemId: number) {
      return request<Job>(`/api/v1/media-items/${mediaItemId}/match`, {
        method: 'POST',
      })
    },
    refetchMediaItemMetadata(mediaItemId: number) {
      return request<Job>(
        `/api/v1/media-items/${mediaItemId}/metadata/refetch`,
        {
          method: 'POST',
        },
      )
    },
    listJobs(filters?: { limit?: number; status?: string; kind?: string }) {
      const query = new URLSearchParams()

      if (typeof filters?.limit === 'number') {
        query.set('limit', String(filters.limit))
      }
      if (filters?.status) {
        query.set('status', filters.status)
      }
      if (filters?.kind) {
        query.set('kind', filters.kind)
      }

      const queryString = query.toString()
      return request<Job[]>(
        `/api/v1/jobs${queryString ? `?${queryString}` : ''}`,
      )
    },
    listSchedules() {
      return request<Schedule[]>('/api/v1/schedules')
    },
    getSchedule(scheduleId: number) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}`)
    },
    createSchedule(input: ScheduleMutationInput) {
      return request<Schedule>('/api/v1/schedules', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateSchedule(scheduleId: number, input: ScheduleMutationInput) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      })
    },
    toggleSchedule(scheduleId: number, enabled: boolean) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}/toggle`, {
        method: 'POST',
        body: JSON.stringify({ enabled }),
      })
    },
    runScheduleNow(scheduleId: number) {
      return request<ScheduleRunNowResult>(`/api/v1/schedules/${scheduleId}/run`, {
        method: 'POST',
      })
    },
    listScheduleHistory(scheduleId: number) {
      return request<ScheduleRun[]>(`/api/v1/schedules/${scheduleId}/history`)
    },
    getPlayback(
      mediaItemId: number,
      playbackOptions: {
        mediaFileId?: number
        clientProfile: ClientProfile
      },
    ) {
      const query = new URLSearchParams({
        client_profile: playbackOptions.clientProfile,
      })

      if (typeof playbackOptions.mediaFileId === 'number') {
        query.set('file_id', String(playbackOptions.mediaFileId))
      }

      return request<PlaybackSource>(
        `/api/v1/media-items/${mediaItemId}/playback?${query.toString()}`,
      )
    },
    getMediaItemProgress(mediaItemId: number) {
      return request<ProgressState>(
        `/api/v1/media-items/${mediaItemId}/progress`,
      )
    },
    updateProgress(input: {
      media_item_id: number
      media_file_id?: number
      position_seconds: number
      duration_seconds?: number
      completed?: boolean
    }) {
      return request<ProgressState>('/api/v1/me/progress', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    continueWatching() {
      return request<ProgressEntry[]>('/api/v1/me/continue-watching')
    },
    latestByLibrary() {
      return request<LatestByLibrarySection[]>('/api/v1/home/latest-by-library')
    },
    homeDiscovery() {
      return request<HomeDiscovery>('/api/v1/home/discovery')
    },
    recentlyAdded(limit = 5) {
      return request<MediaItem[]>(`/api/v1/home/recently-added?limit=${limit}`)
    },
  }
}
