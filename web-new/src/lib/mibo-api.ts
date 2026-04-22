export type ApiErrorShape = {
  code: string
  message: string
}

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

export type MediaSource = {
  id: number
  name: string
  provider: string
  storage_ref: string
  root_path: string
  created_at: string
  updated_at: string
}

export type BrowseTypeFilter = 'all' | 'movie' | 'show'

export type BrowseSort = 'recent' | 'title' | 'year' | 'watch_status'

export type BrowseFilters = {
  type: BrowseTypeFilter
  year: number | null
  sort: BrowseSort
}

export const DEFAULT_BROWSE_FILTERS: BrowseFilters = {
  type: 'all',
  year: null,
  sort: 'recent',
}

type CreateMediaSourceInput = {
  provider: string
  name: string
  root_path: string
  config?: {
    openlist?: {
      base_url: string
      username?: string
      password?: string
    }
  }
}

type CreateLibraryInput = {
  name: string
  type: string
  media_source_id: number
  root_path: string
}

type ApiOptions = {
  baseUrl: string
  token?: string | null
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

    const response = await fetch(`${baseUrl}${pathname}`, {
      ...init,
      headers,
    })

    const payload = (await response.json()) as Envelope<T>

    if (!response.ok || payload.error) {
      throw new ApiError(
        response.status,
        payload.error ?? {
          code: 'request_failed',
          message: `请求失败，状态码 ${response.status}`,
        }
      )
    }

    if (payload.data === undefined) {
      throw new ApiError(response.status, {
        code: 'missing_payload',
        message: '服务端返回了空数据',
      })
    }

    return payload.data
  }

  return {
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
    getSetupStatus() {
      return request<SetupStatus>('/api/v1/setup/status')
    },
    listMediaSources() {
      return request<MediaSource[]>('/api/v1/media-sources')
    },
    createMediaSource(input: CreateMediaSourceInput) {
      return request<MediaSource>('/api/v1/media-sources', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    createLibrary(input: CreateLibraryInput) {
      return request<{ id: number }>('/api/v1/libraries', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
  }
}
