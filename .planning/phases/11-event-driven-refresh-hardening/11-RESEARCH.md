# Phase 11: Event-Driven Refresh Hardening - Research

**Researched:** 2026-04-24
**Status:** Ready for planning

## Research Question

What does Mibo need so Phase 11 can harden storage-event refresh behavior into conservative, coalesced, listener-driven work with periodic reconciliation, while preserving the existing `OpenList -> mibo-media-server -> jobs -> worker` boundary?

## Key Findings

### 1. The current storage-event endpoint has the right boundary but not the required hardening
- `mibo-media-server/internal/httpapi/handlers_storage_events.go` already validates auth, checks path escape, normalizes `create/update/delete/move`, and queues `targeted_refresh` or `sync_library`.
- That matches D-01, D-02, and D-03 at a basic level, but it is still one-event-in -> one-job-out.
- Gap: LIST-03 is not satisfied because there is no explicit debounce or coalescing window; current behavior only relies on job uniqueness for exact keys.

### 2. Mibo already has durable primitives for coalescing without adding external infrastructure
- `mibo-media-server/internal/jobs/service.go` already persists `job_key`, `status`, and `available_at`, which is enough to model a short coalescing window server-side.
- A listener-focused domain service can safely use the jobs table as the persistence substrate for delayed/coalesced listener work.
- This satisfies D-04 and D-05 without adding Redis, Kafka, NATS, a side queue, or a separate listener inbox subsystem.

### 3. Coalescing should produce an intermediate listener job, not direct canonical mutations
- The canonical rule from CONTEXT.md and existing scan behavior is still correct: listener input is only a hint that narrows refresh scope.
- The safest shape is a new internal listener job kind that waits through a debounce window, then enqueues either `targeted_refresh` or `sync_library`.
- This keeps D-03, D-07, D-08, and D-09 intact because scan and reconciliation logic remain the only places that settle media state.

### 4. Existing scan and reconciliation semantics already support the required conservative product behavior
- `library.RunTargetedRefresh` already performs subtree-scoped partial scan plus scoped cleanup and search reindex.
- `scanLibraryWithMode(...partial=true...)` only cleans up rows inside the scoped root, which naturally supports D-06 and D-08: stay local first, enlarge only when needed.
- `ReconcileProvisionalMediaFile` already proves Mibo has conservative fallback reconciliation behavior for ambiguous rename/move cases.
- Therefore Phase 11 should not invent a listener-specific media state machine.

### 5. Periodic reconciliation should remain worker-owned and library-scoped
- LIST-04 and D-10 through D-12 require a fallback path even when listener events are missed.
- Phase 10 already established that recurring background work belongs on the existing jobs/worker path.
- The lightest-weight fit for Phase 11 is one future-dated `listener_reconcile` job per active library, with a fixed default cadence chosen by the agent's discretion.
- Recommended default cadence: **every 6 hours per active library**. This is frequent enough to reduce drift while remaining conservative about storage load.

### 6. The coalescing policy should be concrete, conservative, and small-surface
- Recommended debounce window: **15 seconds** per D-04 and the agent's discretion.
- Primary merge key: `library_id + normalized_root` per D-05.
- Merge rule for nested or sibling paths: promote to the minimal safe common ancestor inside the library root per D-06.
- Fallback rule: if a move/rename or unsupported event cannot be normalized safely, store a full-sync listener intent instead of widening path guesses.

## Recommended Architecture

### Backend
1. Add a dedicated `internal/listener` package inside `mibo-media-server` per D-13.
2. Let the listener package own:
   - storage-event intent normalization after path validation
   - 15-second debounce and path coalescing
   - promotion to common ancestors when nested paths burst together
   - reconcile job seeding for active libraries every 6 hours
3. Add two internal job kinds:
   - `apply_storage_event_refresh`
   - `listener_reconcile`
4. Keep `/api/v1/storage-events` as the ingress contract, but have it call the listener service instead of queueing scan jobs directly.
5. Extend `worker.Runner` to handle the two listener job kinds by converting them into existing `targeted_refresh` / `sync_library` work and by reseeding the next reconcile job.

### Why this fits the codebase
- It reuses the durable jobs table and worker loop.
- It keeps HTTP handlers thin.
- It does not add new external dependencies.
- It preserves scan-owned canonical row mutation semantics.

## Data / Contract Recommendations

### Listener service contracts
- `EventIngestInput`
  - `library_id`
  - `kind`
  - `path`
  - `old_path`
- `RefreshIntent`
  - `library_id`
  - `root_path`
  - `fallback_full_sync`
  - `reason` (`storage_event`)
  - `window_started_at`
  - `window_ends_at`
- `ReconcileIntent`
  - `library_id`
  - `reason` (`listener_reconcile`)
  - `scheduled_for`

