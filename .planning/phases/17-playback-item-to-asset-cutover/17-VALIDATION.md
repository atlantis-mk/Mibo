---
phase: 17
slug: playback-item-to-asset-cutover
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-25
---

# Phase 17 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go package tests |
| **Quick run command** | `cd mibo-media-server && go test ./internal/playback ./internal/httpapi -run 'Test(CatalogPlayback|PlaybackDecision|AssetLink|InventoryPlayback|HLS)' -count=1` |
| **Full suite command** | `cd mibo-media-server && go test ./internal/playback ./internal/httpapi -count=1` |
| **Estimated runtime** | ~55 seconds |

---

## Sampling Rate

- **After every task commit:** Run the task-specific command below or `cd mibo-media-server && go test ./internal/playback ./internal/httpapi -run 'Test(CatalogPlayback|PlaybackDecision|AssetLink|InventoryPlayback|HLS)' -count=1`
- **After every plan wave:** Run `cd mibo-media-server && go test ./internal/playback ./internal/httpapi -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 55 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 17-01-01 | 01 | 1 | PLAY-01, PLAY-02 | T-17-01 / T-17-02 | Asset ranking is deterministic and explicit asset selection cannot escape the requested item boundary | unit | `cd mibo-media-server && go test ./internal/playback -run 'TestCatalogPlayback' -count=1` | ✅ | ⬜ pending |
| 17-01-02 | 01 | 1 | PLAY-01, PLAY-02, PLAY-03 | T-17-03 / T-17-04 | Playback responses resolve through asset/item/file joins and return explainable unplayable payloads | unit | `cd mibo-media-server && go test ./internal/playback -run 'Test(CatalogPlayback|PlaybackDecision)' -count=1` | ✅ | ⬜ pending |
| 17-02-01 | 02 | 2 | PLAY-01, PLAY-02 | T-17-05 | Authenticated catalog playback routes expose item/asset identifiers and keep 200-level decision payloads | integration | `cd mibo-media-server && go test ./internal/httpapi -run 'TestCatalogPlayback' -count=1` | ✅ | ⬜ pending |
| 17-02-02 | 02 | 2 | PLAY-01, PLAY-02 | T-17-06 / T-17-07 | HTTP handlers stay thin, parse `asset_id`, and absolutize returned playback URLs | integration | `cd mibo-media-server && go test ./internal/httpapi -run 'Test(CatalogPlayback|AssetLink)' -count=1` | ✅ | ⬜ pending |
| 17-03-01 | 03 | 3 | PLAY-03 | T-17-08 / T-17-09 | HLS and direct stream routes are keyed by `inventory_files.id` and cannot path-traverse artifact storage | integration | `cd mibo-media-server && go test ./internal/httpapi -run 'TestInventoryPlayback(HLS|Stream)' -count=1` | ✅ | ⬜ pending |
| 17-03-02 | 03 | 3 | PLAY-03 | T-17-10 | Missing inventory files surface as structured unplayable decisions from item playback | integration | `cd mibo-media-server && go test ./internal/httpapi -run 'TestInventoryPlayback(MissingFile|HLS|Stream)' -count=1` | ✅ | ⬜ pending |

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
- [x] Feedback latency < 55s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
