---
phase: 11-event-driven-refresh-hardening
plan: 04
subsystem: api
tags: [go, httpapi, listener, storage-events, openlist, tdd]

requires:
  - phase: 11-event-driven-refresh-hardening/11-02
    provides: authenticated storage-event ingress delegated to listener.RecordStorageEvent
  - phase: 11-event-driven-refresh-hardening/11-03
    provides: worker dispatch for queued listener refresh intents
provides:
  - provider-aware non-local root path validation for OpenList `/` libraries
  - route regression proving `/MovieA.2024.mkv` child events enqueue listener refresh intent
  - preservation of local and non-root path boundary checks
affects: [11-event-driven-refresh-hardening, storage-event-ingress, openlist-root-validation]

tech-stack:
  added: []
  patterns:
    - normalized non-local root `/` accepts child event paths before listener enqueue
    - route tests assert listener payload root and fallback behavior at ingress boundaries

key-files:
  created:
    - .planning/phases/11-event-driven-refresh-hardening/11-04-SUMMARY.md
  modified:
    - mibo-media-server/internal/httpapi/handlers_storage_events.go
    - mibo-media-server/internal/httpapi/router_test.go

key-decisions:
  - "Special-case normalized non-local `/` roots in Mibo HTTP validation instead of moving provider-specific behavior into OpenList."
  - "Keep existing local filepath.Rel validation and non-root non-local prefix checks unchanged."

patterns-established:
  - "Root-provider validation pattern: once a non-local library root normalizes to `/`, every normalized absolute child path remains inside that root."
  - "OpenList-root regression pattern: insert source/library rows directly in sqlite to test route validation and listener enqueue without a live OpenList server."

requirements-completed: [LIST-01, LIST-02]

duration: 6min
completed: 2026-04-24
---

# Phase 11 Plan 04: OpenList Root Storage-Event Validation Summary

**OpenList libraries rooted at `/` now accept valid absolute child storage events and enqueue targeted listener refresh intent.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-24T10:11:47Z
- **Completed:** 2026-04-24T10:18:13Z
- **Tasks:** 2 completed
- **Files modified:** 2 code/test files plus this summary

## Accomplishments

- Added a route-level regression for authenticated OpenList storage-event intake with a library rooted at `/` and child path `/MovieA.2024.mkv`.
- Fixed non-local path validation so normalized root `/` accepts normalized absolute child paths before delegating to `listener.RecordStorageEvent`.
- Preserved escaping protections for local providers and non-root non-local libraries by leaving `filepath.Rel` and the existing prefix check intact.

## Task Commits

Each task was committed atomically:

1. **Task 1: Prove OpenList root child storage events are accepted** - `a155bfd` (`test(11-04): add failing OpenList root storage event regression`)
2. **Task 2: Fix non-local `/` root path validation without weakening boundary checks** - `8d9f715` (`fix(11-04): accept non-local root storage event paths`)

**Plan metadata:** this summary commit.

## Files Created/Modified

- `mibo-media-server/internal/httpapi/handlers_storage_events.go` - Adds the normalized non-local `/` root acceptance branch while preserving all other boundary checks.
- `mibo-media-server/internal/httpapi/router_test.go` - Adds `TestStorageEventEndpointAcceptsOpenListRootChildPath` covering OpenList root ingress, response job kind, and listener payload shape.
- `.planning/phases/11-event-driven-refresh-hardening/11-04-SUMMARY.md` - Records this plan completion.

## Verification

- `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi -run 'TestStorageEventEndpointAcceptsOpenListRootChildPath'` — FAIL before Task 2 as expected during TDD RED (`400` outside root), then covered by passing storage-event suite after the fix.
- `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi -run 'TestStorageEvent'` — PASS
- `cd /root/Mibo/mibo-media-server && go test ./...` — PASS
- Acceptance criteria confirmed: new test includes `/MovieA.2024.mkv`, asserts `202`, `listener.JobKindApplyStorageEventRefresh`, payload `RootPath == "/"`, and `FallbackFullSync == false`; handler still contains `filepath.Rel(...)` and the non-root prefix check.

## Decisions Made

- Special-cased normalized non-local `/` roots in `validateStorageEventPath`, matching the mathematical boundary that all normalized absolute paths are inside `/`.
- Kept the fix inside `mibo-media-server` HTTP trust-boundary validation rather than changing OpenList adapter semantics.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The initial failing route test was the expected TDD RED gate for the planned validation gap.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. Stub scan only found existing test literals/empty checks and the intentional empty-path validation branch; no plan-introduced placeholder or unwired UI data exists.

## Threat Flags

None beyond the plan threat model. The storage-event trust boundary and path validation change were explicitly covered by T-11-04-01 through T-11-04-03.

## TDD Gate Compliance

- RED gate: `a155bfd` added `TestStorageEventEndpointAcceptsOpenListRootChildPath`, which failed before the validation fix with `400` for `/MovieA.2024.mkv` under root `/`.
- GREEN gate: `8d9f715` fixed non-local `/` root validation and made the full `TestStorageEvent` route suite pass.

## Next Phase Readiness

- LIST-01/LIST-02 ingress behavior now covers the default OpenList root `/` scenario.
- Plan 11-05 can focus solely on the remaining concurrent active-intent uniqueness gap without reworking storage-event path validation.

## Self-Check: PASSED

- Found file: `mibo-media-server/internal/httpapi/handlers_storage_events.go`
- Found file: `mibo-media-server/internal/httpapi/router_test.go`
- Found summary file: `.planning/phases/11-event-driven-refresh-hardening/11-04-SUMMARY.md`
- Found commit `a155bfd`
- Found commit `8d9f715`
- Verification passed: `go test ./internal/httpapi -run 'TestStorageEvent'`
- Verification passed: `go test ./...`

---
*Phase: 11-event-driven-refresh-hardening*
*Completed: 2026-04-24*
