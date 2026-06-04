import { useEffect, useMemo, useState } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, LogIn, Plus, UserRound } from 'lucide-react'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { useLoginSession } from '@/hooks/use-login-session'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from '@/components/ui/input-otp'
import { Spinner } from '@/components/ui/spinner'
import { PasswordInput } from '@/components/password-input'

const passwordFormSchema = z.object({
  username: z.string().min(1, '请输入用户名。'),
  password: z.string().min(1, '请输入密码。'),
})

const pinFormSchema = z.object({
  pin: z
    .string()
    .length(4, 'PIN 必须是 4 位数字。')
    .regex(/^\d+$/, 'PIN 只能包含数字。'),
})

interface UserAuthFormProps extends React.HTMLAttributes<HTMLDivElement> {
  redirectTo?: string
}

type PasswordFormValues = z.infer<typeof passwordFormSchema>
type PINFormValues = z.infer<typeof pinFormSchema>

export function UserAuthForm({
  className,
  redirectTo,
  ...props
}: UserAuthFormProps) {
  const navigate = useNavigate()
  const {
    errorMessage,
    hasHydrated,
    isSubmitting,
    login,
    loginUsers,
    loginUsersLoading,
    loginWithPin,
    user,
  } = useLoginSession()
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
  const [showPasswordLogin, setShowPasswordLogin] = useState(false)

  const selectedUser = useMemo(
    () => loginUsers.find((candidate) => candidate.id === selectedUserId),
    [loginUsers, selectedUserId]
  )
  const showSwitcher = loginUsers.length > 0 && !showPasswordLogin

  const passwordForm = useForm<PasswordFormValues>({
    resolver: zodResolver(passwordFormSchema),
    defaultValues: {
      username: import.meta.env.DEV ? 'admin' : '',
      password: import.meta.env.DEV ? 'admin123' : '',
    },
  })
  const pinForm = useForm<PINFormValues>({
    resolver: zodResolver(pinFormSchema),
    defaultValues: { pin: '' },
  })

  useEffect(() => {
    if (!hasHydrated || !user) {
      return
    }

    void navigate({ to: redirectTo || '/', replace: true })
  }, [hasHydrated, navigate, redirectTo, user])

  useEffect(() => {
    if (!errorMessage) {
      return
    }

    toast.error('登录失败', { description: errorMessage })
  }, [errorMessage])

  async function onPasswordSubmit(data: PasswordFormValues) {
    await login(data.username, data.password)
  }

  async function onPINSubmit(data: PINFormValues) {
    if (!selectedUser) {
      return
    }
    await loginWithPin(selectedUser.id, data.pin)
  }

  function openPINLogin(userId: number) {
    setSelectedUserId(userId)
    pinForm.reset({ pin: '' })
  }

  function openPasswordLogin(username = '') {
    setSelectedUserId(null)
    setShowPasswordLogin(true)
    passwordForm.setValue('username', username)
  }

  const authHeader = (
    <div className='flex w-full justify-end'>
      <Link
        to='/sign-up'
        className='text-nowrap underline underline-offset-4 hover:text-primary'
      >
        注册
      </Link>
    </div>
  )

  if (loginUsersLoading && !showPasswordLogin) {
    return (
      <div className={cn('flex justify-center', className)} {...props}>
        <div className='flex items-center gap-2'>
          <Spinner />
          正在加载用户
        </div>
      </div>
    )
  }

  if (showSwitcher && selectedUser) {
    return (
      <div className={cn('grid w-full gap-6', className)} {...props}>
        <Form {...pinForm}>
          <form
            onSubmit={pinForm.handleSubmit(onPINSubmit)}
            className='grid justify-items-center gap-6 text-center'
          >
            <Avatar className='size-20'>
              <AvatarImage
                src={selectedUser.avatar_url}
                alt={selectedUser.username}
              />
              <AvatarFallback className='text-2xl'>
                {getUserInitial(selectedUser.username)}
              </AvatarFallback>
            </Avatar>

            <div>
              <h2 className='font-semibold tracking-tight'>
                {selectedUser.username}
              </h2>
            </div>

            <FormField
              control={pinForm.control}
              name='pin'
              render={({ field }) => (
                <FormItem className='grid justify-items-center'>
                  <FormControl>
                    <InputOTP
                      maxLength={4}
                      aria-label='PIN'
                      autoComplete='current-password'
                      value={field.value}
                      onChange={field.onChange}
                      onComplete={(pin) => {
                        if (!/^\d{4}$/.test(pin)) {
                          return
                        }
                        void loginWithPin(selectedUser.id, pin)
                      }}
                      disabled={isSubmitting}
                    >
                      <InputOTPGroup>
                        {Array.from({ length: 4 }).map((_, index) => (
                          <InputOTPSlot key={index} index={index} masked />
                        ))}
                      </InputOTPGroup>
                    </InputOTP>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className='flex flex-col items-center gap-2'>
              <Button
                type='button'
                variant='ghost'
                onClick={() => setSelectedUserId(null)}
              >
                <ArrowLeft data-icon='inline-start' />
                返回用户选择
              </Button>
              <Button
                type='button'
                variant='link'
                onClick={() => openPasswordLogin(selectedUser.username)}
              >
                使用密码登录
              </Button>
            </div>
          </form>
        </Form>
      </div>
    )
  }

  if (showSwitcher) {
    return (
      <div
        className={cn('mx-auto grid w-full max-w-lg gap-6', className)}
        {...props}
      >
        <div className='mx-auto grid w-fit max-w-full gap-6'>
          <div className='flex flex-wrap items-center justify-center gap-3'>
            {loginUsers.map((profile) => (
              <Button
                key={profile.id}
                type='button'
                variant='outline'
                className='h-auto w-36 flex-col gap-3 rounded-lg px-4 py-5'
                onClick={() => openPINLogin(profile.id)}
              >
                <Avatar className='size-20'>
                  <AvatarImage
                    src={profile.avatar_url}
                    alt={profile.username}
                  />
                  <AvatarFallback className='text-2xl'>
                    {getUserInitial(profile.username)}
                  </AvatarFallback>
                </Avatar>
                <span className='max-w-full truncate'>{profile.username}</span>
              </Button>
            ))}

            <Button
              type='button'
              variant='outline'
              className='h-auto w-36 flex-col gap-3 rounded-lg px-4 py-5'
              onClick={() => openPasswordLogin()}
            >
              <span className='flex size-20 items-center justify-center rounded-full bg-muted'>
                <Plus className='size-8' />
              </span>
              <span className='max-w-full truncate'>添加用户</span>
            </Button>
          </div>

          {authHeader}
        </div>
      </div>
    )
  }

  return (
    <div
      className={cn('grid w-full justify-items-center gap-6', className)}
      {...props}
    >
      <div className='grid justify-items-center gap-3 text-center'>
        <span className='flex size-20 items-center justify-center rounded-full bg-muted'>
          <UserRound className='size-8' />
        </span>
        <div>
          <h2 className='font-semibold tracking-tight'>账号密码登录</h2>
        </div>
      </div>

      <Form {...passwordForm}>
        <form
          onSubmit={passwordForm.handleSubmit(onPasswordSubmit)}
          className='grid w-full content-start gap-4'
        >
          <FormField
            control={passwordForm.control}
            name='username'
            render={({ field }) => (
              <FormItem>
                <FormLabel>用户名</FormLabel>
                <FormControl>
                  <Input
                    placeholder='admin'
                    autoComplete='username'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={passwordForm.control}
            name='password'
            render={({ field }) => (
              <FormItem className='relative'>
                <FormLabel>密码</FormLabel>
                <FormControl>
                  <PasswordInput
                    placeholder='请输入密码'
                    autoComplete='current-password'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
                <Link
                  to='/forgot-password'
                  className='absolute inset-e-0 -top-0.5 text-sm font-medium text-muted-foreground hover:text-foreground'
                >
                  忘记密码？
                </Link>
              </FormItem>
            )}
          />
          <Button disabled={isSubmitting}>
            {isSubmitting ? <Spinner /> : <LogIn data-icon='inline-start' />}
            登录
          </Button>
        </form>
      </Form>

      <div className='flex w-full items-center justify-between'>
        {loginUsers.length > 0 ? (
          <Button
            type='button'
            variant='ghost'
            onClick={() => setShowPasswordLogin(false)}
          >
            <ArrowLeft data-icon='inline-start' />
            返回用户列表
          </Button>
        ) : (
          <span />
        )}
        <Link
          to='/sign-up'
          className='text-nowrap underline underline-offset-4 hover:text-primary'
        >
          注册
        </Link>
      </div>

      {!loginUsersLoading && loginUsers.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <UserRound />
            </EmptyMedia>
            <EmptyTitle>还没有可切换用户</EmptyTitle>
            <EmptyDescription>首次使用请先通过已有账号登录。</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : null}
    </div>
  )
}

function getUserInitial(username: string) {
  return username.trim().slice(0, 1).toUpperCase() || 'U'
}
