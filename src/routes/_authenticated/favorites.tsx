import { createFileRoute } from '@tanstack/react-router'
import FavoritesPage from '@/features/favorites'

export const Route = createFileRoute('/_authenticated/favorites')({
  component: FavoritesPage,
})