### Concrete policy choices
- Debounce window: `15s`
- Reconciliation interval: `6h`
- Merge target: minimal safe common ancestor within library root
- Fallback: full sync when the root cannot be normalized safely

## Standard Stack

### Backend
- Go `net/http`
- GORM
- existing `jobs.Service`
- existing `worker.Runner`
- existing `library.Service` scan and targeted refresh pipeline

### Additions
- **No new external packages required**
- New internal package only: `mibo-media-server/internal/listener`

## Established Patterns To Reuse

### Reuse, do not hand-roll
- strict JSON decode and auth checks in `internal/httpapi`
- durable queueing through `jobs.Service`
- worker-owned async execution through `worker.Runner`
- subtree-safe partial scan behavior in `RunTargetedRefresh`
- existing fallback reconciliation behavior in `scan_reconcile.go`

### Do not hand-roll
- direct `media_items` / `media_files` mutation from listener ingress
- an external event bus or message queue
- a second scheduler separate from the worker loop
- aggressive delete-on-ingest semantics

## Common Pitfalls

1. **Treating listener events as truth**
   - Violates D-03 and risks incorrect deletes or duplicate rows.

2. **Using only current job uniqueness**
   - Violates D-04 because exact-key uniqueness cannot merge adjacent or nested paths over time.

3. **Choosing roots too aggressively**
   - Violates D-06 and D-08 if sibling bursts jump straight to full-library refresh.

4. **Deleting or marking rows missing during event intake**
   - Violates D-07 and bypasses the existing conservative scan/reconcile model.

5. **Making reconciliation optional**
   - Violates LIST-04 and D-10 through D-12.

## Security / Trust Boundaries

1. **storage notifier / authenticated caller -> `/api/v1/storage-events`**
   - Treat all paths and kinds as untrusted.
   - Validate library scope before producing refresh work.

2. **listener service -> jobs table**
   - Build payloads server-side only.
   - Merge only within the same library boundary.

3. **worker -> storage provider scans**
   - Coalescing mistakes can amplify into unnecessary scan load, so promotion and fallback rules must be deterministic.

## Architectural Responsibility Map

| Concern | Owns It |
|---|---|
| Event coalescing, debounce timing, reconcile cadence | `mibo-media-server/internal/listener` |
| Path validation and HTTP auth | `mibo-media-server/internal/httpapi` |
| Durable queue rows and status lifecycle | `mibo-media-server/internal/jobs` |
| Targeted refresh and full sync execution | `mibo-media-server/internal/library` |
| Worker dispatch of listener job kinds | `mibo-media-server/internal/worker` |
| Storage namespace / object listing only | `OpenList` or local provider via `storage.Provider` |

## Testing Guidance

### Existing coverage worth extending
- `mibo-media-server/internal/httpapi/router_test.go` for storage-event route behavior.
- `mibo-media-server/internal/worker/worker_test.go` for queued job execution and targeted refresh behavior.
- new `mibo-media-server/internal/listener/service_test.go` for merge-window, ancestor-promotion, and reconcile scheduling logic.

### Best Phase 11 verification strategy
- Prove debounce and path promotion at the listener-service level first.
- Prove the HTTP route returns accepted listener jobs instead of direct canonical mutation.
- Prove the worker eventually turns coalesced listener jobs into real `targeted_refresh` / `sync_library` jobs.
- Prove a due reconciliation job queues a library resync and reseeds the next reconcile window.

## Validation Architecture

### Quick feedback loop
- Backend quick: `cd /root/Mibo/mibo-media-server && go test ./internal/listener ./internal/httpapi ./internal/worker -run 'Test.*(StorageEvent|Listener|TargetedRefresh|Reconcile)'`

### Full phase validation loop
- Backend full: `cd /root/Mibo/mibo-media-server && go test ./...`

### Wave 0 needs
- No new framework is required. Existing Go test infrastructure is sufficient.

## Planning Constraints

1. Honor D-01 through D-14 exactly.
2. Exclude deferred items:
   - listener health dashboard / LIST-05
   - external message queues or event buses
   - real-time UI push
   - provider-specific watcher platform expansion beyond the current storage-event boundary
3. Keep all listener policy in `mibo-media-server`, never in `OpenList`.
4. Keep scan and reconciliation as the only canonical-state settling paths.
5. Reuse the existing jobs/worker model for both coalesced refresh and periodic reconciliation.

## Recommendation

Phase 11 should be planned as a backend-only hardening slice with three steps: listener-domain coalescing foundation -> HTTP ingress integration -> worker execution plus periodic reconciliation. That sequence delivers LIST-01 through LIST-04 without introducing new infrastructure or weakening the conservative scan/reconcile model.
