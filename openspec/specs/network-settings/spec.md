# network-settings Specification

## Purpose
Define authenticated server network settings management, including read and update APIs, validation, secret certificate handling, UI server-state integration, and runtime activation clarity.

## Requirements

### Requirement: Network Settings API
The system SHALL provide authenticated API endpoints for reading and updating server network settings.

#### Scenario: Read default network settings
- **WHEN** an authenticated user requests the network settings before any network settings have been saved
- **THEN** the system returns default settings for local networks, local/public ports, remote access, proxy trust, TLS mode, port mapping, and streaming limits

#### Scenario: Save network settings
- **WHEN** an authenticated user submits a valid network settings payload
- **THEN** the system persists the settings server-side and returns the saved settings

#### Scenario: Reject unauthenticated access
- **WHEN** a request without a valid session accesses the network settings endpoints
- **THEN** the system rejects the request with an unauthorized response

### Requirement: Network Settings Validation
The system SHALL validate network settings before persistence and reject invalid values with actionable errors.

#### Scenario: Invalid address list entry
- **WHEN** a user submits a local network or remote filter entry that is not a valid IP address or CIDR range
- **THEN** the system rejects the update and identifies the invalid field

#### Scenario: Invalid port
- **WHEN** a user submits a local or public port outside the valid TCP port range
- **THEN** the system rejects the update and identifies the invalid port field

#### Scenario: Invalid enumerated option
- **WHEN** a user submits an unsupported remote filter mode, secure connection mode, or request protocol option
- **THEN** the system rejects the update and identifies the invalid option field

### Requirement: Secret Certificate Fields
The system SHALL protect certificate password values when network settings include TLS certificate configuration.

#### Scenario: Save certificate password
- **WHEN** an authenticated user submits a certificate password
- **THEN** the system stores it as a secret setting and does not expose the raw value in later read responses

#### Scenario: Clear certificate password
- **WHEN** an authenticated user explicitly clears the certificate password
- **THEN** the system removes the stored secret and reports that no certificate password is configured

### Requirement: Network Settings Page Uses Server State
The network settings page at `/settings/network` SHALL use the server settings API as its source of truth.

#### Scenario: Load network settings page
- **WHEN** an authenticated user opens `http://localhost:3000/settings/network`
- **THEN** the page loads the current settings from the server and populates the network form

#### Scenario: Save from network settings page
- **WHEN** a user changes valid network settings and saves the form
- **THEN** the page sends the update to the server, shows saving progress, and displays the returned saved values after success

#### Scenario: Save error from network settings page
- **WHEN** the server rejects a network settings update
- **THEN** the page keeps the user's draft visible and displays the error so the user can correct it

### Requirement: Network Settings Runtime Clarity
The system SHALL communicate whether saved network settings are immediately active or require restart or future runtime integration.

#### Scenario: Settings saved but not immediately active
- **WHEN** a user saves fields that cannot take effect in the current process without restart or additional runtime wiring
- **THEN** the page explains that the settings are saved but may require restart or later runtime support before taking effect

#### Scenario: Configuration-only port mapping
- **WHEN** a user enables automatic port mapping before real port mapping is implemented
- **THEN** the page and API status identify it as a saved preference rather than an active port mapping operation
