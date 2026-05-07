## Context

Mibo's scanner already has a source-first inventory layer and a content-shape scanner. `inventory_files` stores durable file facts, while `content_shape_profiles`, `content_shape_plans`, and `content_shape_assignments` persist directory-level grouping and assignment decisions. The missing layer is a durable per-file signal index: filename/path signals are parsed during scans and materialization, but the structured result is not independently reusable across rescans, large materialization batches, or classifier-version comparisons.

Existing constraints remain important:

- Media library semantic types are being removed; automatic source-first classification is the only intended path.
- Fast scanning must not use ffprobe, file-content hashes, remote metadata providers, artwork downloads, or media reads.
- Existing inventory, content-shape, catalog, subtitles, artwork preselection, probe scheduling, metadata matching, missing cleanup, and projection refresh behavior must stay compatible.
- Old code should be reused where it expresses the current model; redundant fallback paths should be cleaned up after coverage exists rather than preserved indefinitely.

## Goals / Non-Goals

**Goals:**

- Persist reusable per-file filename and path signals tied to file fingerprints and classifier versions.
- Reuse existing filename token/profile extraction as the single parser implementation.
- Make content-shape profile construction prefer indexed file signals instead of repeatedly parsing filenames.
- Keep high-confidence plan materialization fast for large directories and unchanged rescans.
- Prevent low-confidence directory plans from silently polluting catalog with unrelated movie or episode rows.
- Remove or bypass redundant repeated parsing paths once the indexed path is active.

**Non-Goals:**

- Do not add a user-selected movie/show/mixed library type.
- Do not introduce remote metadata or media probing into fast recognition.
- Do not redesign catalog item, media asset, or asset-link schemas beyond what is needed for signal indexing.
- Do not replace the existing content-shape scanner with a separate recognition engine.
- Do not build a new frontend review UI in this change; preserve backend evidence for existing or future review surfaces.

## Decisions

### Decision: Add `inventory_file_signals` as a durable parser-output cache

Add an additive database model for per-file signals keyed by storage provider, storage path, and classifier version. Each row stores a file fingerprint derived from file facts and enough parsed signal fields to drive directory profiling without reparsing the filename.

Rationale:

- `inventory_files` is intentionally factual and should not become semantic parser output.
- `content_shape_profiles` is directory-level and cannot answer per-file assignment questions without either JSON decoding or filename reparsing.
- A separate table allows cache invalidation by classifier version and file fingerprint while keeping current inventory semantics stable.

Alternatives considered:

- Store signals JSON directly on `inventory_files`. Rejected because it mixes physical facts with classifier-versioned semantic evidence and makes invalidation more invasive.
- Only persist signals inside content-shape assignment evidence. Rejected because assignments are downstream of plan compilation and cannot efficiently feed profile construction.

### Decision: Reuse `filenameSignalModel` extraction and map it into the database

The signal index should be populated from the existing filename signal model rather than introducing a second parser. Database columns should cover common indexed fields, with full evidence retained as JSON for diagnostics.

Rationale:

- Reusing existing parser logic avoids diverging title/year/episode/role behavior.
- Columnar common fields make profile aggregation fast and testable.
- Evidence JSON preserves detailed parser output without over-indexing every release hint.

### Decision: Build content-shape profiles from indexed signals first

Directory profile construction should accept a signal lookup for visible video paths. If all visible files have current signals, the profile uses those rows. Runtime token extraction remains as a migration/cache-miss fallback.

Rationale:

- Large directories are the main performance risk; avoiding repeated parse work matters most there.
- Keeping fallback avoids breaking scans while the signal table backfills.
- This reuses current content-shape profile, plan, and assignment code rather than replacing it.

### Decision: Keep signal indexing synchronous but cheap; keep enrichment asynchronous

Signals are derived from already-listed object facts and filenames, so they can be written during scan/materialization setup. Heavy enrichment remains asynchronous.

Rationale:

- The index is only useful if it exists before directory planning.
- Signal extraction is cheap and deterministic compared with ffprobe or provider calls.
- Scan completion still must not wait for remote metadata or technical probing.

### Decision: Treat uncertain content-shape plans as reviewable, not movie fallback pollution

When content-shape confidence is below the high-confidence threshold or review state is required, materialization should preserve evidence and avoid silently creating unrelated movie rows as a fallback wherever possible. Existing fallback code can remain only for compatibility paths that explicitly need visible local placeholders.

Rationale:

- Users asked for accuracy and low involvement, not noisy incorrect catalog rows.
- Reviewable directory-level decisions are safer than converting every ambiguous file into a movie.
- Existing fallback paths should be cleaned up when plan coverage and tests prove they are unnecessary.

## Risks / Trade-offs

- [Risk] More scan-time writes for signal rows. -> Mitigation: batch upserts, file fingerprint reuse, and unchanged-file skips.
- [Risk] Signal schema can drift from parser behavior. -> Mitigation: include classifier version in identity and invalidate on parser changes.
- [Risk] Partial signal coverage during migration can produce inconsistent profiles. -> Mitigation: profile builders fall back to runtime parsing until all visible files have current signals.
- [Risk] Avoiding low-confidence materialization can reduce immediate catalog visibility for ambiguous files. -> Mitigation: preserve inventory visibility and review decisions; support guarded placeholders only where existing UX requires them.
- [Risk] Reusing old content-shape plans after signal changes can hide conflicts. -> Mitigation: directory fingerprints and classifier versions must include signal-relevant file facts and invalidate affected plans.

## Migration Plan

1. Add the `inventory_file_signals` model and migration through existing AutoMigrate flow.
2. Add repository helpers to load, save, and reuse file signals by provider/path/classifier version and file fingerprint.
3. Populate signals during scan/materialization flows after inventory facts are known and before content-shape profile compilation.
4. Update content-shape profile construction to prefer signal rows and keep runtime parsing fallback.
5. Add tests for unchanged signal reuse, file-fingerprint invalidation, classifier-version invalidation, and large directory profile reuse.
6. Tighten low-confidence content-shape fallback behavior and add regression tests for ambiguous directories.
7. Remove or bypass redundant repeated parsing paths once indexed coverage is verified.

Rollback strategy:

- Disable indexed-signal usage and fall back to runtime filename parsing.
- Leave the additive signal table intact and ignored by older logic.
- Existing inventory and content-shape tables remain authoritative for current scan behavior.

## Open Questions

- Should ambiguous files be completely withheld from catalog projection, or materialized as guarded local placeholders for visibility?
- Which exact file fingerprint fields should be mandatory for remote providers that do not expose stable identity or modified time?
- Should source-scoped correction rules update file signals directly, or remain a higher-priority evidence layer above signals?
