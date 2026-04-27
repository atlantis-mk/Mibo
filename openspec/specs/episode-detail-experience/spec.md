# episode-detail-experience Specification

## Purpose
Define episode-specific catalog detail behavior, including parent context, episode artwork, same-season navigation, playable assets, stream information, credits, and availability-aware actions.

## Requirements

### Requirement: Episode detail page preserves series and season context
The system SHALL render an opened episode with explicit parent series and season context rather than treating it as a standalone movie-like item.

#### Scenario: User opens an episode from a series shelf
- **WHEN** a user activates an episode card from a series detail shelf
- **THEN** the episode detail page MUST show the series title, season number, episode number, episode title, and available episode metadata without requiring the user to navigate back to the series page

#### Scenario: Episode parent context is missing
- **WHEN** an episode detail response lacks required parent series or season context
- **THEN** the page MUST still render the episode title and available actions while showing a clear incomplete-metadata state instead of fabricating series context from the file path

### Requirement: Episode detail hero uses episode-appropriate artwork and labels
The system SHALL use episode still imagery and series artwork fallbacks in the episode detail hero.

#### Scenario: Episode has still artwork
- **WHEN** an episode has selected still or backdrop artwork
- **THEN** the detail hero MUST use a 16:9 episode image for the primary visual and display the episode label in `Sx:Ey` form near the episode title

#### Scenario: Episode lacks still artwork
- **WHEN** an episode lacks selected still imagery but the parent series has backdrop or poster artwork
- **THEN** the detail hero MUST use the parent artwork as fallback without forcing the episode into a 2:3 poster-only layout

### Requirement: Episode detail includes same-season navigation
The system SHALL show same-season episodes on an episode detail page when sibling episode data is available.

#### Scenario: Same-season siblings exist
- **WHEN** a user opens an episode whose containing season has other episodes
- **THEN** the detail page MUST render a horizontal same-season shelf labeled with the season and MUST highlight or otherwise mark the currently opened episode

#### Scenario: No sibling episodes exist
- **WHEN** no same-season sibling data is available
- **THEN** the detail page MUST omit the same-season shelf or show a compact unavailable message without blocking the hero and playback actions

### Requirement: Episode detail exposes playable asset choices and media stream information
The system SHALL present the selected episode asset's video, audio, subtitle, and file information when stream metadata is available.

#### Scenario: Asset has probed media streams
- **WHEN** the selected playable episode asset has video, audio, or subtitle stream metadata
- **THEN** the detail page MUST show a video summary and detailed audio/subtitle entries including language, codec, title, channels, and any known default or forced flags

#### Scenario: Asset stream metadata is incomplete
- **WHEN** an episode asset is available but stream metadata is missing or probe status is pending
- **THEN** the detail page MUST show the known asset and file state and keep reprobe or management actions available without inventing stream values

### Requirement: Episode detail people prefer episode-level credits
The system SHALL display episode-level people before falling back to series-level credits.

#### Scenario: Episode credits exist
- **WHEN** provider or catalog data includes directors, guest cast, or cast for the opened episode
- **THEN** the people section MUST render those episode-level people with names, roles, and avatars where available

#### Scenario: Episode credits are absent
- **WHEN** the episode has no episode-level people but the parent series has people data
- **THEN** the page MAY show parent series people as fallback while avoiding empty placeholder person cards

### Requirement: Episode playback actions reflect availability
The system SHALL make episode playback actions depend on the opened episode's linked assets.

#### Scenario: Episode has playable asset
- **WHEN** an episode has at least one available linked asset
- **THEN** play and version-selection actions MUST open playback for the episode catalog item and selected asset

#### Scenario: Episode has no playable asset
- **WHEN** an episode is missing, unaired, or has no linked playable asset
- **THEN** play actions MUST be disabled or replaced with clear unavailable feedback while keeping metadata and governance actions accessible
