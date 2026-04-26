---
status: verifying
trigger: "Investigate and fix the Phase 14 regressions in /Users/atlan/Desktop/IdeaProjects/Mibo caused by the scanner/catalog cutover.\n\nCurrent failing regression evidence from `go test ./...` in `mibo-media-server`:\n- internal/httpapi: TestLibraryItemEndpoints, TestAuthAndProgressEndpoints, TestRecentlyAddedEndpoint, TestLocalPlaybackStreamEndpoint, TestOpenListPlaybackStreamEndpoint, TestLocalPlaybackReturnsHLSPlaylist, TestOpenListPlaybackReturnsHLSPlaylist\n- internal/worker: TestRunOnceProcessesSyncLibraryJob, TestRunOnceGeneratesFallbackArtworkWhenMetadataMissing, TestRunOnceProcessesTargetedRefreshJob, TestPartialSyncDoesNotSoftDeleteUnseenLibraryRows\n- repeated probe failures: `UNIQUE constraint failed: media_streams.stream_index` from `internal/probe/service.go`\n\nAdvisory review findings also flagged:\n1. Stable-identity rename can duplicate catalog items and keep stale links alive\n2. Rescans append duplicate scanner metadata_sources rows\n3. Series/season availability_status can stay stale after deletes\n\nYour job:\n- Diagnose the real root causes, not just patch tests.\n- Apply the smallest correct code changes in the codebase.\n- Preserve the Phase 14 direction: fresh scans write catalog/inventory assets, but do not break prior behavior/tests unless the only correct fix is to update obsolete tests with strong justification.\n- Prefer fixing source code over weakening assertions.\n- Run targeted Go tests as you iterate, then run `go test ./...` in `mibo-media-server` before returning if feasible.\n- Commit any code changes you make with clear commit messages.\n- If you determine a previously failing test is genuinely obsolete and should change, explain why in your final message.\n\nUseful files likely involved:\n- mibo-media-server/internal/library/scan_run.go\n- mibo-media-server/internal/library/scan_catalog.go\n- mibo-media-server/internal/probe/service.go\n- mibo-media-server/internal/httpapi/router_test.go\n- mibo-media-server/internal/worker/worker_test.go\n- mibo-media-server/internal/worker/worker_catalog_scan_test.go\n- mibo-media-server/internal/library/scan_catalog_test.go\n- .planning/phases/14-scanner-writes-catalog-assets/14-REVIEW.md\n\nReturn a concise summary of:\n- root causes fixed\n- files changed\n- commits created\n- final test status"
created: 2026-04-25T00:00:00Z
updated: 2026-04-25T00:42:00Z
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "Phase 14 regressions come from four specific mechanisms: media stream uniqueness is global instead of per file; movie rename rescans reuse inventory files/assets but still create path-keyed catalog items; scanner metadata evidence is inserted on every scan instead of updated in place; and cleanup recomputes availability only for leaf items with direct asset links, not their ancestor series/season rows."
  confirming_evidence:
    - "`TestProbeInventoryFileAllowsSameStreamIndexesAcrossDifferentFiles` fails on the second probe with `UNIQUE constraint failed: media_streams.stream_index`, and `database.MediaStream` only marks `stream_index` as part of the unique index while `file_id` is not in that same unique key."
    - "`TestRunSyncLibraryReusesStableIdentityCatalogRowsOnRename` fails with two `catalog_items` rows after a rename, and `writeCatalogScanMovie` creates/reuses the catalog item by path before stable-identity file/asset reuse can point back to the existing linked item."
    - "`TestRunSyncLibraryDeduplicatesScannerMetadataSourcesOnRescan` fails with two `metadata_sources` rows after an unchanged rescan, and scanner writes call `RecordMetadataSource`, which always inserts a new row."
    - "`TestRunSyncLibraryMarksAncestorAvailabilityMissingWhenEpisodesDeleted` leaves the series row `available`, and `cleanupMissingCatalog` only iterates `scopedCatalogItemIDs`, which are limited to direct asset-linked leaf items."
  falsification_test: "If file-scoped stream uniqueness, linked-item reuse for renamed movies, scanner-source upsert behavior, and ancestor availability recomputation are implemented, the new targeted tests and the previously failing worker/httpapi scans should still fail in the same ways."
  fix_rationale: "Each proposed fix changes the exact persistence boundary causing the regression: scope the stream uniqueness index by file, reuse/update the existing linked movie item instead of creating a second path-keyed row, upsert only the scanner-owned metadata source row for an item, and recompute availability for affected items plus ancestors after cleanup."
  blind_spots: "Legacy httpapi/worker tests may still be obsolete relative to the catalog cutover even after source fixes; full-suite rerun is needed to separate real source regressions from stale expectations."

hypothesis: implemented fixes address the confirmed persistence-layer regressions; remaining failures, if any, should now distinguish real breakage from stale legacy expectations
test: rerun focused regression tests, representative worker/httpapi failures, then full `go test ./...` in mibo-media-server
expecting: focused tests and scan-abort-driven worker/httpapi failures should pass; any residual failures will likely be obsolete legacy-media assertions
next_action: run representative worker/httpapi tests followed by full mibo-media-server Go test suite

## Symptoms

