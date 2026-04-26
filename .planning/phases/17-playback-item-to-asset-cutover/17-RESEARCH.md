# Phase 17 Research — Playback Item-to-Asset Cutover

**Phase:** 17 — Playback Item-to-Asset Cutover  
**Requirements:** PLAY-01, PLAY-02, PLAY-03  
**Date:** 2026-04-25

## Research Goal

Answer: what must change so playback resolves from a catalog item to a selected
`media_asset` + `inventory_file` pair, instead of selecting a legacy
`MediaFile` from a legacy `MediaItem`, while preserving explainable decision
payloads and HLS/direct playback behavior.

## No User Context Artifact

- No phase-specific `CONTEXT.md` exists for Phase 17.
- Planning therefore uses `ROADMAP.md`, `REQUIREMENTS.md`, the quick migration
  plan, shipped catalog-kernel summaries, and current code behavior as the
  authoritative sources.

## Current Codebase Facts

### Playback still depends entirely on legacy `MediaItem` / `MediaFile`

- `mibo-media-server/internal/playback/profile.go` defines `PlaybackRequest`
  with `MediaItemID` and `PreferredFileID`.
- `internal/playback/service.go` loads `database.MediaItem` plus child
  `database.MediaFile` rows, ranks files, then resolves direct/HLS URLs from
  `/api/v1/media-files/{id}` routes.
- `internal/httpapi/handlers_playback.go` and `internal/httpapi/hls.go` are
  keyed by `mediaFileID`, not `inventory_file` or `asset` identifiers.

### Catalog + inventory data already contain the relationships Phase 17 needs

- `database.CatalogItem`, `database.MediaAsset`, `database.AssetItem`,
  `database.InventoryFile`, and `database.AssetFile` are already migrated.
- `internal/catalog/contracts.go` already exposes `CatalogAssetDetail` and
  `CatalogAssetLink`, so API contracts already acknowledge item-to-asset
  relationships.
- `internal/catalog/backfill_movies.go` shows the canonical join pattern for
  finding an asset via `asset_items` and `asset_files` instead of legacy
  `media_files`.

### Inventory-backed playback metadata is split across new tables

- Container and storage identity now live on `inventory_files`.
- Probe/runtime readiness is tracked on `media_assets.probe_status` and
  `media_assets.technical_summary_json`.
- Per-stream details are modeled in `media_streams` keyed by `file_id`
  (`inventory_files.id`), not legacy `MediaFile.id`.

### Current HLS implementation must pivot from legacy file ids to inventory ids

- `internal/httpapi/hls.go` stores artifact folders by `mediaFileID`, resolves
  input sources by loading `database.MediaFile`, and generates playlist URLs
  under `/api/v1/media-files/{id}/hls/...`.
- Phase 17 must move those paths and artifact keys to `inventory_files.id` so
  HLS and direct streaming share the same new-kernel file identity.

### Existing explainable-playback semantics are worth preserving

- `internal/playback/service.go` already returns `PlaybackDecision` with
  `kind`, `selected_by`, `fallback_kind`, and structured `DecisionReason`
  entries.
- `internal/playback/service_test.go` and `internal/httpapi/router_test.go`
  already prove that fallback and unplayable outcomes should stay `200 OK` with
  a decision payload rather than becoming a transport error.

## Existing Patterns To Reuse

### 1. Keep ranking and explainability in `internal/playback`

- Asset selection is business logic, not an HTTP concern.
- The current `selectPlaybackFile(...)` and `assessDirectPlay(...)` helpers are
  the right shape; only their input model needs to move from `MediaFile` to a
  catalog asset + inventory file candidate.

### 2. Keep storage resolution behind provider registry lookups

- Current direct stream and HLS paths both resolve a provider through the
  library -> media source -> `providers.Registry` chain.
- Phase 17 should preserve that boundary; only the database lookup changes from
  legacy file rows to `inventory_files`.

### 3. Use focused router tests instead of growing the giant shared router test

- Existing router tests prove auth and payload behavior, but `router_test.go` is
  already large.
- New focused playback router test files are the cleaner extension point for
  Phase 17.

### 4. Preserve 200-level decision payloads for bad playback states

- Missing files, unsupported formats, and unavailable links should still return
  a non-playable `PlaybackSource` / asset-link payload with structured reasons,
  not a `500`.
- Direct stream and HLS artifact endpoints may still fail when called directly,
  but the top-level playback decision route should absorb these states into a
  clear unplayable response.

## Recommended Phase-17 Implementation Shape

1. **Cut the playback service contracts over first.**
   - Replace legacy request fields with `item_id` + optional `asset_id`.
   - Replace legacy response identifiers with `item_id`, `asset_id`, and
     `inventory_file_id`.

2. **Teach playback selection to query catalog asset candidates.**
   - Load the requested `catalog_items` row.
   - Join `asset_items`, `media_assets`, `asset_files`, and `inventory_files`
     to find playable candidates for that item.
   - When `asset_id` is supplied, validate that it belongs to the requested item
     and resolves to an available source file.

