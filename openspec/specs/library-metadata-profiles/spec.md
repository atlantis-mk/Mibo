# library-metadata-profiles Specification

## Purpose
TBD - created by archiving change add-library-metadata-profiles. Update Purpose after archive.
## Requirements
### Requirement: Libraries bind to an explicit metadata profile
The system SHALL allow each media library to bind to a named metadata profile that defines its metadata acquisition method instead of relying solely on global provider settings or implicit library-type defaults.

#### Scenario: Library uses a reusable profile
- **WHEN** an operator assigns a metadata profile to a library
- **THEN** metadata match and refetch operations for items in that library MUST resolve and execute the assigned profile before calling any online provider stages

#### Scenario: Library retains migrated default behavior
- **WHEN** an existing library is upgraded into the profile-backed system without an operator choosing a new strategy
- **THEN** the system MUST bind that library to a migrated default profile that preserves the closest equivalent of its prior metadata behavior

### Requirement: Metadata profiles define stage-specific provider behavior
The system SHALL allow a metadata profile and copied library strategy to declare stage-specific behavior for candidate search, detail fetch, image sync, people sync, hierarchy sync, and local-source participation, and metadata operations SHALL execute those stages through the unified metadata operation pipeline.

#### Scenario: Anime library uses mixed providers by stage
- **WHEN** a library strategy routes search and detail stages to a custom supported provider instance and image stages to a TMDB provider instance
- **THEN** the system MUST execute each stage according to that stage's configured provider order instead of forcing one provider to satisfy every stage

#### Scenario: Local-only library skips online stages
- **WHEN** a library strategy is configured as local-only or local-evidence-only
- **THEN** the system MUST skip online provider search and detail stages while still allowing local scanner and sidecar evidence to participate in catalog metadata operations

#### Scenario: Strategy determines automated matchability
- **WHEN** an automated match job runs for a catalog item
- **THEN** the system MUST determine whether the item can be matched from the item's library strategy rather than from global provider settings

### Requirement: Libraries may apply constrained profile overrides
The system SHALL support limited per-library overrides on top of a bound metadata profile for values such as preferred metadata language, preferred image language, metadata country code, local evidence behavior, or local-only enforcement without requiring a duplicate profile definition.

#### Scenario: Library overrides profile language
- **WHEN** two libraries share the same metadata profile and one library sets a metadata language override
- **THEN** the overridden library MUST use the override for metadata requests while the other library continues using the profile default

#### Scenario: Library overrides local evidence behavior
- **WHEN** a library enables local evidence application even though its template also includes remote providers
- **THEN** metadata operations for that library MUST include local scanner evidence according to the library override and record whether local evidence seeded or supplied the selected metadata
