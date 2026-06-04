## Context

Mibo currently loads database configuration only from environment variables during process startup, opens the selected database before the web UI is served, and treats `/setup` as a thin first-user registration page. That shape works for default SQLite local development but does not provide a product-managed way to choose Postgres or MySQL before initialization. It also means database connectivity settings cannot be stored in the runtime database because the runtime database depends on those settings already being available.

This change crosses startup configuration, backend database wiring, setup APIs, restart behavior, and frontend onboarding. It also introduces MySQL as a third supported runtime driver, which expands the compatibility surface for migrations and write flows that currently only run under SQLite-heavy tests.

## Goals / Non-Goals

**Goals:**
- Allow first-run users to keep the default SQLite path or switch to Postgres/MySQL from the setup experience before the first administrator is created.
- Persist bootstrap database configuration outside the runtime database so it survives restart and can be reloaded during boot.
- Add setup endpoints to inspect bootstrap state, test candidate connections, and apply a bootstrap configuration safely.
- Ensure restart actually reloads configuration and reconnects using the newly applied database.
- Add MySQL runtime support to the existing SQLite/Postgres boot and migration path.
- Lock database switching once initialization has created the first administrator or when deployment environment variables explicitly manage database configuration.

**Non-Goals:**
- Migrating user or media data from an already-initialized SQLite database into Postgres/MySQL.
- Exposing arbitrary runtime database switching after the first administrator exists.
- Replacing all environment-variable configuration with file-backed configuration.
- Introducing advanced connection pooling or per-driver operational tuning beyond safe defaults needed for startup.

## Decisions

### 1. Introduce a bootstrap config file for database connectivity

Database connectivity will be resolved from a new non-database-backed bootstrap file, with environment variables retaining highest priority.

Proposed precedence:
- Environment variables
- Bootstrap config file
- Built-in defaults

Rationale:
- Database settings cannot live in `system_settings` because the database must already be available to read them.
- Preserving environment priority keeps containerized and managed deployments deterministic.
- A file-backed bootstrap source gives the setup UI somewhere durable to write first-run choices.

Alternatives considered:
- Store bootstrap config in the current SQLite file. Rejected because switching away from SQLite would create circular state and confusing ownership.
- Store bootstrap config only in browser local storage. Rejected because restart would not persist server-side behavior.

### 2. Treat environment-managed database settings as locked in the setup UI

If `MIBO_DATABASE_DRIVER` or `MIBO_DATABASE_DSN` is explicitly provided, setup will surface the active database configuration as environment-managed and read-only.

Rationale:
- The running service must not present editable controls that are overridden on the next boot.
- This prevents a false impression that UI changes will stick in Docker, service managers, or hosted environments where env vars are authoritative.

Alternatives considered:
- Let UI override environment variables. Rejected because it makes deployment behavior surprising and difficult to reason about.
- Hide database configuration entirely when env vars are set. Rejected because visibility without editability is still useful.

### 3. Split setup into bootstrap selection and administrator creation

The setup flow will become a small state machine:
1. Load bootstrap database state
2. Let the user choose SQLite/Postgres/MySQL and edit driver-specific connection details when unlocked
3. Test the candidate connection
4. Apply and trigger restart when the candidate differs from the active runtime configuration
5. Create the first administrator after the service returns on the selected database

Rationale:
- The current setup page assumes the runtime database is already final.
- Separating database selection from administrator creation makes the initialization boundary explicit and prevents first user records from landing in the wrong store.

Alternatives considered:
- Add database fields to the existing single-page admin form. Rejected because it obscures the restart boundary and mixes unrelated failure states.

### 4. Lock database switching after the first administrator exists

Once `user_count > 0`, setup will expose the active database configuration but will not permit changing driver or DSN.

Rationale:
- After first-user creation, switching databases becomes a data migration problem, not a bootstrap configuration problem.
- Refusing the change is safer and much smaller in scope than attempting one-off export/import behavior.

Alternatives considered:
- Best-effort data copy during switch. Rejected because it expands scope into migration, rollback, and consistency handling.

### 5. Extend the backend driver matrix to include MySQL with scoped compatibility guarantees

MySQL support will include:
- Config validation accepts `mysql`
- Database open path uses the GORM MySQL driver
- AutoMigrate completes successfully for a fresh database
- Existing startup write paths used by setup and normal boot remain supported

This change will not promise deep driver-specific performance tuning or historical data migration from other engines.

Rationale:
- The current data model and write patterns are largely GORM-based and appear compatible enough to justify first-class runtime support.
- Keeping the support contract scoped to startup, migrations, and supported write flows makes the first implementation achievable.

Alternatives considered:
- Ship MySQL as a hidden experimental toggle. Rejected because the user explicitly wants it selectable during setup and hidden support creates inconsistent expectations.

### 6. Reload configuration on restart by moving config loading inside the restart loop

The process entrypoint will reload configuration for each boot cycle instead of caching `cfg` before the restart loop.

Rationale:
- Without this, applying bootstrap config and requesting restart still reuses stale database settings.
- This is the minimum change needed to make setup-applied configuration effective.

Alternatives considered:
- Hot-swap the live database connection without restart. Rejected because too many services are constructed around the boot-time `*gorm.DB` graph.

## Risks / Trade-offs

- [Bootstrap file path or permissions are invalid] → Validate writable path during apply and return actionable setup errors before restart.
- [Environment-managed deployments confuse users who expect UI edits to persist] → Surface explicit read-only messaging that cites environment ownership.
- [MySQL migrations expose schema or index differences not covered by SQLite tests] → Add fresh-database integration coverage for MySQL and validate critical startup write paths.
- [Restart leaves the frontend in an indeterminate waiting state] → Design setup polling around `/healthz` or setup bootstrap state with clear retry and failure messaging.
- [Users create an admin in SQLite before switching] → Prevent switching once any user exists and gate the database step before admin creation.

## Migration Plan

1. Add bootstrap config loading and persistence with backwards-compatible defaults that continue to select SQLite when no bootstrap file or env vars exist.
2. Move config loading into the restart loop so restart observes newly written bootstrap state.
3. Add MySQL driver support and validate startup migrations against fresh SQLite, Postgres, and MySQL databases.
4. Introduce setup bootstrap endpoints and keep the old register path working when the active database is already final.
5. Replace the current setup UI with the new multi-step onboarding flow.
6. Deploy with no bootstrap file by default; existing env-based deployments continue unchanged because env precedence wins.

Rollback:
- Remove the bootstrap file and restart to fall back to environment/default configuration.
- If MySQL support proves unstable, hide MySQL from setup while keeping bootstrap file support for SQLite/Postgres.

## Open Questions

- What exact bootstrap file path should be considered stable across desktop, local binary, and container deployments?
- Should the setup connection test run full `AutoMigrate` on a temporary connection or a narrower validation pass before apply?
- Do we want to expose DSN as a raw advanced field in addition to structured host/user/password inputs for Postgres and MySQL?
