## 1. Person Data And Catalog Contracts

- [x] 1.1 Extend `database.Person` with additive nullable profile fields needed for person detail rendering, including provider IDs and life-fact metadata.
- [x] 1.2 Add internal person IDs to `CatalogPersonDetail` and define a person detail response contract that includes profile fields, external identities, and ordered related catalog items.
- [x] 1.3 Update database and contract tests to cover the expanded person schema and JSON response shape.

## 2. Metadata Enrichment And Person Detail API

- [x] 2.1 Extend TMDB credit parsing and person syncing so cast and crew records retain TMDB person IDs alongside existing avatar and role data.
- [x] 2.2 Implement backend person detail query logic that loads the cached person profile, performs best-effort TMDB enrichment when fields are missing or stale, and builds ordered local related works from `item_people` joins.
- [x] 2.3 Add `GET /api/v1/people/{id}` routing and handler coverage for success, sparse metadata fallback, and not-found behavior.

## 3. Frontend Route, Navigation, And Page Layout

- [x] 3.1 Extend `web/src/lib/mibo-api.ts` and `web/src/lib/mibo-query.ts` with person detail types, query keys, and client methods.
- [x] 3.2 Add a new authenticated `/person/$id` route and page in `web/src/routes` and `web/src/features` that renders the portrait-led hero, biography, readable life facts, related works shelf, empty states, and external links.
- [x] 3.3 Update person cards in media detail `PeopleSection` to navigate to the new route with visible focus states and accessible labels.
- [x] 3.4 Add person-aware external identity URL formatting and hero backdrop fallback logic based on the ordered related works payload.

## 4. Verification

- [x] 4.1 Run focused backend tests covering person contract mapping, metadata enrichment, and the new person detail handler.
- [x] 4.2 Run `pnpm typecheck` from `web/` after the new route and person detail UI are wired.
- [x] 4.3 Manually verify that a media detail page can open a person detail page, that sparse people render graceful fallbacks, and that related works navigate back into local media detail pages.
