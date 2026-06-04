import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMiboApi, type OperationsActionResult } from '@/lib/mibo-api'

describe('mibo operations issue api', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('maps issue list responses and forwards filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          items: [
            {
              id: 7,
              fingerprint: 'metadata:7',
              library_id: 3,
              kind: 'metadata',
              scope_kind: 'season',
              scope_key: 'season:7',
              lifecycle_status: 'active',
              severity: 'warning',
              title: 'Season review',
              summary: 'Needs metadata review',
              occurrence_count: 2,
              target_count: 4,
              impact: {
                blocks_scan: false,
                blocks_home_visibility: false,
                blocks_playback: false,
                affected_libraries: 1,
                affected_files: 4,
                affected_items: 3,
              },
            },
          ],
          total: 1,
          page: 2,
          page_size: 10,
        },
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const api = createMiboApi({
      baseUrl: 'http://localhost:3000',
      token: 'abc',
    })
    const result = await api.listOperationsIssues({
      page: 2,
      page_size: 10,
      status: 'active',
      kind: 'metadata',
      action_type: 'apply_candidate',
      library_id: 3,
      q: 'season',
    })

    expect(result.items[0]?.scope_kind).toBe('season')
    expect(result.total).toBe(1)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:3000/api/v1/operations/issues?page=2&page_size=10&status=active&kind=metadata&action_type=apply_candidate&library_id=3&q=season',
      expect.objectContaining({
        credentials: 'include',
        headers: expect.any(Headers),
      })
    )
  })

  it('posts issue actions with structured input and parses per-target results', async () => {
    const payload: OperationsActionResult = {
      action_id: 'issue_exclude',
      status: 'partial',
      message: 'Processed 2 targets.',
      results: [
        {
          target_type: 'inventory_file',
          target_key: 'inventory_file:10',
          status: 'ok',
          message: 'excluded',
        },
      ],
    }
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ data: payload }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const api = createMiboApi({
      baseUrl: 'http://localhost:3000',
      token: 'abc',
    })
    const result = await api.executeOperationsIssueAction(42, {
      action_key: 'issue_exclude',
      reason: 'other',
      confirmation: true,
    })

    expect(result.results?.[0]?.target_key).toBe('inventory_file:10')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:3000/api/v1/operations/issues/42/actions',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          action_key: 'issue_exclude',
          reason: 'other',
          confirmation: true,
        }),
      })
    )
  })

  it('fetches bootstrap database state for setup onboarding', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          active_driver: 'sqlite',
          active_source: 'default',
          active_connection: {
            driver: 'sqlite',
            sqlite_path: 'data/mibo.db',
            password_configured: false,
          },
          draft_connection: {
            driver: 'sqlite',
            sqlite_path: 'data/mibo.db',
            password_configured: false,
          },
          defaults: {
            sqlite_path: 'data/mibo.db',
            postgres_port: 5432,
            mysql_port: 3306,
            ssl_mode: 'disable',
          },
          edit_locked: false,
          initialization_locked: false,
          restart_required: false,
        },
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const api = createMiboApi({ baseUrl: 'http://localhost:3000' })
    const result = await api.getSetupDatabaseState()

    expect(result.active_driver).toBe('sqlite')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:3000/api/v1/setup/database',
      expect.objectContaining({
        credentials: 'include',
      })
    )
  })

  it('posts setup database draft requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          valid: true,
          normalized: {
            driver: 'postgres',
            host: '',
            port: 5432,
            database: '',
            username: '',
            ssl_mode: 'disable',
            password_configured: false,
          },
          message: '数据库类型草稿已保存',
        },
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const api = createMiboApi({ baseUrl: 'http://localhost:3000' })
    const result = await api.persistSetupDatabaseDraft({
      driver: 'postgres',
    })

    expect(result.normalized.driver).toBe('postgres')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:3000/api/v1/setup/database/draft',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          driver: 'postgres',
        }),
      })
    )
  })

  it('posts setup database apply requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 202,
      json: async () => ({
        data: {
          status: 'restarting',
          restart_required: true,
          normalized: {
            driver: 'sqlite',
            sqlite_path: 'data/alternate.db',
            password_configured: false,
          },
          message: '数据库配置已保存，服务正在重启以启用新配置',
        },
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const api = createMiboApi({ baseUrl: 'http://localhost:3000' })
    const result = await api.applySetupDatabase({
      driver: 'sqlite',
      sqlite_path: 'data/alternate.db',
    })

    expect(result.restart_required).toBe(true)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:3000/api/v1/setup/database/apply',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          driver: 'sqlite',
          sqlite_path: 'data/alternate.db',
        }),
      })
    )
  })
})
