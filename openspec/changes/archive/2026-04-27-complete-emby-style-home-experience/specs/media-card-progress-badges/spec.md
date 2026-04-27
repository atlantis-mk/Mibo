## ADDED Requirements

### Requirement: Poster Card Metadata
The system SHALL render media poster cards with poster imagery, display title, media type where useful, and year or year-range status derived from catalog-native fields.

#### Scenario: Series year range
- **WHEN** a catalog series has first and last air dates or an active series status
- **THEN** the poster card displays a year range such as `1999 - 2000` or `2026 - 现在` instead of only a single year

#### Scenario: Missing year data
- **WHEN** a catalog item lacks usable year or air-date data
- **THEN** the poster card displays a stable fallback such as `未知年份` without breaking layout

### Requirement: Green Count Badges
The system SHALL display a green numeric badge on poster cards when a meaningful count is available, using a deterministic priority order for unwatched count, update count, available child count, then total child count.

#### Scenario: Count badge available
- **WHEN** a poster card has a meaningful count from progress or catalog child summary data
- **THEN** the card displays that number in a green badge at the poster's top-right corner

#### Scenario: No count badge available
- **WHEN** a poster card has no meaningful count
- **THEN** the card omits the badge without reserving empty badge space

### Requirement: Card Navigation And Quick Actions
The system SHALL make poster cards navigate to item detail by default while exposing quick play or continue actions when playback is available.

#### Scenario: Open item detail
- **WHEN** the user activates the main body of a poster card
- **THEN** the app navigates to the catalog item's detail page

#### Scenario: Quick continue
- **WHEN** the user activates a continue action on a card with stored progress
- **THEN** the app opens playback using the existing progress-aware playback behavior

#### Scenario: Quick play from start
- **WHEN** the user activates a play-from-start action on a card with playable media
- **THEN** the app opens the playback route with start-from-beginning semantics

### Requirement: Responsive Rail Cards
The system SHALL keep poster-card rails usable on desktop and mobile viewports.

#### Scenario: Desktop rail
- **WHEN** the viewport is wide enough to show multiple poster cards
- **THEN** the rail displays several cards with hover or focus quick actions

#### Scenario: Mobile rail
- **WHEN** the viewport is narrow or touch-oriented
- **THEN** the rail remains horizontally scrollable and card actions remain accessible without relying only on hover
