## ADDED Requirements

### Requirement: Immersive detail hero metadata
The system SHALL render media detail hero metadata using user-facing catalog media fields before technical or governance fields.

#### Scenario: Detail hero shows media facts
- **WHEN** a user opens a catalog-backed media detail page with rating, year or year range, official rating, genres, runtime, status, provider, or season summary data
- **THEN** the hero MUST show the available media facts in a compact readable metadata row without using metadata match confidence as the media rating

#### Scenario: Detail hero lacks optional facts
- **WHEN** a detail item lacks optional media facts such as rating, official rating, genres, or end year
- **THEN** the hero MUST omit those empty facts without showing placeholder values that imply missing technical state

### Requirement: Consumer-oriented hero actions
The system SHALL keep the primary detail action row focused on playback and library consumption actions.

#### Scenario: User opens primary actions
- **WHEN** a user opens a media detail page
- **THEN** the primary action row MUST expose play or continue, watched-state control, favorite control, and a more menu with visible focus and accessible labels

#### Scenario: Management actions are available but secondary
- **WHEN** metadata management, rematch, reprobe, or other governance actions are available for the item
- **THEN** those actions MUST be reachable from a secondary menu or management grouping rather than competing with the primary play action row

### Requirement: Focused season episode shelf
The system SHALL provide a season selector for series details and render the selected numbered season as the primary episode shelf.

#### Scenario: User switches season
- **WHEN** a series has multiple numbered seasons and the user selects a season
- **THEN** the detail page MUST update the episode shelf to show that season's episodes without navigating away from the detail route

#### Scenario: Series has one numbered season
- **WHEN** a series has only one numbered season
- **THEN** the detail page MAY omit the season selector while still labeling the episode shelf with the season name and episode count

### Requirement: Specials shelf separation
The system SHALL render specials separately from numbered season episodes when specials are present.

#### Scenario: Series has specials
- **WHEN** a series hierarchy includes a season number `0` or a season identified as specials
- **THEN** the detail page MUST render those episodes in a separate Specials or 特别篇 shelf instead of mixing them into numbered seasons

### Requirement: Episode cards use episode-aware labels and status
The system SHALL render episode cards with episode-aware labels, availability, date, runtime, synopsis, and progress or watched signals when available.

#### Scenario: Episode has season and episode numbers
- **WHEN** an episode card has season and episode numbers
- **THEN** the title area MUST include an `Sx:Ey` style label before or alongside the episode title

#### Scenario: Episode has progress or watched state
- **WHEN** user progress data is available for an episode
- **THEN** the episode card MUST show a progress or watched signal without hiding the episode still image or synopsis

### Requirement: Related media shelves
The system SHALL render related or similar catalog media as horizontal poster-card shelves when related candidates are available.

#### Scenario: Related candidates exist
- **WHEN** the detail response or related catalog query returns related media candidates
- **THEN** the detail page MUST render a horizontal shelf using poster imagery, title, year or year-range, and count badges where available

#### Scenario: No related candidates exist
- **WHEN** no related media candidates are available
- **THEN** the detail page MUST omit the related shelf without leaving an empty section

### Requirement: User-readable information section
The system SHALL prioritize user-readable media metadata in the bottom information section and keep technical asset or governance details secondary.

#### Scenario: Metadata and technical details both exist
- **WHEN** a detail item has media metadata and asset/governance details
- **THEN** the information section MUST show genres, studios or sources if available, external database links, ratings, dates, and status before technical file, probe, or governance details

#### Scenario: External identities exist
- **WHEN** a detail item has external identities with known providers
- **THEN** the information section MUST show provider labels and external IDs as database link entries, using clickable links where provider URL patterns are known

### Requirement: Detail page focus affordances
The system SHALL ensure interactive controls on the detail page have visible focus behavior and non-interactive decorative elements are not presented as controls.

#### Scenario: Keyboard user tabs through detail controls
- **WHEN** a keyboard or remote-style user navigates through the detail page controls
- **THEN** every focusable control MUST have a visible focus state and an action or clear unavailable feedback
