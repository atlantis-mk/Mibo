## ADDED Requirements

### Requirement: Metadata-level user data
The system SHALL store favorites and aggregate watched state against global metadata identities.

#### Scenario: Favorite follows metadata identity
- **WHEN** a user favorites a metadata identity that is visible in one library
- **THEN** the favorite remains associated with the metadata identity even if another library later exposes a different resource version

#### Scenario: Resource deleted keeps favorite
- **WHEN** a resource linked to a favorited metadata identity is removed from a library
- **THEN** the favorite remains stored on the metadata identity

### Requirement: Resource-level playback data
The system SHALL store version-specific playback state against resources and metadata identities.

#### Scenario: Progress stored for selected resource
- **WHEN** a user plays a specific resource version of a metadata identity
- **THEN** playback position is stored for that user, resource, and metadata identity

### Requirement: Progress inheritance across versions
The system SHALL allow playback to inherit metadata-level progress when no resource-specific progress exists.

#### Scenario: Switch to new version
- **WHEN** a user starts a 4K resource for a metadata identity previously watched on a 1080p resource
- **THEN** playback can resume from metadata-level progress if the 4K resource has no specific progress

### Requirement: Metadata completion aggregation
The system SHALL aggregate resource playback completion into metadata-level watched state.

#### Scenario: Complete one movie version
- **WHEN** a user completes a playable resource linked to a movie metadata identity
- **THEN** the metadata-level user data records the movie as completed

### Requirement: Version selection memory
The system SHALL remember the user's recent or preferred resource when selecting playback for a metadata identity.

#### Scenario: Resume preferred resource
- **WHEN** a user requests playback for a metadata identity without specifying a resource
- **THEN** the system prefers the user's recent or preferred available resource when selecting playback
