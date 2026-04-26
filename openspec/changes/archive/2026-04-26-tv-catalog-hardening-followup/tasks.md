## 1. TV Hierarchy And Metadata Completeness

- [x] 1.1 Audit the remaining gap between `.planning/TV-SERIES-METADATA-GOVERNANCE-PLAN.md` and the implemented catalog metadata flow, then codify the concrete missing descendant behaviors to close in this change.
- [x] 1.2 Complete season and episode metadata persistence so descendant catalog items retain durable provider identities, source evidence, and artwork candidates/selections needed by governance and APIs.
- [x] 1.3 Add or complete hierarchy-native read endpoints for catalog children and TV convenience use cases that still depend on compatibility logic, including explicit missing and next-up style flows if absent.
- [x] 1.4 Add focused backend tests covering descendant evidence, descendant artwork, missing/unaired hierarchy responses, and hierarchy-native TV queries.

## 2. Frontend Catalog Contract Cleanup

- [x] 2.1 Remove primary dependence on `catalogListItemToMediaItem`, `catalogItemDetailToMediaItemDetail`, and similar legacy-shape adapters in search, library, media detail, and governance-entry flows.
- [x] 2.2 Introduce catalog-native presentation helpers or view models only where necessary so UI code can render availability, hierarchy, and asset state without relying on legacy-only fields.
- [x] 2.3 Update media detail and related catalog pages to consume catalog-native hierarchy and asset semantics end-to-end, including reprobe, asset choice, and empty-state handling.
- [x] 2.4 Verify frontend type and build health after adapter removal with focused tests or walkthrough-oriented validation notes.

## 3. Governance Action Completeness

- [x] 3.1 Extend the governance workspace and backend support so linked assets are not only displayed but can be reviewed and corrected when hierarchy or linkage mismatches are detected.
- [x] 3.2 Add UI affordances for hierarchy conflict review, such as provider/local mismatch summaries, child-state review, and safe corrective actions for item-asset relationships.
- [x] 3.3 Ensure field locks, source evidence, image selection, and asset-link corrections compose cleanly without overwriting unrelated governance state.
- [x] 3.4 Add focused backend and frontend coverage for governance correction flows and descendant workspace behavior.

## 4. Cutover Hardening And Legacy Retirement

- [x] 4.1 Add the missing production-grade database safety constraints and indexes that are still absent for the catalog graph, asset linkage, and descendant identity paths.
- [x] 4.2 Complete rebuild and consistency-check coverage for rollups, availability, and catalog search documents so operators can validate safety before cleanup.
- [x] 4.3 Enable catalog reads by default only after the combined backend/frontend verification gates for hierarchy, playback, governance, and compatibility removal pass.
- [x] 4.4 Remove, retire, or isolate the remaining legacy media read/write paths once cutover validation succeeds, preserving only bounded recovery paths if still required.
- [x] 4.5 Update repository-facing operational notes for catalog-read defaults, rebuild workflows, and legacy fallback expectations.
