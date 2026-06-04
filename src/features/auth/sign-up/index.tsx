import { Link } from '@tanstack/react-router'
import { UserPlus } from 'lucide-react'
import { AuthLayout } from '../auth-layout'
import { SignUpForm } from './components/sign-up-form'

export function SignUp() {
  return (
    <AuthLayout>
      <>
        <div className='grid w-[36rem] max-w-[calc(100vw-2rem)] justify-items-center gap-6'>
          <div className='grid justify-items-center gap-3 text-center'>
            <span className='flex size-20 items-center justify-center rounded-full bg-muted'>
              <UserPlus className='size-8' />
            </span>
            <h2 className='font-semibold tracking-tight'>创建账户</h2>
          </div>

          <SignUpForm className='w-full gap-4' />

          <div className='flex w-full justify-end'>
            <Link
              to='/sign-in'
              search={{ redirect: undefined }}
              className='text-nowrap underline underline-offset-4 hover:text-primary'
            >
              登录
            </Link>
          </div>
        </div>

        <p className='fixed inset-x-0 bottom-6 w-full px-4 text-center text-sm text-nowrap text-muted-foreground'>
          创建账户即表示你同意我们的{' '}
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
          。
        </p>
      </>
    </AuthLayout>
  )
}
