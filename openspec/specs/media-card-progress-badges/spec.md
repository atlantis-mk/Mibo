# media-card-progress-badges Specification

## Purpose
Define poster-card metadata, progress badges, navigation, and quick action behavior.

## Requirements

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

### Requirement: Episode rail cards expose current episode state
The system SHALL distinguish the currently opened episode in episode rails and same-season shelves.

#### Scenario: Current episode appears in same-season shelf
- **WHEN** a user opens an episode detail page and the same-season shelf includes that episode
- **THEN** the corresponding episode card MUST visually indicate that it is the current episode without preventing navigation to other episode cards

#### Scenario: Current episode has progress
- **WHEN** the current episode has watched or in-progress state
- **THEN** the card MUST show progress state and current-episode state together without hiding the episode still or title

### Requirement: Episode rail cards use batch progress data
The system SHALL render watched and in-progress states on episode rail cards using catalog item progress data when available.

#### Scenario: Progress data is provided for rail episodes
- **WHEN** an episode rail receives progress state for one or more episode IDs
- **THEN** cards for those episodes MUST show watched or progress labels consistently with existing poster-card progress semantics

#### Scenario: Progress data is not provided
- **WHEN** an episode rail receives no progress state for an episode
- **THEN** the card MUST still render availability, date, runtime, and synopsis without showing incorrect watched or progress indicators
