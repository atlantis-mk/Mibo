---
phase: 10
slug: scheduled-operations-control
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-24
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for scheduled operations control.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` + frontend `pnpm` type/build checks |
| **Config file** | `mibo-media-server/go.mod`, `web/package.json`, `web/tsconfig.json`, `web/vite.config.ts` |
| **Quick run command** | `cd /root/Mibo/mibo-media-server && go test ./internal/schedule ./internal/library ./internal/metadata ./internal/httpapi ./internal/worker -run 'Test.*Schedule' && cd /root/Mibo/web && pnpm typecheck` |
| **Full suite command** | `cd /root/Mibo/mibo-media-server && go test ./... && cd /root/Mibo/web && pnpm build` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** run the relevant focused package tests from the plan's `<verify>` block.
- **After every execution wave:** run `cd /root/Mibo/mibo-media-server && go test ./...`.
- **Before `/gsd-verify-work`:** full backend suite plus frontend `pnpm build` must be green.
- **Max feedback latency:** 60 seconds.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | SJOB-01..SJOB-08 | T-10-01-01 | invalid frequency/scope combos are rejected before persistence | unit | `cd /root/Mibo/mibo-media-server && go test ./internal/schedule -run 'Test.*(Frequency|Validate)'` | ⬜ | ⬜ pending |
| 10-01-02 | 01 | 1 | SJOB-07, SJOB-08 | T-10-01-02 | schedule rows always project next run + latest result fields | unit | `cd /root/Mibo/mibo-media-server && go test ./internal/schedule -run 'Test.*(Create|Update|History)'` | ⬜ | ⬜ pending |
| 10-02-01 | 02 | 2 | SJOB-01, SJOB-04 | T-10-02-01 | scan/cleanup maintenance respects global/library scope and never escapes library roots | unit/integration | `cd /root/Mibo/mibo-media-server && go test ./internal/library -run 'Test.*(ScheduledScan|Cleanup)'` | ⬜ | ⬜ pending |
| 10-02-02 | 02 | 2 | SJOB-05 | T-10-02-02 | invalid-link checks report failures through schedule/job history without direct client execution | unit/integration | `cd /root/Mibo/mibo-media-server && go test ./internal/library -run 'Test.*InvalidLink'` | ⬜ | ⬜ pending |
| 10-03-01 | 03 | 2 | SJOB-02, SJOB-03, SJOB-06 | T-10-03-01 | metadata/trailer/artwork maintenance stays inside metadata ownership and does not mutate arbitrary fields | unit | `cd /root/Mibo/mibo-media-server && go test ./internal/metadata -run 'Test.*(ScheduledMetadata|TrailerSync|ArtworkRefresh)'` | ⬜ | ⬜ pending |
| 10-04-01 | 04 | 3 | SJOB-01..SJOB-08 | T-10-04-01 | schedule API requires auth, validates payloads, and returns schedule-centric history | integration | `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi -run 'Test.*Schedule'` | ⬜ | ⬜ pending |
| 10-04-02 | 04 | 3 | SJOB-07, SJOB-08 | T-10-04-02 | run-now and enable/disable actions enqueue validated work and update next-run snapshots | unit/integration | `cd /root/Mibo/mibo-media-server && go test ./internal/schedule ./internal/httpapi -run 'Test.*(RunNow|Toggle|History)'` | ⬜ | ⬜ pending |
| 10-05-01 | 05 | 4 | SJOB-07, SJOB-08 | T-10-05-01 | due schedules execute through the same worker lifecycle and propagate queued/running/completed/failed states | integration | `cd /root/Mibo/mibo-media-server && go test ./internal/worker -run 'Test.*Schedule'` | ⬜ | ⬜ pending |
| 10-05-02 | 05 | 4 | SJOB-01 | T-10-05-02 | legacy scan interval is only migration input; formal schedules drive future recurring work | integration | `cd /root/Mibo/mibo-media-server && go test ./internal/worker ./internal/schedule -run 'Test.*(Migration|LegacyScan)'` | ⬜ | ⬜ pending |
| 10-06-01 | 06 | 5 | SJOB-01..SJOB-08 | T-10-06-01 | typed frontend contract stays aligned with backend schedule fields | typecheck | `cd /root/Mibo/web && pnpm typecheck` | ⬜ | ⬜ pending |
| 10-06-02 | 06 | 5 | SJOB-07, SJOB-08 | T-10-06-02 | workspace list/detail layers render next run, latest result, and history without raw fetches | build | `cd /root/Mibo/web && pnpm build` | ⬜ | ⬜ pending |
| 10-07-01 | 07 | 6 | SJOB-01..SJOB-08 | T-10-07-01 | integrated tests prove schedule CRUD and worker updates stay connected end to end | full | `cd /root/Mibo/mibo-media-server && go test ./... && cd /root/Mibo/web && pnpm build` | ⬜ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Dedicated schedules workspace is the primary entry, with settings only acting as summary/jump-off | SJOB-01..SJOB-08 | information architecture / discoverability | Open `/schedules` from the app shell and confirm the settings “通知与任务” area links into it instead of hosting the full workspace |
| Run history detail layer is understandable and separate from the main list | SJOB-08 | UX layering | Open one schedule history panel and confirm recent runs appear there, not stuffed into the primary grid/table |
| Toggle / run-now feedback uses queued/running/completed/failed language | SJOB-07 | async UI clarity | Trigger a schedule manually and confirm the UI surfaces queued/running/completed/failed state transitions |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency ≤ 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planned
