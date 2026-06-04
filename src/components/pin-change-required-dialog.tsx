import { useState } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/auth-store'
import { ApiError, createMiboApi, getApiBaseUrl } from '@/lib/mibo-api'
import { miboQueryKeys } from '@/lib/mibo-query'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from '@/components/ui/input-otp'
import { Spinner } from '@/components/ui/spinner'

const pinFormSchema = z.object({
  pin: z
    .string()
    .length(4, 'PIN 必须是 4 位数字。')
    .regex(/^\d+$/, 'PIN 只能包含数字。'),
})

type PINFormValues = z.infer<typeof pinFormSchema>

export function PINChangeRequiredDialog() {
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const accessToken = useAuthStore((state) => state.auth.accessToken)
  const user = useAuthStore((state) => state.auth.user)
  const setSession = useAuthStore((state) => state.auth.setSession)
  const queryClient = useQueryClient()

  const form = useForm<PINFormValues>({
    resolver: zodResolver(pinFormSchema),
    defaultValues: { pin: '' },
  })

  const updatePINMutation = useMutation({
    mutationFn: (pin: string) =>
      createMiboApi({
        baseUrl: getApiBaseUrl(),
        token: accessToken,
      }).updateOwnPin(pin),
    onMutate: () => {
      setErrorMessage(null)
    },
    onSuccess: (updatedUser) => {
      if (accessToken) {
        setSession({ token: accessToken, user: updatedUser })
        queryClient.setQueryData(
          miboQueryKeys.authUser(accessToken),
          updatedUser
        )
      }
      form.reset({ pin: '' })
    },
    onError: (error) => {
      setErrorMessage(
        error instanceof ApiError ? error.message : 'PIN 修改失败，请重试。'
      )
    },
  })

  const isOpen = Boolean(accessToken && user?.requires_pin_change)
  const isSubmitting = updatePINMutation.isPending

  function onSubmit(data: PINFormValues) {
    updatePINMutation.mutate(data.pin)
  }

  return (
    <Dialog open={isOpen}>
      <DialogContent showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>修改 PIN</DialogTitle>
          <DialogDescription>
            当前账号正在使用默认 PIN。请设置新的 4 位 PIN 后继续使用。
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form className='grid gap-4' onSubmit={form.handleSubmit(onSubmit)}>
            <FormField
              control={form.control}
              name='pin'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>新 PIN</FormLabel>
                  <FormControl>
                    <InputOTP
                      maxLength={4}
                      autoComplete='new-password'
                      value={field.value}
                      onChange={field.onChange}
                      disabled={isSubmitting}
                    >
                      <InputOTPGroup>
                        {Array.from({ length: 4 }).map((_, index) => (
                          <InputOTPSlot key={index} index={index} />
                        ))}
                      </InputOTPGroup>
                    </InputOTP>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {errorMessage ? (
              <p className='text-sm text-destructive'>{errorMessage}</p>
            ) : null}

            <DialogFooter>
              <Button disabled={isSubmitting}>
                {isSubmitting ? <Spinner /> : null}
                保存 PIN
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
