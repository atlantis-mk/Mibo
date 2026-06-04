import { createFileRoute } from '@tanstack/react-router'
import { LiveTVIndexPage } from '@/features/live-tv'

export const Route = createFileRoute('/_authenticated/live-tv/')({
  component: LiveTVIndexPage,
})
