## Context

The current scanner already separates expensive metadata matching and media probing into post-scan jobs, but the scan job still performs enough catalog materialization work that new files do not become browseable until classification, catalog writes, sidecar/artwork handling, missing reconciliation, and projection scheduling have progressed. Users with large sources need early feedback that files were found and are playable, even while the final media graph is still converging.

The existing model contains the right durable anchor for this: `inventory_file`. It represents storage facts such as provider, path, stable identity, size, modified time, hashes, and container independently from final catalog semantics. The fast-ingest design builds on that anchor rather than creating final-looking catalog identities too early.

## Goals / Non-Goals

**Goals:**

- Make newly discovered supported video files visible in library browsing before final movie/episode classification, ffprobe, remote metadata matching, artwork, or full projection refresh finishes.
- Preserve `inventory_file` as the stable fast-ingest anchor so later catalog graph reshaping does not lose file continuity.
- Represent discovered media maturity explicitly enough for API clients and UI to show "organizing" state.
- Keep the minimal scan critical path bounded to storage discovery, obvious filtering, inventory persistence, and lightweight visible-entry publication.
- Let asynchronous classification and enrichment upgrade discovered entries into final catalog movie, series, season, episode, asset, image, and metadata state.

**Non-Goals:**

- Replacing the existing media graph scanner, classifier, metadata operation pipeline, or ffprobe pipeline wholesale.
- Guaranteeing perfect movie/episode grouping before a discovered item is first shown.
- Making remote metadata providers, ffprobe, or frame extraction part of fast ingest.
- Changing OpenList or importing code from the upstream `OpenList/` checkout.

## Decisions

### Decision: Anchor fast ingest on `inventory_file`

Fast-ingest visibility SHALL be anchored to the physical file record rather than to a temporary final catalog item identity. Catalog items and asset links can be created or refined later from the file-backed discovery state.

Alternatives considered:

- Temporary catalog items as anchors: simpler for existing browse APIs, but later movie/episode regrouping risks breaking playback history, favorites, manual edits, and governance state attached to an identity that was never final.
- A new `scan_candidate` table as anchor: conceptually clean, but adds another durable identity layer before proving it is necessary. The initial design should reuse `inventory_file` plus explicit maturity/link state unless implementation shows that a candidate table is required.

### Decision: Show media-like discovered cards, not a file-manager view

The user-facing browse experience should render discovered files as media cards with title guesses and organizing badges. This preserves product feel while remaining honest that classification and enrichment are not complete.

Alternatives considered:

- Raw file list: fastest and least ambiguous, but it regresses the media-library experience and forces users to mentally map files to media.
- Hide until final catalog projection: cleanest visually, but fails the fast-ingest goal.

### Decision: Use maturity state as a contract boundary

Discovered entries need explicit maturity such as `discovered`, `classified`, `enriched`, and `review_required`, or equivalent API fields derived from existing rows. Clients should not infer maturity from missing poster/runtime/metadata alone.

Alternatives considered:

- Infer from nullable fields: brittle because absence of poster or runtime can be valid even after enrichment.
- Reuse governance status only: governance status captures curation authority and review state, but fast-ingest maturity is operational scan state and should remain distinguishable.

### Decision: Keep final catalog projection asynchronous from skeleton visibility

Fast ingest should publish a visible discovered entry before expensive projection refresh finishes. Full catalog rollups and search documents can update afterward, but browse APIs must have a defined way to include discovered entries in library scope.

Alternatives considered:

- Synchronously refresh library projections during scan: preserves current catalog-only browse shape, but large libraries keep paying a heavy upfront cost.
- Only expose discovered entries through a separate API: reduces catalog query complexity, but splits client browsing and increases frontend special cases.

### Decision: Upgrade in place by file anchor

When classification creates or refines catalog graph rows, the discovered card should resolve to the final item through the same file or asset link. User-visible continuity should be based on the file anchor, not on keeping a temporary item ID alive forever.

Alternatives considered:

- Mutate temporary item type/path into final item rows: simpler identity continuity, but type changes across movie/series/episode hierarchy are hard to reason about and can corrupt curation semantics.
- Delete and recreate without mapping: simpler backend, but creates flicker and loses continuity.

## Risks / Trade-offs

- Duplicate presentation risk: a discovered file and its final catalog item could both appear during transition -> Mitigate by suppressing discovered entries once their inventory file is linked to an available catalog asset/item in the same browse scope.
- Query complexity risk: browse APIs must merge catalog-backed rows and inventory-backed discovered rows -> Mitigate with a narrow response contract and focused tests for sorting, paging, filtering, and de-duplication.
- User action ambiguity: favorites or manual metadata edits on a not-yet-final discovered card may have unclear target identity -> Mitigate by limiting available actions on discovered cards or recording actions against the file anchor until final catalog linkage exists.
- Cleanup timing risk: moving missing reconciliation later could temporarily show stale content -> Mitigate by keeping obvious file disappearance state visible and ensuring cleanup/reconciliation jobs run with higher priority than enrichment where needed.
- Migration risk: existing catalog-only clients may not expect non-final entries -> Mitigate with explicit maturity fields and backward-compatible defaults for clients that ignore organizing state.
