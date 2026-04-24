# Phase 9: Trailer Discovery & Playback - Plan

**Planned:** 2026-04-24
**Status:** Planned
**Phase Goal:** Users can discover and watch available trailers directly from media detail pages without leaving the detail context.

## Planning Summary

Phase 9 is split into three execution plans so the work lands as one backend-first trailer contract, one detail-page playback UX slice, and one integrated validation pass. The ordering follows `09-CONTEXT.md`: first make trailer data app-owned and detail-ready, then surface the locked detail-area entry and in-page modal playback, then close the phase with selection-rule proof, regression coverage, and frontend polish.

## Plan Inventory

| Plan | Name | Wave | Goal | Requirements |
|------|------|------|------|--------------|
| 01 | Trailer Metadata Sync And Detail Contract | 1 | Extend the TMDB metadata pipeline to fetch, select, persist, and expose one usable trailer result through the existing media detail contract. | TRLR-01, TRLR-02, TRLR-04 |
| 02 | Detail Trailer Entry And In-Context Playback | 2 | Add the detail-area trailer entry and modal playback experience on top of the typed detail contract, while removing the hero placeholder from the formal entry path. | TRLR-02, TRLR-03, TRLR-04 |
| 03 | Integration Validation And Trailer UX Hardening | 3 | Prove the end-to-end trailer flow works across metadata sync, detail rendering, and no-trailer cases, then finish phase-level verification. | TRLR-01, TRLR-02, TRLR-03, TRLR-04 |

## Execution Order

### Plan 01: Trailer Metadata Sync And Detail Contract

**Intent:** Fill the current backend gap identified in `09-CONTEXT.md`: trailer data is not fetched from TMDB, is not persisted in Mibo-owned data, and is not available on `GET /api/v1/media-items/{id}`.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 9-01-01 | Extend the TMDB detail fetch path and metadata service flow to request `videos`, apply the locked trailer selection rule (`official Trailer` -> `Trailer` -> `Teaser`), and normalize one final playable trailer result instead of exposing raw candidate lists to the frontend. | TRLR-01 | T-9-01 | `cd /root/Mibo/mibo-media-server && go test ./internal/metadata -run 'Test.*Trailer'` |
| 9-01-02 | Add app-owned trailer persistence plus detail-query projection so media detail responses carry a stable typed trailer payload and naturally hide the entry when no usable trailer exists. | TRLR-01, TRLR-02, TRLR-04 | T-9-02 | `cd /root/Mibo/mibo-media-server && go test ./internal/library ./internal/httpapi -run 'Test.*Trailer'` |

**Expected outputs:**
- TMDB trailer sync attached to the existing metadata match/refetch lifecycle
- One server-selected trailer result owned by `mibo-media-server`
- Extended detail contract that the frontend can consume without ad hoc TMDB requests

### Plan 02: Detail Trailer Entry And In-Context Playback

**Intent:** Ship the locked Phase 9 user experience: a trailer entry inside the detail/specs area and a modal player that keeps the user inside the existing detail page.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 9-02-01 | Extend the typed web client/query layer and media detail feature so trailer availability comes from the existing detail query, then render the formal trailer entry inside the specs/detail area only when the backend returns a playable trailer. | TRLR-02, TRLR-04 | T-9-03 | `cd /root/Mibo/web && pnpm typecheck` |
| 9-02-02 | Implement an in-page trailer playback modal using the returned trailer payload, preserve the current detail context when opened/closed, and remove or demote the hero `预告片` placeholder so it no longer acts as the formal entry path. | TRLR-03 | T-9-04 | `cd /root/Mibo/web && pnpm build` |

**Expected outputs:**
- Typed trailer shape available in `web/src/lib/mibo-api.ts`
- Detail-area `观看预告片` entry aligned with the Phase 9 decision log
- In-context trailer playback instead of a separate playback route or external navigation

### Plan 03: Integration Validation And Trailer UX Hardening

**Intent:** Close the phase by proving that metadata sync, trailer selection, detail rendering, and no-trailer hiding all stay correct under real backend/frontend flows.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 9-03-01 | Add backend regression coverage for trailer selection priority, detail contract serialization, and no-trailer fallback behavior using the existing TMDB/httptest patterns. | TRLR-01, TRLR-04 | T-9-01 / T-9-02 | `cd /root/Mibo/mibo-media-server && go test ./internal/metadata ./internal/library ./internal/httpapi -run 'Test.*Trailer'` |
| 9-03-02 | Run integrated backend/frontend validation and polish any remaining UI/state issues so modal playback, close behavior, and hidden-entry semantics satisfy all phase success criteria. | TRLR-02, TRLR-03, TRLR-04 | T-9-03 / T-9-04 | `cd /root/Mibo/mibo-media-server && go test ./... && cd /root/Mibo/web && pnpm typecheck && pnpm build` |

**Expected outputs:**
- Regression proof for trailer selection and detail visibility rules
- Full phase verification covering both usable-trailer and no-trailer paths
- Phase-ready execution closeout inputs for later summary/transition work

## Constraints To Preserve During Execution

- Keep work inside `web/` and `mibo-media-server/` only.
- Preserve the `OpenList -> mibo-media-server -> client` boundary.
- Keep trailer sync attached to the existing metadata match/refetch and worker model; do not add request-time TMDB fetches from the detail page.
- Expose exactly one final trailer result to the frontend; do not add a multi-trailer picker in Phase 9.
- Keep playback inside the media detail experience; do not add a dedicated trailer route, external-page jump, proxy, download, or transcoding path.
- Do not pull future requirement `TRLR-05` or scheduled trailer refresh work into this phase.

## Threat References

- **T-9-01:** Wrong TMDB video selection could surface a teaser or non-playable asset even when a better official trailer exists.
- **T-9-02:** Trailer availability could drift between metadata state and detail responses if persistence or query projection is incomplete.
- **T-9-03:** Frontend could reintroduce ad hoc trailer-fetch behavior or show stale/broken entry state when no trailer exists.
- **T-9-04:** Playback UX could accidentally break the locked requirement by routing away from the detail page or leaving the modal without a clean return to context.

## Definition Of Planned

Phase 9 planning is complete when:

1. Each `TRLR-01..TRLR-04` requirement is assigned to a concrete plan/task.
2. Execution order protects the backend trailer contract before the UI playback layer.
3. Validation commands exist for backend sync work, frontend integration work, and phase-level regression checks.
4. Execution can begin with Plan 01 without reopening trailer source, entry placement, or playback-direction questions.

---

*Phase: 09-trailer-discovery-playback*
*Plan created: 2026-04-24*
