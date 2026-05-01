# admin-user-management Specification

## Purpose
TBD - created by archiving change enable-user-management-create. Update Purpose after archive.
## Requirements
### Requirement: Admin user listing
The system SHALL provide an authenticated admin-only endpoint that returns the server's users with their identifiers, usernames, roles, creation timestamps, and update timestamps.

#### Scenario: Admin lists users
- **WHEN** an authenticated administrator requests the user list
- **THEN** the system returns all users ordered consistently by username or creation time
- **AND** each user record excludes password hashes and session tokens

#### Scenario: Non-admin cannot list users
- **WHEN** an authenticated non-admin user requests the user list
- **THEN** the system rejects the request with a forbidden response

#### Scenario: Anonymous user cannot list users
- **WHEN** a request without a valid session token requests the user list
- **THEN** the system rejects the request with an unauthorized response

### Requirement: Admin user creation
The system SHALL allow authenticated administrators to create a user by submitting a username, password, and role.

#### Scenario: Admin creates ordinary user
- **WHEN** an authenticated administrator submits a valid username, valid password, and role `user`
- **THEN** the system creates the account
- **AND** the response returns the created user without password hash or session data

#### Scenario: Admin creates administrator user
- **WHEN** an authenticated administrator submits a valid username, valid password, and role `admin`
- **THEN** the system creates an administrator account
- **AND** the account is included in later admin user list responses

#### Scenario: Duplicate username is rejected
- **WHEN** an authenticated administrator submits a username that already exists after normalization
- **THEN** the system rejects the request without creating another user

#### Scenario: Invalid role is rejected
- **WHEN** an authenticated administrator submits a role other than `user` or `admin`
- **THEN** the system rejects the request without creating a user

#### Scenario: Non-admin cannot create users
- **WHEN** an authenticated non-admin user submits a create-user request
- **THEN** the system rejects the request with a forbidden response

### Requirement: Settings user management UI
The settings user management page SHALL use server-backed user data and enable administrators to create users from the page.

#### Scenario: User list loads from server
- **WHEN** an administrator opens the settings user management page
- **THEN** the page requests the admin user list
- **AND** renders returned users instead of a single local session placeholder

#### Scenario: Administrator creates a user from settings
- **WHEN** an administrator completes the new-user form with valid input and submits it
- **THEN** the page creates the user through the admin API
- **AND** refreshes the displayed user list after success

#### Scenario: Creation failure is visible
- **WHEN** the create-user request fails validation or authorization
- **THEN** the page keeps the form available and shows a failure message to the administrator

