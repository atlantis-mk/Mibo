import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import type { PluginProviderInstance } from '@/lib/mibo-api'
import {
  buildStorageProviderOptions,
  EMPTY_SOURCE_FORM,
  SourceForm,
} from './media-source-form'

describe('SourceForm', () => {
  it('only exposes enabled plugin storage providers with resolve capability', () => {
    const options = buildStorageProviderOptions([
      createPluginProvider({
        id: 8,
        name: 'Storage Ready',
        capabilities: ['storage.resolve', 'storage.browse'],
      }),
      createPluginProvider({
        id: 9,
        name: 'Browse Only',
        capabilities: ['storage.browse'],
      }),
      createPluginProvider({
        id: 10,
        name: 'Disabled Ready',
        capabilities: ['storage.resolve'],
        enabled: false,
      }),
    ])

    expect(options.map((option) => option.value)).toContain('plugin:8')
    expect(options.map((option) => option.value)).not.toContain('plugin:9')
    expect(options.map((option) => option.value)).not.toContain('plugin:10')
  })

  it('renders schema-driven plugin details for plugin-backed media sources', async () => {
    const pluginProvider = createPluginProvider({
      id: 12,
      name: 'NAS Plugin',
      endpoint: 'https://storage.example.com',
      capabilities: ['storage.resolve', 'storage.browse', 'storage.link'],
      configuration: {
        token: '***REDACTED***',
        library_root: '/media',
      },
      manifest: {
        id: 'io.mibo.plugin.storage',
        name: 'NAS Plugin',
        version: '1.0.0',
        protocol_version: '1.0',
        health: { path: '/health' },
        capabilities: [
          {
            capability: 'storage.resolve',
            endpoint: { path: '/storage/resolve' },
          },
        ],
        configuration_schema: {
          fields: [
            {
              key: 'token',
              type: 'secret',
              display: { label: '访问令牌' },
            },
            {
              key: 'library_root',
              type: 'string',
              display: { label: '媒体根目录' },
            },
          ],
        },
      },
    })

    const { getByText, container } = await render(
      <SourceForm
        draft={{
          ...EMPTY_SOURCE_FORM,
          provider: 'plugin:12',
          name: 'NAS',
          rootPath: '/media',
        }}
        onChange={vi.fn()}
        api={null}
        pluginProviderInstances={[pluginProvider]}
      />
    )

    await expect
      .element(getByText('https://storage.example.com'))
      .toBeInTheDocument()
    await expect.element(getByText('mock-storage-plugin')).toBeInTheDocument()
    await expect.element(getByText('访问令牌')).toBeInTheDocument()
    await expect.element(getByText('媒体根目录')).toBeInTheDocument()
    await expect
      .element(getByText('已配置的密钥不会回显，留空即可保持现有值。'))
      .toBeInTheDocument()

    const rootInput = Array.from(container.querySelectorAll('input')).find(
      (input) => input.value === '/media'
    )
    expect(rootInput).toBeTruthy()
  })
})

function createPluginProvider(
  overrides: Partial<PluginProviderInstance>
): PluginProviderInstance {
  return {
    id: 1,
    name: 'Mock Storage Plugin',
    deployment_kind: 'remote',
    endpoint: 'https://plugin.example.com',
    plugin_id: 'io.mibo.plugin.storage',
    plugin_name: 'mock-storage-plugin',
    plugin_version: '1.0.0',
    protocol_version: '1.0',
    capabilities: ['storage.resolve'],
    enabled: true,
    availability_status: 'available',
    manifest: {
      id: 'io.mibo.plugin.storage',
      name: 'Mock Storage Plugin',
      version: '1.0.0',
      protocol_version: '1.0',
      health: { path: '/health' },
      capabilities: [
        {
          capability: 'storage.resolve',
          endpoint: { path: '/storage/resolve' },
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
