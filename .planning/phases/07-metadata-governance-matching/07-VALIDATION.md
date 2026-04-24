---
phase: 7
slug: metadata-governance-matching
status: complete
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-24
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` + frontend `pnpm` type/build checks |
| **Config file** | `mibo-media-server/go.mod`, `web/package.json`, `web/tsconfig.json`, `web/vite.config.ts` |
| **Quick run command** | `cd mibo-media-server && go test ./internal/httpapi ./internal/metadata && cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm typecheck` |
| **Full suite command** | `cd mibo-media-server && go test ./... && cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm build` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd mibo-media-server && go test ./internal/httpapi ./internal/metadata && cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm typecheck`
- **After every plan wave:** Run `cd mibo-media-server && go test ./... && cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm build`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 7-01-01 | 01 | 1 | META-01, META-02 | T-7-01 | mutation endpoints reject malformed payloads and require auth | integration | `cd mibo-media-server && go test ./internal/httpapi -run 'Test.*Metadata'` | ✅ | ✅ green |
| 7-01-02 | 01 | 1 | META-03, META-04 | T-7-02 | season/episode and semi-automatic metadata paths preserve validated IDs and numbers | unit/integration | `cd mibo-media-server && go test ./internal/metadata -run 'Test(ListTV|ListSeason|Match)'` | ✅ | ✅ green |
| 7-02-01 | 02 | 2 | META-01, META-02, META-05, META-06 | T-7-03 | governance route only exposes admin-triggered actions via typed API client | type/build | `cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm typecheck` | ✅ | ✅ green |
| 7-02-02 | 02 | 2 | META-05, META-06 | T-7-04 | async rematch/refetch UI reflects queued/running/completed states without silent failure | build/manual | `cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm build` | ✅ | ✅ green |
| 7-03-01 | 03 | 3 | META-01..META-06 | T-7-01 / T-7-04 | integrated flow persists edits and refreshes detail surfaces after background work | full | `cd mibo-media-server && go test ./... && cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm build` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Draft leave guard appears before navigating away with unsaved edits | META-01 | browser navigation prompt / UX confirmation | Verified in browser: changed title, attempted to leave, dismissing the confirm kept the draft on the governance page |
| Candidate diff/preview is shown before apply | META-05 | visual comparison is product UX | Verified in browser: searched TMDB candidate and saw side-by-side current-vs-candidate preview before apply |
| Queued/running/completed status is understandable for rematch/refetch | META-05, META-06 | async state clarity is UX behavior | Verified in browser: metadata refetch surfaced completion feedback and refreshed the governance/detail data; local worker completed too quickly to reliably capture a visible queued/running intermediate frame |
| Admin entry points are reachable from both detail and admin/global navigation | META-01..META-06 | route discoverability is navigational UX | Verified in browser: entered governance from Settings > 元数据治理 and from media detail > 治理元数据 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete
