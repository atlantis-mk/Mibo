## 1. Backend Hierarchical Browse Model

- [x] 1.1 Audit the existing catalog browse, library projection, and inventory models to define the library-root, folder-node, and item-node response contracts.
- [x] 1.2 Implement a backend hierarchical browse service that resolves accessible library roots and filesystem-derived child folders from inventory-backed library data.
- [x] 1.3 Define deterministic leaf-item mapping rules for recognized metadata items and discovered inventory entries under a folder path.
- [x] 1.4 Add breadcrumb, parent-node, pagination, and node-identity helpers for hierarchical browse requests.

## 2. Backend API And Authorization

- [x] 2.1 Add authenticated HTTP routes and request parsing for hierarchical library browse entry, library-node traversal, and folder-node traversal.
- [x] 2.2 Ensure hierarchical browse applies existing library visibility and item availability rules before returning nodes or counts.
- [x] 2.3 Add serialization types that distinguish `library`, `folder`, and `item` nodes while preserving existing metadata item identifiers for leaf actions.

## 3. Frontend Library Navigation

- [x] 3.1 Extend `frontend/src/lib/mibo-api.ts` and query helpers with hierarchical library browse request and response types.
- [x] 3.2 Update the `/library` route and `frontend/src/features/library` to load library-root nodes first and navigate through folder nodes with stable query state.
- [x] 3.3 Add breadcrumb, back-navigation, and mixed node rendering so folder nodes open deeper levels and item nodes continue to existing detail/playback routes.
- [x] 3.4 Preserve relevant loading, empty, pagination, and organizing-state UI behavior for hierarchical browse results.

## 4. Data Refresh And Compatibility

- [x] 4.1 Decide and implement whether hierarchical folder nodes are computed on demand or refreshed alongside existing library projection workflows.
- [x] 4.2 Ensure rescans, projection rebuilds, and inventory changes refresh hierarchical browse results without requiring manual repair steps.
- [x] 4.3 Confirm hierarchical browse works for local and plugin-backed libraries whose file paths were normalized during scan.

## 5. Verification

- [x] 5.1 Add backend tests for accessible library-root listing, nested folder traversal, invalid path rejection, breadcrumb generation, and mixed item results.
- [x] 5.2 Add backend tests covering visibility filtering, hidden-item exclusion, and deterministic folder assignment for multi-resource metadata items.
- [x] 5.3 Add frontend tests for hierarchical navigation state, breadcrumb behavior, and item-vs-folder interactions where the current test setup supports them.
- [x] 5.4 Run focused frontend and backend test suites for the new hierarchical browse flow.
