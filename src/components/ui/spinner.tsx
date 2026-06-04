import { LoaderCircleIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

function Spinner({ className }: { className?: string }) {
  return <LoaderCircleIcon className={cn('animate-spin', className)} />
}

export { Spinner }
