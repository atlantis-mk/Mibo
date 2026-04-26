---
phase: 15
slug: series-level-metadata-governance-engine
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-25
---

# Phase 15 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing Go package tests are self-contained |
| **Quick run command** | `cd mibo-media-server && go test ./internal/catalog ./internal/metadata -count=1` |
| **Full suite command** | `cd mibo-media-server && go test ./...` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run the task’s targeted package test command.
- **After every plan wave:** Run `cd mibo-media-server && go test ./internal/catalog ./internal/metadata -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 15-01-01 | 01 | 1 | META-04 | T-15-01 / T-15-02 | provider refresh replaces only provider-scoped normalized rows | unit | `cd mibo-media-server && go test ./internal/catalog -run 'TestReplaceItem(Images|People|Tags)' -count=1` | ✅ | ⬜ pending |
| 15-01-02 | 01 | 1 | META-04 | T-15-01 / T-15-02 | selected images and normalized people/tags remain deterministic across reruns | unit | `cd mibo-media-server && go test ./internal/catalog -run 'TestReplaceItem(Images|People|Tags)' -count=1` | ✅ | ⬜ pending |
| 15-02-01 | 02 | 1 | META-02 | T-15-03 / T-15-04 | governed hierarchy helpers reuse existing slots before creating provider-only rows | unit | `cd mibo-media-server && go test ./internal/catalog -run 'Test(ResolveSeriesRoot|UpsertGoverned|ReconcileEpisodeAvailability)' -count=1` | ✅ | ⬜ pending |
| 15-02-02 | 02 | 1 | META-02 | T-15-03 / T-15-04 | availability derives from asset/file links plus air date rather than legacy row state | unit | `cd mibo-media-server && go test ./internal/catalog -run 'Test(ResolveSeriesRoot|UpsertGoverned|ReconcileEpisodeAvailability)' -count=1` | ✅ | ⬜ pending |
| 15-03-01 | 03 | 2 | META-01 / META-03 | T-15-05 / T-15-06 | series match stores source evidence and preserves locked canonical fields | integration | `cd mibo-media-server && go test ./internal/metadata -run 'Test(MatchCatalogSeries|RefreshCatalogSeriesMetadata)' -count=1` | ✅ | ⬜ pending |
| 15-03-02 | 03 | 2 | META-01 / META-02 / META-03 / META-04 | T-15-05 / T-15-06 / T-15-07 | full series refresh is idempotent, normalizes provider data, and refreshes projections | integration | `cd mibo-media-server && go test ./internal/metadata -run 'Test(MatchCatalogSeries|RefreshCatalogSeriesMetadata)' -count=1 && cd mibo-media-server && go test ./internal/catalog -count=1` | ✅ | ⬜ pending |

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
