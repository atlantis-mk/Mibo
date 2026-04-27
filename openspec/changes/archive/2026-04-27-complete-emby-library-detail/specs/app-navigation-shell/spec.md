## ADDED Requirements

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
