import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { LiveTvSettingsPanel } from './live-tv-settings-panel'

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

describe('LiveTvSettingsPanel', () => {
  beforeEach(() => {
    invalidateQueries.mockReset()
    useQuery.mockReset()
    useMutation.mockReset()
    useQuery.mockReturnValue({
      data: null,
      isLoading: false,
      error: null,
    })
    useMutation.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    })
  })

  it('shows login guidance when no token is available', async () => {
    const { getByText } = await render(<LiveTvSettingsPanel token={null} />)

    await expect.element(getByText('登录后可管理直播源')).toBeInTheDocument()
    await expect
      .element(
        getByText('当前页面需要管理员会话来导入直播源、刷新频道并发起播放。')
      )
      .toBeInTheDocument()
  })

  it('loads source data into the form when editing an existing source', async () => {
    let queryCall = 0
    useQuery.mockImplementation(() => {
      queryCall += 1
      if (queryCall === 1) {
        return {
          data: [
            {
              id: 1,
              name: '示例源',
              source_type: 'playlist_url',
              format_hint: 'm3u',
              url: 'https://example.com/playlist.m3u',
              user_agent: 'Mibo Test Agent',
              referrer: 'https://example.com',
              tuner_count: 2,
              import_groups: '新闻;体育',
              import_guide_data: true,
              channel_image_source: 'guide',
              allow_guide_mapping_by_number: true,
              channel_tags: '直播;测试',
              enabled: true,
              refresh: { status: 'success' },
              channel_count: 2,
              created_at: '2026-05-26T00:00:00Z',
              updated_at: '2026-05-26T00:00:00Z',
            },
          ],
          isLoading: false,
          error: null,
        }
      }
      if (queryCall === 2) {
        return {
          data: [],
          isLoading: false,
          error: null,
        }
      }
      return {
        data: null,
        isLoading: false,
        error: null,
      }
    })

    const { getByRole, getByLabelText } = await render(
      <LiveTvSettingsPanel token='session-token' />
    )

    await userEvent.click(getByRole('button', { name: '编辑' }))

    await expect.element(getByLabelText('源名称')).toHaveValue('示例源')
    await expect
      .element(getByLabelText('文件或网址'))
      .toHaveValue('https://example.com/playlist.m3u')
    await expect
      .element(getByLabelText('用户代理 HTTP 标头'))
      .toHaveValue('Mibo Test Agent')
    await expect
      .element(getByLabelText('引用者 HTTP 标头'))
      .toHaveValue('https://example.com')
    await expect.element(getByLabelText('并发流限制')).toHaveValue(2)
    await expect
      .element(getByLabelText('仅导入包含这些组的频道'))
      .toHaveValue('新闻;体育')
    await expect
      .element(getByLabelText('为频道添加标签'))
      .toHaveValue('直播;测试')
  })
})
