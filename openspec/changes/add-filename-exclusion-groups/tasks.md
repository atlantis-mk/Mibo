## 1. Data Model

- [x] 1.1 Add persisted global filename exclusion rule model keyed by normalized filename, reason, enabled state, user audit fields, and timestamps.
- [x] 1.2 Add persisted per-file restore exception model keyed by rule and stable identity when available, with storage path fallback and user audit fields.
- [x] 1.3 Add database migration/autoload coverage for the new models without altering existing `scan_exclusions` behavior.
- [x] 1.4 Add helper functions for normalized filename extraction and case-insensitive matching with extension preserved.

## 2. Backend Service

- [x] 2.1 Implement same-name impact preview from item, asset, or inventory file targets using backend matching semantics across all sources.
- [x] 2.2 Implement filename exclusion rule create/update that hides matching already-scanned files using existing missing/unlink/catalog availability semantics.
- [x] 2.3 Implement single-file restore exception creation for a rule match.
- [x] 2.4 Implement rule restore by disabling the filename exclusion rule while preserving history.
- [x] 2.5 Update scan exclusion decision ordering to honor restore exceptions, manual file exclusions, filename rules, configurable rules, then scan policy skips.
- [x] 2.6 Extend exclusion listing service to return filename rules, affected counts, affected files, and restored match state for management UI.

## 3. HTTP API

- [x] 3.1 Add authenticated preview endpoint for scan exclusion impact from catalog item targets.
- [x] 3.2 Add authenticated endpoint to create a filename exclusion rule from a catalog item target.
- [x] 3.3 Add authenticated endpoint to list filename exclusion rules and their affected/restored matches with enabled filters where applicable.
- [x] 3.4 Add authenticated endpoint to restore one file within a filename exclusion rule.
- [x] 3.5 Add authenticated endpoint to restore or re-enable a filename exclusion rule.
- [x] 3.6 Preserve existing file-scoped scan exclusion endpoints for single-file ignore compatibility.

## 4. Frontend UX

- [x] 4.1 Update media card ignore action to open a preview flow instead of immediately marking the item ignored.
- [x] 4.2 Show affected same-name count and file list before offering `仅忽略当前文件` and `忽略所有同名文件` actions.
- [x] 4.3 Wire single-file ignore to the existing file-scoped behavior and same-name ignore to filename rule creation.
- [x] 4.4 Update scan exclusions settings panel to display filename rules alongside existing manual exclusions without confusing automatic rules.
- [x] 4.5 Add rule detail UI showing excluded versus individually restored files.
- [x] 4.6 Add actions for restoring one file and restoring all same-name files, with clear copy that restored files return on a future scan.

## 5. Tests And Verification

- [x] 5.1 Add backend unit tests for normalized filename matching, all-source behavior, extension preservation, and restore exception precedence.
- [ ] 5.2 Add backend integration tests for rule creation hiding already-scanned files and recalculating catalog availability.
- [ ] 5.3 Add HTTP handler tests for preview, rule creation, single-file restore, rule restore, and authorization.
- [x] 5.4 Add frontend type/API coverage for new response shapes and mutations.
- [ ] 5.5 Run `go test ./...` from `mibo-media-server/`.
- [x] 5.6 Run `pnpm typecheck` from `web/`.
