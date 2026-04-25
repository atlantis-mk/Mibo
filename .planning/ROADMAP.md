# Roadmap: Mibo v3 剧集元数据治理 catalog kernel 迁移

## Milestones

- ✅ **v1 MVP** — Phases 1-6 shipped 2026-04-22. Archive: `.planning/milestones/v1-ROADMAP.md`
- ✅ **v2 Product Discovery And Operations** — Phases 7-11 shipped 2026-04-24. Archive: `.planning/milestones/v2-ROADMAP.md`
- ◆ **v3 剧集元数据治理 catalog kernel 迁移** — Phases 12-20 planned 2026-04-25

## Phases

<details>
<summary>✅ v1 MVP (Phases 1-6) — SHIPPED 2026-04-22</summary>

- [x] Phase 1: Access & Platform Boundary (2/2 plans) — completed 2026-04-22
- [x] Phase 2: Library & Async Sync Foundation (3/3 plans) — completed 2026-04-22
- [x] Phase 3: Semantic Catalog & Discovery (3/3 plans) — completed 2026-04-22
- [x] Phase 4: Playback Entry & Unified Progress (4/4 plans) — completed 2026-04-22
- [x] Phase 5: Playback Decision Intelligence (2/2 plans) — completed 2026-04-22
- [x] Phase 6: Stable Identity & Incremental Refresh (4/4 plans) — completed 2026-04-22

</details>

<details>
<summary>✅ v2 Product Discovery And Operations (Phases 7-11) — SHIPPED 2026-04-24</summary>

- [x] Phase 7: Metadata Governance & Matching (3/3 plans) — completed 2026-04-24
- [x] Phase 8: Native Search & Discovery Filters (4/4 plans) — completed 2026-04-24
- [x] Phase 9: Trailer Discovery & Playback (1/1 plans) — completed 2026-04-24
- [x] Phase 10: Scheduled Operations Control (7/7 plans) — completed 2026-04-24
- [x] Phase 11: Event-Driven Refresh Hardening (5/5 plans) — completed 2026-04-24

</details>

<details open>
<summary>◆ v3 剧集元数据治理 catalog kernel 迁移 (Phases 12-20) — PLANNED</summary>

- [x] Phase 12: Catalog Kernel Contracts & Migration Guards (6/6 plans) — completed 2026-04-25. Goal: freeze DTOs, status flags, projection contracts, and minimum indexes before any cutover work. Requirements: KERN-01, KERN-02, PROD-01. **Plans:** 6 plans. Success criteria: catalog DTOs are explicit; migration flags exist; projection refresh entrypoints are defined; existing tests still boot empty and legacy databases.
   Plans:
   - [x] 12-01-PLAN.md — Freeze explicit catalog DTO contracts for list/detail/season/episode/asset/governance responses.
   - [x] 12-02-PLAN.md — Add durable catalog migration state settings and authenticated observability endpoints.
   - [x] 12-03-PLAN.md — Define catalog projection refresh queue/worker entrypoints and scan-trigger wiring.
   - [x] 12-04-PLAN.md — Add minimum composite indexes plus empty/legacy startup regression coverage.
   - [x] 12-05-PLAN.md — Close DTO summary/value leakage so frozen catalog contracts never expose raw provider blobs.
   - [x] 12-06-PLAN.md — Canonicalize projection rebuild search documents for legacy `show` rows and blank availability states.
- [ ] Phase 13: Legacy Backfill Into Catalog Kernel — Goal: idempotently convert existing `MediaItem` / `MediaFile` / progress data into catalog items, assets, inventory files, images, external IDs, and migration reports. Requirements: MIGR-01, MIGR-02, MIGR-03. **Plans:** 5 plans. Success criteria: backfill is repeat-safe; movies and series hierarchy map correctly; conflicts are reported; all legacy playable rows map to assets.
   Plans:
   - [x] 13-01-PLAN.md — Define durable backfill run/report schema and catalog service contracts.
   - [x] 13-02-PLAN.md — Add authenticated backfill trigger/report APIs and worker dispatch.
   - [x] 13-03-PLAN.md — Backfill legacy movies into catalog items, inventory files, assets, images, and provider identity.
   - [x] 13-04-PLAN.md — Backfill legacy series hierarchy with duplicate-slot and orphan-file reporting.
   - [x] 13-05-PLAN.md — Migrate progress, refresh projections, and finalize repeat-safe backfill runs.
