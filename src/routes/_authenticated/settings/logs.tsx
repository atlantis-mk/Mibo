import { createFileRoute } from '@tanstack/react-router'
import LogsPage from '@/features/logs'

export const Route = createFileRoute('/_authenticated/settings/logs')({
  component: LogsPage,
})
