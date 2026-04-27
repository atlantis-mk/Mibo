## ADDED Requirements

### Requirement: Library poster cards show green count badges
The system SHALL show a green numeric badge on library poster cards when the catalog item has a meaningful child, available, in-progress, or unwatched count.

#### Scenario: Show series count badge
- **WHEN** a series card has a positive badge count from catalog summary data
- **THEN** the poster displays a green numeric badge in the upper-right corner

### Requirement: Library poster cards show year ranges
The system SHALL show a single year for movies and a year range or continuing-series label for series when the data is available.

#### Scenario: Show continuing series year range
- **WHEN** a series has a start year and is still continuing
- **THEN** the library poster card displays a year range equivalent to `2019 - 现在`

### Requirement: Library poster cards preserve quick actions
The system SHALL preserve detail navigation and quick play or continue actions on poster cards where the item is playable.

#### Scenario: Play available item
- **WHEN** a user activates the quick play action on an available library card
- **THEN** the app navigates to playback for that catalog item or its playable asset

### Requirement: Library poster cards support favorite actions
The system SHALL support favorite and unfavorite actions on library poster cards when user favorite state is available.

#### Scenario: Toggle favorite from library card
- **WHEN** a user toggles the favorite action on a library poster card
- **THEN** the favorite state updates for that user and the card reflects the new state
