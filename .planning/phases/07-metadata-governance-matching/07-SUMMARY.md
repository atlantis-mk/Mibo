---
one_liner: Admin metadata governance shipped with a dedicated management page, draft-based editing, candidate preview/apply, and separate rematch/refetch actions backed by the existing jobs workflow.
key_files:
  modified:
    - mibo-media-server/internal/httpapi/router.go
    - mibo-media-server/internal/metadata/service.go
    - mibo-media-server/internal/database/models.go
    - web/src/lib/mibo-api.ts
    - web/src/features/media/index.tsx
  created:
    - web/src/features/metadata-governance/
decisions:
  - Metadata governance remains app-owned in Mibo rather than delegating edits to OpenList.
  - Admin edits use a unified draft save flow instead of field-by-field auto-save.
  - Candidate apply, rematch, and metadata refetch remain separate actions with preview and job-backed feedback.
---

# Phase 7: Metadata Governance & Matching - Summary

## Outcome

Completed Phase 7 end to end: admins can now enter a dedicated metadata governance workflow, edit app-owned metadata through a unified draft session, preview candidate differences before apply, and run rematch or metadata refetch as distinct background actions without leaving the Mibo product boundary.

## Accomplishments

- Added the backend governance contract for manual metadata save/update flows under the existing `/api/v1/media-items/{id}/...` family.
- Extended metadata services so semi-automatic metadata updates and dedicated metadata refetch stay on the existing jobs/worker path instead of becoming ad hoc synchronous actions.
- Shipped a dedicated governance screen rather than overloading the regular media detail page with full editing behavior.
- Added two admin entry paths into governance: from media detail and from the admin/global navigation surface.
- Implemented draft-based editing with unsaved-change protection so multi-field metadata changes can be reviewed and saved together.
- Implemented candidate search/apply UX with a preview step before any metadata candidate overwrites current values.
- Added clear rematch and metadata refetch feedback states so admins can distinguish queued/processing/completed outcomes.

## User-Facing Changes

- Admins can open a dedicated metadata governance page for a media item.
- Admins can edit title, original title, year, overview, artwork, and semi-automatic metadata from one managed workflow.
- Leaving with unsaved changes triggers a confirmation instead of silently discarding edits.
- Applying a metadata candidate shows a current-vs-candidate preview before confirmation.
- Rematch and metadata refetch are exposed as separate actions with understandable status feedback.
- Governance can be reached both from media detail and from the admin/global area.

## Validation

- Automated verification passed per `07-VALIDATION.md`:
  - `go test ./internal/httpapi -run 'Test.*Metadata'`
  - `go test ./internal/metadata -run 'Test(ListTV|ListSeason|Match)'`
  - `go test ./...`
  - `pnpm typecheck`
  - `pnpm build`
- Manual verification completed:
  - unsaved draft leave guard
  - candidate diff/preview before apply
  - understandable async rematch/refetch feedback
  - both governance entry points reachable

## Assumptions

- Phase 7 stayed focused on single-item governance and did not expand into bulk editing or a full editorial CMS.
- Structured cast/season/episode governance remains semi-automatic rather than becoming a fully manual content-ops surface.
- Artwork selection continues to prefer provider candidates over manual URL entry or uploads.

## Follow-On Impact

- Phase 7 establishes Mibo-owned metadata as the quality foundation for Phase 8 native search and discovery filters.
- The dedicated governance contract and persisted edits reduce the risk that later discovery features operate on stale or low-confidence metadata.
