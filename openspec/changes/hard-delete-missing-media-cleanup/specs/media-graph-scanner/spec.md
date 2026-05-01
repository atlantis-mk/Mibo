## ADDED Requirements

### Requirement: Missing media is hard deleted after retention
The system SHALL hard delete media records that remain missing beyond the configured retention period when the missing cleanup job runs.

#### Scenario: Missing file exceeds retention
- **WHEN** an inventory file, its media asset, and its catalog item have remained missing beyond the configured retention window
- **THEN** the cleanup job MUST physically delete the eligible inventory, asset, catalog, and dependent rows from the database
- **AND** the deleted records MUST NOT remain as soft-deleted rows with `deleted_at` set

#### Scenario: Missing file has not exceeded retention
- **WHEN** a media graph is marked missing but its missing age is less than the configured retention window
- **THEN** the cleanup job MUST leave the missing rows in place

### Requirement: Hard cleanup deletes user and governance state
The system SHALL delete user-specific, metadata, and governance records linked to media that is hard deleted by missing cleanup.

#### Scenario: Missing media has user activity
- **WHEN** a hard-deleted catalog item has favorites, playback progress, or user item data
- **THEN** the cleanup job MUST delete those user records together with the catalog item

#### Scenario: Missing media has manual curation
- **WHEN** a hard-deleted catalog item has manual metadata matches, field states, external IDs, selected images, tags, people links, or governance decisions
- **THEN** the cleanup job MUST delete those curation records together with the catalog item
- **AND** those records MUST NOT prevent hard deletion

### Requirement: Hard cleanup preserves available graph branches
The system SHALL avoid deleting catalog parents or shared assets that still have available local media descendants or links.

#### Scenario: Series has one missing and one available episode
- **WHEN** cleanup processes a missing episode in a series that still has another available episode
- **THEN** the cleanup job MUST hard delete the missing episode graph
- **AND** it MUST keep the series, season, and available episode graph

#### Scenario: Movie version is missing but another version remains
- **WHEN** cleanup processes a missing asset for a movie that still has another available source asset
- **THEN** the cleanup job MUST delete only the missing asset and file graph
- **AND** it MUST keep the movie catalog item available

### Requirement: Hard cleanup refreshes projections
The system SHALL refresh catalog projections after hard deleting missing media records.

#### Scenario: Cleanup deletes catalog records
- **WHEN** the cleanup job hard deletes one or more catalog records for a library
- **THEN** the system MUST refresh or queue refresh of catalog rollups and search documents for the affected library scope
- **AND** subsequent browse and search queries MUST NOT return deleted records
