# Phase 19 Research — Metadata Governance UI Rebuild

**Date:** 2026-04-25
**Phase:** 19 — Metadata Governance UI Rebuild
**Requirements:** GOV-01, GOV-02, GOV-03, GOV-04

## Research Goal

Answer what must exist so Phase 19 can rebuild the governance UI around the
catalog kernel instead of the legacy `MediaItem` metadata editor.

## Source Artifacts Read

- `.planning/PROJECT.md`
- `.planning/ROADMAP.md`
- `.planning/REQUIREMENTS.md`
- `.planning/STATE.md`
- `.planning/RETROSPECTIVE.md`
- `.planning/quick/260425-tvg-catalog-kernel-remaining-plan/260425-tvg-PLAN.md`
- `.planning/quick/260425-tvg-catalog-kernel-remaining-plan/260425-tvg-RESEARCH.md`
- `.planning/phases/15-series-level-metadata-governance-engine/15-RESEARCH.md`
- `.planning/phases/16-catalog-api-search-progress-cutover/16-RESEARCH.md`
- `.planning/phases/17-playback-item-to-asset-cutover/17-RESEARCH.md`
- `web/src/lib/mibo-api.ts`
- `web/src/lib/mibo-query.ts`
- `web/src/features/metadata-governance/index.tsx`
- `web/src/features/metadata-governance/workspace.tsx`
- `web/src/features/metadata-governance/detail.tsx`
- `web/src/features/metadata-governance/detail-sections.tsx`
- `web/src/features/metadata-governance/detail-panels.tsx`
- `web/src/features/metadata-governance/formatters.ts`
- `web/src/features/settings/pages.tsx`
- `web/src/features/media/index.tsx`
- `web/src/routes/settings.metadata.index.tsx`
- `web/src/routes/settings.metadata.$id.tsx`
- `mibo-media-server/internal/catalog/contracts.go`

## Phase Goal And Requirement Mapping

**Goal:** rebuild governance UI around catalog field locks, source evidence,
image candidates, external IDs, and asset links.

**Requirements:**

- `GOV-01` — administrator can view and edit canonical catalog fields with
  visible source, confidence, lock status, and edit metadata.
- `GOV-02` — administrator can inspect metadata source evidence and provider
  payload summaries without losing canonical provenance.
- `GOV-03` — administrator can select poster, backdrop, logo, still, and other
  image candidates per catalog item without deleting alternatives.
- `GOV-04` — administrator can inspect and repair item-to-asset links so
  playability issues are explainable and actionable.

## Current Codebase Facts

### The shipped governance UI is still Phase-7 legacy

- `web/src/features/metadata-governance/workspace.tsx` uses
  `createAuthedMiboApi(token).latestByLibrary()` and renders recent legacy media
  cards, not governance workspace data.
- `web/src/features/metadata-governance/detail.tsx` loads
  `getMediaItem(mediaItemId)` and mutates legacy metadata through
  `updateMediaItemMetadata`, `searchMediaItemMetadata`,
  `applyMediaItemMetadataCandidate`, `rematchMediaItem`, and
  `refetchMediaItemMetadata`.
- The current detail page is a whole-item draft editor for `title`,
  `original_title`, `year`, `overview`, `poster_url`, and `backdrop_url`; it
  does not render `field_states`, `source_evidence`, catalog image candidates,
  or asset links.
- `formatters.ts` still maps the legacy `show` type and legacy `match_status`
  vocabulary.

### Backend catalog contracts already expose most of the read surface

- `mibo-media-server/internal/catalog/contracts.go` already defines:
  - `CatalogGovernanceWorkspace`
  - `CatalogFieldState`
  - `CatalogSourceEvidence`
  - `CatalogSelectedImage`
  - `CatalogAssetDetail`
  - `CatalogAssetLink`
