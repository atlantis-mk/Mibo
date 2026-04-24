# Phase 7: Metadata Governance & Matching - Research

**Researched:** 2026-04-24
**Status:** Ready for UI contract gate

## Research Question

What does Mibo need so Phase 7 can add admin-owned metadata governance and matching without breaking the existing `OpenList -> mibo-media-server -> web` boundary or the jobs/worker model?

## Key Findings

### 1. Current backend already has half of the matching surface
- `mibo-media-server/internal/httpapi/router.go` already exposes:
  - `POST /api/v1/media-items/{id}/metadata/search`
  - `POST /api/v1/media-items/{id}/metadata/apply`
  - `POST /api/v1/media-items/{id}/match`
- `mibo-media-server/internal/metadata/service.go` already supports:
  - TMDB candidate search (`SearchCandidates`)
  - candidate apply (`ApplyCandidate`)
  - TMDB season and episode lookup/cache (`ListTVSeasons`, `ListSeasonEpisodes`)
  - auto match (`MatchItem`)
- Gap: there is **no** manual metadata edit API, no draft-save workflow, and no dedicated metadata refetch action yet.

### 2. Metadata ownership already lives in Mibo tables
- `mibo-media-server/internal/database/models.go` stores editable metadata on `MediaItem`:
  `title`, `original_title`, `overview`, `poster_url`, `logo_url`, `backdrop_url`, `genres_json`, `cast_json`, `directors_json`, `year`, `release_date`, `runtime_seconds`, `season_number`, `episode_number`, `external_id`, `metadata_provider`, `metadata_confidence`.
- This matches the phase boundary: governance should mutate app-owned data in `MediaItem`, not push changes into OpenList.

### 3. Detail responses already carry enough read data for a governance screen seed
- `mibo-media-server/internal/library/query.go` returns `MediaItemDetail` with genres, cast, directors, files, `series_tmdb_id`, and `default_season_number`.
- `web/src/features/media/index.tsx` already loads detail and invalidates React Query caches after rematch.
- `web/src/features/media/components/standalone-media-detail.tsx` already renders poster, backdrop, overview, progress, and a rematch CTA; this is the natural source for a “manage metadata” entry point per D-02.

### 4. Long-running operations must stay job-backed
- `router.handleQueueMediaItemMatch` already enqueues through `library.QueueMediaItemMatch(..., true)` and returns `202 Accepted`-style job payload.
- Context decision D-10 requires rematch and metadata refetch to be background-task style with queued / processing status and post-completion refresh.
- Therefore metadata refetch should follow the same async job pattern, not a synchronous metadata HTTP mutation.

### 5. Season / episode governance should be semi-automatic, not a full editor
- D-06 and D-07 explicitly reject a heavy editorial backend for cast and episodic content.
- Existing TMDB season and episode cache endpoints give enough support for:
  - previewing authoritative season/episode data,
  - letting admins selectively adopt refreshed data,
  - keeping detailed episode data sourced from refetch/rematch rather than hand-authoring every field.

### 6. Live frontend shape differs from older planning notes
- AGENTS.md and live code show the active frontend is `web/`, not `web-new/`.
- The Phase 7 CONTEXT.md canonical refs still mention `web-new/...`; those should be treated as stale path hints only.
- Planning and implementation must target live files under `web/src/...`.

## Recommended Architecture

### Backend
1. Add a metadata governance read/write contract under the existing `/api/v1/media-items/{id}/...` family.
2. Keep manual field edits and candidate-application as separate concerns:
   - **draft/manual save API** for owned fields
   - **candidate search/apply API** for TMDB replacement flow
   - **rematch job API** for full re-match
   - **metadata refetch job API** for provider refresh without collapsing it into rematch
3. Persist manual edits on `MediaItem` and only expand schema where Phase 7 requires new persisted distinctions.
4. Return updated `MediaItemDetail`-compatible payloads so the existing detail query invalidation path can refresh both detail and governance screens.

### Frontend
1. Create a dedicated admin governance route/page per D-01.
2. Provide two entry points per D-02:
   - detail page CTA into current item governance
   - admin/global navigation entry into governance workflow
3. Use a single draft form session per D-03, with route-leave protection per D-04.
4. Keep candidate preview/diff separate from apply confirmation per D-09.
5. Surface rematch and metadata refetch as async actions with queued/running/completed feedback per D-10.

## Data / Contract Recommendations

### Manual edit contract
The phase likely needs a dedicated payload for editable fields rather than reusing TMDB apply input:
- base fields: `title`, `original_title`, `year`, `overview`
- artwork selection: chosen `poster_url`, `backdrop_url`
- semi-automatic fields: `genres`, `cast`, episodic basics in a constrained form

### Candidate preview contract
Because D-09 requires preview before apply, search results alone are insufficient. The plan should include either:
- a comparison object produced client-side from current item + selected candidate, or
- a backend preview endpoint returning candidate-vs-current diffs.

Given current code, client-side diffing is cheaper because search results already include the key candidate fields and existing detail data is already loaded.

### Async action status
Reuse the existing jobs surface instead of inventing a second polling mechanism. Governance UI should be able to:
- capture queued job id,
- show queued/running/completed/error state,
- invalidate media detail and governance queries after completion.

## Standard Stack

