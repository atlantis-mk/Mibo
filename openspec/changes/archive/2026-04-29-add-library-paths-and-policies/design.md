## Context

Mibo currently models media access as `MediaSource` records plus `Library` records with a single `RootPath`. Scanning, listener reconciliation, storage indexing, metadata matching, playback, sidecar subtitle binding, scan exclusions, and catalog projections all consume the library root either directly or through queued jobs. The catalog layer is already richer than the library configuration layer: inventory files, media assets, asset links, metadata evidence, field locks, selected images, user progress, and governance actions are separate concepts.

The change should close the library-configuration gap without copying Emby's large `LibraryOptions` object wholesale. Mibo should keep media-source/provider separation and add focused policy objects that can evolve independently.

## Goals / Non-Goals

**Goals:**

- Support multiple enabled source paths per library while keeping each path scoped to a `MediaSource` and normalized provider path.
- Define effective library policies for scanning, metadata, playback, and subtitles with safe defaults that preserve current behavior.
- Route scanner, listener, scheduled jobs, metadata matching, playback, and subtitle binding through the effective policy instead of hard-coded global assumptions.
- Preserve existing libraries by migrating their current `media_source_id` and `root_path` into the new path model.
- Expose policy and path management through API and settings UI without requiring users to configure every policy field during library creation.

**Non-Goals:**

- Replacing the existing `MediaSource` provider abstraction.
- Adding every Emby `LibraryOptions` field in one change.
- Implementing new external metadata providers beyond existing TMDB/TVDB/local sidecar support.
- Changing catalog governance semantics, field lock behavior, or existing scan exclusion precedence.
- Building a full per-user playback policy system.

## Decisions

### Store library paths separately from libraries

Create a `library_paths` table with `library_id`, `media_source_id`, `root_path`, `display_name`, `enabled`, timestamps, and optional deletion metadata. Keep `libraries.media_source_id` and `libraries.root_path` during the first implementation as compatibility columns and migrate each existing library into one enabled path.

Alternative considered: replace `Library.RootPath` with a JSON `paths` column. This was rejected because paths need independent enable/disable state, provider validation, storage indexing, listener coverage, and future UI operations.

### Store policy groups as separate records with typed DTOs

Add policy records for scan, metadata, playback, and subtitle behavior keyed by `library_id`. Use explicit columns for stable operational decisions and JSON only for extensible ordered lists such as provider priority, ignored extensions, or preferred subtitle languages.

Alternative considered: one `library_options_json` blob. This was rejected because scanner, metadata, playback, and UI code need validated defaults, queryable state, and straightforward migrations.

### Resolve an effective library configuration at service boundaries

Introduce a library service helper that returns the library, enabled paths, source/provider handles, and effective policies. Scanner, listener, scheduled jobs, metadata, and playback should call this resolver instead of directly reading `Library.RootPath` unless they are serving old compatibility fields.

Alternative considered: pass policy values through each API request and job payload. This was rejected because queued jobs must stay small, deduplicatable, and resilient to policy changes made after queueing.

### Preserve scan exclusion precedence

Manual scan exclusions and configured scan exclusion rules remain authoritative. Policy-driven ignore settings such as hidden files, ignored extensions, and sample-size thresholds run as additional scanner decisions and should record reason/source in scan summaries where applicable.

Alternative considered: convert all policy ignores into scan exclusion rules. This was rejected because library policy is scoped configuration, while scan exclusion rules are user-governed reusable rules with audit semantics.

### Keep metadata governance above provider policy

Metadata provider enablement, language, region, and provider order affect fetching and matching, but existing governance protections still prevent overwriting locked, manually curated, or review-needed catalog fields.

Alternative considered: let library policy force refetch and overwrite. This was rejected because Mibo's catalog value is trustworthy governance, not aggressive automated replacement.

### Subtitle policy controls binding and playback exposure

Subtitle policy should control whether external sidecar subtitles are bound/exposed, language preferences, strict matching, and whether unavailable subtitle sidecars are tolerated. Playback responses must remain safe and must not expose raw provider auth or signed internals.

Alternative considered: treat subtitles as only playback-time filtering. This was rejected because sidecar binding and stream records are produced during scanning and need library-level policy during ingestion.

## Risks / Trade-offs

- [Risk] Existing code paths may continue to read `Library.RootPath` and miss secondary paths. → Mitigation: add focused resolver helpers and tests for scanner, listener, scheduled scan, and API detail flows using a multi-path library.
- [Risk] Migration could create duplicate scans if old root and new path are both traversed. → Mitigation: treat `Library.RootPath` as compatibility output after migration; traversal uses `library_paths` when present.
- [Risk] Policy defaults may surprise existing users if they change behavior. → Mitigation: defaults must preserve current behavior: one enabled path, scanner enabled, local sidecars enabled as currently implemented, metadata language from existing global settings, external subtitles enabled if current binding exists.
- [Risk] Multi-source libraries can combine providers with different capabilities. → Mitigation: validate each path against its media source and resolve providers per path; do not assume one provider per library in traversal.
- [Risk] UI could become a dense Emby-style options wall. → Mitigation: group advanced policy controls under focused sections and keep create-library flow minimal.

## Migration Plan

1. Add new tables and migrate every existing library into one enabled `library_paths` row using the current `media_source_id` and `root_path`.
2. Create default policy rows lazily or during migration with values that preserve current behavior.
3. Update read APIs to include `paths` and `policies` while retaining existing top-level `media_source_id` and `root_path` for compatibility.
4. Update traversal and policy consumers to use the effective library configuration.
5. Add UI controls for managing paths and policy groups.
6. Rollback is database-backup based; compatibility columns remain populated so old code can still read the original single-root fields if a deployment is rolled back before secondary paths are added.

## Open Questions

- Should disabling the last enabled library path be rejected, or allowed while making the library effectively empty?
- Should global scan settings continue to apply as defaults only, or as an upper-level override for all libraries?
- Should metadata provider order support only known providers initially, or allow unknown provider keys for future plugins?