- Those types already carry the provenance-critical fields Phase 19 needs for
  `GOV-01` and `GOV-02`: `source_id`, `value`, `is_locked`, `lock_reason`,
  `edited_by_user_id`, `edited_at`, `source_name`, `source_type`, `language`,
  `external_id`, `confidence`, and `summary`.
- `CatalogAssetDetail` already carries `asset_type`, `display_name`, `edition`,
  `quality_label`, `status`, `probe_status`, and `links`, which is the minimum
  base for `GOV-04`.

### Phase 19 depends on Phases 16, 17, and 18 shipping their contracts first

- Phase 16 research defines additive governance routes under `/api/v1/items/*`
  and `/api/v1/assets/*`; Phase 19 should consume those routes instead of
  reviving `/api/v1/media-items/*`.
- Phase 17 research defines the new asset/version and inventory-file playback
  semantics that Phase 19 must explain in the asset-link panel.
- Phase 18 is the frontend catalog-item migration. Phase 19 should assume the
  app is already navigating with catalog item ids by the time this phase is
  executed.

## Dependency Gates

### Gate 1 — field-level mutation contract must exist before execution

The current Phase 16 research explicitly names field-lock mutation, image
selection, match/refresh, and asset-link APIs, but it does not yet name a field
value mutation route. Phase 19 cannot satisfy `GOV-01` unless the shipped Phase
16 governance contract exposes either:

1. a field-state mutation route that accepts `field_key` + `value` + lock data,
   or
2. a pair of field-value and field-lock routes that the frontend can compose.

**Planning implication:** Phase 19 plans should stabilize the frontend helper
shape as `updateCatalogFieldState(itemId, { field_key, value, is_locked,
lock_reason })`, then adapt it to the actual Phase 16 route names during
execution.

### Gate 2 — full image candidate lists must be exposed, not only selected images

`CatalogGovernanceWorkspace` currently includes `selected_images`, but `GOV-03`
requires selection among alternatives. That means the shipped Phase 16/19 API
contract must include a full image-candidate list (field name can vary) in
addition to the selected subset.

**Planning implication:** Phase 19 should normalize the raw API payload into a
stable frontend `imageCandidatesByType` shape so UI components do not depend on
the backend field name.

### Gate 3 — asset explainability must stay contract-safe

Phase 19 needs to explain why an item is playable, missing, or broken, but it
must not leak raw storage-provider payloads or internal file-system details.
The asset panel should stay bounded to the catalog/playback contracts from
Phases 16-17.

## Architectural Responsibility Map

| Concern | Correct layer | Why |
|---------|---------------|-----|
| Raw HTTP path strings, payload types, and request methods | `web/src/lib/mibo-api.ts` | frontend/server contract stays in one typed boundary |
| Query keys, cache invalidation, optimistic refresh hooks | `web/src/lib/mibo-query.ts` + feature hooks | existing React Query pattern lives here |
| Governance workspace and detail orchestration | `web/src/features/metadata-governance/*` | feature-private state and UI belong together |
| Route param parsing | `web/src/routes/settings.metadata*.tsx` | route files stay thin |
| Entry links from settings/detail surfaces | `web/src/features/settings/pages.tsx`, `web/src/features/media/index.tsx` | those files already own the navigation affordances |

## Recommended Implementation Strategy

### 1. Freeze a catalog-native frontend contract first

- Stop adding new governance behavior on top of `MediaItemDetail`.
- Add catalog governance types and mutation helpers to `mibo-api.ts`.
- Add feature hooks that normalize the raw workspace payload into UI-friendly
  slices:
  - ordered field-state rows
  - grouped source evidence
  - grouped image candidates
  - flattened asset-link rows

This keeps the rebuild from scattering raw contract knowledge across panels.

### 2. Rebuild the workspace as a real governance entry, not a “recent items” rail

- Replace `latestByLibrary()` with catalog governance list data.
- Show governance status, availability, selected poster, external identity, and
  child summary counts on each card.
