## Context

Mibo already has manual scan exclusions stored as file-scoped records and configurable scan exclusion rules for automatic advertisement filtering. The current manual action on a media card marks the selected catalog item as an advertisement exclusion, hides its linked inventory file and scanner asset, and skips future scans by stable identity or path.

That model is too narrow for repeated ad or wrong-import files that share a filename across folders. The new behavior needs to remain reversible, avoid physical deletion, and preserve existing scan policy/rule behavior while adding a safer same-name batch path.

## Goals / Non-Goals

**Goals:**
- Support global filename exclusion rules across all libraries and storage providers.
- Let the UI preview same-name impact before users choose a single-file ignore or same-name ignore.
- Hide already-scanned matching files immediately after creating a filename rule.
- Preserve restore history and allow single-file exceptions while a filename rule stays enabled.
- Allow rule-level restore by disabling the filename rule.
- Keep existing manual file exclusions and configurable scan exclusion rules working.

**Non-Goals:**
- Do not physically delete media files from local disk or OpenList.
- Do not create fuzzy or tokenized filename exclusions in this change.
- Do not replace configurable filename-token, directory-segment, or path-pattern rules.
- Do not infer fuzzy filename similarity; matching is exact on normalized basename plus extension.

## Decisions

1. Model same-name ignores as global filename exclusion rules, not as many duplicated file exclusions.

   A rule represents the user intent: "ignore files named X everywhere." This keeps list, restore, and future scan behavior coherent. Expanding into many file exclusions would make whole-rule restore and match display harder and would not cover future files cleanly.

   Alternative considered: create one `scan_exclusions` row per matching file. Rejected because future matches would need a separate rule anyway, and rule identity would be implicit.

2. Scope filename rules globally by normalized filename only.

   The user intent for same-name ignore is source-independent: once a filename is identified as unwanted, scans from any library or provider should skip that exact normalized filename. The confirmation preview therefore lists all current matches across sources.

3. Normalize match values as exact filename basenames with extension.

   Store a normalized filename derived from the storage path basename, preserving the extension and using case-insensitive comparison. This treats `ad.mp4` and `ad.mkv` as different files while avoiding case-only misses.

   Alternative considered: basename without extension. Rejected because common names like `sample` across formats would be too broad.

4. Add per-file restore exceptions that override filename rules.

   A single-file restore cannot work reliably if the filename rule continues to match it on the next scan. The scan decision order should check restore exceptions before filename rules, then continue to existing file exclusions and configurable rules.

   Proposed order:
   ```text
   restored exception -> allow
   enabled manual file exclusion -> skip
   enabled filename rule -> skip
   configurable exclusion rule -> skip
   scan policy skip -> skip
   import/update catalog
   ```

   Manual file exclusions remain before filename rules so existing explicit file-level ignores continue to win over broader filename behavior.

5. Hide already-scanned matching files through existing missing/unlink semantics.

   Creating a rule should find matching inventory files across all libraries/providers, mark their inventory/assets missing, remove asset-item links, and recalculate catalog availability. This mirrors current manual exclusion behavior and avoids destructive source-file deletion.

   Alternative considered: hard-delete catalog and inventory rows. Rejected because it would weaken auditability and restore.

6. Expose preview and explicit creation modes through API.

   The frontend should not guess impact by reimplementing backend matching. Add backend endpoints that return the normalized filename, all-source affected files, and counts. The create endpoint should accept mode `file` or `filename_rule`; filename rule creation uses the previewed backend match semantics.

## Risks / Trade-offs

- Filename matching can still exclude legitimate same-name content -> Require preview confirmation and clearly label the rule as applying to all sources.
- Single-file restore needs stable targeting even if paths move -> Prefer `stable_identity_key` for exceptions when available, falling back to normalized path.
- Restoring a rule does not automatically recreate catalog entries until a scan runs -> UI copy should state that restored files are allowed back on next scan; implementation may optionally enqueue a library scan if existing job patterns make that safe.
- Existing file exclusions and new filename rules could overlap -> Keep deterministic scan decision ordering and show source/type in exclusion management.
- Large libraries could make impact preview expensive -> Query inventory rows and filter by normalized basename; if needed, cap UI display while returning total count.

## Migration Plan

- Add database tables or equivalent persisted records for filename exclusion rules and restore exceptions.
- Existing `scan_exclusions` rows remain unchanged and continue to work.
- No automatic conversion of old file exclusions to filename rules.
- Rollback can ignore the new tables and leave existing file exclusions unaffected; hidden files may require a scan or rule restore before rollback if users created rules.

## Open Questions

- Should rule restore optionally trigger a rescan job immediately, or only allow the next scheduled/manual scan to reimport files?
- Should preview display all affected files or return the first page plus total count for very large matches?
