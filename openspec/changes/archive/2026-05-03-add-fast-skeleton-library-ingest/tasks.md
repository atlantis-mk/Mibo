## 1. Model And Contracts

- [x] 1.1 Define the backend representation for fast-ingest maturity state using `inventory_file` as the stable anchor or document why an additional candidate table is required.
- [x] 1.2 Add API response fields needed to distinguish catalog-backed entries from organizing entries without breaking existing catalog browse clients.
- [x] 1.3 Add tests for maturity state serialization and duplicate suppression between inventory-backed and catalog-backed results.

## 2. Fast Scan Path

- [x] 2.1 Split the scan flow so supported video discovery can persist inventory facts and publish skeleton visibility before full catalog materialization.
- [x] 2.2 Ensure scan policy, hidden-file handling, extension filters, size filters, and configurable exclusion rules still apply before skeleton publication.
- [x] 2.3 Move or defer sidecar metadata parsing, artwork selection, detailed classification validation, and expensive projection refresh work out of the minimal ingest critical path where feasible.
- [x] 2.4 Preserve existing behavior for confident scans that can safely create final catalog-backed rows immediately without delaying skeleton ingest.
- [x] 2.5 Add backend tests proving a newly discovered video becomes browseable before probe and metadata jobs complete.

## 3. Classification And Upgrade

- [x] 3.1 Add or adapt asynchronous classification/catalog materialization work that upgrades inventory-backed organizing entries into final catalog graph rows.
- [x] 3.2 Ensure movie, series, season, episode, version, attachment, and review-required outcomes preserve links back to the original `inventory_file` anchor.
- [x] 3.3 Ensure classification failures keep the discovered entry visible with review-required or failure maturity instead of deleting inventory facts.
- [x] 3.4 Add tests for upgrading one discovered file into an episode hierarchy and multiple discovered files into one movie with multiple assets.

## 4. Browse Query Integration

- [x] 4.1 Extend library discovery queries to merge in inventory-backed organizing entries that do not yet have catalog-backed browse results.
- [x] 4.2 Implement duplicate suppression once an inventory file contributes to a catalog-backed item in the same browse scope.
- [x] 4.3 Apply total-count, paging, filters, and title sorting consistently across mixed catalog-backed and organizing entries.
- [x] 4.4 Add API tests covering mixed mature and organizing entries, paging boundaries, sort direction, and filter behavior.

## 5. Frontend Experience

- [x] 5.1 Update library detail types and API mapping to understand organizing/discovered media entries and maturity state.
- [x] 5.2 Render organizing entries as media-grid cards with filename-derived title, placeholder artwork, and clear organizing/review badges.
- [x] 5.3 Hide or reroute final-catalog-only actions on organizing cards until a final catalog identity exists.
- [x] 5.4 Add frontend tests or focused UI state tests for organizing cards upgrading to catalog-backed cards without duplicates.

## 6. Verification

- [x] 6.1 Run focused backend tests for library scanning, catalog browse/projections, worker enrichment, and missing cleanup.
- [x] 6.2 Run frontend typecheck and relevant library browsing tests.
- [x] 6.3 Manually validate a local media scan with `MIBO_STORAGE_PROVIDER=local` and confirm newly discovered files appear before metadata/probe enrichment finishes.
