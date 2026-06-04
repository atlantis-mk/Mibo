import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'

const pluginProviders = [
  {
    id: 12,
    name: 'Anime Metadata',
    deployment_kind: 'remote',
    endpoint: 'https://plugin.example.test',
    plugin_id: 'io.mibo.anime',
    plugin_name: 'Anime Plugin',
    plugin_version: '1.0.0',
    protocol_version: '1.0',
    capabilities: ['metadata.search', 'metadata.detail'],
    enabled: true,
    availability_status: 'available',
    manifest: {
      id: 'io.mibo.anime',
      name: 'Anime Plugin',
      version: '1.0.0',
      protocol_version: '1.0',
      health: { path: '/health' },
      capabilities: [],
      configuration_schema: { fields: [] },
    },
    configuration: {},
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
]

const api = {
  previewRemotePluginManifest: vi.fn(),
  createRemotePluginProviderInstance: vi.fn(),
  updateRemotePluginProviderInstance: vi.fn(),
  disablePluginProviderInstance: vi.fn(),
  refreshPluginProviderHealth: vi.fn(),
  getPluginProviderDetail: vi.fn(),
  installLocalPlugin: vi.fn(),
  startLocalPluginInstallation: vi.fn(),
  stopLocalPluginInstallation: vi.fn(),
  restartLocalPluginInstallation: vi.fn(),
  uninstallLocalPluginInstallation: vi.fn(),
}

vi.mock('@/lib/mibo-query', async () => {
  const { queryOptions } = await import('@tanstack/react-query')
  return {
    createAuthedMiboApi: () => api,
    miboQueryKeys: {
      metadataProviderInstances: (token: string) => [
        'settings',
        'metadata-providers',
        token,
      ],
      metadataProfiles: (token: string) => [
        'settings',
        'metadata-profiles',
        token,
      ],
      pluginProviderInstances: (token: string) => [
        'settings',
        'plugin-providers',
        token,
      ],
      pluginProviderDetail: (token: string, id: number) => [
        'settings',
        'plugin-providers',
        token,
        id,
        'detail',
      ],
      localPluginInstallations: (token: string) => [
        'settings',
        'plugin-local-installations',
        token,
      ],
      internalPlugins: (token: string) => [
        'settings',
        'plugin-internal',
        token,
      ],
      openSubtitlesSettings: (token: string) => [
        'settings',
        'plugin-internal',
        'opensubtitles',
        token,
      ],
      subtitleProviderInstances: (token: string) => [
        'settings',
        'subtitles',
        'providers',
        token,
      ],
      pluginCatalogOverview: (token: string) => [
        'settings',
        'plugin-catalog',
        token,
      ],
    },
    pluginProviderInstancesQueryOptions: (token: string) =>
      queryOptions({
        queryKey: ['settings', 'plugin-providers', token],
        queryFn: () => Promise.resolve(pluginProviders),
      }),
    metadataProviderInstancesQueryOptions: (token: string) =>
      queryOptions({
        queryKey: ['settings', 'metadata-providers', token],
        queryFn: () => Promise.resolve([]),
      }),
    metadataProfilesQueryOptions: (token: string) =>
      queryOptions({
        queryKey: ['settings', 'metadata-profiles', token],
        queryFn: () => Promise.resolve([]),
      }),
    pluginProviderDetailQueryOptions: (token: string, id: number) =>
      queryOptions({
        queryKey: ['settings', 'plugin-providers', token, id, 'detail'],
        queryFn: () =>
          Promise.resolve({
            instance: pluginProviders[0],
            usage: {
              provider_instance_id: 12,
              metadata_profiles: [
                {
                  kind: 'metadata_profile',
                  id: 2,
                  name: 'Default metadata',
                  stage: 'search',
                },
              ],
              library_metadata_strategies: [],
              media_sources: [
                {
                  kind: 'media_source',
                  id: 4,
                  name: 'Plugin media',
                },
              ],
              active_reference_count: 2,
            },
          }),
      }),
    localPluginInstallationsQueryOptions: (token: string) =>
      queryOptions({
        queryKey: ['settings', 'plugin-local-installations', token],
        queryFn: () => Promise.resolve([]),
      }),
    internalPluginsQueryOptions: (token: string) =>
      queryOptions({
        queryKey: ['settings', 'plugin-internal', token],
        queryFn: () => Promise.resolve([]),
      }),
    openSubtitlesSettingsQueryOptions: (token: string) =>
      queryOptions({
        queryKey: ['settings', 'plugin-internal', 'opensubtitles', token],
        queryFn: () =>
          Promise.resolve({
            configured: false,
            api_key_masked: false,
            api_key_count: 0,
            base_url: '',
            languages: '',
            timeout: '',
            source: 'database',
          }),
      }),
    subtitleProviderInstancesQueryOptions: (token: string) =>
      queryOptions({
        queryKey: ['settings', 'subtitles', 'providers', token],
        queryFn: () => Promise.resolve([]),
      }),
    pluginCatalogOverviewQueryOptions: (token: string) =>
      queryOptions({
        queryKey: ['settings', 'plugin-catalog', token],
        queryFn: () =>
          Promise.resolve({
            sources: [
              {
                id: 1,
                name: 'Local Feed',
                trust_level: 'checksum',
              },
            ],
            entries: [
              {
                id: 1,
                name: 'Catalog Plugin',
                version: '1.0.0',
                signature_status: 'checksum',
                compatibility: {
                  compatible: false,
                  reasons: ['current platform is not supported'],
                },
                release_notes: 'Initial release',
              },
            ],
          }),
      }),
  }
})

async function renderPluginCenter() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const { PluginManagementCenter } = await import('./index')
  return render(
    <QueryClientProvider client={queryClient}>
      <PluginManagementCenter token='test-token' />
    </QueryClientProvider>
  )
}