- Preserve the `/settings/metadata` and `/settings/metadata/$id` URLs, but treat
  `$id` as a catalog item id.

### 3. Split the detail page by governance concern

The current single-file draft editor is the wrong shape for Phase 19. The detail
page should be decomposed into focused panels:

- field state / canonical edits / locks
- source evidence
- image candidate selection
- asset link explainability + repair actions
- existing match / refresh actions

This follows the repository’s feature-private component pattern and prevents one
monolithic governance page from growing unbounded.

### 4. Render evidence and payloads as plain text / JSON only

- `CatalogSourceEvidence.summary` is untrusted provider-derived content.
- Do not render HTML from evidence summaries.
- Show summary scalars in cards and raw JSON only inside a collapsed `<pre>`.

### 5. Keep image and asset operations item-scoped

- Selecting an image should only toggle which candidate is selected; it must not
  delete alternatives.
- Linking/unlinking assets should mutate one `asset_items` edge at a time and
  immediately refetch the governance workspace.

## Constraints And Pitfalls

1. **No raw `fetch` inside the feature.** Existing project rules and
   `mibo-codegen-structure` both require frontend/server calls to stay in
   `mibo-api.ts` and React Query helpers.
2. **Do not keep using legacy `mediaItemId` naming.** This phase should use
   catalog `itemId` semantics end-to-end.
3. **Do not edit `web/src/routeTree.gen.ts` manually.** Route files remain the
   source of truth.
4. **Do not delete image candidates in the first pass.** `GOV-03` is selection,
   not destructive media cleanup.
5. **Do not surface raw storage-provider details in asset explanations.** Show
   contract-safe status, probe, edition, quality, and link-role data only.
6. **Frontend tests are not established in this repo.** Verification should rely
   on `pnpm typecheck`, `pnpm build`, and targeted human smoke checks at phase
   verification time.

## Recommended Plan Split

Four execute plans keep the work under context budget while preserving file
ownership:

1. **19-01** — catalog governance frontend contracts, query keys, and hooks.
2. **19-02** — workspace route rebuild and app entry/navigation updates.
3. **19-03** — detail page field-state and source-evidence panels.
4. **19-04** — image-candidate and asset-link explainability panels.

This split keeps the phase interface-first: data contracts and hooks first,
workspace entry second, then detail panels in two focused slices.

## Validation Architecture

### Test Infrastructure

| Property | Value |
|----------|-------|
| Framework | frontend typecheck + production build |
| Config file | `web/tsconfig.json`, `web/vite.config.ts` |
| Quick run command | `cd web && pnpm typecheck` |
| Full suite command | `cd web && pnpm typecheck && pnpm build` |
| Estimated runtime | ~40-60 seconds |

### Sampling Strategy

- After every task commit: run the task’s `pnpm typecheck` command.
- After every plan wave: run `cd web && pnpm typecheck && pnpm build`.
- Before `/gsd-verify-work`: production build must be green.

### Required Proof Points

- Governance workspace lists catalog items, not legacy `latestByLibrary` rows.
- Governance detail shows field states with provenance and lock metadata.
- Evidence summaries are inspectable without raw HTML rendering.
- Image selection updates the selected candidate without deleting alternatives.
- Asset links explain playability using catalog/asset data and permit targeted
  repair actions.

## Research Conclusion

Phase 19 is a **frontend rebuild, not a new dependency phase**. The repo already
has the libraries and UI primitives it needs. The main requirement is to stop
building on the legacy `MediaItem` governance screen and instead consume the
catalog-governance contracts introduced by Phases 16-18.

The critical planning constraint is dependency fidelity: by execution time,
Phase 16 must expose field-state mutation and full image candidate data, and
Phase 17 must have stabilized asset/playability semantics. With those contracts
in place, the Phase 19 work is cleanly decomposable into one contract/hook plan,
one workspace plan, and two detail-panel plans.
