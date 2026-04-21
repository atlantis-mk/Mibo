# Phase 2: Library & Async Sync Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-21
**Phase:** 02-library-async-sync-foundation
**Areas discussed:** Scan Status UX, Scheduled Refresh, Scan Behavior, Media Source Config

---

## Scan Status UX

| Option | Description | Selected |
|--------|-------------|----------|
| Library status badge | Status badge on library cards/settings. Admin triggers scan, sees status update when worker finishes. | |
| Jobs list view | Dedicated Jobs page showing all background tasks with status, errors, retry actions. | |
| Hybrid (badge + jobs) | Status badge for quick feedback, plus Jobs list accessible from settings for detailed monitoring. | ✓ |
| Agent decides | Let the agent decide based on complexity vs value tradeoff. | |

**User's choice:** Hybrid (badge + jobs)
**Notes:** Quick status visible on library cards, detailed job monitoring accessible from settings.

---

## Scheduled Refresh

| Option | Description | Selected |
|--------|-------------|----------|
| Per-library cron-like schedule | Each library has its own schedule (daily at X time, weekly on Y day). Most flexible but more config UI. | |
| Global refresh interval | One system-wide interval (e.g., every 6 hours). All libraries refresh together. Simplest. | ✓ |
| Per-library interval | Each library has a refresh interval (every N hours). Default 24h. Balance of flexibility and simplicity. | |
| Agent decides | Let the agent pick the simplest approach that satisfies LIBR-04. | |

**User's choice:** Global refresh interval
**Notes:** Satisfies LIBR-04 with minimal config complexity.

---

## Scan Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Merge (add/update/delete missing) | Add new files, update changed files, soft-delete items no longer on disk. Keeps playback progress. | ✓ |
| Full rebuild | Delete all catalog items then rescan. Nuclear option, admin must request explicitly. | |
| Agent decides | Go with merge as default, but adjust if Phase 6 needs different semantics. | |

**User's choice:** Merge (add/update/delete missing)
**Notes:** Current scan.go already implements this behavior. Confirmed as the right default.

---

## Media Source Config

| Option | Description | Selected |
|--------|-------------|----------|
| Just provider + root | No additional local storage config. Simple V1. Add more later if real needs emerge. | ✓ |
| Exclude paths | Ability to exclude subdirectories or filename patterns from scan. | |
| Scan depth limit | Limit recursive scan depth to prevent scanning entire drives accidentally. | |
| Agent decides | Pick the simplest approach that works for a typical home NAS setup. | |

**User's choice:** Just provider + root
**Notes:** Keep V1 simple. Exclude patterns and depth limits can come later if real use cases demand.

---

## Deferred Ideas

None — discussion stayed within phase scope

