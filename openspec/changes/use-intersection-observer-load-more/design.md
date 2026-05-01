## Context

The library detail page currently uses TanStack Query's `useInfiniteQuery` to request browse pages with the existing discovery API's `limit` and `offset` parameters. Additional pages are triggered by a scroll listener that calculates distance to the bottom of the library scroll container, while the rendered bottom load-more block already has a dedicated ref suitable for sentinel observation.

This change is scoped to the frontend library browse experience. The backend pagination contract, filters, sorting, totals, favorites behavior, and existing page composition remain unchanged.

## Goals / Non-Goals

**Goals:**

- Trigger additional browse-page requests when the bottom load-more sentinel intersects the library scroll container.
- Keep preloading behavior so the next page can be requested before the user visibly reaches the bottom.
- Preserve existing loading, loaded-all, partial-error, and retry states.
- Avoid duplicate fetches while a next-page request is already in flight or when no more pages are available.

**Non-Goals:**

- Changing the discovery API from offset pagination to cursor pagination.
- Introducing virtualized grid rendering.
- Changing media card layout, filters, sort behavior, or alpha index behavior.
- Generalizing infinite scroll across unrelated pages in this change.

## Decisions

- Use `IntersectionObserver` rather than scroll-distance math for the automatic trigger. This aligns the trigger with the bottom sentinel element already rendered for load-more status and removes manual `scrollHeight` calculations.
- Use the library scroll container as the observer `root`, not the document viewport. The browse page scrolls inside the app shell, so observing against `window` could miss or delay intersections.
- Use a positive `rootMargin` to preserve early loading. The current scroll-distance implementation preloads near the bottom; the observer should do the same rather than waiting until the sentinel is fully visible.
- Keep `useInfiniteQuery`, `getNextPageParam`, and offset/limit page calculation unchanged. The triggering mechanism is the only intended behavior change, which reduces API and cache risk.
- Recreate or update the observer when core guard state changes, including browse-tab eligibility, `hasMore`, `isFetchingNextPage`, and current errors. This keeps observer callbacks from issuing duplicate or invalid requests.

## Risks / Trade-offs

- Incorrect observer root could prevent loading more inside the custom scroll area -> Verify the ref points to the real scroll container and exercise the page on desktop and mobile viewports.
- Too-large `rootMargin` could fetch earlier than necessary -> Keep the margin close to the current preload threshold and rely on existing `isFetchingNextPage` and `hasMore` guards.
- Short result sets may intersect immediately and load multiple pages until the viewport fills -> This is acceptable when more results exist, but fetch guards must prevent concurrent duplicate requests.
- IntersectionObserver support is broad but not server-side -> Create the observer only inside effects after refs are available, preserving client-only behavior.
