## Context

The app currently has user-related settings but no first-class role model. This change spans frontend, backend, and persistence because user-role bindings must be stored, surfaced in the UI, and enforced in admin flows.

## Goals / Non-Goals

**Goals:**
- Add a persistent role model.
- Bind users to one or more roles.
- Surface role assignment in admin-facing UI.
- Make role-aware authorization available for management actions.

**Non-Goals:**
- Full fine-grained permission editor.
- External identity provider integration.
- Hierarchical role inheritance.

## Decisions

- Store roles and user-role bindings in the backend database.
  - Rationale: keeps authorization decisions local and consistent.
  - Alternative: derive roles from client-side config; rejected because it is not enforceable.
- Model bindings as many-to-many between users and roles.
  - Rationale: supports future admin and support scenarios without redesign.
  - Alternative: single role per user; rejected because it is too limiting.
- Expose dedicated role CRUD and assignment APIs.
  - Rationale: keeps user management and role management explicit and testable.
  - Alternative: overload existing user update endpoints; rejected due to unclear semantics.
- Seed a small default role set for initial adoption.
  - Rationale: prevents empty-state lockout and makes rollout usable.
  - Alternative: require manual bootstrap; rejected because it adds operational friction.

## Risks / Trade-offs

- [Authorization gaps] -> Mitigate by applying role checks only to clearly identified management endpoints first.
- [Migration complexity] -> Mitigate by adding defaults and keeping old users accessible during rollout.
- [UI ambiguity] -> Mitigate by keeping role assignment entry points limited to admin screens.

## Migration Plan

1. Add role tables and bindings with defaults.
2. Backfill existing users into a safe default role.
3. Deploy backend APIs and UI together behind existing admin surfaces.
4. Verify role-aware access on management actions.
5. Roll back by disabling role checks while preserving stored assignments if needed.

## Open Questions

- Should users be allowed multiple roles at launch, or should the UI constrain to one primary role?
- Which management actions should be role-gated in the first release?
