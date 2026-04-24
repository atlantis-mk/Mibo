---
phase: 11
slug: event-driven-refresh-hardening
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-24
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `cd /root/Mibo/mibo-media-server && go test ./internal/listener ./internal/httpapi ./internal/worker -run 'Test.*(StorageEvent|Listener|TargetedRefresh|Reconcile)'` |
| **Full suite command** | `cd /root/Mibo/mibo-media-server && go test ./...` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd /root/Mibo/mibo-media-server && go test ./internal/listener ./internal/httpapi ./internal/worker -run 'Test.*(StorageEvent|Listener|TargetedRefresh|Reconcile)'`
- **After every plan wave:** Run `cd /root/Mibo/mibo-media-server && go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | LIST-03 | T-11-01-01 | listener merge window never crosses library boundaries | unit | `cd /root/Mibo/mibo-media-server && go test ./internal/listener -run 'Test.*(Merge|Window|Ancestor)'` | ✅ | ⬜ pending |
| 11-01-02 | 01 | 1 | LIST-04 | T-11-01-02 | reconciliation jobs are future-dated, unique per library, and reseedable | unit | `cd /root/Mibo/mibo-media-server && go test ./internal/listener -run 'Test.*Reconcile'` | ✅ | ⬜ pending |
| 11-02-01 | 02 | 2 | LIST-01 | T-11-02-01 | route rejects escaping paths and unsupported payloads before enqueue | integration | `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi -run 'TestStorageEvent'` | ✅ | ⬜ pending |
| 11-02-02 | 02 | 2 | LIST-02 | T-11-02-02 | route returns accepted listener jobs, never direct media mutation | integration | `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi -run 'TestStorageEvent'` | ✅ | ⬜ pending |
| 11-03-01 | 03 | 3 | LIST-02 | T-11-03-01 | worker converts coalesced listener jobs into existing scan queue work only | integration | `cd /root/Mibo/mibo-media-server && go test ./internal/worker -run 'Test.*(TargetedRefresh|StorageEvent)'` | ✅ | ⬜ pending |
| 11-03-02 | 03 | 3 | LIST-04 | T-11-03-02 | worker seeds and reseeds periodic reconcile jobs for active libraries | integration | `cd /root/Mibo/mibo-media-server && go test ./internal/worker -run 'Test.*Reconcile'` | ✅ | ⬜ pending |

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
