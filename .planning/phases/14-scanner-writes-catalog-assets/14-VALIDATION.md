---
phase: 14
slug: scanner-writes-catalog-assets
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-25
---

# Phase 14 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go package tests |
| **Quick run command** | `cd mibo-media-server && go test ./internal/library -run 'Test(ScanCatalogWriter|ClassifyMediaFile|RunSyncLibrary)' -count=1` |
| **Full suite command** | `cd mibo-media-server && go test ./internal/library ./internal/inventory ./internal/probe ./internal/worker -count=1` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd mibo-media-server && go test ./internal/library -run 'Test(ScanCatalogWriter|ClassifyMediaFile|RunSyncLibrary)' -count=1` or the task-specific command below
- **After every plan wave:** Run `cd mibo-media-server && go test ./internal/library ./internal/inventory ./internal/probe ./internal/worker -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 14-01-01 | 01 | 1 | SCAN-01, SCAN-02 | T-14-01 / T-14-02 | Scanner local evidence stays allowlisted and hierarchy rows use canonical paths | unit | `cd mibo-media-server && go test ./internal/library -run 'TestScanCatalogWriter' -count=1` | ✅ | ⬜ pending |
| 14-01-02 | 01 | 1 | SCAN-01, SCAN-02 | T-14-03 | Shared library service wiring reuses catalog/inventory helpers instead of ad hoc writes | unit | `cd mibo-media-server && go test ./internal/library -run 'TestScanCatalogWriter' -count=1` | ✅ | ⬜ pending |
| 14-02-01 | 02 | 2 | SCAN-01, SCAN-02, SCAN-03 | T-14-04 | Multi-episode and version semantics are explicit before scan-loop cutover | unit | `cd mibo-media-server && go test ./internal/library -run 'Test(ClassifyMediaFileParsesMultiEpisodeRange|RunSyncLibraryWritesCatalogRowsWithoutLegacyMediaTables|RunSyncLibraryCreatesVersionAssetForDuplicateEpisodeSlot)' -count=1` | ✅ | ⬜ pending |
| 14-02-02 | 02 | 2 | SCAN-01, SCAN-02, SCAN-03 | T-14-05 / T-14-06 | Scans create only new-kernel rows and reuse canonical item hierarchy | integration | `cd mibo-media-server && go test ./internal/library -run 'Test(ClassifyMediaFileParsesMultiEpisodeRange|RunSyncLibraryWritesCatalogRowsWithoutLegacyMediaTables|RunSyncLibraryCreatesVersionAssetForDuplicateEpisodeSlot)' -count=1` | ✅ | ⬜ pending |
| 14-03-01 | 03 | 3 | SCAN-01, SCAN-02 | T-14-07 | Probe jobs carry only typed inventory-file payloads | integration | `cd mibo-media-server && go test ./internal/probe ./internal/worker -run 'Test(ProbeInventoryFile|RunOnceProcessesProbeInventoryFileJob)' -count=1` | ✅ | ⬜ pending |
| 14-03-02 | 03 | 3 | SCAN-01, SCAN-02 | T-14-08 / T-14-09 | ffprobe data is normalized into media streams and asset summaries without legacy writes | integration | `cd mibo-media-server && go test ./internal/probe ./internal/worker -run 'Test(ProbeInventoryFile|RunOnceProcessesProbeInventoryFileJob)' -count=1` | ✅ | ⬜ pending |
| 14-04-01 | 04 | 4 | SCAN-03 | T-14-10 | Missing-file cleanup preserves governed metadata and only changes status/availability | integration | `cd mibo-media-server && go test ./internal/library -run 'TestRunSyncLibrary(MarksMissingInventoryWithoutDeletingCatalogItem|KeepsEpisodeAvailableWhenAnotherVersionRemains|ReusesStableIdentityCatalogRowsOnRename)' -count=1` | ✅ | ⬜ pending |
| 14-04-02 | 04 | 4 | SCAN-03 | T-14-11 / T-14-12 | Stable identity reuse and version-aware availability prevent duplicate or wrongly unavailable items | integration | `cd mibo-media-server && go test ./internal/library -run 'TestRunSyncLibrary(MarksMissingInventoryWithoutDeletingCatalogItem|KeepsEpisodeAvailableWhenAnotherVersionRemains|ReusesStableIdentityCatalogRowsOnRename)' -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 45s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
