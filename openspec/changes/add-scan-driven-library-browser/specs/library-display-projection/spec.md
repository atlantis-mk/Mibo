## ADDED Requirements

### Requirement: Library metadata projections persist browse display roots
The system SHALL persist a library-scoped browse display root for each projected metadata item so hierarchical library browsing can determine where that item belongs without recomputing folder placement from raw scan tables on every request.

#### Scenario: Movie projection stores its media-root directory
- **WHEN** a library metadata projection is rebuilt for a movie whose source files live under a single media directory
- **THEN** the projection SHALL store the normalized relative display root path for that movie within the selected library

#### Scenario: Series projection stores a shared series-root directory
- **WHEN** a library metadata projection is rebuilt for a series whose playable files are attached to descendant episodes or seasons
- **THEN** the projection SHALL derive and store a shared normalized display root path that represents the series directory within the selected library

### Requirement: Display-root derivation uses playable source-file evidence
The system SHALL derive projection display roots from playable source files and SHALL ignore non-source linked files when determining the browse placement of a metadata item.

#### Scenario: Subtitle sidecar does not change display root
- **WHEN** a metadata item has subtitle or supplemental linked files outside the main media directory
- **THEN** the system SHALL compute the display root from source-role media files and SHALL NOT move the browse placement to the sidecar directory

#### Scenario: Multi-version movie uses a shared media root
- **WHEN** one movie is linked to multiple source files that differ only by version or edition folders
- **THEN** the system SHALL store a single display root that resolves to the shared movie directory rather than a version-specific child folder

### Requirement: Display-root derivation collapses structural child folders
The system SHALL collapse technical child directories that represent structure or packaging rather than category semantics when computing a projection display root.

#### Scenario: Movie split-part folders collapse to the movie directory
- **WHEN** a movie's source files are stored under folders such as `CD1`, `CD2`, `Part1`, or similar split-part directories
- **THEN** the projection SHALL store the parent movie directory as the display root

#### Scenario: Series season folders collapse to the series directory
- **WHEN** a series' descendant source files are stored under folders such as `Season 01`, `S01`, or `第1季`
- **THEN** the projection SHALL store the series directory above those season folders as the display root

### Requirement: Projection display semantics refresh with existing rebuild flows
The system SHALL refresh stored display-root semantics whenever the corresponding library metadata projection is rebuilt through existing projection refresh workflows.

#### Scenario: Resource relink updates display root
- **WHEN** resource-to-metadata links or source-file placement change and the affected projection is rebuilt
- **THEN** the stored display root and related display fields SHALL be recomputed from the updated scan/link state

#### Scenario: Full library projection rebuild backfills display roots
- **WHEN** the system runs a full projection rebuild for a library
- **THEN** it SHALL populate display-root fields for all rebuilt projection rows in that library
