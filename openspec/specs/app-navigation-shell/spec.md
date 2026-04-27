# app-navigation-shell Specification

## Purpose
Define the authenticated app navigation shell for Mibo media browsing surfaces.

## Requirements

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

### Requirement: Library detail top bar includes primary navigation entries
The system SHALL show a library detail top bar with menu access, a home entry, and the current library or category title.

#### Scenario: Open library detail top bar
- **WHEN** a user opens a library detail page
- **THEN** the top-left area provides menu access, a home entry, and the current library title

### Requirement: Library detail top bar includes utility entries
The system SHALL show search, cast, user, and settings entries in the library detail top bar where viewport space allows.

#### Scenario: Use search from library detail
- **WHEN** a user activates the search entry from a library detail page
- **THEN** the app navigates to or opens the existing search experience

### Requirement: Unsupported cast action is explicit
The system SHALL show a clear unavailable or coming-soon message when the cast entry is present but real casting is not implemented.

#### Scenario: Activate unsupported cast
- **WHEN** a user activates the cast entry before real casting support exists
- **THEN** the page displays a message explaining that casting is not yet available

### Requirement: Product promo action avoids Emby branding
The system SHALL avoid copying Emby Premiere branding and SHALL use only a Mibo-specific placeholder or product action if an upgrade/promo button is shown.

#### Scenario: Show promo action
- **WHEN** a promo or upgrade action is present on the library detail page
- **THEN** its label and destination use Mibo-specific terminology rather than Emby branding

### Requirement: Detail page top navigation uses app shell semantics
The system SHALL render detail-page top navigation entries as real app actions or clearly unavailable actions consistent with the authenticated app shell.

#### Scenario: User activates detail search entry
- **WHEN** the user activates the search entry from the media detail top navigation
- **THEN** the app MUST open the global search surface or navigate to the search route instead of rendering a decorative icon only

#### Scenario: User activates detail user entry
- **WHEN** the user activates the user entry from the media detail top navigation
- **THEN** the app MUST display the user menu with session-relevant actions consistent with the app shell

#### Scenario: User activates unsupported cast entry
- **WHEN** the user activates a cast entry from the media detail top navigation and real casting support is unavailable
- **THEN** the app MUST show clear unavailable or coming-soon feedback instead of silently doing nothing

#### Scenario: User activates settings entry
- **WHEN** the user activates the settings entry from the media detail top navigation
- **THEN** the app MUST navigate to the settings route