describe('PluginManagementCenter', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.confirm = vi.fn(() => true)
    api.refreshPluginProviderHealth.mockResolvedValue(pluginProviders[0])
    api.disablePluginProviderInstance.mockResolvedValue({
      ...pluginProviders[0],
      enabled: false,
    })
    api.getPluginProviderDetail.mockResolvedValue({
      instance: pluginProviders[0],
      usage: {
        provider_instance_id: 12,
        metadata_profiles: [],
        library_metadata_strategies: [],
        media_sources: [],
        active_reference_count: 0,
      },
    })
  })

  it('lists plugin instances and supports health refresh', async () => {
    const screen = await renderPluginCenter()

    await userEvent.click(screen.getByRole('tab', { name: '实例' }))

    await expect.element(screen.getByText('Anime Metadata')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /刷新健康/ }))

    expect(api.refreshPluginProviderHealth).toHaveBeenCalledWith(12)
  })

  it('shows reference warnings before disabling a referenced plugin', async () => {
    api.getPluginProviderDetail.mockResolvedValueOnce({
      instance: pluginProviders[0],
      usage: {
        provider_instance_id: 12,
        metadata_profiles: [
          { kind: 'metadata_profile', id: 2, name: 'Default' },
        ],
        library_metadata_strategies: [],
        media_sources: [],
        active_reference_count: 1,
      },
    })
    const screen = await renderPluginCenter()

    await userEvent.click(screen.getByRole('tab', { name: '实例' }))
    await userEvent.click(screen.getByRole('button', { name: /^禁用$/ }))

    expect(window.confirm).toHaveBeenCalled()
    expect(api.disablePluginProviderInstance).toHaveBeenCalledWith(12)
  })

  it('renders incompatible catalog entries with trust metadata', async () => {
    const screen = await renderPluginCenter()

    await userEvent.click(screen.getByRole('tab', { name: '目录' }))

    await expect.element(screen.getByText('Catalog Plugin')).toBeInTheDocument()
    await expect.element(screen.getByText(/暂不可安装/)).toBeInTheDocument()
    await expect
      .element(screen.getByText(/current platform is not supported/))
      .toBeInTheDocument()
  })
})
