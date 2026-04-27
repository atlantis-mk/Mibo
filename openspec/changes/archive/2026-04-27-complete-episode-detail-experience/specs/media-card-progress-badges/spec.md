## ADDED Requirements

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
