## MODIFIED Requirements

### Requirement: Library creation supports mixed content
The system SHALL replace user-selected mixed content library creation with source-first creation and automatic video classification.

#### Scenario: User creates a source during library creation
- **WHEN** a user creates content from the settings or setup flow
- **THEN** the form SHALL NOT offer movie, show, or mixed library type choices and SHALL accept a storage source and root path instead

#### Scenario: Automatic mixed behavior is used internally
- **WHEN** the scanner processes video files from a source-first path
- **THEN** the scanner SHALL apply automatic movie-vs-series classification without requiring a persisted mixed library type selected by the user

### Requirement: Mixed library scanning excludes known extras from grouping counts
The scanner SHALL ignore known extras when counting media files for automatic video movie-vs-series classification.

#### Scenario: Movie folder contains bonus videos
- **WHEN** automatic video classification encounters a folder with one main movie file plus files identified as `trailer`, `behind-the-scenes`, `sample`, `featurette`, `interview`, or `deleted scene`
- **THEN** the scanner SHALL count only the main movie file for movie-vs-series classification

#### Scenario: Extra keyword matching is bounded
- **WHEN** automatic video classification encounters a valid title that merely contains an extra keyword as part of another word or title token
- **THEN** the scanner SHALL NOT classify that file as an extra using substring-only matching

### Requirement: Mixed library scanning classifies one-file groups as movies
The scanner SHALL classify an automatic video group with exactly one non-extra supported video file as movie content unless stronger evidence indicates another semantic type.

#### Scenario: Single non-extra video creates movie item
- **WHEN** automatic video classification encounters a media group with exactly one non-extra supported video file and no stronger TV evidence
- **THEN** the scanner SHALL create or update catalog movie content for that group using existing movie catalog semantics

### Requirement: Mixed library scanning classifies multi-file groups as TV content
The scanner SHALL classify automatic video groups with more than one non-extra supported video file using evidence-based movie, series, season, episode, version, and review semantics rather than unconditionally treating every multi-file group as TV-like content.

#### Scenario: Multiple episode-like videos create series hierarchy
- **WHEN** automatic video classification encounters a media group with multiple non-extra videos and episode or season evidence
- **THEN** the scanner SHALL create or update a catalog series hierarchy for that group using existing TV catalog semantics

#### Scenario: Multi-file group lacks explicit episode numbers
- **WHEN** automatic video classification encounters a multi-file group without explicit episode numbers and with sufficient series-folder evidence
- **THEN** the scanner SHALL assign deterministic episode ordering from the sorted non-extra files rather than leaving the group unclassified

#### Scenario: Multi-file movie versions are detected
- **WHEN** automatic video classification encounters a multi-file group whose files appear to be versions of the same movie work
- **THEN** the scanner SHALL create or update one movie item with multiple assets instead of creating a series hierarchy

#### Scenario: Multi-file group remains ambiguous
- **WHEN** automatic video classification cannot confidently distinguish series episodes from movie versions or unrelated videos
- **THEN** the scanner SHALL mark the decision for governance review with evidence and confidence instead of silently choosing TV-like content

### Requirement: Existing dedicated library behavior is preserved
The system SHALL remove user-visible dedicated movie-library and show-library scan behavior in favor of source-first automatic video classification.

#### Scenario: User-visible dedicated types are unavailable
- **WHEN** a user creates or edits a source from setup or settings
- **THEN** the UI SHALL NOT expose dedicated movie, show, or mixed library type choices

#### Scenario: Scanner does not require dedicated type hints
- **WHEN** a source-first video scan runs
- **THEN** the scanner SHALL classify movie and TV semantics from source evidence without depending on a movie or show library type hint
