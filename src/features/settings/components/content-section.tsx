import { Separator } from '@/components/ui/separator'

type ContentSectionProps = {
  title: string
  desc: string
  children: React.JSX.Element
}

export function ContentSection({ title, desc, children }: ContentSectionProps) {
  return (
    <div className='faded-bottom flex min-h-0 flex-1 flex-col overflow-y-auto scroll-smooth'>
      <div className='flex min-h-full flex-1 flex-col px-4 py-6 sm:px-6 lg:px-8 xl:px-10'>
        <div className='flex-none'>
          <h3 className='text-lg font-medium'>{title}</h3>
          <p className='text-sm text-muted-foreground'>{desc}</p>
        </div>
        <Separator className='my-4 flex-none' />
        <div className='w-full pb-12'>
          <div className='-mx-1 px-1.5 lg:max-w-xl'>{children}</div>
        </div>
      </div>
    </div>
  )
}
