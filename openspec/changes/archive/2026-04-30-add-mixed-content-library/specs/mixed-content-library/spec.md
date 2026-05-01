## ADDED Requirements

### Requirement: Library creation supports mixed content
The system SHALL allow users to create a media library with a mixed content type for source roots that contain both movie and TV-like groups.

#### Scenario: User selects mixed content during library creation
- **WHEN** a user creates a media library from the settings library form
- **THEN** the form SHALL offer a mixed content option alongside movie and show library types

#### Scenario: Mixed content library is persisted
- **WHEN** the user submits a valid library creation request with the mixed content type
- **THEN** the server SHALL persist the library with that type and include it in normal library list responses

### Requirement: Mixed library scanning excludes known extras from grouping counts
The scanner SHALL ignore known extras when counting media files for mixed-library movie-vs-series classification.

#### Scenario: Movie folder contains bonus videos
- **WHEN** a mixed library scan encounters a folder with one main movie file plus files identified as `trailer`, `behind-the-scenes`, `sample`, `featurette`, `interview`, or `deleted scene`
- **THEN** the scanner SHALL count only the main movie file for mixed content classification

#### Scenario: Extra keyword matching is bounded
- **WHEN** a mixed library scan encounters a valid title that merely contains an extra keyword as part of another word or title token
- **THEN** the scanner SHALL NOT classify that file as an extra using substring-only matching

### Requirement: Mixed library scanning classifies one-file groups as movies
The scanner SHALL classify a mixed-library group with exactly one non-extra supported video file as movie content.

#### Scenario: Single non-extra video creates movie item
- **WHEN** a mixed library scan encounters a media group with exactly one non-extra supported video file
- **THEN** the scanner SHALL create or update catalog movie content for that group using existing movie catalog semantics

### Requirement: Mixed library scanning classifies multi-file groups as TV content
The scanner SHALL classify a mixed-library group with more than one non-extra supported video file as TV-like content using existing series, season, and episode catalog semantics.

#### Scenario: Multiple non-extra videos create series hierarchy
- **WHEN** a mixed library scan encounters a media group with more than one non-extra supported video file
- **THEN** the scanner SHALL create or update a catalog series hierarchy for that group using existing TV catalog semantics

#### Scenario: Multi-file group lacks explicit episode numbers
- **WHEN** a mixed library scan classifies a group as TV-like and its non-extra videos do not include explicit episode numbers
- **THEN** the scanner SHALL assign deterministic episode ordering from the sorted non-extra files rather than leaving the group unclassified

### Requirement: Existing dedicated library behavior is preserved
The system SHALL preserve existing movie-library and show-library scan behavior when the library type is not mixed content.

#### Scenario: Movie library still treats bonus videos as movie extras
- **WHEN** a movie library scan encounters a movie folder with a main file and known extras
- **THEN** the scanner SHALL keep applying existing movie grouping and extra handling instead of using mixed content movie-vs-series classification

#### Scenario: Show library still uses TV scan rules
- **WHEN** a show library scan encounters episode-like files
- **THEN** the scanner SHALL keep applying existing TV classification rules instead of using mixed content one-file movie classification
