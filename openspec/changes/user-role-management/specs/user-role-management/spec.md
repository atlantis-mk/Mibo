## ADDED Requirements

### Requirement: Role management
The system MUST allow administrators to create, update, list, and delete roles.

#### Scenario: Create a role
- **WHEN** an administrator submits a new role name
- **THEN** the system SHALL persist the role and make it available for assignment

#### Scenario: Delete a role
- **WHEN** an administrator deletes an unused role
- **THEN** the system SHALL remove the role from future assignments

### Requirement: User role assignment
The system MUST allow administrators to assign one or more roles to a user and remove assigned roles.

#### Scenario: Assign roles to a user
- **WHEN** an administrator updates a user's role selection
- **THEN** the system SHALL persist the binding between the user and each selected role

#### Scenario: Remove a role from a user
- **WHEN** an administrator removes a role from a user
- **THEN** the system SHALL revoke that role binding immediately

### Requirement: Role-aware management access
The system MUST use a user's roles to determine access to management features.

#### Scenario: Authorized admin access
- **WHEN** a user with an allowed role opens a management action
- **THEN** the system SHALL permit the action

#### Scenario: Unauthorized admin access
- **WHEN** a user without an allowed role attempts the same action
- **THEN** the system SHALL deny the action
