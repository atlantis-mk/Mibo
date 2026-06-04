import { describe, expect, it } from 'vitest'
import type { CatalogListItem, OperationsTask } from '@/lib/mibo-api'
import { getHomeDashboardState, type HomeDashboardData } from './home-state'

function data(overrides: Partial<HomeDashboardData> = {}): HomeDashboardData {
  return {
    items: [],
    continueWatching: [],
    continueWatchingCount: 0,
    contentSections: [],
    mediaOverview: { sections: [] },
    operationsTasks: [],
    ...overrides,
  }
}

function item(overrides: Partial<CatalogListItem> = {}): CatalogListItem {
  return {
    id: 1,
    library_id: 1,
    type: 'movie',
    title: 'Movie',
    availability_status: 'available',
    governance_status: 'ok',
    ...overrides,
  }
}

function blockingTask(overrides: Partial<OperationsTask> = {}): OperationsTask {
  return {
    id: 'storage-auth',
    kind: 'storage_access_required',
    severity: 'blocking',
    title: 'Storage auth expired',
    summary: 'Repair provider auth',
    impact: {
      blocks_scan: true,
      blocks_home_visibility: true,
      blocks_playback: false,
      affected_libraries: 1,
      affected_files: 20,
      affected_items: 10,
    },
    affected: {
      media_sources: [],
      libraries: [
        {
          id: 1,
          name: 'Movies',
          type: 'movie',
          status: 'error',
          media_source_id: 1,
          root_path: '/movies',
        },
      ],
      files: [],
      items: [],
    },
    recommended_actions: [],
    evidence: [],
    ...overrides,
  }
}

describe('getHomeDashboardState', () => {
  it('covers the empty setup state', () => {
    const state = getHomeDashboardState(data())

    expect(state.hasEmptySetupState).toBe(true)
    expect(state.hasDisplayableHomeContent).toBe(false)
    expect(state.isHealthBlocked).toBe(false)
  })

  it('covers the normal populated state', () => {
    const state = getHomeDashboardState(
      data({
        contentSections: [{ key: 'movies', title: '电影', items: [item()] }],
        mediaOverview: {
          sections: [
            { key: 'movies', title: '电影', count: 7, items: [item()] },
          ],
        },
      })
    )

    expect(state.hasDisplayableHomeContent).toBe(true)
    expect(state.movieCount).toBe(7)
    expect(state.showCount).toBe(0)
    expect(state.isPartiallyDegraded).toBe(false)
  })

  it('covers the fully health-blocked state', () => {
    const state = getHomeDashboardState(
      data({
        operationsTasks: [blockingTask()],
      })
    )

    expect(state.hasEmptySetupState).toBe(true)
    expect(state.hasDisplayableHomeContent).toBe(false)
    expect(state.isHealthBlocked).toBe(true)
    expect(state.homeBlockingTask?.id).toBe('storage-auth')
  })

  it('covers the partially degraded state', () => {
    const state = getHomeDashboardState(
      data({ items: [item()], operationsTasks: [blockingTask()] })
    )

    expect(state.hasDisplayableHomeContent).toBe(true)
    expect(state.isHealthBlocked).toBe(false)
    expect(state.isPartiallyDegraded).toBe(true)
  })

  it('treats filtered-out empty rails as a normal empty home state', () => {
    const state = getHomeDashboardState(
      data({
        contentSections: [{ key: 'movies', title: '电影', items: [] }],
      })
    )

    expect(state.contentSections).toEqual([])
    expect(state.hasEmptySetupState).toBe(true)
    expect(state.hasDisplayableHomeContent).toBe(false)
  })
})
