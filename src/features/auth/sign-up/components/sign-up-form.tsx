import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Loader2, UserPlus } from 'lucide-react'
import { toast } from 'sonner'
import { createMiboApi, getApiBaseUrl } from '@/lib/mibo-api'
import { handleServerError } from '@/lib/handle-server-error'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/password-input'

const formSchema = z
  .object({
    username: z.string().trim().min(1, '请输入用户名。'),
    password: z
      .string()
      .min(1, '请输入密码。')
      .min(8, '密码长度至少需要 8 个字符。'),
    confirmPassword: z.string().min(1, '请确认密码。'),
    pin: z
      .string()
      .min(1, '请输入 PIN。')
      .length(4, 'PIN 必须是 4 位数字。')
      .regex(/^\d+$/, 'PIN 只能包含数字。'),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: '两次输入的密码不一致。',
    path: ['confirmPassword'],
  })

export function SignUpForm({
  className,
  ...props
}: React.HTMLAttributes<HTMLFormElement>) {
  const navigate = useNavigate()

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: '',
      password: '',
      confirmPassword: '',
      pin: '',
    },
  })

  const registerMutation = useMutation({
    mutationFn: (data: z.infer<typeof formSchema>) =>
      createMiboApi({ baseUrl: getApiBaseUrl() }).register(
        data.username,
        data.password,
        data.pin
      ),
    onSuccess: async (user) => {
      toast.success(`已为 ${user.username} 创建账户，请登录。`)
      form.reset()
      await navigate({
        to: '/sign-in',
        search: { redirect: undefined },
        replace: true,
      })
    },
    onError: handleServerError,
  })

  function onSubmit(data: z.infer<typeof formSchema>) {
    registerMutation.mutate(data)
  }

  const isLoading = registerMutation.isPending

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className={cn('grid gap-3', className)}
        {...props}
      >
        <FormField
          control={form.control}
          name='username'
          render={({ field }) => (
            <FormItem>
              <FormLabel>用户名</FormLabel>
              <FormControl>
                <Input placeholder='alice' {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='password'
          render={({ field }) => (
            <FormItem>
              <FormLabel>密码</FormLabel>
              <FormControl>
                <PasswordInput placeholder='********' {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='confirmPassword'
          render={({ field }) => (
            <FormItem>
              <FormLabel>确认密码</FormLabel>
              <FormControl>
                <PasswordInput placeholder='********' {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='pin'
          render={({ field }) => (
            <FormItem>
              <FormLabel>PIN</FormLabel>
              <FormControl>
                <PasswordInput
                  placeholder='4 位数字'
                  inputMode='numeric'
                  maxLength={4}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button className='mt-2' disabled={isLoading}>
          {isLoading ? <Loader2 className='animate-spin' /> : <UserPlus />}
          创建账户
        </Button>
      </form>
    </Form>
  )
}
