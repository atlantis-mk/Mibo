import { describe, expect, it } from 'vitest'
import {
  miboQueryKeys,
  operationsIssueDetailQueryOptions,
  operationsIssueEventsQueryOptions,
  operationsIssueListQueryOptions,
} from '@/lib/mibo-query'

describe('playback query keys', () => {
  it('include selected catalog playback variant and start position', () => {
    expect(
      miboQueryKeys.catalogPlayback('token', 12, {
        resourceId: 34,
        variant: '1080p',
        startSeconds: 90,
      })
    ).toEqual([
      'catalog',
      'playback',
      'token',
      12,
      34,
      '1080p',
      90,
      'default-audio',
    ])
  })

  it('include selected inventory playback variant and start position', () => {
    expect(
      miboQueryKeys.inventoryFilePlayback('token', 56, {
        variant: 'audio-repair',
        startSeconds: 120,
      })
    ).toEqual([
      'inventory-file',
      'playback',
      'token',
      56,
      'audio-repair',
      120,
      'default-audio',
    ])
  })
})

describe('operations issue query keys', () => {
  it('namespace list, detail, and events keys separately', () => {
    expect(miboQueryKeys.operationsIssues('token')).toEqual([
      'operations',
      'issues',
      'token',
    ])
    expect(miboQueryKeys.operationsIssueDetail('token', 12)).toEqual([
      'operations',
      'issues',
      'detail',
      'token',
      12,
    ])
    expect(miboQueryKeys.operationsIssueEvents('token', 12)).toEqual([
      'operations',
      'issues',
      'events',
      'token',
      12,
    ])
  })

  it('keeps filter-sensitive issue list keys stable for invalidation', () => {
    const filters = {
      page: 1,
      page_size: 20,
      status: 'active' as const,
      kind: 'metadata' as const,
      action_type: 'apply_candidate' as const,
      library_id: 7,
      q: 'season',
    }

    expect(operationsIssueListQueryOptions('token', filters).queryKey).toEqual([
      'operations',
      'issues',
      'token',
      filters,
    ])
    expect(operationsIssueDetailQueryOptions('token', 7).queryKey).toEqual([
      'operations',
      'issues',
      'detail',
      'token',
      7,
    ])
    expect(operationsIssueEventsQueryOptions('token', 7).queryKey).toEqual([
      'operations',
      'issues',
      'events',
      'token',
      7,
    ])
  })
})
