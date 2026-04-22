import { createFileRoute, stripSearchParams } from '@tanstack/react-router'

import { LegacyMediaItemRoute } from '~/features/app/legacy-app-shell'
import {
  MEDIA_ITEM_SEARCH_DEFAULTS,
  validateMediaItemSearch,
} from '~/lib/app-route-search'

export const Route = createFileRoute('/media/$mediaItemId')({
  validateSearch: validateMediaItemSearch,
  search: {
    middlewares: [stripSearchParams(MEDIA_ITEM_SEARCH_DEFAULTS)],
  },
  component: MediaItemRoute,
})

function MediaItemRoute() {
  const { mediaItemId } = Route.useParams()
  const search = Route.useSearch()

  return <LegacyMediaItemRoute mediaItemId={Number(mediaItemId)} search={search} />
}
