import { useSearch } from '@tanstack/react-router'
import { AuthLayout } from '../auth-layout'
import { UserAuthForm } from './components/user-auth-form'

export function SignIn() {
  const { redirect } = useSearch({ from: '/(auth)/sign-in' })

  return (
    <AuthLayout>
      <>
        <div className='flex w-[36rem] max-w-[calc(100vw-2rem)] flex-col gap-4'>
          <UserAuthForm redirectTo={redirect} />
        </div>
        <p className='fixed inset-x-0 bottom-6 w-full px-4 text-center text-sm text-nowrap text-muted-foreground'>
          点击登录即表示你同意我们的{' '}
          <a
            href='/terms'
            className='underline underline-offset-4 hover:text-primary'
          >
            服务条款
          </a>{' '}
          和{' '}
          <a
            href='/privacy'
            className='underline underline-offset-4 hover:text-primary'
          >
            隐私政策
          </a>
          .
        </p>
      </>
    </AuthLayout>
  )
}
