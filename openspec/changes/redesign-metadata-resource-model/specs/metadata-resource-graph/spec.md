## ADDED Requirements

### Requirement: Global metadata identities
The system SHALL represent canonical works and hierarchy nodes as global metadata identities that are not owned by any library.

#### Scenario: Metadata item has no library owner
- **WHEN** a movie, series, season, or episode metadata identity is created
- **THEN** the stored metadata identity has no `library_id` ownership field

#### Scenario: External identity resolves one metadata item
- **WHEN** two libraries contain resources matched to the same provider, provider type, and external ID
- **THEN** both resources link to the same global metadata identity instead of creating separate library-owned metadata identities

### Requirement: Metadata content typing
The system SHALL distinguish metadata structure type from content form.

#### Scenario: Anime episode typing
- **WHEN** an anime episode resource is linked to metadata
- **THEN** the metadata identity can store `item_type=episode` and `content_form=anime` independently of the library type

#### Scenario: Movie structure with documentary form
- **WHEN** a documentary film is linked to metadata
- **THEN** the metadata identity can store `item_type=movie` and `content_form=documentary`

### Requirement: Resource records model media entities
The system SHALL represent playable and related media entities as resources separate from metadata identities.

#### Scenario: Single playable file becomes resource
- **WHEN** the scanner discovers one playable movie file
- **THEN** the system creates or updates one playable resource and links the file to that resource

#### Scenario: Trailer is related resource
- **WHEN** the scanner classifies a file as a trailer
- **THEN** the system creates or updates a resource that can link to the metadata identity with trailer role

### Requirement: Inventory files are storage objects
The system SHALL model inventory files as storage objects that can attach to resources without defining metadata ownership.

#### Scenario: Same path rescan updates file
- **WHEN** the scanner sees the same media source, provider, and storage path again
- **THEN** the existing inventory file is updated rather than duplicated

#### Scenario: Different paths remain different files
- **WHEN** two files have the same basename but different storage paths
- **THEN** the system stores them as distinct inventory files

### Requirement: Resource file grouping
The system SHALL support one resource linking to one or more inventory files with file roles and part order.

#### Scenario: Multi-part movie resource
- **WHEN** two files are identified as part 1 and part 2 of the same movie resource
- **THEN** both files link to one resource with ordered source file roles

#### Scenario: External subtitle file
- **WHEN** a subtitle sidecar is associated with a playable resource
- **THEN** the subtitle file links to the resource with subtitle role

### Requirement: Resource metadata links
The system SHALL link resources to metadata identities with role, confidence, evidence, source, and review state.

#### Scenario: Movie version link
- **WHEN** a second resource is confidently identified as the same movie metadata identity
- **THEN** the system links the resource to the metadata identity as a version or primary role without creating a duplicate metadata identity

#### Scenario: Weak same-name candidate requires review
- **WHEN** a resource only matches an existing metadata identity by normalized title with no year or external identity evidence
- **THEN** the system records a reviewable candidate instead of automatically linking as a version

### Requirement: Multi-episode resource links
The system SHALL allow one resource to link to multiple episode metadata identities using segment indexes and optional time bounds.

#### Scenario: File spans two episodes
- **WHEN** the scanner identifies a file as covering S01E01 and S01E02
- **THEN** one resource links to both episode metadata identities with distinct segment indexes

### Requirement: Resource library membership
The system SHALL associate resources with libraries through explicit library membership links.

#### Scenario: Resource appears in one library
- **WHEN** a resource is discovered under one library path
- **THEN** the resource has a library membership link for that library

#### Scenario: Resource appears in multiple libraries
- **WHEN** the same resource is visible through two library paths
- **THEN** the resource can have membership links for both libraries without duplicating metadata

### Requirement: Old catalog ownership removal
The system SHALL remove retired code paths that rely on library-owned catalog metadata once equivalent metadata/resource/projection behavior is implemented.

#### Scenario: Retired catalog writer is unused
- **WHEN** the new scanner/materializer writes resources and metadata links
- **THEN** old catalog item creation paths are removed or made unreachable by production routes

#### Scenario: Tests target new model
- **WHEN** tests validate scan, metadata, browse, playback, and search behavior
- **THEN** they assert metadata/resource/projection behavior instead of old `CatalogItem.library_id` ownership
