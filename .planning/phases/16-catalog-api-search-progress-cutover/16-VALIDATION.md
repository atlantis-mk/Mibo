---
phase: 16
slug: catalog-api-search-progress-cutover
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-25
---

# Phase 16 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `cd mibo-media-server && go test ./internal/catalog ./internal/progress ./internal/httpapi -count=1` |
| **Full suite command** | `cd mibo-media-server && go test ./internal/catalog ./internal/progress ./internal/httpapi -count=1` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd mibo-media-server && go test ./internal/catalog ./internal/progress ./internal/httpapi -count=1`
- **After every plan wave:** Run `cd mibo-media-server && go test ./internal/catalog ./internal/progress ./internal/httpapi -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 16-01-01 | 01 | 1 | API-01 / API-02 | T-16-01 / T-16-02 | only top-level movie/series browse rows and catalog hierarchy detail are exposed | unit | `cd mibo-media-server && go test ./internal/catalog -run 'Test(ListItems|GetItemDetail|ListSeriesSeasons)' -count=1` | ✅ | ⬜ pending |
| 16-01-02 | 01 | 1 | API-01 / API-02 | T-16-01 / T-16-02 | catalog DTOs are hydrated from catalog tables without raw model leakage | unit | `cd mibo-media-server && go test ./internal/catalog -run 'Test(ListItems|GetItemDetail|ListSeriesSeasons)' -count=1` | ✅ | ⬜ pending |
| 16-02-01 | 02 | 1 | API-03 | T-16-03 / T-16-04 | governance workspace exposes bounded evidence, selected images, and current asset links | unit | `cd mibo-media-server && go test ./internal/catalog -run 'Test(GetGovernanceWorkspace|UpdateFieldLock|SelectImage|LinkAsset|UnlinkAsset)' -count=1` | ✅ | ⬜ pending |
| 16-02-02 | 02 | 1 | API-03 | T-16-03 / T-16-04 | field locks, image selection, and asset links mutate only the targeted rows | unit | `cd mibo-media-server && go test ./internal/catalog -run 'Test(GetGovernanceWorkspace|UpdateFieldLock|SelectImage|LinkAsset|UnlinkAsset)' -count=1` | ✅ | ⬜ pending |
| 16-03-01 | 03 | 1 | API-04 | T-16-05 / T-16-06 | progress writes validate item/asset ownership before touching `user_item_data` | unit | `cd mibo-media-server && go test ./internal/progress -run 'Test(UpdateCatalogProgress|GetCatalogProgressState)' -count=1` | ✅ | ⬜ pending |
| 16-03-02 | 03 | 1 | API-04 | T-16-05 / T-16-06 | completion updates stay idempotent and do not inflate play counts on repeats | unit | `cd mibo-media-server && go test ./internal/progress -run 'Test(UpdateCatalogProgress|GetCatalogProgressState)' -count=1` | ✅ | ⬜ pending |
| 16-04-01 | 04 | 2 | API-01 / API-02 / API-03 / API-04 | T-16-07 / T-16-08 / T-16-09 | new catalog routes are auth-gated for mutations and return frozen DTOs for reads | integration | `cd mibo-media-server && go test ./internal/httpapi -run 'TestCatalog(ItemRoutes|GovernanceRoutes|ProgressRoutes)' -count=1` | ✅ | ⬜ pending |
| 16-04-02 | 04 | 2 | API-01 / API-02 / API-03 / API-04 | T-16-07 / T-16-08 / T-16-09 | handlers delegate to catalog/progress/metadata services without legacy route regression | integration | `cd mibo-media-server && go test ./internal/httpapi -run 'TestCatalog(ItemRoutes|GovernanceRoutes|ProgressRoutes)' -count=1` | ✅ | ⬜ pending |

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
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
