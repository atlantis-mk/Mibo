import { createFileRoute } from '@tanstack/react-router'
import { Badge } from '@/components/ui/badge'
import { Main } from '@/components/layout/main'

export const Route = createFileRoute('/_authenticated/person/$id')({
  component: PersonPlaceholderPage,
})

function PersonPlaceholderPage() {
  const { id } = Route.useParams()

  return (
    <Main
      className='flex min-h-svh items-center justify-center px-6 py-12'
      fluid
    >
      <div className='max-w-xl space-y-4 text-center'>
        <Badge variant='outline'>人物详情</Badge>
        <h1 className='text-3xl font-semibold tracking-tight'>
          人物页占位已接通
        </h1>
        <p className='text-sm leading-7 text-muted-foreground'>
          当前人物 ID：{id}
        </p>
      </div>
    </Main>
  )
}
