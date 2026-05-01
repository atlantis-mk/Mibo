## Why

Scan exclusion rules are currently reusable but global, which forces every library to share the same filename, directory, and path-pattern filters. Users need rules that can apply only to one media library while keeping global defaults and manual per-item exclusions intact.

## What Changes

- Add optional media-library scope to configurable scan exclusion rules.
- Keep existing global rules as `library_id = null` and apply them to all libraries.
- Allow user-created rules to target either all libraries or one specific library.
- Load scan rules with scope filtering so a scan sees global rules plus rules for the scanned library.
- Delete library-scoped rules when the owning library is deleted.
- Update API and settings UI to expose rule scope and selected library.
- Preserve current system rules as global rules and keep the existing library scan-policy toggle for configurable rules.

## Capabilities

### New Capabilities
- `scoped-scan-exclusion-rules`: Defines global and library-scoped configurable scan exclusion rules.

### Modified Capabilities
- None.

## Impact

- Backend database model/migration for `scan_exclusion_rules.library_id` and scoped uniqueness/key generation.
- Library scan rule loading and exclusion matching.
- Library deletion cleanup for scoped rules.
- Scan exclusion rule create/list/update APIs and frontend types/UI.
- Tests for global compatibility, library-specific matching, deletion cleanup, and UI/API scope behavior.
