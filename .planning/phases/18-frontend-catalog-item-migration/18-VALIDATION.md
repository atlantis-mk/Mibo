---
phase: 18
slug: frontend-catalog-item-migration
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-25
---

# Phase 18 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | TypeScript compiler + Vite production build |
| **Config file** | `web/tsconfig.json`, `web/vite.config.ts` |
| **Quick run command** | `cd web && pnpm typecheck` |
| **Full suite command** | `cd web && pnpm typecheck && pnpm build` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd web && pnpm typecheck`
- **After every plan wave:** Run `cd web && pnpm typecheck && pnpm build`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 18-01-01 | 01 | 1 | UI-01, UI-02 | T-18-01 | Catalog frontend contracts stop carrying legacy `media_file_id` / `media_item_id` assumptions into new browse/detail code | typecheck | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 18-02-01 | 02 | 2 | UI-01 | T-18-02 | Home, library, and search surfaces render `CatalogListItem` states without hiding `availability_status`-driven differences | typecheck | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 18-03-01 | 03 | 3 | UI-02, UI-03 | T-18-03 | Detail and series pages surface selected images, assets, and `available` / `missing` / `unaired` / `no_local_media` states directly from catalog hierarchy | typecheck | `cd web && pnpm typecheck` | ✅ | ⬜ pending |
| 18-04-01 | 04 | 4 | UI-04 | T-18-04 | Playback and progress mutations use catalog item + optional asset identifiers and still build a production bundle | build | `cd web && pnpm typecheck && pnpm build` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Home, library, search, detail, and playback visually present catalog availability and asset-selection states | UI-01, UI-02, UI-03, UI-04 | No frontend visual/e2e harness is installed in `web/` | Run the web app against backend phases 16-17, then validate `/`, `/library/:id`, `/search`, `/media/:id?view=series`, and `/play/:id` with both playable and non-playable catalog items |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 45s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
