## 1. Backend Library Type Support

- [x] 1.1 Add a mixed content library type constant/helper and use it anywhere library type comparisons are centralized.
- [x] 1.2 Ensure library creation and list responses accept and return the mixed content type without special catalog item types.

## 2. Mixed Scan Classification

- [x] 2.1 Extend scanner extra detection so `trailer`, `behind-the-scenes`, `sample`, `featurette`, `interview`, and `deleted scene` are recognized with token-bound matching for mixed grouping counts.
- [x] 2.2 Update directory/group shape resolution so mixed libraries classify exactly one non-extra video as movie content.
- [x] 2.3 Update directory/group shape resolution so mixed libraries classify more than one non-extra video as TV-like content and reuse deterministic fallback episode ordering when episode tokens are absent.
- [x] 2.4 Preserve existing movie and show library scan behavior by scoping the new count-based classification to mixed libraries only.

## 3. Frontend Library Creation

- [x] 3.1 Add a localized mixed content option to the settings library creation form.
- [x] 3.2 Ensure created mixed libraries display with an understandable type label in library management, sidebar, and home/library surfaces that show library type values.

## 4. Verification

- [x] 4.1 Add backend scan tests covering mixed movie-with-extras, mixed multi-video series fallback, and preservation of dedicated movie/show behavior.
- [x] 4.2 Add or update frontend tests where available for the mixed library type option, or manually verify the settings form behavior if no relevant test harness exists.
- [x] 4.3 Run focused backend tests for library scanning and frontend typecheck/build checks needed for touched files.
