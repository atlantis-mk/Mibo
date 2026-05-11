## Context

The current homepage query composes several independent requests: recently added items, continue watching, library list, latest by library, and health issues. The problematic part is the homepage-specific dependency on libraries and `latest-by-library`. Media libraries still matter as source/configuration boundaries, but they should not define the default homepage discovery experience.

The catalog already exposes item type and projection data, while the scanner stores lower-level content-shape evidence. The homepage should consume a stable presentation contract that maps those lower-level facts into user-facing sections instead of leaking raw scanner shape names.

## Goals / Non-Goals

**Goals:**
- Make homepage discovery content-shape-first rather than library-first.
- Delete the homepage frontend request chain that calls `listLibraries()` and `latestByLibrary()` for normal rendering.
- Add a backend home sections endpoint or equivalent home feed method that returns semantic sections with titles, keys, and catalog items.
- Keep Continue Watching, recently-added hero, health-aware empty/degraded states, playback links, and detail links working.
- Keep media-library detail pages and settings behavior intact.

**Non-Goals:**
- Remove library management, library detail browsing, library IDs from governance/playback APIs, or source-scoped diagnostics.
- Expose raw scanner content shape values directly as homepage section keys.
- Redesign catalog metadata classification or scanner shape planning.
- Solve every future content category such as anime, variety, documentary, or collections in this change.

## Decisions

### Decision: Introduce homepage semantic sections

The backend will expose homepage sections as presentation-ready groups. Initial section keys should be stable and user-facing, starting with `movies` and `series` based on catalog item types. Future sections can be added without changing the homepage layout contract.

Example response shape:

```json
[
  {
    "key": "movies",
    "title": "电影",
    "items": []
  },
  {
    "key": "series",
    "title": "剧集",
    "items": []
  }
]
```

Rationale:
- The homepage asks for the sections it needs, not for implementation details like libraries.
- Section keys can remain stable while scanner and catalog internals evolve.
- The frontend can render rails uniformly without knowing SQL grouping rules.

### Decision: Use catalog presentation types before raw scanner shapes

The first implementation will group metadata-backed catalog projections by user-visible catalog type: movies and series. Content-shape scanner evidence remains upstream of catalog materialization and may influence classification, but homepage section names should not be raw values such as `episode_pack` or `movie_versions_folder`.

Rationale:
- Raw shape values describe source organization, not necessarily the content identity users expect.
- Existing catalog cards and routes already understand movie/show/series display behavior.
- This keeps the first change small and avoids introducing a second taxonomy in the UI.

### Decision: Remove the homepage latest-by-library request chain

The frontend home query will stop requesting libraries and latest-by-library sections. Home state should no longer contain `libraries`, `libraryCount`, or `latestByLibrary` solely for homepage rendering.

Rationale:
- This directly removes the inconsistent product behavior.
- Settings and library detail views can continue to request libraries where they are explicitly needed.
- Empty-state logic can be based on displayable content plus health issues rather than the existence of libraries.

### Decision: Keep recently-added hero separate initially

The recently-added hero can continue to use a flat recently-added query or be folded into the new home feed if implementation naturally supports that. The required behavioral change is removing library-grouped homepage discovery, not forcing a single monolithic endpoint.

Rationale:
- Minimizes risk and preserves current hero behavior.
- Allows incremental backend API cleanup after the homepage no longer depends on library grouping.

### Decision: Retire or de-home old endpoint usage deliberately

If `/api/v1/home/latest-by-library` has no remaining product consumer after the migration, remove its frontend API client method and backend route/handler. If another internal screen still uses it, move that screen to a non-home naming path or migrate it to the new sections contract.

Rationale:
- The user's requested outcome is to delete the old request chain, not merely hide it behind renamed UI text.
- Keeping a homepage-named latest-by-library endpoint encourages regression.

## Risks / Trade-offs

- [Risk] Users with mixed sources may lose a quick way to identify which source a card came from. Mitigation: keep source/library details available on item detail, diagnostics, and library detail pages rather than homepage grouping.
- [Risk] Movies and series may be too coarse as first content-shape sections. Mitigation: design the section contract to allow future keys without changing the renderer.
- [Risk] Existing tests assert empty state from zero libraries. Mitigation: update tests to assert zero displayable content plus health/setup conditions instead.
- [Risk] Metadata governance workspace currently reuses latest-by-library. Mitigation: identify and migrate or preserve it outside the homepage request chain before removing backend support.

## Migration Plan

1. Add a catalog/home query method that returns latest catalog items grouped into semantic homepage sections.
2. Add an authenticated route for the new homepage sections contract.
3. Add frontend API types/client method for homepage content sections.
4. Change `homeDataQueryOptions` to request recently added, continue watching, homepage sections, and health issues only.
5. Update homepage state and components from `latestLibrarySections` to `contentSections` or equivalent.
6. Remove frontend `latestByLibrary()` usage from the homepage and update any other remaining consumer.
7. Delete or de-home the old `/api/v1/home/latest-by-library` route/client method when no product consumer remains.
8. Update tests for populated, empty, degraded, and content-section homepage states.

Rollback strategy:
- Keep the old backend latest-by-library implementation until the new endpoint and frontend tests pass.
- If the new section endpoint regresses, temporarily switch only the frontend query back while preserving the new backend code for investigation.
