## 1. Schema And Identity Foundation

- [x] 1.1 Add persistent catalog identity storage with provider, identity type, identity key, source path, confidence, and evidence fields.
- [x] 1.2 Add catalog identity service helpers for upsert, lookup, and reconciliation by scanner/provider/manual identities.
- [x] 1.3 Backfill scanner identities for existing movie, series, season, and episode catalog items where path and hierarchy evidence is sufficient.
- [x] 1.4 Add tests for identity lookup, duplicate prevention, and title-change stability.

## 2. Media Graph And Resolver Core

- [x] 2.1 Define in-memory scan graph structures for directory, file, sidecar, candidate work, candidate asset, and resolver decision nodes.
- [x] 2.2 Refactor library scan traversal to collect directory objects before per-file catalog projection.
- [x] 2.3 Implement filename signal resolver for year, season, episode, multi-episode range, quality, edition, and extra/trailer/sample signals.
- [x] 2.4 Implement directory shape resolver for movie folders, series folders, season folders, flat episode folders, mixed folders, and unknown folders.
- [x] 2.5 Record resolver decision evidence in scanner metadata source payloads for created or updated catalog items.

## 3. TV Graph Projection

- [x] 3.1 Implement series resolver that produces one stable series candidate per TV work directory before episode classification.
- [x] 3.2 Implement season and episode resolver that derives season/episode slots from directory and filename evidence.
- [x] 3.3 Update TV catalog projection to create or reuse series, season, and episode items by scanner identity before title/path fallback.
- [x] 3.4 Preserve multi-episode asset behavior while routing it through graph projection.
- [x] 3.5 Add tests for standard TV folders, flat TV folders, Chinese episode names, inconsistent title prefixes, and multi-episode files.

## 4. Movie Graph Projection

- [x] 4.1 Implement movie resolver that creates one movie work candidate per movie folder or loose single-file movie.
- [x] 4.2 Implement asset resolver behavior for movie main files, versions, trailers, samples, and extras.
- [x] 4.3 Update movie catalog projection so movie folders create one movie item with multiple assets/media sources.
- [x] 4.4 Add tests for single-file movies, movie folders, generic `movie.mkv` files, multi-version movies, and movie extras.

## 5. Sidecar Evidence Integration

- [x] 5.1 Route basename and folder-level sidecar discovery into the media graph instead of applying only at per-file classification time.
- [x] 5.2 Apply `movie.nfo`, `tvshow.nfo`, `season.nfo`, and `metadata.json` as group-level evidence for movie and TV candidates.
- [x] 5.3 Preserve existing subtitle sidecar binding for assets created through graph projection.
- [x] 5.4 Add tests for sidecar group evidence, file sidecar evidence, ambiguous sidecars, malformed sidecars, and locked field preservation.

## 6. Metadata Provider Alignment

- [x] 6.1 Update catalog match queueing so scanner-created TV descendants continue to match through the stable series root.
- [x] 6.2 Ensure provider TV hierarchy sync enriches existing scanner-identified descendants instead of creating duplicate provider-only descendants.
- [x] 6.3 Preserve local scanner episode slots when provider hierarchy lacks a matching season/episode slot and surface governance review evidence.
- [x] 6.4 Add tests for provider sync against scanner-created series, missing provider slots, and descendant identity retention.

## 7. Emby-like DTO Adapter

- [x] 7.1 Define media DTO contracts for common item fields, provider IDs, people, images, media sources, and media streams.
- [x] 7.2 Implement mapper from catalog movie detail to Movie DTO with media sources and streams.
- [x] 7.3 Implement mapper from catalog series detail to Series DTO with child and recursive counts.
- [x] 7.4 Implement mapper from catalog season detail to Season DTO with series context and child count.
- [x] 7.5 Implement mapper from catalog episode detail to Episode DTO with series/season context and media sources.
- [x] 7.6 Add Mibo-owned media DTO API endpoints under `/api/v1/media/items`.
- [x] 7.7 Add tests for runtime tick conversion, sparse stream data, multiple media sources, and parent context fields.

## 8. Verification And Rollout

- [x] 8.1 Add fixture-based scanner tests for the full TV flat-folder and movie multi-version examples from the design.
- [x] 8.2 Run backend focused scan, catalog, metadata, and probe tests.
- [x] 8.3 Run `go test ./...` from `mibo-media-server/` and document any unrelated pre-existing failures.
- [x] 8.4 Update architecture documentation if implementation changes any decisions from the approved design.