expected: scanner/catalog cutover should preserve prior library, playback, progress, and worker behavior while writing catalog/inventory assets without duplicate rows or stale availability/link state
actual: multiple httpapi and worker regressions fail after Phase 14 cutover; probe writes hit UNIQUE constraint failed: media_streams.stream_index; advisory review found rename duplication, metadata_source duplication, and stale series/season availability after deletes
errors: internal/httpapi tests fail; internal/worker tests fail; repeated `UNIQUE constraint failed: media_streams.stream_index` from internal/probe/service.go
reproduction: from mibo-media-server run `go test ./...`; targeted failures listed in trigger
started: after Phase 14 scanner/catalog cutover

## Eliminated

## Evidence

- timestamp: 2026-04-25T00:05:00Z
  checked: .planning/debug/knowledge-base.md
  found: file does not exist
  implication: no prior local debug pattern to test first; proceed with open investigation

- timestamp: 2026-04-25T00:10:00Z
  checked: debugger reference docs and .planning/phases/14-scanner-writes-catalog-assets/14-REVIEW.md
  found: review identified three concrete cutover risks — rename duplicates catalog items/stale links, rescans duplicate scanner metadata_sources, and parent availability_status can remain stale after deletes
  implication: these become primary hypothesis candidates for catalog-side regressions; probe UNIQUE failure remains an additional likely root cause for several test failures

- timestamp: 2026-04-25T00:15:00Z
  checked: `go test ./internal/httpapi -run 'TestLibraryItemEndpoints|TestRecentlyAddedEndpoint'` and `go test ./internal/worker -run 'TestRunOnceProcessesSyncLibraryJob|TestRunOnceProcessesTargetedRefreshJob'`
  found: worker scan jobs fail during `probe_inventory_file` with `UNIQUE constraint failed: media_streams.stream_index`; downstream httpapi tests then return zero media items/recent items because scan job did not finish populating catalog projections
  implication: at least one root cause is in probe stream persistence, and fixing it is prerequisite to validating remaining catalog regressions

- timestamp: 2026-04-25T00:22:00Z
  checked: internal/probe/service.go, internal/database/inventory_models.go, internal/library/scan_catalog.go, internal/catalog/service.go
  found: `ProbeInventoryFile` deletes and reinserts all streams per file, but `database.MediaStream` marks only `stream_index` as `uniqueIndex:idx_media_stream_file_index` while `file_id` is not part of that unique index; movie scan always resolves catalog items by path before/after stable-identity file reuse; scanner scan calls `RecordMetadataSource` on every run and that API always inserts; cleanup recomputes availability only for `scopedCatalogItemIDs`, which returns only items reachable through direct asset links
  implication: code directly matches the observed probe failure and the three advisory catalog regressions

- timestamp: 2026-04-25T00:28:00Z
  checked: new targeted regression tests in internal/library/scan_catalog_test.go and internal/probe/service_inventory_test.go
  found: added coverage for stable-identity movie rename item reuse, scanner metadata source dedupe on rescan, ancestor availability after episode deletion, and probing two files with identical stream indices
  implication: these tests will provide direct falsification/confirmation for each suspected root cause before implementing fixes

- timestamp: 2026-04-25T00:36:00Z
  checked: targeted regression test execution
  found: `TestRunSyncLibraryReusesStableIdentityCatalogRowsOnRename` produced 2 catalog items, `TestRunSyncLibraryDeduplicatesScannerMetadataSourcesOnRescan` produced 2 metadata_sources rows, `TestRunSyncLibraryMarksAncestorAvailabilityMissingWhenEpisodesDeleted` left the series row available, and `TestProbeInventoryFileAllowsSameStreamIndexesAcrossDifferentFiles` failed on the second file with the same stream uniqueness error
  implication: the four suspected mechanisms are now directly confirmed and ready for minimal code fixes

- timestamp: 2026-04-25T00:42:00Z
  checked: fixes in internal/database/inventory_models.go and internal/library/scan_catalog.go plus focused regression reruns
  found: new probe multi-file test and the three new scan catalog regression tests now pass after scoping the media stream unique index by file, reusing linked movie items on stable-identity rename, upserting scanner metadata sources, and recomputing ancestor availability
  implication: the confirmed source regressions are fixed; next verification should measure impact on previously failing worker/httpapi suites and identify any stale tests

## Resolution

root_cause: Phase 14 introduced four persistence regressions: media_stream uniqueness was global by stream_index instead of per file, stable-identity movie renames reused files/assets but not the linked catalog item, scanner metadata evidence rows were append-only on rescans, and availability cleanup skipped ancestor series/season rows that lack direct asset links.
fix: Scoped the media stream unique index to `(file_id, stream_index)`, reused and refreshed the already linked movie catalog item when a stable-identity file/asset is reused, upserted the scanner-owned metadata source row per item instead of always inserting, and recalculated availability for affected items plus ancestors during cleanup.
verification: Focused new regression tests pass; representative worker/httpapi and full-suite verification still pending.
files_changed: ["mibo-media-server/internal/database/inventory_models.go", "mibo-media-server/internal/library/scan_catalog.go", "mibo-media-server/internal/library/scan_catalog_test.go", "mibo-media-server/internal/probe/service_inventory_test.go"]
