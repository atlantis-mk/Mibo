---
phase: 11-event-driven-refresh-hardening
plan: 02
subsystem: api
tags: [go, httpapi, listener, storage-events, jobs, tdd]

requires:
  - phase: 11-event-driven-refresh-hardening/11-01
    provides: listener service with RecordStorageEvent, debounce/coalescing, and listener job payloads
provides:
  - listener-aware /api/v1/storage-events ingress
  - authenticated and library-bounded storage-event validation before listener enqueue
  - route regressions for targeted listener intent, move common ancestors, fallback full sync, auth, and path escaping
affects: [11-event-driven-refresh-hardening, worker-listener-application, storage-event-ingress]

tech-stack:
  added: []
  patterns:
    - thin HTTP handler delegates refresh intent creation to internal/listener
    - app wiring injects one listener service into HTTP routing
    - route tests assert durable listener job payloads instead of direct scan jobs

key-files:
  created:
    - .planning/phases/11-event-driven-refresh-hardening/11-02-SUMMARY.md
  modified:
    - mibo-media-server/internal/app/app.go
    - mibo-media-server/internal/httpapi/router.go
    - mibo-media-server/internal/httpapi/handlers_storage_events.go
    - mibo-media-server/internal/httpapi/router_test.go

key-decisions:
  - "Keep storage-event HTTP intake thin: authenticate, decode, validate library path scope, then delegate to listener.RecordStorageEvent."
  - "Return accepted apply_storage_event_refresh listener jobs from the API instead of direct targeted_refresh or sync_library jobs."

patterns-established:
  - "Listener ingress pattern: HTTP owns trust-boundary validation; internal/listener owns debounce, coalescing, and refresh-intent payloads."
  - "Route regression pattern: storage-event tests assert listener job kind and payload shape for conservative normalization."

requirements-completed: [LIST-01, LIST-02, LIST-03]

duration: not recorded during original execution; summary cleanup completed separately
completed: 2026-04-24
---

# Phase 11 Plan 02: Storage Event Listener Ingress Summary

**Authenticated storage-event ingress now validates library boundaries and delegates create/update/delete/move refresh intent to the listener service.**

## Performance

- **Duration:** Not recorded during original execution; summary cleanup resumed after implementation commits already existed
- **Started:** 2026-04-24T08:43:59Z
- **Completed:** 2026-04-24
- **Tasks:** 2 completed
- **Files modified:** 4 code files plus this summary

## Accomplishments

- Wired `listener.NewService(...)` through `app.New` and `httpapi.New`, adding a listener dependency to `Router`.
- Refactored `handleStorageEvent` to require auth, decode input, validate `path`/`old_path` against the library root, and call `RecordStorageEvent(...)` instead of queueing scan jobs directly.
- Expanded storage-event route regressions to prove listener job creation, auth rejection, escaping-path rejection, move common-ancestor targeting, and fallback full-sync intent.

## Task Commits

Each task was committed atomically:

1. **TDD RED for Task 1/2 route behavior** - `b48d1bb` (`test(11-02): add failing storage event listener route coverage`)
2. **Task 1: Inject listener service and delegate intake** - `9c01d16` (`feat(11-02): route storage events through listener service`)
3. **Task 2: Harden route regression coverage** - `2602f7d` (`test(11-02): harden storage event listener route regressions`)

**Plan metadata:** this summary commit.

## Files Created/Modified

- `mibo-media-server/internal/app/app.go` - Constructs one listener service and passes it to worker/router wiring.
- `mibo-media-server/internal/httpapi/router.go` - Adds the listener dependency to `Router` and accepts injected listener services through existing variadic seams.
- `mibo-media-server/internal/httpapi/handlers_storage_events.go` - Keeps HTTP trust-boundary handling thin and delegates accepted events to `listener.RecordStorageEvent`.
- `mibo-media-server/internal/httpapi/router_test.go` - Proves auth, path boundary checks, listener job response semantics, move common-ancestor normalization, and fallback behavior.
- `.planning/phases/11-event-driven-refresh-hardening/11-02-SUMMARY.md` - Records this plan completion.

## Verification

- `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi -run 'TestStorageEvent'` — PASS
- Confirmed `handlers_storage_events.go` contains `RecordStorageEvent(` and does not call `QueueTargetedRefresh(` directly.
- Confirmed route coverage includes `TestStorageEventEndpointMoveUsesCommonAncestorIntent`, `401` missing-auth assertions, `400` escaping-path assertions, and listener job payload assertions.

## Decisions Made

- Kept the HTTP handler responsible only for authentication, JSON decoding, library/source path validation, and response mapping.
- Kept refresh-intent creation inside `internal/listener` so debounce/coalescing semantics remain centralized.
- Preserved the existing `httpapi.New(..., args ...any)` optional injection pattern to avoid broad constructor churn in tests and older call sites.

## Deviations from Plan

None - plan executed as written. The work stayed inside the planned app/router/handler/test files and did not change worker or frontend files.

## Issues Encountered

None in the committed implementation.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None.

## Threat Flags

None beyond the plan threat model. The storage-event trust boundary was already identified, and this plan preserved auth plus library path validation before listener enqueue.

## TDD Gate Compliance

- RED gate: `b48d1bb` added failing route coverage for listener-oriented storage-event behavior.
- GREEN gate: `9c01d16` routed storage-event intake through the listener service.
- Regression hardening: `2602f7d` expanded route assertions for conservative normalization and boundary safety.

## Next Phase Readiness

- Plan 11-03 can consume accepted `apply_storage_event_refresh` listener jobs from the worker without relying on HTTP to enqueue direct scan jobs.
- The route contract now returns listener jobs consistently, giving worker-side tests a stable ingress artifact to build on.

## Self-Check: PASSED

- Found summary file: `.planning/phases/11-event-driven-refresh-hardening/11-02-SUMMARY.md`
- Found commit `b48d1bb`
- Found commit `9c01d16`
- Found commit `2602f7d`
- Verification command passed: `go test ./internal/httpapi -run 'TestStorageEvent'`

---
*Phase: 11-event-driven-refresh-hardening*
*Completed: 2026-04-24*
