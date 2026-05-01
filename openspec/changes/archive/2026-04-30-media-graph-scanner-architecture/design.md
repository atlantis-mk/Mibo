## Context

Mibo currently has a useful Catalog/Inventory split: `CatalogItem` represents media semantics, `MediaAsset` represents playable/openable resources, `InventoryFile` represents storage files, and `MediaStream` represents probed technical streams. The current scanner still classifies each video file independently before writing catalog rows, which makes TV grouping unstable when files in one directory contain inconsistent title signals.

The approved architecture document in `docs/media-architecture/media-graph-scanner-plan.md` selects Media Graph + Resolver Pipeline + Identity Layer + DTO Adapter as the target design. This design keeps the existing database model where it fits, adds stable identities and resolver evidence where needed, and introduces a separate Emby-like DTO layer instead of reshaping internal catalog contracts around Emby field names.

## Goals / Non-Goals

**Goals:**

- Group TV files by stable directory/work identity before episode-level classification.
- Group movie folders into one movie with multiple assets for versions and extras.
- Preserve physical file facts separately from media semantic decisions.
- Add stable scanner/provider/manual identities to prevent duplicate catalog items across rescans and title corrections.
- Introduce resolver decisions that are explainable and testable.
- Expose Emby-like Movie, Series, Season, Episode, MediaSource, and MediaStream DTOs backed by existing catalog/inventory data.
- Keep the first implementation focused on movie and TV video libraries while leaving room for music and documents.

**Non-Goals:**

- Full Emby API compatibility in the first implementation.
- A graph database or full event-sourcing system.
- A complete rewrite of Catalog, Inventory, Metadata, Playback, or Probe services.
- Music, document, and photo scanning implementation in the initial phase.
- Changing playback routes to depend on legacy media item routes.

## Decisions

### Decision: Use an in-memory Media Graph first

The scanner will build an in-memory graph of directory, file, sidecar, candidate work, and candidate asset nodes during a scan. The graph is projected to existing catalog/inventory tables after resolver decisions are made.

Alternatives considered:

- Persist the full graph immediately. This improves auditability but creates migration and cleanup complexity before the decision model stabilizes.
- Continue direct file-to-catalog writes. This is simpler but preserves the current TV split failure.

Rationale: an in-memory graph fixes grouping while minimizing schema churn. Durable identity and evidence can be added incrementally.

### Decision: Add durable catalog identities

Catalog items need stable identities that are independent of display title. Scanner identities will be based on library, media kind, group path, and slot information. Provider identities will continue to represent TMDB/TVDB/IMDb identities. Manual identities can pin corrected items.

Alternatives considered:

- Store scanner identity only in `MetadataSource.PayloadJSON`. This is easy but weak for unique constraints and reconciliation queries.
- Use catalog `Path` as the only identity. This is insufficient for provider/manual identities and future non-file media.

Rationale: a dedicated identity layer is the smallest durable model that solves duplicate creation and supports future providers.

### Decision: Keep Catalog/Inventory as the persistence target

The resolver pipeline will produce a plan that writes to `CatalogItem`, `MediaAsset`, `AssetItem`, `InventoryFile`, `AssetFile`, `MediaStream`, `MetadataSource`, `CatalogExternalID`, `ItemImage`, and `ItemPerson`.

Alternatives considered:

- Replace the persistence model with an Emby-shaped schema. This would reduce DTO mapping but tie the internal model to one compatibility target.
- Add separate movie/series/episode tables. This may help type-specific fields later but is unnecessary for the first video phases.

Rationale: current tables already model most needed relationships, and a projection approach avoids a high-risk rewrite.

### Decision: Resolve TV by series root, not file title

For TV libraries, a directory containing episode-like files will produce a single series candidate before individual episode slots are resolved. Filename title signals are evidence, not series identity.

Alternatives considered:

- Improve title cleanup and keep per-file series resolution. This cannot guarantee grouping when titles differ by language, release name, or missing series prefix.
- Require users to provide `tvshow.nfo`. This is accurate but too strict.

Rationale: TV directory/work identity solves the observed split and matches how users organize TV folders.

### Decision: Resolve movies by work folder where possible

Movie folders will produce one Movie catalog item and one or more assets for main files, versions, and extras. Single-file movies at a library root remain supported by using the file stem as the group identity.

Alternatives considered:

- Keep one movie item per video file. This fails multi-version and extras semantics.
- Force every movie into its own directory. This breaks existing loose-file libraries.

Rationale: folder grouping handles richer movie libraries while loose-file fallback preserves usability.

### Decision: Add Emby-like DTOs as an adapter

The API will expose Emby-like DTOs under Mibo-owned endpoints first. Internal catalog APIs remain available for governance and debugging.

Alternatives considered:

- Rename internal fields to Emby names. This would make governance and future non-Emby clients harder.
- Implement full Emby compatibility immediately. This adds authentication, route, and behavior expectations outside the scanner scope.

Rationale: DTO mapping gives clients the desired shape without locking the domain model to Emby.

## Risks / Trade-offs

- Resolver complexity can grow quickly -> keep resolvers small, each with focused tests and explicit decision output.
- Identity migration can create duplicates if existing items lack scanner identities -> backfill identities from current path/root relationships before relying on uniqueness.
- Directory grouping can over-group mixed folders -> apply TV-aggressive grouping only to explicit TV libraries first; mixed libraries remain conservative.
- Movie folder grouping can misclassify extras as versions -> use sidecar, filename keywords, duration, and file size together; mark ambiguous cases for review.
- DTO adapter can diverge from internal state -> build DTO tests from catalog fixtures and keep field mapping centralized.

## Migration Plan

1. Add identity storage and backfill scanner identities for existing movie, series, season, and episode catalog rows where enough path/number evidence exists.
2. Introduce in-memory graph and resolver plan generation behind existing scan entrypoints.
3. Route TV and movie scan writes through the resolver plan while preserving current inventory/probe job behavior.
4. Add Emby-like DTO mapper and Mibo-owned read endpoints.
5. Keep old catalog detail endpoints intact during migration.
6. If rollout exposes grouping issues, disable the new resolver path per library type and fall back to the previous scanner while preserving identity rows for later repair.

## Open Questions

- Whether Movie DTO `Path` should default to the work folder or the primary media source file path. Recommendation: work folder by default, primary file in compatibility mode.
- Whether `scan_decisions` should be a first-class table in the first implementation. Recommendation: defer and store decision evidence in `MetadataSource` until governance UI needs richer querying.
- Whether mixed video libraries should be enabled in the first implementation. Recommendation: defer until explicit movie and TV libraries are stable.
