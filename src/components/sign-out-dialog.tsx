import { useState } from 'react'
import { useNavigate, useLocation } from '@tanstack/react-router'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { createMiboApi, getApiBaseUrl } from '@/lib/mibo-api'
import { ConfirmDialog } from '@/components/confirm-dialog'

interface SignOutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SignOutDialog({ open, onOpenChange }: SignOutDialogProps) {
  const [isLoading, setIsLoading] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const { auth } = useAuthStore()

  const handleSignOut = async () => {
    setIsLoading(true)

    try {
      if (auth.accessToken) {
        await createMiboApi({
          baseUrl: getApiBaseUrl(),
          token: auth.accessToken,
        }).logout()
      }
    } catch {
      toast.error('退出登录未完全成功，正在清除本地会话。')
    } finally {
      auth.reset()
      onOpenChange(false)
      setIsLoading(false)
      navigate({
        to: '/sign-in',
        search: { redirect: location.href || '/' },
        replace: true,
      })
    }
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title='退出登录'
      desc='确定要退出登录吗？再次访问账号时需要重新登录。'
      cancelBtnText='取消'
      confirmText='退出登录'
      destructive
      isLoading={isLoading}
      handleConfirm={handleSignOut}
      className='sm:max-w-sm'
    />
  )
}
