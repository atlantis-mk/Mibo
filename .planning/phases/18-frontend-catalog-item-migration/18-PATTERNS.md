# Phase 18 Pattern Map

## Target Files And Best Analogs

| Target file | Closest analogs | Reuse guidance |
|-------------|-----------------|----------------|
| `web/src/lib/mibo-api.ts` | itself, `mibo-media-server/internal/catalog/contracts.go` | add catalog-native TypeScript contracts and endpoint methods beside the legacy ones; do not mutate legacy `MediaItem`/`MediaFile` types in place |
| `web/src/lib/mibo-query.ts` | itself | keep query composition centralized in shared query helpers, not inline in feature pages |
| `web/src/lib/media-presentation.ts` | itself | keep display labels, watched-state derivation, selected-image picking, and route/view helpers in `lib/`, not duplicated across pages |
| `web/src/features/discovery/controls.tsx` | itself | preserve the existing filter-shell layout, but align the actual filter fields to what the additive catalog routes support |
| `web/src/features/home/index.tsx`, `web/src/features/home/home-sections.tsx` | themselves | keep the current hero + rail composition, but swap in catalog item data and explicit availability badges/playability rules |
| `web/src/features/library/index.tsx` | itself | continue using one top-level query and a responsive poster grid; derive badges through shared helpers |
| `web/src/features/search/index.tsx` | itself | keep search history chips and list-card layout, but stop assuming every non-movie result is a legacy `show` row |
| `web/src/features/media/index.tsx` | itself | keep the route container thin: load shared queries, wire mutations/navigation, pass DTOs to presentational components |
| `web/src/features/media/components/standalone-media-detail*.tsx` | themselves | preserve the current detail visual language, but replace primary-file and TMDB/local fallback assumptions with catalog hierarchy/assets |
| `web/src/features/play/index.tsx` | itself | keep player state local in the feature, but move playback/progress requests onto item + optional asset ids |
| `web/src/routes/_app.media.$id.tsx`, `web/src/routes/play.$id.tsx` | themselves | keep route files thin: parse params/search, then hand off to feature entry components |

## Concrete Rules To Preserve

1. **Thin route pattern stays intact.** Route files should only parse params/search and render feature entry components.
2. **Shared data access stays in `lib/`.** Feature pages should keep calling `mibo-api.ts` and `mibo-query.ts`, never raw `fetch`.
3. **Presentation logic belongs in helpers.** Labels, selected-image fallbacks, watched-state mapping, and route-view selection should live in `media-presentation.ts` rather than being rederived in each component.
4. **Existing visual language is preserved.** Keep the cinematic hero, rounded cards, glassy overlays, and large typography from the current home/detail/playback surfaces.
5. **Catalog migration is additive, not heuristic.** Once a surface moves to catalog DTOs, it must stop reading `series_title`, `source_path`, `media_item_id`, and `media_file_id` as primary UI truth.
6. **Progress invalidation follows existing React Query patterns.** Reuse the current `queryClient.invalidateQueries(...)` style from `features/media/index.tsx` after mutations.

## Known Frontend Constraints

1. `DiscoveryControls` currently exposes legacy-only filters (`genre`, `region`, `minRating`, `watchedState`) that the additive catalog item route does not support. Phase 18 should intentionally narrow or remap those controls instead of silently issuing unsupported query params.
2. Current detail and playback pages assume one primary `MediaFile`. Catalog detail/playback must instead model one item with zero or more assets and an explicit unplayable path.
3. Current series detail rendering depends on `buildPresentedMediaItem(...)`, TMDB season lookups, and local fallback episodes. Catalog hierarchy should replace all three.
4. Frontend validation for this phase is `cd web && pnpm typecheck` during iteration and `cd web && pnpm typecheck && pnpm build` before the final playback wave completes.

## Key Interfaces Already In Place

From `mibo-media-server/internal/catalog/contracts.go`:

```go
type CatalogListItem struct {
    ID                 uint
    LibraryID          uint
    Type               string
    Title              string
    AvailabilityStatus string
    GovernanceStatus   string
    ChildSummary       *CatalogChildSummary
    SelectedImages     []CatalogSelectedImage
}

type CatalogItemDetail struct {
    ID                 uint
    Type               string
    Title              string
    AvailabilityStatus string
    SelectedImages     []CatalogSelectedImage
    Seasons            []CatalogSeasonDetail
    Episodes           []CatalogEpisodeDetail
    Assets             []CatalogAssetDetail
}
```

From planned Phase 16 and 17 HTTP routes:

```text
GET  /api/v1/items
GET  /api/v1/items/{id}
GET  /api/v1/series/{id}/seasons
GET  /api/v1/items/{id}/progress
POST /api/v1/me/item-progress
GET  /api/v1/items/{id}/playback
```

From the current frontend route/feature boundary:

```tsx
export const Route = createFileRoute('/play/$id')({
  validateSearch: (search) => ({ fromStart: ... }),
})

export default function PlayExperience({ mediaItemId, fromStart = false }) {
  // feature-local player state + shared query usage
}
```
