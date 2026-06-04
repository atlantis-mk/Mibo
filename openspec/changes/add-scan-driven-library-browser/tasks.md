## 1. Projection Schema

- [x] 1.1 Add `display_root_path`, `display_parent_path`, `display_kind`, and `display_path_source` fields to the `library_metadata_projections` database model and migration path.
- [x] 1.2 Update projection persistence and conflict-update clauses so rebuilds write the new display-path fields.
- [x] 1.3 Ensure projection model tests and schema/index checks cover the new fields where needed.

## 2. Display-Root Derivation

- [x] 2.1 Implement projection-time helpers that collect source-role inventory paths for a projected metadata item within a library.
- [x] 2.2 Implement movie display-root derivation for single-file, multi-version, and split-part layouts using normalized relative paths.
- [x] 2.3 Implement series display-root derivation by rolling up descendant episode/season source files to a shared series root.
- [x] 2.4 Add structural-child normalization for season, split-part, and edition/version folders and record the chosen `display_kind` and `display_path_source`.

## 3. Hierarchical Browse Integration

- [x] 3.1 Update hierarchical browse entry construction to use projection display-root fields as the primary item placement source.
- [x] 3.2 Keep the existing folder-inference path as a compatibility fallback when projection display data is absent or incomplete.
- [x] 3.3 Update folder-collapsing rules so category folders remain browsable while structural media directories can still surface items directly.
- [x] 3.4 Preserve existing item identifiers, authorization filtering, and organizing-state behavior for browse leaves sourced from projection display paths.

## 4. Refresh And Backfill

- [x] 4.1 Wire display-path recomputation into item-level, resource-level, and library-level projection rebuild flows.
- [x] 4.2 Verify existing ingest/projection dirty workflows backfill the new display fields without requiring a separate manual browse repair path.
- [x] 4.3 Define startup or upgrade expectations for libraries whose projections predate the new display-path fields and document fallback behavior.

## 5. Verification

- [x] 5.1 Add backend tests for movie display-root derivation covering plain directories, multi-version folders, split-part folders, and sidecar-file exclusion.
- [x] 5.2 Add backend tests for series display-root derivation from descendant episode resources, including season-folder collapse and mixed-root fallback behavior.
- [x] 5.3 Add hierarchical browse tests confirming projection-driven placement, category-folder preservation, and fallback behavior before backfill.
- [x] 5.4 Run focused backend test suites for catalog projection rebuilds and hierarchical browse behavior after the new fields are introduced.
