## ADDED Requirements

### Requirement: Login Session Listing
The system SHALL let an authenticated user view their own login sessions as manageable devices from `/settings/devices`.

#### Scenario: User opens devices settings page
- **WHEN** an authenticated user navigates to `/settings/devices`
- **THEN** the page displays the user's active login sessions with device name, client type or user agent, last activity time, creation time, expiration time, and current-session status

#### Scenario: Existing session has missing metadata
- **WHEN** a session does not have captured device metadata
- **THEN** the page displays safe fallback labels without hiding the session from the list

#### Scenario: Unauthenticated user requests login sessions
- **WHEN** a request without a valid bearer token asks for login sessions
- **THEN** the API rejects the request with an unauthorized response

### Requirement: Current Session Protection
The system SHALL identify the current login session and protect it from device-management revocation actions.

#### Scenario: Current session is shown
- **WHEN** the login sessions list is returned for an authenticated request
- **THEN** exactly the session represented by the request bearer token is marked as current

#### Scenario: User attempts to revoke current session from device management
- **WHEN** the user tries to revoke the current session from `/settings/devices`
- **THEN** the system prevents that action and keeps the session active, directing current-session sign-out through the existing logout flow

### Requirement: Revoke Other Login Sessions
The system SHALL allow an authenticated user to revoke login sessions for their own account other than the current session.

#### Scenario: Revoke one other session
- **WHEN** the user revokes a non-current session from `/settings/devices`
- **THEN** the API deletes that session, the revoked token can no longer authenticate, and the page refreshes the session list

#### Scenario: Revoke all other sessions
- **WHEN** the user confirms revoking all other sessions
- **THEN** the API deletes all sessions for that user except the current session and the page shows only remaining active sessions

#### Scenario: User attempts to revoke another user's session
- **WHEN** an authenticated user requests revocation for a session that belongs to another user
- **THEN** the API does not revoke the session and returns a not-found or unauthorized response without exposing that the session exists

### Requirement: Login Device Metadata Capture
The system SHALL capture basic request metadata when a login session is created for later display in login device management.

#### Scenario: User logs in through web client
- **WHEN** login succeeds
- **THEN** the created session stores non-secret metadata such as user agent, remote address, device display name, and client type when available

#### Scenario: Login metadata is unavailable
- **WHEN** a login request lacks recognizable client metadata
- **THEN** the session is still created and remains manageable with fallback display values
