## ADDED Requirements

### Requirement: Authenticated clients can read current user settings
The system SHALL provide an authenticated endpoint for retrieving the current user's settings at `/api/v1/me/settings`.

#### Scenario: User reads settings without a saved record
- **WHEN** an authenticated user sends `GET /api/v1/me/settings` and no settings have been saved for that user
- **THEN** the system returns `200 OK`
- **AND** the response contains the complete supported settings document with default values applied

#### Scenario: User reads previously saved settings
- **WHEN** an authenticated user sends `GET /api/v1/me/settings` after saving settings
- **THEN** the system returns `200 OK`
- **AND** the response contains the canonical saved values for that user together with any unchanged defaulted fields

### Requirement: Authenticated clients can replace current user settings
The system SHALL allow an authenticated user to update the current user's settings with `PUT /api/v1/me/settings`.

#### Scenario: First-time user saves settings
- **WHEN** an authenticated user sends a valid `PUT /api/v1/me/settings` request and no settings record exists yet
- **THEN** the system persists a new settings record scoped to that user
- **AND** the system returns `200 OK` with the canonical saved settings document

#### Scenario: Existing user replaces settings
- **WHEN** an authenticated user sends a valid `PUT /api/v1/me/settings` request and a settings record already exists for that user
- **THEN** the system updates that user's existing settings record
- **AND** the system returns `200 OK` with the canonical saved settings document

### Requirement: User settings payloads are validated and normalized
The system SHALL validate supported user settings fields and reject unsupported enum values or structurally invalid payloads.

#### Scenario: Invalid enum value is rejected
- **WHEN** an authenticated user sends `PUT /api/v1/me/settings` with an unsupported enum value such as an unknown theme or subtitle mode
- **THEN** the system returns `400 Bad Request`
- **AND** the response identifies that the request payload is invalid

#### Scenario: Optional string fields are normalized
- **WHEN** an authenticated user sends `PUT /api/v1/me/settings` with optional string fields containing surrounding whitespace
- **THEN** the system trims those fields before persistence
- **AND** the response returns the normalized values

### Requirement: User settings are isolated per authenticated user
The system SHALL scope settings storage and retrieval to the authenticated user only.

#### Scenario: One user's settings do not affect another user
- **WHEN** user A saves settings and user B later sends `GET /api/v1/me/settings`
- **THEN** user B does not receive user A's saved values
- **AND** user B receives only user B's own saved settings or defaults

### Requirement: User settings endpoints require authentication
The system SHALL reject unauthenticated access to current-user settings endpoints.

#### Scenario: Unauthenticated read request is rejected
- **WHEN** a client sends `GET /api/v1/me/settings` without a valid authenticated user session
- **THEN** the system returns `401 Unauthorized`

#### Scenario: Unauthenticated update request is rejected
- **WHEN** a client sends `PUT /api/v1/me/settings` without a valid authenticated user session
- **THEN** the system returns `401 Unauthorized`
