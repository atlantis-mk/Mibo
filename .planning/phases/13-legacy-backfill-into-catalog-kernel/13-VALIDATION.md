---
phase: 13
slug: legacy-backfill-into-catalog-kernel
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-25
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfill(Report|Movies|Series|Progress)' -count=1` |
| **Full suite command** | `cd mibo-media-server && go test ./internal/catalog ./internal/database ./internal/httpapi ./internal/worker -count=1` |
| **Estimated runtime** | ~25 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfill(Report|Movies|Series|Progress)' -count=1`
- **After every plan wave:** Run `cd mibo-media-server && go test ./internal/catalog ./internal/database ./internal/httpapi ./internal/worker -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 25 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 13-01-01 | 01 | 1 | MIGR-02 | T-13-01 / T-13-02 | Only durable report categories and statuses are persisted | unit | `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillRun' -count=1` | ✅ | ⬜ pending |
| 13-01-02 | 01 | 1 | MIGR-02, MIGR-03 | T-13-02 | Run counts derive from persisted entries, not client input | unit | `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillReportQueries' -count=1` | ✅ | ⬜ pending |
| 13-02-01 | 02 | 2 | MIGR-01 | T-13-03 | Worker only runs typed queued payloads | integration | `cd mibo-media-server && go test ./internal/worker -run 'TestRunOnce.*CatalogBackfill' -count=1` | ✅ | ⬜ pending |
| 13-02-02 | 02 | 2 | MIGR-02 | T-13-04 | Backfill routes require auth and expose typed report JSON only | integration | `cd mibo-media-server && go test ./internal/httpapi -run 'TestCatalogMigrationBackfill' -count=1` | ✅ | ⬜ pending |
| 13-03-01 | 03 | 2 | MIGR-01 | T-13-05 | Movie rows reuse prior catalog/file identities instead of duplicating | unit | `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillMovies' -count=1` | ✅ | ⬜ pending |
| 13-03-02 | 03 | 2 | MIGR-03 | T-13-05 | Repeated movie runs keep catalog/assets/files unique | unit | `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillMoviesIdempotent' -count=1` | ✅ | ⬜ pending |
| 13-04-01 | 04 | 2 | MIGR-01, MIGR-02 | T-13-06 | Series grouping prefers provider identity and reports ambiguous slots | unit | `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillSeries' -count=1` | ✅ | ⬜ pending |
| 13-04-02 | 04 | 2 | MIGR-02, MIGR-03 | T-13-06 | Duplicate episode candidates and orphan files are recorded without duplicate catalog rows | unit | `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillSeriesConflicts' -count=1` | ✅ | ⬜ pending |
| 13-05-01 | 05 | 3 | MIGR-01, MIGR-03 | T-13-07 | Progress rows map to catalog item/asset keys idempotently | integration | `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillProgress' -count=1` | ✅ | ⬜ pending |
| 13-05-02 | 05 | 3 | MIGR-01, MIGR-02, MIGR-03 | T-13-07 / T-13-08 | Successful runs refresh projections and update only the backfill-completed timestamp | integration | `cd mibo-media-server && go test ./internal/catalog ./internal/worker ./internal/httpapi -run 'TestLegacyBackfill|TestRunOnce.*CatalogBackfill|TestCatalogMigrationBackfill' -count=1` | ✅ | ⬜ pending |

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
- [x] Feedback latency < 25s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
