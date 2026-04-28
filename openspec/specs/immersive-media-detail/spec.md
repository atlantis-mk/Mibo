# immersive-media-detail Specification

## Purpose
Define catalog-backed immersive media detail page presentation, actions, shelves, and accessibility semantics.

## Requirements

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

### Requirement: Series detail hero plays a local episode target
The immersive media detail page SHALL use a series playback target for the primary series play action when local episodes exist.

#### Scenario: Series has a continue target
- **WHEN** a user opens a series detail page whose detail response identifies an unfinished local episode target
- **THEN** the primary action MUST be labeled as continue playback and MUST open playback for that episode item and selected asset

#### Scenario: Series has a first local episode target
- **WHEN** a user opens a series detail page with locally playable episodes and no unfinished episode target
- **THEN** the primary action MUST be labeled as play and MUST open playback for the earliest local episode target

#### Scenario: Series has no local playback target
- **WHEN** a user opens a series detail page with no locally playable episode target
- **THEN** the primary play action MUST be disabled or replaced with clear unavailable feedback without navigating to an unplayable series item

### Requirement: Series detail shelves show local episode information by default
The immersive media detail page SHALL render the default series episode shelves from local playable episode data only.

#### Scenario: Series contains unavailable provider episodes
- **WHEN** the series hierarchy contains local playable episodes as well as missing or unaired provider-known episodes
- **THEN** the default `剧集信息` shelf MUST display only local playable episode cards and MUST calculate displayed season episode counts from those cards

#### Scenario: A season has no local playable episodes
- **WHEN** a season contains only missing, unaired, or otherwise unplayable episode descendants
- **THEN** the default series detail shelf MUST omit that season rather than showing non-playable episode cards

#### Scenario: User opens an unavailable episode directly
- **WHEN** a user explicitly opens a missing or unaired episode detail page from a governance or missing-episode workflow
- **THEN** the page MUST preserve the episode detail unavailable state and MUST NOT pretend that the episode is locally playable

### Requirement: Detail page renders MediaInfo-style video specifications
The system SHALL render available video stream technical attributes on the immersive media detail page using a MediaInfo-style label/value presentation.

#### Scenario: Primary asset has detailed video technical attributes
- **WHEN** a user opens a catalog-backed media detail page whose primary asset has a video stream with detailed technical attributes
- **THEN** the video information card MUST display fields such as title, codec, profile, level, resolution, aspect ratio, interlace state, frame rate, bitrate, color space, bit depth, pixel format, and reference frames when those values are available

#### Scenario: Multiple video streams exist
- **WHEN** a primary asset has multiple video streams
- **THEN** the video information card MUST distinguish each stream and render the available technical specification fields for each stream

#### Scenario: Video technical attributes are incomplete
- **WHEN** a primary asset has a video stream with only compact metadata such as codec and dimensions
- **THEN** the video information card MUST still render the available compact values and avoid empty rows for unavailable detailed fields

### Requirement: Detail page keeps non-video technical summaries available
The system SHALL preserve existing audio, subtitle, and file technical summaries while expanding the video stream display.

#### Scenario: User views media information after the video display changes
- **WHEN** the media detail page renders expanded video technical specifications
- **THEN** the page MUST still show available audio tracks, subtitle tracks, and file summary information in the media information section