- [ ] Phase 14: Scanner Writes Catalog Assets — Goal: rebuild scanner writes so new scans create inventory files, media assets, asset files, catalog items, and asset-item links directly. Requirements: SCAN-01, SCAN-02, SCAN-03. **Plans:** 4 plans. Success criteria: scans no longer create legacy media rows; movies and episodes create catalog rows; multi-episode and multi-version files link correctly; deletes update availability only.
   Plans:
   - [ ] 14-01-PLAN.md — Define the catalog-first scan writer boundary and direct-write contracts.
   - [x] 14-02-PLAN.md — Switch scan traversal to catalog writes with multi-episode and version asset modeling.
   - [ ] 14-03-PLAN.md — Replace legacy media-file probe jobs with inventory-file probe and media-stream enrichment.
   - [ ] 14-04-PLAN.md — Preserve governed catalog metadata across deletes and stable-identity rescans by updating availability only.
- [ ] Phase 15: Series-Level Metadata Governance Engine — Goal: match and refresh metadata from the series root, generating governed seasons and episodes from provider evidence. Requirements: META-01, META-02, META-03, META-04. Success criteria: series match writes external IDs and sources; seasons/episodes are generated idempotently; locked fields are preserved; images/people/tags normalize into catalog tables.
- [ ] Phase 16: Catalog API, Search, and Progress Cutover — Goal: expose catalog-backed items, series seasons, governance workspace, search, discovery, and progress APIs. Requirements: API-01, API-02, API-03, API-04. **Plans:** 4 plans. Success criteria: list/detail/search use catalog projections; series seasons come from catalog hierarchy; governance reads field states/sources/images/assets; progress writes `user_item_data`.
   Plans:
   - [ ] 16-01-PLAN.md — Build catalog query helpers for browse, detail, and series-season reads.
   - [ ] 16-02-PLAN.md — Add governance workspace and mutation helpers for field locks, image selection, and asset links.
   - [ ] 16-03-PLAN.md — Move catalog progress persistence to `user_item_data` with item/asset validation.
   - [ ] 16-04-PLAN.md — Expose additive `/api/v1/items*`, governance, and catalog progress HTTP routes.
- [ ] Phase 17: Playback Item-to-Asset Cutover — Goal: resolve playback from catalog item to selected asset/version and inventory file rather than legacy media file selection. Requirements: PLAY-01, PLAY-02, PLAY-03. **Plans:** 3 plans. Success criteria: default asset selection works; explicit asset playback works; HLS/direct streams resolve inventory files; missing files return explainable unplayable decisions.
   Plans:
   - [ ] 17-01-PLAN.md — Cut the playback service over to catalog item, asset, and inventory-file identifiers with deterministic asset selection.
   - [ ] 17-02-PLAN.md — Add authenticated catalog item playback and explicit asset-link HTTP routes.
   - [ ] 17-03-PLAN.md — Move direct stream and HLS endpoints to inventory-file ids and keep missing files explainable.
- [ ] Phase 18: Frontend Catalog Item Migration — Goal: migrate home, library, search, detail, series, and playback UI types and queries from `MediaItem` to `CatalogItem`. Requirements: UI-01, UI-02, UI-03, UI-04. **Plans:** 4 plans. Success criteria: `pnpm typecheck` and build pass; core pages render catalog items; missing/unaired/unavailable states are visible; playback can pass asset selection.
   Plans:
   - [ ] 18-01-PLAN.md — Add catalog frontend contracts, shared query keys, and presentation helpers.
   - [ ] 18-02-PLAN.md — Migrate home, library, and search browse surfaces to catalog item queries and availability badges.
   - [ ] 18-03-PLAN.md — Move detail and series hierarchy rendering to catalog item, season, episode, asset, and progress DTOs.
   - [ ] 18-04-PLAN.md — Cut playback entry and progress persistence over to catalog item + optional asset selection.
