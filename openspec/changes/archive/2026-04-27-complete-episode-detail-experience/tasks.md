## 1. Backend Catalog Detail Contracts

- [x] 1.1 Add typed episode parent-context DTO fields to catalog contracts for series ID/title/images, season ID/name/number, episode number/range, and incomplete-hierarchy state.
- [x] 1.2 Extend catalog detail query loading so episode item details populate parent series and containing season context from durable catalog hierarchy records.
- [x] 1.3 Add same-season episode shelf data for episode details or a companion season-sibling query that returns sibling episode IDs, labels, selected images, availability, runtime, overview, and current-item identity.
- [x] 1.4 Batch-load signed-in user progress for sibling episode shelves and keep empty progress valid when no progress rows exist.
- [x] 1.5 Extend catalog asset detail DTOs and loaders with file summary and media stream summaries from `asset_files`, `inventory_files`, and `media_streams`.
- [x] 1.6 Update catalog contract and query tests for episode parent context, same-season shelves, empty optional fields, and asset stream summaries.

## 2. TV Metadata And Matching

- [x] 2.1 Ensure series-root metadata sync enriches existing local episode descendants by season and episode number instead of creating duplicates.
- [x] 2.2 Persist episode-level selected stills, source evidence, external identities, air dates, runtimes, names, and overviews from provider season episode data.
- [x] 2.3 Add provider episode-credit synchronization for directors and episode/guest cast when provider data is available.
- [x] 2.4 Make episode-originated match/refetch actions report descendant-specific results such as identity retained, identity missing, provider slot missing, or hierarchy mismatch.
- [x] 2.5 Add backend metadata tests for episode descendant enrichment, provider slot mismatch preservation, and episode-level people persistence.

## 3. Governance And Repair Actions

- [x] 3.1 Add or extend governance APIs for bounded episode hierarchy correction of season and episode numbers within the current series hierarchy.
- [x] 3.2 Add conflict handling when a hierarchy correction targets an already occupied season/episode slot.
- [x] 3.3 Extend asset-link correction flows to move or copy episode asset links within the current series hierarchy while rejecting unrelated targets.
- [x] 3.4 Surface and update multi-episode asset segment mappings without changing unrelated field locks, image selections, evidence, or identities.
- [x] 3.5 Add HTTP and catalog governance tests for episode numbering repair, asset relink bounds, and multi-episode segment preservation.

## 4. Frontend API And Presentation Mapping

- [x] 4.1 Update `web/src/lib/mibo-api.ts` with episode context, sibling shelf, stream summary, file summary, and progress-aware episode DTO types.
- [x] 4.2 Update `web/src/lib/mibo-query.ts` so episode detail pages can load same-season shelf data and progress without treating the episode as a series root.
- [x] 4.3 Implement episode-aware `buildPresentedCatalogItem()` and presentation helpers for series title, `Sx:Ey` labels, artwork fallback, external episode URLs, and media type-specific layout decisions.
- [x] 4.4 Preserve existing movie and series presentation behavior while routing episode items into the episode detail presentation mode.

## 5. Frontend Episode Detail UI

- [x] 5.1 Refactor detail layout components so episode pages use a 16:9 primary visual and parent series backdrop fallback instead of the generic poster-first layout.
- [x] 5.2 Update the episode hero to show series title, `Sx:Ey - title`, air date, runtime, rating, genres, overview, availability, and clear missing-context messaging.
- [x] 5.3 Add same-season episode shelf rendering for episode pages with current-episode highlighting and progress/watched labels.
- [x] 5.4 Add playback availability handling so missing, unaired, or unlinked episodes disable play actions with clear feedback.
- [x] 5.5 Update people sections to prefer episode-level directors/cast and fallback gracefully to parent series people only when needed.

## 6. Media Stream And Track Display

- [x] 6.1 Add UI models and formatters for video, audio, subtitle, and file summary details including codec, language, title, channels, bitrate, resolution, size, container, and disposition flags when available.
- [x] 6.2 Render episode media information as grouped video/audio/subtitle/file cards instead of generic asset/governance rows.
- [x] 6.3 Add audio and subtitle choice controls that display defaults and unavailable/off states without requiring a playback URL fetch.
- [x] 6.4 Ensure selected asset choices in the detail page pass the same catalog item ID and asset ID to playback navigation.

## 7. Verification

- [x] 7.1 Run focused backend tests for catalog contracts, catalog queries, metadata TV hierarchy, governance routes, and playback asset selection.
- [x] 7.2 Run `go test ./...` from `mibo-media-server/` unless a focused failure identifies a pre-existing unrelated issue.
- [x] 7.3 Run `pnpm typecheck` from `web/`.
- [x] 7.4 Run `pnpm build` from `web/`.
- [x] 7.5 Manually verify the Vinland Saga-style flow: open series, click an episode, confirm episode hero context, same-season shelf, media stream cards, people section, play availability, and rematch/governance entry points.
