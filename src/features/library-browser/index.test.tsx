import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import LibraryBrowser from './index'

const browseLibraryHierarchy = vi.fn()

vi.mock('#/stores/auth-store', () => ({
  useAuthStore: (
    selector: (state: { auth: { accessToken: string } }) => unknown
  ) => selector({ auth: { accessToken: 'test-token' } }),
}))

vi.mock('#/lib/mibo-query', () => ({
  createAuthedMiboApi: () => ({
    browseLibraryHierarchy,
  }),
  miboQueryKeys: {
    libraryHierarchy: (
      token: string,
      libraryId: number | 'root',
      path: string,
      filters: unknown,
      page: number,
      pageSize: number
    ) => [
      'library',
      'hierarchy',
      token,
      libraryId,
      path,
      filters,
      page,
      pageSize,
    ],
  },
}))

vi.mock('#/components/media-poster-card', () => ({
  DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS: {
    imageType: 'primary',
    cardSize: 'default',
    fields: {
      Name: true,
      OriginalTitle: false,
      SortName: false,
      CommunityRating: true,
      CriticRating: false,
      OfficialRating: false,
      ProductionYear: true,
      PremiereDate: false,
      Runtime: true,
      Genres: false,
      Director: false,
      Tags: false,
      Studios: false,
      Tagline: false,
      Overview: false,
      DatePlayed: false,
      Played: false,
      DateCreated: false,
      IsFavorite: false,
    },
  },
  MediaPosterCard: ({ item }: { item: { title: string } }) => (
    <div data-testid='media-card'>{item.title}</div>
  ),
}))

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
}

async function renderLibraryBrowser({
  onFiltersChange = () => {},
}: {
  onFiltersChange?: Parameters<typeof LibraryBrowser>[0]['onFiltersChange']
} = {}) {
  const queryClient = createQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <LibraryBrowser
        libraryId={7}
        browsePath='中国电影'
        page={1}
        pageSize={24}
        filters={{
          q: '',
          type: 'all',
          genre: '',
          region: '',
          year: '',
          minRating: '',
          watchedState: 'all',
          organizingState: 'organized',
          sort: 'title',
          sortDirection: 'asc',
        }}
        scrollContainerRef={{ current: null }}
        onPaginationChange={() => {}}
        onBrowseTargetChange={vi.fn()}
        onFiltersChange={onFiltersChange}
      />
    </QueryClientProvider>
  )
}

describe('LibraryBrowser', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    browseLibraryHierarchy.mockResolvedValue({
      items: [
        {
          node_kind: 'folder',
          node_id: 'folder:china:action',
          library_id: 7,
          library_name: '电影',
          path: '中国电影/动作',
          title: '动作',
          child_count: 1,
          item_count: 2,
        },
        {
          node_kind: 'item',
          node_id: 'item:hero',
          library_id: 7,
          library_name: '电影',
          path: '中国电影',
          title: '英雄',
          item: {
            id: 11,
            metadata_item_id: 11,
            library_id: 7,
            type: 'movie',
            title: '英雄',
            availability_status: 'available',
            governance_status: 'accepted',
          },
        },
      ],
      total: 2,
      limit: 24,
      offset: 0,
      has_more: false,
      current_node: {
        node_kind: 'folder',
        node_id: 'folder:china',
        library_id: 7,
        library_name: '电影',
        path: '中国电影',
        parent_node_id: 'library:7',
      },
      breadcrumbs: [
        { node_kind: 'library', node_id: 'libraries', title: '媒体库' },
        {
          node_kind: 'library',
          node_id: 'library:7',
          library_id: 7,
          library_name: '电影',
          title: '电影',
        },
        {
          node_kind: 'folder',
          node_id: 'folder:china',
          library_id: 7,
          library_name: '电影',
          path: '中国电影',
          title: '中国电影',
        },
      ],
    })
  })

  it('renders breadcrumbs and mixed folder/item nodes', async () => {
    const screen = await renderLibraryBrowser()

    await expect
      .element(screen.getByRole('button', { name: '媒体库' }))
      .toBeInTheDocument()
    await expect
      .element(screen.getByRole('button', { name: /返回上一级/ }))
      .toBeInTheDocument()
    await expect.element(screen.getByText(/^动作$/)).toBeInTheDocument()
    await expect.element(screen.getByText(/^英雄$/)).toBeInTheDocument()
  })

  it('hides the parent navigation button at the browser root', async () => {
    browseLibraryHierarchy.mockResolvedValueOnce({
      items: [],
      total: 0,
      limit: 24,
      offset: 0,
      has_more: false,
      current_node: {
        node_kind: 'library',
        node_id: 'libraries',
      },
      breadcrumbs: [
        { node_kind: 'library', node_id: 'libraries', title: '媒体库' },
      ],
    })

    const screen = await renderLibraryBrowser()

    await expect
      .element(screen.getByRole('button', { name: /返回上一级/ }))
      .not.toBeInTheDocument()
  })

  it('navigates when a folder node is selected', async () => {
    const onBrowseTargetChange = vi.fn()
    const queryClient = createQueryClient()
    const screen = await render(
      <QueryClientProvider client={queryClient}>
        <LibraryBrowser
          libraryId={7}
          browsePath='中国电影'
          page={1}
          pageSize={24}
          filters={{
            q: '',
            type: 'all',
            genre: '',
            region: '',
            year: '',
            minRating: '',
            watchedState: 'all',
            organizingState: 'organized',
            sort: 'title',
            sortDirection: 'asc',
          }}
          scrollContainerRef={{ current: null }}
          onPaginationChange={() => {}}
          onBrowseTargetChange={onBrowseTargetChange}
          onFiltersChange={() => {}}
        />
      </QueryClientProvider>
    )

    await userEvent.click(screen.getByRole('button', { name: /动作/ }))

    expect(onBrowseTargetChange).toHaveBeenCalledWith({
      libraryId: 7,
      path: '中国电影/动作',
    })
  })

  it('clears active title sorting from the toolbar', async () => {
    const onFiltersChange = vi.fn()
    const screen = await renderLibraryBrowser({ onFiltersChange })

    await expect
      .element(screen.getByRole('button', { name: '取消标题排序' }))
      .toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: '取消标题排序' }))

    expect(onFiltersChange).toHaveBeenCalledWith(
      {
        q: '',
        type: 'all',
        genre: '',
        region: '',
        year: '',
        minRating: '',
        watchedState: 'all',
        organizingState: 'organized',
        sort: 'recent',
        sortDirection: 'desc',
      },
      { resetPage: true }
    )
  })
})
