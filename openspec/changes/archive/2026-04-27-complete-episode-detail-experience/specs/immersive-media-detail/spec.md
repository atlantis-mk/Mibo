## ADDED Requirements

### Requirement: Detail layout adapts to media type
The immersive detail page SHALL render movies, series, seasons, and episodes with layouts appropriate to their catalog type.

#### Scenario: Open movie or series detail
- **WHEN** a user opens a movie or series detail page
- **THEN** the existing poster-first detail layout and series season shelf behavior MUST remain available

#### Scenario: Open episode detail
- **WHEN** a user opens an episode detail page
- **THEN** the page MUST use an episode-oriented layout with a 16:9 primary visual, parent series context, episode label, same-season shelf, and episode-specific metadata

### Requirement: Episode hero separates series title and episode title
The immersive detail page SHALL present episode identity without conflating the parent series title and the episode title.

#### Scenario: Episode has series and episode titles
- **WHEN** an episode has parent series title `A` and episode title `B`
- **THEN** the hero MUST show `A` as the primary series context and show `Sx:Ey - B` as episode context or subtitle

#### Scenario: Episode title is missing
- **WHEN** an episode lacks a provider episode title but has season and episode numbers
- **THEN** the hero MUST use a stable fallback such as `第 N 集` while preserving the series and season labels

### Requirement: Episode detail media controls show audio and subtitle choices
The immersive detail page SHALL present available audio and subtitle choices for the selected episode asset.

#### Scenario: Audio and subtitle tracks exist
- **WHEN** the selected asset has audio or subtitle tracks
- **THEN** the hero or media information area MUST show available track choices and indicate the default track when known

#### Scenario: No subtitle tracks exist
- **WHEN** the selected asset has no subtitle tracks
- **THEN** the subtitle control MUST show a clear off or unavailable state rather than an empty dropdown

### Requirement: Episode technical information is grouped by stream type
The immersive detail page SHALL group media details into video, audio, subtitle, and file information for episode assets.

#### Scenario: Stream details are available
- **WHEN** the item detail includes stream summaries for an episode asset
- **THEN** the page MUST render separate readable sections for video, audio, and subtitle details without mixing them into generic governance status rows

#### Scenario: Only asset summary is available
- **WHEN** detailed streams are not available but asset summary data exists
- **THEN** the page MUST render the known asset summary and avoid placeholder rows for unknown codecs or tracks
