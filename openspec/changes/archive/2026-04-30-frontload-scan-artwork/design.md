## Context

The current scan path writes catalog items and inventory files quickly, then queues metadata matching and inventory probing as asynchronous jobs. Artwork becomes visible only after one of those jobs writes selected images: remote metadata matching writes TMDB/MetaTube images, while probing can apply sibling artwork, provider thumbnails, or ffmpeg-generated fallback artwork.

This keeps scans responsive, but it delays artwork even when cheap deterministic signals were already available during directory scanning. The scanner already sees same-folder objects, sidecar evidence, storage object metadata, and item hierarchy decisions before it writes catalog rows.

## Goals / Non-Goals

**Goals:**

- Make artwork appear during the first scan pass when the source is local sibling artwork or provider thumbnail metadata.
- Seed external identities from sidecar metadata during the scan phase so later metadata jobs can fetch detail directly where possible.
- Preserve existing governance semantics: scanner-provided artwork is provisional and must not overwrite authoritative selected images.
- Keep scan latency bounded by avoiding media decoding and remote metadata requests in the blocking scan path.

**Non-Goals:**

- Do not run ffprobe or ffmpeg inline during scan.
- Do not call TMDB, MetaTube, or other remote metadata providers inline during scan.
- Do not introduce a new frontend contract; existing `selected_images` should show earlier artwork automatically.
- Do not download or cache remote artwork during scan.

## Decisions

1. Extend scan artifacts with artwork candidates and external identity hints.

   The scanner should pass discovered sibling artwork, provider thumbnails, and sidecar external IDs through the existing catalog scan write path. This keeps the behavior attached to the item being written and avoids adding request-time enrichment to catalog list APIs.

   Alternative considered: enqueue a separate fast artwork job per file. That would still delay artwork behind worker scheduling and does not use the fact that directory contents are already loaded during scan.

2. Treat scan-phase artwork as provisional selected fallback.

   Scan-phase candidates should become selected only when the item does not already have selected non-scanner artwork for that image type. Metadata-selected, manually selected, and existing governed images remain authoritative.

   Alternative considered: always replace selected artwork with same-folder files. This risks overwriting deliberate user or metadata choices and conflicts with existing governance expectations.

3. Prefer local sibling artwork over provider thumbnails for poster/backdrop slots.

   Sibling artwork is usually curated by the media owner and stable with the file. Provider thumbnails are useful fallback signals but can be transient or low quality, especially for generic file thumbnails.

   Alternative considered: use provider thumbnails first because they are readily available on the storage object. That would make poor thumbnails win over curated poster files.

4. Keep probe fallback behavior asynchronous and non-authoritative.

   The probe service may still apply provider thumbnails, sibling artwork, or ffmpeg-generated artwork for items that missed scan-phase preselection. It must continue to avoid replacing non-generated selected artwork.

   Alternative considered: remove probe artwork fallback after moving scan preselection earlier. That would regress items scanned before this change and items whose sibling data is only available from provider `Get` during probing.

5. Seed sidecar external IDs in the scan write transaction.

   When `.json` or `.nfo` sidecars contain high-confidence external IDs, scan writes should persist catalog external identities with scanner provenance. Metadata jobs can then prefer detail refresh over search when the identity is present.

   Alternative considered: keep sidecar IDs only as source evidence. That preserves audit data but misses the chance to reduce metadata job latency and matching errors.

## Risks / Trade-offs

- Provisional artwork may be lower quality than later metadata artwork -> mark it as scanner/provisional through existing source/provenance fields and allow metadata jobs to replace it.
- Same-folder artwork association can be ambiguous in folders with multiple videos -> only apply folder-level artwork when existing deterministic association rules identify a single target or when the file basename matches the candidate.
- Provider thumbnail URLs can expire or be inaccessible to the browser -> use them only as fallback and allow probe/metadata jobs to replace them.
- Scan writes become slightly more complex -> keep discovery in library scan helpers and image persistence in the catalog scan write path instead of adding new cross-package orchestration.
- Existing items without artwork will not automatically receive scan-phase artwork until rescan -> implementation can rely on normal rescan/refresh rather than adding a migration.

## Migration Plan

- Deploy code without data migration.
- New or refreshed scans will write provisional artwork and external identity hints.
- Existing asynchronous metadata/probe jobs continue to enrich or replace provisional selections.
- Rollback is safe because stored selected image rows remain valid catalog image records; disabling the code only stops future scan-phase preselection.

## Open Questions

- Whether provider thumbnail URLs should be limited to OpenList initially or accepted from any storage provider exposing `ThumbnailURL`.
- Whether scan-phase artwork should use a distinct source name in `item_images` provenance or reuse the existing local scanner metadata source.
