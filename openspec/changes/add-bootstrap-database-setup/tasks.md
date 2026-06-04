## 1. Bootstrap Configuration Foundation

- [x] 1.1 Add a bootstrap database configuration source outside the runtime database, including precedence rules of `env > bootstrap file > defaults`.
- [x] 1.2 Define backend bootstrap state models that report active driver, lock state, initialization lock state, and safe connection summaries for setup.
- [x] 1.3 Move config loading into the restart loop so restart requests boot with freshly persisted bootstrap configuration.
- [x] 1.4 Add automated coverage for bootstrap config resolution, environment-managed locking, and restart-time config reload behavior.

## 2. Database Driver Support

- [x] 2.1 Extend database config validation and open logic to accept `mysql` and wire the GORM MySQL driver.
- [x] 2.2 Normalize SQLite, Postgres, and MySQL candidate connection inputs into the runtime database config shape used by startup and setup.
- [x] 2.3 Verify startup migrations and follow-up default writes succeed on fresh MySQL databases.
- [x] 2.4 Add or update backend tests to cover first-user creation and setup status reads on MySQL.

## 3. Setup Bootstrap APIs

- [x] 3.1 Add a setup endpoint to read bootstrap database state, including active driver, edit locks, initialization locks, and connection field defaults.
- [x] 3.2 Add a setup endpoint to validate candidate SQLite, Postgres, and MySQL configurations before apply.
- [x] 3.3 Add a setup endpoint to apply a validated bootstrap database configuration, persist it outside the runtime database, and request restart.
- [x] 3.4 Enforce backend guards that reject apply requests when environment variables own the database config or when the first administrator already exists.
- [x] 3.5 Add backend tests for bootstrap read, validation success/failure, apply success, apply lock rejection, and restart-required responses.

## 4. Guided Setup Onboarding UI

- [x] 4.1 Redesign the setup page into distinct database-selection, apply/restart, and administrator-creation steps.
- [x] 4.2 Add driver-specific setup form controls for SQLite path and Postgres/MySQL connection details with SQLite selected by default.
- [x] 4.3 Integrate connection test and apply actions into the setup UI with clear success, error, and restart-required messaging.
- [x] 4.4 Add setup waiting-state behavior that polls for service recovery after restart and resumes onboarding only when the new runtime is ready.
- [x] 4.5 Surface read-only messaging for environment-managed or already-initialized database configurations and prevent user edits in those states.
- [x] 4.6 Update frontend setup tests and API client contracts to cover the new onboarding state machine.

## 5. Initialization Guardrails and Documentation

- [x] 5.1 Keep first-administrator creation working only after the active runtime database is finalized and reachable after restart.
- [x] 5.2 Update setup and bootstrap documentation to explain default SQLite behavior, Postgres/MySQL first-run selection, environment lock behavior, and post-initialization switch restrictions.
- [ ] 5.3 Run targeted frontend and backend verification for SQLite, Postgres, and MySQL first-run flows before marking the change ready for apply.
