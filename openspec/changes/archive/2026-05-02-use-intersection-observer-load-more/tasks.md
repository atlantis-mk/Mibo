## 1. Library Load-More Trigger

- [x] 1.1 Inspect the library page scroll container and bottom load-more element refs to confirm the observer root and sentinel target.
- [x] 1.2 Replace the manual scroll-distance load-more effect with an IntersectionObserver effect rooted to the library scroll container.
- [x] 1.3 Configure the observer with early-load margin behavior comparable to the current near-bottom preload threshold.

## 2. State Guards And UX Preservation

- [x] 2.1 Preserve existing `useInfiniteQuery` pagination, offset/limit request parameters, and `getNextPageParam` behavior.
- [x] 2.2 Guard observer-triggered requests so they only run for browse tabs when `hasMore` is true, no next page is already fetching, and no browse error is blocking automatic retry.
- [x] 2.3 Preserve the existing bottom status UI for loading, load-more hint, loaded-all, partial-error, and retry states.

## 3. Verification

- [x] 3.1 Run the frontend typecheck from `web/`.
- [x] 3.2 Manually verify a library with more than one page auto-loads more results when scrolling near the bottom.
- [x] 3.3 Manually verify filters or sort changes still reset to the correct first result page and subsequent automatic loads use the active filters.
