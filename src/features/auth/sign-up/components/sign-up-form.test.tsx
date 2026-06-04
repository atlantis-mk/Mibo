import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, type RenderResult } from 'vitest-browser-react'
import { type Locator, userEvent } from 'vitest/browser'
import { SignUpForm } from './sign-up-form'

const FORM_MESSAGES = {
  usernameEmpty: '请输入用户名。',
  passwordEmpty: '请输入密码。',
  confirmPasswordEmpty: '请确认密码。',
  pinEmpty: '请输入 PIN。',
  pinInvalid: 'PIN 必须是 4 位数字。',
  passwordMismatch: '两次输入的密码不一致。',
} as const

const { navigate, registerMock, toastSuccess, toastError } = vi.hoisted(() => ({
  navigate: vi.fn(),
  registerMock: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigate,
  }
})

vi.mock('@/lib/mibo-api', () => ({
  createMiboApi: () => ({ register: registerMock }),
  getApiBaseUrl: () => 'http://localhost:3000',
}))

vi.mock('@/lib/handle-server-error', () => ({
  handleServerError: (error: unknown) => toastError(error),
}))

vi.mock('sonner', () => ({
  toast: {
    success: toastSuccess,
    error: toastError,
  },
}))

function renderForm() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <SignUpForm />
    </QueryClientProvider>
  )
}

describe('SignUpForm', () => {
  let screen: RenderResult
  let usernameInput: Locator
  let passwordInput: Locator
  let confirmPasswordInput: Locator
  let pinInput: Locator
  let submitButton: Locator

  beforeEach(async () => {
    vi.clearAllMocks()
    registerMock.mockResolvedValue({
      id: 2,
      username: 'alice',
      role: 'user',
      roles: ['user'],
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    })

    screen = await renderForm()
    usernameInput = screen.getByRole('textbox', { name: /^用户名$/i })
    passwordInput = screen.getByLabelText(/^密码$/i)
    confirmPasswordInput = screen.getByLabelText(/^确认密码$/i)
    pinInput = screen.getByLabelText(/^PIN$/i)
    submitButton = screen.getByRole('button', { name: /^创建账户$/i })
  })

  it('renders username, password, PIN, and submit controls only', async () => {
    await expect.element(usernameInput).toBeInTheDocument()
    await expect.element(passwordInput).toBeInTheDocument()
    await expect.element(confirmPasswordInput).toBeInTheDocument()
    await expect.element(pinInput).toBeInTheDocument()
    await expect.element(submitButton).toBeInTheDocument()
    await expect
      .element(screen.getByRole('button', { name: /github/i }))
      .not.toBeInTheDocument()
    await expect
      .element(screen.getByRole('button', { name: /facebook/i }))
      .not.toBeInTheDocument()
  })

  it('shows validation messages when submitting empty form', async () => {
    await userEvent.click(submitButton)

    await expect
      .element(screen.getByText(FORM_MESSAGES.usernameEmpty))
      .toBeInTheDocument()
    await expect
      .element(screen.getByText(FORM_MESSAGES.passwordEmpty))
      .toBeInTheDocument()
    await expect
      .element(screen.getByText(FORM_MESSAGES.confirmPasswordEmpty))
      .toBeInTheDocument()
    await expect
      .element(screen.getByText(FORM_MESSAGES.pinEmpty))
      .toBeInTheDocument()
  })

  it('shows a mismatch error when passwords do not match', async () => {
    await userEvent.fill(usernameInput, 'alice')
    await userEvent.fill(passwordInput, 'password123')
    await userEvent.fill(confirmPasswordInput, 'password456')
    await userEvent.fill(pinInput, '1234')

    await userEvent.click(submitButton)
    await expect
      .element(screen.getByText(FORM_MESSAGES.passwordMismatch))
      .toBeInTheDocument()
  })

  it('requires a four digit PIN', async () => {
    await userEvent.fill(usernameInput, 'alice')
    await userEvent.fill(passwordInput, 'password123')
    await userEvent.fill(confirmPasswordInput, 'password123')
    await userEvent.fill(pinInput, '12')

    await userEvent.click(submitButton)
    await expect
      .element(screen.getByText(FORM_MESSAGES.pinInvalid))
      .toBeInTheDocument()
  })

  it('submits username, password, and PIN to the backend register API', async () => {
    await userEvent.fill(usernameInput, 'alice')
    await userEvent.fill(passwordInput, 'password123')
    await userEvent.fill(confirmPasswordInput, 'password123')
    await userEvent.fill(pinInput, '1234')

    await userEvent.click(submitButton)

    await vi.waitFor(() =>
      expect(registerMock).toHaveBeenCalledWith('alice', 'password123', '1234')
    )
    expect(toastSuccess).toHaveBeenCalledWith('已为 alice 创建账户，请登录。')
    expect(navigate).toHaveBeenCalledWith({
      to: '/sign-in',
      search: { redirect: undefined },
      replace: true,
    })
  })
})
