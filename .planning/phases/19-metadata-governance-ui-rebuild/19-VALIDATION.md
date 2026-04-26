---
phase: 19
slug: metadata-governance-ui-rebuild
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-25
---

# Phase 19 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | frontend typecheck + production build |
| **Config file** | `web/tsconfig.json`, `web/vite.config.ts` |
| **Quick run command** | `cd web && pnpm typecheck` |
| **Full suite command** | `cd web && pnpm typecheck && pnpm build` |
| **Estimated runtime** | ~40-60 seconds |

---

## Sampling Rate

- **After every task commit:** run the task-specific `pnpm typecheck` command.
- **After every plan wave:** run `cd web && pnpm typecheck && pnpm build`.
- **Before `/gsd-verify-work`:** production build must be green.
- **Max feedback latency:** 60 seconds.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 19-01-01 | 01 | 1 | GOV-01 / GOV-02 / GOV-03 / GOV-04 | T-19-01 / T-19-02 | all catalog governance reads and writes go through typed helpers instead of raw legacy endpoints | static | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 19-01-02 | 01 | 1 | GOV-01 / GOV-02 / GOV-03 / GOV-04 | T-19-03 / T-19-04 | feature hooks normalize untrusted payloads and invalidate the right caches after mutation | static | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 19-02-01 | 02 | 2 | GOV-01 / GOV-02 / GOV-03 / GOV-04 | T-19-05 / T-19-06 | workspace routes and cards use catalog item ids and bounded filter state | static | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 19-02-02 | 02 | 2 | GOV-01 / GOV-02 / GOV-03 / GOV-04 | T-19-05 / T-19-06 | settings/detail entry points navigate to the rebuilt governance flow without legacy id drift | static | `cd web && pnpm typecheck && pnpm build` | ✅ | ⬜ pending |
| 19-03-01 | 03 | 2 | GOV-01 | T-19-07 / T-19-08 | field-state edits and lock toggles stay item-scoped and render provenance safely | static | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 19-03-02 | 03 | 2 | GOV-02 | T-19-08 / T-19-09 | source evidence filters and payload views render as text/JSON without HTML injection | static | `cd web && pnpm typecheck && pnpm build` | ✅ | ⬜ pending |
| 19-04-01 | 04 | 3 | GOV-03 | T-19-10 / T-19-11 | image selection preserves alternatives and only changes the selected candidate | static | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 19-04-02 | 04 | 3 | GOV-04 | T-19-12 / T-19-13 | asset-link actions mutate one edge at a time and explain playability without leaking raw provider internals | static | `cd web && pnpm typecheck && pnpm build` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

- Final phase verification should include a local smoke check of `/settings/metadata`
  and `/settings/metadata/{itemId}` after the backend catalog-governance APIs are
  available, but all plan tasks still have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