3. **Reuse direct-play decision logic against inventory-backed metadata.**
   - Use `inventory_files.container`, `media_assets.probe_status`,
     `media_assets.technical_summary_json`, and `media_streams` to derive the
     same decision surface currently built from legacy `MediaFile` columns.

4. **Add new catalog playback routes before frontend migration.**
   - `GET /api/v1/items/{id}/playback`
   - `GET /api/v1/assets/{id}/link`
   - Return absolute URLs via the existing `buildPlaybackURL(...)` helper.

5. **Move direct stream and HLS endpoints to inventory-file ids.**
   - `GET /api/v1/inventory-files/{id}/stream`
   - `GET /api/v1/inventory-files/{id}/hls/index.m3u8`
   - `GET /api/v1/inventory-files/{id}/hls/{name}`

6. **Keep missing files explainable.**
   - If the selected asset resolves to a missing or unreadable inventory file,
     the item-playback route should return `decision.kind="unplayable"` with a
     concrete reason code instead of surfacing the lower-level storage error as
     a transport failure.

## Required Mapping Decisions

### Request contract

Recommended `PlaybackRequest` shape:

- `ItemID uint`
- `AssetID uint` (optional explicit version selection)
- `ClientProfile`
- `AllowHLSFallback bool`

`PreferredFileID` should be removed from new-kernel playback code.

### Response contract

Recommended `PlaybackSource` additions/replacements:

- Replace `media_item_id` with `item_id`
- Replace `media_file_id` with `inventory_file_id`
- Add `asset_id`
- Keep `decision`, `checks`, `playable`, `url`, `direct`
- Keep asset-facing descriptive fields such as `asset_type`, `edition`, and
  `quality_label` so explicit version choice is inspectable

### Asset selection order

Recommended ranking:

1. Explicit `asset_id` if linked to the requested item and still available
2. Otherwise available `asset_items.role="primary"`
3. Otherwise available `role="version"`
4. Within the same role: direct-play-compatible > probe-ready > higher declared
   quality/resolution > lower asset id for deterministic tie-breaking

### Source file resolution

- Resolve playback input files through `asset_files(role="source", part_index=0)`.
- Use `inventory_files.id` as the canonical file identity for stream and HLS
  routes.
- Do not reintroduce any new read dependency on legacy `database.MediaFile`.

### Explainable failure codes

Phase 17 should surface concrete reason codes such as:

- `asset_not_linked_to_item`
- `asset_unavailable`
- `no_available_assets`
- `inventory_file_missing`
- `inventory_file_directory`
- `no_supported_playback_path`

## Main Risks

1. **Legacy-query regression risk** — touching only HTTP routes without changing
   playback service queries would leave hidden `MediaFile` dependencies.
2. **Version-selection ambiguity** — without a deterministic ranking, repeated
   runs could choose different assets for the same item.
3. **File-identity drift** — HLS and direct routes must share `inventory_file`
   identity, or cached artifacts and playback links will diverge.
4. **Availability masking** — storage failures must collapse into explicit
   unplayable decisions on the top-level item route, not 500s.
5. **Boundary leakage** — HTTP handlers must not absorb asset-selection logic
   that belongs in `internal/playback`.

## Validation Architecture

### Fast feedback

- `cd mibo-media-server && go test ./internal/playback -run 'Test(CatalogPlayback|PlaybackDecision)' -count=1`
- `cd mibo-media-server && go test ./internal/httpapi -run 'Test(CatalogPlayback|AssetLink|InventoryPlayback)' -count=1`

### Integration feedback

- `cd mibo-media-server && go test ./internal/httpapi -run 'Test(CatalogPlayback|AssetLink|InventoryPlayback|HLS)' -count=1`

### Full phase regression

- `cd mibo-media-server && go test ./internal/playback ./internal/httpapi -count=1`

### Required proof points

- Playback service can select a default asset for a catalog item.
- Explicit `asset_id` playback selects that version and rejects unrelated
  assets.
- Item playback returns inventory-backed direct or HLS URLs.
- Asset link and stream/HLS routes are keyed by `inventory_files.id`.
- Missing files return a structured unplayable decision from the item-playback
  route.

## Architectural Responsibility Map

| Concern | Correct layer | Why |
|---------|---------------|-----|
| Asset ranking and explainable decisions | `internal/playback` | playback semantics already live here and should remain reusable outside HTTP |
| Storage provider lookup | `internal/playback` + `internal/httpapi/hls.go` | both direct and HLS need registry-backed source resolution |
| Route auth/query parsing | `internal/httpapi/handlers_playback.go` | handlers should stay thin and authenticated |
| Inventory-file stream/HLS serving | `internal/httpapi` | these are transport endpoints, not domain-selection logic |
| Catalog asset joins | `internal/playback` using GORM queries | service owns candidate resolution from new kernel tables |

## Planning Implications

- Phase 17 is backend-only.
- The safest breakdown is three sequential plans: playback service cutover,
  catalog playback HTTP routes, then inventory-file stream/HLS cutover.
- Planning should assume Phase 14/13 supply `asset_items`, `asset_files`, and
  `inventory_files`; Phase 17 consumes that data and turns it into runnable
  playback behavior.
