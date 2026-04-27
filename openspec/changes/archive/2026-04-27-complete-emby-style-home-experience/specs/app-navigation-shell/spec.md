## ADDED Requirements

### Requirement: Real Mibo Sidebar Navigation
The system SHALL replace placeholder documentation sidebar content with real Mibo navigation for Home, Favorites, Search, media libraries, Settings, and relevant management areas.

#### Scenario: Sidebar shows product navigation
- **WHEN** an authenticated user opens the sidebar
- **THEN** the sidebar displays Mibo product navigation rather than documentation sample entries

#### Scenario: Sidebar library navigation
- **WHEN** library data is available to the sidebar
- **THEN** each library entry links to its corresponding library detail route

### Requirement: Homepage Top Navigation
The system SHALL provide a homepage top navigation shell with a menu trigger, Mibo brand, Home/Favorites switch, search entry, cast entry, user entry, and settings entry.

#### Scenario: Search from top navigation
- **WHEN** the user activates the search entry from the homepage top navigation
- **THEN** the app opens the global search surface or focuses a search input that submits to global search

#### Scenario: Settings from top navigation
- **WHEN** the user activates the settings entry from the homepage top navigation
- **THEN** the app navigates to the settings route

#### Scenario: Switch to favorites
- **WHEN** the user activates Favorites in the Home/Favorites switch
- **THEN** the app opens the favorites browsing surface

### Requirement: User Menu
The system SHALL provide a user menu from the app shell that shows the current user and exposes session-relevant actions.

#### Scenario: Open user menu
- **WHEN** the user activates the user entry in the top navigation
- **THEN** the app displays a menu with the current username and available actions such as settings and logout

#### Scenario: Logout
- **WHEN** the user chooses logout from the user menu
- **THEN** the current session is cleared using existing auth behavior and the app returns to the login flow

### Requirement: Cast Entry Behavior
The system SHALL expose a cast entry in the app shell and clearly communicate whether casting is available.

#### Scenario: Cast unsupported
- **WHEN** the user activates the cast entry and real casting support is not implemented
- **THEN** the app displays a clear unavailable or coming-soon message

#### Scenario: Cast available
- **WHEN** the user activates the cast entry and casting support is available
- **THEN** the app opens the available device or cast-control flow
