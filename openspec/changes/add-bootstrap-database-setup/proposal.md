## Why

Mibo currently boots against a database chosen entirely by environment variables, and the `/setup` flow only creates the first administrator inside that already-selected database. That makes SQLite the practical default for first-run users, blocks database selection from the product itself, and leaves no guided path for switching first-run installs to Postgres or MySQL before data is created.

## What Changes

- Add a bootstrap database configuration flow that defaults to SQLite but allows first-run users to choose SQLite, Postgres, or MySQL from the setup experience.
- Add setup APIs for reading bootstrap database state, testing candidate connections, and applying a new bootstrap database configuration before the first admin account is created.
- Introduce a non-database-backed bootstrap configuration source so database driver and DSN can be persisted outside the runtime database and survive restart.
- Update server restart/bootstrap behavior so applying setup database changes reloads configuration and reconnects using the newly selected database.
- Extend backend database support from `sqlite` and `postgres` to include `mysql`, including driver wiring, validation, and compatibility coverage for startup migrations and supported write patterns.
- Redesign the setup UI into a multi-step onboarding flow that separates database selection from administrator creation and clearly explains when database settings are locked or environment-managed.
- Prevent database switching after initialization has created the first administrator, and surface clear read-only messaging when deployment-provided environment variables lock database configuration.

## Capabilities

### New Capabilities
- `bootstrap-database-config`: First-run bootstrap configuration for selecting, validating, applying, and persisting database connectivity outside the runtime database.
- `guided-setup-onboarding`: Multi-step onboarding that guides first-run users through database selection, bootstrap apply/restart, and first administrator creation.
- `mysql-runtime-support`: MySQL runtime compatibility for startup validation, migrations, and supported persistence flows used by setup and normal application boot.

### Modified Capabilities
- None.

## Impact

- Affected backend areas include `mibo-media-server/internal/config`, `internal/database`, `internal/app`, `internal/httpapi`, and startup wiring in `cmd/mibo-media-server`.
- Affected frontend areas include `frontend/src/features/setup`, setup gating, API client contracts, and setup-related route behavior.
- New setup endpoints will be added under `/api/v1/setup/*`.
- A new persisted bootstrap configuration file or equivalent non-database config source will be introduced.
- Backend dependencies will expand to include the GORM MySQL driver and corresponding validation/test coverage.