- [ ] Phase 19: Metadata Governance UI Rebuild — Goal: rebuild governance UI around catalog field locks, source evidence, image candidates, external IDs, and asset links. Requirements: GOV-01, GOV-02, GOV-03, GOV-04. **Plans:** 4 plans. Success criteria: field locks persist and protect refreshes; source evidence is inspectable; image selection updates catalog display; asset links explain playability.
   Plans:
   - [ ] 19-01-PLAN.md — Add catalog governance frontend contracts, query keys, and normalized feature hooks.
   - [ ] 19-02-PLAN.md — Rebuild the governance workspace routes and catalog item entry/navigation.
   - [ ] 19-03-PLAN.md — Replace the legacy detail editor with field-state and source-evidence panels.
   - [ ] 19-04-PLAN.md — Add image candidate selection and asset-link explainability panels.
- [ ] Phase 20: Legacy Model Retirement & Production Hardening — Goal: remove legacy main-path dependencies and harden the new kernel with constraints, indexes, consistency checkers, projection rebuilds, docs, and cleanup strategy. Requirements: PROD-02, PROD-03, PROD-04, MIGR-04. **Plans:** 6 plans. Success criteria: no main path writes legacy tables; constraints/indexes are applied safely; projection repair jobs exist; full backend and frontend validation passes.
   Plans:
   - [ ] 20-01-PLAN.md — Define deterministic catalog consistency audit contracts for duplicate keys, availability drift, and stale projections.
   - [ ] 20-02-PLAN.md — Add authenticated catalog consistency audit/repair jobs and operator endpoints while preserving migration visibility.
   - [ ] 20-03-PLAN.md — Harden startup with explicit catalog-kernel constraint/index backstops and duplicate-safe regression tests.
   - [ ] 20-04-PLAN.md — Remove legacy browse/search/progress runtime dependencies in favor of catalog projections and `user_item_data`.
   - [ ] 20-05-PLAN.md — Remove legacy playback/file runtime routes and keep only the read-only migration compatibility boundary.
   - [ ] 20-06-PLAN.md — Clean frontend legacy contracts and add the production validation plus cleanup runbook.

</details>

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Access & Platform Boundary | v1 MVP | 2/2 | Complete | 2026-04-22 |
| 2. Library & Async Sync Foundation | v1 MVP | 3/3 | Complete | 2026-04-22 |
| 3. Semantic Catalog & Discovery | v1 MVP | 3/3 | Complete | 2026-04-22 |
| 4. Playback Entry & Unified Progress | v1 MVP | 4/4 | Complete | 2026-04-22 |
| 5. Playback Decision Intelligence | v1 MVP | 2/2 | Complete | 2026-04-22 |
| 6. Stable Identity & Incremental Refresh | v1 MVP | 4/4 | Complete | 2026-04-22 |
| 7. Metadata Governance & Matching | v2 Product Discovery And Operations | 3/3 | Complete | 2026-04-24 |
| 8. Native Search & Discovery Filters | v2 Product Discovery And Operations | 4/4 | Complete | 2026-04-24 |
| 9. Trailer Discovery & Playback | v2 Product Discovery And Operations | 1/1 | Complete | 2026-04-24 |
| 10. Scheduled Operations Control | v2 Product Discovery And Operations | 7/7 | Complete | 2026-04-24 |
| 11. Event-Driven Refresh Hardening | v2 Product Discovery And Operations | 5/5 | Complete | 2026-04-24 |
| 12. Catalog Kernel Contracts & Migration Guards | v3 Catalog Kernel Migration | 6/6 | Complete | 2026-04-25 |
| 13. Legacy Backfill Into Catalog Kernel | v3 Catalog Kernel Migration | 5/5 | Complete    | 2026-04-25 |
| 14. Scanner Writes Catalog Assets | v3 Catalog Kernel Migration | 1/4 | In Progress | |
| 15. Series-Level Metadata Governance Engine | v3 Catalog Kernel Migration | 0/0 | Planned | |
| 16. Catalog API, Search, and Progress Cutover | v3 Catalog Kernel Migration | 0/4 | Planned | |
| 17. Playback Item-to-Asset Cutover | v3 Catalog Kernel Migration | 0/3 | Planned | |
| 18. Frontend Catalog Item Migration | v3 Catalog Kernel Migration | 0/4 | Planned | |
| 19. Metadata Governance UI Rebuild | v3 Catalog Kernel Migration | 0/4 | Planned | |
| 20. Legacy Model Retirement & Production Hardening | v3 Catalog Kernel Migration | 0/0 | Planned | |

## Backlog

- Future client-specific polish for TV/mobile can start after the catalog kernel is the primary API and playback model.
