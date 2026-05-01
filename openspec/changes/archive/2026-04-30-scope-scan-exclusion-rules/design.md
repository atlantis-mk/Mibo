## Context

Mibo already stores configurable scan exclusion rules centrally and applies them during library scans. Existing manual scan exclusions are library-specific, but reusable rule definitions are global, so a rule created for one library can affect unrelated libraries.

This change extends the existing rule store instead of introducing a separate per-library rules table. Existing global system and user rules remain valid and continue to apply across libraries unless a library scan policy disables configurable exclusion rules.

## Goals / Non-Goals

**Goals:**
- Allow a scan exclusion rule to be global or scoped to one media library.
- Preserve existing global rule behavior and existing scan policy controls.
- Apply scoped rules only when scanning their owning library.
- Remove library-scoped rules when the owning library is deleted.
- Expose rule scope through backend APIs and frontend settings UI.

**Non-Goals:**
- Replacing manual scan exclusions or changing their precedence.
- Adding inheritance beyond global plus one library scope.
- Importing OpenList or Emby library option models.
- Changing catalog read/write behavior outside scan exclusion decisions.

## Decisions

1. Store rule scope in `scan_exclusion_rules.library_id`.

   `NULL` means global. A non-null value references the owning library. This keeps all configurable rules in one place, preserves current records without data migration beyond adding a nullable column, and allows cleanup with a bounded delete when a library is removed.

   Alternative considered: create `library_scan_exclusion_rules`. That would split rule management and require duplicate CRUD/listing paths for global and scoped rules without a clear benefit.

2. Load active rules by library scope at scan time.

   Scans use enabled global rules plus enabled rules for the current library. Existing scan policy `configurable_exclusion_rules` remains the switch for configurable rule participation; when disabled, neither global nor library-scoped configurable rules are applied.

   Alternative considered: precompute effective rule sets per library. That adds cache invalidation and is unnecessary for the expected rule count.

3. Keep system rules global.

   Seeded/system rules remain `library_id = NULL`. Library-scoped rules are user-configurable and are removed with the library that owns them.

   Alternative considered: allowing system rules per library. There is no current system-owned library-specific source of truth, so this would create lifecycle ambiguity.

4. Use scoped key uniqueness.

   Rule keys must be unique within a rule scope so the same token or pattern can exist globally and in a specific library when needed. Implementations can achieve this by generating scope-aware keys or by adding a scope-aware uniqueness strategy compatible with SQLite's nullable semantics.

   Alternative considered: keep globally unique keys. That would prevent a library from defining a local rule with the same normalized value as a global rule and would make scope semantics harder to explain.

## Risks / Trade-offs

- Existing key uniqueness can conflict with scoped rules -> Generate keys with a scope prefix or use a uniqueness strategy that treats global and library scopes distinctly.
- UI could make duplicate global and scoped rules confusing -> Display rule scope clearly and show the selected library for scoped rules.
- Deleting a library could leave orphaned scoped rules -> Add explicit deletion cleanup and cover it with backend tests.
- Disabling configurable exclusions could be misread as only disabling global rules -> Keep behavior simple: the library policy disables all configurable rules, both global and library-scoped.

## Migration Plan

- Add nullable `library_id` to `scan_exclusion_rules`.
- Leave existing rows with `library_id = NULL` so they remain global.
- Update rule creation to accept optional `library_id` and validate that referenced libraries exist.
- Update scan rule queries to filter by `library_id IS NULL OR library_id = ?`.
- Update library deletion flow to remove rows where `library_id` matches the deleted library.
- Rollback can ignore the nullable column if no scoped rules have been created; once scoped rules exist, rollback would require deleting or globalizing them intentionally.

## Open Questions

- None.
