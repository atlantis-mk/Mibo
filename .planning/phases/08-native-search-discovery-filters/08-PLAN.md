# Phase 8: Native Search & Discovery Filters - Plan

**Planned:** 2026-04-24
**Status:** Planned
**Phase Goal:** Users can find media quickly through one native discovery contract shared by search and browse surfaces.

## Planning Summary

Phase 8 is split into three execution plans so the work lands as vertical slices while protecting the key Phase 8 constraint: search and browse must converge on one shared discovery contract instead of growing two parallel systems. The order follows `08-CONTEXT.md`: first establish the backend contract and search projection, then ship the global search and shared filter experience, then close with integration hardening and full phase validation.

## Plan Inventory

| Plan | Name | Wave | Goal | Requirements |
|------|------|------|------|--------------|
| 01 | Shared Discovery Contract And Search Foundation | 1 | Replace the current stubbed search path with a real backend discovery contract, persistent search history, and a projection/index foundation that browse can grow onto. | SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-07, SRCH-08, FLTR-01, FLTR-02, FLTR-03, FLTR-04, FLTR-05, FLTR-06 |
| 02 | Global Search UI And Shared Filter Controls | 2 | Ship the product-native global search entry, results experience, history rerun flow, and shared filter/sort controls across search and library browse surfaces. | SRCH-05, SRCH-06, SRCH-07, SRCH-08, FLTR-01, FLTR-02, FLTR-03, FLTR-04, FLTR-05, FLTR-06 |
| 03 | Reindex Integration, Contract Consistency, And Validation | 3 | Prove the full discovery loop stays consistent after scan, metadata, progress, and UI interactions, then complete phase verification against all success criteria. | SRCH-01..SRCH-08, FLTR-01..FLTR-06 |

## Execution Order

### Plan 01: Shared Discovery Contract And Search Foundation

**Intent:** Establish the backend shape that Phase 8 depends on: one normalized discovery query contract, app-owned search projection/indexing, and per-user recent search persistence. This plan directly addresses the current stub in `internal/search/service.go`, the browse-only shape in `internal/library/query.go`, and the risk of filter semantics diverging between browse and search.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 8-01-01 | Expand the backend discovery input model and HTTP read contract so search and browse share the same server-side filters for type, year, region, rating, watched-state, and sort without frontend-only translation logic. | SRCH-07, FLTR-01, FLTR-02, FLTR-03, FLTR-04, FLTR-05, FLTR-06 | T-8-01 | `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi` |
| 8-01-02 | Add persistent app-owned search projection and history storage, including SQLite FTS5 readiness checks, searchable title/original-title/people fields, and typed search/history endpoints. | SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-08 | T-8-02 | `cd /root/Mibo/mibo-media-server && go test ./internal/search ./internal/database ./internal/httpapi` |
| 8-01-03 | Extend browse/search service logic so grouped show semantics, watched-state derivation, and shared sort meanings are computed on the server from one contract instead of page-specific behavior. | FLTR-05, FLTR-06 | T-8-03 | `cd /root/Mibo/mibo-media-server && go test ./internal/library ./internal/progress` |

**Expected outputs:**
- Real Phase 8 backend contract under `mibo-media-server/` for discovery search and recent history
- Search projection/index path owned by Mibo rather than ad hoc JSON scans or external middleware
- Backend tests covering contract normalization, history persistence, and watched-state semantics

### Plan 02: Global Search UI And Shared Filter Controls

**Intent:** Turn the existing placeholder frontend search affordance into the locked global search entry, then reuse the same typed contract on both the search surface and library browse surface so filters and sort behave identically.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 8-02-01 | Replace the placeholder sidebar search with a real global Mibo search entry and add typed client/query integration for search execution, recent history loading, and one-click rerun. | SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-08 | T-8-04 | `cd /root/Mibo/web && pnpm typecheck` |
| 8-02-02 | Build the search results surface so hits distinguish movies vs shows, render matched-term highlights, expose unified sort changes, and stay aligned with current app-shell and discovery visuals. | SRCH-05, SRCH-06, SRCH-07 | T-8-05 | `cd /root/Mibo/web && pnpm build` |
| 8-02-03 | Extend library browse and search results to share one filter model for genre, year, region, rating, watched-state, and sort, including the locked three-state watched filter semantics. | FLTR-01, FLTR-02, FLTR-03, FLTR-04, FLTR-05, FLTR-06 | T-8-01 / T-8-03 | `cd /root/Mibo/web && pnpm build` |

**Expected outputs:**
- Global search entry integrated into the existing app shell
- Search results UI with type distinction, highlight rendering, sort controls, and recent history rerun
- Shared frontend filter state and typed API usage across search and library discovery surfaces

### Plan 03: Reindex Integration, Contract Consistency, And Validation

**Intent:** Close the phase by wiring projection freshness into the existing catalog lifecycle and proving that search, browse, metadata edits, and watched-state changes continue to agree after real system updates.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 8-03-01 | Hook reindex/projection refresh into metadata and catalog change paths so search/highlight/filter results stay in sync after scan, rematch, metadata edits, and progress updates. | SRCH-01..SRCH-08, FLTR-01..FLTR-06 | T-8-02 | `cd /root/Mibo/mibo-media-server && go test ./...` |
| 8-03-02 | Add integrated regression checks that the same discovery inputs produce consistent browse/search semantics for grouped shows, sort ordering, and watched-state filtering. | FLTR-05, FLTR-06 | T-8-01 / T-8-03 | `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi ./internal/library ./internal/search` |
| 8-03-03 | Run full backend/frontend validation and manual checks for global search entry, history reopen/rerun, highlight visibility, and shared filters before marking the phase ready for execution closeout. | SRCH-01..SRCH-08, FLTR-01..FLTR-06 | T-8-04 / T-8-05 | `cd /root/Mibo/mibo-media-server && go test ./... && cd /root/Mibo/web && pnpm typecheck && pnpm build` |

**Expected outputs:**
- Search projection freshness tied into the existing Mibo catalog lifecycle
- Regression coverage for shared discovery semantics across browse and search
- Full phase validation ready for later execution summary and roadmap transition

## Constraints To Preserve During Execution

- Keep work inside `web/` and `mibo-media-server/` only.
- Preserve the `OpenList -> mibo-media-server -> client` boundary.
- Keep Phase 8 product-native and app-owned; do not add Elasticsearch, Meilisearch, Bleve, or other external search middleware.
- Keep search and browse on one backend discovery contract; do not add page-local parameter translation layers.
- Preserve the locked watched-state semantics: `未看 / 观看中 / 已看`.
- Do not pull `FLTR-07` library filter or `FLTR-08` resolution filter into this phase.

## Definition Of Planned

Phase 8 planning is complete when:

1. Every `SRCH-01..SRCH-08` and `FLTR-01..FLTR-06` requirement is assigned to a concrete plan/task.
2. Execution order protects the shared discovery contract before UI expansion.
3. Validation commands exist for backend contract work, frontend integration work, and phase-level regression checks.
4. Execution can begin with Plan 01 without reopening Phase 8 scope or implementation-direction questions.

---

*Phase: 08-native-search-discovery-filters*
*Plan created: 2026-04-24*
