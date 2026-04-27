## 1. Catalog Contract And Query Data

- [x] 1.1 Add typed catalog detail fields for displayable tags/genres and related media candidates to backend contracts.
- [x] 1.2 Load item tags for catalog detail responses using existing `tags` and `item_tags` tables, preferring genre tags when available.
- [x] 1.3 Derive deterministic related media candidates from catalog data without adding new persistence.
- [x] 1.4 Update catalog contract/query tests for additive DTO fields and empty-list behavior.

## 2. Frontend API And Presentation Mapping

- [x] 2.1 Update `web/src/lib/mibo-api.ts` types for new detail metadata, tags, related candidates, and episode progress support if exposed.
- [x] 2.2 Extend `web/src/lib/media-presentation.ts` to map ratings, official rating, year range, series status, child summary, tags/genres, external identities, and related items.
- [x] 2.3 Add formatting helpers for media rating, year ranges, season summary, provider labels, and external database URLs near existing detail utilities.

## 3. Detail Hero And Actions

- [x] 3.1 Update the hero metadata row to use media rating instead of metadata confidence and to display year range, official rating, genres, runtime, provider, and season summary when available.
- [x] 3.2 Make watched-state and favorite controls actionable with accessible labels and visible focus states.
- [x] 3.3 Move metadata governance, rematch, and reprobe actions into a more menu or secondary management grouping.
- [x] 3.4 Remove or implement decorative hero controls so every visible button has behavior or clear unavailable feedback.

## 4. Navigation Shell Semantics

- [x] 4.1 Replace decorative detail top-bar search and user icons with real navigation or menu behavior.
- [x] 4.2 Make the detail top-bar cast entry show clear unavailable feedback until casting is implemented.
- [x] 4.3 Keep detail top-bar settings and home entries wired to existing routes with accessible labels.

## 5. Episode, Specials, And Related Shelves

- [x] 5.1 Add local season selection state so numbered series seasons render through a selectable current-season shelf.
- [x] 5.2 Split season number `0` or specials-named seasons into a separate Specials/特别篇 shelf.
- [x] 5.3 Update episode cards to show `Sx:Ey` labels plus availability, date, runtime, synopsis, and progress or watched signals when available.
- [x] 5.4 Add related media poster-card shelves using catalog list item presentation and existing badge/year-range behavior.

## 6. Information Section And Verification

- [x] 6.1 Reorder the bottom information section so genres/tags, sources/studios if available, ratings, dates, and database links appear before technical asset/governance details.
- [x] 6.2 Render known external identities as clickable database links and unknown providers as labeled ID entries.
- [x] 6.3 Run focused backend tests covering catalog DTO/query changes.
- [x] 6.4 Run frontend typecheck/build checks from `web/` and fix regressions introduced by this change.
