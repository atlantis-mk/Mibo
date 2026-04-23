# Roadmap: Mibo

## Milestones

- ✅ **v1 MVP** — Phases 1-6 shipped 2026-04-22. Archive: `.planning/milestones/v1-ROADMAP.md`
- 📋 **v2 Product Discovery And Operations** — Phases 7-11 planned

## Overview

v2 pushes Mibo from a solid media-system baseline into a stronger discovery and operations product. This milestone adds native search and filters, trailer playback, metadata governance, recurring maintenance automation, and safer listener-driven refresh behavior while preserving the existing boundary of storage provider/OpenList → `mibo-media-server` → JSON APIs → clients.

## Phases

- [ ] **Phase 7: Metadata Governance & Matching** - Give admins direct control over metadata quality before broader discovery automation builds on it.
- [ ] **Phase 8: Native Search & Discovery Filters** - Let users find content through one product-native search and filtering experience.
- [ ] **Phase 9: Trailer Discovery & Playback** - Surface and play trusted trailers from media detail pages.
- [ ] **Phase 10: Scheduled Operations Control** - Let admins automate recurring maintenance through managed schedules.
- [ ] **Phase 11: Event-Driven Refresh Hardening** - Keep libraries fresh from storage changes through safe listener-driven refresh and reconciliation.

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
**Plans**: TBD
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
**Plans**: TBD
**UI hint**: yes

### Phase 9: Trailer Discovery & Playback
**Goal**: Users can discover and watch available trailers directly from media detail pages.
**Depends on**: Phase 8
**Requirements**: TRLR-01, TRLR-02, TRLR-03, TRLR-04
**Success Criteria** (what must be TRUE):
  1. When TMDB provides a usable trailer, the media detail page shows a clear watch-trailer entry.
  2. User can play the trailer directly from the detail experience.
  3. When no usable trailer exists, the detail page hides trailer actions instead of showing a broken entry.
**Plans**: TBD
**UI hint**: yes

### Phase 10: Scheduled Operations Control
**Goal**: Admins can automate recurring library maintenance through product-native schedules layered on the existing worker model.
**Depends on**: Phase 9
**Requirements**: SJOB-01, SJOB-02, SJOB-03, SJOB-04, SJOB-05, SJOB-06, SJOB-07, SJOB-08
**Success Criteria** (what must be TRUE):
  1. Admin can create and manage recurring schedules for scans, metadata refetches, trailer syncs, library cleanup, invalid-link checks, and artwork refreshes.
  2. Admin can enable or disable a schedule, run it immediately, and see its next run time.
  3. Admin can review each schedule’s latest result and run history.
**Plans**: TBD
**UI hint**: yes

### Phase 11: Event-Driven Refresh Hardening
**Goal**: The system reacts safely to storage changes by turning listener input into conservative refresh work backed by reconciliation.
**Depends on**: Phase 10
**Requirements**: LIST-01, LIST-02, LIST-03, LIST-04
**Success Criteria** (what must be TRUE):
  1. When storage content is added, updated, deleted, or moved, Mibo turns those changes into automatic targeted refresh work.
  2. Bursty or duplicate storage events are coalesced so the same change window does not create noisy duplicate refresh activity.
  3. Reconciliation can recover missed listener events and bring library state back in sync.
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Access & Platform Boundary | v1 MVP | 2/2 | Complete | 2026-04-22 |
| 2. Library & Async Sync Foundation | v1 MVP | 3/3 | Complete | 2026-04-22 |
| 3. Semantic Catalog & Discovery | v1 MVP | 3/3 | Complete | 2026-04-22 |
| 4. Playback Entry & Unified Progress | v1 MVP | 4/4 | Complete | 2026-04-22 |
| 5. Playback Decision Intelligence | v1 MVP | 2/2 | Complete | 2026-04-22 |
| 6. Stable Identity & Incremental Refresh | v1 MVP | 4/4 | Complete | 2026-04-22 |
| 7. Metadata Governance & Matching | v2 Product Discovery And Operations | 0/TBD | Not started | - |
| 8. Native Search & Discovery Filters | v2 Product Discovery And Operations | 0/TBD | Not started | - |
| 9. Trailer Discovery & Playback | v2 Product Discovery And Operations | 0/TBD | Not started | - |
| 10. Scheduled Operations Control | v2 Product Discovery And Operations | 0/TBD | Not started | - |
| 11. Event-Driven Refresh Hardening | v2 Product Discovery And Operations | 0/TBD | Not started | - |
