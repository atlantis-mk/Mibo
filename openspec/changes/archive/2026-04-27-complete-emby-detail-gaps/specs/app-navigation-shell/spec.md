## ADDED Requirements

### Requirement: Detail page top navigation uses app shell semantics
The system SHALL render detail-page top navigation entries as real app actions or clearly unavailable actions consistent with the authenticated app shell.

#### Scenario: User activates detail search entry
- **WHEN** the user activates the search entry from the media detail top navigation
- **THEN** the app MUST open the global search surface or navigate to the search route instead of rendering a decorative icon only

#### Scenario: User activates detail user entry
- **WHEN** the user activates the user entry from the media detail top navigation
- **THEN** the app MUST display the user menu with session-relevant actions consistent with the app shell

#### Scenario: User activates unsupported cast entry
- **WHEN** the user activates a cast entry from the media detail top navigation and real casting support is unavailable
- **THEN** the app MUST show clear unavailable or coming-soon feedback instead of silently doing nothing

#### Scenario: User activates settings entry
- **WHEN** the user activates the settings entry from the media detail top navigation
- **THEN** the app MUST navigate to the settings route
