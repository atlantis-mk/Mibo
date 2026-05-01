## Why

Scan exclusion currently combines user-marked records with hard-coded advertisement filename and folder checks. Users can restore persisted exclusions, but they cannot inspect, tune, add, disable, or delete automatic scan rules from the product UI, so false positives and site-specific naming conventions require code changes.

## What Changes

- Add configurable scan exclusion rules that can express filename token, directory segment, path pattern, and extension-scoped matching for supported video files.
- Replace the current hard-coded advertisement marker list with seeded configurable rules that preserve existing behavior by default.
- Add authenticated CRUD APIs for scan exclusion rules, including list, create, update, delete, enable, and disable operations.
- Extend the Settings scan exclusions area with a rules management page or section for creating, editing, enabling, disabling, deleting, and testing rule definitions.
- Make rule changes immediately affect subsequent scan decisions without requiring a backend restart or scanner code change.
- Preserve existing user-marked `scan_exclusions` behavior and keep user exclusions separate from automatic configurable rules.
- Report skipped files with enough detail to distinguish user exclusions from configurable rule matches.

## Capabilities

### New Capabilities
- `configurable-scan-exclusion-rules`: Defines configurable automatic scan exclusion rules, management APIs, settings UI behavior, and immediate scanner enforcement.

### Modified Capabilities
- `scan-file-exclusions`: Automatic advertisement filtering changes from hard-coded markers to configurable rule records while preserving existing default behavior.

## Impact

- Affected backend scanner code in `mibo-media-server/internal/library`, especially `scanExclusionDecision` and automatic rule matching.
- Affected database schema and migrations through a new configurable scan exclusion rule model and default seed behavior.
- Affected HTTP routing and handlers with authenticated CRUD endpoints under the existing scan exclusion/settings API surface.
- Affected frontend settings pages in `web/src/features/settings`, API client code in `web/src/lib/mibo-api.ts`, and query keys/options in `web/src/lib/mibo-query.ts`.
- Tests should cover rule CRUD, validation, default seeded behavior, immediate scanner reads, false-positive avoidance, and UI management flows where practical.
