---
phase: 02
slug: library-async-sync-foundation
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-22
---

# Phase 02 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` + `pnpm build` |
| **Config file** | `mibo-media-server/go.mod`, `web/package.json` |
| **Quick run command** | `cd mibo-media-server && go test ./internal/httpapi ./internal/worker -count=1 && cd ../web && pnpm build` |
| **Full suite command** | `cd mibo-media-server && go test ./... -count=1 && cd ../web && pnpm build` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd mibo-media-server && go test ./internal/httpapi ./internal/worker -count=1 && cd ../web && pnpm build`
- **After every plan wave:** Run `cd mibo-media-server && go test ./... -count=1 && cd ../web && pnpm build`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | LIBR-04 | T-02-01 | Only authenticated settings updates can change refresh cadence; invalid intervals are rejected | integration | `cd mibo-media-server && go test ./internal/httpapi -count=1` | ✅ | ⬜ pending |
| 02-01-02 | 01 | 1 | LIBR-03, CATA-06 | T-02-02 | Scheduled scans deduplicate through `EnqueueUnique`; jobs list filters only supported fields | unit+integration | `cd mibo-media-server && go test ./internal/worker ./internal/jobs -count=1` | ✅ | ⬜ pending |
| 02-02-01 | 02 | 2 | LIBR-01, LIBR-02 | T-02-03 | Source/library forms only submit expected fields; no unchecked HTML injection | build | `cd web && pnpm build` | ✅ | ⬜ pending |
| 02-02-02 | 02 | 2 | LIBR-03, LIBR-04, CATA-06 | T-02-04 | Job errors render as text and authenticated retry/settings actions stay behind typed API client calls | build | `cd web && pnpm build` | ✅ | ⬜ pending |
| 02-02-03 | 02 | 2 | LIBR-03, LIBR-04 | T-02-04 / — | Human confirms badge, jobs tab, and refresh controls reflect live backend state | manual+build | `cd web && pnpm build` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Settings shell shows library status badge, jobs tab, and refresh interval editor coherently | LIBR-03, LIBR-04 | Visual/admin interaction flow spans multiple controls | Sign in as `admin`, open Settings, create or open a library, trigger scan, confirm status badge changes and Jobs tab shows the queued/running job; edit refresh interval and confirm success toast/state refresh |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 45s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
