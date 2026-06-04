import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, type RenderResult } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { UserAuthForm } from './user-auth-form'

const FORM_MESSAGES = {
  usernameEmpty: '请输入用户名。',
  passwordEmpty: '请输入密码。',
} as const

const { loginMock, loginWithPinMock, navigate, toastError } = vi.hoisted(
  () => ({
    loginMock: vi.fn(),
    loginWithPinMock: vi.fn(),
    navigate: vi.fn(),
    toastError: vi.fn(),
  })
)
type BrowserLocator = ReturnType<RenderResult['getByRole']>
let mockSessionState = {
  errorMessage: null as string | null,
  hasHydrated: true,
  isSubmitting: false,
  login: loginMock,
  loginUsers: [] as Array<{
    id: number
    username: string
    avatar_url: string
    has_pin: boolean
    updated_at: string
  }>,
  loginUsersLoading: false,
  loginWithPin: loginWithPinMock,
  user: null as {
    id: number
    username: string
    role: string
    created_at: string
    updated_at: string
  } | null,
}

vi.mock('@/hooks/use-login-session', () => ({
  useLoginSession: () => mockSessionState,
}))

vi.mock('sonner', () => ({
  toast: {
    error: toastError,
  },
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigate,
    Link: ({
      children,
      to,
      className,
      ...rest
    }: {
      children?: React.ReactNode
      to: string
      className?: string
    }) => (
      <a href={to} className={className} {...rest}>
        {children}
      </a>
    ),
  }
})
describe('UserAuthForm', () => {
  describe('Rendering without redirectTo', () => {
    let screen: Awaited<ReturnType<typeof render>>
    let usernameInput: BrowserLocator
    let passwordInput: BrowserLocator
    let signInButton: BrowserLocator
    let forgotPasswordLink: BrowserLocator

    beforeEach(async () => {
      vi.clearAllMocks()
      mockSessionState = {
        errorMessage: null,
        hasHydrated: true,
        isSubmitting: false,
        login: loginMock,
        loginUsers: [],
        loginUsersLoading: false,
        loginWithPin: loginWithPinMock,
        user: null,
      }
      screen = await render(<UserAuthForm />)
      usernameInput = screen.getByRole('textbox', { name: /^用户名$/i })
      passwordInput = screen.getByLabelText(/^密码$/i)
      signInButton = screen.getByRole('button', { name: /^登录$/i })
      forgotPasswordLink = screen.getByText(/^忘记密码？$/i)
    })

    it('renders fields, submit button, and forgot password link', async () => {
      await expect.element(usernameInput).toBeInTheDocument()
      await expect.element(passwordInput).toBeInTheDocument()
      await expect.element(signInButton).toBeInTheDocument()
      await expect.element(forgotPasswordLink).toBeInTheDocument()
    })

    it('shows validation messages when submitting empty form', async () => {
      await userEvent.clear(usernameInput)
      await userEvent.clear(passwordInput)
      await userEvent.click(signInButton)

      await expect
        .element(screen.getByText(FORM_MESSAGES.usernameEmpty))
        .toBeInTheDocument()
      await expect
        .element(screen.getByText(FORM_MESSAGES.passwordEmpty))
        .toBeInTheDocument()
    })

    it('submits username and password to the login hook', async () => {
      await userEvent.clear(usernameInput)
      await userEvent.fill(usernameInput, 'admin')
      await userEvent.clear(passwordInput)
      await userEvent.fill(passwordInput, 'admin123')

      await userEvent.click(signInButton)

      await vi.waitFor(() =>
        expect(loginMock).toHaveBeenCalledWith('admin', 'admin123')
      )
    })
  })

  it('submits a selected user PIN to the login hook', async () => {
    vi.clearAllMocks()
    mockSessionState = {
      errorMessage: null,
      hasHydrated: true,
      isSubmitting: false,
      login: loginMock,
      loginUsers: [
        {
          id: 7,
          username: 'alice',
          avatar_url: '',
          has_pin: true,
          updated_at: '2024-01-01T00:00:00Z',
        },
      ],
      loginUsersLoading: false,
      loginWithPin: loginWithPinMock,
      user: null,
    }

    const screen = await render(<UserAuthForm />)
    await userEvent.click(screen.getByRole('button', { name: /alice/i }))
    await userEvent.fill(screen.getByLabelText(/^PIN$/i), '1234')

    await vi.waitFor(() =>
      expect(loginWithPinMock).toHaveBeenCalledWith(7, '1234')
    )
  })

  it('shows login errors as toast messages', async () => {
    vi.clearAllMocks()
    mockSessionState = {
      errorMessage: 'invalid pin',
      hasHydrated: true,
      isSubmitting: false,
      login: loginMock,
      loginUsers: [
        {
          id: 7,
          username: 'alice',
          avatar_url: '',
          has_pin: true,
          updated_at: '2024-01-01T00:00:00Z',
        },
      ],
      loginUsersLoading: false,
      loginWithPin: loginWithPinMock,
      user: null,
    }

    const screen = await render(<UserAuthForm />)

    await vi.waitFor(() =>
      expect(toastError).toHaveBeenCalledWith('登录失败', {
        description: 'invalid pin',
      })
    )
    await expect
      .element(screen.getByText('invalid pin'))
      .not.toBeInTheDocument()
  })

  it('redirects after an already authenticated session is present', async () => {
    vi.clearAllMocks()

    mockSessionState = {
      errorMessage: null,
      hasHydrated: true,
      isSubmitting: false,
      login: loginMock,
      loginUsers: [],
      loginUsersLoading: false,
      loginWithPin: loginWithPinMock,
      user: {
        id: 1,
        username: 'admin',
        role: 'admin',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
    }

    await render(<UserAuthForm redirectTo='/library' />)

    await vi.waitFor(() =>
      expect(navigate).toHaveBeenCalledWith({
        to: '/library',
        replace: true,
      })
    )
  })
})
