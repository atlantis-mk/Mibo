## MODIFIED Requirements

### Requirement: User Menu
The system SHALL provide a user menu from the app shell that shows the current user and exposes session-relevant actions, including access to login device management when available.

#### Scenario: Open user menu
- **WHEN** the user activates the user entry in the top navigation
- **THEN** the app displays a menu with the current username and available actions such as settings, device management, and logout

#### Scenario: Open login device management
- **WHEN** the user chooses the device management action from the user menu
- **THEN** the app navigates to `/settings/devices`

#### Scenario: Logout
- **WHEN** the user chooses logout from the user menu
- **THEN** the current session is cleared using existing auth behavior and the app returns to the login flow
