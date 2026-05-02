## Why

The library detail page already supports incremental browsing, but its automatic load-more trigger is based on manual scroll-distance calculations. Moving the trigger to an IntersectionObserver sentinel makes the behavior easier to reason about, better aligned with the existing bottom load-more element, and less dependent on hand-maintained scroll math.

## What Changes

- Replace the library browse page's scroll-distance load-more trigger with a bottom sentinel observed by IntersectionObserver.
- Preserve the existing `useInfiniteQuery` flow, offset/limit API contract, result totals, loading copy, error retry behavior, and filter/sort semantics.
- Ensure the observer uses the library scroll container as its root and preloads before the user reaches the visible bottom.
- Keep manual retry available when an additional page request fails.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `library-detail-browsing`: Clarify that incremental loading automatically requests additional browse results when the load-more sentinel enters the library scroll viewport while preserving active filters.

## Impact

- Frontend only: `web/src/features/library/index.tsx` is the expected integration point.
- No backend API, schema, or storage changes.
- No new runtime dependencies are expected.
