## Why

`tvg-catalog-kernel-remaining` has already moved Mibo onto a usable catalog-kernel path, but a comparison against `.planning/TV-SERIES-METADATA-GOVERNANCE-PLAN.md` shows that the product still stops short of the intended TV-first operating model.

The current codebase now has real `catalog_items`, `media_assets`, governance workspaces, and catalog playback. However, several gaps remain between the implemented cutover and the TV-governance target:

- frontend media surfaces still normalize catalog DTOs back into legacy `MediaItem`-shaped view models
- TV convenience APIs are still incomplete for real hierarchy-driven operations such as child listing, missing episodes, and next-up behavior
- governance now exposes linked assets and image selection, but asset-link correction and hierarchy-conflict review are still shallow
- metadata matching at the series root exists, but descendant season/episode identity, evidence, and artwork coverage still needs to be made complete and consistent
- production hardening is incomplete: legacy routes remain, catalog reads are not yet the unconditional default, and database constraints plus cleanup gates are not fully enforced

The greenfield TV plan remains directionally correct, but most of its core schema ideas have already landed in some form. The next change should therefore harden and complete the existing incremental catalog design rather than restart the media model from scratch.

## What Changes

- Finish the TV-first catalog behavior that was left partial after the main cutover, especially hierarchy-native APIs and descendant metadata completeness.
- Remove remaining frontend dependence on legacy `MediaItem` presentation adapters for catalog-backed flows.
- Strengthen governance workflows so linked assets, hierarchy mismatches, and image/evidence decisions are actionable rather than mostly informational.
- Add the missing operational and database hardening needed to safely enable catalog reads by default and retire legacy routes.

## Capabilities

### New Capabilities
- `tv-hierarchy-metadata-completion`: Define the remaining TV hierarchy, descendant metadata, and convenience API behavior needed to complete the catalog-backed series model.
- `catalog-frontend-contract-cleanup`: Define the frontend migration from legacy-shaped presentation adapters to catalog-native item, hierarchy, and asset contracts.
- `catalog-governance-actions`: Define the governance workflows required for actionable asset-link correction, hierarchy mismatch review, and descendant governance behavior.
- `catalog-cutover-hardening`: Define the final constraint, validation, default-read, and legacy-retirement requirements needed for a safe production cutover.

### Modified Capabilities
- None.

## Scope Compared To The TV Plan

This follow-up change adopts the TV plan's intent, but narrows it to the gaps that still exist in the current repository:

- keep the existing catalog kernel and asset schema as the foundation
- do not replace the catalog model with a new greenfield schema
- do not redesign OpenList or storage-provider boundaries
- focus on convergence between the current implementation and the TV-first governance/product behavior described in the planning document

## Impact

- Backend packages: `mibo-media-server/internal/catalog`, `internal/httpapi`, `internal/metadata`, `internal/playback`, `internal/progress`, `internal/database`, and `internal/settings`
- Frontend packages: `web/src/lib`, `web/src/features/media`, `web/src/features/search`, `web/src/features/library`, `web/src/features/metadata-governance`, and related routes
- Operational behavior: catalog-read defaults, legacy endpoint retirement, consistency checks, projection rebuilds, and migration safety
