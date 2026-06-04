## ADDED Requirements

### Requirement: MySQL is a supported runtime database driver
The system SHALL accept `mysql` as a first-class runtime database driver for bootstrap configuration and normal application startup.

#### Scenario: MySQL driver is configured
- **WHEN** the configured bootstrap or environment database driver is `mysql`
- **THEN** startup validates the driver as supported
- **AND** the application attempts to open the database using the MySQL runtime driver

### Requirement: MySQL supports fresh startup migrations
The system SHALL complete startup schema initialization for a fresh MySQL database using the same boot flow used for other supported drivers.

#### Scenario: Fresh MySQL database boots successfully
- **WHEN** the application starts against an empty MySQL database with valid credentials
- **THEN** startup migrations complete without unsupported-driver errors
- **AND** setup status endpoints can query the initialized schema normally

### Requirement: Supported setup and boot write flows behave consistently on MySQL
The system SHALL preserve support for setup and startup write paths required to initialize a new deployment on MySQL.

#### Scenario: First administrator is created on MySQL
- **WHEN** setup creates the first administrator after booting on MySQL
- **THEN** the user record is persisted successfully
- **AND** subsequent setup status calls report that a user exists

#### Scenario: Startup writes bootstrap defaults on MySQL
- **WHEN** startup performs its normal default-row and migration follow-up writes against MySQL
- **THEN** those writes complete without driver-specific SQL errors
