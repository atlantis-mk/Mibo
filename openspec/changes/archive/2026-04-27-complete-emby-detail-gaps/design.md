## Context

The media detail page already renders an immersive background, poster, hero copy, playback entry, episode rails, people rails, and a technical information section. The missing pieces are mostly presentation and contract alignment: some catalog metadata is available but not projected into the presentation model, some desired metadata such as genres/tags is stored in catalog tag tables but not exposed on detail DTOs, and several hero/top-bar controls currently look interactive without complete behavior.

The implementation spans frontend presentation, catalog DTO/query projection, and navigation semantics. The page must stay within the existing `routes -> features -> lib` frontend structure and the backend catalog service/contract boundary.

## Goals / Non-Goals

**Goals:**
- Make the media detail page a consumer-facing playback surface with Emby-like metadata hierarchy, content shelves, and actions.
- Reuse existing catalog fields, selected images, external identities, rollups, progress state, and media poster card behavior where possible.
- Keep management/governance actions available but visually secondary to playback and library consumption actions.
- Expose missing catalog detail metadata through typed DTOs rather than ad hoc frontend fetches.
- Preserve desktop and mobile usability while improving keyboard/TV-style focus affordances.

**Non-Goals:**
- Implement real casting, Premiere/subscription features, or external account integrations.
- Add new metadata providers or new persisted provider tables.
- Replace the playback route or the existing progress persistence model.
- Redesign the entire home/library browsing experience outside the detail page and shared navigation entries it depends on.

## Decisions

1. Use existing catalog detail endpoints as the detail-page data boundary.

   The detail page already loads `GET /api/v1/items/{id}` and `GET /api/v1/series/{id}/seasons`. Extending those typed responses keeps data loading centralized in `mibo-api.ts` and avoids feature-local raw fetches. An alternative was to add a detail-page-specific aggregate endpoint, but that would duplicate current item detail and season hierarchy reads before there is a clear performance need.

2. Project display metadata from existing catalog structures first.

   `community_rating`, `official_rating`, `year`, `end_year`, `first_air_date`, `last_air_date`, `series_status`, `child_summary`, `external_identities`, selected images, and item tags should feed the frontend presentation model. New persistence should be avoided unless the provider data cannot already be represented by catalog items, tags, external identities, rollups, or source evidence.

3. Treat genres as catalog tags in detail DTOs.

   The catalog database already has `tags` and `item_tags`, and search projections already consume them. The detail response should expose a user-facing tag list grouped or filtered enough for genres rather than forcing the frontend to parse search text or source payloads. If tag kinds are consistently populated, genre display should prefer `kind=genre`; otherwise it may display the available catalog tags as media metadata.

4. Keep the primary action row consumer-oriented.

   Playback, watched state, favorite, and a more menu remain in the hero action row. Governance actions such as metadata management, rematch, and reprobe move into the more menu or a secondary management area. This preserves current capabilities without making the detail page read like an admin tool.

5. Implement season focus with local UI state, not routing.

   Season selection is a view state of the detail page. Keeping it local avoids changing route contracts and keeps deep links stable. Specials are identified by season number `0` or provider/special naming and rendered in a separate section when present.

6. Build related shelves from catalog-backed candidates.

   The first implementation should use deterministic catalog candidates available from the backend, such as sibling/child recommendations or same-library/same-tag items. Cards should reuse existing poster card presentation and badge/year-range helpers so the detail page does not fork card behavior.

## Risks / Trade-offs

- Catalog tags may not be consistently classified by kind → Display tags defensively and avoid labeling them as precise genres unless the data supports it.
- Related media quality may be basic at first → Use deterministic candidates with clear ordering and leave advanced recommendation scoring out of scope.
- Moving management actions into a menu may reduce discoverability for admins → Keep labels explicit in the more menu and preserve existing navigation targets.
- Extending DTOs can affect contract tests → Update catalog contract tests alongside type changes and keep fields additive.
- TV/keyboard focus can regress if decorative controls remain focusable → Ensure every focusable icon has behavior or is removed from the tab order with clear unavailable feedback where appropriate.

## Migration Plan

- Add backend DTO fields and query projection as additive changes.
- Update frontend API types and presentation mapping to consume both new and existing fields.
- Refactor the detail page UI in place, preserving existing routes and playback navigation.
- Run backend catalog tests for DTO/query changes and frontend typecheck/build from `web/`.
- Rollback is straightforward because changes are additive: frontend can ignore new fields, and backend can continue serving previous clients.

## Open Questions

- Should non-genre tags be shown in the media metadata area when no explicit genre tags exist?
- Should related shelves prefer same-library items, same-genre items, or provider-related children when multiple candidate sets are available?
- Should episode cards launch playback directly or keep detail navigation as the primary click target with a separate play action?
