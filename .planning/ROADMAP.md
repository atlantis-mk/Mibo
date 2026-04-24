# Roadmap: Mibo

## Milestones

- ✅ **v1 MVP** — Phases 1-6 shipped 2026-04-22. Archive: `.planning/milestones/v1-ROADMAP.md`
- 🚧 **v2 Product Discovery And Operations** — Phase 7 shipped; Phases 8-11 planned

## Overview

v2 pushes Mibo from a solid media-system baseline into a stronger discovery and operations product. This milestone adds native search and filters, trailer playback, metadata governance, recurring maintenance automation, and safer listener-driven refresh behavior while preserving the existing boundary of storage provider/OpenList → `mibo-media-server` → JSON APIs → clients.

## Phases

- [x] **Phase 7: Metadata Governance & Matching** - Give admins direct control over metadata quality before broader discovery automation builds on it.
- [x] **Phase 8: Native Search & Discovery Filters** - Let users find content through one product-native search and filtering experience. Completed 2026-04-24.
- [x] **Phase 9: Trailer Discovery & Playback** - Surface and play trusted trailers from media detail pages. Completed 2026-04-24.
- [x] **Phase 10: Scheduled Operations Control** - Let admins automate recurring maintenance through managed schedules. Completed 2026-04-24.
- [x] **Phase 11: Event-Driven Refresh Hardening** - Keep libraries fresh from storage changes through safe listener-driven refresh and reconciliation. Completed 2026-04-24.

## Phase Details

### Phase 7: Metadata Governance & Matching
**Goal**: Admins can correct, enrich, and refresh app-owned media metadata so discovery quality no longer depends only on provider imports.
**Depends on**: Phase 6
**Requirements**: META-01, META-02, META-03, META-04, META-05, META-06
**Success Criteria** (what must be TRUE):
  1. Admin can edit title, original title, year, and overview for a media item and see those values persist on later visits.
  2. Admin can replace poster and backdrop artwork and see updated images on affected media surfaces.
  3. Admin can update genres, cast, and season/episode basics, and the saved metadata appears in the item’s managed record.
  4. Admin can run re-match and metadata refetch as separate actions on a media item.
**Plans**: 3 (`07-PLAN.md`: Plan 01-03)
**UI hint**: yes

### Phase 8: Native Search & Discovery Filters
**Goal**: Users can find media quickly through one native discovery contract shared by search and browse surfaces.
**Depends on**: Phase 7
**Requirements**: SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-05, SRCH-06, SRCH-07, SRCH-08, FLTR-01, FLTR-02, FLTR-03, FLTR-04, FLTR-05, FLTR-06
**Success Criteria** (what must be TRUE):
  1. User can search by title, original title, actor, or director and get results inside Mibo without leaving the product.
  2. Search results clearly distinguish movies and shows, highlight matched terms, and support sort changes.
  3. User can reopen recent searches from preserved search history.
  4. User can apply genre, year, region, rating, watched-state, and shared sort controls consistently across search and browse results.
**Plans**: 4 plans
Plans:
- [x] 08-PLAN.md — Original Phase 8 decomposition used for the first execution pass.
- [x] 08-02-PLAN.md — Add discovery projection foundation and metadata-backed region/rating fields.
- [x] 08-03-PLAN.md — Wire projection freshness into scan, metadata, and progress lifecycles.
- [x] 08-04-PLAN.md — Add regression proof for the remaining discovery freshness gaps.
**UI hint**: yes

### Phase 9: Trailer Discovery & Playback
**Goal**: Users can discover and watch available trailers directly from media detail pages.
**Depends on**: Phase 8
**Requirements**: TRLR-01, TRLR-02, TRLR-03, TRLR-04
**Success Criteria** (what must be TRUE):
  1. When TMDB provides a usable trailer, the media detail page shows a clear watch-trailer entry.
  2. User can play the trailer directly from the detail experience.
  3. When no usable trailer exists, the detail page hides trailer actions instead of showing a broken entry.
**Plans**: 3 (`09-PLAN.md`: Plan 01-03)
**UI hint**: yes

### Phase 10: Scheduled Operations Control
**Goal**: Admins can automate recurring library maintenance through product-native schedules layered on the existing worker model.
**Depends on**: Phase 9
**Requirements**: SJOB-01, SJOB-02, SJOB-03, SJOB-04, SJOB-05, SJOB-06, SJOB-07, SJOB-08
**Success Criteria** (what must be TRUE):
  1. Admin can create and manage recurring schedules for scans, metadata refetches, trailer syncs, library cleanup, invalid-link checks, and artwork refreshes.
  2. Admin can enable or disable a schedule, run it immediately, and see its next run time.
  3. Admin can review each schedule’s latest result and run history.
**Plans**: 7 plans
Plans:
- [x] 10-01-PLAN.md — Add the persisted schedule domain, recurrence math, and schedule-centric history foundations.
- [x] 10-02-PLAN.md — Implement library-owned maintenance executors for scan, cleanup, and invalid-link checks.
- [x] 10-03-PLAN.md — Implement metadata-owned maintenance executors for metadata refetch, trailer sync, and artwork refresh.
- [x] 10-04-PLAN.md — Expose authenticated schedule CRUD/toggle/run-now/history APIs on top of the schedule service.
- [x] 10-05-PLAN.md — Wire due schedules and run-history propagation into the existing worker lifecycle.
- [x] 10-06-PLAN.md — Build the dedicated schedules workspace route and typed frontend schedule contract.
- [x] 10-07-PLAN.md — Finish schedule mutations, history detail UI, and the settings summary entry.
**UI hint**: yes

### Phase 11: Event-Driven Refresh Hardening
**Goal**: The system reacts safely to storage changes by turning listener input into conservative refresh work backed by reconciliation.
**Depends on**: Phase 10
**Requirements**: LIST-01, LIST-02, LIST-03, LIST-04
**Success Criteria** (what must be TRUE):
  1. When storage content is added, updated, deleted, or moved, Mibo turns those changes into automatic targeted refresh work.
  2. Bursty or duplicate storage events are coalesced so the same change window does not create noisy duplicate refresh activity.
  3. Reconciliation can recover missed listener events and bring library state back in sync.
**Plans**: 5 plans
Plans:
- [x] 11-01-PLAN.md — Add the listener-domain service with explicit debounce, path coalescing, and reconciliation coverage. Completed 2026-04-24.
- [x] 11-02-PLAN.md — Route `/api/v1/storage-events` through the listener service with conservative normalization and API regressions. Completed 2026-04-24.
- [x] 11-03-PLAN.md — Extend the worker to apply coalesced listener jobs and maintain periodic reconciliation. Completed 2026-04-24.
- [ ] 11-04-PLAN.md — Close the OpenList `/` root storage-event validation gap.
- [ ] 11-05-PLAN.md — Add atomic active-intent guards for concurrent listener refresh and reconciliation jobs.

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
| 11. Event-Driven Refresh Hardening | v2 Product Discovery And Operations | 3/3 | Complete | 2026-04-24 |
