## Context

The current catalog experience already stores people relationships through `people` and `item_people`, and media detail pages render those relationships in `PeopleSection` as non-navigable cards. That data model is intentionally thin today: `database.Person` only stores `name`, `sort_name`, and `avatar_url`, while the catalog item APIs only expose `CatalogPersonDetail { name, role, avatar_url }`.

The requested change needs a new browsing surface rather than another metadata row. The page should feel like an Emby-style person profile: a large portrait-led hero, readable facts such as birth date and place, a biography block, related works that route back into title detail pages, and direct links out to IMDb and TMDB when known. At the same time, Mibo does not currently have person routes, person detail contracts, or persisted provider IDs for people, so the change crosses frontend routing, catalog contracts, metadata syncing, and database storage.

## Goals / Non-Goals

**Goals:**

- Add a dedicated authenticated person detail route in the web app.
- Make media detail cast and director cards open the person detail route.
- Expose a catalog person detail API keyed by Mibo's internal person ID.
- Persist enough person metadata to render biography, birth facts, portrait, and external links reliably.
- Build the related works shelf from Mibo's local catalog relationships so the page only shows titles that exist in this library.
- Preserve graceful fallbacks when person metadata is sparse or external provider refresh fails.

**Non-Goals:**

- Add person editing, governance, or manual rematch flows.
- Introduce person-scoped favorites or any new per-user person management state.
- Build a standalone people search index or people browsing landing page.
- Depend on a new metadata provider beyond the existing TMDB-based catalog metadata flow.

## Decisions

1. Use internal catalog person IDs as the route and API key.

   The new surface will use `/api/v1/people/{id}` on the backend and a matching `/person/$id` route on the frontend. `CatalogPersonDetail` returned inside media detail payloads will gain the internal `id` so people cards can deep-link without guessing from names.

   This is safer than routing by person name, which is not stable and can collide, and simpler than exposing provider IDs directly in the URL before Mibo has guaranteed provider coverage for every person row.

2. Expand `database.Person` with cached profile fields and refresh them lazily from TMDB when needed.

   The person table should grow additive nullable fields for metadata that the new page needs: TMDB person ID, IMDb ID, biography, birthday, deathday, place of birth, known-for department, and an optional refresh timestamp. During catalog metadata ingestion, cast and crew rows should capture the provider person ID from TMDB credits so Mibo can later resolve richer profile data for that person.

   The person detail read path will use the cached row first. When the row has a TMDB person ID and important profile fields are missing or stale, the service may perform a best-effort refresh from the existing metadata provider configuration before building the response. This avoids fetching full person profiles for every cast member during media scans, while still allowing detail pages to become richer over time.

   An alternative was to fetch every credited person's full profile during item metadata apply or refetch. That would significantly increase metadata fan-out and make item refresh latency dependent on cast size. Another alternative was to keep person metadata entirely ephemeral and fetch live on every person page view, but that would make person browsing fragile when TMDB is unavailable and would not improve local search or cached read quality.

3. Build related works from local `item_people` joins and return them as catalog list items ordered for browsing.

   The person detail page should only send users to titles that already exist in this Mibo workspace. The backend will therefore query `item_people` for the person, join the related `catalog_items`, and map them into `CatalogListItem` cards that reuse existing poster-card components.

   The result order should prioritize useful browsing over raw insertion order: available titles first, lower `item_people.sort_order` first, newer release year or air date next, then stable title ordering. The frontend can then reuse the first related item with usable backdrop artwork as the hero background fallback, while the same ordered list feeds the works rail.

   Using TMDB's external credits as the works shelf source was considered, but that would show titles that are not present in the local library and would break the requested "人物展示 + 作品导流" loop back into Mibo's own catalog.

4. Keep the works area as a single catalog shelf in the first iteration.

   The requested design emphasizes a clear top-to-bottom flow: who the person is, then what they have appeared in. For the first implementation, the response should expose one primary `related_items` shelf for local titles instead of separate cast/director sections or a role timeline. This matches the requested Emby-like browsing behavior and keeps the page compatible with existing `MediaPosterCard` patterns.

   If later product work needs role-separated shelves, the API can evolve to expose grouped sections, but the initial contract should remain minimal.

5. Reuse existing external identity and poster-card patterns, with person-aware URL formatting.

   The person detail contract can expose a small external identity list for TMDB and IMDb, but frontend helpers must understand that person identities use different destination URL shapes from title identities: TMDB person links use `/person/{id}` and IMDb people links use `/name/{id}`.

   Reusing `CatalogListItem` for related titles and the existing provider-label helpers keeps the new screen consistent with the rest of the catalog UI while limiting new presentation-only types.

## Risks / Trade-offs

- [Person rows created before this change will not have TMDB person IDs] -> Existing people can still render a basic page from local name, avatar, and related titles; richer biography and external links will appear after future metadata refetches or any one-time repair/backfill chosen during implementation.
- [Lazy refresh adds latency to the first detail request] -> Prefer cached data when present, use the existing metadata timeout budget, and fall back to stored local data rather than failing the request.
- [Credits may contain duplicate or ambiguous names] -> Deep links must use internal person IDs, and related works queries should join on `person_id` rather than name.
- [Some people have no portrait, biography, or related artwork] -> The UI should render initials, omit empty fact rows, and fall back to a gradient hero instead of showing broken media.
- [A single works shelf loses explicit role context] -> The card list will optimize for navigation now; if role context becomes important later, a grouped shelf extension can be added without changing the core page identity.

## Migration Plan

- Add nullable person profile fields to `database.Person` through the existing AutoMigrate flow.
- Extend TMDB credit parsing so cast and crew rows retain provider person IDs during metadata apply/refetch and future scans.
- Add catalog query and HTTP handler support for `GET /api/v1/people/{id}` with best-effort metadata refresh and local related-item lookup.
- Extend frontend API types and query keys, add a new person route and page, and make existing person cards link to that route.
- Verify rollback safety by keeping all new fields and endpoints additive; if the UI needs to be disabled, the new route can be removed without affecting existing media detail behavior.

## Open Questions

- Should implementation include a one-time maintenance command or worker repair pass to backfill TMDB person IDs for already matched catalog items, or is future metadata refetch sufficient for the initial release?
- Should the person hero show known-for department or a condensed role summary when birth facts are missing, or keep the hero limited to only the requested biography and birth-related facts?
