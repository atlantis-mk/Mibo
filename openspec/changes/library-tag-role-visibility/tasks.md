## 1. Data Model And Admin APIs

- [x] 1.1 Add persistence for library access tags, library-to-tag assignments, and role allow/deny tag rules
- [x] 1.2 Extend library management APIs to list available library access tags and create/update library tag assignments
- [x] 1.3 Extend role management APIs to read and update allow/deny library tag rules
- [x] 1.4 Ensure authenticated user payloads and authorization helpers expose the full assigned role set needed for visibility evaluation

## 2. Visibility Scope Resolution

- [x] 2.1 Implement a shared service that resolves a user's accessible library set from assigned roles, allow rules, deny rules, and the default-open policy
- [x] 2.2 Encode and test the precedence rules for unlabeled libraries, missing allow rules, matching allow rules, and matching deny rules
- [x] 2.3 Define the behavior for direct library-scoped requests when the requested library is outside the resolved accessible library set

## 3. Catalog And Playback Enforcement

- [x] 3.1 Apply accessible-library filtering to catalog browse, home, search, favorites, continue watching, and recently played queries
- [x] 3.2 Update metadata item detail and resource listing flows to return only resources sourced from accessible libraries while preserving display of items available in at least one accessible library
- [x] 3.3 Enforce accessible-library checks in playback candidate selection and direct media resource access paths
- [x] 3.4 Add regression tests for cross-library duplicate content, deny-overrides-allow behavior, and blocked playback/resource access

## 4. Frontend Management And User Experience

- [x] 4.1 Add library management UI for reusing existing access tags, creating new access tags, and assigning tags to libraries
- [x] 4.2 Add role management UI for configuring allow and deny library access tag rules
- [x] 4.3 Surface the default-open warning for unlabeled libraries in the library management experience
- [x] 4.4 Verify home, browse, favorites, detail, and playback UI flows behave correctly when content is filtered by accessible libraries

## 5. Rollout And Verification

- [x] 5.1 Add end-to-end verification that existing users retain current access when no library tags or allow rules are configured
- [x] 5.2 Document the rollout behavior and operator expectations for “unlabeled library = visible to all authenticated users”
- [x] 5.3 Validate rollback behavior by confirming stored tags and role rules can remain in place while enforcement is disabled
