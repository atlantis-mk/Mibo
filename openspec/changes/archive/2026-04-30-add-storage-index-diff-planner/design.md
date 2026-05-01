## Context

Mibo currently refreshes libraries through explicit scan jobs, scheduled scans, and an external `POST /api/v1/storage-events` endpoint. The listener service already debounces storage events, chooses a targeted or full refresh scope, and hands work to existing library scan jobs. The scanner then writes inventory files, media assets, catalog items, probe jobs, metadata jobs, and projection refreshes.

The missing piece is reliable change discovery. Local file-system events can be fast but lossy, especially during recursive directory changes, network volumes, editor/download temp-file workflows, and rename sequences. OpenList does not provide a local file-system event stream and may cache remote storage state. A persistent storage index provides a provider-neutral memory of observed storage state so local watcher hints, OpenList polling, external event hints, and periodic reconciliation can all feed the same planning path.

## Goals / Non-Goals

**Goals:**

- Detect create, update, delete, move, and rename changes automatically for local and OpenList-backed libraries.
- Persist provider observations so missed watcher events and OpenList polling gaps can be corrected by reconciliation.
- Use a diff planner to produce bounded refresh scopes and avoid full scans for ordinary localized changes.
- Preserve the existing scanner as the authoritative writer for catalog and inventory state.
- Reuse the existing listener debounce, targeted refresh, full scan, probe, metadata match, and projection refresh workflows where possible.
- Keep provider-specific details inside storage adapters, provider-specific observers, or source orchestration instead of spreading OpenList/local branching through scanner logic.

**Non-Goals:**

- Do not directly mutate `catalog_items`, `media_assets`, or scanner-owned `inventory_files` state from raw file events.
- Do not modify upstream `OpenList/` code.
- Do not guarantee immediate millisecond-level consistency; the target is near-real-time for local libraries and polling-bounded consistency for OpenList.
- Do not replace manual scans, scheduled scans, or listener reconcile jobs; they remain safety nets.
- Do not implement content hashing for every local file by default, because it is too expensive for large media libraries.

## Decisions

### Persistent Storage Index

Add storage index tables keyed by `library_id`, `storage_provider`, and normalized `storage_path`. Each row records whether the path is a directory, the last observed size, modified time, stable identity, hash evidence, provider/object metadata, observation timestamps, and an observation status such as `present`, `missing`, or `unknown`.

Rationale: a persistent index lets the system compare current provider observations with prior state even when events are missing or out of order. It also gives delete handling enough context to know what disappeared under a removed directory.

Alternative considered: watcher-only event processing. This was rejected because file-system events are hints, not durable state, and OpenList cannot use them directly.

### Scanner-Owned Catalog Writes

The diff planner SHALL enqueue refresh work; it SHALL NOT create or delete catalog rows directly. For present media paths, the planner enqueues targeted refreshes that call existing scan code. For missing paths, it can enqueue a parent or scoped refresh so existing cleanup logic marks inventory and catalog availability as missing.

Rationale: scanner code already owns classification, sidecars, metadata evidence, asset linking, stable identity reuse, probe jobs, and projection refreshes. Keeping one writer avoids drift.

Alternative considered: directly marking files missing from the planner. This is only acceptable if implemented through a scanner-owned service path with the same availability/projection semantics; raw planner writes are rejected.

### Provider Observers Feed One Planner

Implement provider observers as inputs to the storage index and planner:

- Local observer: uses recursive file-system events for quick hints and periodic reconciliation walks for correctness.
- OpenList observer: uses polling snapshots from existing provider `List`/`Get` capabilities and stores observed provider identity/hash metadata.
- HTTP storage events: remain external hints and are normalized into the same planning path.

Rationale: using the same planner keeps behavior consistent across local, OpenList, and external event sources.

Alternative considered: separate local watcher path and OpenList polling path that both enqueue scans independently. This was rejected because it duplicates debounce, move detection, delete handling, and observability logic.

### Identity-Based Move Detection With Fallbacks

The planner identifies moves and renames using stable identity first. For OpenList this uses provider-supplied stable identity, hash evidence, or provider metadata. For local storage, the implementation should add stable file identity from device and inode where supported. If no stable identity is available, the planner falls back to bounded heuristics such as size, modified time, container, and nearby path similarity. If confidence is low, it plans delete plus create rather than a move.

Rationale: accurate move detection keeps existing inventory file, asset, catalog item, and playback references stable. Low-confidence guesses must not merge unrelated movies or episodes.

Alternative considered: path-only detection. This cannot distinguish rename from delete/create and loses the playback path update benefit.

### Debounce and File Stability

Observers and the planner should delay scans for paths that are actively changing. The planner records pending changes and only emits refresh plans after a short quiet period or after repeated observations show stable size/mtime.

Rationale: downloaders and copy operations often expose partial files. Scanning too early causes probe failures and noisy catalog churn.

Alternative considered: immediate scan for every write event. This is too noisy for media libraries and creates excessive jobs.

### Bounded Planning and Full-Scan Fallback

The planner emits the smallest safe refresh scope: file parent directory, changed directory, common ancestor for multiple changes, or full library root when the event is ambiguous or the affected paths are too dispersed. The existing listener merge window can remain in front of targeted refresh jobs, but the planner owns the decision about when a full sync is safer.

Rationale: most changes are localized, but ambiguous bulk operations need correctness over minimal work.

Alternative considered: always targeted scan. This fails for deleted directories when the deleted path no longer resolves and for broad moves that cross distant subtrees.

## Risks / Trade-offs

- Index drift after crashes or provider errors -> periodic reconciliation walks compare provider state with the persistent index and repair drift.
- OpenList polling can be stale because of upstream cache -> use `refresh=true` selectively for reconcile or suspected changed directories, not every polling request.
- Local recursive watchers can exceed OS watch limits -> fall back to reconciliation-only mode for affected libraries and surface status in diagnostics.
- Move heuristics can incorrectly merge unrelated files -> require stable identity for high-confidence moves; use delete-plus-create when confidence is low.
- Storage index tables can grow large -> index by library/path/status and prune stale missing non-media directories after retention if they are not needed for reconciliation.
- Bulk imports can create too many jobs -> coalesce pending changes by library and common ancestor before enqueueing refreshes.
- New background observers increase operational complexity -> keep manual scan and scheduled scan behavior unchanged as fallback and rollout path.

## Migration Plan

1. Add database migrations for storage index and planner state tables without changing current scan behavior.
2. Backfill initial storage index rows during the first reconcile pass per active library.
3. Enable planner-generated targeted refreshes behind a scan/listener setting or conservative default.
4. Enable local observer and OpenList polling per provider after the index path is verified.
5. Keep scheduled full scans and listener reconcile enabled during rollout.
6. Roll back by disabling observers/planner scheduling; existing manual and scheduled scans continue to work, and storage index tables can remain inert.

## Open Questions

- What default polling interval should OpenList use for small, medium, and large libraries?
- Should observer status be exposed only in admin APIs first, or also in the frontend scan settings page?
- Should local inode identity be stored as provider metadata, stable identity, or both?
- What retention period should apply to missing directory index rows after catalog state has been reconciled?
