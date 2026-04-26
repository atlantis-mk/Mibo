## 1. Catalog Contracts And Migration Guards

- [x] 1.1 Audit the current catalog-kernel implementation in `mibo-media-server/internal/catalog` and database models to identify which contract, index, and migration-state pieces from the change are already present versus still missing.
- [x] 1.2 Finalize canonical catalog DTO and type semantics for `series`, `season`, `episode`, `movie`, `extra`, `media_assets`, and linked projection refresh behavior.
- [x] 1.3 Add or complete migration-state persistence for backfill completion, catalog-read enablement, and legacy-cleanup completion.
- [x] 1.4 Add or complete the required uniqueness and lookup indexes for catalog hierarchy, asset links, inventory files, and catalog search documents.

## 2. Legacy Backfill Completion

- [x] 2.1 Extend legacy backfill to cover the remaining mappings for movies, series, seasons, episodes, inventory files, assets, asset links, images, external identities, metadata sources, and progress.
- [x] 2.2 Make repeated backfill runs fully idempotent and emit structured reporting for conflicts, orphan files, and duplicate-episode candidates.
- [x] 2.3 Refresh library projections after backfill mutations and add or update tests for empty-db, legacy-db, and repeated-backfill cases.

## 3. Scan Write Cutover

- [x] 3.1 Refactor scan reconciliation so new scan results upsert `inventory_files`, `media_assets`, `asset_files`, and catalog item hierarchy records instead of creating new legacy media rows.
- [x] 3.2 Add support for episodic hierarchy creation, multi-episode assets, and multi-version asset reuse during scan ingestion.
- [x] 3.3 Update scan delete and rename handling so availability changes are reflected in catalog state without destroying metadata-rich catalog items.
- [x] 3.4 Update scan and worker tests to assert catalog-table growth and legacy-table freeze after cutover.

## 4. Metadata Governance Rebuild

- [x] 4.1 Introduce catalog-item-based matching entrypoints that root TV matching at the series item and retain a migration wrapper only where needed.
- [x] 4.2 Persist provider evidence, canonical field states, and field locks so refetch updates unlocked data without overwriting locked fields.
- [x] 4.3 Generate or update season and episode catalog items from provider detail and merge local episode assets into the provider-derived hierarchy.
- [x] 4.4 Define and test governance-status transitions for matched, review-needed, unmatched, manual, and locked flows.

## 5. Catalog API, Search, And Progress Cutover

- [x] 5.1 Replace library list, item detail, and series-seasons responses with catalog-backed DTO handlers in `internal/httpapi`.
- [x] 5.2 Move governance endpoints and metadata search/apply/refetch flows onto catalog item identities and governance workspace responses.
- [x] 5.3 Switch search indexing and query behavior to `catalog_search_documents` and catalog result types.
- [x] 5.4 Switch progress reads and writes to catalog item and asset identities and update affected tests.
- [x] 5.5 Define explicit migration-period behavior for legacy media endpoints that remain temporarily reachable.

## 6. Playback Cutover

- [x] 6.1 Change playback request and selection logic to resolve from catalog item to asset, asset file, and inventory file.
- [x] 6.2 Implement deterministic asset ordering, explicit `asset_id` selection, and unavailable-item decisions without server errors.
- [x] 6.3 Update playback, streaming, HLS, and progress integration points to stop depending on legacy `media_files.id` as the primary runtime identity.
- [x] 6.4 Add playback tests for single-asset, multi-version, multi-episode, and missing-file cases.

## 7. Frontend Catalog Migration

- [x] 7.1 Add catalog item, detail, asset, season, and governance types to `web/src/lib/mibo-api.ts` and switch query helpers in `web/src/lib/mibo-query.ts` to the new contracts.
- [x] 7.2 Migrate home, library, search, and media detail flows to render catalog item data and catalog empty states instead of legacy media assumptions.
- [x] 7.3 Update series detail and playback entry UI to render catalog season and episode hierarchy plus asset-aware playback choices.
- [x] 7.4 Remove or isolate legacy presentation helpers that depend on fields such as `series_title`, `match_status`, `source_path`, or `files[0]`.

## 8. Governance UI Migration

- [x] 8.1 Rebuild the governance workspace client around catalog governance responses, including canonical fields, locks, source evidence, images, external identities, and linked assets.
- [x] 8.2 Add image-selection and field-lock editing flows that update only the intended catalog governance state.
- [x] 8.3 Add asset-link visibility and conflict-review affordances for partially matched or mismatched series and episode structures.
- [x] 8.4 Verify the migrated governance UI with `pnpm typecheck`, `pnpm build`, and focused manual walkthroughs.

## 9. Final Cutover And Cleanup

- [x] 9.1 Enable catalog reads by default only after backfill, scan, metadata, API, playback, and frontend paths are validated together.
- [x] 9.2 Add or complete rebuild and consistency-check commands for rollups, availability, and catalog search documents before cleanup.
- [x] 9.3 Remove or isolate remaining legacy read and write paths once validation gates pass, while preserving a bounded migration fallback if still required.
- [x] 9.4 Run backend and frontend verification suites covering migration safety, catalog APIs, playback, and frontend type/build health.
- [x] 9.5 Update repository-facing operational notes or documentation for the new catalog-kernel runtime and recovery workflow.
