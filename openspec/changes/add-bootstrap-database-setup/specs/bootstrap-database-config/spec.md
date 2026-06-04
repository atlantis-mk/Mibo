## ADDED Requirements

### Requirement: First-run database bootstrap defaults to SQLite
The system SHALL expose bootstrap database state before setup is complete and SHALL default to SQLite when no environment-managed or persisted bootstrap database configuration is present.

#### Scenario: No bootstrap config exists
- **WHEN** the service starts without database environment variables and without a persisted bootstrap database configuration
- **THEN** the active bootstrap database driver is reported as `sqlite`
- **AND** the default SQLite database path is reported to the setup experience as the editable first-run default

#### Scenario: Persisted bootstrap config exists
- **WHEN** the service starts without database environment variables and with a persisted bootstrap database configuration
- **THEN** the persisted bootstrap driver and connection settings are used as the active bootstrap database configuration

### Requirement: Setup can validate candidate database connections
The system SHALL provide a setup-safe way to validate a candidate SQLite, Postgres, or MySQL connection before applying it as the active bootstrap database configuration.

#### Scenario: Candidate connection is valid
- **WHEN** setup submits a candidate database configuration for validation
- **THEN** the system reports that the connection is valid for the selected driver
- **AND** the response identifies any normalized connection values needed for apply

#### Scenario: Candidate connection is invalid
- **WHEN** setup submits a candidate database configuration that cannot connect or initialize safely
- **THEN** the system rejects the validation request
- **AND** the response includes a user-visible error message describing the failure

### Requirement: Setup can apply bootstrap database configuration before initialization
The system SHALL allow setup to apply a new bootstrap database configuration only before the first administrator account exists.

#### Scenario: Applying a new first-run database choice
- **WHEN** setup applies a validated bootstrap database configuration and no users exist
- **THEN** the configuration is persisted outside the runtime database
- **AND** the system reports that a restart is required to activate the new database

#### Scenario: Attempting to apply after initialization
- **WHEN** setup attempts to apply a new bootstrap database configuration after at least one user exists
- **THEN** the system rejects the request
- **AND** the response states that database switching is locked after initialization

### Requirement: Environment-managed database settings are read-only
The system SHALL treat database configuration as read-only in setup when the active database driver or DSN is explicitly provided by deployment environment variables.

#### Scenario: Environment variables manage the database
- **WHEN** the service starts with explicit database environment variables
- **THEN** setup bootstrap state marks the database configuration as environment-managed
- **AND** setup apply requests to change the database are rejected

#### Scenario: Environment-managed config is displayed
- **WHEN** setup loads bootstrap database state for an environment-managed deployment
- **THEN** the active driver and connection summary are returned for display
- **AND** the state includes a reason that the configuration cannot be edited from setup
