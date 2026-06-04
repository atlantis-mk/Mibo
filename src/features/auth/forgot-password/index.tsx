import { Link } from '@tanstack/react-router'
import { KeyRound } from 'lucide-react'
import { AuthLayout } from '../auth-layout'
import { ForgotPasswordForm } from './components/forgot-password-form'

export function ForgotPassword() {
  return (
    <AuthLayout>
      <div className='grid w-[36rem] max-w-[calc(100vw-2rem)] justify-items-center gap-6'>
        <div className='grid justify-items-center gap-3 text-center'>
          <span className='flex size-20 items-center justify-center rounded-full bg-muted'>
            <KeyRound className='size-8' />
          </span>
          <h2 className='font-semibold tracking-tight'>找回密码</h2>
        </div>

        <ForgotPasswordForm className='w-full gap-4' />

        <div className='flex w-full justify-end'>
          <Link
            to='/sign-up'
            className='text-nowrap underline underline-offset-4 hover:text-primary'
          >
            注册
          </Link>
        </div>
      </div>
    </AuthLayout>
  )
}
