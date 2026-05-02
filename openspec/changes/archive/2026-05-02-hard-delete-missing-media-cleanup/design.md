## Context

Scanning currently marks missing files by setting inventory, asset, and catalog availability/status fields to `missing` while preserving rows with `deleted_at = NULL`. Browse APIs hide unavailable media, but the database retains old scanner facts, catalog items, asset links, streams, progress, favorites, and governance/metadata state indefinitely.

The desired behavior is deliberately destructive: after a retention window, missing media should be hard deleted along with associated user and governance state. Favorites, playback records, manual matches, and manual corrections are not retention guards for this change; they are part of the graph being removed.

## Goals / Non-Goals

**Goals:**

- Add a scheduled cleanup path for missing media that has remained missing beyond a configurable retention period.
- Hard delete eligible rows rather than setting `deleted_at`.
- Delete associated records for inventory files, assets, catalog items, playback progress, favorites/user item data, metadata evidence, external IDs, tags, people, images, streams, and link tables.
- Keep scan behavior safe and immediate: scans mark missing first; cleanup deletes later.
- Make cleanup idempotent and bounded so it can run repeatedly without corrupting remaining catalog data.

**Non-Goals:**

- Preserving favorites, playback history, manual matching, or manual correction data for media that is hard deleted.
- Deleting files from OpenList or the local filesystem.
- Replacing current missing marking semantics during `sync_library`.
- Adding a recycle bin or restore UI.

## Decisions

### Use a retention-based scheduled hard cleanup

Cleanup will operate on media that is already marked missing and older than a retention threshold. The default retention should be conservative enough to avoid deleting records during transient OpenList outages, while still preventing indefinite accumulation. A retention value of `0` can mean delete as soon as cleanup runs, if configuration needs that behavior.

Alternative considered: hard delete immediately during scan. That would make deletion faster but creates higher risk during temporary storage outages and makes scan failures harder to recover from.

### Track missing age explicitly where needed

The implementation should use a reliable timestamp for when a row first became missing. If existing schema does not have a missing timestamp, add one to inventory and/or derive cleanup candidates from rows whose status transitioned to missing during scan. Avoid relying only on `updated_at`, because later maintenance can modify rows without representing recovery or deletion age.

Alternative considered: use `updated_at` for retention. It is simpler but fragile and can keep missing data forever if unrelated updates touch the row.

### Hard delete by graph scope in dependency order

Cleanup should identify root catalog items and related inventory/assets for the missing graph, then delete dependents before principals. The delete order should explicitly cover join and dependent tables to avoid orphaned data and to support SQLite and Postgres without relying on database cascade behavior being complete.

Expected dependent areas include asset files, asset items, media streams, item rollups, search documents, item images, item people, item tags, catalog external IDs, catalog identities, metadata field states, metadata sources/operations where scoped to deleted items, user item data, playback progress, and scan exclusion rows linked to deleted files/assets/items.

Alternative considered: rely entirely on foreign key cascades. The current schema and SQLite usage make explicit deletion safer and more testable.

### Treat user and manual governance data as part of deletion

The cleanup must not skip media because it has favorites, watch progress, manual match state, or manual corrections. Those rows should be deleted together with the missing catalog/inventory graph to match the user's requested destructive cleanup semantics.

Alternative considered: protect user-touched rows. That is safer for general media servers, but conflicts with the explicit requirement to jointly delete those records.

### Rebuild projections after cleanup

After hard deletion, cleanup should refresh catalog projections for affected libraries/root paths so browse/search counts no longer reference deleted items. This can be done inline for small scopes or through existing projection refresh jobs.

Alternative considered: wait for the next scan to rebuild projections. That leaves stale search documents and counts visible longer than necessary.

## Risks / Trade-offs

- [Risk] Temporary OpenList outage marks many files missing and cleanup later deletes them → Mitigation: retain a configurable delay and keep cleanup disabled or conservative by default where appropriate.
- [Risk] Hard delete removes user history and manual curation permanently → Mitigation: make the destructive behavior explicit in settings/admin labels and keep cleanup task logs/results auditable.
- [Risk] Partial cleanup leaves orphaned rows → Mitigation: run cleanup in transactions and add tests that seed related rows across all dependent tables.
- [Risk] Large cleanup transactions can lock SQLite → Mitigation: process candidates in batches and commit between batches.
- [Risk] Deleting parent series/seasons with mixed available/missing children could remove too much → Mitigation: only delete catalog parents when their local descendant graph is fully missing or no longer has available assets.

## Migration Plan

1. Add missing-age fields or equivalent tracking needed to determine cleanup eligibility.
2. Update missing marking code to set the missing timestamp only on transition to missing and clear it on recovery.
3. Add cleanup policy settings and defaults.
4. Implement hard-delete cleanup in `internal/library` with explicit dependency deletion order.
5. Wire cleanup into existing scheduled cleanup job handling and/or worker job kinds.
6. Add tests for destructive cleanup, recovery before retention, mixed available/missing parent handling, and deletion of user/governance data.
7. Deploy with cleanup disabled or conservative by default if migration risk is high; enable once behavior is verified.

## Open Questions

- Should the default missing retention be disabled, 30 days, or another value?
- Should cleanup operate globally, per library, or both through schedule scope?
