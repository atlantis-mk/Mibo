import { useAuthStore } from '#/stores/auth-store'

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

export type SetupStatus = {
  initialized: boolean
  can_enter_app: boolean
  has_users: boolean
  has_media_sources: boolean
  has_libraries: boolean
  user_count: number
  media_source_count: number
  library_count: number
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

export type Track = {
  codec: string
  language: string
  title: string
  channels?: number
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
  matched_query?: string
  reason_summary?: string
}

export type CatalogSelectedImage = {
  image_type: string
  url: string
  language?: string
  width?: number
  height?: number
}

export type CatalogExternalIdentity = {
  provider: string
  provider_type: string
  external_id: string
  is_primary: boolean
  source?: string
  confidence?: number
}

export type CatalogSourceEvidence = {
  source_type: string
  source_name: string
  language?: string
  external_id?: string
  confidence?: number
  fetched_at: string
  expires_at?: string
  summary?: unknown
}

export type CatalogFieldState = {
  field_key: string
  source_id?: number
  value?: unknown
  is_locked: boolean
  lock_reason?: string
  edited_by_user_id?: number
  edited_at?: string
}

export type CatalogChildSummary = {
  child_count: number
  available_count: number
  missing_count: number
  unaired_count: number
  played_count: number
  in_progress_count: number
  latest_air_date?: string
  latest_added_at?: string
}

export type CatalogAssetLink = {
  item_id: number
  role: string
  segment_index: number
  start_seconds?: number
  end_seconds?: number
  confidence?: number
  source?: string
}

export type CatalogListItem = {
  id: number
  library_id: number
  type: string
  title: string
  original_title?: string
  sort_title?: string
  overview?: string
  year?: number
  runtime_seconds?: number
  community_rating?: number
  official_rating?: string
  series_status?: string
  availability_status: string
  governance_status: string
  release_date?: string
  first_air_date?: string
  last_air_date?: string
  child_summary?: CatalogChildSummary
  selected_images?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
}

export type CatalogAssetDetail = {
  id: number
  library_id: number
  asset_type: string
  display_name?: string
  edition?: string
  quality_label?: string
  duration_seconds?: number
  status: string
  probe_status: string
  file_ids: number[]
  links: CatalogAssetLink[]
}

export type CatalogEpisodeDetail = {
  id: number
  library_id: number
  type: string
  title: string
  overview?: string
  year?: number
  parent_index_number?: number
  index_number?: number
  index_number_end?: number
  absolute_number?: number
  runtime_seconds?: number
  availability_status: string
  governance_status: string
  release_date?: string
  first_air_date?: string
  selected_images?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
  assets?: CatalogAssetDetail[]
}

export type CatalogSeasonDetail = {
  id: number
  library_id: number
  type: string
  title: string
  overview?: string
  year?: number
  index_number?: number
  runtime_seconds?: number
  availability_status: string
  governance_status: string
  child_summary?: CatalogChildSummary
  selected_images?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
  episodes?: CatalogEpisodeDetail[]
}

export type CatalogItemDetail = {
  id: number
  library_id: number
  type: string
  title: string
  original_title?: string
  sort_title?: string
  overview?: string
  year?: number
  end_year?: number
  runtime_seconds?: number
  community_rating?: number
  official_rating?: string
  series_status?: string
  availability_status: string
  governance_status: string
  release_date?: string
  first_air_date?: string
  last_air_date?: string
  child_summary?: CatalogChildSummary
  selected_images?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
  seasons?: CatalogSeasonDetail[]
  episodes?: CatalogEpisodeDetail[]
  assets?: CatalogAssetDetail[]
}

export type CatalogGovernanceWorkspace = {
  item_id: number
  library_id: number
  type: string
  title: string
  availability_status: string
  governance_status: string
  selected_images?: CatalogSelectedImage[]
  image_candidates?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
  assets?: CatalogAssetDetail[]
  recommended_children?: CatalogListItem[]
}

export type ProgressState = {
  user_id: number
  item_id?: number
  asset_id?: number
  position_seconds: number
  duration_seconds?: number
  played_percentage?: number
  play_count?: number
  watched: boolean
  completed_at?: string
  last_played_at?: string
}

export type CatalogLatestByLibrarySection = {
  library_id: number
  library_name: string
  items: CatalogListItem[]
}

export type MetadataProviderSettings = {
  configured: boolean
  api_key_masked: boolean
  base_url: string
  image_base_url?: string
  language: string
  timeout: string
  source: string
  implementation: string
}

export type MetadataSettings = {
  tmdb: MetadataProviderSettings
  tvdb: MetadataProviderSettings
}

export type MetadataProviderInput = {
  api_key?: string
  clear_api_key?: boolean
  base_url?: string
  image_base_url?: string
  language?: string
  timeout?: string
}

export type MetadataSettingsInput = {
  tmdb: MetadataProviderInput
  tvdb: MetadataProviderInput
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

export type CatalogDiscoveryResult = CatalogListItem

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
  item_id?: number
  asset_id?: number
  file_id?: number
  title: string
  type: string
  container: string
  url: string
  direct: boolean
  size_bytes: number
  runtime_seconds?: number
  quality_label?: string
  edition?: string
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

let isRedirectingToLogin = false

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

function handleUnauthorizedResponse(token?: string | null) {
  if (!token || typeof window === 'undefined') {
    return
  }

  const { pathname, search, hash } = window.location

  useAuthStore.getState().clearSession()

  if (pathname === '/login' || isRedirectingToLogin) {
    return
  }

  isRedirectingToLogin = true

  const redirect = `${pathname}${search}${hash}`
  const loginUrl = new URL('/login', window.location.origin)
  loginUrl.searchParams.set('redirect', redirect)
  window.location.replace(loginUrl.toString())
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

    if (response.status === 401) {
      handleUnauthorizedResponse(options.token)
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
    getSetupStatus() {
      return request<SetupStatus>('/api/v1/setup/status')
    },
    register(username: string, password: string) {
      return request<User>('/api/v1/auth/register', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      })
    },
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
    getMetadataSettings() {
      return request<MetadataSettings>('/api/v1/settings/metadata')
    },
    updateMetadataSettings(input: MetadataSettingsInput) {
      return request<MetadataSettings>('/api/v1/settings/metadata', {
        method: 'PUT',
        body: JSON.stringify(input),
      })
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
      return request<CatalogListItem[]>(
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
      return request<{ items: CatalogListItem[] }>(
        `/api/v1/discovery${queryString ? `?${queryString}` : ''}`,
      )
    },
    listSearchHistory(limit = 8) {
      return request<SearchHistoryEntry[]>(
        `/api/v1/search/history?limit=${limit}`,
      )
    },
    getCatalogItem(itemId: number) {
      return request<CatalogItemDetail>(`/api/v1/items/${itemId}`)
    },
    listCatalogSeriesSeasons(itemId: number) {
      return request<CatalogSeasonDetail[]>(`/api/v1/series/${itemId}/seasons`)
    },
    getCatalogGovernanceWorkspace(itemId: number) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance`,
      )
    },
    updateCatalogGovernanceField(
      itemId: number,
      input: {
        field_key: string
        value?: unknown
        lock?: boolean
        lock_reason?: string
        force?: boolean
      },
    ) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance/fields`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        },
      )
    },
    selectCatalogGovernanceImage(
      itemId: number,
      input: {
        image_type: string
        url: string
      },
    ) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance/images`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        },
      )
    },
    linkCatalogGovernanceAsset(
      workspaceItemId: number,
      assetId: number,
      input: {
        target_item_id: number
      },
    ) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${workspaceItemId}/governance/assets/${assetId}/links`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        },
      )
    },
    unlinkCatalogGovernanceAsset(
      workspaceItemId: number,
      assetId: number,
      targetItemId: number,
    ) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${workspaceItemId}/governance/assets/${assetId}/links/${targetItemId}`,
        {
          method: 'DELETE',
        },
      )
    },
    searchCatalogItemMetadata(
      itemId: number,
      input: {
        title?: string
        year?: number
        imdb_id?: string
        tmdb_id?: string
        tvdb_id?: string
      },
    ) {
      return request<MetadataSearchCandidate[]>(
        `/api/v1/items/${itemId}/metadata/search`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        },
      )
    },
    applyCatalogItemMetadataCandidate(
      itemId: number,
      input: {
        external_id: string
      },
    ) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/metadata/apply`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        },
      )
    },
    refetchCatalogItemMetadata(itemId: number) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/metadata/refetch`,
        {
          method: 'POST',
        },
      )
    },
    matchCatalogItem(itemId: number) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/match`,
        {
          method: 'POST',
        },
      )
    },
    reprobeInventoryFile(fileId: number) {
      return request<Job>(`/api/v1/inventory-files/${fileId}/probe`, {
        method: 'POST',
      })
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
      return request<ScheduleRunNowResult>(
        `/api/v1/schedules/${scheduleId}/run`,
        {
          method: 'POST',
        },
      )
    },
    listScheduleHistory(scheduleId: number) {
      return request<ScheduleRun[]>(`/api/v1/schedules/${scheduleId}/history`)
    },
    getCatalogPlayback(
      itemId: number,
      playbackOptions: {
        assetId?: number
        clientProfile: ClientProfile
      },
    ) {
      const query = new URLSearchParams({
        client_profile: playbackOptions.clientProfile,
      })

      if (typeof playbackOptions.assetId === 'number') {
        query.set('asset_id', String(playbackOptions.assetId))
      }

      return request<PlaybackSource>(
        `/api/v1/items/${itemId}/playback?${query.toString()}`,
      )
    },
    getCatalogItemProgress(itemId: number) {
      return request<ProgressState>(`/api/v1/items/${itemId}/progress`)
    },
    updateProgress(input: {
      item_id?: number
      asset_id?: number
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
      return request<unknown[]>('/api/v1/me/continue-watching')
    },
    latestByLibrary() {
      return request<CatalogLatestByLibrarySection[]>(
        '/api/v1/home/latest-by-library',
      )
    },
    recentlyAdded(limit = 5) {
      return request<CatalogListItem[]>(
        `/api/v1/home/recently-added?limit=${limit}`,
      )
    },
  }
}
