import { createFileRoute } from '@tanstack/react-router'

import FavoritesPage from '#/features/favorites'

export const Route = createFileRoute('/_app/favorites')({
  component: FavoritesRoute,
})

function FavoritesRoute() {
  return <FavoritesPage />
}
