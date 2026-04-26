---
phase: 20
slug: legacy-model-retirement-production-hardening
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-25
---

# Phase 20 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + pnpm typecheck/build |
| **Config file** | none |
| **Quick run command** | `cd mibo-media-server && go test ./internal/catalog ./internal/database ./internal/httpapi ./internal/library ./internal/playback ./internal/progress ./internal/search -count=1` |
| **Full suite command** | `cd mibo-media-server && go test ./... -count=1 && cd ../web && pnpm typecheck && pnpm build` |
| **Estimated runtime** | ~90 seconds |

---

## Sampling Rate

- **After every task commit:** Run the task's package-scoped command from the table below
- **After every plan wave:** Run `cd mibo-media-server && go test ./... -count=1`
- **Before `/gsd-verify-work`:** Run `cd mibo-media-server && go test ./... -count=1 && cd ../web && pnpm typecheck && pnpm build`
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 20-01-01 | 01 | 1 | PROD-02 | T-20-01 / T-20-02 | audit coverage proves duplicate, availability, and projection issues are detectable without mutating data | integration | `cd mibo-media-server && go test ./internal/catalog -run 'TestCatalogConsistencyAudit' -count=1` | ✅ | ⬜ pending |
| 20-01-02 | 01 | 1 | PROD-02 | T-20-01 / T-20-02 | consistency audit emits deterministic issue codes and scope-bounded reports | integration | `cd mibo-media-server && go test ./internal/catalog -run 'TestCatalogConsistencyAudit' -count=1` | ✅ | ⬜ pending |
| 20-02-01 | 02 | 2 | PROD-02 | T-20-03 / T-20-04 | authenticated route and worker tests prove only typed audit/repair payloads are accepted | integration | `cd mibo-media-server && go test ./internal/httpapi ./internal/worker -run 'Test(CatalogConsistency|RunOnceProcessesCatalogConsistency)' -count=1` | ✅ | ⬜ pending |
| 20-02-02 | 02 | 2 | PROD-02, MIGR-04 | T-20-03 / T-20-04 | operators can queue audit/repair work and still inspect read-only migration state | integration | `cd mibo-media-server && go test ./internal/httpapi ./internal/worker -run 'Test(CatalogConsistency|RunOnceProcessesCatalogConsistency)' -count=1` | ✅ | ⬜ pending |
| 20-03-01 | 03 | 2 | PROD-04 | T-20-05 | startup tests prove hardening fails safely on duplicate rows and succeeds on fresh/legacy/repeated opens | integration | `cd mibo-media-server && go test ./internal/database -run 'TestDatabaseOpenCatalogKernel' -count=1` | ✅ | ⬜ pending |
| 20-03-02 | 03 | 2 | PROD-04 | T-20-05 | explicit index/constraint backstops are applied only after duplicate preflight checks pass | integration | `cd mibo-media-server && go test ./internal/database -run 'TestDatabaseOpenCatalogKernel' -count=1` | ✅ | ⬜ pending |
| 20-04-01 | 04 | 3 | PROD-03 | T-20-06 / T-20-07 | browse/search/progress tests prove catalog tables are sufficient even when legacy runtime tables are absent | integration | `cd mibo-media-server && go test ./internal/library ./internal/progress ./internal/search -run 'Test(CatalogBrowse|CatalogProgress|UserItemData)' -count=1` | ✅ | ⬜ pending |
| 20-04-02 | 04 | 3 | PROD-03 | T-20-06 / T-20-07 | normal runtime reads/writes use `catalog_search_documents` and `user_item_data` only | integration | `cd mibo-media-server && go test ./internal/library ./internal/progress ./internal/search -run 'Test(CatalogBrowse|CatalogProgress|UserItemData)' -count=1` | ✅ | ⬜ pending |
| 20-05-01 | 05 | 3 | PROD-03, MIGR-04 | T-20-08 / T-20-09 | router regression proves catalog/inventory routes remain while legacy playback/file routes disappear from normal runtime | integration | `cd mibo-media-server && go test ./internal/httpapi ./internal/playback -run 'Test(CatalogRuntime|CatalogPlayback)' -count=1` | ✅ | ⬜ pending |
| 20-05-02 | 05 | 3 | PROD-03, MIGR-04 | T-20-08 / T-20-09 | playback code has no legacy `MediaItem` / `MediaFile` main-path lookup left | integration | `cd mibo-media-server && go test ./internal/httpapi ./internal/playback -run 'Test(CatalogRuntime|CatalogPlayback)' -count=1` | ✅ | ⬜ pending |
| 20-06-01 | 06 | 4 | PROD-03 | T-20-10 | frontend compiles against catalog item/asset/inventory contracts only | build | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 20-06-02 | 06 | 4 | PROD-04 | T-20-10 | scripted validation and runbook cover startup, backend tests, typecheck, and build before cleanup | integration | `bash mibo-media-server/scripts/catalog-kernel-production-check.sh` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Read-only migration observability is understandable to operators | MIGR-04 | operator UX and wording cannot be fully judged from CLI output alone | Review the new catalog migration and consistency pages/routes in the web app or API client, confirm labels distinguish read-only compatibility data from normal runtime behavior |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
