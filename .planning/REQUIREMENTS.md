# Requirements: Mibo v3 剧集元数据治理 catalog kernel 迁移

**Defined:** 2026-04-25
**Core Value:** 无论底层媒体文件来自本地磁盘、NAS 还是云盘，用户都能稳定地完成媒体库接入、内容浏览、播放和进度同步。

## v3 Requirements

### Kernel Contracts

- [ ] **KERN-01**: Developer can use explicit catalog DTOs for list, detail, season, episode, asset, and governance responses instead of exposing raw GORM models.
- [ ] **KERN-02**: System can track migration and read-cutover state through durable settings so backfill, catalog read enablement, and legacy cleanup are observable.

### Migration

- [x] **MIGR-01**: Administrator can run an idempotent backfill that maps legacy movies, series, seasons, episodes, files, artwork, external IDs, and progress into the catalog kernel.
- [x] **MIGR-02**: Administrator can inspect a migration report that lists successful rows, skipped rows, conflicts, orphan files, and duplicate episode candidates.
- [x] **MIGR-03**: System can safely run catalog backfill repeatedly without creating duplicate catalog items, assets, files, or links.
- [ ] **MIGR-04**: Administrator can keep a read-only legacy migration path until cleanup is complete, while normal runtime no longer depends on legacy main-path writes.

### Scanner

- [ ] **SCAN-01**: System can scan movies and create or reuse `catalog_items(type=movie)`, `inventory_files`, `media_assets`, `asset_files`, and `asset_items` without creating new legacy media rows.
- [ ] **SCAN-02**: System can scan TV episode files and create or reuse series, season, and episode catalog hierarchy with local evidence and pending governance status.
- [ ] **SCAN-03**: System can model multi-episode files, multi-version episode files, and file deletion by updating asset links and availability instead of deleting governed catalog metadata.

### Metadata Governance Engine

- [ ] **META-01**: Administrator can match a series catalog item to a provider identity and store provider payloads as source evidence.
- [ ] **META-02**: System can generate or update season and episode catalog items from a matched series provider record without duplicating local scanned episodes.
- [ ] **META-03**: System can canonicalize provider fields through `metadata_field_states` while preserving locked or manually edited fields during refresh.
- [ ] **META-04**: System can normalize images, people, tags, ratings, runtime, and air dates into catalog-owned tables and projections.

### Catalog APIs

- [ ] **API-01**: Client can list library items from catalog projections, with movies and series as primary browse units.
- [ ] **API-02**: Client can fetch item detail and series seasons from catalog hierarchy, including available, missing, unaired, and no-local-media states.
- [ ] **API-03**: Administrator can use catalog governance APIs to read and mutate field locks, source evidence, image selection, external IDs, and asset links.
- [ ] **API-04**: Client can search catalog items and update progress using catalog item and asset IDs instead of legacy media item IDs.

### Playback

- [ ] **PLAY-01**: User can start playback from a catalog item and receive a selected playable asset/version with an explainable decision.
- [ ] **PLAY-02**: User can choose a specific asset/version for an item when multiple versions exist.
- [ ] **PLAY-03**: System can resolve HLS or direct stream URLs from asset files and inventory files, and return a clear unplayable response when files are unavailable.

### Frontend Catalog Migration

- [ ] **UI-01**: User can browse home, library, and search surfaces backed by `CatalogItem` data instead of legacy `MediaItem` data.
- [ ] **UI-02**: User can view movie, series, season, and episode details with selected catalog images, assets, progress, and availability states.
- [ ] **UI-03**: User can open series seasons and see available, missing, unaired, and unavailable episodes from the catalog hierarchy.
- [ ] **UI-04**: User can start playback from catalog item detail and optionally select a playable version/asset.

### Governance UI

- [ ] **GOV-01**: Administrator can view and edit canonical catalog fields with visible source, confidence, lock status, and edit metadata.
- [ ] **GOV-02**: Administrator can inspect metadata source evidence and provider payload summaries without losing canonical field provenance.
- [ ] **GOV-03**: Administrator can select poster, backdrop, logo, still, and other image candidates per catalog item without deleting alternatives.
- [ ] **GOV-04**: Administrator can inspect and repair item-to-asset links so playability issues are explainable and actionable.

### Production Hardening

- [ ] **PROD-01**: System has critical indexes and uniqueness guarantees for catalog hierarchy, provider identity, field state, asset links, inventory files, and search projections.
- [ ] **PROD-02**: System has consistency checkers and repair jobs for availability, rollups, search documents, and projection freshness.
- [ ] **PROD-03**: Developer can remove or isolate legacy `MediaItem` / `MediaFile` query, search, metadata, playback, and progress code from normal runtime paths.
- [ ] **PROD-04**: Operator can validate empty database startup, legacy database migration startup, repeated startup, backend tests, frontend typecheck, and frontend build before cleanup.

## Future Requirements

### Client Expansion

- **CLNT-01**: TV and mobile clients can consume the catalog item and asset APIs after the Web client migration stabilizes.
- **CLNT-02**: User can manage alternate ordering modes such as DVD order, absolute order, and manual order through dedicated UI.

### Advanced Governance

- **GOV-05**: Administrator can compare multiple provider candidates side by side before applying a match.
- **GOV-06**: Administrator can bulk resolve duplicate or conflicting series identities across libraries.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Rewriting OpenList storage internals | Storage access remains behind `mibo-media-server/internal/storage`; this milestone is media catalog governance, not storage-provider redesign. |
| Building new mobile or TV apps | Catalog API should support future clients, but this milestone validates Web first. |
| Removing legacy database tables before migration verification | Cleanup must wait until catalog read/write/playback/front-end paths are verified and repair tools exist. |
| Adding external search middleware | Catalog search projections are sufficient for this migration; external infrastructure would add deployment risk before the new model stabilizes. |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| KERN-01 | Phase 12 | Pending |
| KERN-02 | Phase 12 | Pending |
| PROD-01 | Phase 12 | Pending |
| MIGR-01 | Phase 13 | Pending |
| MIGR-02 | Phase 13 | Validated in 13-02 |
| MIGR-03 | Phase 13 | Validated in 13-01 |
| SCAN-01 | Phase 14 | Pending |
| SCAN-02 | Phase 14 | Pending |
| SCAN-03 | Phase 14 | Pending |
| META-01 | Phase 15 | Pending |
| META-02 | Phase 15 | Pending |
| META-03 | Phase 15 | Pending |
| META-04 | Phase 15 | Pending |
| API-01 | Phase 16 | Pending |
| API-02 | Phase 16 | Pending |
| API-03 | Phase 16 | Pending |
| API-04 | Phase 16 | Pending |
| PLAY-01 | Phase 17 | Pending |
| PLAY-02 | Phase 17 | Pending |
| PLAY-03 | Phase 17 | Pending |
| UI-01 | Phase 18 | Pending |
| UI-02 | Phase 18 | Pending |
| UI-03 | Phase 18 | Pending |
| UI-04 | Phase 18 | Pending |
| GOV-01 | Phase 19 | Pending |
| GOV-02 | Phase 19 | Pending |
| GOV-03 | Phase 19 | Pending |
| GOV-04 | Phase 19 | Pending |
| PROD-02 | Phase 20 | Pending |
| PROD-03 | Phase 20 | Pending |
| PROD-04 | Phase 20 | Pending |
| MIGR-04 | Phase 20 | Pending |

**Coverage:**
- v3 requirements: 32 total
- Mapped to phases: 32
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-25*
*Last updated: 2026-04-25 after v3 roadmap creation*
