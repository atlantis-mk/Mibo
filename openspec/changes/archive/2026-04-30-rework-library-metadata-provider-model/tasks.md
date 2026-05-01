## 1. Data Model And Migration

- [x] 1.1 Extend metadata provider persistence to support a system-managed `local_scan` provider type and expose it as a locked, non-configurable provider instance.
- [x] 1.2 Add runtime persistence for per-library metadata strategies with ordered provider instance IDs per stage and library-specific language overrides.
- [x] 1.3 Implement startup bootstrap and migration logic that creates the built-in `local_scan` instance and backfills every library's effective current metadata behavior into a concrete strategy row.

## 2. Backend Runtime

- [x] 2.1 Update metadata resolution to read executable provider order from library metadata strategies instead of `local_only`, `force_local_only`, and legacy metadata policy enable/priority flags.
- [x] 2.2 Add provider-stage capability validation and template-application copy semantics for TMDB and `local_scan` strategy inputs.
- [x] 2.3 Implement `local_scan` detail-stage execution by consuming existing scanner sidecar evidence instead of reading storage directly from the metadata service.
- [x] 2.4 Record provider-instance provenance for strategy-driven metadata executions, including local scan refreshes and remote fallback selections.

## 3. API And Compatibility Layer

- [x] 3.1 Add library metadata strategy management APIs and response contracts that expose ordered stage providers, language overrides, and optional template context.
- [x] 3.2 Rework metadata profile services and HTTP endpoints so profiles behave as reusable templates rather than runtime local-only sentinels.
- [x] 3.3 Remove long-term runtime dependence on `migrated-default-local-only`, `migrated-default-online`, `local_only`, `force_local_only`, and legacy metadata policy enable/priority fields while keeping only the minimum migration compatibility needed during rollout.

## 4. Frontend Configuration Flows

- [x] 4.1 Update `web/src/lib/mibo-api.ts` and related queries to use library metadata strategy and template semantics instead of profile-plus-policy runtime fields.
- [x] 4.2 Replace library create/edit metadata controls with a simple provider choice for common flows and advanced per-stage strategy editing for existing libraries.
- [x] 4.3 Update metadata settings management to show the built-in `local_scan` provider as read-only and present profiles as reusable templates without local-only toggles.

## 5. Verification

- [x] 5.1 Add backend tests for strategy backfill, provider capability validation, and `local_scan` provider bootstrap behavior.
- [x] 5.2 Add backend tests for local scan metadata execution, metadata provenance recording, and library APIs that read or update metadata strategies.
- [x] 5.3 Run `go test ./...` in `mibo-media-server` and `pnpm typecheck` in `web`, and confirm the resulting behavior matches the new library metadata strategy and template model.
