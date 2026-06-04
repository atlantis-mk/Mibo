## ADDED Requirements

### Requirement: Setup separates database selection from administrator creation
The setup experience SHALL present database bootstrap as a distinct onboarding step before first-administrator creation.

#### Scenario: First-run user enters setup
- **WHEN** a first-run user opens the setup route and database configuration is not locked
- **THEN** setup presents a database selection step before the administrator creation step
- **AND** SQLite is preselected as the default driver

#### Scenario: Initialized deployment enters setup
- **WHEN** a deployment already has at least one user
- **THEN** setup does not present editable database selection controls
- **AND** setup proceeds directly to normal sign-in or initialized-state messaging

### Requirement: Setup supports driver-specific connection inputs
The setup experience SHALL allow users to select SQLite, Postgres, or MySQL and SHALL collect the connection details required for the selected driver.

#### Scenario: SQLite is selected
- **WHEN** the user selects SQLite in setup
- **THEN** setup shows the SQLite database path input
- **AND** setup hides Postgres/MySQL-only connection fields

#### Scenario: Postgres or MySQL is selected
- **WHEN** the user selects Postgres or MySQL in setup
- **THEN** setup shows driver-appropriate connection inputs for host, port, database name, username, password, and connection security settings

### Requirement: Setup guides users through apply-and-restart
The setup experience SHALL handle the restart boundary explicitly after a new bootstrap database configuration is applied.

#### Scenario: Applied config differs from active runtime database
- **WHEN** setup successfully applies a bootstrap database configuration that differs from the active runtime database
- **THEN** setup transitions into a waiting state that explains the server is restarting
- **AND** setup resumes onboarding only after the service reports readiness on the new configuration

#### Scenario: Restart does not complete successfully
- **WHEN** setup is waiting for the service to return after apply and the service fails to become ready within the expected interval
- **THEN** setup shows a retryable failure state
- **AND** the user is not advanced to administrator creation

### Requirement: Setup explains locked database configuration
The setup experience SHALL clearly explain why database settings cannot be edited when the deployment is environment-managed or already initialized.

#### Scenario: Environment-managed database is locked
- **WHEN** setup loads bootstrap state with an environment-managed lock
- **THEN** setup shows the active database type and a message that deployment environment settings control it

#### Scenario: Initialized database is locked
- **WHEN** setup loads bootstrap state after the first administrator already exists
- **THEN** setup shows the active database type and a message that switching is disabled after initialization
