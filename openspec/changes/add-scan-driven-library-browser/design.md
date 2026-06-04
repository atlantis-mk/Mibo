## Context

Mibo already separates raw scan/link state from user-facing catalog reads. Scan and recognition populate `inventory_files`, `resource_files`, `resource_metadata_links`, `resource_library_links`, and `recognition_candidates`. Catalog consumers then read `library_metadata_projections`, which are rebuilt through [mibo-media-server/internal/catalog/metadata_projection.go](/Users/atlan/Desktop/IdeaProjects/Mibo/mibo-media-server/internal/catalog/metadata_projection.go) and refreshed through existing ingest dirty/projection jobs.

The current hierarchical browser bypasses that read-model boundary and derives display folders at request time from resource paths plus naming heuristics. That keeps the first version simple, but it makes folder semantics fragile, duplicates logic per request, and cannot reliably distinguish category directories from media directories across movies and series. The safest place to solve this is the projection layer because it is already library-scoped, incrementally rebuilt, and consumed by browse/detail surfaces.

## Goals / Non-Goals

**Goals:**
- Persist a scan-driven display root path for each `library_metadata_projection` row so hierarchical browse can read stable folder placement directly from catalog projections.
- Derive movie and series display roots from source files and ancestor metadata relationships, including roll-up from episodes to series and collapse across technical subdirectories such as season, split-part, and edition folders.
- Keep hierarchical browse URLs and response contracts stable while changing the server-side source of truth from request-time inference to projection fields.
- Preserve backward compatibility by allowing browse-time fallback inference when projection rows have not yet been rebuilt with display-path data.
- Reuse the existing projection refresh lifecycle so rescans and metadata/resource relinks naturally refresh browse placement.

**Non-Goals:**
- Building a separate directory-tree table as the primary implementation for this iteration.
- Introducing folder write actions such as rename, merge, move, or delete.
- Replacing current metadata recognition, resource linking, or playback resolution workflows.
- Perfectly classifying every possible collector naming convention in the first pass.

## Decisions

### 1. Store display semantics on `library_metadata_projections`

Add projection fields such as `display_root_path`, `display_parent_path`, `display_kind`, and `display_path_source` to persist where a metadata item should appear inside a library browser.

Rationale:
- `library_metadata_projections` already has the right `library + metadata item` grain.
- The data is naturally refreshed by existing projection rebuild paths.
- Browse consumers can read a stable path without rejoining raw scan tables on every request.

Alternatives considered:
- A new directory-semantics table only. Deferred because the immediate problem is item placement, not independent folder lifecycle management.
- Keeping all logic in `BrowseHierarchy`. Rejected because it duplicates path reasoning and remains heuristic-heavy.

### 2. Compute display paths during projection rebuild from source-file evidence

Projection rebuild will derive display paths only from `resource_files.role = source` joined to `inventory_files`. For direct movie resources, use the source-file directories. For series projections, gather descendant source files from season/episode metadata and derive a shared series root. Normalize the resulting path against the library root and strip structural child directories such as `Season 01`, `S01`, `第1季`, `CD1`, `Part1`, `1080p`, `BluRay`, and similar technical folders.

Rationale:
- Source files best represent the playable media location and avoid subtitles, extras, or sidecars polluting the display root.
- Descendant roll-up matches how series resources are currently stored in this codebase.
- Structural-child collapse is easier to test when applied once during projection rebuild rather than on each browse request.

Alternatives considered:
- Using all linked files. Rejected because sidecar files can incorrectly raise the common ancestor.
- Using only direct metadata resources. Rejected for series because playable files are usually attached to episodes.

### 3. Treat browse-time inference as a temporary fallback, not the primary path

`BrowseHierarchy` will first use projection display fields. If a row lacks `display_root_path`, browse may fall back to the current inference path so partially rebuilt libraries remain usable during rollout.

Rationale:
- Avoids requiring an immediate full library rebuild before the feature works.
- Gives a safe migration path for existing installations.

Alternatives considered:
- Hard failure until projections are rebuilt. Rejected because it creates unnecessary downtime and rollout friction.

### 4. Keep directory semantics implicit in projections for this change

This change will not create a full persisted folder graph. Folder nodes will continue to be synthesized at read time by grouping projections on their `display_root_path` prefixes. The important shift is that item placement becomes scan-driven and deterministic.

Rationale:
- Keeps the schema change small and compatible with existing browse consumers.
- Solves the user-visible correctness issue without inventing a second long-lived sync model.

Alternatives considered:
- Materializing every folder node in the database. Deferred until there is a proven need for folder-specific lifecycle, analytics, or write operations.

## Risks / Trade-offs

- [Structural-folder heuristics still need maintenance] → Mitigation: constrain them to a small, explicit normalization layer and cover them with focused tests for movies and series.
- [Projection schema changes require backfill] → Mitigation: keep browse fallback logic active until rebuild jobs refresh libraries with the new fields.
- [Series with unusual mixed roots may still produce ambiguous display paths] → Mitigation: choose a deterministic fallback source and record `display_path_source` so diagnostics can explain the result.
- [Grouping folders from projection paths can hide empty physical directories] → Mitigation: accept this because hierarchical browsing is meant to surface playable/recognized content, not mirror every empty path on disk.

## Migration Plan

1. Add projection schema fields for display-path semantics and update model definitions.
2. Extend projection rebuild logic to populate the new fields from scan/link state.
3. Update hierarchical browse to read projection display paths first and keep legacy inference as fallback.
4. Trigger projection rebuilds for existing libraries through existing ingest/projection refresh mechanisms.
5. After rebuild coverage is verified, optionally reduce or remove legacy inference paths in a later cleanup change.

## Open Questions

- Should `display_kind` remain a coarse enum such as `item_root` / `series_root` / `fallback`, or do we want more detailed diagnostics from the start?
- Do we want an admin-visible repair action dedicated to “rebuild display paths,” or is the existing projection refresh UX sufficient?
- Should future changes persist folder-level analytics/count metadata once projection-driven placement has stabilized?