### Frontend
- React 19
- TanStack Router
- TanStack React Query
- existing shadcn/radix-nova UI primitives under `web/src/components/ui`
- `sonner` for toast feedback

### Backend
- Go `net/http`
- GORM + SQLite/Postgres compatibility
- existing `metadata.Service`, `library.Service`, `jobs.Service`, `worker.Runner`

## Established Patterns To Reuse

### Reuse, do not hand-roll
- Typed frontend API surface in `web/src/lib/mibo-api.ts`
- React Query query invalidation pattern from `web/src/features/media/index.tsx`
- JSON envelope responses from `router.go`
- strict `decodeJSON` validation in HTTP handlers
- TMDB access and caching in `metadata.Service`
- async background work via `jobs.Service` and worker dispatch
- shadcn `FieldGroup` + `Field` composition for forms
- `AlertDialog` / `Dialog` / `Sheet` accessibility structure from component library

### Do not hand-roll
- raw `fetch` calls in random feature files
- ad hoc form layout with `space-y-*` wrappers instead of field primitives
- synchronous rematch/refetch requests for long-running work
- direct OpenList-side metadata mutation
- manual image URL primary flow (blocked by D-05)
- bulk editorial subsystems for cast / seasons / episodes (blocked by deferred scope and D-06/D-07)

## Common Pitfalls

1. **Stale path confusion**
   - Older docs/context mention `web-new/`; live implementation is `web/`.

2. **Conflating rematch with refetch**
   - D-08 requires four distinct actions: search candidate, apply candidate, rematch, metadata refetch.

3. **Skipping preview before overwrite**
   - Search result apply must not directly overwrite current metadata; D-09 requires explicit preview/diff first.

4. **Making the governance screen synchronous**
   - Rematch/refetch need job-backed UX with refresh after completion.

5. **Overbuilding episodic editing**
   - Phase 7 is not a full content operations CMS. Keep season/episode editing constrained to basics and provider-assisted updates.

6. **Using frontend-only dirty tracking without route protection**
   - D-04 requires leave confirmation when unsaved changes exist.

## Security / Trust Boundaries

1. **Admin browser -> metadata mutation APIs**
   - Treat all governance payloads as untrusted input.
   - Validate strings, ids, arrays, and season/episode numbers at route entry.

2. **Admin browser -> async metadata jobs**
   - Require authenticated user on every mutation / queue endpoint.
   - Prevent arbitrary job payload injection by constructing job payload server-side from validated ids.

3. **Server -> TMDB**
   - Only use configured TMDB credentials from settings/config.
   - Surface TMDB auth and rate-limit failures as actionable errors.

4. **Candidate apply -> persisted owned metadata**
   - Only allow supported candidate providers / external id formats.
   - Reject malformed external ids via existing `parseExternalID` style validation.

## Architectural Responsibility Map

| Concern | Owns It |
|---|---|
| Metadata persistence | `mibo-media-server/internal/database` + service layer |
| Candidate search/apply/rematch/refetch orchestration | `mibo-media-server/internal/metadata` + `internal/library` + jobs/worker |
| Route registration / input validation | `mibo-media-server/internal/httpapi/router.go` |
| Typed browser contract | `web/src/lib/mibo-api.ts` |
| Governance route, draft UX, preview UX | `web/src/features/...` in `web/` |
| Admin nav entry | `web/src/components/...` / route-linked navigation surface |

## Testing Guidance

### Existing coverage worth extending
- `mibo-media-server/internal/httpapi/router_test.go` already covers metadata search/apply and rematch endpoint behavior.
- `mibo-media-server/internal/metadata/service_test.go` already covers TMDB config resolution, candidate lookup errors, bearer auth, and TV season/episode cache behavior.

### Best Phase 7 verification strategy
- Backend: extend Go tests first for new mutation contracts and new async job endpoint behavior.
- Frontend: rely on typecheck/build plus targeted human verification because no established frontend test runner exists.
- End-to-end manual checks should cover:
  - draft save round-trip,
  - unsaved-leave guard,
  - candidate preview before apply,
  - rematch queued state,
  - metadata refetch queued state,
  - detail/governance screen refresh after completion.

## Validation Architecture

### Quick feedback loop
- Backend quick command: `go test ./internal/httpapi ./internal/metadata -run 'Test(ManualMetadata|Metadata|ListTV|Match)'`
- Frontend quick command: `pnpm typecheck`

### Full phase validation loop
- Backend full: `go test ./...`
- Frontend full: `pnpm build`

### Wave 0 needs
- No new test framework required; backend infrastructure already exists.
- If frontend plan tasks add significant client logic without test coverage, plans should explicitly use human-verify checkpoints after automated `pnpm typecheck` / `pnpm build`.

## Planning Constraints

1. Honor D-01 through D-10 exactly.
2. Exclude deferred items:
   - deep cast/season editorial CMS
   - image upload hosting
   - bulk metadata governance
3. Keep work inside `web/` and `mibo-media-server/` only.
4. Preserve existing system boundary: OpenList stays storage-only.
5. Prefer vertical slices, but do not mix a human checkpoint into the same plan as heavy implementation.

## Recommendation

Phase 7 should be planned only after a UI contract exists, because locked decisions D-01 to D-05 and D-09 to D-10 define major navigation, draft, preview, and async feedback behavior that materially shape the plan.
