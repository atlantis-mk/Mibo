import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { SetupPage } from './index'

const {
  applySetupDatabaseMock,
  getSetupDatabaseStateMock,
  getSetupStatusMock,
  validateSetupDatabaseMock,
} = vi.hoisted(() => ({
  applySetupDatabaseMock: vi.fn(),
  getSetupDatabaseStateMock: vi.fn(),
  getSetupStatusMock: vi.fn(),
  validateSetupDatabaseMock: vi.fn(),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  }
})

vi.mock('@/lib/mibo-api', () => ({
  createMiboApi: () => ({
    applySetupDatabase: applySetupDatabaseMock,
    getSetupStatus: getSetupStatusMock,
    getSetupDatabaseState: getSetupDatabaseStateMock,
    validateSetupDatabase: validateSetupDatabaseMock,
  }),
  getApiBaseUrl: () => 'http://localhost:3000',
}))

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

function renderSetupPage(options?: {
  cachedSetupStatus?: Awaited<ReturnType<typeof createSetupStatus>>
}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  })
  if (options?.cachedSetupStatus) {
    queryClient.setQueryData(['setup', 'status'], options.cachedSetupStatus)
  }

  return render(
    <QueryClientProvider client={queryClient}>
      <SetupPage />
    </QueryClientProvider>
  )
}

describe('SetupPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useRealTimers()
    applySetupDatabaseMock.mockResolvedValue({
      message: '数据库配置已保存',
      restart_required: false,
    })
    getSetupDatabaseStateMock.mockResolvedValue(createSetupDatabaseState())
    validateSetupDatabaseMock.mockResolvedValue({
      message: '数据库连接可用',
      normalized: createSetupDatabaseState().active_connection,
      valid: true,
    })
  })

  it('keeps the account step in a loading state while setup status is unresolved', async () => {
    getSetupStatusMock.mockReturnValue(new Promise(() => undefined))

    const screen = await renderSetupPage()

    await userEvent.click(screen.getByRole('button', { name: /^下一步$/i }))
    await userEvent.click(screen.getByRole('button', { name: /^下一步$/i }))

    await expect
      .element(screen.getByText('正在检测管理员账户'))
      .toBeInTheDocument()
    await expect
      .element(screen.getByRole('textbox', { name: '用户名' }))
      .not.toBeInTheDocument()
  })

  it('shows the account check before the account form when setup status resolves quickly', async () => {
    getSetupStatusMock.mockResolvedValue(createSetupStatus())

    const screen = await renderSetupPage()

    await userEvent.click(screen.getByRole('button', { name: /^下一步$/i }))
    await userEvent.click(screen.getByRole('button', { name: /^下一步$/i }))

    await expect
      .element(screen.getByText('正在检测管理员账户'))
      .toBeInTheDocument()
    await expect
      .element(screen.getByRole('textbox', { name: '用户名' }))
      .not.toBeInTheDocument()
  })

  it('does not show a cached existing-account result before setup status refreshes', async () => {
    getSetupStatusMock.mockReturnValue(new Promise(() => undefined))

    const screen = await renderSetupPage({
      cachedSetupStatus: createSetupStatus({ has_users: true }),
    })

    await expect
      .element(screen.getByText('已检测到现有账号'))
      .not.toBeInTheDocument()
    await expect
      .element(screen.getByText('第一步：选择数据库'))
      .toBeInTheDocument()
  })

  it('shows the account check before an automatic existing-account jump', async () => {
    getSetupStatusMock.mockResolvedValue(createSetupStatus({ has_users: true }))

    const screen = await renderSetupPage()

    await expect
      .element(screen.getByText('正在检测管理员账户'))
      .toBeInTheDocument()
    await expect
      .element(screen.getByText('已检测到现有账号'))
      .not.toBeInTheDocument()
  })

  it('shows the account check after applying a missing bootstrap config', async () => {
    getSetupDatabaseStateMock
      .mockResolvedValueOnce(
        createSetupDatabaseState({ activeSqlitePath: null })
      )
      .mockResolvedValue(createSetupDatabaseState())
    getSetupStatusMock
      .mockResolvedValueOnce(createSetupStatus())
      .mockResolvedValue(createSetupStatus({ has_users: true }))
    applySetupDatabaseMock.mockResolvedValue({
      message: '数据库配置已保存，服务正在重启以启用新配置',
      restart_required: true,
    })

    const screen = await renderSetupPage()

    await userEvent.click(screen.getByRole('button', { name: /^下一步$/i }))
    await userEvent.click(screen.getByRole('button', { name: /^测试连接$/i }))
    await userEvent.click(screen.getByRole('button', { name: /^下一步$/i }))

    await vi.waitFor(
      () => expect(getSetupDatabaseStateMock).toHaveBeenCalledTimes(2),
      { timeout: 2500 }
    )
    await expect
      .element(screen.getByText('正在检测管理员账户'))
      .toBeInTheDocument()
    await expect
      .element(screen.getByRole('textbox', { name: '用户名' }))
      .not.toBeInTheDocument()
  })

  it('keeps checking accounts while the applied database config is not active yet', async () => {
    getSetupDatabaseStateMock.mockResolvedValue(
      createSetupDatabaseState({ activeSqlitePath: null })
    )
    getSetupStatusMock.mockResolvedValue(createSetupStatus())

    const screen = await renderSetupPage()

    await userEvent.click(screen.getByRole('button', { name: /^下一步$/i }))
    await userEvent.click(screen.getByRole('button', { name: /^测试连接$/i }))
    await userEvent.click(screen.getByRole('button', { name: /^下一步$/i }))

    await expect
      .element(screen.getByText('正在检测管理员账户'))
      .toBeInTheDocument()
    await vi.waitFor(
      async () => {
        await expect
          .element(screen.getByRole('textbox', { name: '用户名' }))
          .not.toBeInTheDocument()
      },
      { timeout: 900 }
    )
  })
})

function createSetupStatus(overrides?: { has_users?: boolean }) {
  const hasUsers = overrides?.has_users ?? false
  return {
    initialized: hasUsers,
    can_enter_app: hasUsers,
    has_users: hasUsers,
    has_media_sources: false,
    has_libraries: false,
    user_count: hasUsers ? 1 : 0,
    media_source_count: 0,
    library_count: 0,
  }
}

function createSetupDatabaseState(options?: { activeSqlitePath?: string | null }) {
  const activeSqlitePath = options?.activeSqlitePath
  return {
    active_driver: 'sqlite',
    active_source: 'default',
    active_connection: {
      driver: 'sqlite',
      ...(activeSqlitePath === null
        ? {}
        : { sqlite_path: activeSqlitePath ?? 'data/mibo.db' }),
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
  }
}
