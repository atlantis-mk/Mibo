import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import type {
  MetadataProfile,
  MetadataProviderInstance,
  PluginProviderInstance,
} from '@/lib/mibo-api'
import {
  buildMetadataProfileDraft,
  buildMetadataProfileInput,
  buildProviderInstanceDraft,
  buildProviderInstanceInput,
  buildStageProviderOptions,
  MetadataProviderSettingsPanel,
} from './metadata-provider-settings-panel'

const invalidateQueries = vi.fn()
const useQuery = vi.fn()
const useMutation = vi.fn()

vi.mock('@tanstack/react-query', async () => {
  const actual = await vi.importActual<typeof import('@tanstack/react-query')>(
    '@tanstack/react-query'
  )
  return {
    ...actual,
    useQuery: (...args: unknown[]) => useQuery(...args),
    useMutation: (...args: unknown[]) => useMutation(...args),
    useQueryClient: () => ({
      invalidateQueries,
    }),
  }
})

describe('MetadataProviderSettingsPanel', () => {
  beforeEach(() => {
    vi.useRealTimers()
    invalidateQueries.mockReset()
    useQuery.mockReset()
    useMutation.mockReset()
    useMutation.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    })
  })

  it('filters stage options by capability, enabled state, and plugin health', () => {
    const options = buildStageProviderOptions(
      [
        createBuiltinProvider({
          id: 1,
          name: 'TMDB 主实例',
          provider_type: 'tmdb',
        }),
        createBuiltinProvider({
          id: 2,
          name: 'TVDB',
          provider_type: 'tvdb',
        }),
        createBuiltinProvider({
          id: 3,
          name: 'MetaTube',
          provider_type: 'metatube',
          enabled: false,
        }),
      ],
      [
        createPluginProvider({
          id: 11,
          name: 'Anime Search',
          capabilities: ['metadata.search'],
        }),
        createPluginProvider({
          id: 12,
          name: 'Anime Detail',
          capabilities: ['metadata.detail'],
        }),
        createPluginProvider({
          id: 13,
          name: 'Broken Search',
          capabilities: ['metadata.search'],
          availability_status: 'unavailable',
        }),
      ],
      'metadata.search'
    )

    expect(options.map((option) => option.ref)).toEqual([
      'builtin:1',
      'plugin:11',
    ])
  })

  it('preserves plugin-backed profile references in the editable draft and API input', () => {
    const profile = createProfile({
      id: 5,
      name: '动画默认模板',
      search_provider_refs: ['plugin:9'],
      detail_provider_refs: ['builtin:1', 'plugin:9'],
    })

    const draft = buildMetadataProfileDraft(profile)
    expect(draft.searchProviderRefs).toEqual(['plugin:9'])
    expect(draft.detailProviderRefs).toEqual(['builtin:1', 'plugin:9'])

    expect(buildMetadataProfileInput(draft)).toMatchObject({
      search_provider_ids: [],
      search_provider_refs: ['plugin:9'],
      detail_provider_ids: [],
      detail_provider_refs: ['builtin:1', 'plugin:9'],
    })
  })

  it('maps cooldown deadline to remaining duration in provider drafts', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-29T12:00:00Z'))

    const draft = buildProviderInstanceDraft(
      createBuiltinProvider({
        cooldown_until: '2026-05-29T12:15:00Z',
      })
    )

    expect(draft.cooldownDuration).toBe('15m')
  })

  it('converts cooldown duration to cooldown deadline on save', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-29T12:00:00Z'))

    const input = buildProviderInstanceInput({
      name: 'TMDB',
      providerType: 'tmdb',
      enabled: true,
      availabilityStatus: 'cooldown',
      failureReason: '',
      cooldownDuration: '15m',
      tmdb: {
        apiKey: '',
        clearApiKey: false,
        baseURL: 'https://api.themoviedb.org/3',
        imageBaseURL: 'https://image.tmdb.org/t/p/original',
        language: 'zh-CN',
        timeout: '30s',
        retryCount: '2',
        upstreamProviderFilter: '',
        fallbackEnabled: true,
      },
      tvdb: {
        apiKey: '',
        clearApiKey: false,
        baseURL: '',
        imageBaseURL: '',
        language: '',
        timeout: '',
        retryCount: '',
        upstreamProviderFilter: '',
        fallbackEnabled: true,
      },
      metatube: {
        apiKey: '',
        clearApiKey: false,
        baseURL: '',
        imageBaseURL: '',
        language: '',
        timeout: '',
        retryCount: '',
        upstreamProviderFilter: '',
        fallbackEnabled: true,
      },
    })

    expect(input.cooldown_until).toBe('2026-05-29T12:15:00.000Z')
  })

  it('renders plugin-backed provider instances alongside built-in providers', async () => {
    useQuery.mockImplementation((options?: { queryKey?: unknown[] }) => {
      const key = options?.queryKey?.[1]
      if (key === 'metadata-providers') {
        return {
          data: [
            createBuiltinProvider({
              id: 1,
              name: 'TMDB 主实例',
              provider_type: 'tmdb',
            }),
          ],
          isLoading: false,
          error: null,
        }
      }
      if (key === 'plugin-providers') {
        return {
          data: [
            createPluginProvider({
              id: 9,
              name: 'Anime Plugin',
              capabilities: ['metadata.search', 'metadata.detail'],
            }),
          ],
          isLoading: false,
          error: null,
        }
      }
      return {
        data: [
          createProfile({
            id: 5,
            name: '动画默认模板',
            search_provider_refs: ['plugin:9'],
            detail_provider_refs: ['builtin:1', 'plugin:9'],
          }),
        ],
        isLoading: false,
        error: null,
      }
    })

    const view = await render(
      <MetadataProviderSettingsPanel token='session-token' />
    )

    await expect.element(view.getByText('Anime Plugin')).toBeInTheDocument()
    await expect
      .element(view.getByText('mock-anime-plugin · remote · #9'))
      .toBeInTheDocument()
    await expect.element(view.getByText('TMDB 主实例')).toBeInTheDocument()
    await expect
      .element(view.getByRole('button', { name: '注册远程插件' }))
      .not.toBeInTheDocument()
    await expect
      .element(view.getByRole('button', { name: '新建内置实例' }))
      .not.toBeInTheDocument()
  })
})

