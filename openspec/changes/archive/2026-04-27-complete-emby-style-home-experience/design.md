## Context

The current homepage lives in `web/src/features/home/index.tsx` and `web/src/features/home/home-sections.tsx`. It loads recently added items, continue-watching count, libraries, and latest-by-library sections through `homeDataQueryOptions`, then renders a hero carousel followed by grouped latest rails.

The app already has the pieces needed for a richer media-center shell: `AppTopBar`, `AppSidebar`, `/search`, `/library/$id`, `/media/$id`, `/play/$id`, and `/settings`. However, `AppSidebar` still contains sample documentation navigation, continue-watching is typed as `unknown[]`, the homepage does not expose library entrance cards or favorites, and card metadata is too thin for an Emby-style dashboard.

Backend routes are catalog-oriented for current product flows. New data needs should stay on `/api/v1/items`, `/api/v1/assets`, catalog home endpoints, and `/api/v1/me/*` routes rather than retired legacy media endpoints.

## Goals / Non-Goals

**Goals:**

- Make the homepage a complete dashboard for library entry, recent updates, continuing playback, favorites, search, account actions, settings, and cast entry.
- Add reusable, catalog-native card presentation for poster-first media grids and rails.
- Add durable user-scoped favorites if no existing favorite model is available.
- Replace placeholder navigation with real Mibo navigation in the sidebar and top shell.
- Preserve desktop and mobile usability with vertical page scrolling and horizontal poster rails.

**Non-Goals:**

- Do not implement a paid subscription or Emby Premiere equivalent unless a Mibo product requirement is introduced later.
- Do not build new product flows on retired legacy `/api/v1/media-items/*` or `/api/v1/media-files/*` routes.
- Do not replace playback architecture solely for this homepage change.
- Do not claim full Chromecast or AirPlay support unless real discovery/control and playback handoff are implemented.

## Decisions

### Decision: Extend the existing homepage route

The implementation will evolve the current `Home` feature instead of adding a parallel dashboard route.

Rationale: The current route already handles auth hydration, data loading, top-bar composition, hero presentation, and latest-by-library sections. Extending it minimizes routing churn and keeps the default app entry stable.

Alternative considered: Create a new Emby-specific dashboard route and redirect `/` to it. This was rejected because it duplicates existing homepage behavior and increases migration complexity without improving the user-facing model.

### Decision: Use shared catalog-native card helpers

Homepage, favorites, search, and library screens all need consistent card behavior. The implementation should add shared helpers/components for title, image, type, year range, progress/count badges, and quick play eligibility.

Rationale: This avoids repeating logic across homepage rails, library grids, and favorites while staying aligned with catalog DTOs.

Alternative considered: Render bespoke cards in each section. This was rejected because badge/year/progress behavior would diverge quickly.

### Decision: Persist favorites server-side

Favorites should be user-scoped and durable. If the backend does not already expose such a model, add a small catalog-item favorite store and `/api/v1/me/favorites` routes.

Rationale: Local-only favorites would not survive devices or sessions and would conflict with the logged-in media-center model.

Alternative considered: Store favorite IDs in localStorage. This was rejected because Mibo already has authenticated users and progress data, so favorites should follow the same user-data boundary.

### Decision: Build count badges from existing summaries first

Green poster badges should prefer existing catalog/progress fields before adding new backend summary endpoints.

Rationale: `CatalogListItem.child_summary` already includes child counts and availability counts. The frontend can use those for a useful first version while backend additions are limited to missing data such as favorite state or typed continue-watching payloads.

Alternative considered: Add a full homepage aggregate API immediately. This may still be useful later, but it is unnecessary if existing endpoints can supply acceptable dashboard data.

### Decision: Cast is an explicit shell entry with bounded behavior

The homepage top bar will expose a cast entry. If real cast support is absent, the action must show a clear unavailable or coming-soon message instead of silently doing nothing.

Rationale: The requested UI includes cast access, but real casting is a separate playback/device feature. A bounded UI entry satisfies navigation discoverability without misrepresenting support.

Alternative considered: Hide cast until full support exists. This was rejected because the requested comparison specifically calls out the missing entry point.

## Risks / Trade-offs

- Continue-watching data may not currently have enough typed item detail for poster cards -> add precise client types and only add backend fields if required.
- Favorites persistence touches backend schema and user isolation -> enforce `(user_id, item_id)` uniqueness and catalog item foreign keys, then cover with focused tests.
- Count badges can be semantically ambiguous across movies, shows, seasons, and episodes -> define a deterministic priority order and keep labels simple.
- Cast entry without real casting can confuse users -> use explicit copy that says casting is not yet available when unsupported.
- More homepage data can increase load time -> reuse existing queries where possible and avoid blocking the whole page on optional sections.

## Migration Plan

1. Add shared frontend presentation helpers and reusable poster-card/library-card components.
2. Upgrade homepage data typing and section composition using existing endpoints.
3. Add backend favorites persistence and APIs only if no durable favorite capability exists.
4. Add favorites route or tab surface and wire favorite actions.
5. Replace sidebar sample navigation and add top-bar search, user, settings, and cast actions.
6. Add or refine count/year-range fields only where existing DTOs are insufficient.
7. Verify frontend typecheck/build and focused backend tests for any backend changes.

Rollback strategy: because this is mostly additive UI and user-scoped favorite data, rollback can hide new routes/sections while leaving favorite rows harmless. If backend migrations are added, keep them backward-compatible and avoid deleting existing catalog/user data.

## Open Questions

- Should favorites use a separate `/favorites` route, a homepage tab state, or both? The implementation can support both by making the top switch link to a dedicated favorites route while preserving Home as `/`.
- Should the cast entry remain a disabled dialog for this change, or should real device discovery be planned as a follow-up playback change?
- Should badge counts mean unwatched count, update count, or available child count when multiple values exist? The default design uses the first available meaningful count in that order.
