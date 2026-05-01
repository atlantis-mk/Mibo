## 1. Data Model And Migration

- [x] 1.1 Add nullable `library_id` support to the scan exclusion rule model and database migration.
- [x] 1.2 Preserve existing scan exclusion rules as global rules with no library scope.
- [x] 1.3 Implement scoped key or uniqueness handling so duplicates are rejected only within the same scope.
- [x] 1.4 Add validation that non-null rule scopes reference an existing media library.

## 2. Backend Rule Behavior

- [x] 2.1 Update scan exclusion rule DTOs and request handlers to read and return optional `library_id` scope.
- [x] 2.2 Update rule creation and update logic to support global and library-scoped rules.
- [x] 2.3 Update scan rule loading to apply enabled global rules plus enabled rules scoped to the scanned library.
- [x] 2.4 Ensure `configurable_exclusion_rules = false` disables both global and library-scoped configurable rules.
- [x] 2.5 Delete rules scoped to a library when that library is deleted while preserving global and other-library rules.

## 3. Frontend Rule Management

- [x] 3.1 Extend frontend API types and methods for scan exclusion rule `library_id` scope.
- [x] 3.2 Add rule scope controls to the scan exclusion rule settings UI with global and specific-library choices.
- [x] 3.3 Display each rule's scope in the rule list so global and library-scoped rules are distinguishable.
- [x] 3.4 Show validation errors when a scoped rule cannot be saved.

## 4. Tests And Verification

- [x] 4.1 Add backend tests that existing no-scope rules behave as global rules.
- [x] 4.2 Add backend tests that scans include matching library-scoped rules and exclude other-library rules.
- [x] 4.3 Add backend tests that library deletion removes only rules scoped to the deleted library.
- [x] 4.4 Add backend tests for scoped duplicate rejection and equivalent rules in different scopes.
- [x] 4.5 Run focused backend tests covering scan exclusion rules, library deletion, and scan behavior.
- [x] 4.6 Run frontend typecheck after updating API types and UI.