function createBuiltinProvider(
  overrides: Partial<MetadataProviderInstance>
): MetadataProviderInstance {
  return {
    id: 1,
    name: 'TMDB',
    provider_type: 'tmdb',
    system_managed: false,
    locked: false,
    enabled: true,
    availability_status: 'available',
    configured: true,
    tmdb: {
      configured: true,
      api_key_masked: true,
      base_url: 'https://api.themoviedb.org/3',
      image_base_url: 'https://image.tmdb.org/t/p/original',
      language: 'zh-CN',
      timeout: '30s',
      retry_count: 2,
      source: 'user',
      implementation: 'tmdb',
    },
    ...overrides,
  }
}

function createPluginProvider(
  overrides: Partial<PluginProviderInstance>
): PluginProviderInstance {
  return {
    id: 1,
    name: 'Mock Plugin',
    deployment_kind: 'remote',
    endpoint: 'https://plugin.example.com',
    plugin_id: 'io.mibo.plugin.mock',
    plugin_name: 'mock-anime-plugin',
    plugin_version: '1.0.0',
    protocol_version: '1.0',
    capabilities: ['metadata.search'],
    enabled: true,
    availability_status: 'available',
    manifest: {
      id: 'io.mibo.plugin.mock',
      name: 'Mock Plugin',
      version: '1.0.0',
      protocol_version: '1.0',
      health: { path: '/health' },
      capabilities: [
        {
          capability: 'metadata.search',
          endpoint: { path: '/metadata/search' },
        },
      ],
      configuration_schema: { fields: [] },
    },
    configuration: {},
    created_at: '2026-05-27T00:00:00Z',
    updated_at: '2026-05-27T00:00:00Z',
    ...overrides,
  }
}

function createProfile(overrides: Partial<MetadataProfile>): MetadataProfile {
  return {
    id: 1,
    name: '默认模板',
    description: '',
    system: false,
    locked: false,
    search_provider_ids: [],
    search_provider_refs: [],
    detail_provider_ids: [],
    detail_provider_refs: [],
    preferred_metadata_language: 'zh-CN',
    fallback_enabled: true,
    ...overrides,
  }
}
