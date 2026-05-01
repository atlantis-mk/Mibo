## Why

Manual scan exclusions currently behave like single-file hides, which makes repeated advertisement or wrong-import files tedious to clean up when the same filename appears across multiple directories or sources. Users need a safer batch workflow that can ignore a filename globally, show the affected files, and recover either one file or the whole filename rule.

## What Changes

- Add global filename exclusion rules keyed only by normalized filename.
- Let users choose between ignoring only the selected file or ignoring all same-name files after the UI shows the impact count/list.
- Hide already-scanned files from all sources that match an enabled filename exclusion rule without physically deleting source files or losing recovery history.
- Add per-file restore exceptions so a single file can be restored while the filename rule remains active.
- Add whole-rule restore by disabling the filename exclusion rule, allowing all matching files to be scanned again.
- Update exclusion management UI to show filename rules, affected files, restored exceptions, and restore actions.

## Capabilities

### New Capabilities
- `filename-exclusion-rules`: User-managed filename-based scan exclusion rules, affected-file visibility, and restore behavior.

### Modified Capabilities

## Impact

- Backend library scan exclusion service and scan decision ordering.
- Database schema for filename exclusion rules and per-file restore exceptions or equivalent persisted state.
- HTTP API for previewing same-name impact, creating filename rules, listing affected files, restoring a single file, and restoring a rule.
- Frontend media card ignore action and settings scan exclusions panel.
- Tests for scan matching, existing-file cleanup, restore exceptions, rule restore, and API/UI contracts.
